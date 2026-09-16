# G6 Workstream 4 — DevOps

## Objective

Provide a reproducible production-shaped integration environment in which the G6 enforcement proof can run without hidden developer-machine state.

## Scope

1. Pin the agentgateway artifact/version used by G6.
2. Provide reproducible configuration for:
   - agentgateway;
   - AgentGate;
   - PostgreSQL;
   - MCP backend fixture;
   - MCP client/test harness;
   - identity/JWT fixture where required.
3. Ensure secrets are injected via environment/secret mechanisms, never committed.
4. Configure health/readiness dependencies so the E2E harness can distinguish startup from authorization failure.
5. Make AgentGate-unavailable testing reproducible.
6. Make policy-evaluation-failure testing reproducible.
7. Provide one documented clean-environment command sequence for the G6 harness.
8. Preserve the existing G1–G5 CI baseline and do not weaken existing checks.

## Failure-mode requirements

The environment must make it possible to demonstrate:

```text
AgentGate running    → normal authorization
AgentGate unavailable → gateway denies/fails closed
AgentGate malformed/error response → gateway denies/fails closed
```

Do not configure ext-authz failure mode to ALLOW for the governed route.

## Deliverables

- pinned gateway artifact/config;
- integration compose/Helm/test environment changes as appropriate;
- health/readiness configuration;
- deterministic test startup/shutdown;
- runbook section;
- CI integration where practical;
- handoff report + digest.

## Verification

Demonstrate the clean-environment G6 test sequence and report exact commands/results. Avoid claiming production readiness solely from local Compose success.
