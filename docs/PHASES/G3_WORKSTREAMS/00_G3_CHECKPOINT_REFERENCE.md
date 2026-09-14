# AgentGate v1 — G3 Checkpoint Reference

## Planning authority
**Primary:** `AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN`  
**Secondary:** `AGENTGATE_V1_15_DAY_PRODUCTION_PLAN`

G2 is CLOSED/FROZEN. G3 extends the frozen G1 decision contract and G2 identity/tool-governance boundary; it must not redesign either.

## G3 objective
Establish persistent, versioned, validated policy lifecycle and the governance API contract required by G4 and G5.

## Gate G3 — End of Day 5
**Streams:** Go Backend + Frontend/UI + QA/Security + DevOps.

Must agree on:
- policy states
- policy version/hash
- candidate/active semantics
- create/validate/preview/activate/rollback API
- `workspace_id`
- migration behavior
- error model

DoD:
- running backend API and deterministic migration
- frontend workflow against real service or contract-compatible mock
- invalid candidate cannot activate
- activation is atomic
- rollback selects a known version
- active version is explicit
- concurrent reads never observe invalid active policy
- clean environment can recreate DB and migrations

## Invariants
- Fail closed.
- G1 `decision.Request` / `decision.Result` stays frozen.
- Unknown/unclassified tools remain denied.
- Policy evaluation errors deny.
- Policy provenance remains exact.
- Policy activation is atomic and rollbackable.
- Policy mutation requires authenticated/authorized administration.
- O-001, O-003, O-004, O-005, O-007, O-008 remain open unless explicitly resolved.
- O-006 is closed from G2 and must not be reopened.
- O-002 audit-durability invariant is resolved; exact durable audit mechanism belongs to G5.

## In scope
PostgreSQL policy persistence; version/hash; candidate/active/history lifecycle; validation; governance API; workspace scoping; deterministic migrations; concurrency/atomicity; minimum authenticated admin boundary; frontend contract; independent QA; reproducible DB environment.

## Out of scope
Real MCP enforcement/G6; downstream credentials/G7; final durable decision audit/G5; HITL; mandatory SpiceDB; ML risk scoring; custom MCP proxy; silent OPEN_DECISIONS resolution; G1 contract redesign.

## Contract sequence
Architecture decision → contract → fixture/mock → implementation → contract tests → integration.

Do not create `CLOSURE_SUMMARY.md` until G3 is merged/closed and architect-approved.
