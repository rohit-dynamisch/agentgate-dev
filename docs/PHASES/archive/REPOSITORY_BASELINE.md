# Repository Baseline

**Task:** TASK-00-02 — Repository Reconnaissance and Development Baseline
**Date:** 2026-08-22
**Author:** Senior Software Engineer (AI coding agent)

> **Historical snapshot (archived 2026-09-13):** this describes the repository as it existed on
> 2026-08-22 — documentation-only, before any Go module or application code was created. It is
> permanently a point-in-time record, not a document that will ever describe "current state"
> again. For the actual current state, see `docs/DEVELOPMENT/CURRENT_STATUS.md`; for navigation,
> `docs/README.md`.

---

## Current State

The repository is a **fresh documentation-only repository**. It contains exactly one commit,
which established the AI-driven development workflow and the canonical planning/architecture
documents. No product code, no build tooling, no dependency manifests, and no CI configuration
exist yet.

This matches what `docs/DEVELOPMENT/CURRENT_STATUS.md` already claims ("The project repository
has been newly created. No production implementation has been established yet."). Reconnaissance
confirms that claim is accurate — there is nothing hidden in the tree that contradicts it.

The working tree currently has one staged-but-empty change: `.gitignore` is `git add`-ed but its
staged blob is byte-identical to the committed blob (both are `docs/prompts`, 12 bytes, no
trailing newline). This is a no-op stage, not a real pending edit.

## Repository Structure

```
.
├── .git/
├── .gitignore                          (1 line: "docs/prompts", no trailing newline)
├── CLAUDE.md                           (coding-agent operating rules)
└── docs/
    ├── PROJECT_DEFINITION.md           (product/architecture source of truth, §0–§13)
    ├── TECH_STACK.md                   (technology stack proposal, §0–§5)
    ├── AI/
    │   └── AI_DEVELOPMENT_MODEL.md     (roles, operating loop, task discipline)
    ├── DEVELOPMENT/
    │   ├── MASTER_PLAN.md              (phase plan, Phase 0 → Phase 7)
    │   └── CURRENT_STATUS.md           (status tracker — being updated by this task)
    ├── DECISIONS/
    │   └── OPEN_DECISIONS.md           (O-001 … O-007, unresolved architectural items)
    └── prompts/
        └── 1.md                       (this task's own prompt text; ignored by git)
```

Nothing else exists at any depth. No `agentgate/`, `gateway/`, `client/`, `toy-server/`,
`policies/`, `migrations/`, or `keycloak/` directories — none of the paths referenced by
`PROJECT_DEFINITION.md §11` ("Repo, stack, conventions") currently exist. That section describes
the **POC's** layout as background/history, not the state of this repository.

## Existing Tooling

- **Git:** initialized, single local branch (`master`), no remotes configured, no tags, no other
  branches. `core.filemode=false`, `core.ignorecase=true`, `core.symlinks=false` — Windows-typical
  settings, no unusual overrides. No custom hooks in `.git/hooks/` (only samples).
- **`.gitignore`:** exists but only ignores `docs/prompts` (the task-prompt scratch file this
  agent was invoked with). There is **no** ignore rule yet for language/tooling artifacts (`*.env`,
  `/bin`, `/dist`, `node_modules/`, `vendor/`, IDE folders, Go build caches, etc.) because there is
  no code yet to produce them.
- **README:** none. There is no top-level `README.md` pointing a new contributor at
  `docs/PROJECT_DEFINITION.md` or `CLAUDE.md`.
- **CI/CD:** none. No `.github/workflows/`, no other CI config of any kind, despite
  `TECH_STACK.md §2.9` specifying GitHub Actions (`go test`, `golangci-lint`, `govulncheck`, fuzz,
  testcontainers, SBOM/cosign) as required tooling.
- **Docker / Compose:** no `Dockerfile`, no `docker-compose.yml`/`compose.yaml` anywhere in the
  tree.
- **Package managers / build files:** no `go.mod`, `go.sum`, `package.json`, `Makefile`, or any
  other manifest.
- **Editor/IDE config:** no `.vscode/`, `.editorconfig`, or similar.
- **Local toolchain availability** (checked on this machine, informational only — not committed to
  the repo):
  - Go **1.26.4** available and functional. A local smoke test (`go test` on a throwaway module)
    built and ran a real test binary successfully in this environment. This is worth flagging
    because `PROJECT_DEFINITION.md §11` records a POC-era gotcha ("Windows App Control blocks
    locally-built test binaries; run tests in Docker"). That gotcha did **not** reproduce here —
    it is evidently host/policy-dependent, not a universal fact. Treat it as "verify per machine,"
    not as settled.
  - Docker CLI present, but the Docker daemon is **not currently running/reachable** on this
    machine (`npipe` connection to Docker Desktop failed). `docker compose` plugin (v5.1.1) is
    installed.
  - Node v24.11.0, npm and pnpm present — irrelevant to the Go backend, but available if a future
    phase adds an operator-UI frontend.
  - `make`, `golangci-lint`, `psql` are **not installed** locally.

## Existing Application Code

**None.** No Go source files, no `internal/`, `cmd/`, or `pkg/` trees, no test files, no SQL
migrations, no Cedar policy files, no gateway config. There is nothing to inventory here beyond
confirming the negative.

## Architecture-Relevant Findings

1. **The repo is currently suitable as a monorepo, trivially — because it is empty.** There is no
   existing structural conflict with the intended layout (`agentgate/`, `gateway/`, `client/`,
   `toy-server/`, `policies/`, `migrations/` as separate top-level concerns per
   `PROJECT_DEFINITION.md §11`, refined by whatever Phase 1's task defines). Adopting that layout,
   or any reasoned alternative, is unconstrained by anything already committed. This is a decision
   for Phase 1 task definition, not something reconnaissance should predetermine.
2. **No dependency or configuration drift exists yet**, because no dependencies or configuration
   have been introduced. `TECH_STACK.md`'s stack (Go 1.26+, cedar-go, pgx/v5, agentgateway,
   Keycloak, SpiceDB-optional) is unopposed by anything in the tree.
3. **`docs/prompts/` is gitignored.** This appears to be an intentional convention for storing raw
   task-invocation prompts outside version control. It is consistent (only one file exists, and it
   is excluded), but it is not documented anywhere as a convention — a future contributor won't
   know why it exists or whether new files belong there. Worth a one-line note in `CLAUDE.md` or a
   `docs/prompts/README.md` if the team intends to keep using this pattern, but that is a
   documentation nicety, not a blocker.
4. **Local Go toolchain matches the planned language version** (1.26.4 installed vs. "Go 1.26+"
   required) — no version gap to resolve before Phase 1.

## Conflicts / Concerns

No factual inconsistency was found between `PROJECT_DEFINITION.md` and `TECH_STACK.md` that
requires correcting either document — `TECH_STACK.md` explicitly documents itself as a later,
verified revision of certain `PROJECT_DEFINITION.md` assumptions (RFC 8693 → ID-JAG, MCP protocol
revision), and both documents already cross-reference that divergence honestly. No new conflict was
discovered during this reconnaissance.

One thing to flag, not as a document defect but as a planning input: `PROJECT_DEFINITION.md §11`
("Repo, stack, conventions") reads, at a glance, as if it describes *this* repository's current
layout (`agentgate/cmd/agentgate`, `gateway/config.yaml`, etc.). It does not — those paths do not
exist here. It is POC-era description carried into the definition document as historical context.
Nothing needs to change in the document, since it is captioned as background (§5 "What is built"
refers to the POC's own git history, not this repo's). But a coding agent skimming quickly could
mistake that section for "files that already exist here." Flagging this so the Lead Architect can
decide whether to add a clarifying note; not correcting it unilaterally since it's not a factual
error, just a clarity risk.

No security issue was found in the repository itself (there is no code to have a vulnerability).
The security-relevant open items are already correctly captured in
`docs/DECISIONS/OPEN_DECISIONS.md` (O-001 downstream identity, O-002 audit durability invariant,
O-003 agentgateway conformance boundary, O-004 MCP revision support, O-005 tool
fingerprinting, O-006 argument authorization, O-007 execution identity). None of them are
resolved by, or resolvable from, reconnaissance alone.

## Missing Foundations

These are genuinely absent and will be needed before or during Phase 1, in roughly the order they
will bite:

1. **Top-level repository layout for the actual Go module(s)** — no `go.mod` exists yet. Phase 1's
   task must decide and create the module path/layout (this is implementation, correctly out of
   scope for this task).
2. **`.gitignore` coverage for language artifacts** — currently only ignores `docs/prompts`; will
   need Go build output, `.env` files, IDE folders, etc. once code exists. Not urgent today; will
   become necessary the moment Phase 1 starts committing code.
3. **README.md** — no entry point exists for a human or agent landing in the repo cold. Not
   strictly blocking (`CLAUDE.md` + `docs/` cover it), but its absence means there's no signpost
   from GitHub's default rendered view to `docs/PROJECT_DEFINITION.md`.
4. **CI baseline** (`.github/workflows/`) — `MASTER_PLAN.md` Phase 0 explicitly lists "CI
   baseline" as part of this phase's scope, and it does not exist yet. This is a genuine Phase 0
   gap, not a Phase 1 concern — flagging it because Phase 0's own definition (in
   `MASTER_PLAN.md §4`) is broader than what TASK-00-01/00-02 have delivered so far.
5. **Local dev environment definition** (Docker Compose stack for agentgateway + Postgres +
   Keycloak, or equivalents) — needed once Phase 1's vertical slice is built, not before.
6. **License file** — `TECH_STACK.md §4 (O-4)` records the Apache-2.0 vs AGPL/BUSL choice as an
   open management decision. No `LICENSE` file exists yet, correctly, since that decision is
   unresolved. Flagging only so it isn't forgotten once O-4 is settled.

None of the above blocks *starting* Phase 1's implementation task per se, but item 4 (CI baseline)
is arguably still owed under Phase 0's own stated scope in the Master Plan and should be raised
with the Lead Architect rather than silently carried into Phase 1.

## Recommended Repository Structure

No structural change is being made by this task — the repository is empty of code, so there is
nothing to reorganize, and inventing a target layout is implementation/architecture work reserved
for the Lead Architect and the Phase 1 task definition, not for reconnaissance. `TECH_STACK.md` and
`PROJECT_DEFINITION.md §11` already describe candidate layouts; recommending one here would risk
"inventing requirements," which this task is explicitly told not to do.

## Phase 0 Exit Assessment

**Not yet ready to exit Phase 0 in full**, though close, and nothing found blocks proceeding to
scope Phase 1's task definition.

- The planning/documentation foundation (`PROJECT_DEFINITION.md`, `TECH_STACK.md`,
  `AI_DEVELOPMENT_MODEL.md`, `MASTER_PLAN.md`, `OPEN_DECISIONS.md`, `CLAUDE.md`) is complete,
  internally consistent, and now has an accurate repository baseline (this document) to sit
  alongside it.
- What remains outstanding against Phase 0's own stated scope in `MASTER_PLAN.md §4`
  ("repository structure, AI-agent instructions, development workflow, architecture/decision
  documentation, **CI baseline**, and project status tracking") is the **CI baseline** — it has not
  been established. Repository structure, instructions, decision documentation, and status
  tracking are all in place.
- No blocker prevents Phase 1 task definition from beginning. A CI baseline can reasonably be
  established either as the last remaining Phase 0 item or folded into Phase 1's first task (e.g.
  alongside `go.mod` creation, since there is nothing for CI to build/test/lint until then). This
  is a sequencing call for the Lead Architect, not a decision this task should make unilaterally.
- No open architectural/security decision (O-001 through O-007) blocks *starting* Phase 1, per
  `MASTER_PLAN.md §6`'s own framing — they block specific pieces of implementation (downstream
  identity, audit durability invariant, etc.), not the initial vertical-slice scaffolding.

**Recommendation:** Proceed to Phase 1 task definition. Raise the CI-baseline gap explicitly to
the Lead Architect for a sequencing decision (own Phase 0 task vs. folded into Phase 1's first
task) rather than assuming either answer.
