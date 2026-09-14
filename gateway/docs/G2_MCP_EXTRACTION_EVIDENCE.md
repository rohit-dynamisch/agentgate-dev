# G2 Gateway/MCP — MCP Invocation Extraction Evidence (G2-GW-03)

**Date:** 2026-09-13  
**Status:** Evidence document — distinguishes verified facts from hypotheses  
**O-008:** Explicitly open. This document does NOT resolve it.

---

## Purpose

This document maps the fields that a future `internal/authz` layer must extract from a real `agentgateway` ext_authz callout to populate a `decision.Request`. Each field is marked by confidence level.

---

## Fields required by `decision.Request` (frozen G1 contract)

| Field | Go type | Source in ext_authz callout | Confidence | Notes |
|---|---|---|---|---|
| `execution_id` | `string` | **Unknown / O-008 gap** | ❌ Hypothetical | Must be generated or extracted from MCP session context. No documented agentgateway mechanism. |
| `workspace_id` | `string` | **Unknown / O-008 gap** | ❌ Hypothetical | Deployment-level config or extracted from MCP body. Not provided natively by ext_authz. |
| `identity.agent_id` | `string` | JWT claim (post-validation) forwarded as header | ⚠️ Partially verified | JWT validation confirmed (401 on bare request). Claim→header forwarding: hypothetical, not verified. |
| `identity.on_behalf_of` | `string` | JWT claim forwarded as header | ⚠️ Hypothetical | Same caveat as agent_id. Optional field. |
| `identity.roles` | `[]string` | JWT claim forwarded as header | ⚠️ Hypothetical | Space-separated string claim would need to be split. |
| `tool.backend_id` | `string` | MCP request body | ❌ O-008 gap | agentgateway does not natively parse MCP JSON body into ext_authz fields. |
| `tool.name` | `string` | MCP request body | ❌ O-008 gap | Same — must be parsed by `internal/authz` from raw MCP body. |
| `classification.known` | `bool` | AgentGate-internal (toolregistry lookup) | ✅ Internal | Populated by `internal/toolregistry.Registry.Lookup()` — not from gateway. |
| `classification.risk` | `string` | AgentGate-internal (toolregistry lookup) | ✅ Internal | Same. |
| `arguments` | `map[string]AttributeValue` | MCP request body | ❌ O-008 gap | Filtered through `internal/argdecl.DeclarationSet.Resolve()`. Source is MCP body. |

---

## Confidence levels

| Symbol | Meaning |
|---|---|
| ✅ Verified | Confirmed by G1 real-binary test or by design (internal computation) |
| ⚠️ Partially verified | Plausible based on agentgateway docs, not yet confirmed with a live integration test |
| ❌ Hypothetical / O-008 gap | Not documented or confirmed; depends on O-008 resolution |

---

## The O-008 gap in detail

`agentgateway`'s documented `ext_authz` HTTP protocol fields:

```yaml
extAuthz:
  protocol:
    http:
      path: '<CEL expression>'           # sets the callout URL path
      addRequestHeaders: {...}            # adds headers (CEL values)
      includeResponseHeaders: {...}       # copies response headers back
      redirect: '<CEL expression>'        # optional redirect
```

None of these fields produce an arbitrary JSON request body. They operate on URL and headers only.

The frozen G1 `decision.Request` shape:
```json
{
  "execution_id": "...",
  "workspace_id": "...",
  "identity": {"agent_id": "...", "on_behalf_of": "...", "roles": [...]},
  "tool": {"backend_id": "...", "name": "..."},
  "classification": {"known": true, "risk": "read"},
  "arguments": {}
}
```

**This shape cannot be assembled by agentgateway's config DSL alone.**

The most plausible path (hypothesis, not a decision): `internal/authz` receives the ext_authz callout (which includes the raw MCP request body via `includeBody` or gRPC equivalent) and parses it directly into a `decision.Request`. This hypothesis is documented in `OPEN_DECISIONS.md O-008` and must NOT be treated as decided.

---

## Revision-sensitive fields

The following fields are marked revision-sensitive because their availability depends on the specific MCP protocol revision (O-004) and agentgateway version:

- **MCP body structure**: MCP `2026-07-28` request format must be confirmed before finalizing extraction logic.
- **JWT claim forwarding mechanism**: depends on agentgateway's specific claim-forwarding config, which must be verified (O-003).
- **ext_authz body inclusion**: whether agentgateway's HTTP ext_authz mode can include the raw MCP body in the callout is unconfirmed.

---

## What G2 provides as evidence

- `internal/identity.Mapper` can process a pre-validated `map[string]string` of claims (see `gateway/fixtures/g2/identity_claims.json` for the expected shape).
- `internal/toolregistry.Registry` produces `GovernanceRecord` from a `ToolID` lookup (see `gateway/fixtures/g2/tool_metadata.json`).
- `internal/contextassembly.Assemble()` converts these into a `decision.Request` — the missing link is: who calls it, and with what inputs extracted from the ext_authz callout. That is G6/O-008.
