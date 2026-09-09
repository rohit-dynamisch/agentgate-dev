# AgentGate — Production Security Invariants

**Status:** Day 1 architecture-freeze — accepted invariants and blocking-decision register
**Purpose:** Governing security contract for AgentGate v1 production implementation.
**Scope:** `docs/PHASES/DAY-01-TASK-01.md` (DAY-01/TASK-01) under
`docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md`. This document freezes invariants and bounds
blocking decisions; it does not implement any of them. No later implementation task may contradict
what is stated here without an explicit, accepted exception (§13).

## 1. Trust Boundary

The intended request path is:

AI Agent / MCP Client → agentgateway → AgentGate ext_authz → agentgateway → MCP backend.

agentgateway owns MCP transport, routing, tool discovery and inbound JWT validation. AgentGate is
trusted only when reached through the authenticated gateway-to-AgentGate service boundary. AgentGate
is not a public MCP endpoint and does not speak MCP.

A direct or unauthenticated caller to the AgentGate decision service is not a trusted request. See
§9 (Deployment Trust Topology) for how this is enforced at the network/deployment level, not just
as a logical statement.

## 2. Authorization Invariants

1. Missing, invalid, ambiguous or unusable identity results in DENY.
2. Unknown or unclassified tools result in DENY.
3. No matching policy results in DENY.
4. Cedar evaluation errors result in DENY.
5. Malformed or incomplete authorization input cannot produce ALLOW.
6. Only explicitly declared per-tool argument attributes may influence an authorization decision.
   Relevant tool arguments are authorization attributes when policy depends on them, but arbitrary
   or undeclared request arguments must never silently become policy inputs — this is the
   argument-authorization boundary, and it is intentionally narrow (tracked as O-006; the typed
   per-tool declaration mechanism is a Day 2/Day 3 design, not decided here).
7. Every decision carries the exact policy version/hash used for evaluation — specifically the
   version actually evaluated for that decision, not merely whatever is currently active at write
   time. Policy activation concurrency (§4) must not be allowed to make these drift apart; a
   decision and its audit record must always agree on which policy version produced it.
8. DENY must prevent the governed request from reaching the backend.
9. Irreversible/high-impact operations remain explicitly denied unless a policy grants them.
10. MCP ToolAnnotations are never treated as authorization authority.

## 3. Identity Invariants

The inbound bearer token is not a downstream credential. This is a non-negotiable v1 invariant,
settled at Day 1: **AgentGate must never forward the received bearer token to a downstream MCP
server, under any circumstance, including as a temporary shortcut.**

agentgateway validates the inbound token. AgentGate consumes trusted identity/claims across its
authenticated service boundary according to explicit claims-mapping configuration.

The *mechanism* by which AgentGate conveys the caller's identity to a downstream tool without
forwarding the raw token is **not decided at Day 1** — it is Blocking Decision #1 (§12.1), which
also records an explicit, load-bearing source divergence between `docs/PROJECT_DEFINITION.md` and
`docs/TECH_STACK.md` that must not be silently resolved by picking one document over the other
without saying so. Read §12.1 before implementing anything in this area.

## 4. Policy Invariants

PostgreSQL is the persistent control-plane source for versioned policy. The active policy used for
request decisions is held in AgentGate memory.

A candidate policy must be parsed and validated before activation. Activation is an atomic
replacement of the active in-memory policy. Readers must see either the previous valid policy or
the newly activated valid policy, never a partially loaded policy.

A failed reload leaves the last known-good policy active. Startup with no valid policy must not
authorize requests.

Every authorization decision records the exact policy version/hash evaluated (see §2 item 7).

See Blocking Decision #3 (§12.3) for what remains an implementation-level open question here.

## 5. Audit Invariants

Every authorization decision — ALLOW and DENY — must have a defined durable audit outcome.

The audit record must contain, at minimum, identity context, workspace, tool, decision, reason,
correlation/execution identifier, timestamp and policy version/hash, subject to the defined
redaction policy.

