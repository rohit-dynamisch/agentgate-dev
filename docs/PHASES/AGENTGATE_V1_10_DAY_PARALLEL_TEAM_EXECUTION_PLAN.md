# AgentGate v1 — 10-Day Parallel Team Execution Plan

**Purpose:** Coordinate the remaining 10 days of AgentGate v1 development as parallel workstreams rather than a sequential implementation queue.

**Planning basis:** This plan compresses the existing 15-day production-readiness plan after completion of Day 1 and Day 2. It preserves the project's existing architecture, security invariants, ownership boundaries, and production gates.

**Current state:** Day 1 — Production Architecture Freeze, and Day 2 — Decision Core Productionization, are complete. The remaining work must proceed concurrently across Go Backend, Gateway/MCP, Frontend/UI, QA/Security, and DevOps/Release Engineering.

---

## 1. Shared Architecture and Non-Negotiable Boundaries

The production path remains:

```text
MCP Client / AI Agent
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
  - trusted identity
  - tool/request context
  - Cedar authorization
  - fail-closed enforcement
  - durable audit
  - policy versioning
        |
        v
Real MCP Backend
```

The division of responsibility is fixed:

| Component | Owns | Must not own |
|---|---|---|
| **agentgateway** | MCP transport, routing, JWT validation, tool discovery, ext_authz integration | Cedar policy, durable authorization audit, operator policy lifecycle |
| **AgentGate** | Identity interpretation, tool governance, authorization, Cedar, policy lifecycle, audit | MCP protocol/proxy implementation |
| **MCP backend/tool** | Resource/record-level authorization and business semantics | AgentGate policy decisions |
| **Frontend/UI** | Authenticated operator workflow and visibility | Direct authorization enforcement |
| **QA/Security** | Independent proof of security and production behavior | Product architecture changes without approval |
| **DevOps** | Reproducible build/deploy/runtime environment | Security-policy weakening for deployment convenience |

The following are non-negotiable:

- Missing or invalid identity denies.
- Unknown or unclassified tools deny.
- Missing policy denies.
- Policy evaluation errors deny.
- Malformed authorization input cannot produce an allow.
- Every decision identifies the exact policy version/hash.
- Denied calls must not reach the MCP backend.
- The gateway cannot bypass AgentGate for governed traffic.
- Downstream credentials must not rely on unsafe raw inbound bearer-token passthrough.
- Policy activation must be atomic and rollbackable.
- Policy mutations require authenticated and authorized administration.
- Audit durability semantics must be explicit; silent audit loss is not acceptable.
- No critical/high unresolved security defect may ship.

The v1 scope deliberately excludes real-time per-call HITL, mandatory SpiceDB/resource ownership, ML risk scoring, building a custom MCP proxy, full multi-tenant runtime, advanced compliance exports, and other explicitly deferred features.

---

## 2. Parallel Workstreams

### WS-A — Go Backend / AgentGate Core

**Primary owner:** Go Backend Developer
**Architectural reviewer:** Lead Architect
**Supporting:** QA/Security

#### Objective

Complete the AgentGate control plane and decision service: identity, tool governance, policy persistence/lifecycle, governance APIs, audit, configuration, operational controls, and the tests that prove these behaviors.

#### Days 3–4: Identity, Tool Governance, Policy Persistence

Implement:

- Typed/configurable claims mapping
- Required identity-field validation
- Agent and on-behalf-of identity representation
- Role extraction
- Tool fingerprinting
- Explicit tool classification
- Classification drift detection
- Unknown-tool deny
- PostgreSQL policy schema
- `workspace_id`
- Content-hash policy versions
- Candidate/active/rollback lifecycle
- Atomic activation
- Migrations
- Active in-memory policy loading/reloading

#### Days 5–7: Governance + Dry-Run + Audit

Implement:

- Authenticated policy-management API
- Policy version inspection
- Candidate creation
- Policy validation
- Impact preview
- Activation
- Rollback
- Policy mutation audit
- Historical decision replay
- Dry-run outcome comparison
- Durable decision audit
- Argument redaction
- Policy hash/version recording
- Correlation/execution ID
- Audit hash chaining
- Defined audit failure behavior

