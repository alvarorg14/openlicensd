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

For architecture details and AI assistant guidelines, see [AGENTS.md](AGENTS.md).

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
