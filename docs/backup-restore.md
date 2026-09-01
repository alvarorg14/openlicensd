# PostgreSQL backup and restore

OpenLicensd stores all durable application state in PostgreSQL. The server binary is otherwise stateless: sessions, rate-limit buckets, and Harbor robot accounts are either ephemeral or external.

This runbook covers logical backup and restore with `pg_dump` and `pg_restore`. It applies to production deployments where you bring your own database (Helm, Docker, or binary) and to the bundled PostgreSQL instances used by local Docker Compose.

> **Warning:** OpenLicensd does not publish an RPO/RTO service-level agreement. The guidance below helps operators choose a backup strategy; it is not a product guarantee.

## What this covers

| Deployment | Who owns backups |
|------------|------------------|
| **Helm / Kubernetes** | You. The chart requires an external PostgreSQL database and does not bundle one. |
| **Docker Compose stack** (`make stack-up`) | You. Postgres runs in Compose for evaluation only. |
| **Local dev** (`make dev-db`) | You, if you care about keeping local data. |
| **Binary / single container** | You. Point `OPENLICENSD_DATABASE_URL` at a database you manage. |

OpenLicensd has no built-in backup API, scheduled dump job, or WAL-archiving integration. Use your PostgreSQL tooling or cloud provider's managed backup features.

## What to back up

### Durable (restore these)

| Table | Contents |
|-------|----------|
| `products` | Product catalog |
| `policies` | Expiration rules and default max activations |
| `licenses` | Key hashes, prefixes, expiry, revoke state, validation counters |
| `license_machines` | Machine activations and fingerprints |
| `users` | Admin accounts, bcrypt password hashes, OIDC linkage |
| `schema_migrations` | Applied migration versions (needed for correct startup) |

### Ephemeral (optional in backups)

| Table | Behavior after restore |
|-------|--------------------------|
| `sessions` | Users must sign in again. Expired sessions are cleaned up automatically. |
| `rate_limit_buckets` | Per-IP token buckets reset. Only used when `OPENLICENSD_RATE_LIMIT_BACKEND=postgres`. |

### Not stored in PostgreSQL

| Data | Notes |
|------|-------|
| **Raw license keys** | Only the SHA-256 hash and prefix are stored. Keys shown once at creation cannot be recovered from a backup. |
| **Harbor robot credentials** | Short-lived robots created on demand; re-created on the next `/registry-credentials` call. |
| **In-memory rate limits** | Default backend (`memory`) keeps state in process only. |

## Prerequisites

- **PostgreSQL 16+** on the restore target. OpenLicensd requires 16+; local Compose images use PostgreSQL 18.
- **`pgcrypto` extension** — created by migration `001_licenses.sql`. The restore target must allow `CREATE EXTENSION` (or have `pgcrypto` pre-installed).
- **`pg_dump` / `pg_restore`** client tools. Match the client major version to the server where possible.
- **Connection URL** — the same database referenced by `OPENLICENSD_DATABASE_URL` (see [configuration.md](configuration.md)).

## RPO and RTO guidance

**Recovery Point Objective (RPO)** — how much data you can afford to lose:

| Strategy | Typical RPO | When to use |
|----------|-------------|-------------|
| Daily `pg_dump` | Up to 24 hours | Eval stacks, small installs, infrequent license changes |
| Hourly `pg_dump` (cron) | Up to 1 hour | Self-managed Postgres without WAL archiving |
| Managed PITR (RDS, Cloud SQL, Azure, etc.) | Minutes | Production with frequent license issuance or admin activity |
| Dump before upgrades | Near-zero for that event | Always take a dump immediately before upgrading OpenLicensd or PostgreSQL |

**Recovery Time Objective (RTO)** — how long until the service is back:

RTO is dominated by provisioning or restoring PostgreSQL, updating `OPENLICENSD_DATABASE_URL` if the host changed, and waiting for OpenLicensd pods to pass `/readyz`. The schema is small (UUID primary keys, no large objects), so restore time is usually minutes, not hours, once Postgres is available.

For tighter RPO in production, enable point-in-time recovery on your managed PostgreSQL service rather than relying on logical dumps alone. Vendor-specific console steps are outside this runbook.

## Backup with pg_dump

Use **custom format** (`-Fc`) for flexible restore and compression. Add `--no-owner --no-acl` so restores work under a different database role.

### From a connection URL

```bash
export OPENLICENSD_DATABASE_URL='postgres://user:pass@host:5432/openlicensd?sslmode=require'

pg_dump "$OPENLICENSD_DATABASE_URL" \
  --format=custom \
  --no-owner \
  --no-acl \
  --file="openlicensd-$(date +%Y%m%d-%H%M%S).dump"
```

