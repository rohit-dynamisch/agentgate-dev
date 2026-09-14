# G5 WS3 — DevOps / Durability Environment

## What
Provide the reproducible PostgreSQL/runtime environment needed to prove G5 durability, permissions, restart recovery, failure behavior, and retention/backup expectations.

## Why
Durability cannot be established by unit tests alone. G5 requires real database persistence and restart/failure exercises.

## How
1. Provide deterministic PostgreSQL setup compatible with current migrations.
2. Support fresh migration, persistent storage, restart, and separate runtime/migration privileges.
3. Inject audit-related configuration through environment/config; no production secrets in code.
4. Document redaction-salt source, audit buffer settings (if any), timeouts and retention settings.
5. Provide a repeatable restart/recovery procedure and test it.
6. Document v1 backup/retention/partition expectations without inventing RPO/RTO.
7. Provide deterministic PostgreSQL connectivity/write failure injection for QA.
8. Expose minimal audit-specific operational signals such as write failures, retries/buffer saturation, and persistence latency if applicable.

Do not turn G5 into final G8 production topology. Do not weaken DB privileges for convenience.

Return DONE / CONTRACT / BLOCKED / RISK / NEXT with exact environment verification.