Arguments must not become an accidental secrets/PII store. Each governed tool has an explicit
redaction mode.

Audit integrity must be tamper-evident through hash chaining and restricted application
permissions.

If PostgreSQL is an asynchronous sink, the architecture must define a durable local audit
buffer/journal as the precondition for ALLOW. If the required durable audit record cannot be
persisted, the request must fail closed rather than silently authorizing without an audit record.

A PostgreSQL outage must not by itself invalidate an already-valid in-memory policy while the
defined durable audit mechanism remains operational.

**This resolves `docs/DECISIONS/OPEN_DECISIONS.md` O-002 at the invariant level** — see §12.2 for
the full decision record and what remains as Day 7 implementation work rather than an open
architectural question.

## 6. Service Communication

Production security-sensitive service communication uses authenticated TLS/mTLS.

At minimum:
- agentgateway → AgentGate: mTLS;
- AgentGate → PostgreSQL: TLS;
- administrative API: authenticated HTTPS/TLS.

AgentGate should not be externally reachable as a substitute for the gateway enforcement path (see
§9).

See Blocking Decision #5 (§12.5) for what is bounded here versus what remains a Day 8/Day 10
implementation question.

## 7. Administration

Policy mutation is a security-sensitive control-plane operation.

Administrative operations require authenticated identity and explicit authorization. Candidate
creation, validation, activation and rollback are auditable operations.

The production security model must not depend on unauthenticated local/container access — editing
a mounted policy file or a container's environment is not an acceptable substitute for an
authenticated admin control surface in production.

See Blocking Decision #4 (§12.4).

## 8. Availability and Failure Behavior

Security failures fail closed.

In particular:
- invalid identity → DENY;
- invalid policy → DENY;
- policy evaluation error → DENY;
- audit durability failure → DENY;
- gateway trust failure → DENY;
- downstream credential failure → DENY;
- malformed ext_authz input → DENY (never a parse-error-shaped ALLOW).

## 9. Deployment Trust Topology

This section states, at the deployment level, what §1 states logically: the trust boundary must be
real, not just documented.

- Production topology is single-tenant per deployment (per `docs/PROJECT_DEFINITION.md §7`):
  `workspace_id` is latent in the schema for a future hosted offering, but v1 does not deploy
  multiple tenants behind one runtime.
- agentgateway is the only component in the request path that is externally reachable. AgentGate's
  ext_authz gRPC listener, its admin API, and PostgreSQL are internal-network-only — never exposed
  directly to the internet, and never positioned as an alternate public entry point that could be
  used to bypass gateway JWT validation.
- The admin API/UI is reachable only over an authenticated channel and should be network-restricted
  (e.g. internal network, VPN, or bastion) wherever the deployment model allows it — authentication
  (§7) is necessary but network exposure is a second, independent control, not a substitute for it.
- Any deployment that exposes AgentGate's decision or admin surface directly to untrusted networks
  violates this document's trust boundary regardless of what authentication is layered on top; that
  is a deployment-topology defect, not an acceptable configuration choice.

See Blocking Decision #6 (§12.6) for what is bounded here versus what remains open (the specific
manifests/defaults are a Day 13 deliverable; Day 15's clean-room drill verifies them).

## 10. Backup and Recovery Expectations

PostgreSQL holds the two things this product cannot lose: the versioned policy history and the
audit log. Both must be covered by a defined, tested backup/recovery procedure before v1 is
considered production-ready.

- Recovery must restore both the policy version history and the audit log without silently
  fabricating or silently dropping records. A recovery procedure that reconstructs "a" policy but
  loses the version/hash lineage, or that recovers audit rows but breaks the hash chain (§5), does
  not satisfy this invariant.
