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

### Key Components

| Package | Path | Responsibility |
|---------|------|----------------|
| `api` | `server/internal/api/` | HTTP router, handlers, request/response types |
| `auth` | `server/internal/auth/` | Bcrypt login, HS256 JWT signing, bearer middleware |
| `config` | `server/internal/config/` | Environment variable loading and validation |
| `harbor` | `server/internal/harbor/` | Harbor v2 REST client for ephemeral robot accounts |
| `license` | `server/internal/license/` | Key generation, SHA-256 hashing, validation logic |
| `store` | `server/internal/store/` | PostgreSQL CRUD, validation recording, migrations |
| `static` | `server/internal/static/` | Embedded Nuxt SPA file server |

### Helm Chart

- **Location**: `charts/openlicensd/`
- Deploys Deployment, Service, ServiceAccount, Secret/ExternalSecret, optional Ingress
- Default security: non-root (UID 65532), read-only root filesystem, distroless image

## Data Flow

```
1. Admin logs in via UI or API → receives JWT (24h expiry)
2. Admin creates license → server generates key, stores SHA-256 hash, returns raw key once
3. Client validates key → POST /api/v1/validate → hash lookup → validation result
4. (Optional) Client requests Harbor credentials → validate key → create robot → return credentials
5. UI dev server proxies /api to Go server on :8080; production embeds static files in binary
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENLICENSD_ADDR` | `:8080` | HTTP listen address |
| `OPENLICENSD_DATABASE_URL` | **required** | PostgreSQL connection URL |
| `OPENLICENSD_ADMIN_USER` | **required** | Admin username |
| `OPENLICENSD_ADMIN_PASSWORD_HASH` | **required** | Bcrypt hash of admin password |
| `OPENLICENSD_JWT_SECRET` | **required** | JWT signing secret |
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
| `server/internal/api/server.go` | All routes, handlers, and request/response structs |
| `server/internal/auth/auth.go` | Login, JWT, middleware |
| `server/internal/harbor/harbor.go` | Harbor robot creation and cleanup |
| `server/internal/license/license.go` | Key generation and validation |
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
make dev-db        # Start PostgreSQL via Docker Compose
make dev-server    # Run Go API server (loads .env)
make dev-ui        # Run Nuxt dev server
make ui            # Build static UI into server/internal/static/dist
make server        # Build binary to bin/openlicensd
make build         # ui + server
make test          # Go tests (loads .env if present)
make lint          # go vet + ESLint
make hash-password # Bcrypt hash CLI
make release       # Local GoReleaser release
```

### Prerequisites

- Go 1.26+
- Node.js 24+
- Docker (for local PostgreSQL)
- golangci-lint (CI uses v1.64.8)
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
| Server | `go vet`, `golangci-lint`, `go test`, `go build` |
| UI | `npm ci`, `npm run lint`, `npm run generate` |
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

### Release (`.github/workflows/release.yml`)

On GitHub release publish:

- GoReleaser builds binaries and pushes `ghcr.io/alvarorg14/openlicensd` (amd64 + arm64)
- Helm chart packaged and pushed to `oci://ghcr.io/alvarorg14/charts`

## Quality Assurance Requirements

**Every change MUST pass these checks before completion:**

### 1. Linting (MANDATORY)

- Run `make lint` — must pass
- Server: `go vet` + `golangci-lint`
- UI: ESLint via `npm run lint`

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
- [docs/](docs/) — API spec, architecture, configuration, deployment, Harbor
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contributor workflow
- [SECURITY.md](SECURITY.md) — Security policy and vulnerability reporting
