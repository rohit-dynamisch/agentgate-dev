# AgentGate — Open Decisions

**Status:** Active — O-001, O-003–O-008 open; O-002 resolved
**Date:** 2026-08-22

This file contains unresolved questions that may affect architecture or implementation.

An open decision is not an invitation for coding agents to guess.

## O-001 — Downstream identity / credential propagation

**Priority:** Critical

AgentGate must not forward the inbound bearer credential to a downstream MCP resource.

The production mechanism for obtaining a downstream credential that represents the required caller/on-behalf-of identity, audience, lifetime, and permissions is not yet fully designed.

**Current action:** Resolve before production downstream identity implementation.

**Allowed development approach:** Build interfaces and test doubles around the credential boundary without establishing unsafe token-passthrough behavior.

## O-003 — agentgateway conformance/security boundary

**Priority:** High

The project relies on agentgateway for MCP transport, routing, JWT validation, and ext_authz integration.

The exact behaviors relied upon must be verified through integration/conformance tests rather than assumed from feature availability.

**Current action:** Build targeted integration tests during the first end-to-end slice.

## O-004 — Supported MCP revision(s)

**Priority:** High

The current Technology Stack Plan targets MCP `2026-07-28`, while compatibility with older deployed clients remains unresolved.

**Current action:** Confirm the supported compatibility boundary before finalizing client/backend integration.

## O-005 — Tool identity and schema fingerprint

**Priority:** High

The project needs a precise canonicalization/fingerprinting rule for backend identity + tool name + normalized input schema.

A fingerprint change should cause governance review and deny-by-default behavior for the affected tool until classified.

**Current action:** Design during tool-governance phase.

## O-006 — Argument authorization model

**Priority:** High

Relevant tool arguments must be exposed to policy evaluation without turning arbitrary tool input into an uncontrolled policy surface.

**Current action:** Define typed per-tool policy-input declarations during tool-governance design.

## O-007 — AgentGate execution identity

**Priority:** Medium/High

Current MCP assumptions about protocol sessions must not be carried into the design if the supported MCP revision is stateless.

AgentGate should own an execution/correlation identifier for audit, future aggregate controls, and request grouping.

**Current action:** Define as part of the request-context model.

## O-008 — ext_authz transport mapping to the decision.Request contract

**Priority:** High

**Found:** G1 checkpoint, Gateway/MCP workstream (`g1/gateway-mcp`), confirmed against the real
`agentgateway` binary once available (`gateway/README.md`, "Real-binary verification").

agentgateway's documented `ext_authz` mechanisms — the HTTP protocol variant
(`path`/`addRequestHeaders`/`includeResponseHeaders`/`redirect`, all CEL expressions operating on
URL/headers only) and the gRPC variant (Envoy's generic `CheckRequest`/`CheckResponse`) — have no
documented way to construct an arbitrary JSON request body. Neither natively produces the
`decision.Request` shape (`execution_id`, `workspace_id`, `identity`, `tool`, `classification`,
`arguments`) this project's frozen G1 contract defines. That JSON shape is an
application/testing contract for the decision core, not necessarily agentgateway's native
`ext_authz` wire protocol.

The production mechanism by which the real `internal/authz` (not yet built, Day 3/8) receives an
`ext_authz` callout and translates it into a `decision.Request` is undesigned. The most plausible
approach — `internal/authz` parsing the raw included request body itself (via `internal/mcpreq`,
per the repo-layout notes in `docs/PROJECT_DEFINITION.md §11`) rather than agentgateway's config
DSL assembling AgentGate's bespoke JSON — is a hypothesis, not a decision.

**Current action:** Design `internal/authz`'s ext_authz-to-`decision.Request` translation
mechanism during the Day 3/8 gateway-integration task. Do not have Gateway/MCP or any other
workstream invent a gateway-side adapter/shim to bridge this in the meantime — that would create a
second, competing contract-mapping outside the component that should own it.

**Allowed development approach:** Keep using the direct-HTTP mock/harness pattern already
established (`agentgate/cmd/g1-mock-authz`, `gateway/harness`) for contract-level testing until
`internal/authz` exists to close this gap for real.

## Resolved

### O-002 — Audit durability invariant (resolved 2026-09-09, DAY-01/TASK-01)

**Priority was:** Critical

**Original question:** whether an ALLOW may be returned when the corresponding audit event has not
yet been durably persisted, given the technology plan's asynchronous audit-write concept versus an
architecture review recommending against committing ALLOW without durable audit persistence.

**Resolution:** no — an ALLOW must not be returned unless a durable audit outcome for that exact
decision is guaranteed (a synchronous PostgreSQL write, or a durable local buffer/journal that
survives a process crash, with PostgreSQL as the eventual sink). Audit-durability failure fails the
request closed. Recorded as a binding invariant in
`docs/SECURITY/PRODUCTION-INVARIANTS.md §5` and §12.2 (Blocking Decision #2).

**What remains (not reopened as an architectural question):** the specific durable-buffer
technology/format is an implementation decision for Day 7 of
`docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md`, not an open invariant question — only the
*mechanism*, not the *guarantee*, remains to be built.

## Decision protocol

For each decision:

1. state the question;
2. identify security/product/implementation impact;
3. verify external facts where necessary;
4. choose a solution;
5. record the rationale;
6. update affected architecture/plan documents;
7. only then remove the item from this file.

Open decisions must not be silently resolved in code.
