# HA, scaling, and multi-replica caveats

OpenLicensd is designed to run behind a load balancer with multiple replicas. The server binary is otherwise stateless: durable application data lives in PostgreSQL, and the only multi-replica caveat operators must configure is the rate limit backend.

This runbook covers what is shared across replicas, whether session stickiness is required, how rate limiting behaves, recommended replica counts, and how to size PostgreSQL connections. For Helm install knobs (HPA, PDB, topology spread, NetworkPolicy), see [deployment.md](deployment.md). For rolling upgrades and mixed-version rollouts, see [upgrade.md](upgrade.md).

## What this covers

| Deployment | Scaling model |
|------------|---------------|
| **Helm / Kubernetes** | `replicaCount` or HPA; round-robin Service with no session affinity |
| **Docker Compose stack** (`make stack-up`) | Single app container by default; not intended for multi-replica HA |
| **Binary / single container** | Run multiple processes behind your own load balancer |

## Session stickiness

**OpenLicensd does not require session stickiness.** Any replica can serve any request.

| Concern | Storage | Sticky? |
|---------|---------|---------|
| Admin sessions | SHA-256 hash in PostgreSQL `sessions` table; token in `openlicensd_session` cookie | **No** — any replica validates via the shared database |
| CSRF | Double-submit cookie (`openlicensd_csrf`) + `X-CSRF-Token` header; not stored server-side | **No** |
| OIDC state / nonce / PKCE | Browser cookies (`openlicensd_oidc_*`, 10-minute TTL); session created in Postgres after callback | **No** — login and callback can hit different pods |
| Login lockout | PostgreSQL `users.failed_login_attempts` / `locked_until` | **No** |

The Helm chart Service and Ingress do not set `sessionAffinity`. Do not add sticky sessions unless you have an unrelated operational reason — the application does not need them.

## Shared vs per-replica state

| Component | Shared across replicas? | Notes |
|-----------|-------------------------|-------|
| Products, policies, licenses, machines, users | **Yes** (PostgreSQL) | Source of truth for all replicas |
| Sessions | **Yes** (PostgreSQL) | Expired sessions cleaned up by each replica (idempotent DELETE) |
| Rate limits (`memory` backend) | **No** | Per-process token buckets; effective limit × replica count |
| Rate limits (`postgres` backend) | **Yes** (PostgreSQL `rate_limit_buckets`) | One global per-IP budget; see below |
| Prometheus metrics | **No** | Per-process counters and gauges; scrape every pod |
| Harbor robot accounts | External (Harbor API) | Short-lived; re-created on demand |
| Schema migrations | Serialized (PostgreSQL advisory lock) | Only one replica applies pending migrations at a time |

## Recommended replica count

| Profile | Replicas | When to use |
|---------|----------|-------------|
| **Evaluation / small install** | `1` | Helm default; simplest ops; in-memory rate limits are fine |
| **High availability** | `2` or more | Survive a single pod or node failure; enable PDB |
| **Autoscaling** | HPA `minReplicas: 2`, `maxReplicas` sized to your cluster and database | Variable load; requires postgres rate-limit backend |

Guidelines:

1. **Use at least two replicas** when you need HA. Enable the PodDisruptionBudget (`pdb.enabled: true`) so voluntary disruptions (node drains, rolling upgrades) keep at least one pod available.
2. **Set `config.rateLimit.backend: postgres`** whenever more than one pod runs. With the default `memory` backend, each replica enforces its own per-IP budget — a client can send N× the configured limit across N pods.
3. **Size PostgreSQL `max_connections`** for `(pool max per pod) × (replica count)` plus headroom for admin connections and other services. When `OPENLICENSD_DATABASE_MAX_CONNS=0`, pgx defaults to `max(4, runtime.NumCPU)` per pod.
4. **Set `OPENLICENSD_TRUSTED_PROXIES`** (or `config.trustedProxies` in Helm) when traffic passes through an ingress or load balancer so rate limits key on the real client IP, not the proxy.

> **Note:** OpenLicensd does not publish an HA service-level agreement. The guidance below helps operators plan deployments; it is not a product guarantee.

## Rate limiter behavior

Rate limiting applies only to **unauthenticated** endpoints. Authenticated admin API routes, health probes, and static UI assets are not rate limited.

