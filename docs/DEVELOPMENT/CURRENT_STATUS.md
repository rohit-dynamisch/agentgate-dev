# AgentGate — Current Development Status

**Last updated:** 2026-09-16

This file states only what is true *right now*. It is rewritten in place, not appended to — for history, see each checkpoint's own `CLOSURE_SUMMARY.md` in `docs/PHASES/G{N}_WORKSTREAMS/` or `git log`. Full navigation: `docs/README.md`.

---

## Where we are

**Strategy in force:** [`docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`](../PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md) (parallel workstreams, gated by checkpoints).

### Checkpoint Milestones

- **G1 — Authorization Contract Freeze: PASS / CLOSED / FROZEN** (2026-09-12). Lead Architect verdict recorded 2026-09-12. All five workstreams merged into `development`. Frozen contract: `agentgate/internal/decision.Request`/`Result`, documented in [`docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md`](../PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md).
- **G2 — Identity + Tool Governance boundary: PASS / CLOSED** (2026-09-13). Identity claims mapper (`internal/identity`), tool registry with canonical SHA-256 schema fingerprint & drift detection (`internal/toolregistry`), per-tool argument whitelist registry (`internal/argdecl`, resolving O-006), context assembly (`internal/contextassembly`), gateway inspection evidence, and `g2security` QA suite.
- **G3 — Policy Persistence + Governance API: PASS / CLOSED** (2026-09-13). Policy store (`internal/policystore`, memory & Postgres implementations sharing one behavior test), lifecycle manager (`internal/policymanager`, content-addressed SHA-256 versions, atomic activation, cached engine, rollback, preview), admin governance REST API (`internal/govapi`), frontend governance client/state/views, `g3governance` QA suite, and `deploy/g3` reproducible Postgres environment.
- **G4 — Governance Workflow Integration: PASS / CLOSED / FROZEN** (2026-09-14). Mutation audit events (`internal/auditevents`), governance-to-decision integration service (`internal/governanceintegration`, active-vs-candidate dry-run compare), `POST .../policies/{version}/dryrun` REST endpoint, frontend dry-run models/client/views, `g4integration` QA suite (8 DoD invariants), and `deploy/g4` E2E compose topology. Formally approved by Lead Architect 2026-09-14.
- **G5 — Durable Audit Boundary: PASS / CLOSED / FROZEN** (2026-09-14). Append-only `audit_events` persistence in PostgreSQL (`internal/audit`), tamper-evident SHA-256 row chaining (`prev_hash` + `row_hash`), independent out-of-process `ChainVerifier`, pre-persistence argument redaction (`redact.go`), fail-closed audit enforcement (audit failure => decision `DENY`, resolving O-002), database immutability trigger (`prevent_audit_modification`), database privilege separation (`agentgate_app` vs `agentgate_migrator`), `g5audit` QA suite (9 DoD invariants), and `deploy/g5` reproducible topology. Formally approved by Lead Architect 2026-09-14.
- **G6 — Real MCP End-to-End Enforcement: SUBMITTED FOR REVIEW** (2026-09-16). Implementation and independent QA verification completed across all 4 workstreams (Tasks 1–11). Production Envoy v3 `ext_authz` gRPC service (`internal/authz`), JSON-RPC 2.0 tool call adapter, durable PostgreSQL audit with SHA-256 row chaining, live `agentgateway:v1.4.0` integration, independent black-box E2E enforcement suite (`qa/g6enforcement`, 12/12 DoD scenarios passing, live outage fail-closed verified, live service recovery verified). Handoff package prepared; Lead Architect sign-off pending.
- **Next Checkpoint: G7 — Downstream Scoped Identity & Token Exchange** (O-001 concrete implementation: downstream scoped MCP credentials, caller/on-behalf-of identity propagation without token passthrough).

---

## What exists and runs today

