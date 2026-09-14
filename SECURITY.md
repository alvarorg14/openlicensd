# Security Policy

## Supported Versions

OpenLicensd is volunteer-maintained. There is **no SLA** for security patches — the table below describes which release lines are eligible for fixes, not when a fix will ship. For general support expectations, see [SUPPORT.md](SUPPORT.md). For API compatibility (what changes in a patch vs minor vs major), see [COMPATIBILITY.md](COMPATIBILITY.md).

### GitHub policy snapshot

| Version | Supported |
| ------- | --------- |
| 1.x (when published) | :white_check_mark: — latest 1.x release |
| 0.x | :white_check_mark: until `v1.0.0` is published; EOL after that |
| < 0.1 | :x: |

The latest release on each supported line is always recommended.

### Until `v1.0.0` is published

- **Server / Helm** — the [latest `0.x` release](https://github.com/alvarorg14/openlicensd/releases) (`vX.Y.Z` tags). Helm chart `X.Y.Z` is stamped from the same tag.
- **Go SDK** — the latest `sdk/go/v0.x` release, versioned independently.
- **Older 0.x minors** — no dedicated backport train; maintainers may patch an older minor at their discretion for critical issues only.

### From `v1.0.0` (server) / `sdk/go/v1.0.0` (SDK)

| Line | Security patches | Bug fixes | Notes |
|------|------------------|-----------|-------|
| **Latest 1.x** (`v1.x.y`, Helm `X.Y.Z` from the same tag) | Yes | Yes | Recommended production line. Patches land on the latest 1.x patch of the latest 1.x minor. |
| **Older 1.x minors** (e.g. stay on `1.0` after `1.1`) | No dedicated backport train | No | Upgrade the minor — 1.x is backward compatible per [COMPATIBILITY.md](COMPATIBILITY.md). |
| **0.x** | No | No | **EOL** on the day `v1.0.0` is published. Historical tags remain downloadable. |
| **Previous major after a new major** (e.g. `1.x` after `v2.0.0`) | Security-only for **12 months** from the new major's publish date | No | Then EOL. The exact end date is announced in this file and the new major's release notes. |

The **Go SDK** follows the same table on its own tag series (`sdk/go/v1.x` when `sdk/go/v1.0.0` exists; `sdk/go/v0.x` EOL on that day). SDK vs server API pairing is documented in [docs/sdk/go.md](docs/sdk/go.md).

**Helm** is not a separate support line — chart version tracks the server release.

**What “v1 LTS” means:** the entire `1.x` major is the long-lived line (API stable per COMPATIBILITY.md, security and bugfix patches on the latest 1.x) until `v2.0.0`, then 12 months of security-only patches on `1.x`. It is **not** a freeze of `1.0.0` forever, and it is **not** a commercial SLA.

### End-of-life

- **0.x** — EOL when `v1.0.0` is published. Upgrade via [docs/upgrade.md](docs/upgrade.md) before that date.
- **Previous major** — when a new major is published (e.g. `v2.0.0`), the prior major receives security-only patches for 12 months, then EOL.
- **EOL releases** — tags and container images remain on GitHub/GHCR; they receive no further patches.

## Reporting a Vulnerability

We take the security of OpenLicensd seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please do NOT:

- Open a public GitHub issue
- Discuss the vulnerability in public forums
- Share the vulnerability with others until it has been resolved

### Please DO:

1. **Open a [private security advisory](https://github.com/alvarorg14/openlicensd/security/advisories/new)** with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if you have one)

2. **Include the following information**:
   - Affected component(s)
   - Attack vector
   - Privileges required
   - User interaction required
   - CVSS score (if you can calculate it)

3. **Allow us 90 days** to address the vulnerability before public disclosure

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 7 days
- **Updates**: We will provide regular updates on the status of the vulnerability
- **Resolution**: We will work to resolve the issue as quickly as possible
- **Credit**: With your permission, we will credit you in our security advisories

### Security Best Practices

When deploying OpenLicensd, follow the operator runbooks:

- **[docs/security-hardening.md](docs/security-hardening.md)** — TLS, cookies, trusted proxies, bootstrap admin, OIDC, API tokens, network policies, metrics exposure, database connection security, and multi-replica rate limiting
- **[docs/production-checklist.md](docs/production-checklist.md)** — pre-production, go-live, and ongoing operational checklists (including backup verification and upgrade path)

Quick reference:

1. **Secrets**: Store `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH`, database passwords, OIDC client secrets, and Harbor admin credentials in a secrets manager. Never commit secrets to version control.
2. **TLS**: Terminate TLS at your Ingress or reverse proxy. Set `OPENLICENSD_COOKIE_SECURE=true` in production.
3. **Updates**: Keep OpenLicensd updated to the latest release. See [docs/upgrade.md](docs/upgrade.md).
4. **Password hashing**: Use `make hash-password` to generate bcrypt hashes. Do not store plaintext admin passwords.
5. **Verify container images**: When deploying from GHCR, verify Cosign signatures and GitHub Artifact Attestations. See [docs/deployment.md](docs/deployment.md#releases).

### Known Security Considerations

- **Public endpoints**: `/api/v1/validate`, `/api/v1/registry-credentials`, `/api/v1/auth/login`, and OIDC login/callback are unauthenticated. These endpoints are rate limited per client IP (token bucket). Configure `OPENLICENSD_TRUSTED_PROXIES` when running behind a reverse proxy so limits apply per client rather than per proxy. With the default `memory` backend, limits are per process — effective throughput scales with replica count. Set `OPENLICENSD_RATE_LIMIT_BACKEND=postgres` when running multiple replicas so all pods share one global budget per scope and bucket key. See [docs/scaling.md](docs/scaling.md).
- **Authenticated admin API**: Session and Bearer requests to admin routes are rate limited per user ID or API token ID (defaults: 300/min, burst 60). A leaked token or compromised session cannot hammer the admin API without bound. Revoke compromised tokens and sessions promptly.
- **License key storage**: Full license keys are never stored. Only SHA-256 hashes and a 5-character prefix are persisted.
- **Sessions**: Admin sessions use httpOnly cookies with CSRF protection on unsafe methods. Set `OPENLICENSD_COOKIE_SECURE=true` in production (also enables HSTS response headers).
- **API tokens**: Scoped Bearer tokens for automation are stored as SHA-256 hashes with a display prefix; the raw value is shown exactly once at creation. Revoke tokens from the admin UI or `PATCH /api/v1/api-tokens/{id}/revoke`. Token management requires an admin session — Bearer tokens cannot mint new tokens.
- **Harbor integration**: When enabled, anyone with a valid license key can obtain short-lived Harbor pull credentials. Robot accounts have pull-only access to configured projects.
- **TLS verification**: `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY=true` disables TLS certificate verification for Harbor. Use only in development or trusted internal networks.
- **Container security**: Production images use a distroless base and run as non-root (UID 65532) with a read-only root filesystem.
- **Debug mode**: `OPENLICENSD_HARBOR_DEBUG=true` logs Harbor API requests and may include error details in `502` responses. Disable in production.

### Security Updates

Security updates will be:

- Released as patch versions (e.g., `1.0.1`, `1.1.3`)
- Announced via GitHub releases
- Tagged with `security` label where applicable

### Responsible Disclosure Timeline

- **Day 0**: Vulnerability reported
- **Day 1-2**: Acknowledgment and initial assessment
- **Day 3-7**: Detailed analysis and fix development
- **Day 8-30**: Testing and validation
- **Day 31-60**: Release preparation
- **Day 61-90**: Public disclosure (if not fixed earlier)

### Contact

For security-related issues, please open a [private security advisory](https://github.com/alvarorg14/openlicensd/security/advisories/new).

Thank you for helping keep OpenLicensd and its users safe!
