# AgentGate v1 — 15-Day Production-Readiness Implementation Plan

**Planning date:** 2026-09-03  
**Target:** AgentGate v1 production-ready release candidate  
**Objective:** Build and prove the smallest production-capable AgentGate release, not a demo.

> **Superseded as the execution sequence (2026-09-12):**
> `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md` is the current execution
> sequence — it compresses this plan's remaining days into parallel workstreams with
> checkpoint/gate acceptance (G1, G2, ... GN) once Day 1 and Day 2 below were complete. This
> document's Day 1/Day 2 sections describe work that was actually done (see
> `docs/PHASES/DAY-01-TASK-01.md`, `docs/PHASES/DAY-02-TASK-02.md`); Day 3 onward here was never
> executed under this sequencing and is retained for historical context only. Current status and
> the active checkpoint are always in `docs/DEVELOPMENT/CURRENT_STATUS.md`; see `docs/README.md`
> for the full documentation map.

## 1. V1 Outcome

At Day 15, the production path must be:

```text
AI Agent / MCP Client
        |
        v
agentgateway
  - MCP transport/routing
  - JWT validation
  - tool discovery
  - ext_authz
        |
        v
AgentGate
  - trusted request identity
  - tool/request context
  - Cedar authorization
  - fail-closed enforcement
  - durable audit
  - policy versioning
        |
        v
Real MCP Backend
```

Production acceptance means every governed call receives a deterministic allow/deny decision from a known identity, known tool classification and active policy; denied calls cannot reach the backend; every decision is durably auditable with the exact policy version; unsafe/unknown conditions fail closed; authorized operators can safely validate, activate and roll back policy; and the system is observable, deployable and recoverable.

## 2. V1 Scope

### Required

- MCP `2026-07-28` integration through agentgateway.
- JWT validation at agentgateway.
- AgentGate ext_authz gRPC decision service.
- Identity extraction and configurable claims mapping.
- Cedar authorization, in-process and deny-by-default.
- Explicit tool classification; unknown/new tools deny by default.
- Relevant tool arguments available to policy.
- Versioned PostgreSQL policy storage and in-memory active policy.
- Safe policy validation, activation and rollback.
- Dry-run against historical decisions.
- Policy-change audit trail.
- Durable, tamper-evident decision audit with argument redaction.
- `workspace_id` in persistent data.
- Authenticated/authorized admin API/UI.
- Rate limiting and abuse controls appropriate to v1.
- TLS/mTLS for security-sensitive service communication.
- Health/readiness and graceful shutdown.
- Structured logging and OpenTelemetry.
- Production container/deployment configuration.
- Database migrations and backup/recovery procedure.
- Unit, integration, security, failure-mode, concurrency and regression testing.
- CI with tests, race detection, linting and vulnerability scanning.
- SBOM and signed release artifacts/images where supported.
- Production configuration, operator documentation and runbook.

### Explicitly deferred

Real-time per-call HITL; Slack/Teams approval; mandatory SpiceDB/resource ownership; ML risk scoring; building an MCP proxy; full multi-tenant runtime; advanced compliance exports; response-side governance; advanced aggregate/session authorization unless required by a concrete v1 security invariant.

These are deliberate scope boundaries, not omissions.

## 3. Team Ownership

**Lead Architect / Team Lead:** architecture, security decisions, scope, acceptance criteria, task approval and final review. Does not delegate architectural decisions to coding agents.

**Backend Developer:** AgentGate Go implementation, Cedar, policy store, authz, identity, audit, configuration, migrations and tests.

**AI/ML Engineer:** agentgateway configuration, MCP client/backend test environment, E2E integration, identity-propagation experiments and performance/load harness.

**QA Engineer:** independent acceptance/security testing, failure injection, authorization-bypass testing, regression, concurrency/load and release verification.

**React Native Developer:** admin UI support when assigned; otherwise focus on UX verification or higher-value integration/testing work. No UI polish ahead of security/reliability work.

## 4. AI Execution Rule

Every task follows:

