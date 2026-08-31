# Configuration

OpenLicensd is configured entirely through environment variables. In Kubernetes, these are set via the Helm chart's `config`, `secret`, and `extraEnv` values.

## Environment variables

### Core

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_ADDR` | `:8080` | No | HTTP listen address |
| `OPENLICENSD_DATABASE_URL` | — | **Yes** | PostgreSQL connection URL |
| `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL` | — | Yes on empty DB | Email for the first admin user |
| `OPENLICENSD_BOOTSTRAP_ADMIN_NAME` | `Administrator` | No | Display name for bootstrap admin |
| `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` | — | Yes on empty DB | Bcrypt hash for bootstrap admin password |
| `OPENLICENSD_SESSION_TTL_HOURS` | `24` | No | Session lifetime in hours |
| `OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES` | `60` | No | Interval for deleting expired/revoked sessions (`0` disables) |
| `OPENLICENSD_COOKIE_SECURE` | `true` | No | Set `Secure` flag on session cookies; when `true`, also enables `Strict-Transport-Security` response headers |
| `OPENLICENSD_LOCAL_LOGIN_ENABLED` | `true` | No | Allow email/password login |

### Logging

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_LOG_LEVEL` | `info` | No | Log level: `debug`, `info`, `warn`, or `error` |
| `OPENLICENSD_LOG_FORMAT` | `json` | No | Log output format: `json` or `text` |

Logs are written to stderr. Each HTTP request emits a completion line with `request_id`, `method`, `path`, `status`, `bytes`, `duration_ms`, and `remote_addr`. Handler and integration logs reuse the same `request_id` for correlation.

Example JSON line:

```json
{"time":"2026-08-31T12:00:00.000Z","level":"INFO","msg":"request completed","request_id":"abc-123","method":"POST","path":"/api/v1/validate","status":200,"bytes":84,"duration_ms":12,"remote_addr":"127.0.0.1:12345"}
```

Common structured attributes: `request_id`, `client_ip`, `user_id`, `email`, `license_key_prefix`, `product_code`, `reason`, `err`.

| Level | Examples |
|-------|----------|
| `error` | Internal handler failures (`writeInternalError`), Harbor robot creation failures |
| `warn` | Failed logins, OIDC callback failures, rate-limit denials, best-effort store write failures |
| `info` | Request completion, successful logins, license validation outcomes, migrations, startup/shutdown |
| `debug` | Harbor API traffic when `OPENLICENSD_HARBOR_DEBUG=true` |

Login success and failure records include `email` and `client_ip`. Plan log retention in your aggregator accordingly — license keys are never logged (only `license_key_prefix`).

Set `OPENLICENSD_LOG_FORMAT=text` for human-readable local development output.

### Metrics

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_METRICS_ENABLED` | `true` | No | Enable Prometheus `/metrics` on a dedicated listener |
| `OPENLICENSD_METRICS_ADDR` | `:9090` | No | Metrics listen address (must differ from `OPENLICENSD_ADDR`) |

See [metrics.md](metrics.md) for the full metric catalog and scrape configuration.

### Database pool

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_DATABASE_MAX_CONNS` | `0` | No | Maximum pool connections; `0` uses pgx default (`max(4, NumCPU)`) |
| `OPENLICENSD_DATABASE_MIN_CONNS` | `0` | No | Minimum pool connections; `0` uses pgx default |
| `OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES` | `0` | No | Close idle connections after this many minutes; `0` uses pgx default (`30m`) |
| `OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS` | `0` | No | PostgreSQL `statement_timeout` in seconds; `0` leaves the server default |

The connection URL is parsed first (`pool_*` query parameters are supported). Non-zero env vars override URL pool settings. Effective pool limits appear in the `openlicensd_db_pool_*` Prometheus gauges — see [metrics.md](metrics.md).

