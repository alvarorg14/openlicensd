# Changelog

All notable changes to the OpenLicensd server are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Server releases use `vX.Y.Z` tags. The [Go SDK](https://github.com/alvarorg14/openlicensd/releases?q=sdk%2Fgo) is versioned independently (`sdk/go/vX.Y.Z`).

Release notes are drafted by [Release Drafter](https://github.com/release-drafter/release-drafter) on GitHub. Before publishing a server release, move the draft body from `Unreleased` into a dated version section below (see [CONTRIBUTING.md](CONTRIBUTING.md#server)).

## [Unreleased]

### Bug Fixes

- fix(api): reject demote/disable/delete of the last admin (#227)
- fix(server): normalize API response timestamps to RFC3339 strings (#220)
- fix(server): support true partial PATCH updates for licenses, products, and policies (#218)

### Enhancements

- enhancement(server): rename license machine path param from `machineId` to `machine_id` (#221)
- enhancement(server): rate limit authenticated admin endpoints per user or API token (#176)
## [0.8.0] - 2026-09-08

### New Features

- feat(server): add append-only audit log for admin mutations (#178)
- feat(server): add scoped Bearer API tokens for admin automation (#177)

### Bug Fixes

- fix(test): stabilize flaky postgres rate limit tests (#190)
- fix(docs): correct dark logo, API outline, and Mermaid diagrams (#186)

### Updated Dependencies

- fix(deps): update module golang.org/x/oauth2 to v0.37.0 (#216)
- fix(deps): update module github.com/jackc/pgx/v5 to v5.11.0 (#215)
- chore(deps): update dependency @nuxt/ui to v4.11.1 (#199)
- chore(deps): update dependency @iconify-json/lucide to v1.2.130 (#198)
- fix(deps): update module github.com/go-jose/go-jose/v4 to v4.1.5 (#196)
- fix(deps): update module golang.org/x/crypto to v0.56.0 (#195)
- chore(deps): update dependency @iconify-json/lucide to v1.2.129 (#194)
- chore(deps): update dependency vue-router to v5.3.1 (#189)
- fix(deps): update module github.com/coreos/go-oidc/v3 to v3.21.0 (#179)

### Documentation

- docs: add NOTICE and Apache-2.0 license metadata (#209)
- docs: add SUPPORT.md with support channels and version guidance (#208)
- docs: add CHANGELOG.md and re-enable GoReleaser changelog (#207)
- docs(ci): add GitHub pull request description template (#206)
- docs: publish docs/ as VitePress site on GitHub Pages (#183)
- docs(readme): defer API catalog to OpenAPI spec (#182)
- docs(readme): add comparison with Keygen, Cryptlex, and LicenseSpring (#181)
- docs(readme): add admin UI licenses screenshot (#180)

### CI

- ci(ui): add Playwright smoke test for admin happy path (#205)
- ci: upload Go coverage to Codecov from server and SDK CI (#204)
- ci: pin GitHub Actions to SHAs and Redocly CLI version (#203)
- ci: scope GITHUB_TOKEN permissions across all workflows (#202)
- ci(release): sign GHCR images with Cosign and publish SLSA provenance (#201)
- ci: add Syft SBOM generation for server releases (#200)
- chore(deps): update dependency goreleaser/goreleaser to v2.18.1 (#197)
- chore(deps): update docker/build-push-action action to v7 (#193)
- chore(deps): update aquasecurity/trivy-action digest to ed142fd (#192)
- chore(deps): update actions/upload-pages-artifact action to v5 (#187)
- chore(deps): update actions/deploy-pages action to v5 (#185)
- chore(deps): update actions/configure-pages action to v6 (#184)
- ci: add Trivy container image scanning (#191)
- ci: add CodeQL static analysis workflow (#188)

## [0.7.0] - 2026-09-01

### New Features

- feat(server): Prometheus /metrics endpoint on dedicated listener (#160)

### Improvements

- enhancement(helm): add optional ServiceMonitor for Prometheus (#171)
- enhancement(helm): add optional topologySpreadConstraints to deployment (#170)
- enhancement(helm): add optional NetworkPolicy template (#169)
- enhancement(helm): add optional HorizontalPodAutoscaler template (#168)
- enhancement(helm): add optional PodDisruptionBudget template (#167)
- enhancement(server): document and test liveness/readiness health split (#166)
- enhancement(server): opt-in postgres rate limit backend for multi-replica deploys (#165)
- perf(store): replace correlated activation_count subquery in license list (#164)
- enhancement(server): configurable per-request context deadlines (#163)
- enhancement(server): configurable pgxpool limits and statement timeout (#162)
- enhancement(server): structured slog logging with request IDs (#159)

### Updated Dependencies

- fix(deps): update module github.com/prometheus/client_golang to v1.24.1 (#161)
- chore(deps): update dependency @iconify-json/lucide to v1.2.128 (#158)

### Documentation

- docs(docs): add operator troubleshooting runbook (#175)
- docs(docs): add HA, scaling, and multi-replica runbook (#174)
- docs(docs): add upgrade procedure and migration notes runbook (#173)
- docs(docs): add PostgreSQL backup and restore runbook (#172)

## [0.6.0] - 2026-08-31

### New Features

- feat(api): add GET /api/v1/licenses/{id} (#149)

### Improvements

- enhancement(api): add security headers to embedded UI responses (#155)
- refactor(oidc): remove dead InsecureSkipVerify field (#154)
- enhancement(ui): wire admin password reset into Users page (#153)
- enhancement(api): paginate GET /api/v1/users (#152)
- enhancement(api): rename license activate to unrevoke (#151)
- enhancement(store): drop unused archived_at from products and policies (#148)
- feat(api): enforce shared 8-character password minimum (#147)
- ci(release): stamp Chart and OpenAPI versions at publish time (#144)

### Bug Fixes

- ci(release): fix goreleaser dirty git on openapi stamp (#157)
- fix(ui): bundle embedded icons and complete docker stack env (#156)
- fix(api): increment validation_count only on successful validations (#150)
- fix(server): align module path with the GitHub repository (#143)
- fix(static): serve a stub UI when Nuxt assets are not built (#139)

### Updated Dependencies

- chore(deps): update dependency vue-router to v5.3.0 (#140)

### Documentation

- docs: normalize git tag prefixes to vX.Y.Z (#146)

### CI

- chore(deps): update dependency golangci/golangci-lint to v2.13.2 (#141)
- fix(ci): quote OpenAPI grep pattern to fix YAML parse error (#145)
- chore(cursor): add openlicensd-issue agent skill (#142)

## [0.5.0] - 2026-08-27

### New Features

- feat: add max concurrent machine activations per license (#64)

### Improvements

- feat(ui): collapsible sidebar and OIDC profile photos (#52)
- feat(ui): show deployed version and GitHub link in sidebar (#48)

### Bug Fixes

- fix(ci): roll Go toolchain back to 1.26.6 (#60)

### Updated Dependencies

- chore(deps): update dependency vue to v3.5.42 (#65)
- chore(deps): update dependency @nuxt/ui to v4.11.0 (#62)
- fix(deps): update module github.com/go-chi/chi/v5 to v5.3.2 (#59)
- chore(deps): update go toolchain directive to v1.27.0 (#56)
- chore(deps): update go toolchain directive to v1.26.6 (#54)
- fix(deps): update module golang.org/x/crypto to v0.55.0 (#53)
- chore(deps): update dependency nuxt to v4.5.2 (#51)
- chore(deps): update dependency vue to v3.5.41 (#49)

### Documentation

- docs: add public v1.0 roadmap with GitHub milestone links (#138)

### CI

- chore(deps): update dependency goreleaser/goreleaser to v2.18.0 (#63)
- ci: add CODEOWNERS with default repository owner (#61)
- chore(deps): update dependency golangci/golangci-lint to v2.13.1 (#58)
- chore(deps): update module golang.org/x/vuln to v1.7.0 (#55)
- ci: scope Makefile to server workflows only (#50)
- ci(sdk): simplify CI to Go 1.26 only (#47)
- ci(sdk): split SDK workflows and scope release drafts (#46)

## [0.4.0] - 2026-08-03

### New Features

- feat(sdk): add Go client SDK with release drafter automation (#45)

## [0.3.0] - 2026-08-03

### New Features

- feat(ui): implement OpenLicensd brand redesign (#43)
- feat(maintenance): add periodic expired session cleanup (#38)

### Improvements

- feat(ui): enlarge stat icons and default form controls to full width (#44)
- refactor(helm): move config env vars to ConfigMap (#42)
- feat(server): add per-IP rate limiting for unauthenticated endpoints (#41)
- feat(ui): show license creator in details modal (#39)

### Documentation

- docs(api): complete OpenAPI spec with users, RBAC, and schema fixes (#40)

### CI

- chore(deps): update actions/cache action to v6 (#36)

## [0.2.0] - 2026-07-31

### New Features

- feat(auth): add self-service password change (#34)
- feat(oidc): add generic OIDC SSO with PKCE and JIT provisioning (#30)
- feat(auth): add database-backed users, roles, and session cookies (#29)
- feat: add products and policies for license management (#27)

### Improvements

- feat(docker): add compose stack and fix static root redirect (#37)
- feat(ui): move account controls to sidebar user menu (#33)
- feat(api): add server-side pagination for list endpoints (#32)
- feat(ui): add table filters, details modal, and description truncation (#28)

### Updated Dependencies

- chore(deps): update npm to v12.0.2 (#26)

### Documentation

- docs(deployment): align Helm install and release instructions (#31)

### CI

- ci: align lint with CI, pin tools, and fix release drafter (#35)

## [0.1.1] - 2026-07-29

### Bug Fixes

- fix(docker): mount postgres volume at /var/lib/postgresql for PG 18 (#25)

### Updated Dependencies

- chore(deps): update npm to v12 (#19)
- chore(deps): update dependency typescript to v7 (#18)
- fix(deps): update dependency @nuxt/ui to v4 (#21)
- fix(deps): update dependency vue-router to v5 (#23)
- fix(deps): update dependency nuxt to v4 (#22)
- fix(deps): update module golang.org/x/crypto to v0.54.0 (#15)
- fix(deps): update module github.com/jackc/pgx/v5 to v5.10.0 (#14)
- fix(deps): update module github.com/golang-jwt/jwt/v5 to v5.3.1 (#13)
- fix(deps): update module github.com/go-chi/chi/v5 to v5.3.1 (#12)
- chore(deps): update npm to v11.18.0 (#10)

### Documentation

- docs: update Nuxt 4 and @nuxt/ui v4 references (#24)

### CI

- chore(deps): update postgres docker tag to v18 (#20)
- chore(deps): update actions/setup-node action to v7 (#17)
- chore(deps): update actions/checkout action to v7 (#16)
- chore(deps): update release-drafter/release-drafter digest to 34d8067 (#9)

## [0.1.0] - 2026-07-29

### New Features

- feat(helm): add kubernetes deployment chart with health probes (#7)
- feat(api): add optional Harbor registry credentials endpoint (#5)
- feat: extend license management with edit, delete, activate, and usage tracking (#3)
- feat: initial open source license server implementation (#1)

### Improvements

- feat(ui): modernize dashboard with indigo theme and redesigned login (#4)

### Documentation

- docs: overhaul project documentation and OpenAPI spec (#8)

### CI

- ci(release): add docker release pipeline with goreleaser (#6)
- ci(release-drafter): migrate config to unified category schema (#2)

## [0.0.1] - 2026-07-28

- Initial release

[Unreleased]: https://github.com/alvarorg14/openlicensd/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/alvarorg14/openlicensd/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/alvarorg14/openlicensd/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/alvarorg14/openlicensd/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/alvarorg14/openlicensd/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/alvarorg14/openlicensd/compare/0.3.0...v0.4.0
[0.3.0]: https://github.com/alvarorg14/openlicensd/compare/0.2.0...0.3.0
[0.2.0]: https://github.com/alvarorg14/openlicensd/compare/0.1.1...0.2.0
[0.1.1]: https://github.com/alvarorg14/openlicensd/compare/0.1.0...0.1.1
[0.1.0]: https://github.com/alvarorg14/openlicensd/compare/0.0.1...0.1.0
[0.0.1]: https://github.com/alvarorg14/openlicensd/releases/tag/0.0.1
