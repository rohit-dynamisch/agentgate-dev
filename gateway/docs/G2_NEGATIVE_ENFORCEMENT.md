# G2 Gateway/MCP — Negative Enforcement Preparation (G2-GW-05)

**Date:** 2026-09-13  
**Status:** Evidence document  
**Claim:** This document does NOT claim real AgentGate E2E authorization exists.

---

## 1. JWT-before-ext_authz behavior (verified in G1)

**What was verified:**

A bare HTTP request to `agentgateway` without a JWT header receives `401 Unauthorized` **before** any ext_authz callout is made. This was confirmed with the real `ghcr.io/agentgateway/agentgateway:latest` binary using `jwtAuth: mode: strict`.

**Implication:** An unauthenticated request never reaches `internal/authz`. The JWT validation gate is enforced by `agentgateway` itself, not by AgentGate's authorization code.

**Limitation:** A real signed JWT was never driven end-to-end in G1 (the dev JWKS fixture has zero keys). The `401` for unauthenticated requests was confirmed; actual JWT claim extraction and forwarding was not exercised against the real binary.

---

## 2. Missing/invalid governance metadata → fail-closed conditions

The following conditions map to fail-closed outcomes in the G2 identity and tool governance boundaries. None of these reach Cedar evaluation.

### Identity failures (internal/identity)

| Condition | Failure Class | decision.Result |
|---|---|---|
| `agent_id` claim absent | `missing_claim` | `DENY / invalid_identity` |
| `agent_id` claim blank | `malformed_claim` | `DENY / invalid_identity` |
| `roles` claim absent | `missing_claim` | `DENY / invalid_identity` |
| `roles` resolves empty | `missing_roles` | `DENY / invalid_identity` |
| `on_behalf_of` == `agent_id` | `ambiguous_identity` | `DENY / invalid_identity` |
| `on_behalf_of` present but blank | `malformed_claim` | `DENY / invalid_identity` |

### Tool governance failures (internal/toolregistry)

| Condition | Governance Result | decision.Result |
|---|---|---|
| Tool not in registry | `unknown_tool` | `DENY / unknown_tool` |
| Tool schema drifted | `drift_detected` | `DENY / malformed_request` |
| Tool `Known=true` but `Risk` empty | invalid governance | `DENY / malformed_request` |
| Same tool name, different backend | `unknown_tool` | `DENY / unknown_tool` |
| Unrecognized risk level | invalid governance | `DENY / malformed_request` |

### Argument failures (internal/argdecl)

| Condition | Resolution Result | decision.Result |
|---|---|---|
| Required argument absent | `ResolutionError` | `DENY / malformed_request` |
| Explicit JSON null for any arg | `ResolutionError` | `DENY / malformed_request` |
| Argument type mismatch | `ResolutionError` | `DENY / malformed_request` |
| Undeclared argument present | Silently ignored | (never reaches policy) |

---

## 3. What a real system would deny

In a future production deployment where the ext_authz path is wired end-to-end (pending O-008 resolution), these conditions would cause a deny before Cedar is ever reached:

1. Request with no JWT → `401` from agentgateway (already enforced).
2. JWT with missing `sub` claim → identity mapping failure → `DENY / invalid_identity`.
3. JWT with empty `roles` claim → identity mapping failure → `DENY / invalid_identity`.
4. Call to an unregistered tool → governance lookup returns `Known=false` → `DENY / unknown_tool`.
5. Call to a tool whose schema has changed since registration → drift detected → `DENY`.
6. Call with a null value for a declared argument → null rejection → `DENY / malformed_request`.
7. No Cedar policy loaded → `DENY / no_policy_loaded`.
8. Cedar evaluation error → `DENY / evaluation_error` (never trusted as ALLOW).

---

## 4. What is NOT claimed

- No real AgentGate E2E authorization is demonstrated in G2. The ext_authz → `internal/authz` → `decision.Request` path is not wired (O-008 open).
- The JWT claim forwarding from agentgateway to AgentGate is not verified against the real binary.
- MCP tool name/backend extraction from the request body is unverified.
