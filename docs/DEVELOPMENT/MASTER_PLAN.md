# AgentGate — Master Development Plan

**Status:** Initial planning baseline
**Date:** 2026-08-22
**Target deployment/development completion:** 2026-09-21
**Working demonstration:** 2026-09-22

> **Execution-sequence note (2026-09-09, DAY-01/TASK-01):**
> `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md` is now the current, detailed execution
> sequence for Phase 1 onward — it targets production readiness, not just the minimal e2e slice
> below. The phase breakdown in §4 remains valid strategic context and is not deleted or
> superseded in substance; day-by-day task assignment now follows the 15-day plan rather than
> this document's phase list. See `docs/SECURITY/PRODUCTION-INVARIANTS.md` for the accompanying
> security/architecture freeze.

## 1. Objective

Build the first working AgentGate product increment from the current repository baseline, using AI-driven implementation while preserving architectural control, testability, security, and maintainability.

The plan is intentionally iterative. Detailed tasks are created only for the next implementation horizon; later phases remain higher-level until their dependencies and discoveries are known.

## 2. Current architectural baseline

The current approved direction is:

**AI agent → agentgateway → AgentGate → authorization decision → downstream MCP server**

AgentGate owns identity interpretation, authorization policy evaluation, audit/governance, and operator-facing governance.

agentgateway remains the MCP data-plane/proxy layer.

Cedar remains the policy engine.

PostgreSQL is intended to become the versioned policy/audit store for the production design.

Resource-level authorization remains the responsibility of the downstream system that owns the resource.

The exact downstream credential/identity propagation mechanism remains an open security decision and must not be guessed by implementation agents.

## 3. Development strategy

Start with a vertical slice that proves the core enforcement path before implementing the full operator product.

The first system proof should establish:

1. an MCP request reaches agentgateway;
2. agentgateway invokes AgentGate authorization;
3. AgentGate derives the required request identity/context;
4. Cedar evaluates the request;
5. the decision is audited according to the agreed audit invariant;
6. ALLOW permits backend execution;
7. DENY prevents backend execution.

## 4. Planned phases

### Phase 0 — Repository and development foundation
**Priority:** P0

Establish repository structure, AI-agent instructions, development workflow, architecture/decision documentation, CI baseline, and project status tracking.

### Phase 1 — Minimal end-to-end enforcement slice
**Priority:** P0

Build the smallest working agentgateway → AgentGate → Cedar → audit → backend path.

Primary proof:

**A denied request never reaches the backend.**

### Phase 2 — Decision-core foundation
**Priority:** P0/P1

Harden the Go service structure, configuration, request context, identity abstraction, Cedar policy loading, policy versioning, audit interface, health behavior, and integration testing.

### Phase 3 — Identity and downstream authorization boundary
**Priority:** P0/P1

Implement the verified inbound identity model and resolve/implement the downstream credential mechanism without token passthrough.

This phase cannot be considered production-complete until the downstream identity trust boundary is explicit and tested.

### Phase 4 — Tool governance and argument authorization
**Priority:** P0/P1

Implement tool inventory/classification, deny-by-default for unknown/drifted tools, tool/schema fingerprinting, and security-relevant argument authorization.

### Phase 5 — Policy governance
**Priority:** P0/P1

Implement versioned policy management, structured rule authoring, dry-run/replay, activation, audit of policy changes, and rollback.

### Phase 6 — Production hardening
**Priority:** P1

Address HA, transport security, audit durability/retention, rate/quota controls, degraded-mode behavior, supply-chain controls, and security testing.

### Phase 7 — Demo/release readiness
**Priority:** P0

Freeze demo scope, run end-to-end regression/security tests, validate deployment, document known limitations, and produce the working demonstration.

## 5. Important sequencing rule

Do not implement every future feature immediately.

The next detailed task must be derived from the current repository state and dependencies.

Each completed task updates project status and may change the ordering of later work.

## 6. Current major risks

1. Downstream identity/credential propagation.
2. Audit durability semantics.
3. Exact agentgateway security/integration behavior.
4. Tool inventory drift.
5. Argument-level authorization.
6. Availability/failure behavior.
7. Schedule pressure.

These are tracked separately in `docs/DECISIONS/OPEN_DECISIONS.md`.

## 7. Explicitly deferred

The following are not allowed to expand the initial development scope without an explicit decision:

- real-time per-call human approval;
- AgentGate-owned resource ownership as a required layer;
- custom MCP proxy implementation;
- ML-based risk scoring;
- unrelated product features.

## 8. Planning rule

Before starting each phase, create or update its phase document with concrete tasks, owners, dependencies, acceptance criteria, and tests.

Do not treat this master plan as a substitute for task-level specifications.
