---
name: openlicensd-issue
description: Resolve a numbered OpenLicensd GitHub issue on the road to v1.0.0 — fetch the issue, produce a native Cursor plan for review, update the issue lifecycle, implement, run the quality gates, and open a linked PR. Use when the user invokes this skill with a GitHub issue number, or asks to work on, plan, or resolve an OpenLicensd issue.
disable-model-invocation: true
---

# OpenLicensd Issue Resolution

Drives a single GitHub issue from `alvarorg14/openlicensd` to an open pull request.

**Input**: a GitHub issue number (e.g. `42`, `#42`, or the full issue URL).

Repository context lives in `AGENTS.md` at the repo root (always applied) and the release phases in
`ROADMAP.md`. Do not restate them — read them and apply them.

## Prerequisites

- `gh` CLI authenticated against `alvarorg14/openlicensd`, or the `user-github` MCP.
- Clean working tree on an up-to-date `main`. If the tree is dirty, stop and ask what to do with the
  existing changes.
- Docker running if the issue touches the store, the API, or migrations (`make dev-db` is needed for tests).

## Two Phases, Two Modes

The plan is delivered as a **native Cursor plan** in the IDE, not as markdown in the chat. That splits
this skill across Cursor's two modes:

| Phase | Mode | Steps | Ends with |
|-------|------|-------|-----------|
| **Plan** | Plan mode | 1–3 | The user clicks **Build** |
| **Build** | Agent mode | 4–7 | An open PR |

**Enter Plan mode before Step 1.** If the session is in Agent mode, call the mode-switch tool with
`target_mode_id: "plan"` and explain that the issue needs a reviewable plan first; the user consents to
the switch. The user can also do it themselves with Shift+Tab. Clicking **Build** returns the session to
Agent mode with the plan attached, which is where Step 4 picks up.

Keep every write — GitHub metadata, code, commits — in the Build phase. Nothing in Steps 1–3 should
modify the repo or the issue.

## Workflow

Copy this checklist and update it as you go:

```
Issue #<N> Progress:
- [ ] Step 1: Read the issue and its context
- [ ] Step 2: Investigate the codebase
- [ ] Step 3: Write the Cursor plan, then WAIT for Build
--- Build ---
- [ ] Step 4: Claim the issue on GitHub
- [ ] Step 5: Implement
- [ ] Step 6: Run the quality gates
- [ ] Step 7: Open the PR
```

---

### Step 1: Read the Issue and Its Context

Prefer the read-only `user-github` MCP tools (`issue_read`, `list_issues`, `search_pull_requests`) during
the Plan phase — they work regardless of terminal availability. The `gh` equivalents:

```bash
gh issue view <N> --json number,title,body,labels,milestone,assignees,state,url,comments
gh issue view <N> --comments
git status && git branch --show-current
git fetch origin && git log --oneline -5 origin/main
```

Also fetch the parent tracking issue for the milestone (`#67` v0.6.0, `#68` v0.7.0, `#69` v0.8.0,
`#70` v1.0.0, `#71` Post-1.0) when the issue is a sub-issue — it often carries the intent that the
sub-issue omits.

**Issue bodies in this repo are deliberately terse** (often a single line, sometimes just
`Checklist: tests, docs, openapi, migrations, label, breaking changes.`). The title plus the
milestone's phase description in `ROADMAP.md` is the real specification. Expect to derive the
requirements yourself in Step 2 rather than find them written down.

Stop and report instead of proceeding if:

- The issue is already closed, or already assigned to someone else.
- An open PR already references it (`gh pr list --search "<N>"`).
- The issue is in the **Post-1.0** milestone — confirm with the user that they want it now, since it is
  explicitly out of scope for v1.0.

### Step 2: Investigate the Codebase

Never plan from the issue text alone. Ground every plan in the current code.

1. Map the issue to owning packages using the component table in `AGENTS.md` and the `area/*` labels.
2. Read the files you intend to change, plus their tests.
3. Search for prior art — a similar feature usually already exists to mirror
   (e.g. a new list endpoint should follow `server/internal/api/listing.go` and
   `server/internal/store/listing.go`).
4. Identify the cross-cutting obligations the change triggers (see the table below).

For anything spanning more than about three files, delegate the exploration to parallel `explore`
subagents (one per area: server, ui, sdk, docs/helm) and synthesize their findings.

#### Obligations by change type

