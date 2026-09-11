# Go Backend — G1 Authorization Contract (Handoff)

**Ticket sequence:** `docs/PHASES/G1_WORKSTREAMS/01_GO_BACKEND_G1_DETAILED.md`
**Status:** In progress — updated after each ticket; finalized at AG-GO-G1-06.
**Baseline:** Accepted Day-2 Decision Core (`docs/PHASES/DAY-02-TASK-02.md`), unchanged in shape.

This is the canonical reference for Gateway/MCP, Frontend/UI, QA/Security and DevOps to integrate
against the frozen G1 authorization contract without reading Go source or guessing field semantics.

---

## AG-GO-G1-01 — Existing Day-2 boundary (inventory)

Confirmed unchanged from the accepted Day-2 baseline; `go test ./...` passes before any G1 edit.

| Concern | Location |
|---|---|
| Decision-core entry point | `decision.Engine.Evaluate(req decision.Request) decision.Result` — `agentgate/internal/decision/engine.go` |
| Request/result types | `decision.Request`, `decision.Result` — `agentgate/internal/decision/types.go` (**one** canonical model; no duplicate) |
| Identity | `decision.Identity{AgentID, OnBehalfOf, Roles}` |
| Tool / classification | `decision.ToolRef{BackendID, Name}`, `decision.ToolClassification{Known, Risk}` |
| Declared argument attributes | `decision.AttributeValue` (closed string/int64/bool set via `StringAttr`/`IntAttr`/`BoolAttr`), `Request.Arguments map[string]AttributeValue` |
| Policy version/hash propagation | `decision.Result.PolicyVersion` — sourced from `policy.Engine.Version()` (hex SHA-256 of the loaded Cedar bytes) |
| Validation / fail-closed handling | `validateRequestShape`, `validateIdentity`, tool-classification check, no-policy-loaded check — all run **before** Cedar; a Cedar `HadError` forces DENY regardless of Cedar's own decision |
| Cedar boundary | `agentgate/internal/policy` (only package importing `cedar-go`); `decision` package never imports Cedar types |

No gaps required code changes for G1 beyond what AG-GO-G1-02/03 record below (documentation
clarifications only — the Day-2 shape already satisfies the required semantic fields).

---

## AG-GO-G1-02 — Canonical authorization request

**Type:** `decision.Request` (`agentgate/internal/decision/types.go`) — unchanged shape from Day 2.

| Field | Type | Required? | Semantics |
|---|---|---|---|
| `ExecutionID` | `string` | **Required**, non-empty | Correlates request through decision (and later audit). Preserved unchanged into `Result.ExecutionID` on every path, including every failure path. |
| `WorkspaceID` | `string` | **Required**, non-empty | Scopes the request. v1 is single-tenant; carried for audit correlation and forward compatibility with the latent tenant-aware schema. Not yet consumed by Cedar evaluation itself. |
| `Identity.AgentID` | `string` | **Required**, non-empty | Identifies the calling agent. |
| `Identity.OnBehalfOf` | `string` | Optional | `""` = no delegated human identity. Must not equal `AgentID` when non-empty (see "ambiguous identity" below). |
| `Identity.Roles` | `[]string` | **Required**, non-empty, no blank entries | Evaluated as Cedar principal-group membership. An identity with zero roles is treated as unusable, not silently passed to Cedar's own default-deny. |
| `Tool.BackendID` | `string` | **Required**, non-empty | Identifies the backend the tool belongs to. |
| `Tool.Name` | `string` | **Required**, non-empty | Tool name. `BackendID + "/" + Name` becomes the Cedar resource id. |
| `Classification.Known` | `bool` | **Required** semantically (`false` always denies) | `false` ⇒ unconditional `DENY`/`unknown_tool` before Cedar is ever consulted. |
| `Classification.Risk` | `string` | Required when `Known == true` | Becomes the Cedar resource's `risk` attribute (e.g. `"read"`, `"write"`, `"destructive"`). Ignored/irrelevant when `Known == false`. |
| `Arguments` | `map[string]decision.AttributeValue` | Optional, per-key | Only declared, policy-relevant attributes. **There is no other channel for argument data to reach Cedar** — anything not placed in this map is structurally invisible to policy evaluation. A zero-value `AttributeValue` (not constructed via `StringAttr`/`IntAttr`/`BoolAttr`) is treated as malformed. |

