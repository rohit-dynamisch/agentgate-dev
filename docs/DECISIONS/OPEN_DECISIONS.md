# AgentGate — Open Decisions

**Status:** Active — O-001, O-003, O-004, O-008 open; O-002, O-005, O-006, O-007 resolved
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

### O-002 — Audit durability invariant (resolved 2026-09-14, G5 checkpoint)

**Priority was:** Critical

**Original question:** whether an ALLOW may be returned when the corresponding audit event has not yet been durably persisted, given the technology plan's asynchronous audit-write concept versus an architecture review recommending against committing ALLOW without durable audit persistence.

**Resolution:** no — an ALLOW must not be returned unless a durable audit outcome for that exact decision is guaranteed. Audit-durability failure fails the request closed (`DENY`). Recorded as a binding invariant in `docs/SECURITY/PRODUCTION-INVARIANTS.md §5` and §12.2.

**Implementation (Completed G5, 2026-09-14):** Implemented in `agentgate/internal/audit`: append-only `audit_events` PostgreSQL persistence, tamper-evident SHA-256 row chaining (`prev_hash` + `row_hash`), independent out-of-process `ChainVerifier`, pre-persistence argument redaction (`redact.go`), fail-closed decision enforcement (`service.go`), DB immutability triggers (`prevent_audit_modification`), and DB privilege separation (`agentgate_app` vs `agentgate_migrator`). Formally approved by the Lead Architect 2026-09-14.

### O-006 — Argument authorization model (resolved 2026-09-13, G2 checkpoint)

**Priority was:** High

**Original question:** how to expose relevant tool arguments to policy evaluation without turning
arbitrary tool input into an uncontrolled policy surface.

**Resolution:** per-tool typed declaration registry (`internal/argdecl`). Arbitrary argument
passthrough was rejected in favor of an explicit whitelist model:

- Each tool declares its policy-visible arguments via a `DeclarationSet` specifying name, type
  (`string`, `int64`, `bool`), and required/optional status.
- Only declared arguments are extracted and resolved into `decision.Request.Arguments`.
- **Undeclared arguments** are excluded from policy input — they are structurally invisible to Cedar.
- **Explicit JSON `null`** is rejected (not coerced to a zero value), preventing type-confusion bypasses.
- **Missing required arguments** fail closed (request denied).
- **Omitted optional arguments** are absent rather than fabricated with defaults.
- Resolution produces deterministic typed `AttributeValue` entries that feed the frozen G1
  `decision.Request` contract.

**Why arbitrary passthrough was rejected:** passing raw MCP `arguments` wholesale into Cedar
would create an uncontrolled policy surface — any new or renamed tool argument would silently
become a policy input without governance review. The declaration registry ensures that only
explicitly approved attributes influence authorization decisions.

**Implementation location:** `agentgate/internal/argdecl/` (declaration types, resolution logic,
and comprehensive tests). Context assembly adapter at `agentgate/internal/contextassembly/`
populates `decision.Request.Arguments` from resolved declarations.

**Scope note:** closing O-006 resolves the *policy-input exposure model* — which arguments become
policy-visible and how. It does not mean argument-level authorization is a fully deployed business
authorization system; Cedar policy semantics determine the eventual authorization decisions
once the declared arguments reach policy evaluation.

**Affected architecture documents:**
- `docs/SECURITY/PRODUCTION-INVARIANTS.md` §6 (argument-authorization boundary invariant)
- `docs/PHASES/G2_WORKSTREAMS/G2_CLOSURE_SUMMARY.md` §1 (W1), §4 (O-006 status)
- `docs/PHASES/G2_WORKSTREAMS/results/GO_BACKEND_G2_REPORT.md` §Open Decisions
- `agentgate/internal/decision/types.go` (AttributeValue type, historical O-006 comment)
- `agentgate/internal/decision/doc.go` (historical O-006 reference)

### O-005 — Tool identity and schema fingerprint (resolved 2026-09-13, G2 checkpoint)

**Priority was:** High

**Resolution:** Implemented in `agentgate/internal/toolregistry`. Defines deterministic canonicalization and SHA-256 fingerprinting for backend identity + tool name + schema. Any schema drift triggers `CheckDrift()` detection and deny-by-default behavior until classified.

### O-007 — AgentGate execution identity (resolved 2026-09-13, G2 & G5 checkpoints)

**Priority was:** Medium/High

**Resolution:** Defined `execution_id` correlation identifier populated across `decision.Request` (`internal/contextassembly`) and persisted in `audit_events` (`internal/audit`) for end-to-end request tracing and audit correlation.

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
