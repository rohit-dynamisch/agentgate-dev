# AgentGate v1 — G4 Checkpoint Reference

## Planning authority
Primary: `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`
Secondary: `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md`

G3 is CLOSED/FROZEN. G4 extends the frozen G1 decision core, G2 context-governance boundary, and G3 persistent policy lifecycle without redesigning them.

## Objective
Prove the end-to-end governance workflow: candidate → validation/dry-run → activation → changed authorization outcome → rollback → restored outcome, with policy-mutation audit hooks and accurate UI state.

## Gate G4
A real integrated AgentGate path must demonstrate:
1. known active policy A;
2. candidate B validation;
3. dry-run B without mutating A;
4. predicted/observed impact;
5. atomic activation of B;
6. decision evaluation using B with B provenance;
7. rollback to known A;
8. subsequent evaluation using A with A provenance.

## Workstreams
- W1 Go Backend/Integration: connect G3 lifecycle to G1 decision engine; define mutation-event hook for G5.
- W2 Frontend/UI: integrate validate/dry-run/activate/rollback with server-confirmed state.
- W3 QA/Security: independent black-box proof and negative cases.
- W4 DevOps/Integration: executable deterministic E2E topology/harness.

## Invariants
- Fail closed.
- G1 `decision.Request`/`decision.Result` remains frozen.
- G2 identity/tool/typed-argument boundary remains authoritative.
- G3 candidate/active/historical semantics remain authoritative.
- Dry-run never mutates active state.
- Activation is atomic.
- Rollback targets a known valid persisted version.
- Decision path evaluates the actually active policy.
- Policy provenance identifies the evaluated version.
- UI never claims unconfirmed state.
- Mutation/audit hooks exist for later G5 durable audit; G4 does not build durable audit storage.
- O-001, O-003, O-004, O-005, O-007, O-008 remain open; O-006 remains closed; O-002 durability invariant remains resolved.

## Scope
In: governance→decision integration, dry-run semantics, activation/rollback propagation, provenance correlation, mutation event contract/hook, UI integration, E2E/negative tests, reproducible harness.
Out: G5 durable audit engine, G6 agentgateway/MCP enforcement, G7 downstream credentials, HITL, resource authorization, ML, custom MCP proxy, silent OPEN_DECISIONS resolution.

## Critical boundary
G4 proves AgentGate governance lifecycle → decision-engine integration. It does NOT prove the external agentgateway/MCP transport path. O-008 remains open.

## Success
An operator can safely preview a policy change, activate it, observe authorization change, roll it back, and observe restoration without false state, partial policy state, or bypass of G1/G2 boundaries.
