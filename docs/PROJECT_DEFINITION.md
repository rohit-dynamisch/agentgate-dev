# AgentGate — Project Definition

**Status:** Converged scope, ready for development
**Supersedes:** the v1 scope in `docs/AgentGate_Idea_Brief_By_Management_v0.1.md`,
`docs/AgentGate_Whitepaper (1).md`, and `docs/AgentGate_Technical_Architecture (1).md`
wherever they conflict with this document (see §9 — the conflicts are material).
**Companions:** `WALKTHROUGH.md` (how we reasoned to here),
`ARCHITECTURE_DECISIONS.md` (the constraint analysis behind the big calls).

This is the single source of truth for what we are building. It is written to be
read cold — by a new engineer or by an AI coding assistant — and to contain
enough context that no other document is required to start work.

---

## 0. How to use this document

This is the single source of truth for **AI-assisted development**. It is written
so that an assistant — or a new engineer — reading only this file has enough
context to write correct code without re-deriving decisions or reintroducing
rejected designs.

Three rules for anyone, human or AI, working from this document:

1. **§12 (Decisions log) is binding.** Each entry records something that was
   argued through and settled. Do not reintroduce a rejected option because it
   "seems natural" — the reason it was rejected is written down. If you believe a
   decision is wrong, change §12 explicitly and say so; do not quietly build
   against it.
2. **§1 ("What we are explicitly not") and §5 ("Deliberately dropped") are
   binding.** The most common failure mode for this project is scope creep back
   toward the original whitepaper's ambitions.
3. **§9 flags where we knowingly diverge from management's brief.** That
   divergence is real and unresolved at the management level. Do not present the
   divergent design as if it were approved.

---

## 1. The product in one page

### The problem

AI agents can now call real tools — delete records, send emails, move money, run
code. There is no layer that checks, per individual tool call, whether that
specific action should be allowed, and no record of why it was.

Nothing about "AI" is required to state the problem: replace "AI agent" with
"script" and it still holds. The problem is unattended software calling powerful
tools, with nobody checking each call or writing down what happened.

What makes it *urgent* for AI specifically: an agent's tool choices are driven by
text it ingested, which may be attacker-controlled (prompt injection). So the
blast radius of a manipulated agent is whatever its credentials permit — and
today that is usually "everything the service account can do."

### The mechanism

**AgentGate is a checkpoint in the only path between an agent and its tools.**

```
AI agent  --wants to call a tool-->  AgentGate  --if allowed-->  real tool
```

The agent cannot bypass it. Not a monitor on the side; a gate in the path.

### What we do at that checkpoint

1. **Resolve identity** — which agent, acting on behalf of which human, with what roles
2. **Decide** — allow or deny, per call, from versioned policy
3. **Log** — every call, allowed or denied, with full context and the exact policy version that decided it
4. **Evolve** — an operator reviews the log and adjusts policy; the next call obeys the new rule

That is the whole product. Everything else is implementation detail.

### Who it is for

The buyer is a **platform or security engineer** who has been handed an AI agent
that touches production systems and asked "how do we make sure it can't do
something catastrophic?" Adjacent stakeholders: an engineering lead who needs an
answer before shipping, and a compliance function that needs to answer "what did
the agent do, on whose authority?" after the fact.

They are not asking for a smarter AI. They want a control layer around the one
they already have.

### What we are explicitly not

- Not an LLM guardrails / content-moderation product (we do not inspect model output)
- Not a prompt-injection *detector* — we assume injection is possible and bound the damage
- Not an identity provider — we consume the customer's existing OIDC (Okta, Entra, Keycloak)
- Not an observability platform — we are enforcement + audit, not dashboards-for-everything
- Not a network proxy — that layer is commodity (`agentgateway`) and we do not compete with it

---

## 2. Architecture as decided

