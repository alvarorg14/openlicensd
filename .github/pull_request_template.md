<!--
Fill every section. Delete a section only if it truly does not apply.
Use a conventional commit-style title. Add exactly one policy label:
breaking-change, feature, enhancement, bug, dependencies, documentation,
deprecations, or ci.
-->

### 📝 TL;DR

<!-- One or two sentences. Include `Closes #<N>` when this PR resolves an issue. -->

---

### 🎯 Context & Purpose (The "Why")

- **The Problem:**
- **The Value:**
- **Issue:** <!-- https://github.com/alvarorg14/openlicensd/issues/N -->
- **Milestone:** <!-- e.g. v0.8.0 — Make it adoptable -->

### 🛠️ Solution & Approach (The "How")

<!-- Group by area. Mention files, new env vars with defaults, migration numbers, SDK or OpenAPI changes. -->

-

### 🧪 Testing, Risks & Rollout

- **Feature Flag:** N/A
- **Breaking Changes:** No
- **How to Test:**
  1. `make lint`
  2. `make build`
  3. `make test`
  4. <!-- `make test-sdk` / `make vuln` / `make dev-db-reset` if applicable -->

### 👀 Reviewer Focus

<!-- 1–3 files or decisions that most need attention (auth, rate limiting, store queries, migrations). -->

- **`path/to/file`** —

### Checklist

- [ ] Tests added or updated (`make test`; `make test-sdk` if `sdk/go/` changed)
- [ ] Docs updated (`README.md`, `QUICKSTART.md`, `docs/`, `AGENTS.md` as needed)
- [ ] `docs/openapi.yaml` updated if API endpoints or schemas changed
- [ ] New numbered migration added if the schema changed (never edit an applied migration; reviewers need `make dev-db-reset`)
- [ ] Exactly one policy label applied
- [ ] Breaking changes described above, with a migration path
