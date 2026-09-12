---
name: close-out-a-task
description: Use when finishing any AgentGate implementation task, before declaring it done or committing — verifies the touched module(s), checks diff scope, and commits per repo convention
---

# Close Out a Task

## Overview
AgentGate tasks aren't done because the code compiles (CLAUDE.md). This skill is the
fixed checklist run at the end of every task, before saying "done."

## Steps

1. **Verify only what you touched.** Pick the module(s) that changed:
   - `agentgate/` (Go): `cd agentgate && gofmt -l . && go vet ./... && go build ./... && go test ./...`
   - `gateway/harness` (Go, separate module): same four commands from `gateway/harness/`
   - `frontend/` (npm): `npm run build` / `npm test` from `frontend/`
   - Docker/deploy configs: validate against the real binary/daemon, not just parse — see
     `docs/DEVELOPMENT/CI_BASELINE.md` for what CI actually runs.
   Never skip a module because "it probably still passes."

2. **Check the diff for scope creep.** `git diff` (or `git status` + per-file diff) and confirm
   every changed file is explained by the task. Unrelated formatting/refactor changes get
   reverted or called out explicitly — CLAUDE.md: "Modify unrelated components" is a Do Not.

3. **Confirm the tree is otherwise clean** — no stray debug output, no accidental
   `package-lock.json`/lockfile churn from a validation step, no leftover scratch files outside
   the scratchpad.

4. **Commit** with a message describing what changed and why (not "fix stuff"), ending with:
   ```
   Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
   ```

5. **Report** per CLAUDE.md §Verification: what changed, what was tested, test results, files
   changed, deviations, unresolved issues.

## Common Mistakes
- Running `gofmt -l .` from the repo root instead of `agentgate/` — always `[no test files]` or
  false positives (see the CRLF caveat in `docs/DEVELOPMENT/SETUP.md` §7).
- Treating a green build as sufficient when the task also required doc updates or tests
  (CLAUDE.md's "Implementation requirements" list is not optional per-item).
- Committing without checking `git diff` first, missing an incidental change (e.g. a validation
  step's side effect) that shouldn't ship.