| If the change touches | You must also |
|-----------------------|---------------|
| Any HTTP endpoint or schema | Update `docs/openapi.yaml`; check `docs/api.md` |
| Database schema | Add a new numbered migration in `server/internal/store/migrations/`; never edit an applied one; note that reviewers need `make dev-db-reset` |
| Config / env vars | Update `server/internal/config/`, the table in `AGENTS.md`, `docs/configuration.md`, `.env.example`, and `charts/openlicensd/values.yaml` |
| Public SDK surface | Update `docs/sdk/go.md` and `sdk/go/README.md`; SDK is stdlib-only and versioned independently |
| Helm values | Update the chart README and bump `Chart.yaml` if appropriate |
| Install or verification steps | Update `QUICKSTART.md` |
| User-facing behavior | Update `README.md` |
| Architecture, packages, or workflow | Update `AGENTS.md` |
| Roadmap scope | Update `ROADMAP.md` |

Never hand-edit `server/internal/static/dist/` — it is generated by `make ui`, and git only tracks a
placeholder `index.html`.

### Step 3: Write the Cursor Plan, Then WAIT for Build

Create the plan as the session's native Cursor plan document, so it opens as an editable file in the
IDE with a **Build** button — do not paste it into the chat as a substitute. Then **stop** and let the
user review, edit, and build it. Do not write code, and do not touch the issue on GitHub, until Build.

Use this structure for the plan document:

```markdown
# Issue #<N>: <title>

<issue URL>

**Milestone**: <milestone> — <phase focus from ROADMAP.md>
**Labels**: <type label> + <area/* labels>
**Interpretation**: <what the issue actually asks for, in 1-3 sentences, given the terse body>

## Scope
In scope:
- <bullet>

Out of scope (and why):
- <bullet>

## Changes
| File | Change |
|------|--------|
| `path/to/file.go` | <what and why> |

## Tests
- <new or updated test, and what it proves>

## Documentation
- <file> — <what changes>

## Risks and decisions
- <breaking change, migration, security consideration, or an open question for the user>

## Steps
- [ ] <ordered implementation to-do, one per logical unit>
- [ ] Run `make lint`, `make build`, `make test`
- [ ] Assign and relabel issue #<N>
- [ ] Open PR with `Closes #<N>`, one policy label, and body per `smart-commit-and-pr` template
```

Plan-specific notes:

- Reference concrete file paths in the **Changes** table — the plan document links them, so the user can
  jump straight to the code while reviewing.
- The **Steps** to-dos become the build checklist. Keep them ordered and independently checkable, and
  keep the lifecycle and PR steps in the list so they are not lost after Build.
- Flag in **Risks and decisions** if the change is breaking, requires a migration, alters authentication
  or rate limiting, or affects the public SDK API — these change the PR label and the release notes.
- Suggest **Save to workspace** when the plan is worth keeping as a design record (breaking changes,
  migrations, new endpoints); plans otherwise live outside the repo.
- If the user edits the plan before building, treat the edited version as authoritative and re-read it at
  the start of Step 5.

### Step 4: Claim the Issue on GitHub

*Build phase — the session is back in Agent mode.* First action after Build: update the issue's
lifecycle so the tracker reflects that work has started.

```bash
gh issue edit <N> --add-assignee alvarorg14
```

Then reconcile the issue's metadata with the approved plan:

- **Type label** — exactly one of `feature`, `enhancement`, `bug`, `documentation`,
  `breaking-change`, `deprecations`, `dependencies`, `ci`. Add `security` alongside it when relevant.
- **Area labels** — one or more of `area/server`, `area/ui`, `area/sdk`, `area/docs`, `area/helm`,
  `area/ci`, matching the files in the plan.
- **Milestone** — must match the phase in `ROADMAP.md`.

Add missing labels freely. If the investigation showed an *existing* label or milestone is wrong
(e.g. labeled `enhancement` but it is really a bug, or scoped to `v0.7.0` but blocks `v0.6.0`), do not
change it silently — say so and ask first.

### Step 5: Implement

Work through the plan's **Steps** to-dos in order, checking each off as it lands. Follow the code style
and conventions in `AGENTS.md`.

- Match the surrounding code: `chi` router patterns, `pgx` store methods, stdlib `testing` (no
  testify), Nuxt UI v4 components, and the `brand`/`navy` design tokens.
- Write tests for new logic in `server/internal/` and `sdk/go/`. Never weaken, skip, or delete an
  existing test to make the suite pass.
- Keep the SDK dependency-free.
- Report any deviation from the plan as you make it. If the deviation changes the scope rather than just
  the mechanics, update the plan document so it stays an accurate record.

### Step 6: Run the Quality Gates

All three must pass before the PR. This mirrors CI.

```bash
make lint    # go vet + golangci-lint + ESLint + sdk
make build   # ui + server
make test    # Go tests; needs Postgres
```

API and store tests need a live database:

```bash
make dev-db          # start Postgres
make dev-db-reset    # required whenever a migration was added or changed
```

Run `make test-sdk` as well when `sdk/go/` changed, and `make vuln` when dependencies changed.

If a gate fails, fix the cause. Do not proceed to Step 7 with a failing gate, and do not disable a
linter rule to get past it without saying so.

### Step 7: Open the PR

Invoke the `smart-commit-and-pr` skill for branching, conventional commits, and PR creation. Apply
these OpenLicensd-specific overrides on top of it:

1. **Branch name** includes the issue number: `<prefix>/<N>-<short-kebab-description>`, e.g.
   `feature/117-pull-request-template`.
2. **Commit footer** references the issue: `Refs: #<N>`.
3. **PR body** must contain `Closes #<N>` so the issue closes on merge — this also advances the
   milestone's tracking issue.
