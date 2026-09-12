---
name: propagate-a-fix
description: Use when a review finding (from the Lead Architect, QA, or code review) requires the same fix across multiple checkpoint workstream branches
---

# Propagate a Fix

## Overview
A corrective-closeout pattern for when one real bug affects several workstream branches that
share a contract (e.g. a decision-core bug found by QA also affects Gateway's and Frontend's
fixtures). Fix once, propagate deliberately — don't re-derive the fix per branch and don't
silently expand scope.

## Steps

1. **Fix on the canonical branch first** — the branch that owns the affected code (e.g. the
   Go-backend branch owns `internal/decision`). Add a regression test proving the old behavior
   was wrong, not just that the new behavior is right.

2. **Identify every dependent branch** — any branch whose tests, fixtures, or docs assert the
   old (now-wrong) behavior. Check by searching for the specific behavior, not by assumption.

3. **Merge the canonical fix into each dependent branch.**

4. **Update only the tests that assert the old behavior** — not unrelated cleanup on that branch.
   If you find something else wrong while there, record it (per CLAUDE.md scope discipline)
   rather than fixing it inline.

5. **Re-verify each branch** after merge (full module test suite, not just the changed test).

6. **Disclose, don't hide, anything the review instructions didn't explicitly name** — if the fix
   mechanically implies a third affected test the reviewer didn't call out, say so rather than
   silently leaving it broken or silently expanding the fix further than asked.

## Common Mistakes
- Patching each branch independently — produces subtly different fixes and no single source of
  truth for what the correct behavior is.
- Treating "the Lead named 3 items" as a hard ceiling when a 4th is mechanically the same bug —
  report the 4th, don't silently fix or silently ignore it.
- Using the propagation as an excuse for a broader cleanup pass on branches you're touching anyway.
