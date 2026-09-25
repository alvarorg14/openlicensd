---
name: openlicensd-issue
description: Plan and resolve one numbered GitHub issue in alvarorg14/openlicensd through an approved implementation, quality gates, and a linked pull request. Use when the user provides an OpenLicensd issue number or URL, or asks to plan, work on, or resolve an OpenLicensd GitHub issue.
---

# OpenLicensd Issue Resolution

Drive one `alvarorg14/openlicensd` issue to an open pull request. Accept `42`, `#42`, or an issue URL. Read and apply root `AGENTS.md` and `ROADMAP.md`; use `gh` for GitHub operations.

## Preconditions

- Start from a clean working tree based on current `main`; stop if dirty and ask what to do.
- Require Docker/PostgreSQL for API, store, or migration tests.
- For security-sensitive issues, stop and direct the user to `SECURITY.md` and private advisories.

## Workflow

Track: read issue → investigate → present plan and wait for approval → claim → implement → run gates → open PR.

### 1. Read issue and context

Use read-only commands first:

```bash
gh issue view <N> --json number,title,body,labels,milestone,assignees,state,url,comments
gh issue view <N> --comments
git status && git branch --show-current
git fetch origin && git log --oneline -5 origin/main
```

Read its milestone tracking issue when relevant (`#67` v0.6.0, `#68` v0.7.0, `#69` v0.8.0, `#70` v1.0.0, `#71` Post-1.0). Stop for closed, already-assigned, already-fixed/already-PR'd, or Post-1.0 issues; for tracking issues, list open sub-issues and ask which to take.

### 2. Investigate before planning

Map ownership from `AGENTS.md` and `area/*` labels; read intended files and tests; search for analogous code; identify cross-cutting obligations:

| Change | Also check/update |
|---|---|
| HTTP endpoint/schema | `docs/openapi.yaml`, `docs/api.md` |
| Database schema | New numbered migration; tell reviewers `make dev-db-reset` |
| Config/env | config, `AGENTS.md`, `docs/configuration.md`, `.env.example`, Helm values |
| Public SDK | `docs/sdk/go.md`, `sdk/go/README.md` |
| Helm/install/user behavior | chart README, `QUICKSTART.md`, `README.md` |
| Architecture/roadmap | `AGENTS.md`, `ROADMAP.md` |

Never hand-edit `server/internal/static/dist/`; use `make ui`.

### 3. Present a reviewable plan, then wait

Do not modify code, issue metadata, or GitHub before approval. Use Codex Plan mode when available, otherwise present this in chat:

```markdown
# Issue #<N>: <title>
<issue URL>
**Milestone**: <milestone> — <roadmap focus>
**Labels**: <type> + <areas>
**Interpretation**: <requirement>
## Scope
In scope: <items>
Out of scope: <items and reasons>
## Changes
| File | Change |
|---|---|
| `path` | <what and why> |
## Tests
- <test and proof>
## Documentation
- <file and change>
## Risks and decisions
- <breaking change, migration, security concern, or question>
## Steps
- [ ] <ordered implementation unit>
- [ ] Run `make lint`, `make build`, `make test`
- [ ] Assign/relabel issue and open PR with `Closes #<N>`
```

Flag breaking changes, migrations, auth/rate-limit work, and SDK changes. If asked to implement immediately, still summarize this plan and confirm scope before editing.

### 4. Claim and reconcile after approval

```bash
gh issue edit <N> --add-assignee alvarorg14
```

Ensure exactly one type label (`breaking-change`, `feature`, `enhancement`, `bug`, `documentation`, `deprecations`, `dependencies`, `ci`), plus relevant `security` and `area/*` labels, and the correct roadmap milestone. Ask before changing an existing incorrect label or milestone.

### 5. Implement and verify

Follow the approved steps and project conventions. Add tests; never weaken existing tests. All mandatory gates must pass:

```bash
make lint
make build
make test
```

Use `make dev-db` and `make dev-db-reset` for migration changes; run `make test-sdk` for SDK changes and `make vuln` for dependency changes.

### 6. Open the PR

Use branch `codex/<N>-<short-kebab-description>`, conventional commits with `Refs: #<N>`, and a PR body based on `.github/pull_request_template.md` and `git diff main...HEAD`. Include `Closes #<N>`, exactly one policy label, matching `area/*` labels, gate results, migration/rollout guidance, and reviewer focus. Report branch, commits, PR URL, and deferred follow-ups.

## Recovery

If already fixed, report the fixing commit. If too large, propose a split. If ambiguous, ask before approval. If implementation invalidates the plan, stop and re-plan material changes.
