# G6 Workstream 2 — Go Backend

## Objective

Expose the real AgentGate authorization boundary required by agentgateway and translate its actual authorization request into the frozen `decision.Request` contract without weakening trust semantics.

## Scope

1. Inspect the current `internal/decision`, identity, tool governance, policy, audit, and existing service packages.
2. Define the smallest adapter/interface required to receive the real gateway authorization request.
3. Implement translation from the actual gateway protocol into the frozen decision request.
4. Reuse the existing decision engine; do not duplicate Cedar evaluation.
5. Reuse the existing G5 audited decision path; do not create a second audit path.
6. Validate all security-critical fields before invoking the decision engine.
7. Return an authorization response that maps deterministically to the actual gateway contract.
8. Add unit/contract tests for malformed, missing, ambiguous, and contradictory inputs.
9. Add a real integration endpoint/server if the repository does not already provide one.

## Trust requirements

The adapter must distinguish:

- gateway-authenticated identity from client-provided data;
- gateway-routed backend/tool identity from MCP payload claims;
- typed tool arguments from arbitrary JSON;
- trusted classification/fingerprint from client-supplied metadata.

Never allow a client-controlled value to become an authority merely because it has the expected field name.

## Translation requirements

The adapter must produce the frozen:

```text
decision.Request{
    Principal/identity context,
    Tool context,
    Arguments,
    Action: InvokeTool,
    Workspace,
    Execution/correlation context as available,
}
```

Use the existing G2 tool governance and argument declaration rules. Unknown tools, undeclared arguments, missing required arguments, null where prohibited, fingerprint drift, and ambiguous identity must fail closed.

## Error behavior

All malformed/unsupported authorization requests must produce a deny/error response appropriate to the gateway protocol. They must never fall through to an implicit allow.

AgentGate availability/processing failures must preserve the existing fail-safe semantics.

## Provenance/audit

Do not regress G5:

- evaluated policy version;
- exact policy-byte SHA-256;
- durable decision record;
- redaction;
- tamper-evident chain.

A successful authorization response must only be returned when the existing G5 ALLOW durability requirement has been satisfied.

## Deliverables

- adapter/service implementation;
- protocol/contract types;
- tests;
- endpoint configuration if required;
- durable documentation of the translation boundary;
- handoff report + digest.

## Explicit non-goals

Do not implement G7 downstream credential exchange. Do not forward the inbound bearer token downstream. Do not redesign the MCP proxy.
