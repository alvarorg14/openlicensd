# Security hardening

This guide helps operators harden OpenLicensd for production. It synthesizes deployment, configuration, and integration guidance into one runbook. For a go-live checklist you can copy into your own procedures, see [production-checklist.md](production-checklist.md).

> **Note:** This page covers operational security. To report a vulnerability, follow [SECURITY.md](../SECURITY.md) — do **not** open a public GitHub issue.

## TLS and transport

**What to do:** Terminate TLS at your Ingress or reverse proxy. Serve the admin UI and API only over HTTPS in production. Set `OPENLICENSD_COOKIE_SECURE=true` (the default).

**Why it matters:** Plain HTTP exposes session cookies, CSRF tokens, and admin credentials to network observers. When `OPENLICENSD_COOKIE_SECURE=true`, OpenLicensd also sends `Strict-Transport-Security` response headers so browsers pin HTTPS.

| Setting | Production | Local HTTP dev |
|---------|------------|----------------|
| `OPENLICENSD_COOKIE_SECURE` | `true` | `false` |
| Helm `config.cookieSecure` | `true` | `false` |

Keep `OPENLICENSD_COOKIE_SECURE=false` only for local HTTP development. Browsers will not store secure cookies or HSTS over plain HTTP.

See [deployment.md](deployment.md#ingress-and-tls) for Ingress TLS examples and [configuration.md](configuration.md) for the full variable reference.

## Session cookies and CSRF

**What to do:** Rely on the built-in session model. Do not disable CSRF protection or strip security headers at the proxy.

**Why it matters:** Admin sessions use httpOnly cookies (`openlicensd_session`) with CSRF double-submit protection on unsafe HTTP methods (`POST`, `PATCH`, `DELETE`). OIDC login uses short-lived state, nonce, and PKCE cookies that must survive the redirect round trip — this requires HTTPS when `OPENLICENSD_COOKIE_SECURE=true`.

OpenLicensd sets `Content-Security-Policy`, `X-Frame-Options`, and `X-Content-Type-Options` on all HTTP responses. The embedded admin UI bundles icons at build time because the CSP blocks runtime Iconify fetches.

See [architecture.md](architecture.md) for the request path and [oidc-sso.md](oidc-sso.md) for OIDC cookie behavior.

## Trusted proxies

**What to do:** When OpenLicensd runs behind an ingress controller, load balancer, or reverse proxy, set `OPENLICENSD_TRUSTED_PROXIES` (or Helm `config.trustedProxies`) to the proxy IP addresses or CIDRs.

**Why it matters:** Public and login rate limits key buckets by client IP. Without trusted-proxy configuration, every request appears to come from the proxy address — one shared bucket for all clients, and ineffective per-client throttling.

Example:

```bash
OPENLICENSD_TRUSTED_PROXIES=10.0.0.0/8,172.16.0.0/12
```

See [configuration.md](configuration.md) and [scaling.md](scaling.md) for multi-replica rate-limit guidance.

## Bootstrap admin

**What to do:** Seed the first admin only on a fresh database. Store the bcrypt password hash in a secrets manager or Kubernetes Secret — never commit plaintext passwords or hashes to version control.

**Why it matters:** `OPENLICENSD_BOOTSTRAP_ADMIN_*` runs only when the `users` table is empty. On a restored or existing database, bootstrap env vars are ignored. Default Compose stack credentials (`admin@example.com` / `admin`) are for evaluation only.

Generate a hash:

```bash
make hash-password PASSWORD='your-strong-password'
```

Set:

```bash
OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL=admin@example.com
OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH='$2a$...'
```

After first login, change the password through the admin UI or API. Bootstrap seeding is idempotent across concurrent replicas (PostgreSQL advisory lock).

See [configuration.md](configuration.md) and [troubleshooting.md](troubleshooting.md#bootstrap-admin-failed).

## OIDC single sign-on

**What to do:** Register the exact redirect URI with your identity provider. Store the client secret in a Secret. Consider SSO-only deployments for production.

**Why it matters:** A mismatched redirect URI is the most common OIDC misconfiguration. OIDC state cookies require HTTPS when `OPENLICENSD_COOKIE_SECURE=true` — `state_missing` errors often indicate HTTP access or cookie blocking.

| Hardening step | Setting |
|----------------|---------|
| Exact redirect URI | `https://<host>/api/v1/auth/oidc/callback` must match `OPENLICENSD_OIDC_REDIRECT_URL` byte for byte |
| Verified email | IdP must issue `email_verified: true` in the ID token before OpenLicensd links or creates users |
| SSO-only | `OPENLICENSD_LOCAL_LOGIN_ENABLED=false` |
| Initial admins via SSO | `OPENLICENSD_OIDC_ADMIN_EMAILS=admin@example.com` |
| Client secret | Helm `secret.data.oidcClientSecret` or env var — not in ConfigMap |

Roles are managed locally in OpenLicensd, not synced from group claims. Review the **Users** page after onboarding.

See [oidc-sso.md](oidc-sso.md) for provider walkthroughs and configuration reference.

## API token hygiene

**What to do:** Issue scoped tokens with the minimum role required. Store raw token values in a secrets manager immediately after creation. Revoke compromised tokens promptly.

**Why it matters:** API tokens authenticate as `Authorization: Bearer <token>` without CSRF. The raw value is shown exactly once at creation; only a SHA-256 hash and display prefix are stored. Token management endpoints require an admin **session** — a Bearer token cannot mint or revoke other tokens.

| Practice | Detail |
|----------|--------|
| Least privilege | Use `viewer` or `operator` when `admin` is not required |
| Rotation | Create a replacement token, update automation, revoke the old token |
| Compromise response | Revoke via admin UI or `PATCH /api/v1/api-tokens/{id}/revoke`; audit the audit log |
| Expiry | Set `expires_at` when creating tokens for time-bound automation |

Authenticated admin routes are rate limited per user ID or API token ID (defaults: 300/min sustained, burst 60).

See [api.md](api.md#6-api-tokens-for-automation) and [SECURITY.md](../SECURITY.md#known-security-considerations).

## Network policies

**What to do:** Enable the Helm NetworkPolicy when your cluster CNI supports it. Start with defaults, then tighten ingress and egress to your environment.

**Why it matters:** NetworkPolicy restricts which sources can reach OpenLicensd pods. Default chart settings allow broad ingress and egress so external PostgreSQL, OIDC, and Harbor keep working out of the box.

Minimal enable:

```yaml
networkPolicy:
  enabled: true
```

Tighten ingress to your ingress controller namespace and egress to your database CIDR — see [deployment.md](deployment.md#network-policy) for full examples. When locking down ingress, ensure kubelet probes can still reach the pod.

## Metrics exposure

**What to do:** Scrape Prometheus metrics from inside the cluster only. Do not route the metrics listener through your public Ingress.

**Why it matters:** Metrics are served on a **separate listener** (`OPENLICENSD_METRICS_ADDR`, default `:9090`) at `GET /metrics`. The chart Ingress routes only the API/UI port. Exposing `/metrics` publicly leaks operational detail (request rates, pool health, validation outcomes).

| Setting | Recommendation |
|---------|----------------|
| `OPENLICENSD_METRICS_ENABLED` | `true` for observability |
| `OPENLICENSD_METRICS_ADDR` | `:9090` (must differ from `OPENLICENSD_ADDR`) |
| Ingress | Do not add a path for port 9090 |
| Scraping | ServiceMonitor or in-cluster Prometheus scrape of the `metrics` Service port |

See [metrics.md](metrics.md) and [deployment.md](deployment.md#prometheus-metrics).

## Database connection security

**What to do:** Use TLS to PostgreSQL. Store the password in a Secret. Configure discrete connection variables (v0.9.0+).

**Why it matters:** The database holds password hashes, API token hashes, license metadata, and audit events. A leaked connection string grants full read/write access.

```bash
OPENLICENSD_DATABASE_HOST=postgres.example.com
OPENLICENSD_DATABASE_PORT=5432
OPENLICENSD_DATABASE_USER=openlicensd
OPENLICENSD_DATABASE_PASSWORD=<from-secret>
OPENLICENSD_DATABASE_NAME=openlicensd
OPENLICENSD_DATABASE_SSLMODE=require
```

Use `verify-full` when your CA trust chain supports it. Helm maps these to `config.database.*` and `secret.data.databasePassword`.

See [configuration.md](configuration.md) and [upgrade.md](upgrade.md#upgrading-to-v090--discrete-database-configuration) if migrating from older releases.

## Rate limiting (multi-replica)

**What to do:** With more than one replica, set `OPENLICENSD_RATE_LIMIT_BACKEND=postgres` so all pods share one global rate-limit budget per scope and client key.

**Why it matters:** The default `memory` backend keeps token buckets in process memory. Effective public and login limits scale with replica count — a client can send N × the configured rate by hitting different pods.

Also set trusted proxies (above) so per-IP limits reflect real client addresses. Monitor `openlicensd_rate_limit_errors_total` when using the postgres backend.

See [scaling.md](scaling.md) for replica counts, PDB, and HPA guidance.

## Harbor integration and image trust

**What to do:** Treat Harbor admin credentials as the highest-value secret in a Harbor-enabled deployment. Disable debug and TLS skip-verify flags in production. Verify container images before deploy.

**Why it matters:** When Harbor is enabled, anyone with a valid license key can obtain short-lived pull credentials via `/api/v1/registry-credentials`. Harbor admin credentials grant robot creation across configured projects.

| Setting | Production |
|---------|------------|
| `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY` | `false` |
| `OPENLICENSD_HARBOR_DEBUG` | `false` |
| Harbor admin password | Kubernetes Secret or External Secrets Operator |

Verify release images from GHCR:

```bash
cosign verify \
  --certificate-identity-regexp 'https://github.com/alvarorg14/openlicensd/\.github/workflows/release\.yml@.*' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  ghcr.io/alvarorg14/openlicensd@sha256:<digest>
```

See [harbor-registry-credentials.md](harbor-registry-credentials.md), [deployment.md](deployment.md#releases), and [SECURITY.md](../SECURITY.md).

## Public endpoints

These endpoints are intentionally unauthenticated and rate limited per client IP (or per proxy without trusted-proxy configuration):

- `POST /api/v1/validate`
- `POST /api/v1/registry-credentials` (when Harbor is enabled)
- `POST /api/v1/auth/login`
- OIDC login and callback routes

Plan WAF or ingress rules accordingly. License keys are stored as SHA-256 hashes only — raw keys cannot be recovered from the database.

## Related

- [production-checklist.md](production-checklist.md) — pre-production, go-live, and ongoing checklists
- [deployment.md](deployment.md) — Helm, Docker, and binary install
- [backup-restore.md](backup-restore.md) — backup strategy and restore verification
- [upgrade.md](upgrade.md) — upgrade procedure and rollback path
- [troubleshooting.md](troubleshooting.md) — common failures (OIDC, database, Harbor)
- [SECURITY.md](../SECURITY.md) — vulnerability reporting and supported versions
