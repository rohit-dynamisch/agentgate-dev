# AgentGate G2 — Identity + Tool Governance Boundary

## Purpose
G2 turns the frozen G1 authorization contract into a trustworthy request-context boundary: caller identity mapping, authoritative tool identity/classification, schema fingerprinting, and typed per-tool policy-input declarations.

G2 does **not** build the full production agentgateway→MCP path. It establishes the contracts and authoritative components that later integration will consume.

## Frozen inputs
- G1 `decision.Request` / `decision.Result` remain frozen unless the Lead Architect explicitly approves a change.
- agentgateway owns MCP transport/routing/JWT validation; AgentGate owns identity mapping, authorization decision, and audit.
- Downstream resource authorization remains downstream-owned.
- Fail closed on missing, malformed, ambiguous, or unverifiable security context.
- O-001, O-003, O-004, O-005, O-006, O-007 and O-008 remain open unless explicitly resolved by the Lead Architect.

## Goals
1. Define configuration-driven identity/claims mapping.
2. Establish authoritative tool identity, inventory, and classification.
3. Define deterministic tool schema fingerprinting and drift behavior.
4. Define typed per-tool argument policy-input declarations.
5. Connect these boundaries to frozen G1 `decision.Request` without changing its semantics.
6. Produce independent negative/security evidence.
7. Leave clean interfaces for later persistence and real gateway integration.

## Non-goals
- No downstream token exchange/passthrough implementation.
- No full PostgreSQL persistence.
- No custom MCP proxy.
- No gateway-side JSON adapter for O-008.
- No resource-level authorization.
- No ML risk scoring.
- No mandatory UI framework selection.
- No silent resolution of open decisions.

## Shared Definition of Done
G2 is PASS only when the workstreams agree on:
- identity source and claim-mapping semantics;
- fail-closed identity behavior;
- authoritative tool identity/inventory semantics;
- schema fingerprint canonicalization and drift behavior;
- risk classification semantics;
- typed argument declaration semantics;
- compatibility with frozen G1 `decision.Request`;
- interfaces suitable for later persistence and gateway integration;
- security/negative-test evidence;
- explicit documentation of unresolved decisions.

No workstream may independently redefine a shared contract.

## Workstreams
1. Go Backend — identity, tool registry, argument declarations, domain interfaces.
2. Gateway/MCP — gateway context assumptions, MCP extraction evidence, G6 handoff.
3. Frontend/UI — governance models and operator-facing fixtures.
4. QA/Security — independent abuse and contract verification.
5. DevOps — configuration, secrets, reproducibility, CI.

## Execution discipline
Execute tickets sequentially: **Inspect → Implement → Test → Diff → Record → Proceed.**
Do not guess at open decisions. If blocked by an unresolved architectural question, record the exact question and impact and stop only that boundary.

At the end, produce the normal report + digest under `docs/PHASES/G2_WORKSTREAMS/results/`. After all workstreams are approved and merged, create `CLOSURE_SUMMARY.md` per `/WORKFLOW.md`.
