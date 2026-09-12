# Gateway/MCP — G1 Authorization Contract Integration (WS-B)

This directory is the Gateway/MCP workstream's G1 deliverable: a reviewable `agentgateway`
configuration for the intended production path, plus a verification harness that proves the
frozen Go Backend G1 contract end to end against the real mock, because the real
`agentgateway` binary could not be executed in this environment (see "Docker / real-binary
gap" below). **This is honestly reported as a gap, not hidden.**

## The intended path (AG-GW-G1-01)

```
MCP Client -> agentgateway (:3000, jwtAuth strict) -> ext_authz -> mock AgentGate (:8091 /evaluate)
                                                                          |
                                                                    (on ALLOW, forwards)
                                                                          v
                                                              real MCP backend (not built for G1 --
                                                              explicitly out of scope, see
                                                              00_G1_CHECKPOINT_REFERENCE.md)
```

No custom MCP proxy exists or was introduced anywhere in this change. `agentgateway` remains
the only component that speaks MCP transport/routing/tool-discovery, exactly as
`docs/PROJECT_DEFINITION.md` §2 and `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md`
§1 require.

### Repository inventory before this change (AG-GW-G1-01 step 1-4)

Confirmed empty/absent before this branch (searched the full repo tree):

- No `gateway/` directory, no `agentgateway` configuration of any kind.
- No MCP client or MCP backend fixture.
- No JWT validation settings.
- No `ext_authz` configuration or placeholder.

This is a from-scratch design, not a modification of existing gateway config.

### Bypass check (AG-GW-G1-01 step 5)

`gateway/config/g1-agentgateway.yaml` defines exactly one route (`g1-governed-mcp-route`) and
that route carries the `extAuthz` policy. There is no second route, no alternate listener, and
no config path that reaches `toy-mcp-backend-fixture` without the `extAuthz` policy attached.
No governed gateway->backend route bypassing AgentGate exists in this configuration.

## Files in this directory

| Path | What it is |
|---|---|
| `config/g1-agentgateway.yaml` | The real, reviewable agentgateway configuration for G1 (jwtAuth + extAuthz + route + backend placeholder). Annotated with every design decision and the one unresolved contract-mismatch finding (see below). |
| `config/jwks/` | Placeholder, non-secret dev JWKS fixture so the `jwtAuth.jwks.file` path resolves to something concrete. Not exercised by G1 (see "What is out of scope"). |
| `fixtures/*.json` | The required reusable fixture set (AG-GW-G1-05): ALLOW, DENY (x2, both DENY reasons), missing identity, unknown tool, malformed input (x2, both layers), evaluation error. Plain JSON, no code dependency — usable by QA/Security or anyone else directly. |
| `harness/` | An independent Go module (`gateway/harness`, no dependency on the `agentgate` module) that sends every fixture, over real HTTP, to a running `g1-mock-authz` and asserts the response and enforcement gate. This is the **verification harness** called for in the assignment brief — not a proxy, not MCP transport, not a production component. |
| `harness/cmd/g1report` | A human-readable CLI report over the same fixtures (`go run ./cmd/g1report`). |

## Real-binary verification (added 2026-09-12, once Docker became available)

The original version of this report said Docker Desktop's daemon was unreachable and the real
`agentgateway` binary had never been executed. That changed in a later session. This section
records what was actually done and found — **read it alongside**, not instead of, the original
"Docker / real-binary gap" narrative below, which is left intact for history.

**The real binary was pulled and run:**

```
$ docker pull ghcr.io/agentgateway/agentgateway:latest
Digest: sha256:bf2f339ef326d32def2aaeb44b1b4549801293c19b89e764a4228667d97d9896
```

**Validating the original config surfaced two real schema bugs**, not just an unverified design:

```
$ docker run --rm -v "$(pwd)/gateway/config:/etc/agentgateway" ghcr.io/agentgateway/agentgateway:latest \
    --validate-only -f /etc/agentgateway/g1-agentgateway.yaml
Error: gateways.default: unknown field `routes` at line 1 column 537
```

`routes` is a **top-level key, a sibling of `gateways`** in the real schema — not nested under
`gateways.<name>` as originally written (confirmed against
`https://agentgateway.dev/docs/standalone/main/configuration/security/external-authz/`'s own
minimal example: "Top-level keys are `gateways` and `routes`"). After moving it:

```
Error: routes[0]: unknown field `name` at line 1 column 537
```

Route and backend list items also do not accept a `name` field. After removing `name` from both
the route and the backend entry, and separately fixing the JWKS fixture (see below):

```
Configuration is valid!
```

**Both fixes are now applied directly to `config/g1-agentgateway.yaml`** (not left as a
separate scratch file) — the `routes` top-level placement and the removed `name` fields. A third,
smaller issue was found and fixed the same way: `config/jwks/dev-fixture-jwks.json` originally
had a `_comment` field alongside `keys`, which the real JWKS loader also rejects as unknown; the
file now contains only `{"keys": []}` (the explanation moved to `jwks/README.md`, where it already
lived).

**The binary was then actually started** (not just validated), with the mock's port referenced via
`host.docker.internal:8091`:

```
$ docker run -d --name g1-agentgateway -e AGENTGATE_MOCK_ADDR=host.docker.internal:8091 \
    -w /etc/agentgateway -v "$(pwd)/gateway/config:/etc/agentgateway" -p 3000:3000 \
    ghcr.io/agentgateway/agentgateway:latest -f /etc/agentgateway/g1-agentgateway.yaml

$ docker logs g1-agentgateway
...
2026-09-12T13:08:30.969952Z  info  app  serving UI at http://localhost:15000/ui
2026-09-12T13:08:30.969966Z  info  agent_core::readiness  Task 'agentgateway' complete (107.832667ms), marking server ready
2026-09-12T13:08:30.970044Z  info  management::hyper_helpers  listener established  address=127.0.0.1:15000 component="admin"
2026-09-12T13:08:30.970212Z  info  proxy::gateway  started bind  bind="bind/3000"
```

The process stayed up, bound `:3000`, and self-reported ready. **An unauthenticated request was
then sent to confirm JWT enforcement is real, not just configured:**

```
$ curl -s -o /dev/null -w "http_status=%{http_code}\n" http://localhost:3000/
http_status=401
```

`401`, with no ext_authz call ever reaching the mock (nothing appeared in the mock's own logs for
this request) — jwtAuth `strict` mode is genuinely enforced before ext_authz is ever consulted,
exactly matching the intended trust boundary. Separately, the admin UI (`:15000`) was confirmed
bound to `127.0.0.1`/`::1` only (inside the container, never published) — consistent with, not a
violation of, the "admin surface must not be externally reachable" invariant.

**What this resolves:** the real `agentgateway` binary genuinely runs, loads this exact config,
and enforces JWT authentication before anything downstream — the "was the binary ever executed"
question is closed, and the config file itself no longer contains any known schema error.

**What this does NOT resolve:** the "Contract-mismatch finding" below (agentgateway's documented
`extAuthz.protocol.http` fields have no way to construct the mock's bespoke JSON request body) is
an architecture question, not a Docker-availability question. Running the real binary with a
validly-signed JWT would still hit this exact wall the moment the request reached the
`extAuthz` policy — the fixture JWKS deliberately has zero keys (`{"keys": []}`), so no signed
token can validate against it anyway, and generating a real key pair + signed token was judged
out of scope for this verification pass (it would only reconfirm, empirically, something the
published agentgateway docs already establish structurally). This finding stands exactly as
originally reported, now with additional confidence: it is the *only* remaining gap, not one of
several unknowns.

## Docker / real-binary gap (original narrative, kept for history — see above for the update)

Verified at the start of this work and again now:

```
$ docker version
...
failed to connect to the docker API at npipe:////./pipe/dockerDesktopLinuxEngine: ...
```

Docker Desktop's daemon is not reachable in this sandbox. **The real `agentgateway` binary was
never executed.** Nothing in this deliverable claims otherwise. Consequences:

- `config/g1-agentgateway.yaml` is a *design artifact*, reviewed against agentgateway's
  published documentation (fetched live during this ticket — see "Schema verification"
  below), not a config that has been loaded by the real binary and observed to work.
- The proof of the request/response *semantics* (identity, tool, classification, arguments,
  ALLOW/DENY/error handling, fail-closed behavior) is real and executed — against the real
  mock (`agentgate/cmd/g1-mock-authz`, which wraps the real `decision.Engine` + real Cedar
  boundary) — but the proof that agentgateway itself, as configured, would produce exactly
  this request shape on the wire is **not** executed. See "Contract-mismatch finding" below
  for the specific, load-bearing reason this matters beyond "couldn't run Docker."

### Schema verification performed

`config/g1-agentgateway.yaml`'s `jwtAuth` and `extAuthz` blocks were checked against
agentgateway's own published docs, fetched during this ticket:

- `https://agentgateway.dev/docs/standalone/main/configuration/security/jwt-authn/`
- `https://agentgateway.dev/docs/standalone/main/configuration/security/external-authz/`

Confirmed schema facts used in the config: `jwtAuth.mode`/`issuer`/`audiences`/`jwks.file`
attach at the `gateways.<name>` level; `extAuthz.host` + `extAuthz.protocol.{grpc,http}`
attach at a route or backend's `policies`; the `http` protocol variant's documented fields are
`path`, `addRequestHeaders`, `includeResponseHeaders`, `redirect` (all CEL-expression based).
This matches `docs/TECH_STACK.md`'s independently-recorded correction that the JWKS field is
`jwks.file`/`jwks.remote.jwksPath`, not `jwks.url`.

### Contract-mismatch finding (reported per AG-GW-G1-02's collaboration rule, not silently worked around)

**Finding:** agentgateway's documented `extAuthz.protocol.http` shape (`path`,
`addRequestHeaders`, `includeResponseHeaders`, `redirect` — all CEL string expressions that
manipulate the *URL and headers* of the callout) has no documented field that assembles an
arbitrary JSON **request body** matching the mock's bespoke wire shape:

```json
{"execution_id": "...", "workspace_id": "...", "identity": {...}, "tool": {...},
 "classification": {...}, "arguments": {...}}
```

The `grpc` protocol variant follows the Envoy `ext_authz` proto family (`CheckRequest`/
`CheckResponse`, with generic HTTP-attribute fields like headers and raw request bytes), which
also does not natively speak the mock's specific JSON field names (`identity.agent_id`,
`classification.known`, etc.) — those are AgentGate's own domain vocabulary, not something
Envoy's `ext_authz` protocol or agentgateway's HTTP variant construct for you.

