# WS-D — QA/Security — G1 Report

**Branch:** g1/qa-security (off g1/go-backend commit a6d11f2), commit — see `git log -1` on this branch at merge time.

This document is WS-D's canonical G1 evidence deliverable
(`docs/PHASES/G1_WORKSTREAMS/04_QA_SECURITY_G1_DETAILED.md`). It is produced as an
independent verification stream: it does not import or re-run the Go Backend
stream's own unit tests as if they were QA's; it builds the real
`cmd/g1-mock-authz` binary and drives it over its documented HTTP wire contract
(`docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md`) from a separate test
package, `agentgate/qa/g1blackbox`, that imports no `internal/*` package.

## Tickets completed

**AG-QA-G1-01 — Authoritative G1 test matrix.** Done. See "Test matrix and
scenarios/results" below: every row has a literal JSON request body and an
executable assertion in `agentgate/qa/g1blackbox/blackbox_test.go`. DoD met:
yes.

**AG-QA-G1-02 — Independent negative tests.** Done. 27 top-level test functions
(plus subtests) in `agentgate/qa/g1blackbox/blackbox_test.go` cover: identity
removed entirely, identity structurally malformed (wrong JSON types for
`roles`/`agent_id`, an undocumented nested field, missing/empty/blank-entry
roles, `on_behalf_of == agent_id`), unknown/unclassified tool (including a
caller that asserts `known:false` while still supplying a `risk`), malformed
arguments (type mismatch, unknown declared type, and the JSON-`null` edge
case), simulated policy/evaluation failure (`broken` role), and unsupported
top-level request shapes (JSON array, bare JSON string, empty body,
non-JSON garbage, wrong HTTP method, wrong path). Every assertion checks an
actual blocking outcome (`DENY` in the decision body, or a transport-level
400/404/405 rejection) — never merely "an error string appeared." DoD met:
yes — `assertDeny`/`assertNeverAllow` call `t.Fatalf` (not just `t.Errorf`) the
instant any mandatory unsafe case is observed to return `ALLOW`, so the suite
fails loudly rather than quietly.

**AG-QA-G1-03 — Validate Go contract and fixtures independently.** Done. Field
names, types, and required/optional semantics were checked against the
*actual* running mock (not just the doc) via direct `curl` probes and the
`TestContract_*` tests. One discrepancy found and documented (not a security
defect — see Security findings). DoD met: yes, with the one documented
exception below rather than a silent pass.

