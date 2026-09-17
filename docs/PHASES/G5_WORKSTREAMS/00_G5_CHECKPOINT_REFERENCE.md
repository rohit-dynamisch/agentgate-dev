# AgentGate — G5 Checkpoint Reference: Durable Audit Boundary

**Target:** End of Day 7  
**Streams:** Go Backend + QA/Security + DevOps

## Objective
Turn the G4 audit hook into a durable, provenance-rich, redacted, tamper-evident authorization audit boundary.

G5 does **not** redesign G1 decision contracts, G2 identity/tool governance, G3 policy lifecycle, G4 governance workflow, G6 agentgateway/MCP enforcement, or G7 downstream credentials.

## Source-of-truth requirements
The 10-day plan requires agreement on: required audit fields, redaction rules, policy provenance, correlation ID, `workspace_id`, hash-chain behavior, audit failure semantics, DB permissions, and retention/backup expectations. For both ALLOW and DENY, QA must retrieve a durable record with required provenance; QA must prove sensitive arguments are not logged, policy version/hash is present, tampering is detectable, committed records survive restart, and defined audit-failure behavior occurs.

## Existing technical direction
The technical stack specifies PostgreSQL, one row per decision, policy-version hash, `prev_hash`/`row_hash` hash chaining, no runtime UPDATE/DELETE of history, per-tool `full`/`hash`/`omit` argument redaction with deployment salt for hashing, monthly partitioning/retention, and an asynchronous-write/local-buffer direction. G5 must reconcile that asynchronous direction with the invariant that audit loss cannot silently occur; do not guess.

## Required audit contract
At minimum:
- `workspace_id`
- ALLOW/DENY
- reason/result classification
- trusted principal/agent identity
- on-behalf-of identity where applicable
- required roles/claims representation
- authoritative backend/tool identity
- required tool classification/risk context
- execution/correlation ID
- timestamp
- redacted arguments
- exact policy version and policy hash
- `prev_hash` and `row_hash` plus any chain metadata required for verification

Reuse existing contracts/identifiers wherever possible.

## Redaction
Redaction happens **before persistence/canonical hashing**, never only at presentation. Raw credentials, bearer tokens and secrets must never be persisted. Supported modes are `full`, `hash`, `omit`; undeclared/sensitive material must default to non-disclosing behavior. Hashing uses a deployment-controlled salt.

## Provenance
Every durable record must preserve the exact policy version/hash corresponding to the policy bytes actually evaluated.

## Hash chain
Baseline:
`row_hash = SHA256(prev_hash || canonical(row_without_row_hash))`

Canonical serialization must be deterministic and documented. The chain scope (global vs per-workspace) must be explicit. Runtime credentials must not rewrite/delete history. An independent verifier must detect tampering.

## Audit failure semantics — blocking contract
Before implementation, the team must explicitly define:
- whether ALLOW can return without durable acceptance;
- whether DENY can return without durable acceptance;
- PostgreSQL outage behavior;
- insert failure behavior;
- buffer-full behavior if buffering exists;
- retry/backpressure;
- restart/recovery behavior;
- operator-visible health/metrics.

A process-local queue is not itself durable. The selected behavior must prevent accidental authorization and silent audit loss.

## DB permissions
Runtime application credentials must have only required privileges and must not UPDATE/DELETE historical audit rows. Schema migration privileges are separate.

## Retention/backup
Document partitioning, retention ownership, backup expectation, restore expectation, and chain behavior at retention boundaries. Do not invent RPO/RTO guarantees.

## Shared DoD
1. ALLOW durable.
2. DENY durable.
3. Exact policy version/hash persisted.
4. `workspace_id` persisted and isolated.
5. Sensitive arguments redacted before persistence.
6. Correlation/execution ID persisted.
7. Independent hash verification detects tampering.
8. Runtime DB role cannot UPDATE/DELETE history.
9. Restart preserves already-committed records.
10. Audit failure follows explicit tested fail-safe semantics.
11. Retention/backup behavior documented.
12. Unit/integration/security/failure/concurrency tests pass.
13. No G4 contract silently changes.

## Proof sequence
```text
ALLOW -> durable row -> provenance -> redaction -> hash verifies
DENY  -> durable row -> provenance -> redaction -> hash verifies
tamper -> verifier FAIL
restart -> committed rows remain
audit failure -> defined fail-safe behavior, no silent loss/accidental authorization
runtime UPDATE/DELETE -> rejected
```