### Authentication / trust-provenance assumption (resolved, documented — not a new field)

`decision.Request` carries **no** trust-provenance field (e.g. "how was this identity verified").
This was a genuine ambiguity in the G1 checkpoint reference's required-semantics list ("authenticated
identity ... trust provenance") and is resolved as follows, rather than guessed silently:

> **`decision.Engine` performs no signature or token verification of its own and never will** — it
> trusts every field on `Request` exactly as given. Trust provenance for v1 is **structural, not a
> per-request data field**: the only legitimate caller of `Evaluate` in production is the
> `ext_authz` gRPC layer (`internal/authz`, Day 3/8), which must populate `Identity` only from
> claims that `agentgateway` has already cryptographically validated
> (`docs/SECURITY/PRODUCTION-INVARIANTS.md §1, §3`). A request built any other way (including
> every call through the G1 mock, see AG-GO-G1-04) is **not** an authenticated request in the
> production sense — it is a test/integration fixture. Adding a "trust provenance" field to
> `Request` today would encode something structurally true of every real caller and would be a
> field with no v1 evaluation purpose, which Day 2 and this ticket's operating rules both prohibit.

**If Gateway/QA/Architect disagree with this resolution**, treat it as open rather than acted-on
silently — flag it back; no code depends on the opposite choice being wrong.

### Denial/failure semantics (fail-closed matrix)

| Condition | `Result.Decision` | `Result.Reason` | `Result.PolicyVersion` |
|---|---|---|---|
| Explicit Cedar permit matched | `ALLOW` | `policy_allow` | evaluated version |
| Explicit Cedar forbid matched | `DENY` | `policy_deny` | evaluated version |
| No Cedar policy matched (default-deny) | `DENY` | `no_matching_policy` | evaluated version |
| Missing/empty `AgentID` | `DENY` | `invalid_identity` | `""` (Cedar never reached) |
| `OnBehalfOf == AgentID` (non-empty) | `DENY` | `invalid_identity` | `""` |
| Empty or blank-entry `Roles` | `DENY` | `invalid_identity` | `""` |
| `Classification.Known == false` | `DENY` | `unknown_tool` | `""` |
| Missing `ExecutionID`/`WorkspaceID`/`Tool.BackendID`/`Tool.Name` | `DENY` | `malformed_request` | `""` |
| Zero-value/invalid `AttributeValue` in `Arguments` | `DENY` | `malformed_request` | `""` |
| Cedar reports an internal evaluation error (`Diagnostic.Errors` non-empty) | `DENY` | `evaluation_error` | evaluated version (Cedar *was* reached) |
| No policy loaded in the `Engine` at all | `DENY` | `no_policy_loaded` | `""` |

`ExecutionID` is preserved unchanged into `Result.ExecutionID` on **every** row above, including
every early/structural denial.

**Verified:** `go test ./internal/decision/... -v` (17 cases covering every row above plus
determinism and argument-dependent-rule cases).

---

## AG-GO-G1-03 — Canonical authorization result

**Type:** `decision.Result` (`agentgate/internal/decision/types.go`) — unchanged shape from Day 2.

| Field | Type | Semantics |
|---|---|---|
| `Decision` | `decision.Decision` (`"ALLOW"` \| `"DENY"`) | Exactly two values. There is no third "error" decision — every failure path resolves to `DENY`. |
| `Reason` | `decision.ReasonCode` (stable string enum — see table above) | Deterministic category; audit/callers never need to parse free text to know what happened. |
| `Message` | `string` | Human-readable detail. **Never populated from raw Cedar diagnostic text** — always an AgentGate-authored string (see below). May be empty on `ALLOW`. |
| `PolicyVersion` | `string` | The exact policy version/hash Cedar evaluated for this decision — hex SHA-256 of the loaded Cedar source bytes. Empty **only** when Cedar was never reached (see the denial matrix above). Never backfilled from "whatever is currently active." |
| `ExecutionID` | `string` | Copied unchanged from `Request.ExecutionID`. |

