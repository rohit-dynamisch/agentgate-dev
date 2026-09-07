# AgentGate

**A checkpoint in the only path between an AI agent and its tools.**

[![CI](https://github.com/rushi-dynmsh/agentgate-dev/actions/workflows/ci.yml/badge.svg)](https://github.com/rushi-dynmsh/agentgate-dev/actions/workflows/ci.yml)
[![Go Reference](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go)](agentgate/go.mod)
[![Status](https://img.shields.io/badge/status-early%20development-orange)](docs/DEVELOPMENT/CURRENT_STATUS.md)
[![License](https://img.shields.io/badge/license-TBD-lightgrey)](#license)

---

> ⚠️ **Early development.** AgentGate is not usable yet. There is a Go service scaffold that
> starts, serves health checks, and shuts down cleanly — but no authorization decision, policy
> evaluation, identity resolution, or audit logging exists in this codebase yet. See
> [Project status](#project-status) before assuming anything below is implemented.

## What is AgentGate?

AI agents can now call real tools — delete records, send emails, move money, run code. Nothing
in most stacks checks, per individual tool call, whether that specific action should be allowed,
and nothing writes down why it was.

**AgentGate is a governance layer for AI agent tool calls.** It sits behind
[`agentgateway`](https://agentgateway.dev) (the MCP data-plane proxy) as the authorization
decision point:

```text
AI agent  --wants to call a tool-->  agentgateway  --ext_authz-->  AgentGate  --allow/deny-->  real tool
```

For every tool call it will:

1. **Resolve identity** — which agent, acting on behalf of which human, with what roles.
2. **Decide** — allow or deny, per call, from versioned [Cedar](https://www.cedarpolicy.com/)
   policy.
3. **Log** — every call, allowed or denied, with full context and the exact policy version that
   decided it.
4. **Evolve** — an operator reviews the log and adjusts policy; the next call obeys the new rule.

The agent cannot bypass it — it isn't a monitor on the side, it's a gate in the path, and it
requires **no changes to the agent, its framework, or its harness.**

For the full product rationale, non-goals, and the architectural reasoning behind it, read
**[`docs/PROJECT_DEFINITION.md`](docs/PROJECT_DEFINITION.md)** — start there before anything else
in this repository.

## Project status

This project is being developed in phases, tracked in
[`docs/DEVELOPMENT/CURRENT_STATUS.md`](docs/DEVELOPMENT/CURRENT_STATUS.md). As of now:

| Capability | Status |
|---|---|
| Go module, service scaffold, config, logging, health/readiness | ✅ Implemented |
| CI (format/vet/test/lint/vulnerability checks) | ✅ Implemented |
| Cedar policy evaluation | ⬜ Not implemented |
| Identity / claims resolution | ⬜ Not implemented |
| `agentgateway` ↔ AgentGate `ext_authz` integration | ⬜ Not implemented |
| Audit logging | ⬜ Not implemented |
| Policy governance UI, dry-run/rollback | ⬜ Not implemented |
| Downstream credential handling | ⬜ Not implemented — depends on an [open decision](docs/DECISIONS/OPEN_DECISIONS.md) |

Nothing here enforces anything yet. Do not deploy it expecting governance behavior.

## Quick start

Full instructions, prerequisites, and a smoke-test checklist are in
**[`docs/DEVELOPMENT/SETUP.md`](docs/DEVELOPMENT/SETUP.md)**. Short version:

```bash
git clone git@github.com:rushi-dynmsh/agentgate-dev.git
cd agentgate-dev/agentgate

go build ./...
go test ./...
go run ./cmd/agentgate
```

Then, in another terminal:

```bash
curl http://localhost:8090/healthz   # -> ok
curl http://localhost:8090/readyz    # -> ready
```

Stop it with `Ctrl+C`. See [`docs/DEVELOPMENT/SETUP.md`](docs/DEVELOPMENT/SETUP.md) for
configuration options, troubleshooting, and the checks CI runs on every PR.

## Repository layout

```text
agentgate-dev/
├── agentgate/                 the Go module — product code
│   ├── cmd/agentgate/         executable entry point
│   └── internal/
│       ├── config/            typed, validated startup configuration
│       ├── logging/            structured (JSON) logging
│       ├── httpserver/        health/readiness HTTP surface, graceful shutdown
│       ├── authz/             (boundary only) future ext_authz decision entry point
│       ├── identity/          (boundary only) future claims → identity resolution
│       ├── policy/            (boundary only) future Cedar policy loading/evaluation
│       └── audit/             (boundary only) future decision audit log
├── docs/                      canonical project documentation (see below)
└── .github/workflows/ci.yml   CI pipeline
```

## Documentation

All durable project context lives in [`docs/`](docs/), not in chat history or PR descriptions —
that's a deliberate project rule (see [`CLAUDE.md`](CLAUDE.md)). Start here:

| Document | What it covers |
|---|---|
| [`docs/PROJECT_DEFINITION.md`](docs/PROJECT_DEFINITION.md) | The single source of truth: product scope, architecture, what's explicitly *not* being built, decisions log |
| [`docs/TECH_STACK.md`](docs/TECH_STACK.md) | Technology choices and rationale |
| [`docs/DEVELOPMENT/MASTER_PLAN.md`](docs/DEVELOPMENT/MASTER_PLAN.md) | Phase-by-phase development plan |
| [`docs/DEVELOPMENT/CURRENT_STATUS.md`](docs/DEVELOPMENT/CURRENT_STATUS.md) | What's actually done right now |
| [`docs/DEVELOPMENT/SETUP.md`](docs/DEVELOPMENT/SETUP.md) | Local setup, build/test/run, troubleshooting |
| [`docs/DEVELOPMENT/CI_BASELINE.md`](docs/DEVELOPMENT/CI_BASELINE.md) | What CI checks, and what's deliberately deferred |
| [`docs/DECISIONS/OPEN_DECISIONS.md`](docs/DECISIONS/OPEN_DECISIONS.md) | Unresolved architectural/security questions — not guesses to fill in |
| [`docs/AI/AI_DEVELOPMENT_MODEL.md`](docs/AI/AI_DEVELOPMENT_MODEL.md) | How AI-assisted development is structured on this project |
| [`docs/PHASES/`](docs/PHASES/) | Per-phase task specifications |
| [`docs/TEAM/`](docs/TEAM/) | Team ownership per phase |

## Contributing

This project is developed primarily through AI-assisted engineering under bounded, documented
tasks — see [`CLAUDE.md`](CLAUDE.md) and
[`docs/AI/AI_DEVELOPMENT_MODEL.md`](docs/AI/AI_DEVELOPMENT_MODEL.md) for how work is scoped and
reviewed. If you're picking up a task:

1. Read the relevant docs in [`docs/`](docs/) before touching code.
2. Check [`docs/DEVELOPMENT/CURRENT_STATUS.md`](docs/DEVELOPMENT/CURRENT_STATUS.md) and
   [`docs/DECISIONS/OPEN_DECISIONS.md`](docs/DECISIONS/OPEN_DECISIONS.md) — don't reintroduce a
   rejected design or guess past an open decision.
3. Follow the setup and smoke-test steps in
   [`docs/DEVELOPMENT/SETUP.md`](docs/DEVELOPMENT/SETUP.md) before opening a PR.
4. Keep changes scoped to the task at hand; report deviations rather than expanding scope
   silently.

There is no public issue-triage process yet — this repository is not open for external
contributions at this stage.

## Security

AgentGate is, by design, a security-sensitive, in-path authorization system — see
[`docs/PROJECT_DEFINITION.md`](docs/PROJECT_DEFINITION.md) §2–§4 for the trust model as currently
designed. It has not been implemented or audited yet, and no security disclosure process exists
yet. Do not report vulnerabilities publicly; ask a project maintainer for a contact channel.

## License

**Not yet decided.** Licensing (Apache-2.0 vs. AGPL-3.0/BUSL) is an open, management-level
decision tracked as item O-4 in [`docs/TECH_STACK.md`](docs/TECH_STACK.md#4-decisions-still-open-block-parts-of-the-plan).
No license is granted for use of this code until that decision is made and a `LICENSE` file is
added.
