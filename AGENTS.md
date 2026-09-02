# AGENTS.md - AI Assistant Guidelines

This document provides context and guidelines for AI coding assistants working on the OpenLicensd project.

## Project Overview

**OpenLicensd** is an open source license server for creating, managing, and validating license keys. It ships as a single Go binary with an embedded Nuxt admin UI, backed by PostgreSQL.

### Key Purpose

- Generate and manage human-readable license keys (Crockford Base32 format)
- Provide a public validation endpoint for client applications
- Track license usage (validation count and last validated time)
- Enforce max concurrent machine activations per license key (policy default, per-license override)
- Optionally issue short-lived Harbor registry credentials to licensed clients
- Serve an admin UI for license management

## Architecture

### Backend (Go)

- **Location**: `server/cmd/openlicensd/` (main entry), `server/internal/` (core logic)
- **Module**: `github.com/alvarorg14/openlicensd/server`
- **Go version**: 1.26+
- **Key dependencies**: `go-chi/chi`, `golang-jwt/jwt`, `jackc/pgx`, `google/uuid`

### Frontend (Nuxt)

- **Location**: `ui/`
- **Framework**: Nuxt 4 (Vue 3), `@nuxt/ui` v4, TypeScript
- **Build output**: `server/internal/static/dist/` (embedded via `//go:embed`; git tracks a placeholder `index.html`, run `make ui` or GoReleaser to generate the full SPA)
- **Public assets**: `ui/public/` (favicon, self-hosted fonts)

### Documentation site (VitePress)

- **Location**: `docs/` (content) and `docs/.vitepress/` (site config)
- **Framework**: VitePress 1.x with `vitepress-openapi` for the embedded OpenAPI reference
- **Published at**: `https://alvarorg14.github.io/openlicensd/` (GitHub Pages via `.github/workflows/docs.yml`)
- **Content**: `docs/*.md` served in place; `README.md`, `QUICKSTART.md`, and `CONTRIBUTING.md` included via VitePress file includes (no duplication)
- **Build**: `make docs-build` (or `cd docs && npm run docs:build`); `make docs-dev` for local preview

### Go SDK

- **Location**: `sdk/go/`
- **Module**: `github.com/alvarorg14/openlicensd/sdk/go`
- **Go version**: 1.26+
- **Dependencies**: stdlib only
- **Scope**: public validation API (`/validate`, `/registry-credentials`, health probes)
- **Release tags**: `sdk/go/vX.Y.Z` (independent from server tags)

### Brand & Design Tokens

- **Font**: Space Grotesk (self-hosted in `ui/public/fonts/`, Medium 500 for titles with `-0.02em` letter-spacing via `tracking-brand`)
- **Colors**: Custom `brand` (primary blue) and `navy` (neutral) scales defined in `ui/assets/css/main.css`
- **Primary**: `#2F6FFF` (brand-500), **Accent**: `#4D8DFF` (brand-400)
- **Neutrals**: Background `#F7F9FC`, Surface `#FFFFFF`, Border `#E6EBF3`, Secondary text `#6E7C93`, Primary text `#1A2238`, Dark Navy `#111C34`, Midnight `#1C2438`
- **Logo components**: `ui/components/BrandMark.vue`, `ui/components/BrandWordmark.vue` (inline SVGs with `currentColor` + accent `#2F6FFF`)
- **Source assets**: `docs/brand/` (mark/wordmark SVGs for light and dark, font files)
- **Nuxt UI config**: `ui/app.config.ts` sets `primary: 'brand'`, `neutral: 'navy'`

### Key Components

| Package | Path | Responsibility |
|---------|------|----------------|
| `api` | `server/internal/api/` | HTTP router, handlers, request/response types |
| `auth` | `server/internal/auth/` | Bcrypt login, session cookies, CSRF, Bearer API tokens, role middleware |
| `clientip` | `server/internal/clientip/` | Trusted-proxy aware client IP resolution |
| `config` | `server/internal/config/` | Environment variable loading and validation |
| `harbor` | `server/internal/harbor/` | Harbor v2 REST client for ephemeral robot accounts |
| `oidc` | `server/internal/oidc/` | OIDC discovery, PKCE authorization code flow, ID token verification |
| `license` | `server/internal/license/` | Key generation, SHA-256 hashing, validation logic |
| `maintenance` | `server/internal/maintenance/` | Background tasks (expired session cleanup) |
| `logging` | `server/internal/logging/` | Structured `slog` output, request-scoped loggers, HTTP request logging middleware |
| `metrics` | `server/internal/metrics/` | Prometheus registry, HTTP middleware, license validation counters, pgxpool collector |
| `ratelimit` | `server/internal/ratelimit/` | Per-IP token bucket rate limiting for unauthenticated endpoints; in-memory (default) or Postgres-backed shared buckets |
| `store` | `server/internal/store/` | PostgreSQL CRUD for products, policies, licenses, machines; validation recording; migrations |
| `static` | `server/internal/static/` | Embedded Nuxt SPA file server |
| `version` | `server/internal/version/` | Build version string injected via ldflags; exposed as `server_version` on `GET /api/v1/auth/me` |
| `openlicensd` | `sdk/go/` | Go client SDK for license validation |