### Go backend (`agentgate/`)
- Production service executable `cmd/agentgate` (config, structured JSON logging, health/readiness HTTP endpoints, graceful shutdown, and Envoy v3 `ext_authz` gRPC service on `:9001`).
- Frozen Cedar authorization decision core (`internal/decision`, `internal/policy`, `internal/fixturepolicy`) + test-only mock binary `cmd/g1-mock-authz`.
- Production Envoy v3 `ext_authz` gRPC service & adapter (`internal/authz`) converting JSON-RPC 2.0 `tools/call` into `decision.Request` with tool governance and argument whitelist enforcement.
- Identity claims mapper (`internal/identity`) with fail-closed mapping across 4 failure classes.
- Tool registry (`internal/toolregistry`) with canonical SHA-256 schema fingerprinting and drift detection.
- Argument declaration whitelist registry (`internal/argdecl`) and context assembler (`internal/contextassembly`).
- Policy persistence store (`internal/policystore`) with Memory and PostgreSQL implementations, partial unique active index, and embedded schema migrations.
- Policy lifecycle manager (`internal/policymanager`) with atomic activation, concurrency-safe cached engine swap, rollback, preview, and mutation audit events (`internal/auditevents`).
- Admin governance REST API (`internal/govapi`) with constant-time key comparison authentication.
- Governance-decision integration bridge (`internal/governanceintegration`) with dry-run candidate-vs-active comparison.
- **Durable audit boundary (`internal/audit`):** Postgres append-only persistence, SHA-256 row chaining, independent `ChainVerifier`, pre-persistence argument redaction, fail-closed enforcement, and DB immutability triggers.

### Gateway / MCP (`gateway/` & `deploy/g6/`)
- Reviewable `agentgateway` configuration (`gateway/config/g1-agentgateway.yaml` and `deploy/g6/agentgateway.yaml`) targeting AgentGate via `policies.extAuthz` (Envoy v3 gRPC protocol, request body inclusion).
- Pinned `agentgateway:v1.4.0` verified with empirical probe tests (`docs/PHASES/G6_WORKSTREAMS/G6_GATEWAY_CONTRACT.md`).
- Independent Go verification harness (`gateway/harness/`) asserting wire fixtures over real HTTP.

### Frontend contract layer (`frontend/`)
- Framework-agnostic TypeScript library (`src/api/governanceClient.ts`, `src/models/`, `src/state/`, `src/view/`) with full Vitest test coverage (63 tests across 7 suites) for governance, dry-run comparison, and rollback rendering.

### Deploy environments (`deploy/`)
- `deploy/g3/`: Reproducible Postgres container + readiness probe + migration verification script.
- `deploy/g4/`: Integrated governance-to-decision E2E topology with curl lifecycle runbook.
- `deploy/g5/`: Reproducible Postgres topology with DB privilege separation (`agentgate_app` vs `agentgate_migrator`) and audit immutability triggers.
- `deploy/g6/`: Integrated 4-service topology (`g6-postgres`, `g6-agentgate`, `g6-agentgateway`, `g6-probe-mcp`) on `g6net` with automated clean-run matrix runner `deploy/g6/run-e2e-matrix.ps1`.

### QA & Security proof suites (`agentgate/qa/`)
- Six independent QA suites importing zero `internal/*` packages:
  1. `qa/g1blackbox`: Out-of-process contract verification against mock binary.
  2. `qa/g2security`: Identity, tool abuse, argument whitelist, and trust boundary proofs.
  3. `qa/g3governance`: Policy persistence, atomic activation, and Postgres lifecycle invariants.
  4. `qa/g4integration`: Full governance-to-decision loop (8 DoD invariants).
  5. `qa/g5audit`: Durable audit persistence, SHA-256 hash chaining, tamper detection, redaction, fail-closed, and DB privilege separation (9 DoD invariants).
  6. `qa/g6enforcement`: Full black-box E2E enforcement suite (12 DoD scenarios, live outage fail-closed, live recovery, Postgres SHA-256 chain verification).

---

## Not yet started (The Next Boundaries)

- Downstream credential mechanism & token exchange (O-001, Gate G7).
- Supported MCP revision multi-version matrix (O-004).
- Shipped UI application (frontend is currently contract/view layer only).

---

## Current blockers

None active. Next checkpoint **G7 — Downstream Scoped Identity & Token Exchange** is defined.

---

## Open architectural decisions

See [`docs/DECISIONS/OPEN_DECISIONS.md`](../DECISIONS/OPEN_DECISIONS.md):
- **Open:** O-001 (downstream identity), O-004 (supported MCP revision).
- **Resolved:** O-002 (audit durability, resolved G5), O-003 (gateway conformance, resolved G6), O-005 (tool fingerprinting, resolved G2), O-006 (argument authorization model, resolved G2), O-007 (execution identity, resolved G2/G5), O-008 (ext_authz transport mapping, resolved G6).

---

## Status-update rule

Rewrite this file in place after any checkpoint transition or other meaningful state change. Do not append historical narrative here — that belongs in the relevant checkpoint's own docs. Do not claim a capability is complete until implementation and required verification have actually occurred.
