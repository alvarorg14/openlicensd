# Troubleshooting OpenLicensd

This runbook is the first stop when OpenLicensd fails to start, pods stay unready, or optional features (OIDC SSO, Harbor registry credentials) misbehave. It covers how to collect evidence, then symptom tables for database connectivity, migration failures and advisory locks, OIDC, and Harbor.

For setup walkthroughs, see [oidc-sso.md](oidc-sso.md) and [harbor-registry-credentials.md](harbor-registry-credentials.md). For upgrades, backups, and HA, see [upgrade.md](upgrade.md), [backup-restore.md](backup-restore.md), and [scaling.md](scaling.md).

## What this covers

| Failure class | When to use this section |
|---------------|--------------------------|
| **Process will not start** | CrashLoopBackOff, `connection refused` on probes, immediate exit after deploy |
| **Database connectivity** | `store init failed`, `/readyz` returns `503`, intermittent DB outages |
| **Migration failures and locks** | `apply migration …` errors, startup hangs during rolling upgrade |
| **OIDC SSO** | Login redirect failures, `Single sign-on failed`, discovery errors at startup |
| **Harbor registry credentials** | `502 failed to issue registry credentials`, route `404`, license `403` |

## Collecting evidence

Start with logs and health probes before changing configuration.

### Logs

OpenLicensd emits structured JSON logs by default (`OPENLICENSD_LOG_FORMAT=json`). Search for these startup messages:

| Log message | Meaning |
|-------------|---------|
| `config load failed` | Invalid or missing environment variables |
| `store init failed` | Database connect, ping, or migration failure |
| `api server init failed` | OIDC discovery or Harbor URL parse failure |
| `bootstrap admin failed` | Empty database without bootstrap admin env vars |
| `listening` | HTTP server bound; probes can reach the process |

**Kubernetes:**

```bash
kubectl logs -n openlicensd deploy/openlicensd --tail=200
kubectl describe pod -n openlicensd -l app.kubernetes.io/name=openlicensd
```

**Docker Compose / single container:** inspect container stdout.

Set `OPENLICENSD_LOG_LEVEL=debug` temporarily for more detail (including successful probe requests). Failed readiness checks (`503` on `/readyz`) still emit a `warn` line at the default `info` level.

### Health probes

| Endpoint | Purpose | Success | Failure |
|----------|---------|---------|---------|
| `GET /healthz` | Liveness — process answered HTTP | `200` | — |
| `GET /readyz` | Readiness — PostgreSQL reachable | `200` | `503` with `{"error":"database unavailable"}` |

```bash
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/healthz
curl -s localhost:8080/readyz
```

**Important:** `/healthz` does **not** check the database. A pod can be alive (`/healthz` 200) but not ready (`/readyz` 503) when PostgreSQL is down.

The Helm chart maps probes as follows:

| Kubernetes probe | Path |
|------------------|------|
| Liveness | `/healthz` |
| Readiness | `/readyz` |
| Startup | `/readyz` (allows migrations to finish before liveness begins) |