#### Days 8–10: Integration + Security + Runtime Hardening

Integrate the finalized ext_authz/gateway contract.

Implement/verify:

- Downstream identity/credential boundary selected for v1
- Authenticated/authorized admin surface
- Typed validated configuration
- Secret injection
- TLS/mTLS configuration
- Request size/time limits
- Rate limiting
- Health/readiness
- Graceful shutdown
- Structured logging
- OpenTelemetry
- Metrics
- Dependency failure behavior
- Policy reload behavior
- Concurrency correctness

#### Backend deliverables

- AgentGate Go service
- Database migrations
- Policy lifecycle
- Governance API
- Durable audit
- Configuration
- Operational endpoints
- Unit/integration tests
- Security/failure/concurrency tests
- API/interface documentation

#### Backend DoD

Backend work is complete only when:

1. All externally consumed request/response contracts are documented and tested.
2. Identity and tool classification are fail-closed.
3. Policy is persistent, versioned, validated, activatable and rollbackable.
4. Every decision contains policy provenance.
5. Audit behavior matches the defined durability semantics.
6. Admin operations are authenticated and authorized.
7. The service passes unit, race, integration, security and failure tests applicable to its scope.
8. No shared API is changed without updating its contract tests and notifying dependent streams.

---

### WS-B — Gateway / MCP Integration

**Primary owner:** Gateway/MCP Engineer
**Architectural reviewer:** Lead Architect
**Supporting:** Go Backend + QA/Security

#### Objective

Prove that the real MCP traffic path is governed by AgentGate through agentgateway, without requiring client-side governance code.

#### Days 3–4: Integration Harness and Contract

Build the gateway-side integration environment around:

```text
MCP Client
→ agentgateway
→ JWT validation
→ ext_authz
→ AgentGate
→ policy decision
→ agentgateway
→ MCP backend
```

Establish:

- agentgateway configuration
- JWT validation configuration
- ext_authz connection
- Request-body handling
- Identity propagation
- MCP client
- Instrumented/toy MCP backend
- Reproducible local/integration environment

The gateway engineer should initially use the Day-2 AgentGate contract and mocks where the final backend implementation is not yet available.

#### Days 5–7: MCP Enforcement Integration

Verify:

- MCP `2026-07-28`
- Tool discovery consistency
- Tool names/schema/fingerprint behavior
- ext_authz request construction
- Identity propagation
- Authorization response handling
- JSON-RPC denial behavior
- No gateway-side bypass path

Build deterministic integration tests for:

- Valid identity + allowed tool → backend reached
- Valid identity + denied tool → backend not reached
- Unknown tool → denied
- Malformed ext_authz input → denied
- Missing/invalid identity → denied
- AgentGate unavailable → defined fail-closed behavior

#### Days 8–10: Real E2E + Credential Boundary

Run the complete path against the real AgentGate implementation.

Support the approved downstream identity mechanism and verify:

- Correct audience
- Scoped/short-lived credentials where applicable
- No raw inbound bearer passthrough
- Caller/on-behalf-of identity remains auditable
- Credential failure denies
- Replay/confusion risks are addressed

#### Gateway/MCP deliverables

- agentgateway production configuration
- MCP test client
- Instrumented MCP backend
- Integration harness
- E2E tests
- Identity propagation test cases
- Documented gateway/AgentGate contract

#### Gateway/MCP DoD

This stream is complete only when:

1. A real MCP client reaches a real MCP backend through agentgateway.
2. AgentGate is invoked for governed calls.
3. ALLOW reaches the backend.
4. DENY does not reach the backend.
5. Unknown/malformed/unauthenticated conditions cannot bypass authorization.
6. Gateway configuration is reproducible from a clean environment.
7. MCP version/tool-schema behavior is tested.
8. The downstream credential boundary has been security-reviewed.

---

### WS-C — Frontend / Admin UI

**Primary owner:** Frontend/UI Developer
**API owner:** Go Backend
**Security reviewer:** QA/Security

#### Objective

Provide the minimum operator interface required for safe policy governance. UI polish is secondary to security and reliability.

