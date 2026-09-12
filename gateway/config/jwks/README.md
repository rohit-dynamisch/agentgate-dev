# Dev-only JWKS fixture placeholder

`dev-fixture-jwks.json` (referenced by `../g1-agentgateway.yaml`'s `jwtAuth.jwks.file`) is a
**placeholder** so the config field resolves to a concrete path. Its key material is a
throwaway, locally-generated fixture key with no relationship to any real identity provider —
never a production credential, and never used to sign anything real.

It is not exercised by the G1 harness or the mock (`agentgate/cmd/g1-mock-authz`): JWT
validation and claims-mapping (JWT claims -> `decision.Identity`) are Day 3/G2 scope
(`internal/identity`, not built yet per `GO_BACKEND_G1_CONTRACT.md`'s known limitations).
It exists only so ticket AG-GW-G1-01's "JWT validation settings" location is concrete on paper.

Do not copy this file into any production or CI configuration.
