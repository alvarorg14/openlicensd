# Contributing to OpenLicensd

Thank you for your interest in contributing!

## Getting started

1. Fork the repository and clone it locally.
2. Install Go 1.26+ and Node.js 24+.
3. Start local PostgreSQL: `make dev-db`
4. Copy environment variables: `cp .env.example .env`
5. Make your changes and add tests where appropriate.
6. Run `make test` and `make lint` before opening a pull request.

Run `make help` to see all available development targets.

## Development workflow

```bash
# List all targets
make help

# Start PostgreSQL
make dev-db

# Run API server (loads .env)
make dev-server

# Run Nuxt dev server
make dev-ui

# Build UI static files
make ui

# Build binary
make build

# Run Go tests
make test

# Lint (go vet + golangci-lint + ESLint)
make lint

# Generate admin password hash
make hash-password PASSWORD=yourpassword
```

## Docs site

The documentation site is built with VitePress from the `docs/` directory and published to GitHub Pages.

```bash
# Run the docs dev server (hot reload)
make docs-dev

# Build the static site (same check as CI)
make docs-build
```

When editing `docs/*.md`, changes appear in the dev server immediately. Root-level guides (`README.md`, `QUICKSTART.md`, `CONTRIBUTING.md`) are included in the site via VitePress file includes — edit those source files, not the wrapper pages in `docs/`.

## Releasing

Releases are managed through [Release Drafter](https://github.com/release-drafter/release-drafter). Merging labeled pull requests to `main` updates draft releases automatically.

### Git tags

Server releases use `vX.Y.Z` or `vX.Y.Z-rc.N` (for example `v0.5.0`, `v0.5.0-rc.1`). Do not publish unprefixed server tags.

Historical server tags `0.0.1` through `0.3.0` predate this convention and remain in the repository for reference only.

The Go SDK uses `sdk/go/vX.Y.Z` or `sdk/go/vX.Y.Z-rc.N`, independent of the server version.

### Server

1. Open the repository's **Releases** page on GitHub.
2. Review the stable or prerelease draft (tag format: `vX.Y.Z` or `vX.Y.Z-rc.N`).
3. Publish the draft when ready.

Publishing triggers `.github/workflows/release.yml`, which builds binaries and container images via GoReleaser and publishes the Helm chart in a separate workflow job. The release job stamps `docs/openapi.yaml` `info.version` from the git tag (strip leading `v`) and attaches the stamped copy to the GitHub release. The stamp is publish-time only — git keeps the `0.0.0-dev` placeholder, and the workflow marks the file skip-worktree so GoReleaser still sees a clean worktree.

`charts/openlicensd/Chart.yaml` and `docs/openapi.yaml` keep `0.0.0-dev` placeholders in git (like the Go binary's `"dev"` default). Do not bump these on every release — the workflow stamps published artifacts from the tag. Install from OCI with `--version X.Y.Z`; when installing from the source chart, set `image.tag` explicitly.

### Go SDK

The Go SDK is versioned independently from the server. Release Drafter maintains separate drafts scoped to SDK-owned paths (`sdk/**`, `docs/sdk/**`, and SDK workflow/config files; tag format: `sdk/go/vX.Y.Z`).

1. Open the repository's **Releases** page on GitHub.
2. Review the **Go SDK** stable or prerelease draft.
3. Publish the draft when ready.

Publishing triggers `.github/workflows/sdk-release.yml`, which lints, tests, and warms the Go module proxy for pkg.go.dev.

Consumers install with:

```bash
go get github.com/alvarorg14/openlicensd/sdk/go@v0.1.0
```

See [docs/sdk/go.md](docs/sdk/go.md) for integration documentation.

For architecture details and AI assistant guidelines, see [AGENTS.md](AGENTS.md).

## Database migrations

Schema changes use numbered SQL files in `server/internal/store/migrations/`. Migrations are forward-only — there are no down files.

- Add a new file with the next sequence number (for example `010_my_change.sql`).
- Never edit a migration that has already been applied in any environment.
- Reviewers need `make dev-db-reset` when testing a new migration locally.

See [docs/upgrade.md](docs/upgrade.md) for how migrations behave in production upgrades and rollbacks.

## Pull requests

- Keep changes focused and well-scoped.
- Update documentation when behavior or configuration changes.
- Use conventional commit messages when possible (e.g. `feat:`, `fix:`, `docs:`).
- Ensure CI passes before requesting review.
- Add exactly one policy label to your PR: `breaking-change`, `feature`, `enhancement`, `bug`, `dependencies`, `documentation`, `deprecations`, or `ci`.

## Reporting issues

Please include:

- OpenLicensd version (release tag or image tag)
- Deployment method (Helm, Docker, binary, local dev)
- PostgreSQL version
- Whether Harbor integration is enabled
- Relevant server logs
- Steps to reproduce

## Security

If you discover a security vulnerability, please report it via a [private GitHub security advisory](https://github.com/alvarorg14/openlicensd/security/advisories/new). See [SECURITY.md](SECURITY.md) for details.

## Code of conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold this code.