#### Days 3–4: UI Contract and Skeleton

Do not wait for the backend implementation.

Define against a mocked API:

- Policy list
- Policy version details
- Candidate policy
- Validation result
- Dry-run result
- Activation
- Rollback
- Audit events
- Tool inventory/classification

Freeze the frontend-facing API contract with the Backend stream.

#### Days 5–7: Governance Workflow

Implement:

```text
View active policy
→ create candidate
→ validate
→ inspect impact/dry-run
→ activate
→ inspect audit
→ rollback
```

The UI must make state explicit:

- Candidate vs active
- Policy version/hash
- Validation status
- Changed outcomes
- Activation status
- Rollback target
- Audit event

The UI must never imply that a candidate is active before the backend confirms activation.

#### Days 8–10: Security + Integration

Integrate against the authenticated admin API.

Verify:

- Unauthenticated user cannot manage policies
- Unauthorized user cannot activate/rollback
- Failed API calls do not produce false success
- Stale policy state is detectable
- Destructive operations require appropriate confirmation
- Audit results are visible
- Error states are explicit

#### Frontend deliverables

- Minimum admin UI
- Typed API client/models
- Policy lifecycle screens
- Tool inventory/classification view
- Dry-run impact view
- Audit view
- Authentication/authorization integration
- UI test coverage for critical workflows

#### Frontend DoD

1. Every displayed policy state comes from the backend.
2. Candidate/active/rollback states cannot be confused.
3. Activation and rollback only appear successful after server confirmation.
4. Unauthorized operations are rejected by the backend and handled correctly by the UI.
5. The complete governance workflow works against the real API.
6. UI tests cover failure and stale-state behavior.
7. No UI work introduces a bypass or alternate policy-control path.

---

### WS-D — QA / Security

**Primary owner:** QA/Security Engineer
**Security acceptance:** Lead Architect
**Dependencies:** Receives mocks/contracts from every other stream

#### Objective

Act as an independent verification stream rather than simply testing the implementation after it is finished.

#### Days 3–4: Test Harness + Security Invariants

Build the acceptance harness around explicit security properties:

- Missing identity → deny
- Invalid identity → deny
- Unknown tool → deny
- Unclassified tool → deny
- Missing policy → deny
- Policy error → deny
- Malformed request → deny
- Argument constraints enforced
- Exact policy version attached
- Deny does not reach backend

Create an instrumented MCP backend that records whether a request was actually executed.

#### Days 5–7: Integration, Abuse and Policy Tests

Test:

- Policy candidate validation
- Dry-run isolation
- Activation
- Rollback
- Concurrent reads during activation
- Unauthorized policy mutation
- Audit creation
- Argument redaction
- Audit-chain integrity
- Replay
- Classification drift
- Tool spoofing
- Identity spoofing
- Malformed ext_authz payloads

#### Days 8–10: End-to-End and Failure Testing

Execute full-path tests and begin failure injection:

- AgentGate unavailable
- PostgreSQL unavailable
- Stale/invalid policy
- Policy reload failure
- Audit failure
- Gateway unavailable
- Backend unavailable
- Timeout
- Restart
- Concurrent activation
- Retry storm
- Oversized requests
- Rapid policy mutation

Measure and record:

- Whether authorization remains fail-closed
- Whether audit semantics remain correct
- Whether recovery works
- Whether duplicate/replayed requests cause unsafe behavior

#### QA/Security deliverables

- Automated acceptance suite
- E2E enforcement suite
- Negative/security suite
- Failure-injection suite
- Concurrency tests
- Regression suite
- Test report
- Security findings
- Release acceptance checklist

#### QA/Security DoD

1. Critical authorization paths have automated tests.
2. ALLOW→backend and DENY→no-backend are proven on the real path.
3. Security invariants are tested independently of implementation details.
4. Failure modes have explicit expected outcomes.
5. No critical/high security defect remains open at release.
6. All accepted lower-severity risks are documented.
7. Final clean-room deployment and release acceptance pass.

---

### WS-E — DevOps / Release Engineering

**Primary owner:** DevOps/Release Engineer
**Supporting:** Gateway/MCP + Go Backend
**Security input:** QA/Security