### Rate limiting

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_TRUSTED_PROXIES` | — | No | Comma-separated trusted proxy IPs or CIDRs. When the direct peer matches, `X-Forwarded-For` is used to resolve the client IP |
| `OPENLICENSD_RATE_LIMIT_ENABLED` | `true` | No | Enable per-IP rate limiting on unauthenticated endpoints |
| `OPENLICENSD_RATE_LIMIT_PUBLIC_PER_MINUTE` | `600` | No | Sustained request rate for `/validate` and `/registry-credentials` |
| `OPENLICENSD_RATE_LIMIT_PUBLIC_BURST` | `60` | No | Burst capacity for public endpoints |
| `OPENLICENSD_RATE_LIMIT_LOGIN_PER_MINUTE` | `30` | No | Sustained request rate for `/auth/login` and OIDC login/callback |
| `OPENLICENSD_RATE_LIMIT_LOGIN_BURST` | `10` | No | Burst capacity for login endpoints |
| `OPENLICENSD_RATE_LIMIT_IDLE_MINUTES` | `10` | No | Minutes before an unused per-IP bucket is evicted from memory |

Limits are per process. With multiple replicas, effective throughput scales with replica count. Set `OPENLICENSD_TRUSTED_PROXIES` when running behind an ingress or load balancer.

### OIDC SSO (optional)

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_OIDC_ENABLED` | `false` | No | Enable OIDC SSO |
| `OPENLICENSD_OIDC_ISSUER_URL` | — | Yes when enabled | OIDC issuer URL |
| `OPENLICENSD_OIDC_CLIENT_ID` | — | Yes when enabled | OAuth client ID |
| `OPENLICENSD_OIDC_CLIENT_SECRET` | — | Yes when enabled | OAuth client secret |
| `OPENLICENSD_OIDC_REDIRECT_URL` | — | Yes when enabled | Callback URL registered with the IdP |
| `OPENLICENSD_OIDC_SCOPES` | `openid,profile,email` | No | Comma-separated scopes |
| `OPENLICENSD_OIDC_DEFAULT_ROLE` | `viewer` | No | Role for newly provisioned SSO users |
| `OPENLICENSD_OIDC_PROVIDER_NAME` | `SSO` | No | SSO button label on the login page |
| `OPENLICENSD_OIDC_ADMIN_EMAILS` | — | No | Comma-separated emails that receive `admin` on first SSO login |

See [oidc-sso.md](oidc-sso.md) for provider setup walkthroughs and troubleshooting.