```text
Architect defines task
→ coding agent reads governing docs/repo
→ implement ONLY assigned scope
→ run required tests
→ inspect diff
→ report changes/tests/limitations
→ architect reviews
→ next task starts only after acceptance
```

Agents must not silently change architecture, start later tasks, add speculative dependencies, weaken fail-closed behavior, or expand scope.

# 5. Fifteen-Day Schedule

## DAY 1 — Production Architecture Freeze

**Owner:** Lead Architect + QA

Define production security invariants:
- gateway is the intended entry path;
- AgentGate trusts only the authenticated gateway path;
- missing/invalid identity => deny;
- unknown/unclassified tool => deny;
- no matching policy => deny;
- policy evaluation error => deny;
- audit durability semantics are explicit;
- policy activation is atomic;
- every decision identifies its exact policy version;
- denied calls cannot reach the backend.

Resolve/bound blocking decisions:
- downstream identity mechanism;
- audit durability semantics;
- policy reload semantics;
- admin authentication;
- TLS/mTLS boundaries;
- deployment topology;
- backup/recovery expectations.

**Deliverable:** `docs/SECURITY/PRODUCTION-INVARIANTS.md`

**Exit:** invariants and blocking decisions accepted.

## DAY 2 — Decision Core Productionization

**Owner:** Backend Developer

Implement/refine:
- typed authorization request/result;
- identity/context model;
- Cedar engine boundary;
- tool classification input;
- argument attributes;
- deny-by-default behavior;
- policy version/hash propagation;
- deterministic error handling.

Test explicit allow/deny, missing identity, unknown tool, malformed request, policy errors, argument-dependent rules and policy-version propagation.

**Exit:** authorization core is independently testable and fail-closed.

## DAY 3 — Identity + Tool Governance Boundary

**Owner:** Backend Developer + AI/ML Engineer; QA verifies

Implement:
- typed claims-mapping configuration;
- required identity-field validation;
- agent and on-behalf-of identity;
- roles;
- tool fingerprinting;
- explicit classification;
- classification drift detection;
- unknown tool => deny.

Verify gateway JWT configuration and trusted claims boundary.

Test missing/malformed claims, changed schema, newly introduced tools and spoofed classification.

**Exit:** identity ambiguity and unknown tools can never become an accidental allow.

## DAY 4 — Policy Persistence and Lifecycle

**Owner:** Backend Developer

Implement:
- PostgreSQL policy schema;
- `workspace_id`;
- content-hash versioning;
- policy states;
- retrieval;
- active in-memory policy;
- atomic activation;
- rollback;
- migrations.

Test create/validate/activate/restart/reload/rollback, malformed policy and concurrent reads during activation.

**Exit:** persistent, versioned and rollbackable policy lifecycle.

## DAY 5 — Policy Governance API/UI

**Owner:** Backend Developer + UI support

Implement minimum production workflow:
1. list policies;
2. inspect versions;
3. create candidate;
4. validate;
5. preview impact;
6. activate;
7. rollback;
8. audit the mutation.

Prefer structured policy input with generated Cedar shown read-only.

All admin operations require authentication and authorization.

**Exit:** authorized operator can manage policy without editing container files.

## DAY 6 — Dry-Run and Change Safety

**Owner:** Backend Developer + QA

Implement candidate-policy replay against recent historical decisions.

Show:
- decisions whose outcome changes;
- newly allowed calls;
- newly denied calls;
- impact summary.

Dry-run must never modify active policy. Audit dry-run and activation separately.

**Exit:** policy blast radius can be inspected before activation.

## DAY 7 — Audit Durability and Integrity

**Owner:** Backend Developer; QA independently verifies

Implement:
- durable decision records;
- identity, tool, decision, reason and timestamp;
- policy version/hash;
- correlation/execution ID;
- workspace ID;
- argument redaction;
- hash chaining;
- restricted database permissions;
- defined audit-write failure behavior.

Test allowed/denied records, chain integrity, tampering detection, redaction and restart behavior.

**Exit:** every decision has the defined durable audit outcome and tampering is detectable.

## DAY 8 — Agentgateway + MCP Production Integration

**Owner:** AI/ML Engineer + Backend Developer; QA verifies

Prove the real path:

```text
MCP Client
→ agentgateway
→ JWT validation
→ ext_authz
→ AgentGate
→ Cedar
→ audit
→ agentgateway
→ MCP backend
```

Verify MCP `2026-07-28`, routing, ext_authz body handling, identity propagation and tool/schema consistency.

Critical E2E proof:
- ALLOW reaches backend;
- DENY does not reach backend.

**Exit:** real MCP traffic is enforced by AgentGate.

## DAY 9 — Downstream Identity / Credential Boundary

**Owner:** Lead Architect + Backend Developer + AI/ML Engineer

Implement the approved v1 downstream identity mechanism.

Requirements:
- never blindly forward inbound bearer tokens;
- correct downstream audience;
- short-lived/scoped credentials where applicable;
- caller/on-behalf-of identity remains auditable;
- replay/confusion risks addressed;
- failures fail closed.

If the mechanism cannot be safely implemented, narrow the deployment scope rather than ship an unsafe design.

**Exit:** downstream identity is security-reviewed and consistent with the selected MCP/identity model.

## DAY 10 — Admin Security + Configuration Hardening

**Owner:** Backend Developer; QA verifies

Implement/verify:
- authenticated admin API/UI;
- admin authorization;
- typed validated configuration;
- no insecure secret defaults;
- TLS/mTLS configuration;
- secure secret injection;
- request-size/time limits;
- rate limiting for sensitive admin operations.

Test unauthenticated/unauthorized access, malformed configuration, secret leakage and abuse.

**Exit:** no unauthenticated policy-control surface remains.

## DAY 11 — Failure Modes, Availability and Recovery

**Owner:** QA + Backend Developer + AI/ML Engineer

Test:
- AgentGate unavailable;
- PostgreSQL unavailable;
- stale/invalid active policy;
- policy reload failure;
- audit failure;
- gateway/backend unavailable;
- timeouts;
- malformed ext_authz input;
- concurrent activation;
- service/database restart.

Implement/document explicit degraded-mode behavior. Security failures must not become unintended authorization.

Verify health/readiness, graceful shutdown, startup behavior and recovery.

**Exit:** dependency failures are understood, tested and cannot accidentally permit unauthorized calls.

## DAY 12 — Observability, Performance and Abuse Controls

**Owner:** Backend Developer + AI/ML Engineer + QA

Implement/verify:
- structured JSON logs;
- request/execution correlation;
- OTel traces;
- decision-latency histogram;
- allow/deny metrics;
- policy reload metrics;
- audit failure metrics;
- dependency health metrics;
- sensitive-argument non-logging.

Measure:
- authorization latency;
- concurrent decisions;
- policy reload under traffic;
- audit throughput;
- database contention;
- gateway-to-AgentGate latency.

Test retry storms, repeated calls, oversized requests and rapid policy mutation.

**Exit:** production behavior is measured and observable.

## DAY 13 — Production Packaging, CI/CD and Supply Chain

**Owner:** Backend Developer + AI/ML Engineer

Implement/verify:
- production image;
- non-root runtime;
- minimal base;
- read-only filesystem where feasible;
- deployment configuration;
- migrations;
- Helm/deployment manifests as applicable;
- CI build/test/race/lint/vulnerability scan;
- integration tests;
- relevant fuzz tests;
- SBOM;
- signed release artifacts/images where supported.

**Exit:** clean environment can build and deploy the release reproducibly.

## DAY 14 — Full Security + Production Readiness Review

**Owner:** QA leads; Lead Architect accepts

Attempt:
- authorization bypass;
- identity spoofing;
- tool spoofing/classification bypass;
- policy rollback abuse;
- unauthorized policy mutation;
- audit tampering/omission;
- replay;
- malformed MCP/ext_authz inputs;
- concurrency/race failures;
- dependency outages;
- credential-boundary failures;
- container/runtime issues.

Review threat model, invariants, architecture decisions, limitations, vulnerabilities, runbook and recovery procedure.

**Exit:** no unresolved critical/high security issue may ship. Any accepted lower-severity risk must be documented.

## DAY 15 — Clean-Room Deployment + V1 Acceptance