#### Objective

Make AgentGate reproducibly buildable, deployable, observable and operable from a clean environment.

#### Days 3–4: Environment Foundation

Prepare:

- Docker build
- Local/integration compose environment
- PostgreSQL
- agentgateway
- MCP backend
- AgentGate
- Configuration injection
- Migration execution path
- Health/readiness checks

Ensure the environment can be recreated without manual hidden state.

#### Days 5–7: Deployment + CI

Implement/verify:

- Production image
- Non-root runtime
- Minimal base
- Read-only filesystem where feasible
- Deployment configuration
- Migration process
- Helm/deployment manifests as applicable
- CI tests
- Race detection
- Linting
- Vulnerability scanning

Add integration test execution where environment permits.

#### Days 8–10: Supply Chain + Operations

Complete:

- Reproducible release build
- SBOM
- Signed artifacts/images where supported
- Secrets injection
- TLS/mTLS configuration
- Backup/recovery procedure
- Deployment rollback procedure
- Logs/metrics/traces export
- Resource limits
- Operational runbook
- Clean-room deployment script/instructions

#### DevOps deliverables

- Production container
- Compose/integration environment
- Deployment manifests
- CI/CD pipeline
- Migration process
- SBOM
- Signing
- Backup/recovery procedure
- Operator runbook
- Clean-room deployment procedure

#### DevOps DoD

1. A clean checkout can build the release.
2. The release can start with documented configuration.
3. Database migrations execute deterministically.
4. Health/readiness behavior is correct.
5. Runtime does not require insecure defaults.
6. CI passes required tests/scans.
7. Release artifacts are traceable and reproducible.
8. Backup/recovery and deployment rollback are demonstrated.
9. A clean environment can run the full E2E acceptance path.

---

## 3. Integration Gates

The team should not synchronize continuously. Synchronize at explicit contracts and evidence gates.

### Gate G1 — Authorization Contract Freeze

**Target:** Day 3
**Streams:** Go Backend + Gateway/MCP + QA/Security + Frontend/API consumer

**Must agree on:**

- ext_authz request structure
- Identity fields
- Tool identifier
- Tool classification
- Arguments/context representation
- Decision result
- Denial/error semantics
- Policy version/hash
- Correlation/execution ID
- Workspace identifier
- Authentication assumptions

**Required evidence:**

- Typed contract in code
- Representative allow/deny payloads
- Malformed-input examples
- Mock server/client
- Contract tests

**DoD:**

The gateway can call a mock AgentGate and QA can independently assert:

```text
valid request       → deterministic decision
missing identity    → deny
unknown tool        → deny
malformed request   → deny
policy error        → deny
```

The Frontend can also begin against a versioned API contract without waiting for implementation.

**Merge rule:** After G1, changing a shared payload requires explicit architecture review and dependent-test updates.

---

### Gate G2 — Identity + Tool Governance Boundary

**Target:** End of Day 4
**Streams:** Go Backend + Gateway/MCP + QA/Security

**Must agree on:**

- Trusted JWT claims
- Configurable claims mapping
- Required identity fields
- Agent/on-behalf-of semantics
- Role representation
- Tool fingerprint
- Classification
- Classification drift behavior

**DoD:**

QA demonstrates:

```text
valid claims + known classified tool → normal evaluation
missing/ambiguous claims             → DENY
new/unknown tool                     → DENY
changed tool schema/fingerprint      → DENY or explicit reclassification
spoofed classification               → DENY
```

The gateway configuration produces the exact identity/tool context expected by AgentGate.

---

### Gate G3 — Policy Persistence + API Contract

**Target:** End of Day 5
**Streams:** Go Backend + Frontend/UI + QA + DevOps

**Must agree on:**

- Policy states
- Policy version/hash
- Candidate/active semantics
- Create/validate/preview/activate/rollback API
- `workspace_id`
- Migration behavior
- Error model

**DoD:**

Backend provides a running API and migration.

Frontend completes the workflow against either the real service or a contract-compatible mock.

QA verifies:

- Invalid candidate cannot activate
- Activation is atomic
- Rollback selects a known version
- Active version is explicit
- Concurrent reads never observe an invalid active policy

