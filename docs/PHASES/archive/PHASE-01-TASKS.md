# AgentGate — Phase 1 Task Register

**Date:** 2026-08-22  
**Status:** Planned

> **Superseded as the execution sequence (2026-09-09, DAY-01/TASK-01):**
> `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md` is now current. Mapping for continuity:
>
> | Old task | Status | New sequence |
> |---|---|---|
> | TASK-01-01 | Complete | carried forward as-is (Go module/service scaffold) |
> | TASK-01-02 (Cedar decision core) | Superseded | Day 2 — Decision Core Productionization |
> | TASK-01-03 (agentgateway integration) | Superseded | Day 8 — Agentgateway + MCP Production Integration |
> | TASK-01-04 (MCP backend + E2E proof) | Superseded | Day 3 (tool governance) / Day 8 (E2E proof) |
> | TASK-01-05 (hardening + review) | Superseded | Day 11 (failure modes) / Day 14 (security review) |
>
> Do not start any TASK-01-0x task below under this document's numbering; follow the 15-day plan
> instead. This table is retained for historical continuity, not deleted.

| Task | Priority | Owner | Depends on | Result |
|---|---|---|---|---|
| TASK-01-01 | P0 | Backend/Go | Phase 0 | Go module + service scaffold |
| TASK-01-02 | P0 | Backend/Go | 01-01 | Cedar decision core |
| TASK-01-03 | P0 | Integration | 01-01 + decision interface | agentgateway → ext_authz → AgentGate |
| TASK-01-04 | P0 | QA + Integration | 01-03 | MCP backend/client + E2E proof |
| TASK-01-05 | P0 | QA + Architect | 01-02–01-04 | security review + phase exit |

## Execution rule

Do not give one coding agent all five tasks. Execute incrementally and review after each boundary.

TASK-01-01 is the first implementation task.

TASK-01-02 begins only after the module/package structure from TASK-01-01 is reviewed.

TASK-01-03 may proceed once the ext_authz-facing contract is stable enough.

TASK-01-04 proves the real enforcement boundary.

TASK-01-05 is final Phase 1 verification.

Work discovered for downstream identity, tool governance, policy governance, production audit durability, HA, or deployment becomes follow-up work unless it blocks the Phase 1 proof.
