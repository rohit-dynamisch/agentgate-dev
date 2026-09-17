# G6 Workstream 1 — Gateway/MCP

## Objective

Build and prove the real `MCP Client → agentgateway → AgentGate → MCP Backend` path.

## Scope

1. Inventory the repository's current agentgateway artifact/configuration.
2. Confirm the exact version/commit/image actually used by the test environment.
3. Inspect the real ext-authz interface supported by that version.
4. Determine the exact request-body and metadata behavior for MCP requests.
5. Configure JWT authentication and the governed MCP route without weakening the existing identity boundary.
6. Connect the real ext-authz path to the AgentGate authorization endpoint.
7. Ensure the gateway's failure mode is fail-closed.
8. Build a deterministic real MCP backend fixture with an observable call counter.
9. Build/use a real MCP client fixture capable of issuing the target MCP tool call.
10. Add reproducible positive/negative gateway integration tests.

## Critical constraint

Do not invent a bespoke JSON body, custom security header, or trust convention until the actual pinned agentgateway contract proves it is available and trustworthy.

If the gateway supports a native MCP-aware external authorization interface, evaluate it first. If it exposes only Envoy ext_authz semantics, implement against the actual supported Envoy contract and document how MCP request data is carried.

Do not replace AgentGate with agentgateway's native `mcpAuthorization`; that can be used only for gateway-local transport/protocol concerns if needed, not as a substitute for the AgentGate governance decision.

## Required gateway evidence

Commit durable documentation/configuration showing:

- gateway version;
- route/backend topology;
- JWT configuration;
- ext-authz configuration;
- request-body configuration and maximum size;
- timeout/failure mode;
- TLS/mTLS settings applicable to the authorization channel;
- exact MCP protocol revision under test;
- how the tool name and arguments are exposed to AgentGate;
- how trusted identity reaches AgentGate;
- how the gateway behaves when AgentGate is unavailable.

## Test matrix

At minimum implement:

- allow → backend count 1;
- deny → backend count 0;
- unknown tool → backend count 0;
- invalid identity → backend count 0;
- malformed authorization input → backend count 0;
- AgentGate unavailable → backend count 0;
- policy evaluation failure → backend count 0;
- tool identity/body mismatch → backend count 0.

Where gateway buffering limits apply, include a test proving oversized/truncated authorization input cannot accidentally become an ALLOW.

## Deliverables

- updated gateway configuration;
- real MCP client fixture/harness;
- observable MCP backend fixture;
- integration tests;
- contract/config documentation;
- exact run instructions;
- handoff report + digest.

## Verification

Run the narrow gateway/MCP suite first, then the full relevant regression suite. Report exact commands and results.