### Harbor (optional)

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_HARBOR_ENABLED` | `false` | No | Enable Harbor registry credentials endpoint |
| `OPENLICENSD_HARBOR_URL` | — | Yes when enabled | Harbor base URL |
| `OPENLICENSD_HARBOR_ADMIN_USERNAME` | — | Yes when enabled | Harbor admin username for robot creation |
| `OPENLICENSD_HARBOR_ADMIN_PASSWORD` | — | Yes when enabled | Harbor admin password for robot creation |
| `OPENLICENSD_HARBOR_PROJECTS` | — | Yes when enabled | Comma-separated Harbor project namespaces |
| `OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS` | `1` | No | Robot account lifetime in days (minimum `1`) |
| `OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX` | `openlicensd` | No | Prefix for generated robot account names |
| `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY` | `false` | No | Skip TLS verification for Harbor |
| `OPENLICENSD_HARBOR_DEBUG` | `false` | No | Log Harbor API requests/responses; include error details in `502` responses |

See [harbor-registry-credentials.md](harbor-registry-credentials.md) for Harbor setup details.

## Generating secrets

### Bootstrap admin password hash

```bash
make hash-password PASSWORD=your-secure-password
```

Set the output as `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH`.

### Local development

```bash
cp .env.example .env
# Edit .env — set OPENLICENSD_COOKIE_SECURE=false for HTTP dev
make dev-db-reset   # required after schema changes
make dev-server
```

The defaults use a local PostgreSQL instance started by `make dev-db`.

## Helm values mapping

### `config` → environment variables

| Helm value | Environment variable |
|------------|---------------------|
| `config.addr` | `OPENLICENSD_ADDR` |
| `config.sessionTTLHours` | `OPENLICENSD_SESSION_TTL_HOURS` |
| `config.sessionCleanupIntervalMinutes` | `OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES` |
| `config.cookieSecure` | `OPENLICENSD_COOKIE_SECURE` |
| `config.localLoginEnabled` | `OPENLICENSD_LOCAL_LOGIN_ENABLED` |
| `config.log.format` | `OPENLICENSD_LOG_FORMAT` |
| `config.log.level` | `OPENLICENSD_LOG_LEVEL` |
| `config.metrics.addr` | `OPENLICENSD_METRICS_ADDR` |
| `config.metrics.enabled` | `OPENLICENSD_METRICS_ENABLED` |
| `config.database.maxConns` | `OPENLICENSD_DATABASE_MAX_CONNS` |
| `config.database.minConns` | `OPENLICENSD_DATABASE_MIN_CONNS` |
| `config.database.maxConnIdleMinutes` | `OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES` |
| `config.database.statementTimeoutSeconds` | `OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS` |
| `config.bootstrapAdmin.email` | `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL` |
| `config.bootstrapAdmin.name` | `OPENLICENSD_BOOTSTRAP_ADMIN_NAME` |
| `config.trustedProxies` | `OPENLICENSD_TRUSTED_PROXIES` |
| `config.rateLimit.enabled` | `OPENLICENSD_RATE_LIMIT_ENABLED` |
| `config.rateLimit.publicPerMinute` | `OPENLICENSD_RATE_LIMIT_PUBLIC_PER_MINUTE` |
| `config.rateLimit.publicBurst` | `OPENLICENSD_RATE_LIMIT_PUBLIC_BURST` |
| `config.rateLimit.loginPerMinute` | `OPENLICENSD_RATE_LIMIT_LOGIN_PER_MINUTE` |
| `config.rateLimit.loginBurst` | `OPENLICENSD_RATE_LIMIT_LOGIN_BURST` |
| `config.rateLimit.idleMinutes` | `OPENLICENSD_RATE_LIMIT_IDLE_MINUTES` |
| `config.oidc.enabled` | `OPENLICENSD_OIDC_ENABLED` |
| `config.oidc.issuerUrl` | `OPENLICENSD_OIDC_ISSUER_URL` |
| `config.oidc.clientId` | `OPENLICENSD_OIDC_CLIENT_ID` |
| `config.oidc.redirectUrl` | `OPENLICENSD_OIDC_REDIRECT_URL` |
| `config.oidc.scopes` | `OPENLICENSD_OIDC_SCOPES` |
| `config.oidc.defaultRole` | `OPENLICENSD_OIDC_DEFAULT_ROLE` |
| `config.oidc.providerName` | `OPENLICENSD_OIDC_PROVIDER_NAME` |
| `config.oidc.adminEmails` | `OPENLICENSD_OIDC_ADMIN_EMAILS` |
| `config.harbor.enabled` | `OPENLICENSD_HARBOR_ENABLED` |
| `config.harbor.url` | `OPENLICENSD_HARBOR_URL` |
| `config.harbor.projects` | `OPENLICENSD_HARBOR_PROJECTS` |
| `config.harbor.robotDurationDays` | `OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS` |
| `config.harbor.robotNamePrefix` | `OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX` |
| `config.harbor.insecureSkipVerify` | `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY` |
| `config.harbor.debug` | `OPENLICENSD_HARBOR_DEBUG` |

### `secret.data` → environment variables

| Helm value | Environment variable |
|------------|---------------------|
| `secret.data.databaseUrl` | `OPENLICENSD_DATABASE_URL` |
| `secret.data.bootstrapAdminPasswordHash` | `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` |
| `secret.data.oidcClientSecret` | `OPENLICENSD_OIDC_CLIENT_SECRET` |
| `secret.data.harborAdminUsername` | `OPENLICENSD_HARBOR_ADMIN_USERNAME` |
| `secret.data.harborAdminPassword` | `OPENLICENSD_HARBOR_ADMIN_PASSWORD` |

## Secret provisioning modes

### `create` (default)

Helm creates a Secret from `secret.data.*` values.

### `existing`

Set `secret.mode: existing` and `secret.existingSecret` to reference a pre-created Secret.

### `externalSecrets`

Set `secret.mode: externalSecrets` and configure `secret.externalSecrets.remoteRefs` to sync from an external secrets manager.
