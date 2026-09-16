# G5 WS1 — Go Backend / Durable Audit

## What
Implement PostgreSQL-backed durable decision audit, redaction, provenance, hash chaining/verification, DB privilege boundaries, decision-path integration, and failure/recovery semantics.

## Why
G4 proves governance-to-decision integration and mutation hooks. V1 requires durable, provenance-rich, tamper-evident evidence for authorization decisions without turning audit into a secret/PII store.

## How
1. Inspect and reuse G1 Request/Result, G2 identity/tool context, G3 policy store, and G4 `GovernanceDecisionService`/audit hook.
2. Freeze a typed audit contract before implementation.
3. Add migration/schema supporting workspace, outcome, reason, identity, tool context, correlation/execution ID, timestamp, redacted arguments, policy version/hash, `prev_hash`, `row_hash`.
4. Redact before canonical serialization. Never persist raw bearer tokens/credentials.
5. Implement `full`/`hash`/`omit` according to declared tool policy; salted hashing uses deployment-controlled salt.
6. Implement deterministic canonicalization and SHA-256 chaining; provide an independent verifier.
7. Integrate both ALLOW and DENY without duplicate policy evaluation.
8. Explicitly resolve audit-failure semantics before coding the failure path. If async buffering is used, define the actual durability point; a process-local queue alone is insufficient.
9. Separate migration privileges from runtime privileges; runtime cannot UPDATE/DELETE history.
10. Provide the minimum retrieval/verification path needed by QA/operations; do not expand public APIs unnecessarily.
11. Test persistence, provenance, redaction, tampering, workspace isolation, privilege denial, restart, dependency failure, concurrency, and G1-G4 regressions.

Do not implement G6 MCP E2E, G7 credentials, SpiceDB, response governance, or unrelated UI/API redesign.

If a shared contract must change, stop that decision-dependent work and escalate; do not silently redesign.

Return DONE / CONTRACT / BLOCKED / RISK / NEXT plus exact tests and a code digest.
