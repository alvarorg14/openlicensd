# Quick Start

Get **OpenLicensd** running in minutes.

## Prerequisites

Choose one deployment path:

| Path | Requirements |
|------|-------------|
| **Helm** | Kubernetes cluster, [Helm 3](https://helm.sh/docs/intro/install/), [kubectl](https://kubernetes.io/docs/tasks/tools/) |
| **Docker Compose** | [Docker](https://docs.docker.com/get-docker/), [Docker Compose](https://docs.docker.com/compose/) |
| **Docker** | [Docker](https://docs.docker.com/get-docker/), a PostgreSQL instance |
| **Local dev** | Go 1.26+, Node.js 24+, Docker (for PostgreSQL) |

## Helm (production)

Container images are published to `ghcr.io/alvarorg14/openlicensd`. The Helm chart is published to `oci://ghcr.io/alvarorg14/charts/openlicensd`.

Replace `X.Y.Z` with the [latest release](https://github.com/alvarorg14/openlicensd/releases) version:

```bash
helm install openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --version X.Y.Z \
  --namespace openlicensd \
  --create-namespace \
  --set config.bootstrapAdmin.email=admin@example.com \
  --set secret.data.databaseUrl="postgres://user:pass@host:5432/openlicensd?sslmode=require" \
  --set secret.data.bootstrapAdminPasswordHash="$(make hash-password PASSWORD=yourpassword)"
```

Or install from the chart source:

```bash
helm install openlicensd ./charts/openlicensd \
  --namespace openlicensd \
  --create-namespace \
  -f my-values.yaml
```

### Verify (Helm)

```bash
kubectl get pods -n openlicensd -l app.kubernetes.io/name=openlicensd
kubectl port-forward -n openlicensd svc/openlicensd 8080:8080
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
```

Open http://localhost:8080 and sign in with your configured admin credentials.

## Docker Compose (fastest)

Run PostgreSQL and OpenLicensd together from the published image:

```bash
make stack-up
```

Open http://localhost:8080 and sign in with:

- Email: `admin@example.com`
- Password: `admin`

> **Warning:** The default bootstrap password is for evaluation only. Generate your own hash with `make hash-password PASSWORD=yourpassword` and set `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` before any real use (for example via a `.env.stack` file and `COMPOSE_ENV_FILES=.env.stack make stack-up`).

[`docker-compose.stack.yml`](docker-compose.stack.yml) exposes the full `.env.example` variable set with stack-safe defaults. If you use `.env.stack`, do not copy `.env` or `.env.example` verbatim — keep `OPENLICENSD_DATABASE_URL` pointed at the `postgres` Compose service (the compose file sets this for you).

### Verify (Docker Compose)

```bash
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
```

Stop the stack:

```bash
make stack-down
```

To remove the stack's database volume as well:

```bash
make stack-down ARGS=-v
```

## Docker

```bash
docker run -d \
  --name openlicensd \
  -p 8080:8080 \
  -e OPENLICENSD_DATABASE_URL="postgres://user:pass@host:5432/openlicensd?sslmode=disable" \
  -e OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL=admin@example.com \
  -e OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH='$2a$10$...' \
  -e OPENLICENSD_COOKIE_SECURE=true \
  ghcr.io/alvarorg14/openlicensd:latest
```

### Verify (Docker)

```bash
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
```

Open http://localhost:8080.

## Binary

Download from [GitHub Releases](https://github.com/alvarorg14/openlicensd/releases) or build from source:

```bash
make build
```

> **Note:** `make build` runs `make ui` first so the binary embeds the full admin UI. A plain `go build` embeds a placeholder page instead.

```bash
export OPENLICENSD_DATABASE_URL=postgres://user:pass@host:5432/openlicensd?sslmode=disable
export OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL=admin@example.com
export OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH=$(make hash-password PASSWORD=your-secure-password)
export OPENLICENSD_COOKIE_SECURE=false

./bin/openlicensd
```

Open http://localhost:8080.

## Local development

1. Start PostgreSQL:

```bash
make dev-db
```

If you are upgrading from a version without products/policies, reset the local database:

```bash
make dev-db-reset
```

2. Copy environment variables:

```bash
cp .env.example .env
```

If you regenerate `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH`, wrap the bcrypt value in single quotes in `.env` (e.g. `'$2a$10$...'`) so Docker Compose does not misinterpret `$` characters.

3. In one terminal, start the API server:

```bash
make dev-server
```

4. In another terminal, start the UI dev server:

```bash
make dev-ui
```

Open http://localhost:3000 and sign in with:

- Email: `admin@example.com`
- Password: `admin`

## Try the API

### Login

```bash
curl -s -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin"}'

CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')
```

### Create a product and policy

```bash
PRODUCT_ID=$(curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Acme Widget","code":"acme-widget"}' | jq -r .id)

POLICY_ID=$(curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d "{\"product_id\":\"$PRODUCT_ID\",\"name\":\"Perpetual\"}" | jq -r .id)
```

### Create a license

```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/licenses \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d "{\"label\":\"Test License\",\"product_id\":\"$PRODUCT_ID\",\"policy_id\":\"$POLICY_ID\"}" | jq
```

Save the `key` from the response — it is shown only once.

### Validate a license

```bash
curl -s -X POST http://localhost:8080/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{"key":"YOUR-KEY-HERE","product":"acme-widget"}' | jq
```

## Uninstall (Helm)

```bash
helm uninstall openlicensd -n openlicensd
```

## Next Steps

- Full configuration reference: [docs/configuration.md](docs/configuration.md)
- API documentation: [docs/api.md](docs/api.md) and [docs/openapi.yaml](docs/openapi.yaml)
- Harbor registry credentials: [docs/harbor-registry-credentials.md](docs/harbor-registry-credentials.md)
- Deployment guide: [docs/deployment.md](docs/deployment.md)
- Upgrade procedure: [docs/upgrade.md](docs/upgrade.md)
- PostgreSQL backup and restore: [docs/backup-restore.md](docs/backup-restore.md)
