# G2 Gateway/MCP — G6 Handoff Specification (G2-GW-06)

**Date:** 2026-09-13  
**Purpose:** Document the intended authorization sequence for the G6 gateway-integration checkpoint.  
**O-008 Status:** **EXPLICITLY OPEN.** The mechanism by which an ext_authz callout becomes a `decision.Request` is undesigned. This document describes the *intended sequence*, not a resolved design.

---

## Overview

G6 will wire the real `internal/authz` gRPC/HTTP service to `agentgateway`'s ext_authz callout, closing the O-008 gap. This document specifies what each hop must produce, what the next hop needs, and where the open gap lies.

---

## Intended sequence

```mermaid
sequenceDiagram
    participant MCP_Client as MCP Client
    participant GW as agentgateway<br/>(JWT auth + ext_authz)
    participant Authz as internal/authz<br/>(NOT YET BUILT — G6)
    participant Identity as internal/identity<br/>(G2 ✅)
    participant Registry as internal/toolregistry<br/>(G2 ✅)
    participant ArgDecl as internal/argdecl<br/>(G2 ✅)
    participant Assembly as internal/contextassembly<br/>(G2 ✅)
    participant Engine as internal/decision.Engine<br/>(G1 ✅)
    participant Cedar as internal/policy<br/>(Cedar boundary, G1 ✅)

    MCP_Client->>GW: MCP request (with JWT)
    GW->>GW: JWT validation (strict mode)<br/>→ 401 if invalid/absent

    Note over GW,Authz: ⚠️ O-008 GAP: how ext_authz callout<br/>carries MCP body to internal/authz<br/>is UNDESIGNED. Do not resolve here.

    GW->>Authz: ext_authz callout<br/>(HTTP or gRPC — format TBD, O-008)
    
    Authz->>Authz: Extract validated JWT claims<br/>from callout context
    Authz->>Identity: Mapper.Map(claims)<br/>→ MappedIdentity or MappingError
    
    alt identity mapping fails
        Identity-->>Authz: MappingError (missing/malformed/ambiguous/no-roles)
        Authz-->>GW: DENY (invalid_identity)
        GW-->>MCP_Client: 403 Forbidden
    end

    Authz->>Authz: Extract tool.backend_id, tool.name<br/>from MCP body (O-008 — parsing TBD)
    Authz->>Registry: Registry.Lookup(toolID, liveFingerprint)<br/>→ GovernanceRecord
    
    alt tool unknown or drifted
        Registry-->>Authz: GovernanceRecord{Known:false} or DriftDetected
        Authz-->>GW: DENY (unknown_tool / drift)
        GW-->>MCP_Client: 403 Forbidden
    end

    Authz->>ArgDecl: DeclarationSet.Resolve(rawArgs)<br/>→ map[string]ResolvedArg or ResolutionError
    
    alt argument resolution fails
        ArgDecl-->>Authz: ResolutionError (null/type-mismatch/missing-required)
        Authz-->>GW: DENY (malformed_request)
        GW-->>MCP_Client: 403 Forbidden
    end

    Authz->>Assembly: Assemble(AssemblyInput)<br/>→ decision.Request
    Assembly->>Engine: Engine.Evaluate(request)<br/>→ decision.Result

    Engine->>Cedar: Cedar policy evaluation
    Cedar-->>Engine: Allow / Deny / Error
    Engine-->>Assembly: Result{Decision, Reason, PolicyVersion}
    Assembly-->>Authz: decision.Result

    alt decision == ALLOW
        Authz-->>GW: ALLOW (with audit record)
        GW->>MCP_Client: Forward to MCP backend
    else decision == DENY
        Authz-->>GW: DENY
        GW-->>MCP_Client: 403 Forbidden
    end
```

---

## Per-hop contract

### Hop 1: MCP Client → agentgateway
- **Input**: MCP request with JWT in `Authorization: Bearer` header
- **agentgateway's job**: JWT signature validation, audience/issuer check
- **Output to next hop**: validated claims (mechanism TBD — O-003, O-008), raw MCP body

### Hop 2: agentgateway → internal/authz (ext_authz callout)
- **⚠️ O-008 GAP**: format of callout undesigned
- **What internal/authz needs** from this hop:
  - Pre-validated JWT claims (as a `map[string]string` for `identity.Mapper`)
  - Raw MCP request body (to extract `tool.backend_id`, `tool.name`, raw arguments)
  - A correlation/execution ID (for `decision.Request.ExecutionID`)
  - Workspace scoping context (for `decision.Request.WorkspaceID`)
- **Most plausible approach** (hypothesis): `internal/authz` reads the raw body from the callout and parses it into a `decision.Request` via `internal/mcpreq` (per `docs/PROJECT_DEFINITION.md §11`). NOT a decision.

### Hop 3: internal/authz → internal/identity
- **Input**: `map[string]string` of gateway-validated claims
- **Contract**: `identity.Mapper.Map(claims)` → `MappedIdentity` or `*MappingError`
- **G2 status**: ✅ Implemented and tested

### Hop 4: internal/authz → internal/toolregistry
- **Input**: `toolregistry.ToolID{BackendID, ToolName}` + optional live schema fingerprint
- **Contract**: `Registry.Lookup(id, liveFingerprint)` → `GovernanceRecord`
- **G2 status**: ✅ Implemented and tested

### Hop 5: internal/authz → internal/argdecl
- **Input**: `map[string]json.RawMessage` of raw tool arguments + `DeclarationSet` for this tool
- **Contract**: `DeclarationSet.Resolve(raw)` → `map[string]ResolvedArg` or `*ResolutionError`
- **G2 status**: ✅ Implemented and tested

### Hop 6: internal/authz → internal/contextassembly → decision.Engine
- **Input**: `MappedIdentity` + `GovernanceRecord` + `map[string]ResolvedArg`
- **Contract**: `contextassembly.Assemble(AssemblyInput)` → `decision.Request`; then `Engine.Evaluate(request)` → `decision.Result`
- **G2 status**: ✅ Implemented and tested (end-to-end in `TestAssemble_FeedsDecisionEngine`)

---

## Open items for G6

1. **O-008**: Design `internal/authz`'s ext_authz transport handling — HTTP or gRPC, body inclusion mechanism.
2. **O-003**: Verify agentgateway's actual claim forwarding behavior against real binary with signed JWT.
3. **O-004**: Confirm MCP protocol revision for body parsing.
4. **Audit durability**: `DENY / ALLOW` must both produce a durable audit record before the response is returned (O-002, resolved invariant).
5. **Execution ID generation**: Who generates `ExecutionID`? Likely `internal/authz` on callout receipt.
