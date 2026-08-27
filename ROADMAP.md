# Roadmap

OpenLicensd is on a path toward a stable **v1.0.0** release. v1.0 is a promise of API stability and production operability — not just a feature checklist.

Work is organized into four release phases, plus a post-1.0 backlog. Track progress on GitHub:

| Phase | Milestone | Tracking issue | Focus |
|-------|-----------|----------------|-------|
| **v0.6.0** | [v0.6.0](https://github.com/alvarorg14/openlicensd/milestone/1) | [#67](https://github.com/alvarorg14/openlicensd/issues/67) | Fix the foundations — module paths, version hygiene, API completeness, UI correctness |
| **v0.7.0** | [v0.7.0](https://github.com/alvarorg14/openlicensd/milestone/2) | [#68](https://github.com/alvarorg14/openlicensd/issues/68) | Make it operable — observability, scaling docs, Helm hardening |
| **v0.8.0** | [v0.8.0](https://github.com/alvarorg14/openlicensd/milestone/3) | [#69](https://github.com/alvarorg14/openlicensd/issues/69) | Make it adoptable — API tokens, audit log, CI supply chain, docs site |
| **v1.0.0** | [v1.0.0](https://github.com/alvarorg14/openlicensd/milestone/4) | [#70](https://github.com/alvarorg14/openlicensd/issues/70) | Commit to stability — deprecation policy, support window, release |
| **Post-1.0** | [Post-1.0](https://github.com/alvarorg14/openlicensd/milestone/5) | [#71](https://github.com/alvarorg14/openlicensd/issues/71) | Roadmap features — offline licenses, entitlements, webhooks, more SDKs |

Each milestone has a tracking issue with linked sub-issues. See the [Issues](https://github.com/alvarorg14/openlicensd/issues) page filtered by milestone.

---

## v0.6.0 — Fix the foundations

Correctness and consistency before adding scope: a working embedded UI from a fresh clone, a canonical Go module path, normalized release tags, consolidated password policy, and small API gaps (`GET /licenses/{id}`, user pagination, admin password reset in the UI).

## v0.7.0 — Make it operable

Production readiness: structured logging, Prometheus metrics, database pool tuning, request deadlines, distributed or documented rate limiting, meaningful health checks, and Helm templates for PDB, HPA, NetworkPolicy, and ServiceMonitor. Operational runbooks for backup, upgrade, scaling, and troubleshooting.

## v0.8.0 — Make it adoptable

Adoption blockers: scoped API tokens for automation, an append-only audit log, README polish (screenshot, positioning), a published docs site, supply-chain CI (CodeQL, Trivy, SBOM, cosign), UI smoke tests, and SDK v1.0 prep (`doc.go`, documentation parity, test gaps).

## v1.0.0 — Commit to stability

Document the API stability and deprecation policy, define the support window, clarify platform scope (Linux binaries), and tag `v1.0.0` plus `sdk/go/v1.0.0`.

## Post-1.0

Features that expand the product surface without blocking the stability promise: offline signed licenses, entitlements and metadata, webhooks, client-side machine deactivation, bulk import/export, key rotation, expiring-soon notifications, and additional client SDKs (TypeScript/Node first).

---

Contributions welcome. Pick an issue labeled [`good first issue`](https://github.com/alvarorg14/openlicensd/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) or ask in [Discussions](https://github.com/alvarorg14/openlicensd/discussions) (enable Discussions on the repo if not yet active).