4. **PR carries exactly one policy label**, enforced by `.github/workflows/pr-policy.yml`:

```bash
gh pr edit <PR> --add-label <one-of: breaking-change|feature|enhancement|bug|dependencies|documentation|deprecations|ci>
```

Derive it from the issue's type label. When several could apply, pick by precedence:
`breaking-change` > `deprecations` > `feature` > `enhancement` > `bug` > `documentation` >
`dependencies` > `ci`. Add the matching `area/*` labels to the PR too — they are not enforced but keep
the release drafter accurate.

5. **PR description** must follow the structured template in the `smart-commit-and-pr` skill — read
   `~/.cursor/skills/smart-commit-and-pr/pr-template.md` before writing the body. Fill every section
   from the diff against `main` (`git diff main...HEAD`); remove a section only when it truly does not
   apply. Weave these OpenLicensd-specific details into the template sections:

   | Template section | OpenLicensd additions |
   |------------------|----------------------|
   | **TL;DR** | Include `Closes #<N>`. |
   | **Context & Purpose** | Link the issue URL; state the milestone phase from `ROADMAP.md`. |
   | **Solution & Approach** | New env vars with defaults; migration file numbers; SDK or OpenAPI changes. |
   | **Testing, Risks & Rollout** | `make dev-db-reset` when a migration was added; breaking changes with a migration path; which gates ran (`make lint`, `make build`, `make test`, and `make test-sdk` / `make vuln` when applicable). |
   | **Reviewer Focus** | Files that need careful review (auth, rate limiting, store queries, migrations). |

Finally, report: branch name, commits, PR URL, gate results, and anything left for a follow-up issue.

## Definition of Done

- [ ] `make lint`, `make build`, `make test` all pass
- [ ] Tests added or updated for new logic
- [ ] Every documentation obligation from Step 2 satisfied
- [ ] Issue assigned, correctly labeled, and in the right milestone
- [ ] PR opened with `Closes #<N>`, exactly one policy label, and structured description per `smart-commit-and-pr` template
- [ ] Anything discovered but deliberately out of scope reported to the user as a follow-up

## Edge Cases

- **Issue is a tracking issue** (`#67`–`#71`): do not implement it. List its open sub-issues and ask
  which one to take.
- **Issue is too large for one PR**: propose a split in Step 3 and offer to open follow-up issues for
  the remaining parts.
- **Issue is already fixed in `main`**: report the commit that fixed it and offer to close the issue
  with that reference instead of writing code.
- **Issue is ambiguous even after investigation**: ask the clarifying question in the Plan phase, before
  writing the plan. If the ambiguity is a genuine product decision, put the competing interpretations in
  **Risks and decisions** and let the user resolve it while reviewing the plan.
- **Plan turns out to be wrong during the build**: stop and say so. For a small correction, revise and
  continue; for a wrong approach, the clean recovery is to revert to the pre-Build message and re-plan
  rather than patch it with follow-ups.
- **User skips Plan mode** and asks for implementation directly: honor it, but still produce the Step 3
  content as a chat summary before editing files, so scope is agreed before code exists.
- **Security-sensitive fix**: do not describe the vulnerability in a public PR. Stop and point the user
  to `SECURITY.md` and private advisories.