```
   AI agent (any MCP client — unmodified)
        │  MCP over HTTP + Bearer JWT
        ▼
   ┌──────────────────────────────────────────┐
   │ agentgateway  (off-the-shelf, Rust)      │   :3000  MCP endpoint
   │  • MCP transport, routing, tool discovery│   :15000 admin UI
   │  • validates JWT signature (jwtAuth)     │
   │  • calls out per request (ext_authz gRPC)│
   └───────────────┬──────────────────────────┘
                   │  "allow or deny?" + validated claims
                   ▼
   ┌──────────────────────────────────────────┐
   │ agentgate  (OUR PRODUCT, Go)             │   :9000 gRPC ext_authz
   │  • identity: read claims (no re-verify)  │   :8090 HTTP UI + API
   │  • policy:   Cedar role/risk decision    │
   │  • audit:    write every decision        │
   │  • operator UI: review log, edit policy  │
   └───────────────┬──────────────────────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
   Postgres              (optional) SpiceDB
   audit log +           relationship graph —
   policy versions       only for tools that
                         cannot self-authorize

   agentgateway --forwards on allow--> real MCP server(s)
```

### Division of labour — the load-bearing distinction

| Concern | Owner | Why |
|---|---|---|
| MCP protocol, transport, tool discovery | **agentgateway** | Linux Foundation project, backed by Microsoft/AWS/Cisco/IBM/Red Hat. We cannot out-engineer it and must not try. |
| JWT signature validation | **agentgateway** | One verification point. Configured via `jwtAuth` in `gateway/config.yaml`. |
| Identity → decision → audit | **agentgate (ours)** | This is the entire product. |
| Per-record ownership ("does alice own record 2") | **the tool/backend itself** | It is the system of record. See §4. |

**agentgate never speaks MCP.** It answers a separate internal gRPC question.
This is why it can be restarted, swapped, or reasoned about independently.

### How the agent sees it

To the agent, agentgateway *is* the MCP server. To the real MCP server,
agentgateway *is* the client. Standard reverse-proxy double-impersonation. The
agent needs **zero modification** — no SDK, no custom error handling, no
awareness that governance exists. This is a hard requirement, not a preference
(see §3).

### Tool discovery is free; governance is not

agentgateway auto-discovers the backend's tools via `tools/list` — we never
register tools by hand. Any spec-compliant MCP server works with only an address
in config.

But **discovery is not classification.** Nothing in MCP says "this tool deletes
data." Someone must tell agentgate which tools are risky. That is an operator
decision captured as config, and it can never be inferred.

---

## 3. The single most important decision: no real-time human-in-the-loop in v1

This reverses management's brief. It needs to be understood, not skimmed.

### The constraint

Real-time per-call human approval requires the agent to *wait* while a human
decides. But:

- **The model is stateless between turns.** It emits a tool call and stops. The
  thing that waits is the **execution harness** (Claude Desktop, LangGraph, a
  custom loop) — which we do not control and cannot standardise.
- **Approval time is unbounded** (minutes to hours). **HTTP connection lifetime
  is bounded** (seconds to minutes) by client libraries, load balancers, and
  proxies we also do not control.

These two facts cannot be reconciled. No gateway design closes the gap without
*something on the client side* knowing a request is parked and that it should
return later.

### What we proved in the POC

We built it. It works — with **our own client**. The mechanism: return `deny` to
the gateway, but with a JSON-RPC error body carrying
`{"code": -32003, "data": {"status": "pending_approval", "transaction_id": "..."}}`.
Our client reads that and polls.

A standard MCP client sees a JSON-RPC error and crashes, retries blindly, or
reports failure. So the POC's HITL is **real but non-portable** — it requires an
SDK in every agent, every framework, every language. That kills the zero-friction
adoption goal, which for an open-source product is *the* adoption goal.

Approaches evaluated and their verdicts are in `ARCHITECTURE_DECISIONS.md` §4.
Summary: agentgateway+SDK (works, needs client code), custom lightweight gateway
holding the connection (transparent, but the timeout wall is unmoved), MCP
progress notifications / durable-execution frameworks (promising, unproven,
non-universal).

