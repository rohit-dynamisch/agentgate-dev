# G5 Closure Summary — Durable Audit Boundary

**Checkpoint status:** PASS / CLOSED / FROZEN (Formally approved by Lead Architect on 2026-09-14)  
**Date:** 2026-09-14  
**Branch:** `development`  

---

## 1. What Shipped

Checkpoint G5 establishes a durable, tamper-evident, redacted, and cryptographically verifiable authorization audit boundary for AgentGate. Prior to G5, decisions evaluated against the Cedar policy engine were transient in-process results, and policy mutations were tracked only via in-memory listeners. With G5:

1. **Every authorization decision and policy lifecycle mutation is durably recorded**:
   - Persisted to PostgreSQL in an append-only `audit_events` table indexed by `(workspace_id, sequence_number)`.
   - Guaranteed gap-free, monotonic sequence numbers per workspace via transaction-scoped PostgreSQL advisory locks (`pg_advisory_xact_lock(hashtext(workspace_id))`).
2. **Audit Durability Invariant (O-002 Resolution)**:
   - Evaluated **`ALLOW` + audit write failure &rarr; `DENY` (`ReasonEvaluationError`)**. Under no circumstances can an authorization proceed without confirmed durable audit persistence.
   - Evaluated **`DENY` + audit write failure &rarr; `DENY`**. Security is fail-closed across all operational failure paths.
3. **Pre-Persistence Argument Redaction & Salted Hashing**:
   - Arguments are sanitized **strictly before** canonical serialization, hashing, or database insertion.
   - Supports `full`, `hash` (deterministic salted SHA-256), and `omit` (`[REDACTED]`).
   - Defensive security overrides automatically enforce `omit` on sensitive parameter names (`password`, `token`, `secret`, `key`, `credential`, etc.) or bearer token patterns, overriding unsafe configurations.
4. **Cryptographic Hash Chaining & Independent Tamper Verifier**:
   - Chained row hashes: `row_hash = SHA256(prev_hash || canonical_payload)` starting from `GenesisHash` (64 hex zeroes).
   - Deterministic canonicalization: Sorts principal roles and argument keys alphabetically, normalizes UTC RFC3339 timestamps, and produces reproducible JSON bytes.
   - `ChainVerifier` sequentially validates workspace audit logs from sequence 1 to N, detecting sequence gaps, duplicates, altered payloads, and broken predecessor links.
5. **Database Immutability & Privilege Separation**:
   - PostgreSQL engine-level trigger `prevent_audit_modification()` blocks all `UPDATE` and `DELETE` queries.
   - Runtime database role (`agentgate_app`) restricted to `SELECT, INSERT` only; schema migration role (`agentgate`) segregated.
6. **Exact Policy Hash Provenance**:
   - Distinguishes `policy_version` (version identifier, e.g. `"v1.0.0"`) from `policy_hash` (hex-encoded SHA-256 of exact evaluated Cedar policy source bytes).
   - Invariant enforced: `audit.policy_hash == SHA256(exact evaluated bytes)` and `audit.policy_hash != audit.policy_version`.
   - Pre-Cedar structural denials leave both fields empty, preserving the frozen G1 invariant that unreached policy provenance is never fabricated.

### Independent Proof
- **Independent QA Security Proof Suite (`agentgate/qa/g5audit/audit_test.go`)**: 9/9 black-box tests pass, verifying durable ALLOW, durable DENY, policy change provenance preservation, secret redaction, tamper detection, audit outage fail-closed behavior, workspace isolation, concurrent writes, and process restart survival.
- **Unit Suite (`agentgate/internal/audit`)**: 19/19 tests pass, including the `TestAuditedDecisionService_ExactPolicyHashProvenance` regression test.
- **DevOps Environment (`deploy/g5/docker-compose.yml`, `init-roles.sql`, `README.md`)**: Fully reproducible Docker Compose topology with PostgreSQL 16 Alpine, automated schema migrations, RBAC privilege separation, and operational verification procedures.
- **Full Regression**: Zero regressions across G1 black-box, G2 identity/tool, G3 governance, G4 integration, and frontend Vitest suites (63/63 passing).

---

## 2. Architecture & Data Flow

The following diagram illustrates the **currently wired G1&rarr;G5 production authorization path** versus the **still-unwired G6 agentgateway/MCP proxy path**:

```mermaid
flowchart TD
    subgraph ClientLayer["Client & Caller Layer"]
        AGENT["Agent / Caller"]
        MCP_CLIENT["MCP Client<br/>(Claude Desktop / Cursor)"]
    end

    subgraph UnwiredGateway["Gate G6 MCP Gateway Boundary (STILL UNWIRED - O-008)"]
        AGW["agentgateway (MCP Proxy)<br/>Reverse Proxy / ext_authz Filter"]
        CALLOUT["ext_authz Callout<br/>(HTTP / gRPC CheckRequest)"]
        AGW -.->|Future Callout<br/>(G6 Scope / O-008)| CALLOUT
    end

    subgraph WiredAgentGate["AgentGate Authorization & Audit Core (G1 - G5 WIRED)"]
        ADS["internal/audit<br/>AuditedDecisionService.Evaluate(req)"]
        REDACT["internal/audit<br/>Redactor.RedactArguments()"]
        
        subgraph GovernanceIntegration["G4 Integration Seam"]
            GDS["internal/governanceintegration<br/>GovernanceDecisionService"]
            MGR["internal/policymanager<br/>Manager (Active Cache)"]
            PROV["ProvenanceProvider<br/>(Version + Byte SHA-256)"]
        end

        subgraph DecisionEngine["G1 Decision Core"]
            DENG["internal/decision<br/>Engine.Evaluate(req)"]
            CEDAR["internal/policy<br/>Cedar Engine"]
        end

        subgraph CryptographicChain["Audit Cryptographic Chaining"]
            LOCK["PostgreSQL Advisory Lock<br/>pg_advisory_xact_lock(ws)"]
            CANON["ComputeCanonicalPayload()<br/>(Sorted Keys/Roles, RFC3339)"]
            HASH["ComputeRowHash()<br/>SHA256(prev_hash || canonical)"]
        end

        subgraph FailClosedBoundary["O-002 Fail-Closed Boundary"]
            FC{"ALLOW & Write Failed?"}
            RET_DENY["Return DENY<br/>(ReasonEvaluationError)"]
            RET_RES["Return Decision<br/>(ALLOW or DENY)"]
        end
    end

    subgraph DurableStorage["PostgreSQL 16 Storage (deploy/g5)"]
        AUDIT_DB[(audit_events Table<br/>002_create_audit_events.sql)]
        TRIGGER["prevent_audit_modification()<br/>Trigger: Blocks UPDATE/DELETE"]
        ROLE["agentgate_app Role<br/>(SELECT, INSERT Only)"]
        AUDIT_DB --- TRIGGER
        AUDIT_DB --- ROLE
    end

    subgraph VerificationTool["Independent Audit Verification"]
        VERIFIER["internal/audit<br/>ChainVerifier.VerifyWorkspace()"]
        VERIFIER -->|Read & Verify Hashes| AUDIT_DB
    end

    %% Unwired connections
    MCP_CLIENT -.->|MCP JSON-RPC Traffic| AGW
    CALLOUT -.->|Undesigned Wire Mapping<br/>(Deferred to G6 / O-008)| ADS

    %% Wired live connections
    AGENT -->|decision.Request| ADS
    ADS --> GDS
    GDS --> MGR
    GDS --> DENG
    DENG --> CEDAR
    CEDAR -->|Decision + Byte SHA-256| DENG
    GDS --> PROV

    ADS --> REDACT
    ADS --> LOCK
    ADS --> CANON
    CANON --> HASH
    HASH -->|Append Decision| AUDIT_DB
    AUDIT_DB -->|Write Success / Failure| FC
    FC -->|Write Failed on ALLOW| RET_DENY
    FC -->|Write Succeeded or DENY| RET_RES
    RET_DENY --> AGENT
    RET_RES --> AGENT

    %% Styling
    classDef unwired fill:#f9f9f9,stroke:#999,stroke-width:2px,stroke-dasharray: 5 5;
    classDef wired fill:#e8f4fd,stroke:#1e88e5,stroke-width:2px;
    classDef storage fill:#f3e5f5,stroke:#8e24aa,stroke-width:2px;

    class UnwiredGateway,AGW,CALLOUT unwired;
    class WiredAgentGate,ADS,REDACT,GovernanceIntegration,GDS,MGR,PROV,DecisionEngine,DENG,CEDAR,CryptographicChain,LOCK,CANON,HASH,FailClosedBoundary,FC,RET_DENY,RET_RES,VerificationTool,VERIFIER wired;
    class DurableStorage,AUDIT_DB,TRIGGER,ROLE storage;
```

