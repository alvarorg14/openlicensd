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
| `OPENLICENSD_COOKIE_SECURE` | `true` | No | Set `Secure` flag on session cookies |

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
| `config.cookieSecure` | `OPENLICENSD_COOKIE_SECURE` |
| `config.bootstrapAdmin.email` | `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL` |
| `config.bootstrapAdmin.name` | `OPENLICENSD_BOOTSTRAP_ADMIN_NAME` |
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
| `secret.data.harborAdminUsername` | `OPENLICENSD_HARBOR_ADMIN_USERNAME` |
| `secret.data.harborAdminPassword` | `OPENLICENSD_HARBOR_ADMIN_PASSWORD` |

## Secret provisioning modes

### `create` (default)

Helm creates a Secret from `secret.data.*` values.

### `existing`

Set `secret.mode: existing` and `secret.existingSecret` to reference a pre-created Secret.

### `externalSecrets`

Set `secret.mode: externalSecrets` and configure `secret.externalSecrets.remoteRefs` to sync from an external secrets manager.