### What we build instead: detective governance

**Policy decides instantly, every call. Everything is logged. A human reviews the
log and changes the rule, not the instance.**

This is the firewall / WAF / SELinux-permissive model:

```
AUDIT mode    all calls pass, everything logged, operator learns what agents do
              → used during onboarding and policy discovery

ENFORCE mode  policy strictly applied, denials final, operator reviews and
              evolves policy for next time
```

Why this is the right v1:
- **Zero client-side changes** — works with every harness, forever
- The binary `ext_authz` constraint stops being a limitation and becomes the design
- Policy becomes *empirical* — shaped by observed behaviour, not upfront guessing
- The audit log stops being a side feature and becomes the core product

### The honest limit, and how we cover it

For **irreversible** actions, detective governance is too late — the record is
already deleted. Our answer is **not** a real-time approval queue. It is:

> Irreversible operations are **denied by default**. An operator must explicitly
> grant that capability, per agent, in policy, **before deployment**. The human
> decision happens at configuration time, not execution time.

This is how production database access already works: you do not approve each
query, you grant a service account scoped permissions at deploy time and review
them periodically. It is arguably *stronger* than a runtime queue, because it
forces intentionality before the agent ever runs rather than reactively under
time pressure.

### Deferred, not abandoned

Team lead's decision: build detective governance first, then explore real-time
HITL once there is a base. Research directions are recorded in
`ARCHITECTURE_DECISIONS.md` §6 (MCP progress notifications, durable-execution
integration, pre-authorization, bounded SSE hold). **Do not build toward HITL in
v1.** Do not let the POC's parking code shape new design.

---

## 4. The second reversal: agentgate does not track resource ownership

### What the POC does

SpiceDB holds `alice owns record 1, 2`, seeded by hand at startup. A destructive
call must pass Cedar (role) **and** SpiceDB (ownership).

### Why that is wrong at scale

A real company's "who owns what" already lives in their CRM, their database,
their document store. Copying it into SpiceDB creates a **second source of truth
that must be kept in sync** — and every sync strategy is bad: one-time import
goes stale, webhook sync needs the customer's system to emit events it probably
does not, live query adds a network hop to every decision against an API most
systems do not expose.

### The correct model

**AgentGate governs tool calls and callers. It does not govern resources.**

Whether the `id` in a call belongs to the caller is the **tool's**
responsibility. AgentGate forwards the caller's identity; the tool does its own
per-record check using its own data, which it already has perfectly.

This is standard OAuth **on-behalf-of / token exchange** — the downstream service
enforces access using the original caller's identity. Our `on_behalf_of` claim
already existing means this fits what we built rather than fighting it.

### Consequence for the build

- **SpiceDB drops from required to optional.** Keep the code; do not make it a
  dependency; do not put it in the default compose stack for v1.
- Real enterprise systems (CRM, DB, document store) self-authorize natively —
  that is what they are built for.
- Tools too "dumb" to self-check are, by observation, tools not given access to
  anything valuable. The fallback case is narrow and low-stakes.

### Open security item

Naively forwarding the raw token gives every tool a credential broader than the
one action needs (wrong audience, replayable). The correct form is **RFC 8693
token exchange** — trade the incoming token for a short-lived one scoped to this
tool and this call. **Not built. Must be designed before we forward tokens.**

---

## 5. What is built, what is not, what is deliberately dropped

The POC is 8 phases, all committed, all verified running (`git log`: Phase 0 →
Phase 7).

### Built and production-quality

| Capability | Where |
|---|---|
| Cedar policy evaluation, deny-by-default | `agentgate/internal/policy/engine.go` |
| Identity from gateway-validated claims (agent + on-behalf-of + roles) | `agentgate/internal/identity/identity.go` |
| Postgres audit log — every decision, with content-hash policy version | `agentgate/internal/audit/audit.go` |
| JSON-RPC-shaped denials (HTTP 200 + error body, not raw 403) | `agentgate/internal/authz/server.go` |
| ext_authz gRPC server | `agentgate/internal/authz/server.go` |
| Read-only dashboard (audit / pending / relations, 3s poll) | `agentgate/internal/approval/dashboard.go` |
| Gateway JWT validation + ext_authz wiring | `gateway/config.yaml` |

