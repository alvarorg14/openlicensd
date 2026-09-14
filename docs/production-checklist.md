# Production checklist

Use this checklist before your first production deploy, at go-live, and during periodic operational reviews. For background on each item, see [security-hardening.md](security-hardening.md).

Copy sections into your own runbook or tick items inline.

## Pre-production (before first deploy)

### Versioning and artifacts

- [ ] Pin a semver release — do not rely on floating tags in production:
  - Helm: `--version X.Y.Z` from `oci://ghcr.io/alvarorg14/charts/openlicensd`
  - Container: `ghcr.io/alvarorg14/openlicensd:X.Y.Z` (not `latest`)
- [ ] Verify the container image Cosign signature and GitHub Artifact Attestation (see [deployment.md](deployment.md#releases))

### Secrets and configuration

- [ ] Store secrets in a secrets manager, Kubernetes Secret, or External Secrets Operator — not in git or plain ConfigMaps:
  - `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` (bcrypt via `make hash-password`)
  - `OPENLICENSD_DATABASE_PASSWORD`
  - `OPENLICENSD_OIDC_CLIENT_SECRET` (when OIDC enabled)
  - Harbor admin credentials (when Harbor enabled)
- [ ] Set bootstrap admin email and password hash for a fresh database:
  - `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL`
  - `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH`
- [ ] Configure discrete database connection variables (v0.9.0+):
  - `OPENLICENSD_DATABASE_HOST`, `PORT`, `USER`, `PASSWORD`, `NAME`
  - `OPENLICENSD_DATABASE_SSLMODE=require` (or `verify-full` when supported)
  - Optional: `OPENLICENSD_DATABASE_OPTIONS` for extra libpq parameters

### Transport and cookies

- [ ] Terminate TLS at Ingress or reverse proxy
- [ ] Set `OPENLICENSD_COOKIE_SECURE=true` (Helm `config.cookieSecure: true`)
- [ ] Confirm production URL uses `https://` end to end

### Proxy and scaling

- [ ] Set `OPENLICENSD_TRUSTED_PROXIES` (Helm `config.trustedProxies`) when behind an ingress or load balancer
- [ ] If running more than one replica, set `OPENLICENSD_RATE_LIMIT_BACKEND=postgres` (Helm `config.rateLimit.backend: postgres`)
- [ ] Enable PodDisruptionBudget when `replicaCount >= 2` (Helm `pdb.enabled: true`)

### Network and observability

- [ ] Decide on NetworkPolicy — enable and tighten per [deployment.md](deployment.md#network-policy) if your CNI supports it
- [ ] Confirm `/metrics` is **not** exposed on public Ingress (separate listener on `:9090`)
- [ ] Configure in-cluster Prometheus scrape or ServiceMonitor (see [metrics.md](metrics.md))

### OIDC (when enabled)

- [ ] Register exact redirect URI: `https://<host>/api/v1/auth/oidc/callback`
- [ ] Set `OPENLICENSD_OIDC_REDIRECT_URL` to match byte for byte
- [ ] Configure the IdP to verify email addresses and include `email_verified: true` in ID tokens
- [ ] List initial admin emails in `OPENLICENSD_OIDC_ADMIN_EMAILS` or plan local bootstrap admin
- [ ] Consider `OPENLICENSD_LOCAL_LOGIN_ENABLED=false` for SSO-only production

### Harbor (when enabled)

- [ ] Restrict Harbor admin credentials to pull-only robot scope via configured projects
- [ ] Set `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY=false`
- [ ] Set `OPENLICENSD_HARBOR_DEBUG=false`

## Go-live verification

Run these checks after deploy and before directing production traffic.

### Health probes

- [ ] `GET /healthz` returns `200` (liveness — no dependency checks)
- [ ] `GET /readyz` returns `200` (readiness — PostgreSQL ping)

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://licenses.example.com/healthz
curl -s -o /dev/null -w "%{http_code}\n" https://licenses.example.com/readyz
```

### Authentication

- [ ] Sign in to the admin UI (local password or OIDC SSO)
- [ ] Confirm CSRF-protected mutations work (create a test product)
- [ ] If OIDC: verify redirect URI, session cookies, and profile photo (when `picture` claim present)

### Core license flow

- [ ] Create a test product and policy
- [ ] Issue a test license and store the raw key securely (shown once)
- [ ] `POST /api/v1/validate` returns valid for the test key

```bash
curl -s -X POST https://licenses.example.com/api/v1/validate \
  -H 'Content-Type: application/json' \
  -d '{"license_key":"YOUR-TEST-KEY"}'
```

### Optional integrations

- [ ] Harbor: `POST /api/v1/registry-credentials` returns credentials for a licensed client (when enabled)
- [ ] Metrics: in-cluster scrape of `:9090/metrics` succeeds
- [ ] Audit log: confirm a test mutation appears in **Audit Log** or `GET /api/v1/audit-events`

## Ongoing operations

Review on a schedule (for example quarterly) or after significant changes.

### Backups

- [ ] PostgreSQL backups run on a defined schedule (daily minimum for small installs; PITR for production — see [backup-restore.md](backup-restore.md))
- [ ] Backup retention meets your RPO target
- [ ] **Restore drill completed** — restore to a non-production database and verify:
  - `GET /readyz` returns `200`
  - Admin sign-in works (sessions from before restore are invalid)
  - Products, policies, and licenses are present
  - See [backup-restore.md](backup-restore.md#verify-after-restore)

### Upgrades

- [ ] Read release notes and [CHANGELOG.md](../CHANGELOG.md) before each upgrade
- [ ] Take `pg_dump` immediately before upgrading (see [upgrade.md](upgrade.md#before-you-upgrade))
- [ ] Pin target version — Helm `--version X.Y.Z`, image tag `X.Y.Z`
- [ ] Confirm `/readyz` before and after upgrade
- [ ] Roll back via database restore + previous image if a migration commits and the new version fails

### Access and tokens

- [ ] Review admin users — disable accounts that no longer need access
- [ ] Audit API tokens — revoke unused or over-privileged tokens
- [ ] Rotate automation tokens on a defined schedule
- [ ] Confirm no plaintext secrets in ConfigMaps, git, or CI logs

### Monitoring

- [ ] Alert on `/readyz` failures and pod restarts
- [ ] Monitor `openlicensd_rate_limit_errors_total` when using postgres rate-limit backend
- [ ] Monitor database pool gauges (`openlicensd_db_pool_*`) under load
- [ ] Review audit log for unexpected admin mutations

### Security maintenance

- [ ] Upgrade OpenLicensd to the latest patch release
- [ ] Review [SECURITY.md](../SECURITY.md) supported versions policy
- [ ] Re-verify Cosign signatures when deploying new image digests

## Related

- [security-hardening.md](security-hardening.md) — detailed hardening guidance
- [deployment.md](deployment.md) — Helm, Docker, and binary install
- [backup-restore.md](backup-restore.md) — backup commands and restore verification
- [upgrade.md](upgrade.md) — upgrade procedure and v0.9.0 migration
- [scaling.md](scaling.md) — multi-replica and HA guidance
- [troubleshooting.md](troubleshooting.md) — common failure diagnosis
