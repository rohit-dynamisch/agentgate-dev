# DAY-02 / TASK-02 — Decision Core Productionization

**Owner:** Backend Developer  
**Review:** Lead Architect + QA  
**Phase:** AgentGate v1 15-Day Production Readiness  
**Status:** Complete — accepted. Its deliverable (the `internal/decision`/`internal/policy`
decision core) is now the frozen G1 contract, documented in
`docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md`; this task spec is kept as the historical
record of what was assigned. The 15-day sequencing this task belonged to was itself superseded
after this task by `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`.  
**Dependency:** DAY-01/TASK-01 accepted

## Objective

Productionize the AgentGate authorization decision boundary so it is independently testable, typed, deterministic, and fail-closed.

This task establishes the decision-core contract that later gateway integration, identity, tool governance, policy persistence, and audit implementations will consume.

## Governing Documents

Read before implementation:

- `CLAUDE.md`
- `docs/PROJECT_DEFINITION.md`
- `docs/TECH_STACK.md`
- `docs/PHASES/archive/MASTER_PLAN.md`
- `docs/DEVELOPMENT/CURRENT_STATUS.md`
- `docs/DECISIONS/OPEN_DECISIONS.md`
- `docs/SECURITY/PRODUCTION-INVARIANTS.md`
- `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md`

The Day 1 security invariants are binding. Do not reinterpret them.

## Scope

Implement/refine only these decision-core concerns:

1. Typed authorization request.
2. Typed authorization result.
3. Request/execution context.
4. Identity representation sufficient for authorization.
5. Cedar engine/evaluation boundary.
6. Tool classification input.
7. Explicit declared argument attributes.
8. Deny-by-default behavior.
9. Exact policy version/hash propagation.
10. Deterministic error/reason semantics.
11. Unit tests covering the required decision matrix.

The decision core must be independently testable without agentgateway, PostgreSQL, UI, or a real downstream credential provider.

## Required Authorization Context

The design must have an explicit representation for the information needed by policy evaluation. At minimum, determine the typed representation for:

- execution/correlation ID;
- workspace ID;
- agent identity;
- on-behalf-of identity where present;
- roles;
- backend/tool identity;
- tool classification state;
- declared authorization argument attributes;
- policy version/hash.

Do not add fields merely because they might be useful later. Every field added to the authorization contract must have a current v1 purpose.

## Decision Semantics

The result must distinguish at least:

- ALLOW;
- DENY.

It must also provide a deterministic reason/category suitable for audit and caller-facing enforcement.

Failure conditions must never be represented as ALLOW.

At minimum, test:

- valid explicit allow;
- valid explicit deny;
- missing identity;
- ambiguous/unusable identity;
- unknown/unclassified tool;
- no matching policy;
- malformed/incomplete authorization request;
- Cedar evaluation error;
- argument-dependent rule;
- undeclared argument cannot silently influence authorization;
- exact policy version/hash is preserved in the result;
- execution ID is preserved through the decision context.

## Cedar Boundary

Keep Cedar behind a narrow application-owned boundary.

The decision core should not make callers depend directly on Cedar implementation details where a stable AgentGate domain contract is sufficient.

Do not introduce a broad framework or speculative policy abstraction.

Policy bytes may remain a test/dev fixture for this task. PostgreSQL policy persistence is Day 4 work.

## Tool Classification Boundary

The request must carry an explicit tool-classification state/input.

Do not implement the complete fingerprinting/drift system here; that is Day 3.

However, the decision core must already be capable of denying an unknown/unclassified tool.

## Argument Boundary

Do not treat arbitrary request arguments as policy authority.

Only explicitly declared authorization attributes may enter policy evaluation.

Do not implement the complete per-tool declaration/governance system here; establish only the typed boundary needed by Day 2.

## Policy Version / Hash

The decision result must carry the exact policy version/hash that was evaluated.

Do not derive it later from the current active policy after evaluation.

If the evaluator has no valid policy identity, the decision cannot be ALLOW.

## Out of Scope

Do not implement:

- PostgreSQL schema/migrations;
- policy persistence or activation;
- policy governance API/UI;
- dry-run/replay;
- claims-mapping configuration;
- complete tool fingerprinting/drift detection;
- agentgateway configuration;
- MCP protocol handling;
- downstream credential exchange;
- raw token forwarding;
- audit persistence;
- rate limiting;
- deployment/TLS;
- OpenTelemetry;
- new speculative dependencies.

Do not restructure unrelated packages.

## Testing

Tests must be table-driven where appropriate and cover both positive and negative paths.

The test suite must prove that security failures cannot become ALLOW.

At minimum include cases for:

| Case | Expected |
|---|---|
| Explicit policy allow | ALLOW |
| Explicit policy deny | DENY |
| Missing identity | DENY |
| Ambiguous identity | DENY |
| Unknown tool | DENY |
| No matching policy | DENY |
| Malformed request | DENY |
| Cedar evaluation failure | DENY |
| Argument constraint satisfied | ALLOW/DENY according to policy |
| Unauthorized undeclared argument influence | DENY or ignored according to the declared contract; never implicit authority |
| Policy version/hash | exact evaluated value returned |
| Execution ID | exact input value preserved |

Also test that the decision result is deterministic for identical inputs and policy.

## Verification

Before reporting completion:

- run `gofmt`;
- run `go vet ./...`;
- run `go test ./...`;
- run race tests if the environment supports them;
- inspect the complete diff;
- verify no task-out-of-scope files or dependencies were introduced.

Report:
- files changed;
- domain/API decisions made;
- tests run and results;
- deviations from the task;
- unresolved items;
- limitations.

## Exit Criteria

Day 2 is complete only when:

1. The authorization core has a clear typed contract.
2. The core is independently testable.
3. Cedar is behind the AgentGate decision boundary.
4. Missing/invalid identity cannot ALLOW.
5. Unknown/unclassified tools cannot ALLOW.
6. Policy errors cannot ALLOW.
7. Undeclared arbitrary arguments cannot silently become authorization inputs.
8. Exact evaluated policy version/hash is carried by the decision.
9. Execution/correlation identity is preserved.
10. Decision outcomes and failure reasons are deterministic.
11. Required unit tests pass.
12. No Day 3+ implementation has been started.

Stop and wait for Lead Architect review. Do not begin Day 3.