### "No raw Cedar diagnostics" guarantee

`Result.Message` is always one of a small, fixed set of AgentGate-authored strings (e.g. `"no
policy matched this request; deny-by-default"`, `"an explicit policy denied this request"`,
`"cedar policy evaluation reported an error"`, or a validation message like `"agent id is
missing"`) — `decision.Engine` never copies `cedar-go`'s `Diagnostic.Errors[i].Message` or any
other Cedar-internal string into `Result`. This was already true of the Day-2 implementation;
this ticket makes it an explicit, documented guarantee so a future change cannot regress it
silently.

**Verified:** existing Day-2 tests already assert `Reason`/`Decision` for every category above;
`TestEvaluate_CedarEvaluationError` specifically proves a Cedar-side error never becomes `ALLOW`.

---

## AG-GO-G1-04 — The deterministic AgentGate mock

**What it is:** `cmd/g1-mock-authz` wraps the **real** `decision.Engine` (therefore the real Cedar
boundary) loaded with the canonical fixture policy (`internal/fixturepolicy.CedarSource`) behind a
minimal JSON/HTTP transport (`internal/mockauthz`). It is not a fake — every response is a genuine
Cedar decision. It is a temporary parallelization tool and is never started from `cmd/agentgate`'s
production path (`docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`, Rule 2).

### How to start it

```bash
cd agentgate
go run ./cmd/g1-mock-authz
# listens on :8091 by default; override with G1_MOCK_AUTHZ_ADDR, e.g.:
G1_MOCK_AUTHZ_ADDR=":9000" go run ./cmd/g1-mock-authz
```

### Wire contract

`POST /evaluate`, JSON body and response. All other paths/methods return `404`/`405`.

**Request:**

```json
{
  "execution_id": "exec-123",
  "workspace_id": "ws-1",
  "identity": {
    "agent_id": "agent-1",
    "on_behalf_of": "",
    "roles": ["reader"]
  },
  "tool": {
    "backend_id": "backend-a",
    "name": "read_schedule"
  },
  "classification": {
    "known": true,
    "risk": "read"
  },
  "arguments": {
    "amount": { "type": "int", "value": 500 }
  }
}
```

- `arguments` is optional and omittable. Each entry's `type` is one of `"string"`, `"int"`,
  `"bool"`; `value` must match. An unknown `type` or mismatched `value` becomes an invalid
  attribute, which the real decision core denies as `malformed_request` (not a transport error).
- **Unknown top-level fields are rejected at the transport layer (HTTP 400)** — this catches
  contract drift immediately instead of silently ignoring a typo.

**Response** (always `HTTP 200` once the body decoded — the decision is data, not a status code,
exactly like the real `ext_authz` boundary):

```json
{
  "decision": "ALLOW",
  "reason": "policy_allow",
  "message": "",
  "policy_version": "504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19",
  "execution_id": "exec-123"
}
```

A request body that fails to even decode as JSON (or that fails `DisallowUnknownFields`) returns
`HTTP 400` with a small `{"error": "..."}` body — it never became a `decision.Request`, so there is
no `decision.Result` to report.

### The four required categories, with fixture identities that reach them

Use `internal/fixturepolicy`'s exported constants (`RoleReader`, `RoleAdmin`, `RolePayer`,
`RoleBroken`, `RiskRead`, `RiskWrite`, `RiskDestructive`) to avoid typos — reproduced here as plain
strings for `curl` convenience:

| Category | How to reach it | Example |
|---|---|---|
| **ALLOW** | `roles: ["reader"]` + `risk: "read"` (or `"admin"` + `"read"`/`"write"`, or `"payer"` + `"write"` with `amount <= 1000`) | see request example above |
| **DENY** | `roles: ["reader"]` + `risk: "write"` (no matching policy) **or** any role + `risk: "destructive"` (explicit forbid) | `reason` distinguishes: `no_matching_policy` vs `policy_deny` |
| **malformed input** | omit `identity.agent_id`, or send an unknown top-level field, or send invalid JSON | `HTTP 400` (transport) or `HTTP 200` + `reason: "malformed_request"`/`"invalid_identity"`/`"unknown_tool"` (decision-core), depending on which layer the defect is at — both are documented above |
| **simulated failure** (Cedar evaluation error) | `roles: ["broken"]`, omit `arguments.amount` | `HTTP 200`, `reason: "evaluation_error"`, `decision: "DENY"` — a **genuine** Cedar error via the fixture's deliberately unguarded rule, not a hand-rolled fake |