### Built but being demoted

| Capability | Disposition |
|---|---|
| Approval parking + UI + TOCTOU re-check (`approval/`) | Works, but is real-time HITL. **Not v1.** Keep the code; remove from the default path. The TOCTOU re-check logic is still valuable — reuse it for policy-change re-evaluation. |
| SpiceDB ownership (`relations/`) | Per §4, optional. Not in the v1 default stack. |
| Slack notifications (`slack/`) | Tied to the approval flow. Dormant until HITL returns. |

### Not built — required for v1

| Gap | Note |
|---|---|
| **Policy stored in Postgres, versioned** | Today it is a file. Cedar does not care — it takes bytes. Only the loader changes. |
| **Policy governance UI** — review a denial, change the rule | This is the core differentiator. Nothing else matters if it does not exist. |
| **Dry-run + rollback on policy change** | See §6. The most persuasive feature in the product. |
| **Tool classification screen** — discovered tools, operator assigns risk | Unclassified tool ⇒ **deny by default**. Closes a live hole. |
| **Claims mapping as config, not code** | Today claim names are hardcoded to our Keycloak. Needs `claims-mapping.yaml`. |
| **Authentication on the admin UI** | Currently open on the network. Blocker for any real deployment. |
| **`workspace_id` on every table / object id** | An afternoon now; a migration nightmare later. See §7. |
| **Rate limiting / per-session quotas** | An agent in a retry loop makes 10,000 individually-allowed calls. No defence today. |
| **Single-binary quickstart** | Six containers is a brutal first-run for an OSS project. See §7. |
| **Audit retention / partitioning** | Unbounded growth. Bites at month three. |

### Deliberately dropped — do not re-add

- Real-time per-call human approval in v1 (§3)
- AgentGate-owned resource ownership graph as a required layer (§4)
- Building our own MCP proxy (agentgateway is commodity infrastructure)
- ML-based risk scoring (deterministic policy first; false negatives are worse than no classifier)
- Public/consumer 2FA-style approval flows (different product surface)

---

## 5a. Per-deployment configuration model

AgentGate can discover **facts** automatically (tool names, argument schemas,
token claims). It can never discover **intent** (what is risky, what is allowed,
what ownership means). Every deployment requires a human to state intent once;
enforcement is automatic thereafter.

The onboarding checklist:

| # | What the operator provides | Mechanism | Frequency |
|---|---|---|---|
| 1 | Their IdP's issuer + JWKS URL | `gateway/config.yaml` | Once |
| 2 | Where roles / user / agent-id live in their tokens | `claims-mapping.yaml` (§5, not built) | Once |
| 3 | Backend MCP server address(es) | `gateway/config.yaml` | Per backend |
| 4 | Risk classification per discovered tool | Tool inventory UI (§5, not built) | Per tool, then on drift |
| 5 | Which roles may call which tool classes | Policy editor (§6, not built) | Ongoing — this is the loop |
| 6 | Explicit grants for irreversible operations | Policy (§3) | Per capability, before deploy |

Items 1–3 are infrastructure wiring. Items 4–6 are the actual product.

---

## 6. The policy governance loop — the actual differentiator

Every call produces one audit row: who (agent + human + roles), what tool, what
arguments, what was decided, and the hash of the policy that decided it.

An operator reviews periodically — like firewall logs, not per-call:

- *"Denied 40× on `read_schedule`, but that is harmless"* → loosen the rule
- *"`send_email` was allowed, but should need a stricter role"* → tighten the rule

### Policy storage: keep Cedar, move the source of truth

Cedar is **not file-based** — the POC's *loader* is. Cedar builds a policy set
from bytes; those bytes can come from a Postgres row just as easily as a file.

