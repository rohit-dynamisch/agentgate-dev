---
name: write-handoff-package
description: Use when work needs to go back to the Lead Architect (ChatGPT) for review — produces the report+digest pair used for the human-to-ChatGPT review loop
---

# Write Handoff Package

## Overview
The human relays work between Claude (implementer) and ChatGPT (Lead Architect), who has no
repo access. This skill produces the pair of files ChatGPT actually needs to review a
workstream/task, saved where they won't pollute permanent docs.

## Steps

1. **Location:** the current checkpoint's gitignored `results/` folder, e.g.
   `docs/PHASES/G{N}_WORKSTREAMS/results/`. Never commit these, never reference them from durable
   docs or code.

2. **Write the report** (`0X_<WORKSTREAM>_REPORT.md` or similar): what was implemented, test
   results, files changed, any deviations from the assigned task ticket, open questions for the
   Lead. This is the full detail — assume the reader wants to verify, not just skim.

3. **Write the digest** (`0X_<WORKSTREAM>_DIGEST.md`): a condensed version — a code digest ChatGPT
   can review without pulling the whole repo. Include the key changed files' actual content or
   tight excerpts, not just a file list, since ChatGPT can't open the repo itself.

4. **Keep the pair scoped to one workstream/task** — don't bundle unrelated work into one
   package; the Lead reviews one thing at a time.

## Common Mistakes
- Saving these outside `results/` (they'd get committed, or worse, treated as durable docs).
- Writing a digest that's just a list of filenames — ChatGPT needs actual content to review.
- Omitting deviations/open questions — the whole point of the loop is catching problems before
  they're assumed resolved.
