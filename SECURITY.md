# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |
| < 0.1   | :x:                |

The latest release is always recommended. Older 0.x releases may not receive backports unless the vulnerability is critical.

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

When deploying OpenLicensd:

1. **Secrets**: Store `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` and Harbor admin credentials in a secrets manager. Never commit secrets to version control.
2. **TLS**: Terminate TLS at your Ingress or reverse proxy. Do not expose the server over plain HTTP in production.
3. **Network policies**: Restrict network access to the admin UI and API where possible.
4. **Updates**: Keep OpenLicensd updated to the latest release.
5. **Password hashing**: Use `make hash-password` to generate bcrypt hashes. Do not store plaintext admin passwords.
6. **Harbor credentials**: Harbor admin credentials are the highest-value secret in a Harbor-enabled deployment. Use Kubernetes Secrets or External Secrets Operator.
7. **Browser security headers**: The server sets `Content-Security-Policy`, `X-Frame-Options`, and `X-Content-Type-Options` on all HTTP responses. When `OPENLICENSD_COOKIE_SECURE=true`, it also sets `Strict-Transport-Security`. Keep `OPENLICENSD_COOKIE_SECURE=false` for local HTTP development so browsers do not pin HSTS.

### Known Security Considerations

- **Public endpoints**: `/api/v1/validate`, `/api/v1/registry-credentials`, `/api/v1/auth/login`, and OIDC login/callback are unauthenticated. These endpoints are rate limited per client IP (token bucket). Configure `OPENLICENSD_TRUSTED_PROXIES` when running behind a reverse proxy so limits apply per client rather than per proxy. With the default `memory` backend, limits are per process — effective throughput scales with replica count. Set `OPENLICENSD_RATE_LIMIT_BACKEND=postgres` when running multiple replicas so all pods share one global per-IP budget. See [docs/scaling.md](docs/scaling.md).
- **License key storage**: Full license keys are never stored. Only SHA-256 hashes and a 5-character prefix are persisted.
- **Sessions**: Admin sessions use httpOnly cookies with CSRF protection on unsafe methods. Set `OPENLICENSD_COOKIE_SECURE=true` in production (also enables HSTS response headers).
- **API tokens**: Scoped Bearer tokens for automation are stored as SHA-256 hashes with a display prefix; the raw value is shown exactly once at creation. Revoke tokens from the admin UI or `PATCH /api/v1/api-tokens/{id}/revoke`. Token management requires an admin session — Bearer tokens cannot mint new tokens.
- **Harbor integration**: When enabled, anyone with a valid license key can obtain short-lived Harbor pull credentials. Robot accounts have pull-only access to configured projects.
- **TLS verification**: `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY=true` disables TLS certificate verification for Harbor. Use only in development or trusted internal networks.
- **Container security**: Production images use a distroless base and run as non-root (UID 65532) with a read-only root filesystem.
- **Debug mode**: `OPENLICENSD_HARBOR_DEBUG=true` logs Harbor API requests and may include error details in `502` responses. Disable in production.

### Security Updates

Security updates will be:

- Released as patch versions (e.g., 0.1.1, 0.1.2)
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