```
policies table: id, workspace_id, version, source_text, hash,
                created_by, created_at, activated_at, state
```

The existing content-hash version scheme (chosen originally to handle
uncommitted policy edits) is exactly what a DB-backed store needs — versions are
content-addressed regardless of origin. Keep file loading as the dev/bootstrap
path.

**Why keep Cedar rather than Rego/OPA or a custom rules table:** Cedar is
deny-by-default *by construction* and statically analyzable. For a security
product whose failure mode is "we allowed something we should not have," that
property beats expressiveness. It is also an in-process Go library (no sidecar),
which matters for the availability story in §8. A custom rules table means
writing our own evaluator — a mistake with a long tail.

### Operators must not hand-write Cedar

Store **structured rules** as rows and **compile them to Cedar**:

```
Operator edits:  role=reader → tools where risk=read
                 role=admin  → tools where risk ∈ {read, write}

Compiler emits:  permit(principal in Role::"reader", action, resource)
                 when { resource.risk == "read" };

Generated Cedar is stored, hashed, versioned, shown read-only in the UI.
A raw-Cedar escape hatch covers rules the builder cannot express.
```

UI-editable data + Cedar's semantics + a human-readable artifact for audit — all
three. Same pattern as Cloudflare WAF (rule builder plus expression editor).

### Dry-run before activation — build this

A policy change is a production change with blast radius. Before activating,
replay the candidate policy against the last N logged decisions:

> *"This change affects 12 of the last 500 decisions: 9 previously-denied calls
> would now be allowed (listed), 3 previously-allowed would now be denied
> (listed)."*

Cheap to build — we already store full request context on every row. Paired with
instant rollback (activate previous version by id), it turns "edit and hope" into
"see the effect before it ships." This is the single most compelling thing in the
product, and it exists *only because* we log full context on every decision.

### Policy mutation is a critical attack surface

Changing policy is as powerful as policy itself. Required before this ships:

- [ ] Authentication on who may change policy
- [ ] Audit trail of every policy change (who, when, what, which denial triggered it)
- [ ] Ideally a second reviewer, distinct from whoever saw the denial
- [ ] Rate limiting on policy evolution (resist social-engineered loosening)
- [ ] Policy version recorded on every decision (**already done**)

---

## 7. Form factor, licensing, and tenancy

Management requires **open source**. That decision drives everything here.

### Single-tenant container, tenant-aware schema

"Container **or** multi-tenant hybrid" is a false dichotomy — multi-tenancy is a
feature flag on a data model, not a form factor.

**Build a single-tenant container. Put `workspace_id` in every table and SpiceDB
object id from day one, defaulting to `default`.**

An OSS adopter has one team, one environment. Multi-tenancy is pure cost to them.
But retrofitting a tenant key into a 40M-row audit table later is genuinely
painful, while adding a column that is always `default` costs an afternoon.
This is the Grafana / Gitea / Authentik / Infisical path: single-tenant OSS core,
tenancy latent in the schema, hosted offering later on the same codebase.

### Three decisions to make now (hard to reverse)

**License.** Apache-2.0 maximises adoption and lets anyone host it. AGPL-3.0 (or
BUSL) preserves our ability to run the only viable hosted version. Decide before
external contributions arrive — relicensing later means chasing every
contributor.

**The one-liner.** Six containers (Keycloak, Postgres, SpiceDB, gateway,
agentgate, tools) is a brutal first-run, and OSS adoption dies on setup friction
more than on missing features. Needs a genuine one-command mode: single binary,
SQLite instead of Postgres, no SpiceDB, accept any OIDC issuer instead of
bundling Keycloak. Every dependency **optional and upgradeable**. The codebase
already treats SpiceDB and Slack as optional — extend that discipline to Postgres
and the IdP.

**Core vs commercial boundary.** Decide before writing the policy UI. Suggested:
engine, policy loop, audit log, single-workspace UI = core. Admin SSO,
multi-workspace, retention/archival, compliance reporting = commercial edge.