**AG-QA-G1-04 — Verify Gateway interoperability.** Partially done, as expected
per the assignment's own cross-stream note. The six mandatory scenarios were
run directly and repeatedly against the real, running mock over its real HTTP
interface (`TestMandatory_*` in `blackbox_test.go`), which independently
proves the decision-core contract at the wire boundary. Full
agentgateway-mediated interoperability was **not** verified: `git log
g1/gateway-mcp` and `git diff g1/go-backend g1/gateway-mcp` were checked and
show the Gateway/MCP branch is identical to `g1/go-backend` at commit
`a6d11f2` — no Gateway/MCP artifact exists yet to test against. DoD met:
adjusted-for-reality yes (ALLOW continues, DENY blocks, failure blocks, no
governed bypass observable — all proven at the mock's wire boundary); the
gateway-mediated proof is explicitly flagged outstanding, not silently
skipped.

**AG-QA-G1-05 — Review trust boundaries.** Done. See "Security findings" below
for the explicit written answers to every required question. One point
(client-controlled tool classification) is a documented, accepted trust model
per the Go Backend contract itself, not a newly discovered gap — but it is
still called out explicitly per the DoD's instruction to escalate rather than
silently decide. DoD met: yes.

**AG-QA-G1-06 — Produce the G1 evidence report.** Done — this document, plus
the identical report delivered as the task's final chat response.

## Test matrix and scenarios/results

All rows executed against `go run ./cmd/g1-mock-authz` (built fresh, started
on a random free port, exercised over real HTTP) via
`agentgate/qa/g1blackbox/blackbox_test.go`. All PASS as of this branch's HEAD.

| # | Scenario | Request (abbreviated) | Expected | Observed | Test |
|---|---|---|---|---|---|
| 1 | Valid identity + allow | `identity.roles:["reader"]`, `tool.risk:"read"`, `known:true` | ALLOW / `policy_allow`, 64-hex `policy_version` | ALLOW / `policy_allow` | `TestMandatory_ValidIdentity_Allow` |
| 2 | Valid identity + deny (explicit forbid) | `identity.roles:["admin"]`, `tool.risk:"destructive"` | DENY / `policy_deny`, policy_version present | DENY / `policy_deny` | `TestMandatory_ValidIdentity_Deny_ExplicitForbid` |
| 3 | Valid identity + deny (no matching policy) | `identity.roles:["reader"]`, `tool.risk:"write"` | DENY / `no_matching_policy` | DENY / `no_matching_policy` | `TestMandatory_ValidIdentity_Deny_NoMatchingPolicy` |
| 4 | Missing identity | `identity` field omitted entirely | DENY / `invalid_identity`, policy_version empty | DENY / `invalid_identity`, empty | `TestMandatory_MissingIdentity_Deny` |
| 5 | Unknown/unclassified tool | `classification.known:false` | DENY / `unknown_tool` | DENY / `unknown_tool` | `TestMandatory_UnknownTool_Deny` |
| 6 | Malformed input (undocumented field) | extra top-level `"bogus_field"` | Blocked (400) — never ALLOW | HTTP 400 | `TestMandatory_MalformedInput_Deny` |
| 7 | Policy/evaluation failure | `identity.roles:["broken"]`, `amount` omitted | DENY / `evaluation_error`, policy_version present | DENY / `evaluation_error` | `TestMandatory_PolicyEvaluationFailure_Deny` |
| 8 | Identity removed entirely | no `identity` key at all | DENY / `invalid_identity` | DENY / `invalid_identity` | `TestNegative_IdentityRemovedEntirely` |
| 9 | Identity: `roles` wrong JSON type (string not array) | `"roles":"reader"` | Blocked (400) | HTTP 400 | `TestNegative_IdentityRolesWrongJSONType` |
| 10 | Identity: `agent_id` wrong JSON type (number) | `"agent_id":123` | Blocked (400) | HTTP 400 | `TestNegative_IdentityAgentIDWrongJSONType` |
| 11 | Identity: extra nested field | `"identity":{...,"impersonate":"agent-admin"}` | Blocked (400) | HTTP 400 | `TestNegative_IdentityExtraNestedField` |
| 12 | Identity: `roles` missing | `identity` present, `roles` key absent | DENY / `invalid_identity` | DENY / `invalid_identity` | `TestNegative_IdentityMissingRoles` |
| 13 | Identity: `roles: []` | empty array | DENY / `invalid_identity` | DENY / `invalid_identity` | `TestNegative_IdentityEmptyRolesArray` |
| 14 | Identity: blank role entry | `["reader"," "]` | DENY / `invalid_identity` | DENY / `invalid_identity` | `TestNegative_IdentityBlankRoleEntry` |
| 15 | Identity: `on_behalf_of == agent_id` | both `"agent-1"` | DENY / `invalid_identity` | DENY / `invalid_identity` | `TestNegative_IdentityOnBehalfOfEqualsAgentID` |
| 16 | Unknown tool + spoofed risk while `known:false` | `known:false, risk:"read"` | DENY / `unknown_tool` (risk must not rescue it) | DENY / `unknown_tool` | `TestNegative_UnknownTool_ClassificationKnownFalseWithRiskSet` |
| 17 | Unknown tool via omitted `classification` | `classification` key absent | DENY / `unknown_tool` | DENY / `unknown_tool` | `TestNegative_UnknownTool_ClassificationOmittedEntirely` |
| 18 | Argument type mismatch | `{"type":"int","value":"500"}` | DENY / `malformed_request` | DENY / `malformed_request` | `TestNegative_ArgumentsTypeMismatch` |
| 19 | Argument unknown declared type | `{"type":"float","value":500}` | DENY / `malformed_request` | DENY / `malformed_request` | `TestNegative_ArgumentsUnknownDeclaredType` |
| 20 | Simulated failure via `broken` role | `amount` omitted | DENY / `evaluation_error` | DENY / `evaluation_error` | `TestNegative_PolicyEvaluationFailure_BrokenRole` |
| 21 | Top-level shape: JSON array | body `[1,2,3]` | Blocked (400) | HTTP 400 | `TestNegative_TopLevelShape_JSONArray` |
| 22 | Top-level shape: bare JSON string | body `"just a string"` | Blocked (400) | HTTP 400 | `TestNegative_TopLevelShape_JSONString` |
| 23 | Top-level shape: empty body | body `` | Blocked (400) | HTTP 400 | `TestNegative_TopLevelShape_EmptyBody` |
| 24 | Top-level shape: non-JSON garbage | body `not json{{{` | Blocked (400) | HTTP 400 | `TestNegative_TopLevelShape_NotJSONAtAll` |
| 25 | Wrong method: GET | `GET /evaluate` | Blocked (405) | HTTP 405 | `TestNegative_WrongMethod_GET` |
| 26 | Wrong method: PUT | `PUT /evaluate` | Blocked (405) | HTTP 405 | `TestNegative_WrongMethod_PUT` |
| 27 | Wrong path | `POST /authorize` | Blocked (404) | HTTP 404 | `TestNegative_WrongPath` |
| 28 | **JSON `null` for declared int argument** | `{"amount":{"type":"int","value":null}}`, `payer` role | *(open question — see finding)* | **ALLOW** / `policy_allow` (null silently coerced to `0`) | `TestSecurityFinding_JSONNullArgument_SilentlyBecomesZeroValue_Int` |
| 29 | JSON `null` vs omitted argument on `broken` role | compare `amount` omitted vs `amount:null` | *(open question)* | Omitted → `evaluation_error`; `null` → `no_matching_policy` (different reason codes — proves null ≠ absent) | `TestSecurityFinding_JSONNullArgument_ChangesBrokenRoleOutcome` |
| 30 | Policy version provenance present when Cedar reached | ALLOW and explicit-forbid DENY | Same non-empty 64-hex value both times | Confirmed equal | `TestContract_PolicyVersionProvenance_PresentWhenCedarReached` |
| 31 | Policy version provenance empty when Cedar not reached | missing identity / unknown tool / malformed | `policy_version == ""` in all three | Confirmed | `TestContract_PolicyVersionProvenance_EmptyWhenCedarNotReached` |
| 32 | `execution_id` preserved on every path | allow / deny / missing-identity / unknown-tool / eval-error | Non-empty, unchanged from request | Confirmed on all 5 | `TestContract_ExecutionIDPreservedOnEveryPath` |
| 33 | **`risk` omitted while `known:true`** | `classification:{"known":true}` (no `risk`) | *(contract says "required")* | DENY / `no_matching_policy` — **not** `malformed_request** | `TestContract_RiskOmittedWhenKnownTrue_NotRejectedAsMalformed` |
| 34 | Client asserts own classification, arbitrary tool name | `tool.name:"delete_all_customer_records"`, `known:true, risk:"read"` | Evaluated purely on asserted `risk` | ALLOW (name ignored, only `risk` matters) | `TestTrustBoundary_ClientCanAssertOwnClassification_NoServerSideVerification` |
| 35 | Arbitrary/unrecognized `risk` string | `risk:"mostly_harmless"` | Fails closed (no rule matches) | DENY / `no_matching_policy` | `TestTrustBoundary_ArbitraryRiskString_FailsClosedWhenUnrecognized` |
| 36 | 400 error body never looks like a decision | non-JSON body | No `"decision"` field in error JSON | Confirmed absent; `"error"` field present | `TestTrustBoundary_ErrorResponseHasNoDecisionField` |
| 37 | Determinism | identical body twice | Identical response both times | Confirmed | `TestDeterminism_IdenticalInputSameOutput` |

**Run command:**

```bash
cd agentgate
go test ./qa/g1blackbox/... -v
```

**Result:** `PASS` — all 27 top-level tests (37 including subtests) pass. Full
module: `go test ./...` — all packages `ok`, none `FAIL`.

## Security findings

### 1. JSON `null` for a declared argument is silently coerced to the type's zero value, not rejected as malformed

**What was checked (per the assignment brief's explicit instruction):** whether
a `null` JSON value for a declared int/string/bool argument is rejected as
malformed or silently accepted.

**Finding:** it is silently accepted. `mockauthz.attributeWire.toDomain()`
calls `json.Unmarshal(a.Value, &n)` (or `&s`/`&b`) directly into a
non-pointer Go scalar. Go's `encoding/json` treats a JSON `null` unmarshaled
into a non-pointer target as a documented no-op: it neither errors nor
modifies the target, which therefore keeps its zero value (`0` / `""` /
`false`) — and `err == nil`. The wire layer then constructs a **valid**
`decision.AttributeValue` (e.g. `IntAttr(0)`) rather than the zero-value
`AttributeValue{}` that `decision.Engine` treats as malformed.

**Why this matters:** it is not merely an academic type-coercion quirk. It was
reproduced end-to-end: `{"amount":{"type":"int","value":null}}` sent as a
`payer` role against a `risk:"write"` tool evaluates **identically** to
sending `{"amount":{"type":"int","value":0}}` — both **ALLOW**, because the
fixture policy's rule is `amount <= 1000`. A caller (or an agent influenced by
injected content) that supplies `null` for a required argument — because the
value was missing, unavailable, or attacker-manipulated — gets treated as if
it had legitimately supplied a small, policy-favorable value, not as an
invalid/absent input. This is the general shape of a "null byte"/type-confusion
bypass class of bug, here manifesting as argument-authorization silently
downgrading to a favorable default rather than denying.

Separately, on the `broken` role fixture: omitting `amount` entirely produces
a genuine Cedar evaluation error (`context.amount` is referenced without a
`has` guard against a truly absent key) → correctly denies as
`evaluation_error`. Sending `amount: null` instead changes the *mechanism*:
`context has amount` becomes `true` (the key is now "present" with value
`0`), so the rule's `context.amount > 10` evaluates to `false` rather than
erroring, and the request falls through to ordinary deny-by-default
(`no_matching_policy`). Both outcomes happen to be DENY here only because this
particular fixture rule's threshold (10) is not crossed by 0 — the *same*
underlying null-coercion mechanism produced an unsafe ALLOW in the `payer`
case above where the threshold (1000) is not exceeded by 0 either, but the
rule direction is a permit rather than a further condition. In short: whether
this class of bug is safe or unsafe depends entirely on which side of a
threshold `0`/`""`/`false` happens to fall on for a given policy — it is not
a property the current implementation is actually guarding, it is incidental.

**Severity:** Medium-High. It does not currently bypass the *unknown-tool* or
*invalid-identity* invariants (those are unaffected), but it is a real gap in
"malformed authorization input cannot produce an ALLOW" (per
`docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md` §1 and
`docs/SECURITY/PRODUCTION-INVARIANTS.md` §2 item 5) for the argument channel
specifically. It should be treated as an open item for the argument-attribute
validation work already tracked as O-006 (per-tool argument declaration),
not something QA is resolving unilaterally here.

**Recommendation (not a unilateral fix — flagged for Architect/Go Backend
review):** `attributeWire.toDomain()` should distinguish "value is JSON
`null`" from "value is a valid zero" (e.g. check `bytes.Equal(a.Value,
[]byte("null"))` before attempting the type-specific unmarshal, and treat
`null` as invalid input) so it produces the same zero-value
`decision.AttributeValue{}` that already triggers `ReasonMalformedRequest`.

**Reproduction:** `TestSecurityFinding_JSONNullArgument_SilentlyBecomesZeroValue_Int`
and `TestSecurityFinding_JSONNullArgument_ChangesBrokenRoleOutcome` in
`agentgate/qa/g1blackbox/blackbox_test.go`.

### 2. Contract/fixture discrepancy: `Classification.Risk` documented as "required when Known == true" but not enforced as malformed

`docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md` (AG-GO-G1-02) states
`Classification.Risk` is "Required when `Known == true`." In the actual
running mock, sending `classification:{"known":true}` with `risk` omitted is
**not** rejected as `malformed_request`. It reaches Cedar with an empty
`resource.risk` string, matches no fixture rule, and denies as
`no_matching_policy`. The outcome is still safely DENY — this is **not** a
fail-open bug — but it is a discrepancy between the documented "required"
semantic and the actual validation performed (`validateRequestShape` in
`internal/decision/engine.go` does not check `Classification.Risk` at all).
Recorded per AG-QA-G1-03's DoD ("no undocumented required field or semantic
is discovered, or exactly what was found is documented") rather than silently
passed over. Low severity; a documentation-precision item, not a security
defect, but worth a one-line contract-doc correction ("empty risk when
`known:true` reaches Cedar and denies via `no_matching_policy`, not a
structural `malformed_request` check").

**Reproduction:** `TestContract_RiskOmittedWhenKnownTrue_NotRejectedAsMalformed`.

### 3. Trust boundary review (AG-QA-G1-05) — explicit answers

- **Which identity fields are trusted, and on what basis?** None, at the mock
  boundary. `decision.Engine.Evaluate` performs no signature/token
  verification and trusts every field on `Request` exactly as given — this is
  explicitly documented as a structural (not per-request) trust decision in
  `GO_BACKEND_G1_CONTRACT.md`'s "Authentication / trust-provenance assumption"
  section. In production, the *only* legitimate populator of `Identity` is the
  not-yet-built `ext_authz` layer (`internal/authz`), which must derive it
  from claims `agentgateway` has already cryptographically validated
  (`docs/SECURITY/PRODUCTION-INVARIANTS.md` §1, §3). Every request sent
  through the G1 mock — including every test in this suite — is, by
  construction, an unauthenticated test fixture, not a production-trust
  request. This is a documented, accepted limitation of the mock, not a
  newly discovered gap, but it is restated here explicitly as required by the
  DoD.
- **Which values are supposed to originate from authenticated gateway state,
  vs. what the mock currently just accepts as given?** In production:
  `Identity.AgentID`, `Identity.OnBehalfOf`, and `Identity.Roles` should all
  derive from gateway-validated JWT claims via a claims-mapping configuration
  (not yet built — Day 3/G2 scope, `internal/identity`). The mock accepts all
  three exactly as sent over HTTP, with zero verification, because it has no
  gateway or JWT layer in front of it at all — this is by design for G1
  parallelization, and is explicitly called out as "known limitation #2" in
  `GO_BACKEND_G1_CONTRACT.md`.
- **Which tool fields are client-controlled?** All of them, at this layer:
  `Tool.BackendID`, `Tool.Name`, `Classification.Known`, and
  `Classification.Risk` are all populated directly from the wire request with
  no independent, server-side tool-inventory lookup. Nothing in the G1
  contract cross-checks a claimed classification against any authoritative
  source.
- **Can a client assert its own tool classification, and what happens if it
  tries?** Yes, completely — proven by
  `TestTrustBoundary_ClientCanAssertOwnClassification_NoServerSideVerification`:
  a request naming a tool `delete_all_customer_records` but asserting
  `known:true, risk:"read"` is **ALLOW**ed for a `reader` role, purely because
  the asserted risk label is favorable — the tool's name has zero bearing on
  the decision. This matches the documented v1 architecture (tool
  classification is meant to be curated upstream, in an operator-managed tool
  inventory that doesn't exist yet — Day 3+ scope, O-005) rather than being a
  bug introduced by this contract, but it means: **whichever component
  populates `decision.Request` in production is a full trust boundary for
  tool risk, and that component does not yet exist.** This is escalated here
  explicitly rather than assumed resolved, per the DoD's "escalate rather
  than decide" instruction — it is the single most important reason G1 cannot
  be read as "the tool-governance boundary is closed."
- **Are arguments authorization context rather than trusted authorization
  state?** Yes, by construction: `Request.Arguments` only ever contains
  explicitly declared attributes the caller chooses to send; there is no
  channel for undeclared data to reach Cedar (confirmed by reading
  `internal/decision/types.go` and `engine.go`, and consistent with
  `docs/SECURITY/PRODUCTION-INVARIANTS.md` §2 item 6). However, "context, not
  state" does not mean "safe by default" — Finding #1 above shows that even
  within this narrow, declared channel, a `null` value is not validated
  strictly enough to prevent a caller from supplying an effectively-arbitrary
  favorable value under the guise of "supplying" a required argument.
- **Are sensitive values unnecessarily exposed anywhere (error messages,
  logs)?** None found in this scope. `Result.Message` is never populated from
  raw Cedar diagnostic text (verified by reading `engine.go` — every `Message`
  is a small fixed AgentGate-authored string) and the mock's own transport
  error bodies only echo the JSON decode error string (e.g. "cannot unmarshal
  string into Go struct field..."), which describes the *shape* of the
  caller's own malformed request, not any secret or credential. No
  argument values, tokens, or identity fields beyond what the caller itself
  submitted were observed being reflected back. This scope does not cover
  logging (the mock logs only a single startup line;
  `internal/logging`/structured request logging is Day 3+/out of G1 scope).
- **Can an error response ever be mistaken for ALLOW by a naive caller?** Not
  by inspecting the JSON body: a 400 error body has no `"decision"` field at
  all (confirmed by `TestTrustBoundary_ErrorResponseHasNoDecisionField`), so a
  caller that actually parses the documented `Result` shape cannot confuse it
  with `ALLOW`. **However**, a caller that checks *only* the HTTP status code
  (e.g. treating any `200` as "proceed") would be wrong to do so: **every**
  decision-core outcome — `ALLOW` and every `DENY` reason alike — is returned
  as HTTP 200, by design (the contract states decision is data in the body,
  not conveyed by status code, mirroring the real `ext_authz` boundary). This
  is not a defect, but it is a real integration risk worth stating plainly for
  any consumer of this contract (Gateway/MCP, Frontend/UI): **you must parse
  and check the `decision` field; HTTP 200 alone means nothing.** Recorded
  here as an explicit trust-boundary fact rather than assumed obvious.

## Evidence

**Build the mock:**

```bash
cd agentgate
go build -o /tmp/g1-mock-authz.exe ./cmd/g1-mock-authz
# BUILD_OK
```

**Run it and probe manually (excerpted; full session covered every row in the
test matrix above):**

```bash
$ curl -s -X POST localhost:8091/evaluate -d '{"execution_id":"e1","workspace_id":"w1","identity":{"agent_id":"a1","roles":["reader"]},"tool":{"backend_id":"b","name":"read_schedule"},"classification":{"known":true,"risk":"read"}}'
{"decision":"ALLOW","reason":"policy_allow","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"e1"}

$ curl -s -X POST http://localhost:8091/evaluate -d '{"execution_id":"n1","workspace_id":"w1","identity":{"agent_id":"a1","roles":["payer"]},"tool":{"backend_id":"b","name":"transfer"},"classification":{"known":true,"risk":"write"},"arguments":{"amount":{"type":"int","value":null}}}'
{"decision":"ALLOW","reason":"policy_allow","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"n1"}

$ curl -s -X POST http://localhost:8091/evaluate -d '{"execution_id":"z1","workspace_id":"w1","identity":{"agent_id":"a1","roles":["payer"]},"tool":{"backend_id":"b","name":"transfer"},"classification":{"known":true,"risk":"write"},"arguments":{"amount":{"type":"int","value":0}}}'
{"decision":"ALLOW","reason":"policy_allow","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"z1"}
# ^ identical outcome to null=value, confirming null -> int zero-value coercion

$ curl -s -X POST http://localhost:8091/evaluate -d '{"execution_id":"b1","workspace_id":"w1","identity":{"agent_id":"a1","roles":["broken"]},"tool":{"backend_id":"b","name":"whatever"},"classification":{"known":true,"risk":"write"}}'
{"decision":"DENY","reason":"evaluation_error","message":"cedar policy evaluation reported an error","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"b1"}

$ curl -s -X POST http://localhost:8091/evaluate -d '{"execution_id":"b2","workspace_id":"w1","identity":{"agent_id":"a1","roles":["broken"]},"tool":{"backend_id":"b","name":"whatever"},"classification":{"known":true,"risk":"write"},"arguments":{"amount":{"type":"int","value":null}}}'
{"decision":"DENY","reason":"no_matching_policy","message":"no policy matched this request; deny-by-default","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"b2"}
```

**Independent black-box suite:**

```bash
$ cd agentgate
$ go test ./qa/g1blackbox/... -v
...
--- PASS: TestMandatory_ValidIdentity_Allow (0.01s)
--- PASS: TestMandatory_ValidIdentity_Deny_ExplicitForbid (0.00s)
--- PASS: TestMandatory_ValidIdentity_Deny_NoMatchingPolicy (0.00s)
--- PASS: TestMandatory_MissingIdentity_Deny (0.00s)
--- PASS: TestMandatory_UnknownTool_Deny (0.00s)
--- PASS: TestMandatory_MalformedInput_Deny (0.00s)
--- PASS: TestMandatory_PolicyEvaluationFailure_Deny (0.00s)
... [27 top-level tests, 37 including subtests, all PASS]
PASS
ok  	github.com/Dynamisch-LLC/agentgate/qa/g1blackbox	8.446s
```

**Full module regression (confirms this suite introduces no regressions and
the Go Backend stream's own tests still pass):**

```bash
$ go build ./...
# (no output — success)
$ go test ./...
?   	github.com/Dynamisch-LLC/agentgate/cmd/agentgate	[no test files]
?   	github.com/Dynamisch-LLC/agentgate/cmd/g1-mock-authz	[no test files]
?   	github.com/Dynamisch-LLC/agentgate/internal/audit	[no test files]
?   	github.com/Dynamisch-LLC/agentgate/internal/authz	[no test files]
ok  	github.com/Dynamisch-LLC/agentgate/internal/config
ok  	github.com/Dynamisch-LLC/agentgate/internal/decision
?   	github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy	[no test files]
ok  	github.com/Dynamisch-LLC/agentgate/internal/httpserver
?   	github.com/Dynamisch-LLC/agentgate/internal/identity	[no test files]
ok  	github.com/Dynamisch-LLC/agentgate/internal/logging
ok  	github.com/Dynamisch-LLC/agentgate/internal/mockauthz
ok  	github.com/Dynamisch-LLC/agentgate/internal/policy
ok  	github.com/Dynamisch-LLC/agentgate/qa/g1blackbox
```

**Gateway/MCP artifact check (for AG-QA-G1-04):**

```bash
$ git log --oneline -3 g1/gateway-mcp
a6d11f2 feat(g1): freeze Go Backend G1 authorization contract
c6dbdbd checkpoint G1 plans
7adceda test(policy): verify context isolation and parsing errors
$ git diff --stat g1/go-backend g1/gateway-mcp
# (empty — no diff; the branch is at the same commit, no Gateway/MCP artifact exists yet)
```

**Fixtures used:** `internal/fixturepolicy.CedarSource` (canonical G1 fixture
policy; role/risk constants `RoleReader`/`RoleAdmin`/`RolePayer`/`RoleBroken`,
`RiskRead`/`RiskWrite`/`RiskDestructive`) — read, not modified. No fixture or
production code was changed by this workstream.

## Limitations

1. **Gateway-mediated interoperability is unproven in this run.** This is the
   central, expected limitation flagged in the assignment brief itself: WS-B
   (Gateway/MCP) is being executed in a separate, isolated parallel worktree
   and has produced no artifact yet reachable from this branch (`g1/gateway-mcp`
   is byte-identical to `g1/go-backend` at commit `a6d11f2`). Everything in
   this report proves the decision-core contract at the mock's real HTTP wire
   boundary — genuinely valuable, since it is the real `decision.Engine` and
   real Cedar behind that HTTP layer — but it does **not** prove that a real
   `agentgateway` instance, real JWT validation, and a real MCP client compose
   correctly with AgentGate end to end. That proof is explicitly Gate G6
   ("Real MCP Enforcement Gate," Day 8) in
   `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`, not G1.
2. **`go test -race` was not run.** Consistent with the Go Backend stream's
   own documented limitation (no cgo/gcc in this Windows sandbox); this
   applies equally to the new `qa/g1blackbox` package. CI (`ubuntu-latest`)
   should run the race-enabled suite.
3. **This suite tests the mock, not `internal/authz`'s not-yet-built
   production `ext_authz` gRPC layer.** By design — that layer does not exist
   yet (Day 3/Day 8 scope). Its own contract tests, when built, must be
   re-verified independently by QA in a later ticket; this report does not
   claim to have done so.
4. **Argument-attribute vocabulary is closed but unvalidated per-tool**
   (documented Go Backend limitation #4, carried forward unchanged): any tool
   can currently be sent any declared attribute name. This interacts with
   Finding #1 above (the null-coercion gap) but is a distinct, already-known,
   already-tracked limitation (O-006), not a new finding.
5. **Audit, policy persistence, dry-run/rollback, downstream credentials, and
   admin authentication are explicitly out of scope for G1** (per
   `docs/PHASES/G1_WORKSTREAMS/00_G1_CHECKPOINT_REFERENCE.md`, "Explicitly out
   of scope") and were not tested here. They are separately gated (G3–G5, G7)
   later in the plan.
6. **No fuzz testing was performed** on the JSON decoder beyond the specific
   malformed-shape cases enumerated in the test matrix. A dedicated fuzz
   target for the `/evaluate` body parser is called out as a stack-level
   requirement in `docs/TECH_STACK.md` §2.9 but is not part of G1's scope per
   the checkpoint reference.

## PASS/FAIL/CONDITIONAL recommendation

**CONDITIONAL PASS for the G1 Authorization Contract Freeze**, with one
required follow-up before the contract may be relied upon for
argument-bearing tools, and one Gateway-interoperability gap that must close
before any gate beyond G1 (this is expected and already scheduled, not a
surprise).

**Reasoning:**

- All ten G1 Definition-of-Done items in
  `docs/PHASES/G1_WORKSTREAMS/00_G1_CHECKPOINT_REFERENCE.md` that are within
  QA's independent power to verify were independently verified and PASS: the
  canonical request/result contract is real and stable, required/optional
  fields and trust semantics are documented (with one precision gap noted, not
  hidden), ALLOW/DENY/failure semantics are deterministic, the mock is
  callable over its documented HTTP contract, and QA independently verified
  every mandatory negative/fail-closed case (missing identity, unknown tool,
  malformed input, policy/evaluation failure) directly against the running
  binary, with an assertion design (`t.Fatalf` on any unsafe `ALLOW`) that
  fails loudly rather than silently.
- Per this ticket's own exit condition ("G1 cannot PASS from Go unit tests
  alone; Gateway interoperability and independent negative tests must pass"):
  the independent negative tests **do** pass, run from an external,
  non-importing black-box suite. Gateway interoperability, however, could
  only be partially demonstrated — at the mock's wire boundary, not through a
  real `agentgateway` — because no Gateway/MCP artifact exists yet to
  integrate against in this parallel-execution window. That is a scheduling
  fact of Day 3, expected by the plan itself (Gate G6, "Real MCP Enforcement
  Gate," is Day 8), not a defect discovered by QA.
- One genuine, reproducible security-relevant gap was found: **JSON `null`
  for a declared argument silently becomes the type's zero value instead of
  being rejected as malformed**, and this was proven to change a real
  decision outcome (`payer`/`amount` case: ALLOW where a caller supplying no
  genuine amount arguably should not be treated identically to one supplying
  a validated `0`). This does not currently break the four headline
  invariants (missing identity, unknown tool, no-policy, and Cedar-error all
  still deny correctly) but it is a real crack in "malformed input cannot
  produce ALLOW" specifically for the argument channel, and should be fixed
  (see recommendation in Finding #1) before any policy that gates on a
  numeric/string/bool argument threshold is trusted in a real deployment.
- A second, lower-severity discrepancy (contract says `risk` is "required"
  when `known:true`; the implementation does not enforce this as
  `malformed_request`, though it still safely denies) is a documentation-
  precision item, not a security defect, and is recorded for the doc to be
  tightened rather than blocking the freeze.
- The trust-boundary review surfaced one point that is not new but is worth
  restating as an explicit escalation per this ticket's DoD: **the component
  that will populate `decision.Request` with tool classification in
  production does not exist yet**, and until it does, "unknown/unclassified
  tool → DENY" is proven correct at the decision-core boundary but the
  upstream classification-authority boundary itself remains unbuilt and
  therefore unverifiable by QA at this checkpoint. This is Day 3+ scope
  (O-005), already known, and is restated here rather than silently assumed
  resolved.

**What would turn this into an unconditional PASS:** (a) a decision from the
Architect on the JSON-null argument-coercion finding (fix now vs. explicitly
accept and track under O-006), and (b) — per the plan's own schedule, not an
immediate blocker — the Day 8/Gate-G6 real-agentgateway-mediated
interoperability proof this ticket could not obtain in this parallel window.
