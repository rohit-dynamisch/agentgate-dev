# AgentGate — Phase 1: Minimal End-to-End Enforcement Slice

**Date:** 2026-08-22  
**Status:** Planned  
**Priority:** P0

## Objective

Build the smallest real AgentGate system that proves:

> A request denied by AgentGate does not reach the downstream MCP backend.

## Target flow

```text
MCP client / test client
        |
        v
  agentgateway
        |
        | ext_authz
        v
    AgentGate
        |
        +--> request context / identity
        +--> Cedar policy evaluation
        +--> audit
        |
        v
    ALLOW / DENY
        |
        | ALLOW
        v
 downstream MCP test backend
```

agentgateway remains the MCP data-plane/proxy layer. AgentGate is the authorization decision point.

## Phase 1 must prove

1. A request travels through agentgateway to AgentGate.
2. AgentGate constructs authorization context.
3. Cedar evaluates a deny-by-default policy in-process.
4. ALLOW permits backend execution.
5. DENY prevents backend execution.
6. The backend can prove whether it was reached.
7. A decision can produce an audit event.
8. Authorization failure does not accidentally produce ALLOW.
9. Each request has an AgentGate-owned execution/correlation ID.
10. The enforcement property is covered by integration tests.

## Explicitly out of scope

- final downstream credential / identity propagation
- token passthrough
- production ID-JAG integration
- PostgreSQL-backed production policy management
- policy editing UI
- dry-run/replay
- tool inventory governance
- tool/schema fingerprint governance
- argument-sensitive authorization
- SpiceDB
- production HA/deployment
- WORM audit checkpointing
- human-in-the-loop approval
- ML risk scoring

## Phase 1 implementation sequence

### TASK-01-01 — Go module and repository scaffold
**Owner:** Backend/Go owner  
**Priority:** P0

Create one Go module, executable entry point, package boundaries, typed configuration, startup/shutdown, basic health/readiness, logging, and initial tests.

**Exit:** application builds, starts, tests pass, and CI runs the real Go checks.

### TASK-01-02 — AgentGate authorization decision core
**Owner:** Backend/Go owner  
**Priority:** P0

Implement request context, Cedar loading/evaluation, deny-by-default behavior, decision result, and execution ID.

**Exit:** unit tests prove ALLOW/DENY independently of agentgateway.

### TASK-01-03 — agentgateway ↔ AgentGate ext_authz integration
**Owner:** Integration owner  
**Priority:** P0

Configure agentgateway and connect its authorization hook to AgentGate. Verify the request behavior and failure handling relied upon by the design.

**Exit:** a real MCP request reaches AgentGate through agentgateway and receives a decision.

### TASK-01-04 — Test MCP backend and end-to-end enforcement proof
**Owner:** QA + integration owner  
**Priority:** P0

Create a minimal MCP backend/client test environment and prove ALLOW reaches the backend while DENY does not.

**Exit:** automated end-to-end enforcement test passes.

### TASK-01-05 — Phase 1 hardening and review
**Owner:** QA + Lead Architect review  
**Priority:** P0

Run negative/security tests, verify fail-closed behavior, inspect telemetry/audit evidence, remove unnecessary complexity, and document limitations.

**Exit:** all Phase 1 acceptance criteria pass.

## Parallelization

TASK-01-01 is the dependency anchor. After its boundaries are reviewed, core authorization, gateway integration, and test infrastructure can proceed in parallel where file overlap is low.

No two agents may independently create or restructure the Go module.

## Acceptance criteria

Phase 1 is complete only when:

- a real Go module exists;
- CI runs actual Go validation;
- AgentGate starts successfully;
- Cedar evaluation occurs in-process;
- deny-by-default is demonstrated;
- agentgateway invokes AgentGate;
- ALLOW permits backend execution;
- DENY prevents backend execution;
- authorization failure cannot accidentally produce ALLOW;
- execution ID exists through decision/audit context;
- integration tests verify the enforcement boundary;
- the inbound bearer token is not forwarded to the backend;
- unresolved production identity/audit questions remain documented;
- no unnecessary production features entered the scope.

## Exit artifacts

Update `docs/DEVELOPMENT/CURRENT_STATUS.md`, relevant decision documents, and phase/task status with the behavior actually demonstrated.