### Eventual shapes

| Form | Target |
|---|---|
| Self-hosted container (**v1**) | Enterprises wanting on-prem governance |
| Embedded Go library | Teams integrating governance into an existing service |
| Managed SaaS | Teams that do not want to operate infrastructure |

---

## 8. Remaining open problems

Ranked by how badly each bites.

**Tool risk classification.** Name-prefix heuristics (`delete_`) die on contact
with reality — `archive_customer` is destructive, `delete_temp_cache` is not.
Needs explicit per-tool metadata plus **deny-by-default on unknown tools** and an
operator alert on inventory drift (a backend adding a tool tomorrow is silently
callable today).

**Argument-level authorization.** We hardcode extracting `id`. Real tools:
`send_email(to, body)` cares about `to`; `run_query(sql)` cares enormously;
`transfer(from, to, amount)` cares about all three plus a threshold. Needs a
per-tool declaration of which arguments are policy inputs, or every new tool is a
code change.

**No response-side governance.** All enforcement is request-side. An allowed
`read_record` may return PII that lands in the model's context and then anywhere.
Exfiltration-via-allowed-reads is genuinely AI-native — traditional API consumers
do not have a context window that gets summarised into a chat message. This is
both a gap and an unclaimed differentiator.

**No aggregate or session-level limits.** Every call in a 10,000-call retry loop
is individually, correctly allowed. MCP gives us a session id — thread it into
audit and policy context so "this agent has already read 200 customer records this
run" can inform a decision. Per-call context-free decisions are a structural
blind spot.

**Multi-backend, and credentials underneath it.** Real deployments front 5–20 MCP
servers. agentgateway multiplexes and auto-prefixes tool names, so the routing is
handled — but **agentgate would hold credentials to every tool it fronts.** That
is a high-value secrets store and will come up in every security review. Needs an
external secret-manager story.

**Availability and degraded mode.** agentgate down = every agent down.
Fail-closed is correct for security and awful for uptime; own the tradeoff
explicitly. Cedar being in-process is an asset (role checks survive a Postgres
outage). Decide and make configurable: SpiceDB unreachable → deny ownership-gated
calls (current) or fall back to role-only for reads? Postgres down → refuse to
serve, or serve and buffer audit to disk?

**Ownership-data provenance for the narrow fallback case.** §4 relocates this to
the tool for real systems. For the residual "dumb tool" case, the sync problem is
unresolved (import goes stale / webhooks need customer support / live query needs
an API that rarely exists). Acceptable because the case is narrow and low-stakes —
but do not pretend it is solved.

