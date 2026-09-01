# Deployment

OpenLicensd can be deployed as a Helm chart on Kubernetes, as a Docker container, or as a standalone binary.

## Helm (recommended)

Container images are published to `ghcr.io/alvarorg14/openlicensd` on release (image tags: `X.Y.Z`, `X.Y`, `latest`; git tags: `vX.Y.Z`). The Helm chart is published to `oci://ghcr.io/alvarorg14/charts/openlicensd`.

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

### Prometheus metrics

Metrics are served on a **separate listener** (`OPENLICENSD_METRICS_ADDR`, default `:9090`) at `GET /metrics`. This port is not routed by the chart Ingress — scrape it from inside the cluster or with a port-forward.

With [Prometheus Operator](https://prometheus-operator.dev/) (or kube-prometheus-stack) installed, enable the optional ServiceMonitor:

```yaml
serviceMonitor:
  enabled: true
  # Optional: labels for your Prometheus serviceMonitorSelector
  labels:
    release: kube-prometheus-stack
```

The chart also exposes a named `metrics` port on the Service when `config.metrics.enabled` is true (default), so in-cluster scrapes and port-forwards target the metrics listener without routing it through Ingress.

```bash
kubectl port-forward -n openlicensd svc/openlicensd 9090:9090
curl -s localhost:9090/metrics | head
```

See [metrics.md](metrics.md) for the full metric catalog.

### Multiple replicas

The default rate limit backend (`memory`) keeps buckets in process memory, so each replica enforces its own per-IP budget. When scaling beyond one pod (for example with an HPA), set `config.rateLimit.backend: postgres` so replicas share limits via PostgreSQL:

```yaml
config:
  rateLimit:
    backend: postgres
```

This adds a database write on each rate-limited request. See [scaling.md](scaling.md) for session stickiness, recommended replica counts, and other multi-replica caveats.

For production deployments with multiple replicas, enable the optional PodDisruptionBudget so voluntary disruptions (node drains, rolling upgrades) respect availability:

```yaml
replicaCount: 2
pdb:
  enabled: true
  maxUnavailable: 1
```

The default `maxUnavailable: 1` is safe even with a single replica. Avoid `minAvailable: 1` when `replicaCount: 1` — it blocks all voluntary disruptions. Use `minAvailable` only when running at least two replicas (for example `minAvailable: 1` with `replicaCount: 2` keeps one pod up during drains).

To autoscale based on CPU utilization, enable the optional HorizontalPodAutoscaler. The cluster must have [metrics-server](https://github.com/kubernetes-sigs/metrics-server) (or another metrics API provider) installed so the HPA can read resource usage:

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 80
pdb:
  enabled: true
  maxUnavailable: 1
config:
  rateLimit:
    backend: postgres
```

When `autoscaling.enabled` is true, the chart omits `spec.replicas` on the Deployment so the HPA owns replica count. Set `config.rateLimit.backend: postgres` whenever running more than one pod so rate limits are shared across replicas.

### Topology spread constraints

When running multiple replicas (fixed count or HPA), spread pods across failure domains so a single zone or node outage does not take out every replica. The chart exposes a pass-through `topologySpreadConstraints` list on the Deployment — supply full constraint objects; nothing is rendered when the list is empty.

Use `ScheduleAnyway` unless your nodes are reliably labeled and you need hard enforcement (`DoNotSchedule` can leave pods Pending on unlabeled nodes). The `labelSelector` must match the chart's pod labels (`app.kubernetes.io/name` and `app.kubernetes.io/instance`, where the instance is the Helm release name):

```yaml
replicaCount: 3
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: openlicensd
        app.kubernetes.io/instance: openlicensd
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: openlicensd
        app.kubernetes.io/instance: openlicensd
pdb:
  enabled: true
  maxUnavailable: 1
config:
  rateLimit:
    backend: postgres
```

Replace `openlicensd` in `app.kubernetes.io/instance` with your release name if it differs.

### Network policy

For clusters that enforce `NetworkPolicy` (requires a CNI plugin such as Calico or Cilium), enable the optional policy to restrict pod traffic:

```yaml
networkPolicy:
  enabled: true
```

With defaults (`allowExternal: true`, `allowExternalEgress: true`), the policy allows TCP ingress to the HTTP port and metrics port (when metrics are enabled) from any source, and allows all egress so external PostgreSQL, OIDC, and Harbor endpoints keep working.

Tighten ingress to only the ingress controller namespace:

```yaml
networkPolicy:
  enabled: true
  allowExternal: false
  extraIngress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
```

When locking down ingress, ensure kubelet/node traffic can still reach the pod for liveness and readiness probes — add an `extraIngress` rule for the node CIDR or namespace if your CNI requires it.

Tighten egress to only PostgreSQL (DNS is included automatically when `allowExternalEgress` is false):

```yaml
networkPolicy:
  enabled: true
  allowExternalEgress: false
  extraEgress:
    - to:
        - ipBlock:
            cidr: 10.0.0.0/8
      ports:
        - protocol: TCP
          port: 5432
```

Replace the CIDR with your database network. Add additional `extraEgress` rules for OIDC or Harbor when those features are enabled.

### Upgrade

Replace `X.Y.Z` with the target release version. Take a `pg_dump` before upgrading — see [upgrade.md](upgrade.md) for the full procedure (migrations, rolling updates, rollback).

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

[`docker-compose.stack.yml`](../docker-compose.stack.yml) exposes the full `.env.example` variable set with stack-safe defaults. To override settings, use a `.env.stack` file and `COMPOSE_ENV_FILES=.env.stack make stack-up`; do not copy `.env` or `.env.example` verbatim — keep `OPENLICENSD_DATABASE_URL` pointed at the `postgres` Compose service (the compose file sets this for you).

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

Backups are operator-owned — the Helm chart does not bundle a database. See [backup-restore.md](backup-restore.md) for `pg_dump`/`pg_restore` procedures and RPO/RTO guidance.

For local development, start PostgreSQL with Docker Compose:

```bash
make dev-db
```

## Health checks

OpenLicensd splits liveness and readiness the way Kubernetes expects:

| Endpoint | Purpose | Success | Failure |
|----------|---------|---------|---------|
| `GET /healthz` | Liveness — process answered HTTP | `200` | — |
| `GET /readyz` | Readiness — PostgreSQL reachable | `200` | `503` |

**Do not put the database on liveness.** If `/healthz` pinged Postgres, a brief database outage would restart every pod instead of only removing them from the Service endpoints. `/readyz` is the meaningful check: it runs `Store.Ping` with a 2-second timeout and returns `503 {"error":"database unavailable"}` when the pool cannot reach the server.

The Helm chart maps probes as follows:

| Kubernetes probe | Path |
|------------------|------|
| Liveness | `/healthz` |
| Readiness | `/readyz` |
| Startup | `/readyz` (allows migrations to finish before liveness begins) |

Successful probe requests are logged at `debug` when `OPENLICENSD_LOG_LEVEL=debug`; failed readiness checks (`503`) still emit a `warn` line at the default `info` level.

Verify locally:

```bash
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/healthz
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/readyz
```

## Releases

Publish a GitHub release to trigger cross-platform binaries, Docker images, and Helm chart packaging:

```bash
gh release create vX.Y.Z --generate-notes
```

Replace `vX.Y.Z` with the version tag (for example `v0.5.0`). The release workflow runs on `release: published`, not on tag push alone.

GoReleaser builds Linux amd64/arm64 binaries and pushes Docker images to `ghcr.io/alvarorg14/openlicensd` (image tags are semver without the `v` prefix, e.g. `0.5.0`). The Helm chart is packaged and pushed to `oci://ghcr.io/alvarorg14/charts` in a separate workflow job; `helm package --version/--app-version` stamps the chart from the tag. `charts/openlicensd/Chart.yaml` in git is a `0.0.0-dev` placeholder — use `--version X.Y.Z` when installing from OCI, or set `image.tag` when installing from source.

The release job also stamps `docs/openapi.yaml` `info.version` from the tag and attaches the stamped copy to the GitHub release. The stamp is publish-time only — git keeps the `0.0.0-dev` placeholder, and the workflow marks the file skip-worktree so GoReleaser still sees a clean worktree.

## Related

- [QUICKSTART.md](../QUICKSTART.md) — get running quickly
- [configuration.md](configuration.md) — all environment variables
- [upgrade.md](upgrade.md) — upgrade procedure and migration notes
- [backup-restore.md](backup-restore.md) — PostgreSQL backup and restore
- [scaling.md](scaling.md) — HA, scaling, and multi-replica caveats
- [troubleshooting.md](troubleshooting.md) — common failures: database, migrations, OIDC, Harbor
- [charts/openlicensd/README.md](../charts/openlicensd/README.md) — Helm chart reference
