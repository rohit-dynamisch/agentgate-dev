# AgentGate — Phase 1 Team Ownership

**Date:** 2026-08-22  
**Status:** Proposed

This is an ownership model for AI-assisted work.

## Team Lead

Owns project direction, final human decisions, scope escalation, and major security/architecture approvals. Does not need to code.

## Backend / Node.js Developer — Go implementation owner

Owns TASK-01-01 and TASK-01-02: Go module, service scaffold, decision core, and associated tests.

Use the coding agent heavily, but keep changes bounded by the task specifications.

## AI/ML Software Engineer — Integration owner

Owns TASK-01-03: agentgateway configuration, ext_authz integration, integration experiments, and documenting verified gateway behavior.

Current external technical facts are escalated to the Lead Architect.

## QA

Owns independent verification for TASK-01-04 and TASK-01-05: negative/security tests, end-to-end acceptance, regression checks, and evidence that denied requests do not reach the backend.

QA should challenge the implementation rather than reproduce only the happy path.

## Frontend / React Native Developer

No Phase 1 production frontend dependency.

Do not create an operator UI merely to keep this role occupied. Support only where it does not distract from the P0 enforcement proof; future governance UI belongs to a later phase.

## Coordination rules

- One owner for the Go module structure.
- Shared interfaces are established before parallel implementation.
- Avoid concurrent edits to the same critical files.
- Every developer reports files changed, tests run, results, deviations, and blockers.
- Repository Markdown is the coordination source of truth.