**Owner:** Entire team; Lead Architect final acceptance

Run from release artifacts:

```text
Deploy
→ configure IdP
→ configure gateway
→ connect MCP backend
→ load policy
→ classify tools
→ allowed call
→ denied call
→ prove denial never reached backend
→ inspect audit
→ create candidate policy
→ dry-run
→ activate
→ verify changed behavior
→ rollback
→ verify previous behavior
→ restart services
→ verify recovery
```

Verify logs, metrics, traces, backup/restore, migrations, admin authentication, security controls and release integrity.

**Final deliverables:**
- v1 release candidate;
- deployment guide;
- operator/admin guide;
- security/threat model;
- runbook;
- backup/recovery procedure;
- known limitations;
- complete test report;
- release checklist;
- updated architecture decisions.

# 6. Production Gates

### Authorization
- [ ] deny-by-default
- [ ] identity required
- [ ] unknown tools denied
- [ ] policy errors denied
- [ ] argument constraints tested
- [ ] policy version attached to every decision

### Enforcement
- [ ] ALLOW reaches backend
- [ ] DENY cannot reach backend
- [ ] gateway cannot bypass AgentGate
- [ ] malformed requests cannot produce accidental ALLOW

### Identity
- [ ] JWT validated before AgentGate
- [ ] claims mapping configurable
- [ ] ambiguous identity denied
- [ ] downstream credential boundary secured
- [ ] no raw-token passthrough

### Policy
- [ ] candidate validation
- [ ] dry-run
- [ ] activation
- [ ] rollback
- [ ] mutation audit
- [ ] authenticated administration

### Audit
- [ ] allowed decisions recorded
- [ ] denied decisions recorded
- [ ] policy version recorded
- [ ] identity recorded
- [ ] redaction enforced
- [ ] tampering detectable
- [ ] durability semantics tested

### Operations
- [ ] health/readiness
- [ ] graceful shutdown
- [ ] metrics
- [ ] traces
- [ ] structured logs
- [ ] dependency failures documented/tested
- [ ] backup/recovery verified

### Release
- [ ] clean build
- [ ] unit tests
- [ ] integration tests
- [ ] race tests
- [ ] fuzz tests where applicable
- [ ] vulnerability scan
- [ ] SBOM
- [ ] signed artifacts where supported
- [ ] deployment drill
- [ ] security review complete

# 7. Non-Negotiable Tradeoffs

V1 optimizes for **security correctness and deployability over feature breadth**.

Acceptable deferrals:
- real-time HITL;
- mandatory SpiceDB;
- full multi-tenant runtime;
- advanced compliance exports;
- response-side governance;
- advanced session/aggregate policy;
- UI polish.

Unacceptable shortcuts:
- disabling fail-closed behavior;
- unsafe credential forwarding;
- allowing unknown tools;
- unauthenticated policy changes;
- silently losing audit records;
- shipping without rollback;
- unresolved critical/high security defects;
- bypassing AgentGate for convenience;
- treating ToolAnnotations as authoritative security metadata.

# 8. Definition of Done

AgentGate v1 is production-ready only when:

1. A real MCP client works without client-side governance code.
2. agentgateway owns MCP/gateway responsibilities.
3. AgentGate owns identity, policy and audit.
4. Every governed tool call gets a deterministic authorization decision.
5. Unknown/unsafe conditions fail closed.
6. Policy is persistent, versioned, validated, activatable and rollbackable.
7. Operators can safely evolve policy using audit evidence.
8. Decisions are durably and tamper-evidently recorded.
9. Administration is authenticated and authorized.
10. Downstream identity does not rely on unsafe token passthrough.
11. Dependency failures cannot accidentally authorize calls.
12. The system is observable and performance characteristics are measured.
13. The release artifact is reproducible and deployable.
14. Backup/recovery and operational procedures are tested.
15. Security and integration tests prove the enforcement boundary.
16. No critical/high unresolved security defect remains.
17. The complete Day-15 clean-room deployment succeeds.

**V1 target:** a small but genuinely production-capable governance enforcement layer—not a feature-complete enterprise platform.