For an extra-consistent snapshot on a live database, add `--serializable-deferrable` (may wait briefly for concurrent transactions to finish).

### Local dev database (`make dev-db`)

Compose project name: `openlicensd`.

```bash
docker compose -p openlicensd exec -T postgres \
  pg_dump -U openlicensd -d openlicensd \
  --format=custom \
  --no-owner \
  --no-acl \
  > "openlicensd-dev-$(date +%Y%m%d-%H%M%S).dump"
```

### Docker Compose stack (`make stack-up`)

Compose project name: `openlicensd-stack`. Stop the app first if you need a quiescent database; for eval data a live dump is usually sufficient.

```bash
docker compose -p openlicensd-stack -f docker-compose.stack.yml exec -T postgres \
  pg_dump -U openlicensd -d openlicensd \
  --format=custom \
  --no-owner \
  --no-acl \
  > "openlicensd-stack-$(date +%Y%m%d-%H%M%S).dump"
```

### Scheduling

Automate dumps with cron, Kubernetes CronJob, or your cloud provider's backup service. Store dump files off-host (object storage, backup vault) and encrypt them at rest. Dump files contain license metadata, user password hashes, and session token hashes.

## Restore with pg_restore

### Before you start

1. **Stop OpenLicensd** (scale Deployment to 0 replicas, or stop the binary/container) so nothing writes to the database during restore.
2. **Target an empty database** on PostgreSQL 16+ (recommended). Creating a fresh database avoids conflicting objects.
3. **Ensure `pgcrypto` is available** on the target instance.

### Restore into a new database

```bash
# Create an empty database (as a superuser or role with CREATEDB)
createdb -h host -U admin openlicensd_restored

pg_restore \
  --dbname="postgres://user:pass@host:5432/openlicensd_restored?sslmode=require" \
  --no-owner \
  --no-acl \
  --verbose \
  openlicensd-20260101-120000.dump
```

Update `OPENLICENSD_DATABASE_URL` (or Helm `secret.data.databaseUrl`) to point at the restored database, then start OpenLicensd.

Migrations run automatically on startup. If `schema_migrations` was restored with the dump, already-applied migrations are skipped.

### Restore over an existing database (disaster recovery)

Only when you intend to replace all data:

```bash
pg_restore \
  --dbname="$OPENLICENSD_DATABASE_URL" \
  --clean \
  --if-exists \
  --no-owner \
  --no-acl \
  --verbose \
  openlicensd-20260101-120000.dump
```

`--clean` drops objects before recreating them. Use with care.

### Docker Compose stack

```bash
# Stop the app, keep Postgres running
docker compose -p openlicensd-stack -f docker-compose.stack.yml stop openlicensd

docker compose -p openlicensd-stack -f docker-compose.stack.yml exec -T postgres \
  pg_restore -U openlicensd -d openlicensd --clean --if-exists --no-owner --no-acl \
  < openlicensd-stack-20260101-120000.dump

docker compose -p openlicensd-stack -f docker-compose.stack.yml start openlicensd
```

### Verify after restore

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://licenses.example.com/readyz
# Expect 200
```

Sign in to the admin UI and confirm products, policies, and licenses are present. Validation counters and `last_validated_at` are restored with the `licenses` table.

> **Warning:** Restoring the database does **not** recover raw license keys that were only shown at creation time. Distribute keys through your own secure channel or re-issue licenses if keys are lost.

## After restore

| Topic | Behavior |
|-------|----------|
| **Sessions** | All admin sessions are invalid. Users sign in again (local password or OIDC). |
| **Bootstrap admin** | `OPENLICENSD_BOOTSTRAP_ADMIN_*` only seeds a user when the `users` table is empty. A restored `users` table keeps existing accounts. |
| **Rate limiting** | `rate_limit_buckets` may be empty or stale; per-IP limits reset harmlessly. |
| **Harbor robots** | Previously issued credentials may be invalid. Clients request new credentials via `/registry-credentials`. |

## What not to do

- Do not run `make dev-db-reset` or `make stack-down ARGS=-v` against a volume you intend to keep — both drop the `postgres_data` volume.
- Do not restore a dump from a newer PostgreSQL major into an older server.
- Do not leave OpenLicensd running against a half-restored database.

## Related

- [upgrade.md](upgrade.md) — upgrade procedure and when to take a dump before upgrading
- [deployment.md](deployment.md) — Helm, Docker, binary install, and PostgreSQL requirements
- [configuration.md](configuration.md) — `OPENLICENSD_DATABASE_URL` and pool settings
- [QUICKSTART.md](../QUICKSTART.md) — get running quickly
- [charts/openlicensd/README.md](../charts/openlicensd/README.md) — Helm chart reference
