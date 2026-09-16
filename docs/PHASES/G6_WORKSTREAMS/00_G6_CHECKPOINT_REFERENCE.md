# AgentGate G6 — Checkpoint Reference
## Real MCP Enforcement Gate

**Status:** Active checkpoint plan  
**Predecessors:** G1–G5 PASS / CLOSED / FROZEN  
**Pinned Gateway Image:** `cr.agentgateway.dev/agentgateway:v1.4.0@sha256:771afaf093065477fa296eb90dcb618a0300165f12a32f80bbdd1427fab900ec` (see `deploy/g6/IMAGE_DIGESTS.md`)  
**Primary planning source:** `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`  
**Secondary source:** `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md`

## 1. Objective

G6 proves that the real production-shaped request path is enforced by AgentGate:

```text
Real MCP Client
    ↓
agentgateway
    ↓
JWT authentication / trusted request context
    ↓
agentgateway external authorization
    ↓
AgentGate
    ↓
identity + tool governance + Cedar
    ↓
durable audit
    ↓
agentgateway
    ↓
Real MCP Backend
```

The gate is not passed by a mock-only path, a direct AgentGate call, an agentgateway-native policy that duplicates AgentGate, or a test that only proves an HTTP status. The proof must establish whether the real MCP backend was reached.

The shared execution plan defines G6 as the most important remaining integration gate and requires real proof for allowed, denied, unknown-tool, invalid-identity, malformed-ext_authz, AgentGate-unavailable, and policy-evaluation-failure cases.

## 2. Architectural boundary

The responsibilities remain frozen:

- **agentgateway:** MCP protocol/transport, routing, JWT validation, MCP request handling, external authorization integration.
- **AgentGate:** trusted identity/context interpretation at its boundary, tool governance, Cedar authorization, decision provenance, durable audit.
- **MCP backend:** downstream resource authorization and business semantics.

AgentGate must not become an MCP proxy. agentgateway must not become the authoritative AgentGate policy engine.

G6 specifically closes the previously open integration contract around how an actual agentgateway authorization request becomes the frozen AgentGate decision request.

## 3. Critical architectural rule: do not invent the adapter

Before implementing the adapter, the Gateway/MCP and Go Backend streams must inspect the **actual pinned agentgateway version and configuration/schema in this repository**.

The implementation must establish, with repository evidence, all of the following:

1. exact agentgateway version/commit or image;
2. exact ext-authz protocol used in the deployed path;
3. whether request body is included and under what limits/semantics;
4. which MCP metadata is available to the authorization service;
5. how validated JWT identity/claims reach the authorization service;
6. whether the request body is complete, partial, truncated, or otherwise transformed;
7. exact success/deny/error behavior of the gateway on authorization-service failures;
8. exact MCP route/backend behavior after authorization.

If the pinned agentgateway contract cannot safely express the frozen AgentGate request without a security-sensitive guess, stop that part and escalate the concrete contract gap. Do not silently invent headers, JSON fields, trust rules, or a second authorization protocol.

## 4. G6 shared contract

The integration must produce a deterministic translation:

```text
agentgateway authorization request
    ↓
AgentGate adapter
    ↓
decision.Request
```

The adapter must populate only information that is authenticated/trusted according to the frozen G1/G2 boundary.

At minimum the integration must establish:

- workspace identity;
- authenticated principal identity;
- roles;
- agent identity;
- on-behalf-of identity where present;
- authoritative backend/tool identity;
- tool risk/classification/fingerprint;
- typed tool arguments;
- fixed action `InvokeTool`;
- execution/correlation identity where available;
- request/provenance context required by the existing decision/audit contract.

The adapter must reject missing, ambiguous, malformed, contradictory, or untrusted security-critical fields.

It must never derive authorization authority from:

- client-controlled classification;
- unverified tool annotations;
- arbitrary client headers;
- MCP body fields that contradict trusted routing metadata;
- an inbound bearer token merely because it is present.

