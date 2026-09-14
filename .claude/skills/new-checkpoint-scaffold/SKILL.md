---
name: new-checkpoint-scaffold
description: Use when starting a new checkpoint (G2, G3, ...) — creates the standard folder shape documented in docs/README.md so nothing new has to be learned each checkpoint
---

# New Checkpoint Scaffold

## Overview
Every checkpoint folder has the same shape (`docs/README.md` kind 4). Creating it consistently
means G2, G3... need zero new conventions, and closure/handoff skills know where to write.

## Steps

Create `docs/PHASES/G{N}_WORKSTREAMS/` containing:

```
docs/PHASES/G{N}_WORKSTREAMS/
├── 00_G{N}_CHECKPOINT_REFERENCE.md   shared definition-of-done for this checkpoint (placeholder
│                                      until the Lead Architect's plan fills it in)
├── 0X_<WORKSTREAM>_DETAILED.md       one per active workstream — per-workstream task tickets
├── results/                          create empty, then add to .gitignore if not already
│                                      covered by a pattern
```

Also create `docs/prompts/G{N}/` (gitignored) for the raw prompts fed to Claude for this
checkpoint, mirroring `docs/prompts/g1/`'s naming (`0X_<WORKSTREAM>_PROMPT.md`).

Then:
1. Add the new checkpoint's entry to `docs/README.md`'s file index and "Quick answers" section
   once it exists (even as `[in progress]`).
2. Confirm `results/` and `docs/prompts/G{N}/` are actually gitignored — check `.gitignore`, don't
   assume the existing patterns already cover the new path.
3. Leave `<CONTRACT_OR_EVIDENCE_DOCS>.md` and `CLOSURE_SUMMARY.md` for later — those are written
   as durable output/at close (see write-checkpoint-closure), not scaffolded empty.

## Common Mistakes
- Scaffolding `CLOSURE_SUMMARY.md` as an empty file up front — it should not exist until the
  checkpoint actually closes.
- Forgetting to verify `results/`/`prompts/G{N}/` are gitignored, leaking ephemeral files into a
  commit.
- Copying G1's actual workstream ticket content instead of the shape — content is written per
  checkpoint, not copy-pasted.
