# Current State Survey — AgentGate repo (docs-site research note)

> Repository snapshot researched on **2026-09-14** (branch `development`,
> HEAD `e45ba3f`). This note records the verified facts the documentation site
> is built on. It is a working artifact — see `docs-site/artefacts/README.md`.

## Verified via git history (52 non-merge commits, oldest→newest)

- 2026-08-22 bootstrap + workflow + CI baseline; 2026-08-24 Phase-1 start.
- 2026-09-07 Go module scaffold; 2026-09-09 invariants + 15-day plan;
  2026-09-10 policy context tests; 2026-09-11 G1 contracts.
- **G1 closed 2026-09-12** (freeze + corrective closeout commits
  `edefea6`, `c52a0e7`, `d338cad`, `983aa82`, `dad718d`, `2ae61ad`, `a6d11f2`).
- **G2 closed 2026-09-13** (`c6a5acf`, `f7eb783`, `ffb02cd`, `d1d4907`,
  `0d6a6b6`, `5af42d1`, `6da6bcb`, `e81de55` — O-006 resolution).
- **G3 closed 2026-09-13** (`b77d7a9`, `f91db10`, `e62f245`, `0d616f6`,
  `6f34d88`, `3ccf0e0`, `b998ec2`, `b70eacf`).
- **G4 PASS/CLOSED/FROZEN 2026-09-14** (`39f944b`, `393cdd9`, `a482e33`,
  `a3ce597`, `85afa5c`, `48d74fc`, `fef6484`, `ed3f4d8`, `865e0e7`,
  `e45ba3f` = formal closure summary). Next gate: **G5 — Durable Audit
  Boundary** (per G4 closure §6).

## Module / toolchain facts (verified from files)

- Go module `github.com/Dynamisch-LLC/agentgate`, `go 1.26` in
  `agentgate/go.mod` (CI pins toolchain from the same file).
- Deps: `cedar-go` (policy), `pgx/v5` (Postgres). Frontend: strict TS,
  ES2022/NodeNext, Vitest, zero runtime deps, no UI framework.
- Node for docs-site: v24.11.0, npm 11.6.1. Docusaurus 3.10.2 (classic TS) +
  `@docusaurus/theme-mermaid`.

## Code-level anchors (file:line — verified by reading)

- Config: `agentgate/internal/config/config.go` — env vars at :65-82,
  defaults at :22-28, Validate() at :93-104 (blank HTTP_ADDR/ADMIN_TOKEN or
  non-positive SHUTDOWN_TIMEOUT fail startup; LOG_LEVEL ∈ debug|info|warn|error).
- HTTP surface: `agentgate/internal/httpserver/server.go` — /healthz (200 "ok"),
  /readyz (:101-120, readinessCheck → 503 "dependency not ready: …").
- Decision domain: `agentgate/internal/decision/types.go` — Decision
  ALLOW/DENY only (:10-13); 8 reason codes (:19-55); Identity (:62), ToolRef
  (:80), ToolClassification (:91), AttributeValue closed set string/int64/bool
  (:104-139); Request (:155) / Result (:182): all five result JSON fields always
  emitted (wire.go :136-142), execution_id echoed on every path.
- Mock transport: `agentgate/internal/mockauthz/handler.go` — only `/evaluate`,
  POST-only (:53-61), `DisallowUnknownFields` (:66) → transport 400;
  valid-but-invalid body → HTTP 200 DENY. wire.go :46-91 attribute decoding
  (explicit-null → zero attr → malformed_request).
- Fixture policy: `agentgate/internal/fixturepolicy/fixturepolicy.go` — roles
  reader/admin/payer/broken :16-22, risks :24-26, arg `amount` :30, CedarSource
  with deliberate unguarded `context.amount` ("broken" → genuine
  evaluation_error) :47-89.
- Governance: govapi handler routes :39-46 (validate/rollback/activate/preview/
  dryrun/get/list/create — **all** behind AdminAuthMiddleware); middleware
  constant-time; default admin token `agentgate-admin-secret-dev`;
  `NewHandler(mgr, token, govIntegration)` — nil integration → dryrun
  unavailable (:24-31). types.go holds all request/response wire shapes.
- Lifecycle: `policystore` interface :10, memory :11 file, postgres :19 file,
  embedded migration `001_create_policies.sql`, partial-unique active index;
  `ComputeVersion` = SHA-256. `policymanager.Validate/CreateCandidate
  (content-hash idempotent)/Activate (atomic)/Rollback/Preview/
  GetActiveEngine/LoadActivePolicies`.
- G4: `auditevents` MutationEvent + Noop/RecordingListener;
  `governanceintegration` GovernanceDecisionService
  (EvaluateWithActivePolicy, DryRunCompare); `cmd/agentgate` wires it.
- QA proofs: `qa/g1blackbox` builds the real mock binary as subprocess
  (FreeTCPPort:99-106, waitReady:108-121); `g2security` in-process abuse
  matrix; `g3governance` + `g4integration` in-process httptest wrapping the
  real govapi handler; postgres-gated store test via
  `AGENTGATE_TEST_POSTGRES_URL` (store_test.go :189-207).
- Gateway: `gateway/config/g1-agentgateway.yaml` single governed route;
  `gateway/harness` own module, `WouldForwardToBackend` (scenario.go :152)
  true **only** on 200 + ALLOW; harness skips if mock unreachable
  (g1_scenarios_test.go :31).

## Discrepancies / honest notes (flagged in docs, NOT silently resolved)

1. `docs/DEVELOPMENT/CURRENT_STATUS.md` stale (says "G2 not yet started";
   last content 2026-09-12) while git/closures show through G4 closed.
2. `deploy/g1/*` compose "written but never run"; G1 smoke was direct-process
   (`DEVOPS_G1_ENVIRONMENT.md`). DeviOps G1 closure claims docker build+run —
   flag kept: treat g1 container path as unverified.
3. `deploy/g3`/`deploy/g4` qa-probe containers pass `AGENTGATE_BASE_URL`/
   `AGENTGATE_ADMIN_TOKEN` env, but the g3governance/g4integration suites
   build their own in-process httptest (memory store) and do not read those;
   only `AGENTGATE_TEST_POSTGRES_URL` is consumed (policystore). A
   "test against the running container" mode is intended, not implemented.

## Open decisions (as of snapshot)

Open: O-001 (critical, downstream creds), O-003, O-004, O-005, O-007
(O-007 also carries multi-node cache invalidation per G4 closure §2), O-008
(explicitly deferred to G6 — no gateway-side shim allowed). Resolved:
O-002 (2026-09-09, audit-durability invariant; concrete engine = G5),
O-006 (2026-09-13, argdecl whitelist).

## Site-building choices made from this survey

- `docusaurus.config.ts`: url `https://rushi-dynmsh.github.io`, baseUrl `/`
  (GitHub-Pages-shaped placeholders; deploy target not fixed — noted in site
  README), mermaid on, `onBrokenLinks: throw`, editUrl → repo
  `tree/main/docs-site/`, blog disabled.
- Site `docs/` content is a synthesis; the repo `docs/` tree remains
  authoritative on conflict.