## 5. MCP request consistency

The integration must verify that the MCP tool identity used by AgentGate corresponds to the actual routed MCP request.

At minimum test:

```text
route/backend identity + MCP tool name + tool arguments
```

against the actual request reaching the backend.

Header/body disagreement, malformed JSON, malformed MCP tool-call structure, unknown tool, and tool-schema/fingerprint mismatch must not produce an ALLOW.

Where agentgateway itself already enforces protocol consistency, G6 must still prove that AgentGate receives the same authoritative tool identity expected by the frozen governance model.

## 6. Fail-closed semantics

The following are mandatory:

```text
AgentGate unavailable          → no backend call
AgentGate timeout/error        → no backend call
malformed authz request        → no backend call
invalid/ambiguous identity     → no backend call
unknown tool                   → no backend call
policy evaluation failure      → no backend call
audit failure on ALLOW         → existing G5 fail-safe semantics apply
```

An explicit gateway fail-open configuration is prohibited for the governed path.

The test must prove backend call count, not merely response code.

## 7. Required end-to-end scenarios

| Scenario | Expected result | Backend call count |
|---|---|---:|
| authenticated + known tool + ALLOW | request succeeds | exactly 1 |
| authenticated + known tool + DENY | authorization failure | 0 |
| unknown tool | DENY | 0 |
| invalid/missing identity | DENY | 0 |
| malformed ext_authz request | DENY/error | 0 |
| AgentGate unavailable | fail closed | 0 |
| policy evaluation failure | DENY/error | 0 |
| tool/schema mismatch | DENY | 0 |
| replay/duplicate request fixture if relevant to current gateway behavior | defined safe result | must not bypass authorization |

The final row is not a new v1 feature. It is a regression/security check only if the current gateway path makes it relevant.

## 8. Required durable evidence

The checkpoint must leave committed, reviewable evidence for:

- exact agentgateway version/configuration;
- ext-authz contract;
- AgentGate adapter contract;
- MCP test backend;
- real MCP client/test harness;
- positive and negative E2E tests;
- fail-closed configuration;
- relevant timeout/error settings;
- test instructions;
- known limitations and unresolved issues.

Ephemeral reports/digests remain outside git according to `WORKFLOW.md`.

## 9. Definition of Done

G6 passes only when all are true:

1. A real MCP client sends a real governed tool call through agentgateway.
2. The request reaches AgentGate through the actual configured authorization path.
3. AgentGate evaluates the existing Cedar policy and records the decision using the G5 audit path.
4. An allowed request reaches the real MCP backend exactly once.
5. A denied request reaches the backend zero times.
6. Unknown tools reach the backend zero times.
7. Invalid/ambiguous identity reaches the backend zero times.
8. Malformed authorization input reaches the backend zero times.
9. AgentGate unavailability fails closed.
10. Policy evaluation failure reaches the backend zero times.
11. The integration proves that the authorization identity/tool context is not attacker-controlled.
12. The test environment is reproducible from the repository.
13. Existing G1–G5 regression suites remain green.
14. No critical/high security defect remains open for the G6 path.
15. Any remaining architecture question is explicitly escalated; no open decision is silently resolved.

## 10. Out of scope

Do not use G6 to implement:

- downstream credential exchange/re-assertion (G7);
- resource-level authorization inside MCP backends;
- real-time HITL;
- mandatory SpiceDB;
- ML risk scoring;
- response-side governance;
- session/aggregate limits;
- UI redesign;
- unrelated production-hardening work scheduled for G8/G9.

## 11. Review sequence

The Lead Architect will review G6 in this order:

```text
1. actual pinned gateway contract
2. authorization request translation
3. trust boundary
4. fail-closed configuration
5. real backend reachability proof
6. audit/provenance proof
7. negative tests
8. regression
9. deployment reproducibility
```

A passing HTTP status is insufficient evidence.