- A restore must not leave AgentGate authorizing requests against an unknown or unverified policy
  state. Recovery re-establishes a specific, known-good active policy version — it never results in
  "no policy loaded yet, so allow" or "policy state unclear, so allow." The startup invariant in §4
  (no valid policy → no authorization) applies identically after a restore as it does on first
  boot.
- The specific backup mechanism (e.g. `pg_dump`, WAL archiving, managed-service snapshots) is not
  decided at Day 1 — it is deployment-topology- and operator-dependent. What is fixed at Day 1 is
  the *guarantee* the mechanism must satisfy, above.

See Blocking Decision #7 (§12.7).

## 11. Scope Boundaries

V1 excludes real-time per-call HITL, Slack/Teams approval, mandatory SpiceDB, full multi-tenant
runtime, ML risk scoring, a custom MCP proxy, response-side governance, and advanced
aggregate/session authorization unless required by a concrete security invariant.

Resource ownership remains the backend/tool's responsibility. AgentGate governs the caller and tool
call, not individual records.

## 12. Blocking Decisions Register

Per `docs/PHASES/DAY-01-TASK-01.md`, each blocking decision below states: the decision, its
rationale/security impact, its implementation consequence, and any experiment/validation still
required. A decision with "yes" under experiment/validation is **bounded, not fully resolved** —
the invariant it must satisfy is fixed now; the concrete mechanism is deliberately left to the day
named.

### 12.1 Downstream identity / credential mechanism

**Decision:** Raw inbound bearer token passthrough to downstream MCP backends is prohibited,
unconditionally, for v1 (§3). The exact mechanism for conveying caller identity to a downstream
tool without passthrough is **not decided at Day 1**.

**Explicit source divergence (preserved, not silently rewritten):**

- `docs/PROJECT_DEFINITION.md §4` frames the "correct form" as **RFC 8693 token exchange** — "trade
  the incoming token for a short-lived one scoped to this tool and this call" — and records it as
  an open item: "Not built. Must be designed before we forward tokens."
