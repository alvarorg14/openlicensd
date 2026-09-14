# Support

OpenLicensd is an open-source, self-hosted project. There is no commercial support offering — you deploy and operate it on your own infrastructure.

## Getting help

| Need | Where to go |
|------|-------------|
| Documentation | [https://alvarorg14.github.io/openlicensd/](https://alvarorg14.github.io/openlicensd/) |
| Questions and usage | [GitHub Discussions](https://github.com/alvarorg14/openlicensd/discussions) |
| Bugs and feature requests | [GitHub Issues](https://github.com/alvarorg14/openlicensd/issues) (use the issue templates) |
| Security vulnerabilities | [Private security advisory](https://github.com/alvarorg14/openlicensd/security/advisories/new) — see [SECURITY.md](SECURITY.md) |

When opening a bug report, include your OpenLicensd version, deployment method, PostgreSQL version, whether Harbor is enabled, relevant logs, and steps to reproduce. See [CONTRIBUTING.md](CONTRIBUTING.md#reporting-issues) for the full checklist.

## Supported versions

Run a release on a supported line. The canonical policy — including end-of-life dates and what “v1 LTS” means — is [SECURITY.md — Supported Versions](SECURITY.md#supported-versions).

**Until `v1.0.0` is published:**

- **Server / Helm** — [latest `0.x` release](https://github.com/alvarorg14/openlicensd/releases) (`vX.Y.Z` tags).
- **Go SDK** — latest `sdk/go/v0.x` release, versioned independently from the server.

**From `v1.0.0` / `sdk/go/v1.0.0`:**

- **Latest 1.x** is the recommended production line (security and bugfix patches).
- **0.x** is end-of-life — upgrade via [docs/upgrade.md](docs/upgrade.md).
- **Older 1.x minors** do not receive a dedicated backport train; upgrade to the latest 1.x minor (additive per [COMPATIBILITY.md](COMPATIBILITY.md)).

There is no commercial SLA or paid long-term support. “v1 LTS” means the entire `1.x` major receives patches until `v2.0.0`, then 12 months of security-only patches — not a freeze of `1.0.0` forever.

For API stability, SemVer scope, and the deprecation process from v1.0.0 onward, see [COMPATIBILITY.md](COMPATIBILITY.md).

## Response expectations

This is a volunteer-maintained project. There is **no SLA** for GitHub Issues or Discussions.

Maintainers respond on a best-effort basis. Security advisories follow the timelines in [SECURITY.md](SECURITY.md); those timelines do not apply to general support requests.

## Out of scope

Maintainers generally cannot help with:

- Modified forks or unofficial builds
- Third-party infrastructure beyond the documented [Helm](docs/deployment.md), Docker/GHCR, and **Linux** binary deployment paths (see [Platforms](docs/deployment.md#platforms))
- Custom integrations not covered in the documentation

## Commercial support

There is no paid support, hosted offering, or enterprise support contract for OpenLicensd. The project is licensed under [Apache 2.0](LICENSE) and intended for self-hosted use.
