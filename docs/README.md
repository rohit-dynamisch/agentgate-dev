# AgentGate — Documentation Map

Start here whenever you're not sure where something lives or what's currently true. This page
doesn't duplicate content — it says what exists, where, and why, so nothing has to be
rediscovered by scrolling through chat history.

**Right now:** `docs/DEVELOPMENT/CURRENT_STATUS.md` — one page, always current. If you only read
one thing, read that.

**How we work:** `docs/DEVELOPMENT/AI_DEVELOPMENT_MODEL.md` — roles, the ChatGPT↔Claude↔you loop,
and the folder conventions this map describes.

## The five kinds of document in this repo

### 1. Core project context — rarely changes, read once

The permanent "what and why," written to be read cold by a new engineer or a new AI session with
no other context required. Change these only when something actually, factually changes about
the product/stack — not for routine progress.

### 2. Living status and decisions — always current, no history to scroll

Rewritten in place as things change. Never append a history log to these — that belongs in a
checkpoint's own docs (kind 4/5 below).

### 3. Strategy — exactly one active plan at a time

There have been several planning strategies as the project's understanding matured. Old
strategies are **never deleted** — deleting them would erase the reasoning trail — but each
carries an explicit "superseded by X" banner at the top the moment it stops being current, so
nobody executes a dead plan by accident. **The active plan is always the one with no "superseded"
banner on it.**

### 4. Checkpoints — one self-contained folder per checkpoint, same shape every time

Each checkpoint (G1, G2, ...) gets `docs/PHASES/G{N}_WORKSTREAMS/`, always structured the same
way so there's nothing new to learn at G2, G3...:

```
docs/PHASES/G{N}_WORKSTREAMS/
├── 00_G{N}_CHECKPOINT_REFERENCE.md   the shared definition-of-done for this checkpoint
├── 0X_<WORKSTREAM>_DETAILED.md       per-workstream task tickets (what Claude executes)
├── <CONTRACT_OR_EVIDENCE_DOCS>.md    durable output: frozen contracts, security evidence,
│                                      environment references — committed, permanent
├── CLOSURE_SUMMARY.md                written once the checkpoint merges — the durable,
│                                      human-readable "what shipped" record (see kind 5)
└── results/                          GITIGNORED — ephemeral reports + digests for the
                                       human↔ChatGPT review loop; never permanent, never
                                       referenced from code or from durable docs
```

Raw prompts (the literal text fed to Claude, usually written by ChatGPT) live in
`docs/prompts/G{N}/` — also gitignored; they're working material, not project history.

### 5. Checkpoint closure summaries — the durable "what happened" record

At the end of each checkpoint, once its branches merge, one `CLOSURE_SUMMARY.md` is written into
that checkpoint's folder: a short, plain-language recap of what shipped, what was decided, and
what's carried forward. (Not yet written for G1 — added the first time a future checkpoint
closes under this convention.)

### 6. Archive — superseded or point-in-time, kept for history, never "current" again

`docs/PHASES/archive/` — one flat folder, no sub-structure, for anything that has permanently
stopped describing the present: dead strategies once their "superseded" banner is added, and
one-time snapshot reports (like a repository recon done before any code existed). Nothing here
is deleted; everything here is annotated with why it's here and what replaced it.

## Full file index