- `docs/TECH_STACK.md §0.2` and `§2.4` later record an externally-verified correction: **MCP forbids
  token passthrough** by spec (a server/proxy "must not forward the token it received to a
  downstream server"), and the spec-sanctioned mechanism for carrying "agent acting for user X"
  through a shared client is the **MCP Enterprise-Managed Authorization extension (ID-JAG, an
  Identity Assertion JWT)** — not RFC 8693. `docs/TECH_STACK.md §0.2` states this explicitly:
  *"RFC 8693 token exchange is not referenced by the spec. This overrides the RFC-8693 assumption
  in the project definition's open questions."* `docs/TECH_STACK.md §2.4` further directs: "Plan
  around ID-JAG, treat RFC 8693 as a fallback only for IdPs that cannot do ID-JAG."
- **This document adopts `docs/TECH_STACK.md`'s conclusion as the v1 direction**, per its own
  explicit statement that it overrides `docs/PROJECT_DEFINITION.md`'s RFC-8693-as-primary framing.
  `docs/PROJECT_DEFINITION.md §4` is not amended by this document and is not being silently
  rewritten — it remains the historical record of the earlier assumption; `docs/TECH_STACK.md` is
  the later, externally-verified, and therefore controlling source for *which mechanism family* is
  in play. Neither document is deleted or edited to hide the disagreement.
- The *specific* mechanism within the ID-JAG-oriented direction (ID-JAG re-assertion? a
  per-call minted credential? a gateway-injected header the downstream tool trusts?) is undesigned
  — `docs/TECH_STACK.md`'s own O-1 calls this "the most important open technical question" and
  explicitly says "the replacement is undesigned." This document does not resolve that; it only
  fixes the *prohibition* (no passthrough, ever) and the *direction* (ID-JAG-primary, RFC 8693 as
  IdP-capability fallback only, not the default plan).

**Rationale/security impact:** naive token forwarding grants every downstream tool a credential
broader than the single action needs (wrong audience, replayable) and is forbidden by the MCP spec
itself; ID-JAG is the spec-sanctioned mechanism for the on-behalf-of relationship this product's
identity model depends on.

**Implementation consequence:** no task before Day 9 may implement any downstream credential
forwarding, including as a "temporary" or "for now" shortcut — `docs/PROJECT_DEFINITION.md §5`'s
"Deliberately dropped" discipline and `docs/PHASES/PHASE-01-MINIMAL-E2E-ENFORCEMENT.md`'s explicit
out-of-scope list already exclude this from earlier work, and this document extends that
prohibition forward through Day 8. `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md` Day 9 is
reserved for this design; if it cannot be safely implemented by Day 9, the plan's own instruction
governs: narrow the deployment scope rather than ship an unsafe design.

**Experiment/validation still required:** yes. Day 9 must verify ID-JAG issuance/acceptance against
the actual chosen IdP(s) and confirm agentgateway's actual support (or lack of it) for the chosen
mechanism, then select and implement the concrete mechanism. This is unresolved by design at Day 1
— tracked as `docs/DECISIONS/OPEN_DECISIONS.md` O-001, which remains open (not resolved by this
document; see the OPEN_DECISIONS.md update note in the task report).

### 12.2 Audit durability semantics

**Decision:** An ALLOW must not be returned unless a durable audit outcome for that exact decision
is guaranteed — either a synchronous PostgreSQL write, or a durable local buffer/journal that
survives a process crash with PostgreSQL as the eventual persistent sink. A failure to durably
record the audit outcome fails the request closed (DENY); it must never silently authorize without
a record.

**Rationale/security impact:** an ALLOW with no corresponding durable record breaks the product's
core provenance guarantee (`docs/PROJECT_DEFINITION.md §6`) — the entire "detective governance"
model depends on every decision being reconstructable later. This directly answers the question
`docs/DECISIONS/OPEN_DECISIONS.md` O-002 raised (whether an ALLOW may be returned before its audit
event is durably persisted): no.

**Implementation consequence:** Day 7 must implement the durable local buffer/journal and the
audit-write failure path that denies rather than allows-without-audit. `docs/TECH_STACK.md §2.5`'s
"async write off the hot path" idea is not rejected outright, but it must be backed by a durable
local buffer that satisfies the guarantee above — an async write with no durability guarantee
before the response is returned does not satisfy this invariant.

**Experiment/validation still required:** at the implementation level only (Day 7) — the specific
durable-buffer technology/format (e.g. an embedded write-ahead log, a local queue with fsync) is not
decided at Day 1; that is an implementation decision, not an open architectural question. This is
why this document treats O-002 as resolved at the invariant level: what remains is *how*, not
*whether*.

### 12.3 Policy reload / activation

**Decision:** the active policy lives in AgentGate memory; PostgreSQL is the persistent source of
truth; activation is an atomic in-memory swap (readers see old-valid or new-valid, never partial);
a failed reload/validation leaves the last known-good policy active; startup with no valid policy
must not authorize any request.

**Rationale/security impact:** non-atomic activation could evaluate a request against a
half-updated, internally inconsistent policy set; allowing requests before any valid policy has
loaded would be an unbounded fail-open window at every cold start.

**Implementation consequence:** Day 4 implements content-hash-versioned policy rows and an atomic
in-memory swap (e.g. replacing an immutable evaluator instance behind a single pointer/reference),
plus startup logic that blocks authorization until a valid policy is loaded.

**Experiment/validation still required:** yes, at the implementation level — Day 4's own test scope
already calls for "concurrent reads during activation"; this is expected engineering validation, not
an open Day 1 question.

### 12.4 Admin authentication

**Decision:** every policy-mutating operation (create candidate, validate, activate, rollback)
requires authenticated identity and explicit authorization. The production security model must not
depend on unauthenticated local/container access.

**Rationale/security impact:** policy mutation is as powerful as the policy itself
(`docs/PROJECT_DEFINITION.md §6`) — an unauthenticated admin surface is equivalent to having no
authorization system at all, regardless of how strong the request-path enforcement is.

**Implementation consequence:** Day 5 (policy governance API/UI) must not ship any mutation path
reachable without authentication; Day 10 hardens it further. Every mutation is itself an audited
event (§7, §5).

**Experiment/validation still required:** the concrete admin-identity mechanism (reuse the same
OIDC IdP as the request path, or a separate operator credential) is not decided at Day 1 — this is
scoped as Day 5/Day 10 implementation work, deliberately deferred since it does not change the Day 1
invariant ("must be authenticated," independent of which mechanism).

### 12.5 TLS/mTLS

**Decision:** agentgateway↔AgentGate uses mTLS; AgentGate↔PostgreSQL uses TLS; the administrative
API uses authenticated HTTPS/TLS (§6). AgentGate must not be externally reachable as a weaker
alternate entry path around the gateway (§9).

**Rationale/security impact:** the gateway→AgentGate link carries the actual authorization decision
request/response. Unauthenticated or unencrypted, it lets anyone on the network path spoof
decisions or read policy-sensitive traffic, defeating the §1 trust-boundary model regardless of
what Cedar/audit logic is implemented behind it.

**Implementation consequence:** Day 10 wires actual certificates/mTLS configuration. Day 1 does not
select a certificate-management approach (self-signed for dev, cert-manager, customer-provided PKI)
— that is deployment-topology-dependent (§9, §12.6) and intentionally not decided here.

**Experiment/validation still required:** yes — verifying agentgateway's actual mTLS
support/configuration surface for its `ext_authz` callout is Day 8/Day 10 integration work, related
to `docs/DECISIONS/OPEN_DECISIONS.md` O-003 (agentgateway conformance), and is not resolved at Day
1.

### 12.6 Deployment topology

**Decision:** see §9 in full. Summary: single-tenant per deployment; agentgateway is the only
externally-reachable component; AgentGate's decision/admin surfaces and PostgreSQL are
internal-network-only; the admin surface is never a substitute public entry point.

**Rationale/security impact:** this is the deployment-side enforcement of §1 — the request-path and
identity invariants above are meaningless if AgentGate's ports are reachable directly from the
internet, bypassing agentgateway's JWT validation entirely.

**Implementation consequence:** Day 13 (packaging/deployment) must ship default manifests/config
that do not expose AgentGate's authz or admin ports externally; operator documentation (a Day 15
deliverable) must state this requirement explicitly for self-hosted operators assembling their own
network topology.

