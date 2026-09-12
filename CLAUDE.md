# AgentGate — Coding Agent Instructions

## Purpose

This repository is being developed with AI-assisted development. Claude Code and other coding agents act as senior software engineers executing bounded tasks defined by the project planning and architecture documents.

The repository documentation is the durable project context. Do not rely on chat history when the required information can be represented in the repository.

## Before changing code

1. Read the relevant project documentation.
2. Identify the current phase and assigned task.
3. Inspect the existing implementation before designing changes.
4. Follow approved architecture and recorded decisions.
5. Identify assumptions and dependencies before implementation.

## Do not

- Invent requirements.
- Silently change architecture.
- Reintroduce rejected designs.
- Add dependencies without justification.
- Modify unrelated components.
- Treat POC behavior as authoritative when production architecture differs.
- Forward credentials merely because it is convenient.
- Weaken authorization or fail-open behavior to make tests pass.

If an implementation requires an unresolved architectural or security decision, stop at that boundary and report it rather than silently deciding.

## Implementation requirements

Every completed task must include, where applicable:

- implementation
- automated tests
- relevant integration/security tests
- documentation updates
- status update
- review of the final diff

A task is not complete merely because the code compiles.

## Security posture

AgentGate is an in-path authorization and governance system.

Default security behavior is fail-closed unless an explicitly approved requirement states otherwise.

For security-sensitive changes, identify:

- trusted inputs
- attacker-controlled inputs
- validation
- authorization decision
- credential boundary
- failure behavior
- audit behavior

Do not assume that tool metadata, tool annotations, request arguments, or backend data are trusted security authority without an explicit trust model.

## Scope discipline

Work only on the assigned task.

If you discover another issue:

1. determine whether it blocks the current task;
2. if it is blocking, report it;
3. otherwise record it for later work rather than expanding the current task.

## Verification

Run the tests and validation required by the task.

Report:

- what changed
- what was tested
- test results
- files changed
- deviations from the task
- unresolved issues

## Project documents

The canonical project documents are under `docs/`. Start with **`docs/README.md`** — the
documentation map — which points to everything below and says which strategy/plan document is
currently active (that changes over time; the map doesn't).

Read directly, in order:

- `docs/PROJECT_DEFINITION.md`
- `docs/TECH_STACK.md`
- `docs/DEVELOPMENT/CURRENT_STATUS.md`
- `docs/DECISIONS/OPEN_DECISIONS.md`
- `docs/SECURITY/PRODUCTION-INVARIANTS.md` — binding; no later work may contradict it without an
  explicit, recorded exception
- `docs/AI/AI_DEVELOPMENT_MODEL.md`

Do not create competing architecture documents without an explicit reason.
