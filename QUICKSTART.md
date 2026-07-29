# Quick Start

Get **OpenLicensd** running in minutes.

## Prerequisites

Choose one deployment path:

| Path | Requirements |
|------|-------------|
| **Helm** | Kubernetes cluster, [Helm 3](https://helm.sh/docs/intro/install/), [kubectl](https://kubernetes.io/docs/tasks/tools/) |
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
  --set secret.data.databaseUrl="postgres://user:pass@host:5432/openlicensd?sslmode=require" \
  --set secret.data.adminPasswordHash="$(make hash-password PASSWORD=yourpassword)" \
  --set secret.data.jwtSecret="$(openssl rand -hex 32)"
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

## Docker

```bash
docker run -d \
  --name openlicensd \
  -p 8080:8080 \
  -e OPENLICENSD_DATABASE_URL="postgres://user:pass@host:5432/openlicensd?sslmode=disable" \
  -e OPENLICENSD_ADMIN_USER=admin \
  -e OPENLICENSD_ADMIN_PASSWORD_HASH='$2a$10$...' \
  -e OPENLICENSD_JWT_SECRET="$(openssl rand -hex 32)" \
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

```bash
export OPENLICENSD_DATABASE_URL=postgres://user:pass@host:5432/openlicensd?sslmode=disable
export OPENLICENSD_ADMIN_USER=admin
export OPENLICENSD_ADMIN_PASSWORD_HASH=$(make hash-password PASSWORD=your-secure-password)
export OPENLICENSD_JWT_SECRET=$(openssl rand -hex 32)

./bin/openlicensd
```

Open http://localhost:8080.

## Local development

1. Start PostgreSQL:

```bash
make dev-db
```

2. Copy environment variables:

```bash
cp .env.example .env
```

If you regenerate `OPENLICENSD_ADMIN_PASSWORD_HASH`, wrap the bcrypt value in single quotes in `.env` (e.g. `'$2a$10$...'`) so Docker Compose does not misinterpret `$` characters.

3. In one terminal, start the API server:

```bash
make dev-server
```

4. In another terminal, start the UI dev server:

```bash
make dev-ui
```

Open http://localhost:3000 and sign in with:

- Username: `admin`
- Password: `admin`

## Try the API

### Login

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r .token)
```

### Create a license

```bash
curl -s -X POST http://localhost:8080/api/v1/licenses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"label":"Test License"}' | jq
```

Save the `key` from the response — it is shown only once.

### Validate a license

```bash
curl -s -X POST http://localhost:8080/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{"key":"YOUR-KEY-HERE"}' | jq
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