| Scope | Routes | Default sustained rate | Default burst |
|-------|--------|------------------------|---------------|
| **Public** | `POST /api/v1/validate`, `POST /api/v1/registry-credentials` (when Harbor enabled) | 600/min | 60 |
| **Login** | `POST /api/v1/auth/login`, `GET /api/v1/auth/oidc/login`, `GET /api/v1/auth/oidc/callback` | 30/min | 10 |

Each client IP gets an independent token bucket. Denied requests return HTTP `429` with a `Retry-After` header.

### Backends

| Backend | Env / Helm | Behavior |
|---------|------------|----------|
| **`memory`** (default) | `OPENLICENSD_RATE_LIMIT_BACKEND=memory` | Buckets live in process memory. Each replica enforces its own budget. Unused buckets are evicted after `OPENLICENSD_RATE_LIMIT_IDLE_MINUTES` (default 10). |
| **`postgres`** | `OPENLICENSD_RATE_LIMIT_BACKEND=postgres` | Buckets stored in PostgreSQL `rate_limit_buckets`. All replicas share one global per-IP budget. Adds a database write on each rate-limited request. On backend errors, requests are **allowed** (fail-open); monitor `openlicensd_rate_limit_errors_total` — see [metrics.md](metrics.md). |

Example: with `OPENLICENSD_RATE_LIMIT_PUBLIC_PER_MINUTE=600` and three replicas using the `memory` backend, a single client IP can sustain up to ~1800 requests/minute across the cluster. Switch to `postgres` to enforce 600/minute globally.

Full variable reference: [configuration.md](configuration.md#rate-limiting).

## PostgreSQL connection budget

Each OpenLicensd pod maintains its own connection pool. Total database connections scale linearly with replica count.

| Variable | Default | Effect |
|----------|---------|--------|
| `OPENLICENSD_DATABASE_MAX_CONNS` | `0` (pgx default: `max(4, NumCPU)`) | Maximum connections per pod |
| `OPENLICENSD_DATABASE_MIN_CONNS` | `0` | Minimum idle connections per pod |
| `OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES` | `0` (pgx default ~30 min) | Close idle connections after this duration |
| `OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS` | `0` | Server-side `statement_timeout` |

Pool gauges (`openlicensd_db_pool_*`) are exported per pod. When sizing Postgres, leave headroom beyond `(max conns per pod) × (replicas)`.

## Health probes and load balancing

| Endpoint | Purpose | Database check? |
|----------|---------|-----------------|
| `GET /healthz` | Liveness — process answered HTTP | **No** |
| `GET /readyz` | Readiness — PostgreSQL reachable | **Yes** (2-second timeout) |

**Do not put the database on liveness.** A transient Postgres outage should remove pods from the Service endpoints, not restart every replica. The Helm chart maps liveness to `/healthz` and readiness/startup to `/readyz`.

During startup, `/readyz` returns `503` until migrations finish. Old pods continue serving until new pods pass readiness. See [upgrade.md](upgrade.md) for mixed-version rollout caveats.

## Prometheus metrics

Metrics are served on a **separate listener** (`OPENLICENSD_METRICS_ADDR`, default `:9090`) at `GET /metrics`. Counters such as `openlicensd_license_validations_total` are per-process — scrape every pod or aggregate in Prometheus. Do not assume a single replica's counter represents cluster-wide traffic.

Enable the optional ServiceMonitor in Helm when using Prometheus Operator — see [deployment.md](deployment.md#prometheus-metrics) and [metrics.md](metrics.md).

## Helm recipe for HA

Minimal values for a two-replica HA deployment:

```yaml
replicaCount: 2

pdb:
  enabled: true
  maxUnavailable: 1

config:
  rateLimit:
    backend: postgres
  trustedProxies: "10.0.0.0/8"  # replace with your ingress/proxy CIDR
```

For autoscaling, topology spread, and NetworkPolicy examples, see [deployment.md](deployment.md#multiple-replicas).

## Related

- [deployment.md](deployment.md) — Helm install, Ingress, HPA, PDB, topology spread, NetworkPolicy
- [upgrade.md](upgrade.md) — rolling updates, migrations, mixed-version rollouts
- [backup-restore.md](backup-restore.md) — PostgreSQL backup and restore
- [configuration.md](configuration.md) — environment variables and Helm value mapping
- [metrics.md](metrics.md) — Prometheus metric catalog
- [oidc-sso.md](oidc-sso.md) — OIDC cookie flow (HA-safe across replicas)
- [charts/openlicensd/README.md](../charts/openlicensd/README.md) — Helm chart reference
