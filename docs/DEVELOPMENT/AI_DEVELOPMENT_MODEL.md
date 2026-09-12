# AgentGate — AI Development Model

**Status:** Finalized (checkpoint mechanics added 2026-09-12)
**Date:** 2026-08-22
**Owner:** Lead Software Architect

## 1. Purpose

AgentGate is being developed primarily through AI-assisted engineering.

The development model separates architectural leadership from implementation execution:

- **Lead Software Architect / ChatGPT:** architecture, technical research, external verification, sequencing, project planning, decision management, and cross-team coordination.
- **Senior Software Engineering AI agents:** implementation, refactoring, testing, local verification, and bounded technical execution.
- **Human developers:** operate the coding agents, review results, provide domain context, and own the resulting code.
- **Team lead:** project direction and final human decisions.
- **QA:** independent verification and regression/security testing.

The objective is not maximum code generation. The objective is a correct, understandable, testable, extensible codebase that the team owns.

## 2. Operating loop

The project follows:

**Architecture → Plan → Bounded Task → AI Implementation → Tests → Review → Documentation → Verified Status → Next Task**

Important architectural knowledge must be recorded in the repository rather than remaining only in chat.

## 2a. Checkpoint mechanics (the operating loop, made concrete)

The project executes as a sequence of checkpoints (**G1, G2, ... GN**), each covering one or more
parallel workstreams (Go Backend, Gateway/MCP, Frontend/UI, QA/Security, DevOps). Within a
checkpoint, the loop in §2 runs as:

1. **Plan.** The Lead Architect produces a checkpoint reference (the shared definition-of-done)
   and, per workstream, a detailed task spec plus the literal prompt to hand to a coding agent.
2. **Implement.** The human relays each prompt to a coding agent (often several in parallel, one
   per workstream, on their own branch). The agent implements only its assigned scope, tests it,
   and produces two distinct kinds of output — never blur them:
   - **Durable output** — typed contracts, security evidence, environment references. Committed
     to the repo; this is permanent project knowledge, referenced by later work.
   - **Ephemeral handoff output** — an implementation report and a condensed code digest, written
     for the human to forward to the Lead Architect. Never committed as permanent history, never
     referenced from code or from durable docs. See `docs/README.md` §4 for exactly where each
     kind lives.
3. **Review.** The human forwards the ephemeral handoff output (plus any durable docs the Lead
   Architect asks for) to the Lead Architect, who reviews it — approves, or sends back concrete
   corrections, which the human relays for another implementation pass.
4. **Close.** Once every workstream in the checkpoint is approved, its branches merge, any
   corrective closeout lands, and the coding agent writes one `CLOSURE_SUMMARY.md` for the
   checkpoint (durable, plain-language) before the next checkpoint starts.

A coding agent must never silently redefine another workstream's contract, start a later
checkpoint's work, or resolve an open decision on its own (§4, §5, §8 below still apply in full —
this section only makes the recurring loop's mechanics and artifacts explicit).

## 3. Lead Architect responsibilities

The Lead Architect:

- maintains the architectural direction;
- verifies external technical facts when current information matters;
- identifies security and trust boundaries;
- separates blocking decisions from evolvable implementation details;
- produces implementation plans and bounded tasks;
- coordinates dependencies between workstreams;
- reviews deviations discovered during implementation;
- maintains project status and open decisions;
- prevents scope drift and premature complexity.

The Lead Architect does not require every architectural detail to be solved before development starts.

## 4. Coding-agent responsibilities

A coding agent:

- reads the relevant repository documentation before implementation;
- inspects current code before changing it;
- executes only the assigned task;
- follows approved architecture and decisions;
- writes and updates tests;
- reports deviations and unresolved issues;
- updates required documentation;
- stops when an unresolved architectural/security decision blocks safe implementation.

A coding agent must not silently redesign the system.

## 5. External-information boundary

Coding agents may not have reliable internet access.

When implementation depends on a current external fact, the Lead Architect verifies it using authoritative sources and records the resulting engineering conclusion in the repository.

The coding agent should implement from the verified repository artifact rather than independently guessing the external behavior.

## 6. Task discipline

Every implementation task should define:

- objective;
- context;
- dependencies;
- expected files/components;
- implementation requirements;
- acceptance criteria;
- tests;
- security considerations;
- out-of-scope work;
- documentation/status updates.

Tasks should be small enough to review and verify independently.

## 7. Multi-agent coordination

Multiple agents may work concurrently.

Parallel work should be organized around component boundaries and agreed interfaces.

If two tasks require substantial changes to the same critical component, sequence the work or establish the shared interface first.

Avoid parallel editing of the same files unless explicitly coordinated.

## 8. Change management

New discoveries are classified as:

- BUG
- SECURITY ISSUE
- ARCHITECTURE CHANGE
- REQUIREMENT CHANGE
- TECHNICAL DEBT
- IMPLEMENTATION DETAIL

A discovery is either fixed in the current task, scheduled for a later task, or escalated as an architectural decision.

Do not silently alter requirements or architecture.

## 9. Definition of done

A task is done when its acceptance criteria are satisfied, relevant tests pass, security implications have been considered, required documentation is updated, and no unexplained architectural deviation remains.

A phase is done only when its exit criteria are satisfied.

## 10. Schedule discipline

The current target is:

- Development/deployment deadline: **2026-09-21**
- Working demonstration: **2026-09-22**

Work must be prioritized as:

- **P0:** required for the working demo
- **P1:** required for a credible production-quality foundation
- **P2:** important but deferrable
- **P3:** future

Schedule pressure must not be used to justify unsafe security shortcuts.
