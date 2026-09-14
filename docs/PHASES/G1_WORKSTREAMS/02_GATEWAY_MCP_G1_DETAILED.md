# WS-B — Gateway / MCP — G1 Detailed Execution Plan

**Owner:** Gateway/MCP  
**Collaborators:** Go Backend, QA/Security, DevOps  
**Execution:** Sequential tickets. Use the Go mock until the real AgentGate service is ready.

## AG-GW-G1-01 — Inspect the existing agentgateway/MCP path

**Goal:** Understand the current repository topology before modifying configuration.

**Steps:**
1. Locate agentgateway configuration.
2. Locate MCP client/backend fixtures.
3. Locate JWT validation settings.
4. Locate existing ext_authz configuration or placeholder.
5. Identify any direct governed gateway→backend route.
6. Record exact files/configuration to modify.

**DoD:** The intended G1 path is concrete and reproducible:

```text
MCP Client -> agentgateway -> ext_authz -> mock AgentGate
```

No custom MCP proxy is introduced.

## AG-GW-G1-02 — Connect agentgateway to the mock AgentGate

**Goal:** Make the gateway call the deterministic Go mock.

**Steps:**
1. Configure the mock endpoint through environment/configuration.
2. Configure ext_authz using the transport supported by the repository.
3. Apply authorization before governed continuation.
4. Keep test configuration isolated.
5. Add a startup/reachability check where appropriate.

**DoD:** One representative MCP request causes the expected authorization request to reach the mock.

**Collaboration:** If a required field cannot be represented, return the issue to Go Backend/Architect instead of creating a gateway-only field.

## AG-GW-G1-03 — Map the canonical authorization request

**Goal:** Prove the gateway produces the exact G1 request.

**Steps:**
1. Use a known authenticated test identity.
2. Use a known tool identifier/classification.
3. Include only the agreed arguments/context.
4. Include correlation/workspace fields only as defined.
5. Capture a sanitized request fixture.
6. Compare it field-by-field with the Go contract.

**DoD:** No undocumented required field exists; missing identity or unknown tool cannot be transformed into an implicit ALLOW.

## AG-GW-G1-04 — Enforce ALLOW/DENY/error responses

**Goal:** Prove the gateway uses the authorization result as an enforcement decision.

**Steps:**
1. Configure mock ALLOW and send the request.
2. Configure mock DENY and send the same request.
3. Configure mock failure and send the request.
4. Observe gateway behavior for each case.

**DoD:**

```text
ALLOW -> continuation permitted
DENY  -> governed continuation blocked
ERROR -> governed continuation not implicitly allowed
```

## AG-GW-G1-05 — Create reusable fixtures and negative tests

**Required fixtures:** ALLOW, DENY, missing identity, unknown/unclassified tool, malformed input, authorization failure.

**Rules:** deterministic; sanitized; no real tokens/secrets; compatible with QA harness.

**DoD:** Fixtures are consumed by automated gateway tests and match the frozen contract.

## AG-GW-G1-06 — G1 interoperability proof

Run all mandatory G1 scenarios through the gateway. Record outgoing request shape, mock result and gateway behavior.

**DoD:** All scenarios pass; configuration is reproducible; no governed bypass exists; QA can independently rerun the cases; Architect receives the evidence.
