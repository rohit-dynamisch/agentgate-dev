# DevOps — G1 Environment Reference (Handoff)

**Ticket sequence:** `docs/PHASES/G1_WORKSTREAMS/05_DEVOPS_G1_DETAILED.md`
**Status:** Complete for G1. Written after AG-OPS-G1-01 through AG-OPS-G1-05 executed against
branch `g1/devops` (based on `g1/go-backend` commit `a6d11f2`).
**Audience:** Go Backend, Gateway/MCP, Frontend/UI, QA/Security — anyone who needs to reach the
G1 mock AgentGate without inspecting this workstream's machine.

This document, plus `deploy/g1/docker-compose.yml` and `deploy/g1/Dockerfile.mock-authz`, is the
complete environment/configuration artifact for G1. No other file or tribal knowledge should be
required to reproduce it.

---

## AG-OPS-G1-01 — Existing conventions inspected (inventory)

| Concern | Found | Disposition |
|---|---|---|
| Go build | `agentgate/go.mod` (module `github.com/Dynamisch-LLC/agentgate`, Go 1.26), zero external deps beyond `cedar-go` | Extend: use as-is, no Makefile exists or is needed for G1 |
| CI | `.github/workflows/ci.yml` — single `go` job, gated on `agentgate/go.mod` existing; runs gofmt/vet/test -race/golangci-lint/govulncheck | Extend later (integration-test job) — **not modified in this ticket**; G1 does not require a CI change, since the mock is already covered by the existing unconditional `go test -race ./...` step |
| Docker/compose files | None existed anywhere in the repository before this workstream | Create fresh: `deploy/g1/docker-compose.yml`, `deploy/g1/Dockerfile.mock-authz` |
| Gateway configuration | None exists yet — owned by the parallel Gateway/MCP (WS-B) stream, in progress in its own worktree | Not touched; referenced only, per this ticket's explicit instruction not to fabricate WS-B's output |
| Test services | None (no Postgres/SpiceDB/Keycloak wiring exists yet — those are Day 5+/WS-A scope, explicitly out of scope for G1 per `docs/PHASES/G1_WORKSTREAMS/00_G1_CHECKPOINT_REFERENCE.md`, "Explicitly out of scope") | Not created |
| The mock itself | `agentgate/cmd/g1-mock-authz` (built on `g1/go-backend`, commit `a6d11f2`) — a plain Go binary, `go run`-able, no Docker required | Package it (Dockerfile) without requiring Docker to consume it |

**Conclusion:** almost everything for G1 is created fresh (topology doc, compose file, mock
Dockerfile) because nothing like it existed; the one thing genuinely extended is the existing Go
build convention (`go build ./cmd/g1-mock-authz`), which required no changes to reuse.

**DoD met:** yes — topology and file list below were fixed before any file was created.

---

## AG-OPS-G1-02 — Reproducible G1 topology

```text
MCP Client   -- (WS-B stub/placeholder; not this workstream's artifact) -->
agentgateway -- (WS-B stub/placeholder; not this workstream's artifact) -->
mock AgentGate  (agentgate/cmd/g1-mock-authz — real, built and run by this workstream)
```

Only the **mock AgentGate** leg is real in this environment. `agentgateway` and an MCP client are
owned by the parallel Gateway/MCP (WS-B) workstream, executing in a separate isolated worktree;
their real artifacts were not available here. Per this task's explicit instruction, the topology
below documents *where* they plug in (as stubs/placeholders in `deploy/g1/docker-compose.yml`,
gated behind a `gateway-stub-unavailable` compose profile so they are never accidentally started)
rather than inventing their actual configuration.

### Services, ports, env vars (every one, explicit)

| Service | Real or stub? | Host:container port | Required env vars | Notes |
|---|---|---|---|---|
| `mock-authz` (`cmd/g1-mock-authz`) | **Real** | `8091:8091` (compose) / any free port directly (see below) | `G1_MOCK_AUTHZ_ADDR` (default `:8091` if unset) | Stateless HTTP/JSON service, `POST /evaluate` only |
| `agentgateway` | **Stub/placeholder — not WS-B's real output** | `3000` (MCP endpoint), `15000` (admin UI), per `docs/PROJECT_DEFINITION.md §2` | `AGENTGATEWAY_EXT_AUTHZ_TARGET` (documented placeholder pointing at `mock-authz:8091`) | Never started by default (compose profile gate) |
| `mcp-client` | **Stub/placeholder — not WS-B/QA's real output** | n/a | none defined | Never started by default (compose profile gate) |