DevOps can recreate the database and apply migrations from a clean environment.

---

### Gate G4 — Governance Workflow Integration

**Target:** End of Day 7
**Streams:** Go Backend + Frontend/UI + QA/Security

**Must prove:**

```text
candidate
→ validate
→ dry-run
→ inspect changed outcomes
→ activate
→ observe changed authorization
→ rollback
→ observe previous authorization
```

**DoD:**

1. Dry-run never mutates active policy.
2. Activation changes the active policy atomically.
3. Rollback restores a previous known version.
4. UI accurately reflects backend state.
5. Policy mutation is audited.
6. QA can prove the authorization result before and after activation.

---

### Gate G5 — Durable Audit Boundary

**Target:** End of Day 7
**Streams:** Go Backend + QA/Security + DevOps

**Must agree on:**

- Required audit fields
- Redaction rules
- Policy provenance
- Correlation ID
- `workspace_id`
- Hash-chain behavior
- Audit failure semantics
- Database permissions
- Retention/backup expectations

**DoD:**

For both ALLOW and DENY, QA can retrieve a durable record containing the required provenance.

QA also proves:

- Sensitive arguments are not logged
- Policy version/hash is present
- Tampering is detectable
- Restart does not silently lose already-committed records
- Defined audit failure behavior occurs

---

### Gate G6 — Real MCP Enforcement Gate

**Target:** Day 8
**Streams:** Gateway/MCP + Go Backend + QA/Security + DevOps

**Critical proof:**

```text
MCP Client
  ↓
agentgateway
  ↓
AgentGate
  ↓
Cedar
  ↓
MCP Backend
```

**DoD** — the real environment proves:

| Scenario | Expected |
|---|---|
| Authenticated + allowed tool | Backend receives exactly one governed call |
| Authenticated + denied tool | Backend receives zero calls |
| Unknown tool | Zero backend calls |
| Invalid identity | Zero backend calls |
| Malformed ext_authz request | Zero backend calls |
| AgentGate unavailable | Defined fail-closed behavior |
| Policy evaluation failure | Zero backend calls |

This is the most important integration gate in the remaining schedule.

---

### Gate G7 — Downstream Credential Boundary

**Target:** Day 9
**Streams:** Go Backend + Gateway/MCP + QA/Security + Lead Architect

**DoD:**

The selected v1 mechanism demonstrates:

- Correct downstream audience
- Appropriate credential scope/lifetime
- No raw inbound bearer-token passthrough
- Caller/on-behalf-of identity remains auditable
- Invalid/expired credential behavior is understood
- Replay/confusion risks are addressed
- Failure cannot accidentally authorize a call

If this cannot be safely demonstrated, deployment scope must be narrowed rather than weakening the security boundary.

---

### Gate G8 — Production Environment Readiness

**Target:** End of Day 10
**Streams:** DevOps + Go Backend + Gateway/MCP + QA

**DoD** — from a clean environment:

1. Images build or are pulled from release artifacts.
2. Configuration is injected securely.
3. Migrations execute.
4. All services become ready.
5. MCP client connects through agentgateway.
6. AgentGate enforces authorization.
7. Audit records are persisted.
8. Logs/metrics/traces are emitted.
9. Restart/recovery works.
10. No manual hidden configuration is required.

This gate means the system is integrable as a deployable product, not merely runnable on one developer machine.

---

### Gate G9 — Release Candidate Security Gate

**Target:** End of Day 10 / final acceptance
**Streams:** QA/Security + all streams + Lead Architect

**DoD:**

The release candidate passes:

- Authorization bypass testing
- Identity spoofing tests
- Tool spoofing/classification tests
- Unauthorized policy mutation tests
- Rollback abuse tests
- Audit tampering/omission tests
- Malformed MCP/ext_authz tests
- Concurrency/race tests
- Dependency outage tests
- Credential-boundary tests
- Container/runtime checks
- CI security scanning

No unresolved critical/high security defect remains.

---

## 4. Ten-Day Dependency Matrix

**Legend:**

