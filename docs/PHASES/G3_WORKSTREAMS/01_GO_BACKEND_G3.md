# G3-W1 — Go Backend

## What
Implement PostgreSQL policy persistence and the governance API:
- policy identity/version/content/hash
- candidate vs active vs historical state
- validation
- create/list/get/validate/preview/activate/rollback
- `workspace_id`
- deterministic migrations
- atomic activation and rollback
- concurrency-safe active reads
- authenticated/authorized policy mutation
- contract, unit, PostgreSQL integration, and concurrency tests

## Why
G3 is the persistence/API boundary for the governance loop. Ambiguous lifecycle transitions, non-atomic activation, stale reads, or cross-workspace access could change authorization behavior incorrectly.

## How
First inspect current policy/decision/config packages and frozen G1/G2 contracts. Reuse existing policy representation; do not create a second model.

Make state transitions explicit. Validation must not mutate active policy. Activation must be atomic. Rollback must target a known persisted version. Readers must see a complete valid active version, never an intermediate state.

Keep policy provenance semantics distinct from G2 tool fingerprinting. Do not silently redefine O-005.

Define the API contract before broad UI integration. Errors must be deterministic and non-sensitive. UI authorization is never sufficient; the backend enforces administration.

## Required tests
Invalid candidate cannot activate; validation leaves active unchanged; activation atomic; unknown rollback rejected; active version explicit; workspace isolation; concurrent reads/activations safe; unauthorized mutation rejected; malformed input safely rejected; provenance preserved; migrations deterministic; restart preserves policy.

## Stop/escalate
Escalate rather than guess if G1 semantics must change, admin authorization requires an unresolved architecture decision, workspace semantics conflict with the frozen runtime model, or atomic activation cannot be guaranteed.
