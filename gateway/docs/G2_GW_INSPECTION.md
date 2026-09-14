# G2 Gateway/MCP — Inspection Report (G2-GW-01)

**Date:** 2026-09-13  
**Workstream:** G2 Gateway/MCP  
**Ticket:** G2-GW-01 — Inspect current state

---

## 1. Current agentgateway configuration

File: [`gateway/config/g1-agentgateway.yaml`](../config/g1-agentgateway.yaml)

### What G1 established (verified facts)

- **Schema validated** against real `ghcr.io/agentgateway/agentgateway:latest` binary (2026-09-12).
- **Started successfully**: binary accepted config, bound `:3000`, marked ready.
- **JWT enforcement confirmed**: unauthenticated request correctly returns `401` — `jwtAuth: mode: strict` is enforced by the real binary.
- **Route structure**: `routes` is a top-level key (sibling of `gateways`), NOT nested. Verified: the original file had it nested; the real binary rejected it with `unknown field 'routes'`. Fixed in G1.
- **ext_authz wired**: `policies.extAuthz.host: "${AGENTGATE_MOCK_ADDR}"`, HTTP protocol, path `/evaluate`.

### What remains hypothetical / unverified

- **O-008 gap**: `agentgateway`'s documented `extAuthz.protocol.http` fields (`path`, `addRequestHeaders`, `includeResponseHeaders`, `redirect`) operate on URL and headers only — there is no documented mechanism to assemble an arbitrary JSON body. The frozen G1 `decision.Request` shape (`execution_id`, `workspace_id`, `identity`, `tool`, `classification`, `arguments`) cannot be produced from gateway config DSL alone. This is the exact O-008 finding; it is NOT resolved in G2.
- **Real signed JWT end-to-end**: the dev JWKS fixture has zero keys, so no real signed JWT was ever driven through `agentgateway` in G1. JWT enforcement was confirmed independently (bare request → 401).
- **MCP client connection**: no real MCP client session was driven through the gateway in G1.

---

## 2. G1 harness

- **Location**: `gateway/harness/` — independent Go module.
- **What it does**: sends G1 fixture JSONs over real HTTP to `cmd/g1-mock-authz`. Skips (not silently passes) if mock is unreachable.
- **G1 fixture categories covered**: ALLOW, DENY, missing_identity, unknown_tool, malformed_input, evaluation_error.
- **G2 relevance**: G2 fixtures are in `gateway/fixtures/g2/`. The harness is NOT extended for G2 (G2 does not build the ext_authz integration; that is G6 work per O-008).

---

## 3. Open decisions relevant to Gateway/MCP

| Decision | Status | G2 Action |
|---|---|---|
| **O-003** — agentgateway conformance/security boundary | Open | Build targeted integration tests in first E2E slice (not G2) |
| **O-004** — Supported MCP revision(s) | Open | Confirm boundary before finalizing client/backend integration |
| **O-008** — ext_authz transport mapping to decision.Request | Open | **Do NOT resolve in G2.** Document gap; design in G6. |

---

## 4. What ext_authz can provide (today)

Based on agentgateway documentation and G1 verification:

| Source | Available in ext_authz callout | Notes |
|---|---|---|
| Request URL | ✅ Path, query params | Via CEL in `path` expression |
| Request headers | ✅ Any header | Via `addRequestHeaders` CEL |
| JWT claims (post-validation) | ⚠️ Hypothetical | Agentgateway may forward validated claims as headers — **not yet verified** |
| MCP request body | ❌ Not via config DSL | No documented field assembles arbitrary JSON body — O-008 |
| Tool name / backend | ❌ Not natively | Would need to be parsed from MCP body by `internal/authz` — O-008 hypothesis |

---

## 5. G6 handoff expectations

G2 establishes the **boundary contracts** (identity mapping, tool governance, argument declarations). The actual ext_authz → `internal/authz` → `decision.Request` wiring is G6 work (pending O-008 resolution).