**Experiment/validation still required:** none blocking Day 1. Day 15's clean-room deployment drill
must verify the actual shipped defaults enforce this, not merely that it is documented.

### 12.7 Backup/recovery

**Decision:** see §10 in full. Summary: PostgreSQL (policy + audit) must be covered by a tested
backup/recovery procedure; recovery must restore both without fabricating or dropping records; a
restore must never leave AgentGate authorizing against an unknown/unverified policy state.

**Rationale/security impact:** without a defined recovery guarantee, a database failure could
either permanently lose the audit trail (defeating the product's core promise) or come back in a
state that silently fails open.

**Implementation consequence:** Day 13 defines and tests an actual backup/recovery procedure. The
specific mechanism is not decided at Day 1 — it is deployment-topology- and operator-dependent.

**Experiment/validation still required:** yes — Day 15's clean-room deployment drill explicitly
includes "restart services → verify recovery" and backup/restore as acceptance steps; the concrete
mechanism and its validation is deferred there, not decided now.

## 13. Acceptance Rule

No later implementation task may contradict these invariants.

Any required exception must be recorded as an explicit architecture decision and accepted by the
Lead Architect before implementation.

This document itself requires Lead Architect acceptance and QA review before Day 2 may begin (per
`docs/PHASES/DAY-01-TASK-01.md` Exit Criteria). It does not authorize any implementation on its own.