During startup, migrations run **before** the HTTP server listens. Until `listening` appears in logs, probes see **connection refused** — this is normal for a short window. A second replica waiting on the migration advisory lock can hang longer (see [Migration failures and locks](#migration-failures-and-locks)).

## Process will not start

OpenLicensd exits with code `1` on any fatal startup error. The process never binds `:8080`, so Kubernetes shows CrashLoopBackOff or startup probe failures.

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `config load failed` + `OPENLICENSD_DATABASE_URL is required` | Missing database URL | Set `OPENLICENSD_DATABASE_URL` (or Helm `secret.data.databaseUrl`) |
| `config load failed` + validation error on pool settings | Invalid `OPENLICENSD_DATABASE_MIN_CONNS` / `MAX_CONNS` | Fix pool env vars; see [configuration.md](configuration.md) |
| `config load failed` + `at least one login method must be enabled` | Both local login and OIDC disabled | Set `OPENLICENSD_LOCAL_LOGIN_ENABLED=true` or `OPENLICENSD_OIDC_ENABLED=true` |
| `store init failed` | Database unreachable, auth failure, or migration error | See [Database connectivity](#database-connectivity) and [Migration failures and locks](#migration-failures-and-locks) |
| `api server init failed` + `discover oidc provider` | Wrong or unreachable OIDC issuer | Fix `OPENLICENSD_OIDC_ISSUER_URL`; verify network and TLS from the pod |
| `api server init failed` + `invalid harbor url` / `parse harbor url` | Malformed `OPENLICENSD_HARBOR_URL` | Use a full URL with scheme and host (e.g. `https://harbor.example.com`) |
| `bootstrap admin failed` + `no users exist` | Fresh database without bootstrap admin | Set `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL` and `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` |

When OIDC or Harbor is **enabled**, missing required env vars fail at config load. When **disabled**, those routes are not registered and the server starts without them.

## Database connectivity

Database errors surface during startup (`store init failed`) or at runtime (`/readyz` 503).

### Startup failures

Error messages are wrapped with these prefixes:

| Prefix | Stage |
|--------|-------|
| `parse database URL:` | Invalid connection string |
| `connect to database:` | Pool creation failed |
| `ping database:` | Server reachable but ping failed |

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `connection refused` in ping error | PostgreSQL down or wrong host/port | Verify Postgres is running; check URL host and port |
| `password authentication failed` | Wrong credentials | Update `OPENLICENSD_DATABASE_URL` user/password |
| `database "…" does not exist` | Database not created | Create the database or fix the URL path |
| TLS / SSL errors | `sslmode` mismatch | Adjust `sslmode` in the URL (`require`, `verify-full`, etc.) |
| `store init failed` after long hang | Network policy or firewall | Confirm the pod can reach Postgres on the configured port |

On success, logs show `database pool configured` followed by `database connected`.

### Runtime failures

If PostgreSQL becomes unreachable **after** a successful start:

- The process **stays running** (liveness `/healthz` remains `200`).
- Readiness `/readyz` returns `503 {"error":"database unavailable"}`.
- Kubernetes removes the pod from Service endpoints but does **not** restart it via liveness.

Restore database connectivity; the pod should pass readiness again without a restart.

### Statement timeout

When `OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS` is set, all pooled connections inherit a PostgreSQL `statement_timeout`. Queries (including migrations and `/readyz` ping) that exceed the limit are cancelled.

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `apply migration …` cancelled / timeout during upgrade | Statement timeout too aggressive for migration SQL | Temporarily increase or unset the timeout for the upgrade window |
| `/readyz` 503 under load | Slow queries or saturated pool | Tune pool size (`OPENLICENSD_DATABASE_MAX_CONNS`) and Postgres capacity; see [scaling.md](scaling.md) |

## Migration failures and locks

Migrations are embedded SQL files applied automatically on startup. There is no separate migration CLI.

| Behavior | Detail |
|----------|--------|
| **Forward-only** | Applied versions are recorded in `schema_migrations`; no down-migration |
| **Transactions** | Each file runs in one transaction; SQL errors roll back that file |
| **Concurrency** | Replicas serialize on `pg_advisory_lock` (lock ID `OPENLICE`) |
| **Ordering** | Files in `server/internal/store/migrations/` sorted by filename |

### Log messages

| Message | Meaning |
|---------|---------|
| `migration applied` + `version=<file>` | One migration file committed |
| `migrations complete` + `applied=<n>` | Batch of pending migrations finished |
| `schema up to date` | Nothing pending |
| `database connected` | Migrations finished; HTTP server can start |

### Symptom table

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| Startup hang, no `listening`, no migration error yet | Another replica holds the advisory lock | Wait for the first pod to finish; check its logs for `migration applied` |
| `store init failed` + `apply migration <file>:` | SQL error (permissions, missing extension, incompatible schema) | Read the wrapped Postgres error; fix permissions or restore from backup |
| `store init failed` + `acquire migration lock:` | Cannot acquire advisory lock (connection issue) | Check database connectivity |
| New pod CrashLoop after upgrade, old pods still serving | Migration failed before commit | No schema change applied — fix the error and redeploy; see [upgrade.md](upgrade.md) rollback section |
| Downgrade binary after successful migration | `schema_migrations` ahead of binary | **Do not** only roll back the image — restore a pre-upgrade `pg_dump`; see [upgrade.md](upgrade.md) |

Example destructive step: `008_drop_archived_at.sql` drops columns with no automatic undo. Always take a `pg_dump` before upgrading ([backup-restore.md](backup-restore.md)).

## OIDC SSO

OIDC is optional. When `OPENLICENSD_OIDC_ENABLED=true`, misconfiguration can prevent startup or cause runtime login failures.

### Startup failures

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `api server init failed` + `discover oidc provider` | Wrong issuer URL, IdP down, or TLS failure | Verify `OPENLICENSD_OIDC_ISSUER_URL`; test discovery from the pod |
| `config load failed` + missing OIDC field | Required env not set when enabled | Set issuer, client ID, client secret, and redirect URL |

OIDC routes return `404` when the feature is disabled.

### Runtime login failures

Most callback errors redirect to `/login?error=sso_failed`. The login page shows:

> Single sign-on failed. Please try again or use your email and password.

The server logs a **warn** line with a `reason` code (the underlying error is **not** logged for exchange/user/session failures):

| `reason` | When |
|----------|------|
| `provider_error` | IdP returned `?error=…` (e.g. `redirect_uri_mismatch`) |
| `state_missing` | OIDC state cookie missing (often `COOKIE_SECURE` over HTTP) |
| `state_mismatch` | State cookie does not match callback |
| `nonce_missing` | Nonce cookie missing |
| `verifier_missing` | PKCE verifier cookie missing |
| `code_missing` | No authorization `code` in callback |
| `exchange_failed` | Token exchange or ID token verification failed |
| `user_resolution_failed` | User create/link failed or user disabled |
| `session_creation_failed` | Session could not be created |

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `redirect_uri_mismatch` from IdP | Redirect URI not registered | Align IdP with `OPENLICENSD_OIDC_REDIRECT_URL` exactly |
| `reason=exchange_failed` | Wrong client secret, missing `email` claim, clock skew | Verify secret and scopes; ensure IdP returns `email`; sync NTP |
| `reason=state_missing` / `nonce_missing` | Cookies not stored | Use HTTPS, or set `OPENLICENSD_COOKIE_SECURE=false` for local HTTP only |
| SSO button missing | OIDC disabled | Set `OPENLICENSD_OIDC_ENABLED=true` and restart |
| `504 request timeout` during callback | Slow IdP or database | Increase `OPENLICENSD_REQUEST_TIMEOUT_SECONDS`; check DB latency |

For provider setup, redirect URI registration, and role provisioning, see [oidc-sso.md](oidc-sso.md).

## Harbor registry credentials

Harbor integration is optional. When `OPENLICENSD_HARBOR_ENABLED=true`, config errors or an unparseable URL prevent startup; connectivity and auth errors appear at request time.

### Startup failures

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `config load failed` + missing Harbor field | Required env not set when enabled | Set URL, admin username/password, and projects |
| `api server init failed` + `invalid harbor url` / `parse harbor url` | Malformed URL | Use `https://harbor.example.com` (scheme + host required) |

The `/api/v1/registry-credentials` route returns `404` when Harbor is disabled.

### Runtime failures

| Status | `error` value | Cause |
|--------|---------------|-------|
| `400` | `invalid request body` / `key is required` | Malformed request |
| `403` | `not_found` | License key not found |
| `403` | `expired` | License expired |
| `403` | `revoked` | License revoked |
| `403` | `product_mismatch` | `product` query/body does not match license |
| `403` | `fingerprint_required` | Policy requires machine fingerprint |
| `403` | `activation_limit` | Max concurrent activations reached |
| `502` | `failed to issue registry credentials` | Harbor robot creation failed |
| `500` | `failed to validate license` | Database error during validation |
| `504` | `request timeout` | Request deadline exceeded |

When `OPENLICENSD_HARBOR_DEBUG=true`, `502` responses and logs include the underlying Harbor error. Logs also emit `registry credentials harbor create robot failed` at error level.

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `502` + TLS errors in debug output | Self-signed Harbor certificate | Set `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY=true` (dev/trusted networks only) |
| `502` + `401` / `403` from Harbor | Wrong admin credentials or insufficient permissions | Verify `OPENLICENSD_HARBOR_ADMIN_USERNAME` / `_PASSWORD`; admin must create system-level robots |
| `502` + project not found | Typo in `OPENLICENSD_HARBOR_PROJECTS` | Confirm namespace exists in Harbor |
| `502` + connection timeout | Harbor unreachable from pod | Check network policy, DNS, and firewall |
| `harbor cleanup failed` in logs | Admin lacks delete permission | Credentials still issued; fix Harbor permissions or ignore if robots are short-lived |
| Route `404` | Harbor not enabled | Set `OPENLICENSD_HARBOR_ENABLED=true` and restart |

For robot account behavior, project configuration, and security notes, see [harbor-registry-credentials.md](harbor-registry-credentials.md).

## Related

- [configuration.md](configuration.md) — environment variables and Helm values
- [deployment.md](deployment.md) — Helm install, health probes, and Ingress
- [upgrade.md](upgrade.md) — migrations, rolling updates, and rollback
- [backup-restore.md](backup-restore.md) — `pg_dump`/`pg_restore` when migrations commit
- [scaling.md](scaling.md) — HA, rate limiting, and connection budget
- [oidc-sso.md](oidc-sso.md) — OIDC setup and provider walkthroughs
- [harbor-registry-credentials.md](harbor-registry-credentials.md) — Harbor setup and robot accounts
- [metrics.md](metrics.md) — Prometheus metrics for pool and rate limiter health
- [QUICKSTART.md](../QUICKSTART.md) — get running quickly
- [charts/openlicensd/README.md](../charts/openlicensd/README.md) — Helm chart reference
