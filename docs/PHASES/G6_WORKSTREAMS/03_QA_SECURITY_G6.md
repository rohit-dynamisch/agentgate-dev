# G6 Workstream 3 — QA/Security

## Objective

Independently prove that the real enforcement boundary cannot be bypassed and that the backend call count matches the authorization decision.

## Scope

Build a black-box E2E security suite against:

```text
real MCP client
→ real agentgateway
→ real AgentGate
→ real MCP backend fixture
```

The suite must not call AgentGate directly for the primary G6 proof.

## Required assertions

For every scenario capture:

- client request;
- gateway response/error;
- AgentGate decision/audit evidence;
- backend invocation count;
- backend received tool name;
- backend received arguments where safe;
- correlation/execution identity where available.

## Mandatory scenarios

1. valid identity + allowed known tool → exactly one backend call;
2. valid identity + denied known tool → zero backend calls;
3. unknown tool → zero;
4. missing identity → zero;
5. ambiguous identity → zero;
6. malformed authorization request → zero;
7. AgentGate unavailable → zero;
8. policy evaluation failure → zero;
9. tool fingerprint/schema mismatch → zero;
10. malicious classification/tool metadata supplied by the client → zero unless independently authorized;
11. malformed JSON/body/header disagreement → zero;
12. oversized/truncated authorization input, if applicable → zero or explicit safe rejection.

## Security properties

Prove:

- no fail-open path;
- no direct backend bypass;
- no identity spoofing through client-controlled headers/body;
- no tool spoofing;
- no classification spoofing;
- no malformed request becoming ALLOW;
- no duplicate backend invocation for one governed call;
- no G5 audit regression.

## Evidence

Tests should fail loudly if backend count is wrong. Prefer a backend fixture that increments an atomic counter and records each received request.

Record exact test commands and environment versions.

## Deliverables

- independent E2E test suite;
- negative/security tests;
- fixtures/helpers;
- security evidence/report;
- handoff digest.

Do not declare production readiness. Report PASS/FAIL per scenario and list any residual risks.
