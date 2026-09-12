# AgentGate — Setup and Getting Started

**Audience:** anyone on the team setting up this repository locally for the first time.
**Repository state this guide matches:** post-G1 (authorization contract frozen) — see
`docs/DEVELOPMENT/CURRENT_STATUS.md` for the current checkpoint, and `docs/README.md` for full
navigation. This guide covers the `agentgate/` Go module in detail (§3-8); the `gateway/`,
`frontend/`, and `deploy/` trees each have their own README/setup notes at their root.

---

## 1. Prerequisites

| Tool | Version | Why |
|---|---|---|
| [Go](https://go.dev/dl/) | **1.26 or newer** | Matches `agentgate/go.mod` (`go 1.26`) and `docs/TECH_STACK.md`'s "Go 1.26+" |
| Git | any recent version | Clone/branch/commit |
| A C compiler (gcc/clang/MSVC-via-mingw) | any | Only needed to run `go test -race` locally — the race detector requires cgo. Not needed to build or run the app. |

Nothing else is required yet. The module currently has **zero external dependencies** — no
Docker, database, or other service is needed to build, run, or test it at this stage.

Optional, if you want to run the same static-analysis/vulnerability checks CI runs:

- [`golangci-lint`](https://golangci-lint.run/welcome/install/)
- [`govulncheck`](https://go.dev/blog/govulncheck) — no separate install needed, see §4.

## 2. Clone the repository

```bash
git clone git@github.com:rushi-dynmsh/agentgate-dev.git
cd agentgate-dev
```

(Use the HTTPS remote instead if you have not set up an SSH key with GitHub:
`https://github.com/rushi-dynmsh/agentgate-dev.git`.)

## 3. Repository layout (what you'll see)

```
agentgate-dev/
  CLAUDE.md                  coding-agent operating rules — read before making changes
  docs/                      canonical project documentation — start with docs/README.md
  .github/workflows/ci.yml   CI — runs the checks in §4 automatically on push/PR
  agentgate/                 the Go module — THE product code lives here
    go.mod                   module github.com/Dynamisch-LLC/agentgate, go 1.26
    cmd/agentgate/           production executable entry point (main.go)
    cmd/g1-mock-authz/       non-production JSON/HTTP wrapper around the decision core,
                             used for G1 cross-stream integration — never used by cmd/agentgate
    internal/config/         typed, validated startup configuration
    internal/logging/        structured (JSON) logging
    internal/httpserver/     health/readiness HTTP endpoints, graceful shutdown
    internal/decision/       the frozen authorization decision core (Request/Result, fail-closed)
    internal/policy/         the narrow Cedar boundary — only package that imports cedar-go
    internal/fixturepolicy/  shared canonical Cedar test/dev fixture
    internal/mockauthz/      JSON wire layer for cmd/g1-mock-authz
    internal/authz/          (boundary only — real ext_authz gRPC service, not implemented yet)
    internal/identity/       (boundary only — JWT claims-mapping, not implemented yet)
    internal/audit/          (boundary only — not implemented yet)
    qa/g1blackbox/           independent black-box test suite (imports no internal/* package)
  gateway/                   agentgateway configuration + an independent Go verification harness
  frontend/                  TypeScript models/parsing/state-machine for the frozen contract
                             (no production UI framework chosen yet)
  deploy/g1/                 Docker Compose topology for the G1 mock
```

All Go commands below are run **from inside `agentgate/`** — that's where `go.mod` lives, not
the repo root. `gateway/harness` is its own separate Go module (its own `go.mod`); `frontend/` is
an npm/Vitest project, not Go.

## 4. Build and test it

```bash
cd agentgate

# Formatting — must produce no output
gofmt -l .

# Static analysis
go vet ./...

# Build everything
go build ./...

# Run the test suite
go test ./...
```

Expected result: `gofmt -l .` prints nothing, and `go test ./...` reports `ok` for
`internal/config`, `internal/logging`, `internal/httpserver`, `internal/decision`,
`internal/policy`, `internal/mockauthz`, and `qa/g1blackbox` (the boundary-only packages —
`internal/authz`, `internal/identity`, `internal/audit`, and `internal/fixturepolicy`, which has
no tests of its own — report `[no test files]`; that's expected, not a failure).

Optional, matching CI more closely:

```bash
# Race detector — requires a C compiler (cgo). Skip if you don't have one installed;
# CI's ubuntu-latest runner always has gcc, so this still runs there either way.
go test -race ./...

# Linter (install golangci-lint first if you don't have it)
golangci-lint run --timeout=5m

# Known-vulnerability scan (downloads on first run, needs network)
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## 5. Run the application

```bash
cd agentgate
go run ./cmd/agentgate
```

By default it starts an HTTP server on `:8090` with structured JSON logs on stdout. In another
terminal, confirm it's alive:

```bash
curl -i http://localhost:8090/healthz   # liveness — always 200 once the process is up
curl -i http://localhost:8090/readyz    # readiness — 200 once the listener is bound
```

Stop it with `Ctrl+C` — it shuts down gracefully (in-flight requests are given up to the
configured shutdown timeout to finish before exit).

### Configuration

All configuration is via environment variables (prefix `AGENTGATE_`); everything has a sane
default, so none of this is required for a first run.

| Variable | Default | Meaning |
|---|---|---|
| `AGENTGATE_ENV` | `development` | Free-form deployment label, logged at startup |
| `AGENTGATE_HTTP_ADDR` | `:8090` | HTTP listen address |
| `AGENTGATE_LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `AGENTGATE_SHUTDOWN_TIMEOUT` | `10s` | Max time graceful shutdown waits for in-flight requests |

Example with overrides:

```bash
AGENTGATE_HTTP_ADDR=":9090" AGENTGATE_LOG_LEVEL="debug" go run ./cmd/agentgate
```

Invalid configuration (e.g. `AGENTGATE_LOG_LEVEL=verbose`, or a non-duration
`AGENTGATE_SHUTDOWN_TIMEOUT`) causes the process to fail fast on startup with a clear error on
stderr, rather than run in an undefined state.

## 6. What "working" looks like (smoke-test checklist)

Before you start making changes, confirm your setup is good:

- [ ] `cd agentgate && gofmt -l .` prints nothing
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] `go test ./...` — all listed packages show `ok`
- [ ] `go run ./cmd/agentgate` logs `"starting agentgate"` then `"http server listening"`
- [ ] `curl http://localhost:8090/healthz` → `200 ok`
- [ ] `curl http://localhost:8090/readyz` → `200 ready`
- [ ] `Ctrl+C` logs a shutdown message and the process exits (not just vanishes)

If any of these fail, see §7.

## 7. Troubleshooting

- **`go: go.mod file not found`** — you're running Go commands from the repo root instead of
  `agentgate/`. `cd agentgate` first.
- **`go test -race` fails with "requires cgo"** — you don't have a C compiler on `PATH`. This
  only affects the race-detector run; plain `go test ./...` is unaffected, and CI always runs
  `-race` on Linux where gcc is preinstalled. Install a compiler (e.g. `mingw-w64` on Windows,
  `build-essential` on Debian/Ubuntu, Xcode CLI tools on macOS) only if you specifically need to
  run the race detector locally.
- **Port already in use** — something else is on `:8090`; set
  `AGENTGATE_HTTP_ADDR=":<other-port>"`.
- **`govulncheck`/`golangci-lint` not found** — they're optional locally (CI runs them on every
  PR regardless); see the install links in §1 if you want to run them yourself.

## 8. Before you push

- Read `CLAUDE.md` (repo-wide AI/engineering rules) and `docs/DEVELOPMENT/AI_DEVELOPMENT_MODEL.md`.
- Check `docs/DEVELOPMENT/CURRENT_STATUS.md` for the current checkpoint and
  `docs/DECISIONS/OPEN_DECISIONS.md` for unresolved architectural questions — don't guess past
  them.
- Run the checklist in §6 (CI will run the equivalent checks on your PR either way).
- `docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md` describes how work is
  currently divided across workstreams/checkpoints.

## 9. Where to go next

- `docs/README.md` — the documentation map; start here if you're not sure where something lives.
- `docs/PROJECT_DEFINITION.md` — what AgentGate is and why (read this first if you're new).
- `docs/TECH_STACK.md` — the technology choices and rationale.
- `docs/DEVELOPMENT/CI_BASELINE.md` — exactly what CI checks and why.
- `docs/PHASES/archive/REPOSITORY_BASELINE.md` — the state of the repo before any code existed
  (historical).