### Helm Chart

- **Location**: `charts/openlicensd/`
- Deploys Deployment, Service, ServiceAccount, ConfigMap, Secret/ExternalSecret, optional Ingress, HorizontalPodAutoscaler, PodDisruptionBudget, NetworkPolicy, and ServiceMonitor
- Default security: non-root (UID 65532), read-only root filesystem, distroless image
- Source `Chart.yaml` `version` / `appVersion` are `0.0.0-dev` placeholders; `.github/workflows/release.yml` stamps the packaged OCI chart from the git tag (`helm package --version/--app-version`)

### Version placeholders

Like the Go binary's `"dev"` default (`server/internal/version/version.go`), source-tree metadata versions are placeholders stamped at release:

| File | Git value | Stamped at release |
|------|-----------|-------------------|
| `charts/openlicensd/Chart.yaml` | `0.0.0-dev` | Helm chart job (`helm package --version/--app-version`) |
| `docs/openapi.yaml` `info.version` | `0.0.0-dev` | Release job (`sed` + GoReleaser `extra_files`) |
| Go `version.Version` | `"dev"` | ldflags (Makefile / GoReleaser) |

Do not commit version bumps to `main` after each publish.

## Data Flow

```
1. Admin logs in via UI or API → receives session cookie (httpOnly) + CSRF cookie; OIDC logins sync `picture_url` from the ID token `picture` claim when present. Automation uses scoped API tokens via `Authorization: Bearer` (no CSRF)
2. Admin creates products and policies → defines expiration rules per product
3. Admin creates license (product + policy required) → server generates key, derives expiry from policy, stores SHA-256 hash, returns raw key once
4. Client validates key → POST /api/v1/validate (optional product code) → hash lookup → validation result
5. Admin lists resources → GET /api/v1/licenses|products|policies|users|api-tokens|audit-events with server-side pagination, search, filters, and sorting
6. Successful admin mutations append an audit event (actor, action, resource, IP, user agent) to `audit_events`; admins read the log via GET /api/v1/audit-events or the Audit Log UI page
7. (Optional) Client requests Harbor credentials → validate key → create robot → return credentials
8. UI dev server proxies /api to Go server on :8080; production embeds static files in binary
9. Structured JSON logs (configurable via `OPENLICENSD_LOG_LEVEL` / `OPENLICENSD_LOG_FORMAT`) include a `request_id` on every HTTP request and handler log line for correlation
10. Prometheus metrics (configurable via `OPENLICENSD_METRICS_ENABLED` / `OPENLICENSD_METRICS_ADDR`) are served on a dedicated listener at `/metrics`, separate from the API/UI port
11. Rate limiting on unauthenticated endpoints uses per-IP token buckets; with `OPENLICENSD_RATE_LIMIT_BACKEND=postgres`, buckets are shared across replicas via PostgreSQL
12. UI sidebar (collapsible via `UDashboardSidebar`, state persisted in a cookie) shows deployed server version and OIDC profile photo from `GET /api/v1/auth/me` (`server_version`, `picture_url` fields)
13. Health probes: `GET /healthz` is liveness (no dependency checks); `GET /readyz` is readiness (PostgreSQL ping, 2s timeout). Kubernetes and the Helm chart map liveness/readiness/startup to these paths.
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENLICENSD_ADDR` | `:8080` | HTTP listen address |
| `OPENLICENSD_DATABASE_URL` | **required** | PostgreSQL connection URL |
| `OPENLICENSD_DATABASE_MAX_CONNS` | `0` | Maximum pool connections (`0` = pgx default) |
| `OPENLICENSD_DATABASE_MIN_CONNS` | `0` | Minimum pool connections (`0` = pgx default) |
| `OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES` | `0` | Idle connection lifetime in minutes (`0` = pgx default) |
| `OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS` | `0` | PostgreSQL statement timeout in seconds (`0` = server default) |
| `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL` | — | Seed first admin when users table is empty |
| `OPENLICENSD_BOOTSTRAP_ADMIN_NAME` | `Administrator` | Display name for bootstrap admin |
| `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` | — | Bcrypt hash for bootstrap admin password |
| `OPENLICENSD_SESSION_TTL_HOURS` | `24` | Session lifetime in hours |
| `OPENLICENSD_REQUEST_TIMEOUT_SECONDS` | `30` | Per-request context deadline in seconds (`0` disables) |
| `OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES` | `60` | Interval for deleting expired/revoked sessions (`0` disables) |
| `OPENLICENSD_COOKIE_SECURE` | `true` | Set `Secure` flag on session cookies |
| `OPENLICENSD_LOCAL_LOGIN_ENABLED` | `true` | Allow email/password login |
| `OPENLICENSD_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, or `error` |
| `OPENLICENSD_LOG_FORMAT` | `json` | Log output format: `json` or `text` |
| `OPENLICENSD_METRICS_ENABLED` | `true` | Enable Prometheus `/metrics` on a dedicated listener |
| `OPENLICENSD_METRICS_ADDR` | `:9090` | Metrics listen address (must differ from `OPENLICENSD_ADDR`) |
| `OPENLICENSD_TRUSTED_PROXIES` | — | Trusted proxy IPs/CIDRs for client IP resolution |
| `OPENLICENSD_RATE_LIMIT_ENABLED` | `true` | Enable per-IP rate limiting on unauthenticated endpoints |
| `OPENLICENSD_RATE_LIMIT_BACKEND` | `memory` | Rate limit backend: `memory` (per-replica) or `postgres` (shared across replicas) |
| `OPENLICENSD_RATE_LIMIT_PUBLIC_PER_MINUTE` | `600` | Sustained rate for `/validate` and `/registry-credentials` |
| `OPENLICENSD_RATE_LIMIT_PUBLIC_BURST` | `60` | Burst capacity for public endpoints |
| `OPENLICENSD_RATE_LIMIT_LOGIN_PER_MINUTE` | `30` | Sustained rate for login and OIDC endpoints |
| `OPENLICENSD_RATE_LIMIT_LOGIN_BURST` | `10` | Burst capacity for login endpoints |
| `OPENLICENSD_RATE_LIMIT_IDLE_MINUTES` | `10` | Minutes before unused per-IP buckets are evicted |
| `OPENLICENSD_OIDC_ENABLED` | `false` | Enable OIDC SSO |
| `OPENLICENSD_OIDC_ISSUER_URL` | — | OIDC issuer URL (required when enabled) |
| `OPENLICENSD_OIDC_CLIENT_ID` | — | OAuth client ID (required when enabled) |
| `OPENLICENSD_OIDC_CLIENT_SECRET` | — | OAuth client secret (required when enabled) |
| `OPENLICENSD_OIDC_REDIRECT_URL` | — | OIDC callback URL (required when enabled) |
| `OPENLICENSD_HARBOR_ENABLED` | `false` | Enable Harbor credentials endpoint |
| `OPENLICENSD_HARBOR_URL` | — | Harbor base URL (required when enabled) |
| `OPENLICENSD_HARBOR_ADMIN_USERNAME` | — | Harbor admin username |
| `OPENLICENSD_HARBOR_ADMIN_PASSWORD` | — | Harbor admin password |
| `OPENLICENSD_HARBOR_PROJECTS` | — | Comma-separated project namespaces |
| `OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS` | `1` | Robot lifetime in days |
| `OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX` | `openlicensd` | Robot name prefix |
| `OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY` | `false` | Skip TLS verification |
| `OPENLICENSD_HARBOR_DEBUG` | `false` | Harbor debug logging |

## Key Files

| File | Purpose |
|------|---------|
| `server/cmd/openlicensd/main.go` | Entry point: config, store, HTTP server, graceful shutdown |
| `server/internal/api/server.go` | Router, login, health probes, shared helpers |
| `server/internal/api/listing.go` | Shared list query parsing and paginated response envelope |
| `server/internal/api/licenses.go` | License handlers |
| `server/internal/api/products.go` | Product handlers |
| `server/internal/api/policies.go` | Policy handlers |
| `server/internal/api/api_tokens.go` | API token handlers |
| `server/internal/api/audit.go` | Audit capture middleware and route map |
| `server/internal/api/audit_events.go` | Audit event list handler |
| `server/internal/api/validate.go` | Validation and registry credentials handlers |
| `server/internal/api/oidc.go` | OIDC login and callback handlers |
| `server/internal/auth/auth.go` | Session auth, Bearer API tokens, CSRF, role middleware |
| `server/internal/harbor/harbor.go` | Harbor robot creation and cleanup |
| `server/internal/oidc/oidc.go` | OIDC discovery, PKCE, ID token verification |
| `server/internal/license/license.go` | Key generation and validation |
| `server/internal/store/listing.go` | Shared list query building for paginated store queries |
| `server/internal/store/store.go` | PostgreSQL operations |
| `server/internal/store/api_tokens.go` | API token CRUD |
| `server/internal/store/audit_events.go` | Append-only audit event storage |
| `server/internal/store/migrations/` | SQL migrations (run on startup) |
| `ui/nuxt.config.ts` | Nuxt config, API proxy in dev |
| `charts/openlicensd/values.yaml` | Helm defaults |
| `docs/openapi.yaml` | OpenAPI 3.1 specification |
| `docs/.vitepress/config.ts` | VitePress site config (nav, sidebar, GitHub Pages base path) |
| `.goreleaser.yaml` | Release and container image publishing |

## Development Workflow

Use `make` targets as the canonical development commands:

```bash
make help          # List all targets
make dev-db        # Start PostgreSQL via Docker Compose (docker-compose.yml)
make dev-db-reset  # Reset local PostgreSQL volume (required when schema changes)
make dev-server    # Run Go API server (loads .env)
make dev-ui        # Run Nuxt dev server
make docs-dev      # Run VitePress docs dev server
make docs-build    # Build VitePress docs site
make stack-up      # Start Postgres + openlicensd from GHCR (docker-compose.stack.yml)
make stack-down    # Stop full stack (ARGS=-v to drop its data)
make stack-logs    # Tail full stack logs
make ui            # Build static UI into server/internal/static/dist
make server        # Build binary to bin/openlicensd
make build         # ui + server
make test          # Go tests (loads .env if present)
make test-sdk      # Go SDK tests
make lint          # go vet + golangci-lint + ESLint (same as CI)
make lint-server   # go vet + golangci-lint
make lint-ui       # ESLint
make lint-sdk      # go vet + golangci-lint on sdk/go
make vuln          # govulncheck
make hash-password # Bcrypt hash CLI
make release       # Local GoReleaser release
```

### Prerequisites

- Go 1.26+
- Node.js 24+
- Docker (for local PostgreSQL)
- golangci-lint v2.12.2 (installed automatically by `make lint-server`)
- Helm 3 (for chart validation)

## Testing Conventions

- Standard library `testing` only (no testify)
- Table-driven tests where appropriate
- Test files live beside source: `*_test.go`
- API tests require a live PostgreSQL instance (`OPENLICENSD_DATABASE_URL`)
- Run: `make test` or `cd server && go test ./...`

## CI & Release

### CI (`.github/workflows/ci.yml`)

Triggers on push/PR to `main` (skips when only SDK-owned paths change; see path filters in the workflow):

| Job | Command |
|-----|---------|
| Server | `make lint-server`, `go test`, `go build` |
| UI | `npm ci`, `make lint-ui`, `npm run generate` |
| GoReleaser | `goreleaser check`, snapshot release |
| Helm | `helm lint`, `helm template`, `helm package` |
| OpenAPI | `@redocly/cli lint docs/openapi.yaml` |

### Docs (`.github/workflows/docs.yml`)

Triggers on push/PR to `main` when `docs/**`, root guides, or the workflow file change:

| Job | Command |
|-----|---------|
| Build | `npm ci`, `npm run docs:build` in `docs/` |
| Deploy | GitHub Pages (push to `main` only) |

### SDK CI (`.github/workflows/sdk-ci.yml`)

Triggers on push/PR to `main` when `sdk/**` or the workflow file changes:

| Job | Command |
|-----|---------|
| Go SDK | `make lint-sdk`, `make test-sdk` (Go 1.26) |

### Vulnerability scanning (`.github/workflows/vuln.yml`)

Runs [`govulncheck`](https://go.dev/security/vuln/) separately from CI:

| Trigger | Purpose |
|---------|---------|
| Weekly schedule (Mondays 06:00 UTC) | Catch new CVEs without a commit |
| `workflow_dispatch` | Manual on-demand scan |
| Pull request to `main` | Early visibility (non-blocking) |

### Dependency updates (Renovate)

[Renovate](https://docs.renovatebot.com/) is configured in [`renovate.json`](renovate.json) to propose updates for Go modules, npm packages, Docker base images, and GitHub Actions. Pull requests are labeled `dependencies` or `ci` to satisfy the PR policy below.

### PR Policy (`.github/workflows/pr-policy.yml`)

Pull requests must carry **exactly one** label:

- `breaking-change`, `feature`, `enhancement`, `bug`, `dependencies`, `documentation`, `deprecations`, `ci`

### Release Drafter

On push to `main`, maintains draft releases for the server and Go SDK independently. Each drafter workflow uses path filters aligned with SDK-owned vs server paths (`Makefile` is server-owned):

| Draft | Workflow | Config | Tag format |
|-------|----------|--------|------------|
| Server (stable + prerelease) | `.github/workflows/release-drafter.yml` | `release-drafter-template.yml` | `vX.Y.Z`, `vX.Y.Z-rc.N` |
| Go SDK (stable + prerelease) | `.github/workflows/sdk-release-drafter.yml` | `release-drafter-sdk.yml` | `sdk/go/vX.Y.Z`, `sdk/go/vX.Y.Z-rc.N` |

SDK drafts include pull requests that touched SDK-owned paths (`sdk/**`, `docs/sdk/**`, and SDK workflow/config files). Pure SDK-only PRs are excluded from server drafts via `pre-exclude` in `release-drafter-template.yml`.

### Release (`.github/workflows/release.yml`)

On GitHub release publish (server tags only — SDK releases are excluded):

- GoReleaser builds binaries and pushes `ghcr.io/alvarorg14/openlicensd` (amd64 + arm64)
- Release job stamps `docs/openapi.yaml` `info.version` from the tag and attaches it to the GitHub release
- Helm chart packaged and pushed to `oci://ghcr.io/alvarorg14/charts` (separate job; `--version/--app-version` from tag)

### SDK Release (`.github/workflows/sdk-release.yml`)

On GitHub release publish (SDK tags only — server releases are excluded):

- Runs `make lint-sdk` and `make test-sdk`
- Warms the Go module proxy for pkg.go.dev

SDK and server versions are independent. Server tags use a `v` prefix (`v0.5.0`); SDK tags use the Go module format (`sdk/go/v0.1.0`).

## Quality Assurance Requirements

**Every change MUST pass these checks before completion:**

### 1. Linting (MANDATORY)

- Run `make lint` — must pass (runs `lint-server` + `lint-ui` + `lint-sdk`, same as CI)
- Server: `make lint-server` (`go vet` + `golangci-lint`)
- UI: `make lint-ui` (ESLint)
- SDK: `make lint-sdk` (`go vet` + `golangci-lint` on `sdk/go`)

### 2. Build (MANDATORY)

- `make build` must succeed
- Never commit code that doesn't build

### 3. Testing (MANDATORY)

- `make test` must pass
- Do NOT skip or delete tests to make them pass
- Add tests for new features when appropriate

### 4. Documentation (MANDATORY)

- Update `README.md` for user-facing changes
- Update `QUICKSTART.md` if install/verify steps change
- Update `docs/openapi.yaml` if API changes
- Update `docs/sdk/go.md` if SDK behavior changes
- Update `AGENTS.md` for architectural changes
- Run `make docs-build` when documentation content or site config changes
- Update Helm chart README if values change

### Pre-Commit Checklist

- [ ] `make lint` passes
- [ ] `make build` succeeds
- [ ] `make test` passes
- [ ] Documentation updated where needed
- [ ] PR has exactly one policy label
- [ ] Conventional commit message used when possible

## Code Style

- **Go**: Follow standard Go conventions, use `gofmt`
- **TypeScript/Vue**: Follow Nuxt and ESLint conventions
- **Error handling**: Always handle errors; use `log.Printf` for server logging
- **Context**: Use `context.Context` for database and HTTP operations

## Important Warnings

**DO NOT**:

- Commit code that doesn't build or pass tests
- Skip linting, testing, or documentation updates
- Open public issues for security vulnerabilities (use private advisories)
- Store plaintext passwords or full license keys in the database
- Mark tasks complete without verification

**DO**:

- Use `make` targets for development
- Add tests for new logic in `server/internal/`
- Update `docs/openapi.yaml` when changing API endpoints or schemas
- Update docs when behavior or configuration changes
- Use conventional commits (`feat:`, `fix:`, `docs:`, etc.)

## Related Documentation

- [README.md](README.md) — User-facing overview and reference
- [QUICKSTART.md](QUICKSTART.md) — Get running in minutes
- [docs/](docs/) — API spec, architecture, comparison with alternatives, configuration, deployment, upgrade, backup/restore, scaling, troubleshooting, OIDC SSO, Harbor
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contributor workflow
- [SECURITY.md](SECURITY.md) — Security policy and vulnerability reporting
