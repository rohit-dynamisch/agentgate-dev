# G2 Workstream 01 — Go Backend

## Mission
Build the AgentGate-side identity and tool-governance boundaries that can safely populate the frozen G1 decision context. Keep identity resolution, tool inventory, fingerprinting, and argument declarations separate from the Cedar decision engine.

## Tickets

### G2-GO-01 — Inspect and freeze package boundaries
Read `internal/decision`, the existing `internal/authz` placeholder, G1 contract, project docs, and open decisions. Identify reusable code, protected boundaries, proposed packages/interfaces, and later dependencies. Do not modify G1 types. Verify `go test ./...`.

### G2-GO-02 — Identity mapping contract
Define a configuration-driven mapping abstraction from trusted gateway-validated claims/context to AgentGate identity fields. No hardcoded JWT claim names. Distinguish valid, missing, malformed, and ambiguous identity. Fail closed. Do not validate tokens here. Add tests for every failure class.

### G2-GO-03 — Tool identity model
Define canonical tool identity using backend/resource identity, tool name, and normalized input-schema material. Make canonicalization deterministic. Meaningful schema changes must alter the fingerprint; invalid/ambiguous schema must fail closed. Do not silently make an algorithm choice normative if the repository decision is not frozen.

### G2-GO-04 — Tool inventory/classification boundary
Create an authoritative governance abstraction containing canonical tool identity, known/unknown state, risk, schema fingerprint, and policy-input declarations. Unknown tools cannot inherit by name similarity. Fingerprint mismatch must fail closed. Do not add resource-level authorization.

### G2-GO-05 — Typed argument declarations
Define per-tool declarations that explicitly whitelist policy inputs and their types. Undeclared args never enter policy attributes; declared values must match type; JSON null is rejected; missing optional differs from explicit null; duplicate/conflicting declarations are invalid; declaration/schema mismatch fails closed. Never expose arbitrary argument maps to Cedar.

### G2-GO-06 — Context assembly adapter
Create a narrow adapter from validated identity + authoritative tool metadata + declared arguments into the existing G1 `decision.Request`. Only populate fields already represented by the frozen contract. Never invent missing security context. Do not solve O-008 in gateway config.

### G2-GO-07 — Shared fixtures
Provide durable fixture definitions/examples for valid identity, missing identity, unknown tool, fingerprint drift, type mismatch, undeclared arg, null arg, missing risk, and valid allow/deny context. Coordinate semantics with QA/Gateway.

### G2-GO-08 — Verification/hand-off
Run `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test ./...`; inspect scope; use close-out checklist; produce report + digest; commit only assigned scope.

## DoD
- No hardcoded JWT claim assumptions.
- No token passthrough.
- Cedar remains isolated to the policy boundary.
- No arbitrary args to Cedar.
- Fingerprint drift cannot silently remain authorized.
- Missing/ambiguous security context fails closed.
- All G1 tests remain green.