```
docs/
├── README.md                                    you are here — the map
├── PROJECT_DEFINITION.md                        [core] the single source of truth: product
│                                                 scope, architecture, what we explicitly aren't
│                                                 building, the decisions log (§12)
├── TECH_STACK.md                                [core] technology choices + rationale; §4's own
│                                                 "O-N" open-items table is historical — see its
│                                                 2026-09-13 status note for what's resolved vs.
│                                                 superseded by OPEN_DECISIONS.md
│
├── DECISIONS/
│   └── OPEN_DECISIONS.md                        [living] the canonical unresolved-question
│                                                 registry (O-001...O-008); resolved items move to
│                                                 its own "Resolved" section, never deleted
│
├── SECURITY/
│   └── PRODUCTION-INVARIANTS.md                 [living, binding] the security contract. No
│                                                 later work may contradict it without an
│                                                 explicit, recorded exception
│
├── DEVELOPMENT/
│   ├── CURRENT_STATUS.md                        [living] where the project is *right now* —
│                                                 rewritten in place, not appended to
│   ├── AI_DEVELOPMENT_MODEL.md                  [core] roles (Lead Architect/ChatGPT, coding
│                                                 agent/Claude, human coordinator), the checkpoint
│                                                 operating loop (§2a), task discipline, change
│                                                 management
│   ├── SETUP.md                                 [living reference] local build/run/test guide
│                                                 for the agentgate/ Go module
│   ├── CI_BASELINE.md                           [living reference] what CI checks and why
│   └── OSS_READINESS.md                         [living backlog] what's needed before public/OSS
│                                                 launch (license decision, community-health
│                                                 files) — deliberately deferred, tracked here so
│                                                 it isn't lost
│
├── PHASES/
│   ├── AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md
│   │                                             [ACTIVE STRATEGY] parallel workstreams gated by
│   │                                             checkpoints (G1, G2, ... GN) — the plan actually
│   │                                             in force right now
│   ├── AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md   [superseded, kept] sequential 15-day plan;
│   │                                             superseded by the parallel plan above after its
│   │                                             own Day 1/Day 2 completed
│   ├── DAY-01-TASK-01.md                        [complete, kept] the Day-1 task spec that
│   │                                             produced PRODUCTION-INVARIANTS.md — historical
│   │                                             record of what was assigned, still cited by
│   │                                             living docs
│   ├── DAY-02-TASK-02.md                        [complete, kept] the Day-2 task spec that
│   │                                             produced the decision core now frozen as the G1
│   │                                             contract
│   │
│   ├── G1_WORKSTREAMS/                          [CLOSED CHECKPOINT — G1: PASS/CLOSED/FROZEN,
│   │   │                                         2026-09-12] durable record of the first
│   │   │                                         checkpoint; see docs/README.md kind 4 for the
│   │   │                                         standard shape every checkpoint folder follows
│   │   ├── 00_G1_CHECKPOINT_REFERENCE.md        the shared definition-of-done all 5 workstreams
│   │   │                                         were held to
│   │   ├── 01_GO_BACKEND_G1_DETAILED.md         per-workstream task tickets (as assigned)
│   │   ├── 02_GATEWAY_MCP_G1_DETAILED.md
│   │   ├── 03_FRONTEND_UI_G1_DETAILED.md
│   │   ├── 04_QA_SECURITY_G1_DETAILED.md
│   │   ├── 05_DEVOPS_G1_DETAILED.md
│   │   ├── GO_BACKEND_G1_CONTRACT.md            [durable, frozen] the canonical
│   │                                             decision.Request/Result wire contract — the
│   │                                             single most-referenced doc from G1 onward
│   │   ├── QA_SECURITY_G1_EVIDENCE.md           [durable] independent black-box test evidence +
│   │                                             security findings (incl. corrective-closeout
│   │                                             addendum)
│   │   ├── DEVOPS_G1_ENVIRONMENT.md             [durable] reproducible environment/topology
│   │                                             reference, incl. real-container verification
│   │   └── results/                             GITIGNORED — ephemeral reports + condensed code
│   │                                             digests (one pair per workstream) used only for
│   │                                             the human↔ChatGPT review loop
│   │
│   └── archive/                                 [historical — see docs/README.md kind 6]
│       ├── MASTER_PLAN.md                       original Phase 0-7 plan; superseded after Phase 0
│       ├── PHASE-01-MINIMAL-E2E-ENFORCEMENT.md  2nd-generation Phase-1 plan; superseded
│       ├── PHASE-01-TASKS.md                    its task register; superseded
│       ├── TASK-01-01-GO-MODULE-AND-REPOSITORY-SCAFFOLD.md
│       │                                         completed task spec, kept for history
│       └── REPOSITORY_BASELINE.md               one-time repo recon snapshot (2026-08-22,
│                                                 before any code existed) — permanently
│                                                 historical, never "current" again
│
└── prompts/                                     GITIGNORED — raw prompt text fed to Claude,
    ├── phase0/                                  working material, not project history
    │   ├── 01_repository_bootstrap_prompt.md
    │   ├── 02_ci_baseline_prompt.md
    │   └── 03_task-01-01_prompt.md
    └── g1/
        ├── 01_GO_BACKEND_G1_PROMPT.md
        ├── 02_GATEWAY_MCP_G1_PROMPT.md
        ├── 03_FRONTEND_UI_G1_PROMPT.md
        ├── 04_QA_SECURITY_G1_PROMPT.md
        └── 05_DEVOPS_G1_PROMPT.md
```

## Quick answers

- **"What are we building and why?"** → `docs/PROJECT_DEFINITION.md`
- **"What's the state of things today?"** → `docs/DEVELOPMENT/CURRENT_STATUS.md`
- **"What plan are we actually following?"** → the strategy doc with no superseded banner —
  currently `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`
- **"What happened at G1 / G2 / ...?"** → that checkpoint's `CLOSURE_SUMMARY.md` (once written),
  or the full detail in its `docs/PHASES/G{N}_WORKSTREAMS/` folder
- **"What's the frozen authorization contract?"** → `docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md`
- **"Is there an open question blocking this?"** → `docs/DECISIONS/OPEN_DECISIONS.md`
- **"What security invariant governs this?"** → `docs/SECURITY/PRODUCTION-INVARIANTS.md`
- **"How do I set up and run this locally?"** → `docs/DEVELOPMENT/SETUP.md`
- **"What does CI check?"** → `docs/DEVELOPMENT/CI_BASELINE.md`
- **"What's left before this can be public/OSS?"** → `docs/DEVELOPMENT/OSS_READINESS.md`
- **"Is this document still current, or old?"** → if it's under `docs/PHASES/archive/`, or it has
  a "superseded by X" banner at the top, it's history, not instruction. Everything else is
  current as of its own last-edit date.
