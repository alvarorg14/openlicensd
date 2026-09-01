# Upgrading OpenLicensd

OpenLicensd applies PostgreSQL schema migrations automatically on startup. There is no down-migration path — once a migration commits, the only way to undo a schema change is to restore a pre-upgrade database backup and run the previous binary or image.

This runbook covers upgrade steps for Helm, Docker Compose, single-container, and binary deployments, how migrations behave during rolling updates, and how to roll back when something goes wrong.

> **Warning:** OpenLicensd does not publish an upgrade SLA. The guidance below helps operators plan upgrades; it is not a product guarantee.

## What this covers

| Deployment | Upgrade method |
|------------|----------------|
| **Helm / Kubernetes** | `helm upgrade` with a pinned chart version; rolling pod replacement |
| **Docker Compose stack** (`make stack-up`) | Pull a pinned image tag and recreate the app container |
| **Single container** | Stop, replace the image, restart with the same `OPENLICENSD_DATABASE_URL` |
| **Binary** | Replace the executable and restart the process |

For HA, replica-count, and rate-limiting caveats during multi-replica deployments, see the scaling documentation (issue [#99](https://github.com/alvarorg14/openlicensd/issues/99) — not covered here.

## Before you upgrade

1. **Read the release notes** — open the [GitHub release](https://github.com/alvarorg14/openlicensd/releases) for the target version. Check the **Breaking Changes** section first. Pull requests labeled `breaking-change` land there and may require config, API client, or operational changes before you deploy.
2. **Pin a version** — do not rely on floating tags in production:
   - Helm: `--version X.Y.Z` (OCI chart from `oci://ghcr.io/alvarorg14/charts/openlicensd`)
   - Docker image: `ghcr.io/alvarorg14/openlicensd:X.Y.Z` (not `latest`)
   - Docker Compose stack: `OPENLICENSD_IMAGE_TAG=X.Y.Z make stack-up`
3. **Take a backup** — always run `pg_dump` immediately before upgrading. See [backup-restore.md](backup-restore.md) for commands and format. This is your rollback path if a migration commits and you need to revert.
4. **Confirm current health** — `GET /readyz` should return `200` before you start. A database that is already unreachable will block the new version from starting too.

## How migrations work

Migrations live in `server/internal/store/migrations/` as numbered SQL files (for example `009_rate_limit_buckets.sql`). They are embedded in the binary and applied automatically when the server connects to PostgreSQL.

| Behavior | Detail |
|----------|--------|
| **Forward-only** | There are no down-migration files. Applied migrations are recorded in `schema_migrations` and never re-run. |
| **Ordering** | Files are sorted by filename and applied in sequence. |
| **Transactions** | Each file runs inside a single transaction. On SQL error, that file rolls back and startup aborts. |
| **Concurrency** | Multiple replicas starting at once serialize on a PostgreSQL advisory lock (`pg_advisory_lock`). Only one process applies pending migrations at a time. |
| **Readiness** | `store.Migrate` runs before the HTTP server listens. `/readyz` returns `503` until migrations finish and the database ping succeeds. |
| **Logging** | At `info` level, look for `migration applied` (per file), `migrations complete` (batch), or `schema up to date` (nothing pending), followed by `database connected`. |

Example of a destructive forward step: migration `008_drop_archived_at.sql` drops columns with no automatic undo.

The Helm chart maps the Kubernetes **startup** probe to `/readyz` (5 s period, 30 failure threshold ≈ 150 s) so new pods stay unready until migrations complete. Old pods continue serving until the new pod passes readiness.

## Upgrade steps

### Helm (Kubernetes)

Replace `X.Y.Z` with the target release version:

```bash
helm upgrade openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --version X.Y.Z \
  --namespace openlicensd \
  --reuse-values
```

Wait for the rollout:

```bash
kubectl rollout status deployment/openlicensd -n openlicensd
kubectl get pods -n openlicensd -l app.kubernetes.io/name=openlicensd
```

Verify:

```bash
kubectl port-forward -n openlicensd svc/openlicensd 8080:8080
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/readyz
# Expect 200
```

Sign in to the admin UI and spot-check licenses, products, and policies.

`--reuse-values` keeps your existing ConfigMap and Secret settings. To change configuration at the same time, pass `-f my-values.yaml` or `--set` flags alongside `--reuse-values` as needed.

### Docker Compose stack

Pin the image tag and recreate the app container (Postgres data is preserved in the `postgres_data` volume):

```bash
OPENLICENSD_IMAGE_TAG=X.Y.Z make stack-up
```

Verify:

```bash
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/readyz
```

### Single container

```bash
docker stop openlicensd
docker pull ghcr.io/alvarorg14/openlicensd:X.Y.Z
docker rm openlicensd
docker run -d \
  --name openlicensd \
  -p 8080:8080 \
  -e OPENLICENSD_DATABASE_URL="postgres://user:pass@host:5432/openlicensd?sslmode=require" \
  # ... same env vars as before ...
  ghcr.io/alvarorg14/openlicensd:X.Y.Z
```

Use the same environment variables (especially `OPENLICENSD_DATABASE_URL`) as the previous container.

### Binary

```bash
# Stop the running process (systemd, supervisor, etc.)
cp bin/openlicensd bin/openlicensd.bak
# Replace bin/openlicensd with the new release binary
./bin/openlicensd
```

Or build from source at the release tag:

```bash
git fetch --tags
git checkout vX.Y.Z
make build
```

## Mixed-version rolling updates

During a Kubernetes rolling update, old and new pods may run simultaneously:

1. A new pod acquires the advisory lock and applies pending migrations.
2. Once a migration commits, **all** pods — including ones still running the old binary — query the new schema.
3. **Additive** migrations (new tables, new nullable columns) are usually safe across mixed versions.
4. **Destructive** migrations (DROP, RENAME, NOT NULL on existing rows) can break old replicas until they are replaced.

For releases with destructive schema changes noted in the release notes:

- Scale to a single replica before upgrading, **or**
- Accept a brief window where old pods may error until the rollout completes.

After the rollout, confirm all pods are on the new image and `/readyz` is `200` on each endpoint you expose.

## Rollback

Rollback strategy depends on whether migrations committed.

### New version never became ready

If the new pod or process failed during migration (startup probe timeout, crash loop, SQL error in logs) and **no** `migration applied` line appeared for the target version:

1. Redeploy the previous chart version, image tag, or binary.
2. No database restore is required — the failed migration rolled back inside its transaction.

Helm example:

```bash
helm rollback openlicensd -n openlicensd
```

### Migrations committed

If `schema_migrations` contains versions from the new release, **redeploying the old binary alone is not enough**. The old binary expects the old schema.

1. **Stop OpenLicensd** (scale Deployment to 0, or stop the container/binary).
2. **Restore the pre-upgrade dump** — follow [backup-restore.md](backup-restore.md). Use the `pg_dump` you took immediately before upgrading.
3. **Start the previous version** (pinned image tag or binary matching the restored schema).

> **Warning:** Restoring a dump replaces all database state as of backup time. License validations, admin sessions, and other changes made after the dump are lost.

`helm rollback` without a database restore only helps when the new release never applied migrations.

## Breaking changes

OpenLicensd uses GitHub labels and Release Drafter to surface breaking changes:

| Signal | Meaning |
|--------|---------|
| PR label `breaking-change` | Included in the release notes **Breaking Changes** section; Release Drafter bumps the major version component |
| PR label `deprecations` | Included in the **Deprecations** section |
| [docs/openapi.yaml](openapi.yaml) | HTTP API contract (stamped at publish time on releases) |

**Before v1.0**, these signals communicate intent but do not constitute a formal API stability or deprecation policy. That policy is planned for the v1.0 milestone ([#70](https://github.com/alvarorg14/openlicensd/issues/70)).

When a release lists breaking changes, read each item and verify:

- New or renamed environment variables ([configuration.md](configuration.md))
- HTTP API request/response changes ([api.md](api.md), [openapi.yaml](openapi.yaml))
- Database schema changes that affect mixed-version rollouts (see above)
- Helm values changes ([charts/openlicensd/README.md](../charts/openlicensd/README.md))

## What not to do

- Do not skip the pre-upgrade `pg_dump` — it is the only supported rollback path after migrations commit.
- Do not edit migration files that have already been applied in any environment. Add a new numbered file instead (see [CONTRIBUTING.md](../CONTRIBUTING.md)).
- Do not run a newer OpenLicensd binary against a database restored from a dump taken while a **newer** version was running, then downgrade the binary without restoring — `schema_migrations` and the schema must match the binary.
- Do not leave old replicas serving after a destructive migration if the release notes warn about schema incompatibility.
- Do not use `make dev-db-reset` or `make stack-down ARGS=-v` when you intend to keep data — both drop the Postgres volume.

## Related

- [backup-restore.md](backup-restore.md) — `pg_dump`/`pg_restore` and RPO/RTO guidance
- [deployment.md](deployment.md) — Helm, Docker, binary install, and health probes
- [configuration.md](configuration.md) — environment variables and Helm values
- [architecture.md](architecture.md) — components and data model
- [QUICKSTART.md](../QUICKSTART.md) — get running quickly
- [charts/openlicensd/README.md](../charts/openlicensd/README.md) — Helm chart reference