- **P** = can progress independently
- **C** = requires a contract/checkpoint but can continue using mocks before integration
- **I** = integration required
- **V** = verification/acceptance

| Workstream | D3 | D4 | D5 | D6 | D7 | D8 | D9 | D10 |
|---|---|---|---|---|---|---|---|---|
| **Go Backend** | Identity/tool governance | Policy persistence | Governance API | Dry-run | Durable audit | E2E integration | Credential boundary | Admin/runtime hardening |
| **Gateway/MCP** | Mock contract + env | Identity/tool integration | MCP enforcement | E2E harness | Contract regression | **REAL E2E** | Credentials | Recovery/perf |
| **Frontend/UI** | Mock API + skeleton | Contract freeze | Real API integration | Governance workflow | Dry-run/audit UI | Auth integration | Failure states | Final UX verification |
| **QA/Security** | Test harness | Boundary tests | Policy tests | Dry-run/activation tests | Audit/security tests | **Enforcement proof** | Credential tests | Failure/security sweep |
| **DevOps** | Integration env | Migrations/config | Image + CI | Deployment | CI/integration | Release env | SBOM/signing/backup | Clean-room readiness |

The streams deliberately overlap. For example, Frontend does not wait for Day 5 to begin; it starts against the API contract. Gateway does not wait for the final Go implementation; it starts with a mock AgentGate. QA does not wait for production code; it builds the enforcement harness and negative tests. DevOps does not wait for the final feature set; it creates the reproducible environment immediately.

---

## 5. Shared Milestone Timeline

### Days 3–4 — Contracts and Foundations

**Shared milestone:** Identity + Tool Governance + Integration Contracts + Deployable Test Environment

Parallel outputs:

```text
Go Backend
  → identity/tool governance

Gateway/MCP
  → agentgateway + MCP environment + AgentGate mock

Frontend
  → API contract + UI skeleton

QA/Security
  → acceptance/security harness

DevOps
  → reproducible integration environment
```

**Required gates:** G1, G2

The purpose of this stage is to remove interface ambiguity early. It is not necessary for every implementation to be finished before the team moves forward.

---

### Days 5–7 — Governance Product Core

**Shared milestone:** Persistent Policy Lifecycle + Admin Workflow + Dry-Run + Durable Audit

Parallel outputs:

```text
Go Backend
  → policy DB/API/dry-run/audit

Frontend
  → operator workflow

QA/Security
  → lifecycle/security verification

Gateway/MCP
  → enforcement regression against mocks/real service as available

DevOps
  → CI/deployment/migration/release foundations
```

**Required gates:** G3, G4, G5

At the end of Day 7, the governance loop must be operational:

```text
Audit evidence
    ↓
Candidate policy
    ↓
Validation
    ↓
Dry-run
    ↓
Activation
    ↓
Observed decision change
    ↓
Rollback
```

---

### Days 8–9 — Real Enforcement + Identity Boundary

**Shared milestone:** Real MCP enforcement with secure downstream identity

Parallel outputs:

```text
Gateway/MCP
  → real MCP traffic

Go Backend
  → real ext_authz + credential boundary

QA/Security
  → enforcement + credential security proof

Frontend
  → authenticated administration + failure handling

DevOps
  → production-like runtime
```

**Required gates:** G6, G7

This is the critical-path integration period. Any interface defect discovered here must be fixed jointly rather than worked around independently.

---

### Day 10 — Production Readiness Consolidation

**Shared milestone:** Clean, reproducible, secure release candidate

All streams converge:

```text
Code
 + Gateway
 + UI
 + Tests
 + Container
 + Config
 + Database
 + Audit
 + Observability
 + Documentation
        ↓
Release Candidate
```

**Required gates:** G8, G9

The output must be a release candidate that can be deployed from a clean environment and subjected to the remaining production-readiness work.

---

## 6. Daily Synchronization Protocol

The team should use a lightweight synchronization model rather than daily sequential handoffs.

Each stream reports only:

```text
DONE:
What was completed and tested.

CONTRACT:
Any interface/API/schema/config contract produced or changed.

BLOCKED:
Only genuine external dependency blockers.

RISK:
Any security, correctness, integration, or schedule risk.

NEXT:
The next independently executable task.
```

