# AGENTS.md - AI Assistant Guidelines

This document provides context and guidelines for AI coding assistants working on the OpenLicensd project.

## Project Overview

**OpenLicensd** is an open source license server for creating, managing, and validating license keys. It ships as a single Go binary with an embedded Nuxt admin UI, backed by PostgreSQL.

### Key Purpose

- Generate and manage human-readable license keys (Crockford Base32 format)
- Provide a public validation endpoint for client applications
- Track license usage (validation count and last validated time)
- Optionally issue short-lived Harbor registry credentials to licensed clients
- Serve an admin UI for license management

## Architecture

### Backend (Go)

- **Location**: `server/cmd/openlicensd/` (main entry), `server/internal/` (core logic)
- **Module**: `github.com/openlicensd/openlicensd/server`
- **Go version**: 1.26+
- **Key dependencies**: `go-chi/chi`, `golang-jwt/jwt`, `jackc/pgx`, `google/uuid`

### Frontend (Nuxt)

- **Location**: `ui/`
- **Framework**: Nuxt 4 (Vue 3), `@nuxt/ui` v4, TypeScript
- **Build output**: `server/internal/static/dist/` (embedded via `//go:embed`)
- **Public assets**: `ui/public/` (favicon, self-hosted fonts)

### Go SDK

- **Location**: `sdk/go/`
- **Module**: `github.com/alvarorg14/openlicensd/sdk/go`
- **Go version**: 1.24+
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
| `auth` | `server/internal/auth/` | Bcrypt login, session cookies, CSRF, role middleware |
| `clientip` | `server/internal/clientip/` | Trusted-proxy aware client IP resolution |
| `config` | `server/internal/config/` | Environment variable loading and validation |
| `harbor` | `server/internal/harbor/` | Harbor v2 REST client for ephemeral robot accounts |
| `oidc` | `server/internal/oidc/` | OIDC discovery, PKCE authorization code flow, ID token verification |
| `license` | `server/internal/license/` | Key generation, SHA-256 hashing, validation logic |
| `maintenance` | `server/internal/maintenance/` | Background tasks (expired session cleanup) |
| `ratelimit` | `server/internal/ratelimit/` | Per-IP token bucket rate limiting for unauthenticated endpoints |
| `store` | `server/internal/store/` | PostgreSQL CRUD for products, policies, licenses; validation recording; migrations |
| `static` | `server/internal/static/` | Embedded Nuxt SPA file server |
| `openlicensd` | `sdk/go/` | Go client SDK for license validation |

### Helm Chart

- **Location**: `charts/openlicensd/`
- Deploys Deployment, Service, ServiceAccount, ConfigMap, Secret/ExternalSecret, optional Ingress
- Default security: non-root (UID 65532), read-only root filesystem, distroless image

## Data Flow