---

### Codebase Walkthrough

| Location | Purpose | Gate Role |
|---|---|:---:|
| `agentgate/internal/audit/types.go` | `DecisionRecord`, `StoredRecord`, `GenesisHash`, event types, error definitions | G5 Domain Contracts |
| `agentgate/internal/audit/store.go` | `Store` persistence interface (`AppendDecision`, `AppendMutation`, `GetLatestRecord`, `Verify`) | G5 Storage Interface |
| `agentgate/internal/audit/postgres.go` | PostgreSQL store (`pgx/v5`) with advisory transaction locks and SQL queries | G5 DB Persistence |
| `agentgate/internal/audit/memory.go` | Thread-safe memory store with `SetFault` and `SimulateTamper` hooks | G5 Test Double |
| `agentgate/internal/audit/redact.go` | Pre-persistence argument redactor with full/hash/omit and sensitive key protection | G5 Privacy Boundary |
| `agentgate/internal/audit/chain.go` | Deterministic canonical serialization and cryptographic row chaining | G5 Cryptographic Core |
| `agentgate/internal/audit/verifier.go` | `ChainVerifier` verifying sequence continuity, link integrity, and payload fidelity | G5 Audit Verifier |
| `agentgate/internal/audit/service.go` | `AuditedDecisionService` intercepting decisions, enforcing O-002 fail-closed semantics | G5 Audit Interceptor |
| `agentgate/internal/audit/migrations/002_create_audit_events.sql` | PostgreSQL schema migration creating `audit_events` and immutability trigger | G5 Migration |
| `agentgate/internal/policymanager/manager.go` | Extended with `activeVersions`, `CreateCandidateWithVersion`, and `GetActiveProvenance` | G5 Provenance Source |
| `agentgate/internal/governanceintegration/integration.go` | Forwards `GetActiveProvenance` from manager to decision service | G5 Provenance Seam |
| `agentgate/qa/g5audit/audit_test.go` | Independent black-box QA security proof suite covering 9 core acceptance scenarios | G5 QA / Security |
| `deploy/g5` | Docker Compose topology, RBAC SQL initialization (`init-roles.sql`), operational verification guide | G5 DevOps |

---

### Key Design Decisions & Rationale

1. **Pre-Persistence Redaction Boundary**:
   - Arguments are sanitized *before* computing the canonical JSON payload, before hashing, and before database insertion.
   - *Why*: Prevents raw credentials, synthetic passwords, or Bearer tokens from ever existing in persistent storage, cryptographic digests, or audit logs.
2. **Fail-Closed on ALLOW Audit Outage (O-002 Resolution)**:
   - If policy evaluation returns `ALLOW` but audit persistence fails (database down, disk full, transaction aborted), the service overrides the outcome to `DENY` with reason code `evaluation_error`.
   - *Why*: Enforces the core security invariant that no agent action can proceed without an immutable, durable authorization record.
3. **Workspace-Scoped Advisory Locking**:
   - Uses `pg_advisory_xact_lock(hashtext(workspace_id))` to sequence audit records per workspace.
   - *Why*: Guarantees gap-free, monotonic sequence numbering per workspace without table-level lock contention, enabling parallel horizontal scaling across workspaces.
4. **Exact Evaluated Policy Hash Provenance**:
   - `policy_version` stores the version identifier (e.g. `"v1.0.0"`), while `policy_hash` stores `sha256(evaluated policy source bytes)`.
   - *Why*: Satisfies both human-readable lifecycle tracking and cryptographic policy byte verification, eliminating provenance drift.
5. **Database Engine Immutability Defense in Depth**:
   - Combined PostgreSQL UPDATE/DELETE trigger with restricted runtime user permissions (`agentgate_app` has `SELECT, INSERT` only).
   - *Why*: Protects historical audit evidence from alteration even if application credentials are compromised or an administrative mistake occurs.

---

### Trade-offs & Known Rough Edges

1. **Synchronous Database Persistence Overhead**:
   - Every authorization decision incurs a synchronous PostgreSQL insert. For latency-sensitive paths, write-ahead logging or micro-batched transactional flushing may be considered in future releases.
   - *Trade-off*: Absolute fail-closed audit safety (O-002) was prioritized over zero-latency in-memory returns.