A stream is **not blocked** merely because another stream has not finished its implementation if a mock, contract, fixture, or local stub can be used.

A stream is genuinely blocked only when its next task requires a shared contract or environment behavior that has not yet been frozen.

---

## 7. Integration Rules

### Rule 1 — Freeze Contracts Before Implementations

For shared interfaces, agree on the payload/schema/error semantics first.

The preferred sequence is:

```text
Architecture decision
→ contract
→ mock/fixture
→ independent implementation
→ contract test
→ real integration
```

This prevents the Go, Gateway, UI, and QA streams from waiting on one another.

### Rule 2 — Mocks Are Temporary Parallelization Tools

Mocks are allowed for:

- AgentGate ext_authz
- Governance APIs
- Policy responses
- Audit responses
- Gateway/backend test boundaries

Mocks must not become permanent substitutes for the real enforcement path.

Every critical mock contract must eventually be replaced or validated against the real implementation.

### Rule 3 — Security Boundaries Cannot Be Bypassed for Parallelism

Parallel development must never introduce:

- Direct gateway → backend bypass
- Client-side authorization as a substitute for AgentGate
- UI-only authorization
- Raw inbound token forwarding
- Unknown-tool implicit allow
- Fail-open behavior
- Silent audit loss

### Rule 4 — Shared Contract Changes Require an Integration Gate

If a developer changes:

- ext_authz payload
- Identity schema
- Tool classification schema
- Policy API
- Audit schema
- Policy state model
- Deployment configuration contract

the change must include updated contract tests and notification to every dependent stream.

### Rule 5 — Architect Owns Cross-Stream Decisions

Coding agents may implement assigned work but must not silently resolve architectural ambiguity.

When a stream discovers an architectural conflict, stop only that decision-dependent work, record the conflict, and continue all unrelated work.

---

## 8. Critical Path vs Parallel Work

The remaining schedule should distinguish between the **critical path** and work that can happen in parallel.

### Critical path

```text
G1 Contract Freeze
   ↓
G2 Identity/Tool Boundary
   ↓
G3 Policy/API Boundary
   ↓
G4 Governance Workflow
   ↓
G5 Audit Boundary
   ↓
G6 Real MCP Enforcement
   ↓
G7 Credential Boundary
   ↓
G8 Production Environment
   ↓
G9 Security/Release Acceptance
```

### Parallel lanes

The following should not wait on the critical path:

```text
Frontend:
mocked API → UI workflow → real API

QA:
test harness → negative tests → integration tests → failure tests

DevOps:
integration environment → container → CI → deployment → release artifacts

Gateway:
mock AgentGate → MCP path → real AgentGate → credential tests
```

This is the main schedule optimization. The team is not trying to make every stream finish every day; it is trying to make every stream continuously produce artifacts that can be integrated at the next gate.

---

## 9. Ownership of the Final Production Proof

No individual workstream can declare AgentGate v1 production-ready.

Production readiness is a shared property proven only when:

- **Go Backend** proves the authorization and governance logic is correct.
- **Gateway/MCP** proves real MCP traffic cannot bypass that logic.
- **Frontend/UI** proves operators can safely manage policy through the authenticated control plane.
- **QA/Security** independently proves the security invariants, failure behavior, and enforcement boundary.
- **DevOps** proves the system is reproducibly deployable and recoverable.
- **Lead Architect** confirms that the implementation still matches the frozen architecture, scope, threat model, and non-negotiable security invariants.

The final acceptance condition remains:

```text
Real MCP client
→ agentgateway
→ JWT validation
→ AgentGate
→ identity/tool governance
→ Cedar policy
→ durable audit
→ real MCP backend
```

with demonstrated:

```text
ALLOW  → backend reached
DENY   → backend not reached
UNKNOWN/UNSAFE → DENY
POLICY CHANGE → validate → dry-run → activate → rollback
FAILURE → defined fail-closed behavior
AUDIT → durable + provenance + tamper detection
DEPLOY → clean-room reproducible
```

This parallel model is the execution structure for the remaining 10 days; it does not change the AgentGate v1 product scope or architecture.
