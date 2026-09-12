# AgentGate — Documentation Map

Start here whenever you're not sure where something lives or what's currently true. This page
doesn't duplicate content — it says what exists, where, and why, so nothing has to be
rediscovered by scrolling through chat history.

**Right now:** `docs/DEVELOPMENT/CURRENT_STATUS.md` — one page, always current. If you only read
one thing, read that.

**How we work:** `docs/AI/AI_DEVELOPMENT_MODEL.md` — roles, the ChatGPT↔Claude↔you loop, and the
folder conventions this map describes.

## The five kinds of document in this repo

### 1. Core project context — rarely changes, read once

The permanent "what and why," written to be read cold by a new engineer or a new AI session with
no other context required.

| Doc | Covers |
|---|---|
| `docs/PROJECT_DEFINITION.md` | The single source of truth: product scope, architecture, what we explicitly aren't building, the decisions log (§12) |
| `docs/TECH_STACK.md` | Technology choices and rationale, verified external facts |
| `docs/AI/AI_DEVELOPMENT_MODEL.md` | The development workflow itself — see below |
| `CLAUDE.md` (repo root) | Coding-agent operating instructions |

Change these only when something actually, factually changes about the product/stack — not for
routine progress.

### 2. Living status and decisions — always current, no history to scroll

| Doc | Covers |
|---|---|
| `docs/DEVELOPMENT/CURRENT_STATUS.md` | Where the project is *right now*: current checkpoint, what's done, what's next. Rewritten in place, not appended to. |
| `docs/DECISIONS/OPEN_DECISIONS.md` | Unresolved architectural questions (O-001, O-002...). Resolved items move to that file's own "Resolved" section — never deleted, never left stale. |
| `docs/SECURITY/PRODUCTION-INVARIANTS.md` | The binding security contract. No later work may contradict it without an explicit, recorded exception. |

### 3. Strategy — exactly one active plan at a time

There have been several planning strategies as the project's understanding matured (a phase-based
plan → a 15-day sequential plan → the current 10-day parallel-checkpoint plan). Old strategies are
**never deleted** — deleting them would erase the reasoning trail — but each carries an explicit
"superseded by X" banner at the top the moment it stops being current, so nobody executes a dead
plan by accident.

**The active plan is always the one with no "superseded" banner on it.** As of 2026-09-12, that's:

`docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md` — parallel workstreams, gated by
checkpoints (G1, G2, ... GN).

Superseded plans (kept for history, each self-explaining why it stopped applying):
`docs/DEVELOPMENT/MASTER_PLAN.md`, `docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md`,
`docs/PHASES/archive/`.

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
│                                      human-readable "what shipped" record (see §5)
└── results/                          GITIGNORED — ephemeral reports + digests for the
                                       human↔ChatGPT review loop; never permanent, never
                                       referenced from code or from durable docs
```

Raw prompts (the literal text fed to Claude, usually written by ChatGPT) live in
`docs/prompts/G{N}/` — also gitignored; they're working material, not project history.

### 5. Checkpoint closure summaries — the durable "what happened" record

At the end of each checkpoint, once its branches merge, one `CLOSURE_SUMMARY.md` is written into
that checkpoint's folder: a short, plain-language recap of what shipped, what was decided, and
what's carried forward — the thing you'd hand someone to explain the checkpoint without making
them read every report and diff. This is the only new artifact this workflow adds beyond what
already existed for G1; everything else above was already true, just not written down in one
place.

## Quick answers

- **"What are we building and why?"** → `docs/PROJECT_DEFINITION.md`
- **"What's the state of things today?"** → `docs/DEVELOPMENT/CURRENT_STATUS.md`
- **"What plan are we actually following?"** → the strategy doc with no superseded banner (§3
  above names it)
- **"What happened at G1 / G2 / ...?"** → that checkpoint's `CLOSURE_SUMMARY.md`, or the full
  detail in its `docs/PHASES/G{N}_WORKSTREAMS/` folder
- **"Is there an open question blocking this?"** → `docs/DECISIONS/OPEN_DECISIONS.md`
- **"What security invariant governs this?"** → `docs/SECURITY/PRODUCTION-INVARIANTS.md`
- **"How do I set up and run this locally?"** → `docs/DEVELOPMENT/SETUP.md`
- **"What does CI check?"** → `docs/DEVELOPMENT/CI_BASELINE.md`
- **"What's left before this can be public/OSS?"** → `docs/DEVELOPMENT/OSS_READINESS.md`
