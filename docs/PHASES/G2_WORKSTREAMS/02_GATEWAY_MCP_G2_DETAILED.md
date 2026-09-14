# G2 Workstream 02 — Gateway / MCP

## Mission
Establish gateway-side evidence and request-context assumptions required for later real enforcement without prematurely resolving O-008.

## Tickets

### G2-GW-01 — Inspect current state
Review verified G1 agentgateway config/harness, MCP assumptions, O-003/O-004/O-008. Document exactly what ext_authz can provide, what request context/body is available, and what is verified versus hypothetical.

### G2-GW-02 — Claims/context fixture
Create fixtures for gateway-validated identity claims/context entering AgentGate's identity boundary. Include valid, missing, conflicting, and malformed cases. Do not treat fixtures as JWT-validation proof.

### G2-GW-03 — MCP invocation extraction evidence
Using the project's current MCP assumptions, document fields the future `internal/authz` layer must extract to populate G1. Mark revision-sensitive fields; do not resolve O-004 by assumption.

### G2-GW-04 — Tool metadata fixtures
Define fixtures for backend/resource identity, tool name, input schema, and invocation arguments. This is a fixture boundary, not a second governance implementation.

### G2-GW-05 — Negative enforcement preparation
Retain/prove JWT-before-ext_authz behavior and represent missing/invalid governance metadata as fail-closed conditions. Do not claim real AgentGate E2E authorization yet.

### G2-GW-06 — G6 handoff specification
Document the intended evidence/sequence for gateway → ext_authz → `internal/authz` → identity/tool context → `decision.Engine`. Keep O-008 explicitly open.

### G2-GW-07 — Verification
Run gateway harness and real-binary validation where available. If unavailable, report the limitation instead of replacing it with parser-only claims. Produce report + digest.

## DoD
- G1 gateway behavior remains green.
- Verified facts are separated from assumptions.
- O-008 remains open unless explicitly resolved.
- No gateway-side AgentGate JSON adapter.
- No false claim of production E2E enforcement.