2. **Real MCP Gateway Enforcement is Not Yet Wired (O-008)**:
   - AgentGate's decision and audit core evaluates `decision.Request` structures via Go integration harnesses and HTTP handlers. Network-level reverse proxying and translation of incoming MCP JSON-RPC callouts from `agentgateway` remains unwired.
   - *Resolution*: This is explicitly scoped to **Gate G6**.

---

## 3. Where to Look (Code Reading List)

1. [`agentgate/internal/audit/service.go`](file:///d:/PROJECTS/AgentGate_Hackathon/agentgate-repo/agentgate/internal/audit/service.go): The core audit boundary implementing the O-002 fail-closed evaluation flow and authoritative provenance capture.
2. [`agentgate/internal/audit/chain.go`](file:///d:/PROJECTS/AgentGate_Hackathon/agentgate-repo/agentgate/internal/audit/chain.go): Canonical payload serialization and cryptographic row chaining algorithms.
3. [`agentgate/internal/audit/redact.go`](file:///d:/PROJECTS/AgentGate_Hackathon/agentgate-repo/agentgate/internal/audit/redact.go): Argument redaction engine with defensive overrides for secrets and Bearer tokens.
4. [`agentgate/qa/g5audit/audit_test.go`](file:///d:/PROJECTS/AgentGate_Hackathon/agentgate-repo/agentgate/qa/g5audit/audit_test.go): The independent QA black-box security suite proving all 9 acceptance criteria.

---

## 4. Open Decisions Status

In accordance with `docs/DECISIONS/OPEN_DECISIONS.md`:

- **O-001 (Downstream identity / credential propagation):** Carried forward (deferred to G7).
- **O-002 (Audit durability invariant):** **FORMALLY CLOSED & RESOLVED in G5.** Implemented as fail-closed on `ALLOW` persistence failure and verified by independent QA.
- **O-003 (agentgateway conformance/security boundary):** Carried forward (G6).
- **O-004 (Supported MCP revision boundary):** Carried forward (G6).
- **O-005 (Tool identity and schema fingerprint):** Carried forward.
- **O-006 (Argument authorization model):** Closed in G2.
- **O-007 (AgentGate execution identity):** Carried forward.
- **O-008 (ext_authz transport mapping to decision.Request):** **CENTRAL FOCUS OF G6.** Real network-level proxying and MCP transport mapping belongs to G6; G5 was not reopened to solve it.

---

## 5. Definition of Done Checklist (G5)

- [x] Append-only PostgreSQL audit persistence with sequence monotonic ordering per workspace (`internal/audit/postgres.go`).
- [x] Database-level immutability enforced via `prevent_audit_modification()` trigger (`migrations/002_create_audit_events.sql`).
- [x] Runtime database user privilege separation (`agentgate_app` has `SELECT, INSERT` only).
- [x] Argument redaction engine supporting `full`, `hash`, and `omit` modes (`internal/audit/redact.go`).
- [x] Automatic defensive redaction for sensitive keys and Bearer tokens (`internal/audit/redact.go`).
- [x] Cryptographic row chaining: `row_hash = SHA256(prev_hash || canonical_payload)` (`internal/audit/chain.go`).
- [x] Independent chain verifier detecting gaps, duplicates, altered payloads, and broken links (`internal/audit/verifier.go`).
- [x] O-002 fail-closed semantics implemented on `ALLOW` persistence failure (`internal/audit/service.go`).
- [x] Exact evaluated policy version and policy source byte SHA-256 hash provenance preserved (`internal/audit/service.go`).
- [x] Independent QA security proof suite verifying all 9 core acceptance scenarios (`qa/g5audit/audit_test.go`).
- [x] Reproducible Docker Compose topology with healthchecks and operational verification instructions (`deploy/g5`).
- [x] Zero regressions across G1, G2, G3, G4, and frontend test suites.

---

## 6. Handoff to G6

With Gate G5 formally approved by the Lead Architect and frozen:
- AgentGate possesses an immutable, durable, tamper-evident authorization evidence layer.
- Provenance, privacy redaction, and fail-closed durability are guaranteed.

The next checkpoint is **Gate G6 — agentgateway / MCP Enforcement**:
- Wire `agentgateway` as an authentic reverse proxy in front of real downstream MCP servers.
- Resolve **O-008**: Design and build the translation seam between `agentgateway`'s native `ext_authz` protocol (HTTP/gRPC) and AgentGate's frozen `decision.Request` contract.
- Verify end-to-end enforcement on live MCP tool invocations.