**EU AI Act export.** Management's v1 item 5. Not built and not analysed. The
audit log has the right shape for it (Article 12 wants "which rules were in
effect" — our policy-version hash answers exactly that), so this is a query +
formatting layer, not new plumbing. But it is unstarted, and §9 explains why that
now matters more.

---

## 9. Divergence from management's brief — read this before planning

Management's v1 scope had five items. We have moved away from two of them, and a
third is unstarted. This is a deliberate, documented engineering conclusion, not
drift — but it has not yet been agreed with management, and it should be.

| Management v1 item | Status |
|---|---|
| 1. MCP proxy/SDK attaching identity + OBO | **Aligned.** Done via agentgateway + our identity layer. |
| 2. Per-action policy check: allow / deny / **require-approval** | **Partial.** Allow/deny shipped. `require-approval` dropped from v1 (§3). |
| 3. **Human approval in Slack/Teams, resolving in seconds** | **Reversed.** Not achievable portably across agent harnesses. Deferred (§3). |
| 4. Immutable audit log + **ReBAC lineage graph** | **Partial.** Audit log is solid. ReBAC ownership demoted to optional (§4). |
| 5. One compliance export mapped to EU AI Act | **Unstarted** (§8). |

### Why this matters strategically

The whitepaper (§10) names our defensible wedge as three things: (a) a genuine
relationship/lineage graph, (b) first-class human approval with state
re-verification, (c) audit mapped explicitly to regulatory articles.

Our evolution has **demoted (a), deferred (b), and left (c) unbuilt.** All three
legs of the stated competitive differentiation are currently weakened. The
whitepaper also positions HITL as "a literal implementation of EU AI Act Article
14" — a claim we can no longer make in real time.

### What we should say instead

The honest, still-strong pitch: **the policy governance loop with dry-run,
rollback, and complete decision provenance** (§6). That is real, buildable,
differentiated, and nobody in the MCP space has it. It is a *different* wedge
from the one the whitepaper claims — better in some ways (deployable, zero client
friction, no unsolvable dependency), weaker in others (no real-time Article 14
story).

Two things follow, and both are decisions for management, not for engineering:

1. Confirm the HITL deferral is accepted, and that the Article 14 story becomes
   "policy-time human oversight + full auditability" rather than "real-time
   approval."
2. Decide whether the EU AI Act export moves up in priority, since it is now the
   only remaining leg of the original three-part wedge.

**Do not resolve these by building. Raise them.**

---

## 10. Build order

1. **Policy storage → Postgres, versioned.** Unblocks everything. Small change —
   Cedar never cared about files.
2. **`workspace_id` everywhere.** An afternoon now.
3. **Tool classification + deny-unknown-by-default.** Closes a live hole.
4. **Policy governance UI with dry-run and rollback.** The differentiator. The
   most persuasive moment in the product.
5. **Claims mapping as config** (`claims-mapping.yaml`) — unhardcode identity.
6. **Authentication on the admin UI.** Blocker for any real deployment.
7. **Single-binary quickstart** (SQLite, no SpiceDB, no bundled Keycloak).
8. **Demote approval/relations/slack** out of the default path — keep the code.

Nothing here depends on resolving real-time HITL. That is the main practical
benefit of the direction chosen: the entire v1 is buildable without first solving
the one problem that has no clean solution.

---

## 11. Repo, stack, conventions

```
agentgate/            OUR PRODUCT — Go
  cmd/agentgate/      entry point: wires everything, starts :9000 gRPC + :8090 HTTP
  internal/authz/     ext_authz gRPC server — the decision entry point
  internal/identity/  claims → {agent, on_behalf_of, roles}  [claim names hardcoded — to fix]
  internal/policy/    Cedar engine + tests. Version = SHA-256 of policy bytes
  internal/audit/     Postgres writer + reader (pgx/v5, retry on cold start)
  internal/mcpreq/    parse JSON-RPC body → {id, method, tool, args}
  internal/approval/  parking + UI + dashboard + executor  [demote: HITL, not v1]
  internal/relations/ SpiceDB ownership                    [demote: optional, §4]
  internal/slack/     approval notifications               [dormant with HITL]

gateway/config.yaml   agentgateway config: jwtAuth + extAuthz + mcp backend
policies/*.cedar      role rules (→ moving to Postgres, §6)
migrations/*.sql      001 audit_log, 002 pending_approvals
keycloak/*.json       demo realm: alice (admin), bob (reader)
toy-server/           deliberately dumb demo MCP server (Go, mark3labs/mcp-go)
client/               CLI standing in for an agent; -user does OAuth password grant
docs/                 management brief, whitepaper, technical architecture (historical)
archieve/             superseded POC guides (execution/testing/demo)
```

**Stack:** Go 1.26 · cedar-go v1.8.0 · authzed-go v1.10.0 · pgx/v5 ·
mark3labs/mcp-go v0.32.0 · Postgres 16 · Keycloak 26.3 · agentgateway (upstream image)

**Conventions**
- Beginner-friendly comments — the team is new to Go, MCP, Cedar, SpiceDB
- One meaningful commit per phase/unit of work, not per file
- Policies and config are **mounted files**, never baked into images
- Fail closed: any error in the decision path denies

**Environment gotchas (real, encountered)**
- Recreating the `agentgate` container gives it a new IP; agentgateway holds the
  stale gRPC connection and 403s everything. Fix: `docker compose restart agentgateway`.
- Windows App Control blocks locally-built test binaries. Run tests in Docker:
  `docker run --rm -v "${PWD}:/repo" -w //repo/agentgate -e GOFLAGS=-buildvcs=false golang:1.26-alpine go test ./...`
- PowerShell 5.1 mangles multi-line git commit messages. Use `git commit -F <file>`.
- Postgres `initdb.d` migrations run **only** on first volume init. Later ones:
  `docker exec -i agentgate-postgres psql -U agentgate -f /docker-entrypoint-initdb.d/<file>.sql`
- Toy server data is in-memory — `docker compose restart toy-mcp-server` restores records.
- SpiceDB dev mode is in-memory — graph resets on restart; agentgate re-seeds on reconnect.

---

## 12. Decisions log

| Decision | Chosen | Rejected | Reason |
|---|---|---|---|
| Build our own MCP proxy? | No — use agentgateway | Custom Rust/Go proxy | LF-governed, corporate-backed commodity. Unwinnable, and not the product. |
| JWT validation location | agentgateway (`jwtAuth` strict) | Re-verify in agentgate | Single verification point; agentgate trusts the only path in. |
| Policy engine | Cedar | Rego/OPA, custom rules table | Deny-by-default by construction, statically analyzable, in-process. Rego makes accidental permissiveness easy. |
| Policy version id | SHA-256 of policy bytes | Git commit hash | Content-addressed — works for uncommitted edits and DB-stored policy alike. |
| Policy source of truth | Postgres, versioned (**to build**) | Flat file (current) | Needed for UI editing, dry-run, rollback. Cedar takes bytes; only the loader changes. |
| Operator policy authoring | Structured rules compiled to Cedar | Raw Cedar in a textarea | Operators must not hand-write policy syntax. |
| Denial wire format | JSON-RPC error, HTTP 200, code -32003 | HTTP 403 | MCP clients parse JSON-RPC; a raw 403 breaks client libraries. |
| Human-in-the-loop | Detective governance; policy-time approval | Real-time per-call queue | Harness cooperation cannot be guaranteed; unbounded approval time vs bounded connection life (§3). |
| Irreversible actions | Deny by default; explicit pre-deployment grant | Runtime approval queue | Moves the human decision to config time, where it is unconstrained. |
| Resource ownership | The tool authorizes its own records | AgentGate mirrors ownership in SpiceDB | Avoids a second source of truth and an unsolvable sync problem (§4). |
| SpiceDB | Optional component | Required layer | Only needed for tools that cannot self-authorize — a narrow, low-stakes set. |
| Tenancy | Single-tenant runtime, tenant-aware schema | Multi-tenant v1 | OSS adopters are single-tenant; the schema seam is cheap now, expensive later. |
| SpiceDB consistency (when used) | `fully_consistent` | `minimize_latency` | A re-check must see a revocation that already committed. |

---

## 13. Summary — the shortest accurate description

> AgentGate is an open-source governance layer for AI agent tool calls. It runs
> behind `agentgateway`, and for every tool call it decides — from a versioned
> Cedar policy, using the caller's verified identity — whether the call is
> permitted, then writes a permanent record of the decision and the exact policy
> that produced it. Operators review that record and evolve policy from observed
> behaviour, with a dry-run showing the effect of a change before it ships.
> Resource-level permissions stay with the tools that own the data; irreversible
> operations require an explicit grant configured before deployment. It requires
> no changes to the agent, its framework, or its harness.

What makes it defensible: **AI-native identity** (agent + human-on-behalf-of +
roles as one model), **zero client-side integration** (works with every harness
today, unlike SDK-based rivals), **policy evolution with dry-run** (no MCP
gateway offers this), and **complete decision provenance** (every call answerable
as a query, including which policy version decided it).
