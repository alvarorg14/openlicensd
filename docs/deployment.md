# Deployment

OpenLicensd can be deployed as a Helm chart on Kubernetes, as a Docker container, or as a standalone binary.

## Helm (recommended)

Container images are published to `ghcr.io/alvarorg14/openlicensd` on release (tags: `vX.Y.Z`, `X.Y`, `latest`). The Helm chart is published to `oci://ghcr.io/alvarorg14/charts/openlicensd`.

### Install

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

### Ingress and TLS

Enable Ingress in your values:

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: licenses.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: openlicensd-tls
      hosts:
        - licenses.example.com
```

### Harbor integration

Enable Harbor registry credentials:

```yaml
config:
  harbor:
    enabled: true
    url: https://harbor.example.com
    projects: myproject,another-project
    robotDurationDays: 1

secret:
  data:
    harborAdminUsername: admin
    harborAdminPassword: your-harbor-admin-password
```

See [harbor-registry-credentials.md](harbor-registry-credentials.md) for full setup.

### Verify

```bash
kubectl get pods -n openlicensd -l app.kubernetes.io/name=openlicensd
kubectl port-forward -n openlicensd svc/openlicensd 8080:8080
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
```

### Upgrade

```bash
helm upgrade openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --version X.Y.Z \
  --namespace openlicensd \
  --reuse-values
```

### Uninstall

```bash
helm uninstall openlicensd -n openlicensd
```

## Docker

### Docker Compose (evaluation)

For a self-contained local stack (PostgreSQL + OpenLicensd), use [docker-compose.stack.yml](../docker-compose.stack.yml):

```bash
make stack-up
```

Open http://localhost:8080 and sign in with `admin@example.com` / `admin`. This uses the default bootstrap credentials and is intended for evaluation only — change the password hash before any real use.

Stop with `make stack-down` (add `ARGS=-v` to drop the stack's database volume). The stack runs under a separate Compose project (`openlicensd-stack`) from the dev database (`docker-compose.yml`), so `make dev-db-reset` does not affect it.

### Single container

Pull the image:

```bash
docker pull ghcr.io/alvarorg14/openlicensd:latest
```

Run with required environment variables:

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

The production image is built from a multi-stage Dockerfile (Node 24 UI build → Go 1.26 compile → distroless non-root).

## Binary

Download a release binary from [GitHub Releases](https://github.com/alvarorg14/openlicensd/releases) or build from source:

```bash
make build
```

Run:

```bash
export OPENLICENSD_DATABASE_URL=postgres://user:pass@host:5432/openlicensd?sslmode=disable
export OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL=admin@example.com
export OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH=$(make hash-password PASSWORD=your-secure-password)
export OPENLICENSD_COOKIE_SECURE=false

./bin/openlicensd
```

The server serves the API and embedded UI on `OPENLICENSD_ADDR` (default `:8080`).

## PostgreSQL

OpenLicensd requires PostgreSQL 16+. Migrations run automatically on startup from `server/internal/store/migrations/`.

For local development, start PostgreSQL with Docker Compose:

```bash
make dev-db
```

## Health checks

| Endpoint | Purpose | Success | Failure |
|----------|---------|---------|---------|
| `GET /healthz` | Liveness | `200` | — |
| `GET /readyz` | Readiness (DB ping) | `200` | `503` |

The Helm chart configures Kubernetes probes against these endpoints.

## Releases

Publish a GitHub release to trigger cross-platform binaries, Docker images, and Helm chart packaging:

```bash
gh release create vX.Y.Z --generate-notes
```

Replace `vX.Y.Z` with the version tag (for example `v0.5.0`). The release workflow runs on `release: published`, not on tag push alone.

GoReleaser builds Linux amd64/arm64 binaries and pushes Docker images to `ghcr.io/alvarorg14/openlicensd` (image tags are semver without the `v` prefix, e.g. `0.5.0`). The Helm chart is packaged and pushed to `oci://ghcr.io/alvarorg14/charts` in a separate workflow job; `helm package --version/--app-version` stamps the chart from the tag. `charts/openlicensd/Chart.yaml` in git is a `0.0.0-dev` placeholder — use `--version X.Y.Z` when installing from OCI, or set `image.tag` when installing from source.

The release job also stamps `docs/openapi.yaml` `info.version` from the tag and attaches the spec to the GitHub release.

## Related

- [QUICKSTART.md](../QUICKSTART.md) — get running quickly
- [configuration.md](configuration.md) — all environment variables
- [charts/openlicensd/README.md](../charts/openlicensd/README.md) — Helm chart reference