**Why this is expected, not a defect to paper over:** `GO_BACKEND_G1_CONTRACT.md`'s own known
limitation #2 already states "the mock has no JWT/agentgateway integration; Gateway/MCP must
call `POST /evaluate` directly with an already-constructed request." The real production
integration point is `internal/authz` (Day 3/8, not built), which is presumably where an
Envoy-`CheckRequest`-shaped (or agentgateway-HTTP-shaped) callout gets translated into a
`decision.Request` — most plausibly by having AgentGate's real `ext_authz` service parse the
included raw request body itself (via `internal/mcpreq`, per `docs/PROJECT_DEFINITION.md`'s
repo layout: "`internal/mcpreq/` parse JSON-RPC body -> `{id, method, tool, args}`"), not by
agentgateway's config DSL assembling AgentGate's bespoke JSON.

**What I did, given this:** I did **not** invent a gateway-side adapter/shim to bridge this gap
— that would mean writing translation logic outside the frozen contract's owning components,
exactly what the assignment brief forbids ("do not create a gateway-only authorization
contract that diverges from the frozen Go contract... stop and record the exact
ambiguity/mismatch"). Instead:

1. `config/g1-agentgateway.yaml` shows the honest, best-effort mapping (HTTP protocol, host
   pointed at the mock, a comment at the exact field explaining the gap) rather than a
   config that pretends to fully close it.
2. The verification harness (`gateway/harness`) proves the actual contract semantics —
   constructing the documented wire JSON directly and sending it over plain HTTP — standing in
   for "whatever the real `internal/authz` translation layer will eventually produce," exactly
   as `GO_BACKEND_G1_CONTRACT.md` anticipates for every caller of the G1 mock.

**This is an open item for Go Backend/Architect, not resolved here:** the concrete mechanism
by which agentgateway's `ext_authz` callout (gRPC `CheckRequest` or HTTP) becomes a
`decision.Request` is undesigned, symmetrically with how O-1 (downstream identity) is already
tracked as undesigned in `docs/SECURITY/PRODUCTION-INVARIANTS.md` §12.1. I did not silently
resolve it by picking a translation scheme.

## What is exercised, and how (AG-GW-G1-02, 03, 04, 05, 06)

### Reproduction — exact commands, from a clean checkout

```bash
# Terminal 1 — start the real, frozen G1 mock (wraps the real decision.Engine + Cedar boundary)
cd agentgate
go run ./cmd/g1-mock-authz
# listens on :8091 by default

# Terminal 2 — run the automated interoperability proof
cd gateway/harness
go test ./... -v

# Optional — human-readable report table over the same fixtures
cd gateway/harness
go run ./cmd/g1report
```

If the mock is not running, `go test` **skips** (with an explicit instruction to start it) —
it never silently reports success without a real HTTP round trip having happened.

### What was actually observed (this session, against the live mock)

All 8 fixtures (7 required-category cases + 1 bonus transport-level case) were sent to a real,
locally running `g1-mock-authz` process and every assertion passed. Representative output
(`go test ./... -v`, this session):

```
--- PASS: TestG1Scenarios/allow_reader_read           http=200 decision=ALLOW reason=policy_allow                policy_version=504c632d... backend_reached=true
--- PASS: TestG1Scenarios/deny_explicit_destructive    http=200 decision=DENY  reason=policy_deny                 policy_version=504c632d... backend_reached=false
--- PASS: TestG1Scenarios/deny_no_matching_policy      http=200 decision=DENY  reason=no_matching_policy          policy_version=504c632d... backend_reached=false
--- PASS: TestG1Scenarios/missing_identity             http=200 decision=DENY  reason=invalid_identity            policy_version=""        backend_reached=false
--- PASS: TestG1Scenarios/unknown_tool                 http=200 decision=DENY  reason=unknown_tool                policy_version=""        backend_reached=false
--- PASS: TestG1Scenarios/malformed_argument            http=200 decision=DENY  reason=malformed_request           policy_version=""        backend_reached=false
--- PASS: TestG1Scenarios/evaluation_error_broken_role  http=200 decision=DENY  reason=evaluation_error            policy_version=504c632d... backend_reached=false
--- PASS: TestG1Scenarios/transport_unknown_field       http=400 decision=""    reason=""                          policy_version=""        backend_reached=false
PASS
```

`WouldForwardToBackend` — the harness's enforcement gate, defined in `harness/scenario.go` as
"HTTP 200 AND decision == ALLOW, nothing else" — was `true` for exactly one fixture (the ALLOW
case) and `false` for all seven others, including the evaluation-error case. The test suite
additionally hard-fails (not just reports a mismatch) if any `evaluation_error`- or
non-200-category fixture ever produces `WouldForwardToBackend == true`.

### AG-GW-G1-03 — field-by-field comparison against the frozen contract

Fixture `01_allow_reader_read.json`'s `request` object was compared field-by-field against
`GO_BACKEND_G1_CONTRACT.md`'s AG-GO-G1-02 table:

| Contract field | Fixture value | Match |
|---|---|---|
| `execution_id` (required, non-empty) | `"exec-g1-0001"` | yes |
| `workspace_id` (required, non-empty) | `"ws-g1-demo"` | yes |
| `identity.agent_id` (required, non-empty) | `"agent-reader-01"` | yes |
| `identity.on_behalf_of` (optional, `""` = none) | `""` | yes |
| `identity.roles` (required, non-empty, no blanks) | `["reader"]` | yes |
| `tool.backend_id` (required, non-empty) | `"toy-mcp-backend"` | yes |
| `tool.name` (required, non-empty) | `"read_schedule"` | yes |
| `classification.known` (required) | `true` | yes |
| `classification.risk` (required when known) | `"read"` | yes |
| `arguments` (optional, per-key declared attributes) | `{}` (omitted content) | yes |

No undocumented field appears in any fixture. Every fixture uses only fields the contract
defines — there is no gateway-only field anywhere in this deliverable.

`05_unknown_tool.json` and `04_missing_identity.json` specifically prove the DoD requirement
"missing identity or unknown tool cannot become an implicit ALLOW": both assert
`decision: DENY` and `backend_reached: false` against the real mock, not merely by inspection.

### AG-GW-G1-04 — ALLOW/DENY/error enforcement, observed against the real mock

| Case | Fixture | Mock result | Harness enforcement gate |
|---|---|---|---|
| ALLOW | `01_allow_reader_read` | `ALLOW` / `policy_allow` | continuation permitted (`true`) |
| DENY (explicit forbid) | `02_deny_explicit_destructive` | `DENY` / `policy_deny` | continuation blocked (`false`) |
| DENY (no matching policy) | `03_deny_no_matching_policy` | `DENY` / `no_matching_policy` | continuation blocked (`false`) |
| ERROR (Cedar evaluation error) | `07_evaluation_error_broken_role` | `DENY` / `evaluation_error` | continuation blocked (`false`), never implicitly allowed |

### AG-GW-G1-05 — reusable fixtures

See `fixtures/README.md`. All six required categories plus two bonus cases (a second DENY
reason, and a transport-layer malformed case) are covered. Fixtures are plain JSON with no
harness-specific framework coupling, so QA/Security's own harness can load and replay them
directly.

### AG-GW-G1-06 — interoperability proof summary

All mandatory G1 scenarios ran, against the real mock, in this session, with the exact
commands documented above (reproducible from a clean checkout: only `go` is required, no
Docker, no network access, no secrets). No governed bypass path exists in the gateway
configuration (see "Bypass check" above). The one thing this proof does **not** cover is
agentgateway's own behavior — see "Docker / real-binary gap" and the recommendation in the
final report.

## What is explicitly out of scope here (not built, by design)

- A running MCP backend. Real MCP E2E is explicitly out of G1 scope
  (`00_G1_CHECKPOINT_REFERENCE.md`, "Explicitly out of scope").
- JWT claims-mapping / `internal/identity` (Day 3/G2 work, not built yet).
- Any code inside `agentgate/internal/decision`, `agentgate/internal/policy`,
  `agentgate/internal/mockauthz`, or `agentgate/internal/fixturepolicy` — all untouched, owned
  by Go Backend's frozen G1 contract.
- A custom MCP proxy of any kind. `harness/` does not parse or route MCP; it only sends the
  documented JSON body to `/evaluate` and reads back the documented JSON result.