All four were exercised manually against the running binary (see AG-GO-G1-05 for the automated
form) and behave exactly as this table states.

---

## AG-GO-G1-05 — Executable contract tests

**Location:** `agentgate/internal/mockauthz/handler_test.go` — 12 tests exercising the actual wire
boundary (JSON over HTTP via `httptest`), not just the underlying `decision.Engine` (which has its
own separate, thorough Day-2 test suite in `internal/decision/engine_test.go`, 17 cases).

**Run them:**

```bash
cd agentgate
go test ./internal/mockauthz/... -v
```

**Covers:** valid ALLOW, valid DENY (explicit forbid *and* no-matching-policy, asserted
separately), missing identity, unknown tool, malformed transport body, an undocumented field
(contract-drift guard), Cedar evaluation error, the argument-dependent rule on both sides of its
threshold, method-not-allowed, unknown-path, and same-input determinism. Policy provenance
(`policy_version`) is asserted to be the exact hex SHA-256 of `fixturepolicy.CedarSource` wherever
Cedar was reached, and empty wherever it deliberately was not.

These tests fail if any unsafe condition becomes `ALLOW`, if `policy_version` drifts from the exact
evaluated hash, or if the wire shape changes without an accompanying test update — which is exactly
the freeze boundary this ticket exists to establish.

---

## AG-GO-G1-06 — Cross-stream freeze

**Handoff contents (this document + the code it describes):**

- Canonical request/result: `decision.Request` / `decision.Result`
  (`agentgate/internal/decision/types.go`) — field-by-field semantics in AG-GO-G1-02/03 above.
- Representative ALLOW/DENY/malformed/error payloads: AG-GO-G1-04's wire examples table, and the
  literal JSON bodies used in `internal/mockauthz/handler_test.go`.
- Mock: `cmd/g1-mock-authz` (start command and wire contract in AG-GO-G1-04).
- Contract-test command: `go test ./internal/mockauthz/... -v` (AG-GO-G1-05).
- Shared fixture policy (usable by any stream wanting deterministic Cedar behavior without writing
  their own): `internal/fixturepolicy.CedarSource`, with exported role/risk constants.

**Known limitations (carried forward, not silently resolved):**

1. **Trust provenance is documented, not a data field** — see the resolved-ambiguity note under
   AG-GO-G1-02. Every request through the G1 mock is, by definition, unauthenticated in the
   production sense; only the real `ext_authz` layer (Day 3/8) constitutes a trusted caller.
2. **The mock has no JWT/agentgateway integration** — Gateway/MCP must call `POST /evaluate`
   directly with an already-constructed request; it does not simulate `agentgateway`'s JWT
   validation or claims extraction (that boundary is Day 3/G2, `internal/identity`, not built yet).
3. **`workspace_id` is carried but not evaluated** — no fixture policy rule depends on it; it is
   present in the contract for correlation/audit forward-compatibility only (see
   `docs/SECURITY/PRODUCTION-INVARIANTS.md §9`).
4. **The argument-attribute vocabulary is closed** (string/int64/bool) and there is no per-tool
   declaration registry yet (O-006) — any tool can be sent any declared attribute name for now;
   governing *which* attributes are legitimate per tool is Day 3+ work.
5. **`go test -race`** could not be run in the local development sandbox used for this work (no
   cgo/gcc on that Windows host) — confirmed passing without `-race` locally; CI (`ubuntu-latest`)
   provides gcc and will run the race-enabled suite for real.

**Freeze rule in effect:** as of this document, changing `decision.Request`, `decision.Result`, or
the `mockauthz` wire shape requires Architect review and updated tests in both
`internal/decision` and `internal/mockauthz`
(`docs/PHASES/G1_WORKSTREAMS/00_G1_CHECKPOINT_REFERENCE.md`, DoD item 10).
