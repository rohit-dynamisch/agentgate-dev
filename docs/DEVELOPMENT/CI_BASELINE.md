# CI Baseline

**Task:** TASK-00-03 — Establish CI Baseline
**Date:** 2026-08-22
**Author:** Senior Software Engineer (AI coding agent)
**Workflow file:** `.github/workflows/ci.yml`

---

## Purpose

Provide the minimum useful continuous-integration foundation so that, from the moment Phase 1
introduces the Go module, every push and pull request is automatically checked for formatting,
correctness, and known vulnerabilities — without anyone having to remember to wire CI up after the
fact. `docs/DEVELOPMENT/MASTER_PLAN.md §4` lists a "CI baseline" as part of Phase 0's own scope,
and `docs/DEVELOPMENT/REPOSITORY_BASELINE.md` recorded it as the one Phase 0 item still
outstanding. This task closes that gap.

CI is deliberately kept to validation only at this stage: formatting, static analysis, tests, lint,
and vulnerability scanning. It is not a deployment pipeline and is not trying to anticipate
Phase 1's actual module layout.

## Current Repository Limitation — No Application Code Yet

The repository is documentation-only (confirmed in `docs/DEVELOPMENT/REPOSITORY_BASELINE.md`):
there is no `go.mod`, no Go source, no tests, and no dependency manifest of any kind. This means a
CI workflow that unconditionally runs `go build`/`go test`/etc. would fail on every commit today —
not because anything is broken, but because there is nothing to build.

Rather than inventing a placeholder Go module or dummy production code purely to give CI something
to pass on (explicitly out of scope for this task), the workflow **detects whether `go.mod` exists**
as its first step and gates every subsequent Go-specific step on that detection:

- If `go.mod` is absent, the job logs a notice explaining why the Go steps are skipped and
  otherwise finishes successfully — a false CI failure serves nobody, but the workflow does not
  pretend to have validated something that does not exist.
- If `go.mod` is present, every Go-specific step runs unconditionally and must pass. There is no
  code path where the presence of a Go module lets a failing check be silently skipped — a broken
  `go vet`, failing test, lint violation, or vulnerability finding fails the job.

This is the intended and only conditional logic in the workflow. No other part of the pipeline
branches on repository state.

## Checks Planned / Enforced by CI

Once `go.mod` exists, the single `go` job in `.github/workflows/ci.yml` runs, in order:

1. **`gofmt -l .`** — fails the build if any file is not formatted per `gofmt`. This is the
   cheapest, fastest signal and runs first.
2. **`go vet ./...`** — static analysis for common correctness mistakes.
3. **`go test -race ./...`** — the full test suite, race-detector enabled. Per
   `docs/TECH_STACK.md §1`, agentgate is a concurrent, latency-sensitive decision service; the race
   detector is treated as a default requirement, not an opt-in extra.
4. **`golangci-lint`** (`golangci/golangci-lint-action@v6`, default rule set — no project-specific
   `.golangci.yml` exists yet, so the action's own defaults apply until Phase 1 defines one) — the
   linter named in `docs/TECH_STACK.md §2.9` / §1.
5. **`govulncheck`** (`go run golang.org/x/vuln/cmd/govulncheck@latest ./...`) — known-vulnerability
   scanning against the module's actual dependency graph, also named in `docs/TECH_STACK.md §2.9`.

All five checks were selected directly from `docs/TECH_STACK.md`; nothing was added that the tech
stack document does not already call for.

Triggers: every push to `main`/`master` and every pull request. A `concurrency` group cancels
superseded runs on the same ref to avoid queueing stale jobs. `permissions: contents: read` is set
at the workflow level (least privilege — nothing in this workflow needs write access).

## Intentionally Deferred

Per the task's explicit scope boundary and `docs/DEVELOPMENT/MASTER_PLAN.md`'s phased plan, the
following are **not** part of this baseline and are left for later phases:

- Docker image builds
- SBOM generation (CycloneDX)
- Cosign image signing
- Any deployment or release workflow
- Integration tests against real backing services (Postgres, agentgateway, testcontainers) —
  `docs/TECH_STACK.md §2.9` calls for these, but they require the application and its dependencies
  to exist first
- Fuzz testing of the request/body parser — same reason; the parser does not exist yet
- A project-specific `.golangci.yml` ruleset — deferred to whichever Phase 1 task first adds Go
  code, so the ruleset can be chosen against real code rather than speculatively
- Build-matrix concerns (multiple OS/Go versions) — not warranted before there is a single module
  to build

None of these were partially implemented or stubbed. They are absent, not disabled.

## How CI Will Evolve as Phase 1 Introduces the Go Module

No changes to `.github/workflows/ci.yml` are expected to be required merely because `go.mod` is
created — the "Detect Go module" step will find it and every subsequent step activates
automatically. Expected follow-on work, to be scoped as its own task(s) rather than assumed here:

- Add a `.golangci.yml` once the module's package layout exists, so lint rules can target real
  code instead of being guessed in advance.
- Consider splitting the single job into a matrix (e.g., lint vs. test) only if runtime or
  parallelism actually becomes a problem — not preemptively, per the instruction to prefer one
  workflow over unnecessary fragmentation at this stage.
- Add integration-test and fuzz jobs once there is a request parser and a docker-compose dev stack
  to test against (later phase, per `docs/TECH_STACK.md §2.9`).
- Add Docker/SBOM/signing/deployment workflows only when those phases are reached.

## Assumptions

- CI runs on GitHub-hosted `ubuntu-latest` runners — no self-hosted runner infrastructure is
  assumed or required at this stage.
- `go-version-file: go.mod` will be used to pin the Go toolchain version once the module exists,
  so the Go version CI uses always matches whatever the module declares (currently planned as
  "Go 1.26+" per `docs/TECH_STACK.md §1`) — this avoids hardcoding a Go version in the workflow
  that could drift from the module's own declaration.
- `golangci-lint` and `govulncheck` have no version pinned in `docs/TECH_STACK.md`, so this
  workflow does not invent one: it uses the lint action's own default tool version and runs
  `govulncheck` via `go run ...@latest` against the real dependency graph, rather than pinning a
  release tag that has not been through the same review as the versions the tech stack document
  does specify.
- No repository secrets, external services, or network-dependent test infrastructure are required
  by this workflow — it needs nothing beyond the checked-out repository and the standard GitHub
  Actions toolchain setup actions.
