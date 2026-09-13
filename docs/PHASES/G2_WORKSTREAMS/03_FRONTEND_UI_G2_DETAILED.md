# G2 Workstream 03 — Frontend / UI

## Mission
Extend the framework-agnostic G1 model layer so a future operator UI can represent identity mappings, tool inventory, schema fingerprints, risk, drift, and argument declarations without inventing backend semantics.

## Tickets

### G2-FE-01 — Inspect G1 scaffold
Read existing frontend models/tests and project/UI docs. Record reusable models and required governance additions.

### G2-FE-02 — Identity governance models
Model claim-source/mapping configuration, mapping status/errors, and operator-visible validation state. Never model raw bearer tokens as identity evidence.

### G2-FE-03 — Tool inventory models
Model backend/resource identity, tool name, schema fingerprint, fingerprint/drift state, known/unknown state, and risk. Unknown/drifted tools need explicit states.

### G2-FE-04 — Argument declaration models
Model argument name/type and required/optional semantics only where supported by backend contract. Keep undeclared arguments and explicit null distinguishable.

### G2-FE-05 — Fixtures/state handling
Add fixtures for valid governance, unknown tool, schema drift, malformed identity, argument type mismatch, null, stale governance, and API error. Preserve DENY/error/stale semantics.

### G2-FE-06 — Cross-stream review
Compare every model against the Go contract and durable docs. Do not invent fields for UI convenience; record discrepancies.

### G2-FE-07 — Verification
Run frontend build/tests, inspect scope, produce report + digest, and commit only assigned scope.

## DoD
- No framework commitment.
- No token storage/handling.
- Governance states preserve fail-closed semantics.
- Models correspond to backend concepts.
- No G1 contract drift.
