# DAY-01 / TASK-01 — Production Security Invariants & Architecture Freeze

**Owner:** Lead Architect + QA  
**Execution:** Claude Code / Senior Software Engineer  
**Phase:** AgentGate v1 15-Day Production Readiness  
**Status:** Complete — accepted. Its deliverable, `docs/SECURITY/PRODUCTION-INVARIANTS.md`, is a
living binding document; this task spec is kept as the historical record of what was assigned.
The 15-day sequencing this task belonged to was itself superseded after Day 2 by
`docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`.

## Objective

Convert the existing project definition, technology stack, open decisions and 15-day production plan into one explicit security/architecture contract before further implementation.

This task is documentation and architecture-freeze work. Do not implement Day 2+ production functionality.

## Read First

- `CLAUDE.md`
- `docs/PROJECT_DEFINITION.md`
- `docs/TECH_STACK.md`
- `docs/PHASES/archive/MASTER_PLAN.md`
- `docs/DEVELOPMENT/CURRENT_STATUS.md`
- `docs/DECISIONS/OPEN_DECISIONS.md`
- existing Phase 1 planning/task documents
- the current 15-day production-readiness plan

## Required Deliverable

Create/update:

`docs/SECURITY/PRODUCTION-INVARIANTS.md`

It must establish:
- gateway trust boundary;
- identity/claims boundary;
- deny-by-default authorization invariants;
- unknown-tool behavior;
- argument-authorization boundary;
- exact policy version/hash propagation;
- policy validation, activation and reload semantics;
- audit durability and integrity semantics;
- downstream credential/token-passthrough prohibition;
- admin authentication/authorization boundary;
- TLS/mTLS boundaries;
- failure/degraded-mode behavior;
- deployment trust topology;
- backup/recovery expectations;
- explicit v1 scope exclusions.

## Blocking Decisions

For each decision, record:
- decision;
- rationale/security impact;
- implementation consequence;
- experiment/validation still required, if any.

At minimum cover:
1. downstream identity/credential mechanism;
2. audit durability semantics;
3. policy reload/activation;
4. admin authentication;
5. TLS/mTLS;
6. deployment topology;
7. backup/recovery.

The project definition contains an older RFC 8693/token-exchange assumption. The technology plan records that MCP token passthrough is forbidden and identifies ID-JAG as the compliant direction. Do not silently rewrite the history; explicitly record the divergence and the chosen v1 boundary.

## Do Not Implement

Do not add:
- Cedar production logic;
- PostgreSQL migrations;
- admin APIs/UI;
- gateway configuration;
- downstream credential exchange;
- new dependencies;
- speculative interfaces;
- scope outside Day 1.

## Planning Consistency

If older Phase 1 sequencing conflicts with the 15-day plan, do not delete historical material. Make the minimum documentation change required to establish the 15-day plan as the current execution sequence and map/supersede the older sequence.

Update `docs/DECISIONS/OPEN_DECISIONS.md` only where Day 1 genuinely resolves an item.

Update `docs/DEVELOPMENT/CURRENT_STATUS.md` only after this task is actually accepted.

## QA Review

Check for:
- contradictory rules;
- authorization bypass paths;
- ambiguous trust boundaries;
- failure modes that can become ALLOW;
- missing audit semantics;
- missing policy-version correlation;
- unsafe credential forwarding;
- admin-control bypasses.

## Verification and Report

Inspect the final diff and run available checks. Report:
- files changed;
- decisions made;
- unresolved items;
- checks/tests;
- limitations.

## Exit Criteria

Day 1 is complete only when:
1. `docs/SECURITY/PRODUCTION-INVARIANTS.md` exists and is internally consistent.
2. Blocking security decisions are explicit.
3. No unresolved security ambiguity is disguised as implementation detail.
4. The 15-day plan is the single current execution sequence.
5. QA has reviewed the invariants.
6. Lead Architect has accepted the document.

Stop after Day 1. Do not begin Day 2 without explicit acceptance.
