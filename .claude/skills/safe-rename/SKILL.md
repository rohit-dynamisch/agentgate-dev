---
name: safe-rename
description: Use before moving, renaming, or deleting any tracked documentation or code file that other files might reference
---

# Safe Rename

## Overview
AgentGate docs cross-reference each other by path (`docs/README.md`, `CLAUDE.md`, `/WORKFLOW.md`
all point at specific files). Moving a file without fixing its references breaks navigation
silently — nobody notices until someone follows a dead link. This repo's convention is
"supersede, never delete": prefer a redirect stub over breaking a path outright.

## Steps

1. **Grep all references first** — search the whole repo (not just docs/) for the file's path,
   both as a full path and as its bare filename (references vary in how they write it).
2. **Fix every reference found** to point at the new location, before or as part of the move.
3. **Move the file** (`git mv`, so history follows it).
4. **If the old path is likely to be referenced from outside this repo or from memory** (e.g. it
   was in CLAUDE.md's canonical reading list), leave a redirect stub at the old path pointing to
   the new one instead of just deleting — see `docs/DEVELOPMENT/AI_DEVELOPMENT_MODEL.md` for the
   pattern.
5. **Verify build/tests still pass** if the moved file is code, not just docs.
6. **Re-grep** the old path after the move — it should only appear in stubs/archive
   "superseded by" banners, nowhere else.

## Common Mistakes
- Deleting a doc outright instead of leaving a superseded-banner or redirect stub — this repo's
  policy is supersede, never delete (see `docs/README.md` kind 3 and kind 6).
- Renaming and fixing references in two separate, unreviewed steps — a partial multi-path
  operation (e.g. `git add`/grep-replace across many files) can silently skip files that error
  out; verify the full set actually changed, don't assume a batch command succeeded everywhere.
- Missing references in non-doc files (Go doc comments, README files in `gateway/`/`frontend/`)
  because the grep was scoped to `docs/` only.