No other service, port, or environment variable is part of the G1 topology. In particular:

- No database, no SpiceDB, no Keycloak — none is required to prove the G1 contract
  (`decision.Request`/`decision.Result` over the mock's HTTP wire), and none is started.
- No TLS/mTLS — out of scope for G1 (production hardening, not contract testing).
- No hidden developer-machine state: every variable above has an explicit default or is marked
  required; nothing is assumed to pre-exist on any given machine.

**DoD met:** yes — every service/port/dependency/env var above is explicit; no hidden state.

---

## AG-OPS-G1-03 — Packaging the mock

1. **Build via repository convention** — plain `go build`, no new tooling:
   ```bash
   cd agentgate
   go build -o <output-path> ./cmd/g1-mock-authz
   ```
   Verified working in this environment (see AG-OPS-G1-05 evidence below).

2. **Container packaging** — `deploy/g1/Dockerfile.mock-authz` was written (multi-stage
   `golang:1.26` build → `gcr.io/distroless/static-debian12:nonroot` runtime, non-root user,
   `EXPOSE 8091`). **This Dockerfile has not been built or run** — Docker Desktop's daemon was
   unreachable in this environment (re-verified; see Limitations). Its syntax has not been
   validated by an actual `docker build` for the same reason. It is provided as the reviewable
   target packaging, to be built the moment a working daemon is available.

3. **Service addressing** — `G1_MOCK_AUTHZ_ADDR` (env var, already implemented by the Go Backend
   stream in `cmd/g1-mock-authz/main.go`) controls the listen address; default `:8091`. This is
   the only addressing configuration the mock needs or accepts.

4. **Readiness behavior — gap identified, not silently patched:** the mock has **no dedicated
   health/readiness endpoint**. `GET /` (or any path other than `/evaluate`) returns `404`; any
   method other than `POST` on `/evaluate` returns `405`. There is no `/healthz`/`/readyz`. This
   workstream did **not** modify `cmd/g1-mock-authz` or `internal/mockauthz` to add one — that
   would be scope creep into the Go Backend stream's owned code
   (`docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md` names `internal/mockauthz` as
   Go-Backend-owned, and this workstream's brief says "note that as a gap for the Go Backend
   stream rather than silently modifying their code"). **Recorded gap for Go Backend:** add a
   trivial `GET /healthz` (200, no auth, no decision-core involvement) to `cmd/g1-mock-authz` if a
   container-orchestrated readiness probe is wanted later. Until then, the documented readiness
   substitute is: poll `GET /` (or any non-`/evaluate` path) and treat any HTTP response
   (currently `404`) as "process is up and routing requests" — this was the actual technique used
   in the AG-OPS-G1-05 smoke test below, and it is sufficient for deterministic startup ordering
   in a compose/process context (it proves the HTTP listener is bound and serving, which is what
   "readiness" means for a stateless, dependency-free service like this one).

5. **Isolation from anything resembling production config:** `deploy/g1/` is a new, clearly-named
   directory (`g1` in the path) containing only this mock's packaging; nothing under it is
   referenced by `cmd/agentgate`'s production path, and the Dockerfile/compose file both carry
   explicit "temporary G1 mock, never production" comments at the top, matching the source code's
   own existing comments in `cmd/g1-mock-authz/main.go` and `internal/mockauthz/handler.go`.

6. **Test-only assumptions, stated explicitly:**
   - The mock has no authentication of any kind on `/evaluate` — anyone who can reach the port can
     submit any request. This is correct and by design for a G1 contract-test fixture, and would be
     a severe defect in a production ext_authz service. It must never be deployed on a
     network-reachable host outside a contract-test context.
   - The mock loads a fixed, hardcoded fixture policy (`internal/fixturepolicy.CedarSource`) — it
     is not configurable at runtime and is not meant to be; G1 tests are written against the
     specific fixture identities/roles/risks documented in `GO_BACKEND_G1_CONTRACT.md`.
   - The mock is fully stateless — no database, no in-memory session, no request history. This is
     verified empirically in AG-OPS-G1-05 below (restart test).

**DoD met:** yes — Gateway and QA can reach the mock from the environment defined here (a plain
process on a documented port, or the documented-but-unbuilt container). The one thing NOT
achieved is proving container reachability, because no container was actually built or run (see
Limitations).

---

## AG-OPS-G1-04 — Frozen G1 configuration

This section is the single source of truth for configuring the G1 environment. It contains no
secrets because none are needed.

### Mock AgentGate endpoint (real, this workstream)

| Item | Value |
|---|---|
| Binary | `agentgate/cmd/g1-mock-authz` (built via `go build ./cmd/g1-mock-authz` from inside `agentgate/`) |
| Listen address | `:8091` by default; override with env var `G1_MOCK_AUTHZ_ADDR` (e.g. `":9000"`) |
| Endpoint | `POST /evaluate`, JSON body/response — full wire contract in `docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md`, AG-GO-G1-04 |
| Auth | **None.** Explicitly a G1-only characteristic (see AG-OPS-G1-03 point 6), not a production plan. |
| Readiness | No dedicated endpoint (documented gap, see AG-OPS-G1-03 point 4); use `GET /` (any HTTP response = process is up) as the practical substitute |

### Gateway ext_authz endpoint (planned/documented — WS-B's scope, referenced not fabricated)

Per `docs/PROJECT_DEFINITION.md §2` and `docs/PHASES/G1_WORKSTREAMS/02_GATEWAY_MCP_G1_DETAILED.md`
(AG-GW-G1-02), the real `agentgateway` would be configured with an `ext_authz` gRPC/HTTP callout
target pointed at the mock's address above (i.e., wherever `mock-authz` is reachable —
`localhost:8091` for a direct-process setup, `mock-authz:8091` inside the documented compose
network). **This workstream does not own and has not created that gateway configuration file** —
it is WS-B's deliverable. The `AGENTGATEWAY_EXT_AUTHZ_TARGET` variable in
`deploy/g1/docker-compose.yml` is a placeholder documenting where that wiring would go, not a real
gateway config.

### MCP test endpoint (WS-B/QA scope, referenced not fabricated)

Per the same WS-B ticket document, an MCP test client and instrumented backend are Gateway/MCP's
deliverables (`AG-GW-G1-05`, "reusable fixtures"). No such endpoint exists in this workstream's
output; none was fabricated.

### Test authentication assumptions

**There are none.** The mock has no authentication, no JWT validation, and no claims extraction —
by design, since that boundary (`internal/identity`, real `ext_authz`) is Day 3/8 Go Backend work,
not G1 scope (see `GO_BACKEND_G1_CONTRACT.md`'s "trust provenance" resolution). Every caller of the
mock is, by definition, an unauthenticated test/integration fixture. This is stated here plainly as
a **G1-only characteristic** — it must not be read as a production authentication design, and it
must not be carried forward past this checkpoint without an explicit decision.

### Every required environment variable

| Variable | Applies to | Default | Required? |
|---|---|---|---|
| `G1_MOCK_AUTHZ_ADDR` | `cmd/g1-mock-authz` | `:8091` | No — has a working default |
| `AGENTGATEWAY_EXT_AUTHZ_TARGET` | documented placeholder only (`deploy/g1/docker-compose.yml`) | `mock-authz:8091` | N/A — not a real config surface yet |

**No secrets exist in this environment and none were committed.** The mock requires no credentials,
API keys, tokens, or connection strings of any kind.

**DoD met:** yes — every value above is explicit; another developer or agent could configure this
from this document alone.

---

## AG-OPS-G1-05 — Clean-environment smoke test (direct-process substitute)

**Honest framing, stated plainly:** Docker Desktop's daemon was unreachable in this sandbox
(`docker version` connects as a client but fails to reach
`npipe:////./pipe/dockerDesktopLinuxEngine`; re-verified at the start of this ticket — see
Limitations). **No container was built or run.** What follows is a direct-OS-process smoke test —
a legitimate, verifiable substitute for a container-level clean-room run, but it is **not**
equivalent to one, and it is labeled as a substitute throughout this document and the final report.

### Exact commands, in order, as run in this environment

```bash
# 1. Fresh build (equivalent to "no undocumented manual step" — go build is the only step)
cd agentgate
go build -o <scratch-dir>/g1-mock-authz.exe ./cmd/g1-mock-authz

# 2. Start (run 1), redirecting stdout/stderr to log files, on a documented port
G1_MOCK_AUTHZ_ADDR=":8092" ./g1-mock-authz.exe > run1_stdout.log 2> run1_stderr.log &

# 3. Verify readiness — poll GET / until any HTTP response is received
curl -s -o /dev/null -w "%{http_code}" http://localhost:8092/
# -> READY after 1 try (HTTP 404 on GET /)

# 4. ALLOW case
curl -s -X POST http://localhost:8092/evaluate -H "Content-Type: application/json" -d '{
  "execution_id": "exec-allow-1", "workspace_id": "ws-1",
  "identity": {"agent_id": "agent-1", "on_behalf_of": "", "roles": ["reader"]},
  "tool": {"backend_id": "backend-a", "name": "read_schedule"},
  "classification": {"known": true, "risk": "read"}, "arguments": {}
}'

# 5. DENY case (no_matching_policy)
curl -s -X POST http://localhost:8092/evaluate -H "Content-Type: application/json" -d '{
  "execution_id": "exec-deny-1", "workspace_id": "ws-1",
  "identity": {"agent_id": "agent-1", "on_behalf_of": "", "roles": ["reader"]},
  "tool": {"backend_id": "backend-a", "name": "write_schedule"},
  "classification": {"known": true, "risk": "write"}, "arguments": {}
}'

# 5b. DENY case (explicit forbid), extra coverage
curl -s -X POST http://localhost:8092/evaluate -H "Content-Type: application/json" -d '{
  "execution_id": "exec-deny-2", "workspace_id": "ws-1",
  "identity": {"agent_id": "agent-1", "on_behalf_of": "", "roles": ["admin"]},
  "tool": {"backend_id": "backend-a", "name": "delete_all"},
  "classification": {"known": true, "risk": "destructive"}, "arguments": {}
}'

# 6. Negative/malformed cases (four, exceeding the "at least one" DoD)
#   6a. missing identity.agent_id -> decision-core DENY invalid_identity (HTTP 200)
#   6b. unknown top-level field -> transport HTTP 400 (contract-drift guard)
#   6c. invalid JSON body -> transport HTTP 400
#   6d. classification.known=false -> decision-core DENY unknown_tool (HTTP 200)
# (exact bodies in this ticket's execution log / repeatable from GO_BACKEND_G1_CONTRACT.md)

# 7. Collect diagnostic logs (already redirected in step 2)
cat run1_stdout.log
cat run1_stderr.log

# 8. Stop (find the real Windows PID, since MSYS bash's $! is a job-control
#    wrapper PID, not the actual process — verified via netstat/tasklist)
netstat -ano | grep ":8092" | grep LISTENING
taskkill //F //PID <pid-from-above>

# 9. Restart (run 2) on the same port, same command as step 2/3
G1_MOCK_AUTHZ_ADDR=":8092" ./g1-mock-authz.exe > run2_stdout.log 2> run2_stderr.log &
curl -s -o /dev/null -w "%{http_code}" http://localhost:8092/   # readiness again

# 10. Repeat the exact ALLOW/DENY/negative requests from steps 4-6 verbatim
#     against the freshly-started process and diff the responses byte-for-byte
#     against run 1's responses.

# 11. Stop run 2 the same way as step 8.
```

### Actual output captured (verbatim, this environment, this session)

**Startup (run 1), stdout:**
```
{"time":"2026-09-12T16:43:39.5169122+05:30","level":"INFO","msg":"g1-mock-authz starting","addr":":8092","note":"non-production G1 contract mock — see docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md"}
```
Readiness: `READY after 1 tries (HTTP 404 on GET /)`

**ALLOW:**
```json
{"decision":"ALLOW","reason":"policy_allow","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"exec-allow-1"}
```

**DENY (no_matching_policy):**
```json
{"decision":"DENY","reason":"no_matching_policy","message":"no policy matched this request; deny-by-default","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"exec-deny-1"}
```

**DENY (explicit forbid):**
```json
{"decision":"DENY","reason":"policy_deny","message":"an explicit policy denied this request","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"exec-deny-2"}
```

**Negative 1 (missing agent_id) — HTTP 200:**
```json
{"decision":"DENY","reason":"invalid_identity","message":"agent id is missing","execution_id":"exec-neg-1"}
```

**Negative 2 (unknown top-level field) — HTTP 400:**
```json
{"error":"request body could not be decoded as the evaluate-request contract: json: unknown field \"unexpected_field\""}
```

**Negative 3 (invalid JSON) — HTTP 400:**
```json
{"error":"request body could not be decoded as the evaluate-request contract: invalid character 'n' looking for beginning of object key string"}
```

**Negative 4 (unknown/unclassified tool) — HTTP 200:**
```json
{"decision":"DENY","reason":"unknown_tool","message":"tool is not classified","execution_id":"exec-neg-4"}
```

**Stop (run 1):** `taskkill //F //PID 28940` → `SUCCESS: The process with PID 28940 has been terminated.`
Post-stop: `netstat` shows port 8092 no longer LISTENING. `run1_stderr.log` was empty — clean exit,
no panic, no crash trace.

**Restart (run 2)**, same port, same command:
```
{"time":"2026-09-12T16:44:32.8809466+05:30","level":"INFO","msg":"g1-mock-authz starting","addr":":8092", ...}
```
Readiness: `READY after 1 tries (HTTP 404 on GET /)`.

**Repeat ALLOW (`exec-allow-1`) against run 2 — byte-identical to run 1:**
```json
{"decision":"ALLOW","reason":"policy_allow","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"exec-allow-1"}
```

**Repeat DENY (`exec-deny-1`) against run 2 — byte-identical to run 1:**
```json
{"decision":"DENY","reason":"no_matching_policy","message":"no policy matched this request; deny-by-default","policy_version":"504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19","execution_id":"exec-deny-1"}
```

**Repeat negative (`exec-neg-1`) against run 2 — byte-identical to run 1:**
```json
{"decision":"DENY","reason":"invalid_identity","message":"agent id is missing","execution_id":"exec-neg-1"}
```

**Conclusion: no hidden state.** Run 2 is a distinct OS process (different PID: 28940 → 14528),
started fresh, with no shared volume, file, or IPC channel between runs, and it produced
byte-identical decisions for identical inputs, including the same `policy_version` hash. This
confirms the mock's documented design (it is a pure function of `(fixture policy, request)` with
no session/database/cache) rather than merely asserting it. `run2_stderr.log` was empty after the
final stop as well.

**Automated contract tests also re-run clean in this environment** (belt-and-suspenders on top of
the manual smoke test above):
```bash
cd agentgate
go test ./internal/mockauthz/... -v   # 12/12 PASS, ok, 4.409s
```

**DoD met:** yes — every command above was actually executed in this session, in this order, from
a fresh `go build`, with no undocumented manual step. The one substitution made (direct process
instead of container) is disclosed, not hidden.

---

## AG-OPS-G1-06 — Handoff evidence summary

See the top-level final report (delivered alongside this document) for the consolidated
PASS/FAIL/CONDITIONAL recommendation. This document plus:

- `deploy/g1/docker-compose.yml` (documented, unexecuted target topology)
- `deploy/g1/Dockerfile.mock-authz` (documented, unbuilt target packaging)

...together are the complete AG-OPS-G1-06 handoff artifact. Go Backend, Gateway/MCP, and QA can
each reproduce the direct-process environment from AG-OPS-G1-05's command list alone, using only
this repository at commit `a6d11f2` or later on `g1/devops`.

**Explicitly out of scope and not attempted, per this checkpoint's brief:** production container
image hardening, Helm charts, SBOM generation, image signing, or any release pipeline. Those
remain Day 8-10 / WS-E production-readiness scope (see the 10-day plan), not G1.
