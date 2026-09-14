# G4-W3 — QA/Security

## What
Independently prove the complete governance lifecycle and its security properties through public HTTP/service boundaries.

## Why
G4 must demonstrate that persisted policy changes actually affect authorization and that dry-run/rollback cannot corrupt or bypass the decision boundary.

## Core scenario
Policy A allows tool X. Candidate B changes the rule.
1. Confirm A active and evaluate X.
2. Validate B.
3. Dry-run B and verify predicted B outcome.
4. Confirm A remains active and live outcome remains A.
5. Activate B.
6. Verify live outcome is B and provenance is B.
7. Roll back to A.
8. Verify live outcome and provenance return to A.

## Negative/security cases
- invalid candidate cannot affect live authorization
- dry-run cannot mutate active policy
- failed activation cannot partially change decisions
- unknown rollback cannot alter behavior
- workspace A cannot affect B
- stale cache cannot authorize using superseded policy
- load/evaluation failure denies
- unauthorized mutation denied
- no credential leakage
- mutation event correlates workspace/version/action

Run all G1/G2/G3 regressions.

## Blocking findings
Any persisted-active vs evaluated-policy mismatch, fail-open path, or ambiguous mutation-event contract.
