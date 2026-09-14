# G4-W1 — Go Backend / Governance Integration

## What
Integrate the G3 active-policy lifecycle with the frozen G1 decision engine. Implement/verify dry-run, activation/rollback propagation, provenance correlation, cache correctness, and the policy mutation event/hook contract needed by G5.

## Why
G3 proved persistence and lifecycle state, but not that changing the active persisted policy changes the policy actually used for authorization. G4 closes that integration gap.

## How
Inspect G1 decision contracts, G2 context assembly, and G3 policy manager/store before changing code. Establish the integration boundary first.

The decision engine must consume the currently active validated policy. Dry-run evaluates a candidate without changing active state or cache. Activation updates the evaluation source atomically. Rollback restores a known prior version and subsequent evaluations use it. Provenance must match the evaluated version.

Emit a structured mutation event/hook with sufficient workspace/version/action/correlation information for G5, without implementing G5's durable audit engine.

## Tests
- B dry-run changes predicted result while A remains active
- activation B changes live decision outcome
- rollback restores A outcome
- failed activation leaves A effective
- cache cannot serve stale policy after activation
- provenance exactly follows active version
- mutation event/hook emitted according to contract
- workspace isolation retained
- load/evaluation failure denies

## Escalate
Any need to change G1 Request/Result, bypass G2 assembly, weaken fail-closed behavior, or invent unresolved audit semantics.
