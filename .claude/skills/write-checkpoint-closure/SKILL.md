---
name: write-checkpoint-closure
description: Use when a checkpoint (G1, G2, ...) has merged and closed, before considering that checkpoint fully done — produces the CLOSURE_SUMMARY.md required by /WORKFLOW.md
---

# Write Checkpoint Closure

## Overview
Every closed checkpoint gets exactly one `docs/PHASES/G{N}_WORKSTREAMS/CLOSURE_SUMMARY.md`, so a
human reviewer never has to reconstruct what AI-written code does from chat history. Required by
`/WORKFLOW.md` §4. Write it even if closure happens retroactively — same bar, not relaxed.

## Required sections (in order)

1. **What shipped** — plain-language paragraph: what the system can now do that it couldn't
   before, and what independently proves it (tests, other workstreams' integration).
2. **Codebase walkthrough**
   - At least one diagram (Mermaid `flowchart` renders natively on GitHub) showing how the new
     pieces connect. Show what's real vs. not-yet-wired (dotted lines) rather than hiding gaps.
   - A table: location → purpose, for every new package/directory.
   - Key design decisions and why (not just what).
   - Trade-offs and known rough edges — say what wasn't proven, not just what was.
3. **Where to look** — an ordered reading list of the 2-4 files that best explain the contract,
   for someone who wants to go read code next.
4. **Decisions and carried-forward items** — what was resolved during the checkpoint (with fix
   commit references) vs. what's deliberately left open, cross-referencing
   `docs/DECISIONS/OPEN_DECISIONS.md` entry IDs.

## Rules
- Format is flexible (Mermaid, tables, prose) but every section above is required — substance
  over ritual, per `/WORKFLOW.md` §4.
- Diagrams show reality, including what's stubbed/not-yet-wired — never omit a gap to make the
  picture look cleaner.
- This file is committed and permanent (unlike `results/`, which is gitignored).
- Look at `docs/PHASES/G1_WORKSTREAMS/CLOSURE_SUMMARY.md` as the reference example of shape and
  depth before writing a new one.

## Common Mistakes
- Treating this as a changelog/diff list instead of a walkthrough a non-implementer can follow.
- Skipping the diagram because "the table covers it" — the diagram is required, not optional.
- Leaving open items undocumented instead of naming them and pointing at `OPEN_DECISIONS.md`.
