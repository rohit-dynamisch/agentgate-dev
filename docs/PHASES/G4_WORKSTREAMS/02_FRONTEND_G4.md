# G4-W2 — Frontend/UI Governance Workflow

## What
Wire G3 governance clients/store into validate → dry-run → activate → rollback with explicit server-confirmed state, impact results, version/hash provenance, and accurate errors.

## Why
G4 is a user-visible governance workflow. The UI must represent server state, not infer authorization or activation locally.

## How
Reuse G3 contracts and extend only where the G4 integration contract requires it. Distinguish candidate, active, historical, in-flight, confirmed, failed, and stale/network states as supported by the backend contract.

Dry-run output must be shown as a preview, not as proof of a live authorization result. Activation/rollback display changes only after server confirmation.

## Tests
- dry-run leaves active display unchanged
- activation updates only after confirmation
- rollback restores prior state after confirmation
- failed mutation preserves confirmed state
- network/stale state cannot show false success
- provenance displayed consistently
