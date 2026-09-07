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

OpenLicensd is pre-1.0. There is no long-term support (LTS) commitment yet — that will be defined as part of the v1.0 release.

Until then:

- **Server** — use the [latest published release](https://github.com/alvarorg14/openlicensd/releases) (`vX.Y.Z` tags).
- **Go SDK** — use the latest published SDK release (`sdk/go/vX.Y.Z` tags), versioned independently from the server.

For which versions receive security patches, see [SECURITY.md — Supported Versions](SECURITY.md#supported-versions).

## Response expectations

This is a volunteer-maintained project. There is **no SLA** for GitHub Issues or Discussions.

Maintainers respond on a best-effort basis. Security advisories follow the timelines in [SECURITY.md](SECURITY.md); those timelines do not apply to general support requests.

## Out of scope

Maintainers generally cannot help with:

- Modified forks or unofficial builds
- Third-party infrastructure beyond the documented [Helm](docs/deployment.md), Docker, and binary deployment paths
- Custom integrations not covered in the documentation

## Commercial support

There is no paid support, hosted offering, or enterprise support contract for OpenLicensd. The project is licensed under [Apache 2.0](LICENSE) and intended for self-hosted use.
