# G2 Workstream 05 — DevOps / Release Engineering

## Mission
Make G2 identity/tool-governance configuration reproducible and safe across local development and CI without pulling later production deployment hardening into this checkpoint.

## Tickets

### G2-DO-01 — Inspect conventions
Read repository docs, CI baseline, G1 deployment, config loading, and secret handling. Identify checked-in non-secret config versus environment/secret injection versus fixtures.

### G2-DO-02 — Identity mapping config
Provide a reproducible non-secret example showing configurable claim names and required/optional behavior. Never commit credentials/tokens.

### G2-DO-03 — Governance fixture format
Provide deterministic fixtures for tool identity, risk, schema fingerprint, and argument declarations. Do not make this the production persistence mechanism if G3 owns persistence.

### G2-DO-04 — Negative config validation
Validate missing identity mapping, conflicting mappings, malformed schema, missing risk, invalid fingerprint, duplicate declarations, and unsupported argument types. Invalid security config must fail closed.

### G2-DO-05 — Reproducible topology
Extend/create a minimal topology for AgentGate, governance fixtures, QA probes, and gateway fixture where applicable. Do not claim production E2E while O-008 remains unresolved.

### G2-DO-06 — CI integration
Add applicable G2 checks without weakening existing gates or making CI dependent on unavailable local-only infrastructure.

### G2-DO-07 — Closeout
Validate with real tools/daemon where required, inspect scope, ensure no secret/config churn, produce report + digest, and commit only assigned scope.

## DoD
- No secrets committed.
- Invalid security configuration cannot become permissive.
- G1 checks remain green.
- G2 verification is reproducible.
- Production deployment hardening remains deferred.
