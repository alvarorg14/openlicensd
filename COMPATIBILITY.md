# API Compatibility and Deprecation Policy

Starting with **v1.0.0**, OpenLicensd commits to [Semantic Versioning 2.0](https://semver.org/spec/v2.0.0.html) for the covered surfaces below. This document defines what that promise includes, what it excludes, how deprecations ship, and how breaking changes are labeled and released.

For which versions receive security patches and general support expectations, see [SUPPORT.md](SUPPORT.md) and [SECURITY.md](SECURITY.md). For upgrade steps when a release lists breaking changes, see [docs/upgrade.md](docs/upgrade.md).

## Versioning model

Server releases use `vX.Y.Z` tags. The [Go SDK](https://github.com/alvarorg14/openlicensd/releases?q=sdk%2Fgo) is versioned independently (`sdk/go/vX.Y.Z`). The Helm chart OCI version is stamped from the server tag at publish time.

From v1.0.0 onward:

| Release | When |
|---------|------|
| **Patch** (`v1.0.x`) | Bug fixes, security patches, documentation, CI, and dependency updates that do not change covered contracts |
| **Minor** (`v1.x.0`) | Additive features; deprecations (old behavior keeps working through all of 1.x) |
| **Major** (`v2.0.0`) | Any removal or incompatible change to a covered surface |

Deprecated behavior introduced in a minor release **remains functional through all of 1.x** and is removed only in the next major release.

**Security exception:** When a vulnerability has no compatible fix, a patch release may change behavior in a way that would otherwise require a major bump. Such changes are called out in the release notes and [SECURITY.md](SECURITY.md). Which release lines receive patches is defined in [SECURITY.md — Supported Versions](SECURITY.md#supported-versions); this document defines API shape only.

## Covered surfaces

These are part of the v1 stability promise. The HTTP contract is defined in [docs/openapi.yaml](docs/openapi.yaml); prose in [docs/api.md](docs/api.md) must not contradict it.

### HTTP `/api/v1`

- Paths, methods, status codes, request/response JSON schemas, documented query parameters, and the role-based access matrix
- **Pagination envelope:** `{items, page, page_size, total, total_pages}`
- **Timestamps:** RFC3339 UTC strings at second precision
- **Partial PATCH:** omitted keys are left unchanged (true partial PATCH since v0.9.0)
- **Frozen validation dual envelope:**
  - `POST /api/v1/validate` — business outcomes always return HTTP **200** with `{valid, reason?, ...}`
  - `POST /api/v1/registry-credentials` — invalid licenses return HTTP **403** with `{error: "<ValidationReason>"}`
  - Shared `ValidationReason` codes are pinned in OpenAPI
- **License key format:** Crockford Base32 `XXXXX-XXXXX-XXXXX-XXXXX-XXXXX`; validation accepts case- and dash-insensitive variants

### Authentication wire format

- Session cookies: `openlicensd_session`, `openlicensd_csrf`
- CSRF header: `X-CSRF-Token` (required on session-based unsafe methods)
- API tokens: `Authorization: Bearer olsd_...`

### Health probes

- `GET /healthz` — liveness (no dependency checks)
- `GET /readyz` — readiness (PostgreSQL ping; `503` when unavailable)

### Configuration and deployment

- Documented `OPENLICENSD_*` environment variable **names**, types, and documented default **meanings** (see [docs/configuration.md](docs/configuration.md))
- Documented Helm values keys in [charts/openlicensd](charts/openlicensd) (`values.schema.json` rejects unknown keys)

### Metrics

- Documented Prometheus metric names and label keys in [docs/metrics.md](docs/metrics.md) (`openlicensd_*` prefix)
- New metric series and new label **values** are additive
- The `reason` label on `openlicensd_license_validations_total` stays aligned with `ValidationReason`

## Explicitly not covered

- **Admin UI** layout, copy, routes, and styling
- **PostgreSQL schema** as a public API (forward-only migrations; operators follow [docs/upgrade.md](docs/upgrade.md))
- **Log line text** (adding JSON log fields is fine)
- **Human-readable `{error: "..."}` message strings**, except where OpenAPI pins a specific code (for example `forbidden`, `request body too large`, registry-credentials `ValidationReason`)
- **JSON object key order** — clients must ignore unknown fields
- **Server Go internals** and the `github.com/alvarorg14/openlicensd/server` module
- **Rate-limit numeric capacity** as a performance SLA (env var names and the fail-open default *are* covered)
- **Which OS binaries are published** — documented in [docs/deployment.md](docs/deployment.md#platforms) (Linux amd64/arm64 only; expanding the matrix later is additive, not an API change)
- **Which versions receive security patches** — see [#126](https://github.com/alvarorg14/openlicensd/issues/126)

## Breaking vs additive changes

### Breaking (requires a major release)

- Remove or rename an `/api/v1` path, method, documented field, status code, or `ValidationReason` code
- Change cookie/header names or the Bearer token scheme
- Remove or rename a documented environment variable or Helm values key
- Change a documented default (including `OPENLICENSD_RATE_LIMIT_FAIL_OPEN=true`)
- Tighten a documented limit so previously accepted requests are rejected (for example lowering the 1000-machine activation ceiling)
- Rename or remove a documented metric name or label key
- Change `/validate` from HTTP 200 with a `valid` envelope to a non-200 for business-invalid licenses

### Additive (minor release)

- New endpoints, optional JSON fields, or new documented `reason` codes
- New environment variables or Helm keys with safe defaults
- New metrics or additional label values
- Exposing `licenses.metadata` over HTTP ([#130](https://github.com/alvarorg14/openlicensd/issues/130))
- Raising the 1000-machine activation ceiling

## Deprecation and breaking-change process

OpenLicensd maps changes to release notes through GitHub PR labels and [Release Drafter](https://github.com/release-drafter/release-drafter):

| PR label | Release notes section | SemVer bump |
|----------|----------------------|-------------|
| `deprecations` | **Deprecations** | Minor |
| `breaking-change` | **Breaking Changes** | Major |

Contributors add exactly one policy label per PR (see [CONTRIBUTING.md](CONTRIBUTING.md#pull-requests)).

### Deprecating behavior (minor)

1. Prefer additive design — introduce the replacement before deprecating the old name.
2. Label the PR `deprecations`.
3. Document the deprecation in [CHANGELOG.md](CHANGELOG.md) and mark the old name in docs/OpenAPI.
4. Keep the old behavior working through all of 1.x.

### Removing behavior (major)

1. Open only on a major release branch or when targeting `v2.0.0`.
2. Label the PR `breaking-change`.
3. Add a **Breaking Changes** entry to [CHANGELOG.md](CHANGELOG.md) with a migration path.
4. Add operator notes to [docs/upgrade.md](docs/upgrade.md) when the change affects deployments.
5. Fill the **Breaking Changes** section in the PR template.

## Related artifacts

| Artifact | Tag format | Compatibility scope |
|----------|------------|---------------------|
| Server binary / container | `vX.Y.Z` | Covered surfaces in this document |
| Helm chart (OCI) | `X.Y.Z` (stamped from server tag) | Documented values keys |
| Go SDK | `sdk/go/vX.Y.Z` | Exported Go API in its own SemVer series; HTTP `/api/v1` rules are this document |
| OpenAPI spec | `info.version` stamped at publish | HTTP contract; git keeps `0.0.0-dev` placeholder |

The Go SDK compatibility matrix (which server versions a given SDK release supports) is maintained in [docs/sdk/go.md](docs/sdk/go.md).

## v1 freeze notes

These are intentional 1.x behaviors, not bugs:

### `licenses.metadata` is schema-only

Migration `015_license_metadata.sql` added a nullable `metadata JSONB` column on `licenses`. It is **not** exposed on license HTTP request or response DTOs. Do not read or write this column as a supported interface. Exposing it over the API is planned for Post-1.0 ([#130](https://github.com/alvarorg14/openlicensd/issues/130)) and would be an **additive** (minor) change.

### Unlimited activations cap at 1000 machines

When `max_activations` is `null` (documented as unlimited), the server still enforces a hard ceiling of **1000 active machine fingerprints per license**. Beyond that, `/validate` returns `reason: "activation_limit"` and omits `max_activations` in the response. Raising this ceiling is a minor change; lowering it is breaking.

### Postgres rate-limit default is fail-open

`OPENLICENSD_RATE_LIMIT_FAIL_OPEN` defaults to `true`. When `OPENLICENSD_RATE_LIMIT_BACKEND=postgres`, bucket store errors allow the request (fail-open) unless the operator sets `false` (fail-closed with `429`). Changing this default to fail-closed in a future release would be **breaking**. See [docs/configuration.md](docs/configuration.md) and [docs/scaling.md](docs/scaling.md).

## Pre-1.0 history

Releases before **v1.0.0** did not follow this policy. Do not treat 0.x tags as bound by these rules.

Notable pre-freeze changes that were labeled `enhancement` rather than `breaking-change`:

| Change | Release | Notes |
|--------|---------|-------|
| Removed `OPENLICENSD_DATABASE_URL` and Helm `secret.data.databaseUrl` | v0.9.0 ([#280](https://github.com/alvarorg14/openlicensd/pull/280)) | Breaking for operators; originally filed under Enhancements in [CHANGELOG.md](CHANGELOG.md) — corrected retroactively |
| OpenAPI path parameter `{machineId}` → `{machine_id}` | v0.9.0 ([#257](https://github.com/alvarorg14/openlicensd/pull/257)) | Breaking for OpenAPI-generated clients that bind the param name; raw URL shape unchanged |

When upgrading from 0.x to 1.x, read [docs/upgrade.md](docs/upgrade.md) and the [CHANGELOG](CHANGELOG.md) for each intermediate release.

## Related documentation

- [docs/api.md](docs/api.md) — authentication, curl examples, frozen validation contract
- [docs/openapi.yaml](docs/openapi.yaml) — OpenAPI 3.1 specification
- [docs/upgrade.md](docs/upgrade.md) — upgrade procedure and breaking-change checklist
- [CONTRIBUTING.md](CONTRIBUTING.md) — PR labels and release workflow
- [SUPPORT.md](SUPPORT.md) — support channels and version guidance
- [SECURITY.md](SECURITY.md) — vulnerability reporting and supported versions
