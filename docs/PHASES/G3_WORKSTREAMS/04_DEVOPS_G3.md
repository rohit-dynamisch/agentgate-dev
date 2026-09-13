# G3-W4 — DevOps/Release

## What
Make PostgreSQL policy persistence reproducible:
- integration DB environment
- deterministic migrations
- configuration injection
- health/readiness dependency behavior
- CI integration checks where supported
- clean startup/recreation

## Why
G3 is not complete if persistence relies on hidden local state. The 10-day plan explicitly requires clean-environment migration reproduction.

## How
Reuse existing topology/conventions. Start from an empty DB, run migrations, verify schema, verify repeat behavior, keep secrets out of source, avoid insecure defaults, and document startup/health expectations.

Test empty DB → migration → usable AgentGate; repeat migration; clean checkout startup; invalid DB configuration; DB unavailable/false-health behavior.

Do not implement G5/G6/G7 scope.
