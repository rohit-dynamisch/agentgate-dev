# G3-W3 — QA/Security

## What
Independently verify policy lifecycle, persistence, API security, workspace isolation, concurrency, and failure semantics.

## Why
Persistence becomes part of the authorization control plane. A race, cross-workspace lookup, stale active pointer, or unauthorized mutation can alter authorization without changing Cedar.

## How
Test behavior independently of implementation details.

Lifecycle: invalid candidate rejected; validate does not mutate active; activation produces exactly one valid active version; rollback selects known version; unknown rollback leaves active unchanged.

Concurrency: readers during activation observe only complete valid versions; concurrent activation cannot leave multiple active versions.

Isolation: workspace A cannot read/mutate workspace B.

Security: unauthenticated/unauthorized mutations rejected; malformed input safe; no credential leakage.

Persistence: clean migration; repeat migration behavior; restart persistence; DB unavailable has defined non-authorizing behavior.

Escalate undefined stale-write semantics, ambiguous admin authorization, any persistence failure that could permit unsafe ALLOW, or contract mismatch.
