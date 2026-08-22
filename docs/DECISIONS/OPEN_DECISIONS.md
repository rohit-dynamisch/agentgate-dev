# AgentGate — Open Decisions

**Status:** Initial
**Date:** 2026-08-22

This file contains unresolved questions that may affect architecture or implementation.

An open decision is not an invitation for coding agents to guess.

## O-001 — Downstream identity / credential propagation

**Priority:** Critical

AgentGate must not forward the inbound bearer credential to a downstream MCP resource.

The production mechanism for obtaining a downstream credential that represents the required caller/on-behalf-of identity, audience, lifetime, and permissions is not yet fully designed.

**Current action:** Resolve before production downstream identity implementation.

**Allowed development approach:** Build interfaces and test doubles around the credential boundary without establishing unsafe token-passthrough behavior.

## O-002 — Audit durability invariant

**Priority:** Critical

The project needs an explicit guarantee defining whether an ALLOW may be returned when the corresponding audit event has not yet been durably persisted.

The current technology plan contains an asynchronous audit-write concept, while the recent architecture review recommends that an ALLOW not be committed without durable audit persistence.

**Current action:** Decide the invariant before finalizing the production decision/audit path.

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
