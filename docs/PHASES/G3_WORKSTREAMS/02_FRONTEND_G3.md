# G3-W2 — Frontend/UI

## What
Build the typed governance API/workflow foundation for:
- policy list/version details
- candidate policy
- validation
- preview/dry-run boundary
- activation
- rollback
- explicit candidate vs active state
- version/hash
- `workspace_id`
- backend error/stale-state handling

## Why
The 10-day plan requires parallel frontend progress. The UI must consume a stable API contract and must never infer authorization state or claim activation success before server confirmation.

## How
Freeze typed models from the backend contract. Keep candidate and active states distinct. Separate user intent from server-confirmed state. Use mocks/fixtures only as temporary contract tools. Never store raw credentials/tokens.

Test activation failure, rollback failure, stale state, unauthorized mutation, candidate != active, and version/hash display.

Do not invent policy states or database semantics. Do not make UI authorization a security boundary.
