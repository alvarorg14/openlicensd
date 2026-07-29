# Configuration

OpenLicensd is configured entirely through environment variables. In Kubernetes, these are set via the Helm chart's `config`, `secret`, and `extraEnv` values.

## Environment variables

### Core

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `OPENLICENSD_ADDR` | `:8080` | No | HTTP listen address |
| `OPENLICENSD_DATABASE_URL` | — | **Yes** | PostgreSQL connection URL |
| `OPENLICENSD_ADMIN_USER` | — | **Yes** | Admin username |
| `OPENLICENSD_ADMIN_PASSWORD_HASH` | — | **Yes** | Bcrypt hash of admin password |
| `OPENLICENSD_JWT_SECRET` | — | **Yes** | Secret for signing JWT tokens |

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

### Admin password hash

```bash
make hash-password PASSWORD=your-secure-password
```

Set the output as `OPENLICENSD_ADMIN_PASSWORD_HASH`.

> **Docker Compose:** Wrap bcrypt hashes in single quotes in `.env` (e.g. `'$2a$10$...'`) so `$` characters are not interpreted as variable interpolation.

### JWT secret

Generate a random secret:

```bash
openssl rand -hex 32
```

## Local development

Copy the example environment file:

```bash
cp .env.example .env
```

The defaults use a local PostgreSQL instance started by `make dev-db` with username/password `admin`.

## Helm values mapping

The Helm chart in `charts/openlicensd/` maps values to environment variables.

### Non-secret config (`config`)

| Helm value | Environment variable |
|------------|---------------------|
| `config.addr` | `OPENLICENSD_ADDR` |
| `config.adminUser` | `OPENLICENSD_ADMIN_USER` |
| `config.harbor.enabled` | `OPENLICENSD_HARBOR_ENABLED` |
| `config.harbor.url` | `OPENLICENSD_HARBOR_URL` |
| `config.harbor.projects` | `OPENLICENSD_HARBOR_PROJECTS` |
| `config.harbor.robotDurationDays` | `OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS` |
| `config.harbor.robotNamePrefix` | `OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX` |
| `config.harbor.insecureSkipVerify` | `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY` |
| `config.harbor.debug` | `OPENLICENSD_HARBOR_DEBUG` |

### Secrets (`secret`)

| Helm value | Environment variable |
|------------|---------------------|
| `secret.data.databaseUrl` | `OPENLICENSD_DATABASE_URL` |
| `secret.data.adminPasswordHash` | `OPENLICENSD_ADMIN_PASSWORD_HASH` |
| `secret.data.jwtSecret` | `OPENLICENSD_JWT_SECRET` |
| `secret.data.harborAdminUsername` | `OPENLICENSD_HARBOR_ADMIN_USERNAME` |
| `secret.data.harborAdminPassword` | `OPENLICENSD_HARBOR_ADMIN_PASSWORD` |

### Secret provisioning modes

| Mode | Description |
|------|-------------|
| `create` | Chart creates a Kubernetes Secret from `secret.data` values |
| `existing` | Reference an existing Secret via `secret.existingSecret` |
| `externalSecrets` | Use External Secrets Operator via `secret.externalSecrets` |

Example with External Secrets:

```yaml
secret:
  mode: externalSecrets
  externalSecrets:
    refreshInterval: 1h
    secretStoreRef:
      name: my-secret-store
      kind: ClusterSecretStore
    remoteRefs:
      - secretKey: OPENLICENSD_DATABASE_URL
        remoteRef:
          key: openlicensd/database
          property: url
```

## Related

- [deployment.md](deployment.md) — Helm install and upgrade
- [harbor-registry-credentials.md](harbor-registry-credentials.md) — Harbor configuration
- [.env.example](../.env.example) — local development template