```
1. Admin logs in via UI or API → receives session cookie (httpOnly) + CSRF cookie
2. Admin creates products and policies → defines expiration rules per product
3. Admin creates license (product + policy required) → server generates key, derives expiry from policy, stores SHA-256 hash, returns raw key once
4. Client validates key → POST /api/v1/validate (optional product code) → hash lookup → validation result
5. Admin lists resources → GET /api/v1/licenses|products|policies with server-side pagination, search, filters, and sorting
6. (Optional) Client requests Harbor credentials → validate key → create robot → return credentials
7. UI dev server proxies /api to Go server on :8080; production embeds static files in binary
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENLICENSD_ADDR` | `:8080` | HTTP listen address |
| `OPENLICENSD_DATABASE_URL` | **required** | PostgreSQL connection URL |
| `OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL` | — | Seed first admin when users table is empty |
| `OPENLICENSD_BOOTSTRAP_ADMIN_NAME` | `Administrator` | Display name for bootstrap admin |
| `OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH` | — | Bcrypt hash for bootstrap admin password |
| `OPENLICENSD_SESSION_TTL_HOURS` | `24` | Session lifetime in hours |
| `OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES` | `60` | Interval for deleting expired/revoked sessions (`0` disables) |
| `OPENLICENSD_COOKIE_SECURE` | `true` | Set `Secure` flag on session cookies |
| `OPENLICENSD_LOCAL_LOGIN_ENABLED` | `true` | Allow email/password login |
| `OPENLICENSD_TRUSTED_PROXIES` | — | Trusted proxy IPs/CIDRs for client IP resolution |
| `OPENLICENSD_RATE_LIMIT_ENABLED` | `true` | Enable per-IP rate limiting on unauthenticated endpoints |
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
| `server/internal/api/validate.go` | Validation and registry credentials handlers |
| `server/internal/api/oidc.go` | OIDC login and callback handlers |
| `server/internal/auth/auth.go` | Session auth, CSRF, role middleware |
| `server/internal/harbor/harbor.go` | Harbor robot creation and cleanup |
| `server/internal/oidc/oidc.go` | OIDC discovery, PKCE, ID token verification |
| `server/internal/license/license.go` | Key generation and validation |
| `server/internal/store/listing.go` | Shared list query building for paginated store queries |
| `server/internal/store/store.go` | PostgreSQL operations |
| `server/internal/store/migrations/` | SQL migrations (run on startup) |
| `ui/nuxt.config.ts` | Nuxt config, API proxy in dev |
| `charts/openlicensd/values.yaml` | Helm defaults |
| `docs/openapi.yaml` | OpenAPI 3.1 specification |
| `.goreleaser.yaml` | Release and container image publishing |

## Development Workflow

Use `make` targets as the canonical development commands:

```bash
make help          # List all targets
make dev-db        # Start PostgreSQL via Docker Compose (docker-compose.yml)
make dev-db-reset  # Reset local PostgreSQL volume (required when schema changes)
make dev-server    # Run Go API server (loads .env)
make dev-ui        # Run Nuxt dev server
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

Triggers on push/PR to `main`:

| Job | Command |
|-----|---------|
| Server | `make lint-server`, `go test`, `go build` |
| UI | `npm ci`, `make lint-ui`, `npm run generate` |
| Go SDK | `make lint-sdk` (Go 1.26), `go vet` + `make test-sdk` (Go 1.24 + 1.26 matrix) |
| GoReleaser | `goreleaser check`, snapshot release |
| Helm | `helm lint`, `helm template`, `helm package` |
| OpenAPI | `@redocly/cli lint docs/openapi.yaml` |

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

### Release Drafter (`.github/workflows/release-drafter.yml`)

On push to `main`, maintains draft releases for the server and Go SDK independently:

| Draft | Config | Tag format |
|-------|--------|------------|
| Server (stable + prerelease) | `release-drafter-template.yml` | `vX.Y.Z`, `vX.Y.Z-rc.N` |
| Go SDK (stable + prerelease) | `release-drafter-sdk.yml` | `sdk/go/vX.Y.Z`, `sdk/go/vX.Y.Z-rc.N` |

SDK drafts include only pull requests that touched `sdk/go/`.

### Release (`.github/workflows/release.yml`)

On GitHub release publish (server tags only — SDK releases are excluded):

- GoReleaser builds binaries and pushes `ghcr.io/alvarorg14/openlicensd` (amd64 + arm64)
- Helm chart packaged and pushed to `oci://ghcr.io/alvarorg14/charts`

### SDK Release (`.github/workflows/sdk-release.yml`)

On GitHub release publish (SDK tags only — server releases are excluded):

- Runs `make lint-sdk` and `make test-sdk`
- Warms the Go module proxy for pkg.go.dev

SDK and server versions are independent. Server tags use a `v` prefix (`v0.2.0`); SDK tags use the Go module format (`sdk/go/v0.1.0`).

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
- Update `docs/sdk-go.md` if SDK behavior changes
- Update `AGENTS.md` for architectural changes
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
- [docs/](docs/) — API spec, architecture, configuration, deployment, OIDC SSO, Harbor
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contributor workflow
- [SECURITY.md](SECURITY.md) — Security policy and vulnerability reporting
