# AgentGate — Technology Stack Plan

**Date:** 2026-08-20
**Status:** Proposal for review. Nothing here is committed until we agree on it.
**Companion:** [`PROJECT_DEFINITION.md`](PROJECT_DEFINITION.md) — what we are building and why.
This document is *what we build it with*.

> The POC is a throwaway demo. This plan is for the production product built from scratch.
> Where a choice carries over from the POC, it is because it was independently the right
> choice — not because the code exists.

---

## 0. Facts this plan is built on (verified, Aug 2026)

Three external facts were verified against primary sources and they shape the stack. They are
stated up front because several stack choices follow directly from them.

1. **Current MCP spec is `2026-07-28` and it is stateless.** No `initialize` handshake, no
   protocol-level sessions, no `Mcp-Session-Id`. Every POST carries required routing headers
   (`Mcp-Method`, `Mcp-Name`) and servers **must** reject header/body mismatch. Tool metadata
   includes `ToolAnnotations` (`readOnlyHint`, `destructiveHint`, …) — but these are normatively
   **untrusted hints**, not authority.
   *Source: modelcontextprotocol.io/specification/2026-07-28.*

2. **MCP forbids token passthrough.** A server/proxy **must not** forward the token it received
   to a downstream server, and **must** validate that it is itself the token's audience. The
   sanctioned way to carry "agent acting for user Alice" through a shared client is the
   **Enterprise-Managed Authorization** extension (ID-JAG, an Identity Assertion JWT). RFC 8693
   token exchange is *not* referenced by the spec. This overrides the RFC-8693 assumption in the
   project definition's open questions.
   *Source: MCP authorization spec + security best practices, 2026-07-28.*

3. **agentgateway is real, Apache-2.0, and current (v1.4.1, Jul 2026).** AAIF/Linux-Foundation
   governed, backed by AWS, Cisco, IBM, Microsoft, Red Hat and others. It supports the
   `2026-07-28` spec, `ext_authz` (gRPC, with request-body inclusion), and `jwtAuth`. Two
   corrections to POC-era assumptions: the JWKS field is **`jwks.file`** (standalone) /
   **`jwks.remote.jwksPath` + `backendRef`** (Kubernetes), *not* `jwks.url`; and it carries **no
   maintainer production-readiness claim** ("active development"). "Progressive Disclosure" is a
   Solo **commercial** feature, not in the OSS project.
   *Source: agentgateway.dev docs, GitHub releases, LF press releases.*

---

## 1. The stack at a glance

| Layer | Choice | License | Why (one line) |
|---|---|---|---|
| **Language** | Go 1.26+ | — | Single static binary, in-process Cedar, strong concurrency, the ecosystem for this |
| **MCP proxy / data plane** | agentgateway (OSS) | Apache-2.0 | AAIF commodity infra; do not rebuild (PROJECT_DEFINITION D-01) |
| **Target protocol** | MCP `2026-07-28` | — | Current stable; stateless; header-routable |
| **Policy engine** | Cedar via `cedar-go` | Apache-2.0 | Deny-by-default, statically analyzable, embeddable, readable |
| **Policy storage** | PostgreSQL (versioned rows) | PostgreSQL | Source of truth for policy; enables dry-run/rollback |
| **Audit store** | PostgreSQL, hash-chained | PostgreSQL | Tamper-evident append-only decision log |
| **DB driver / pool** | `pgx/v5` | MIT | The standard high-performance Postgres client for Go |
| **Migrations** | `golang-migrate` **or** `goose` | MIT | Versioned, repeatable schema; POC had none |
| **Identity (dev)** | Keycloak 26.x | Apache-2.0 | Full-featured OIDC for local/testing; supports ID-JAG flows |
| **Identity (prod)** | Customer's OIDC IdP | — | We consume, never issue (D-04). Okta / Entra / Auth0 / Keycloak |
| **On-behalf-of** | MCP Enterprise-Managed Auth (ID-JAG) | — | Spec-sanctioned per-user identity through a shared client |
| **Relationships (optional)** | SpiceDB | Apache-2.0 | Only for tools that cannot self-authorize (D-06) |
| **Config** | env + typed validated struct | — | 12-factor; fail-fast on bad config; no secret defaults |
| **Observability** | OpenTelemetry (traces + metrics), `log/slog` | Apache-2.0 / — | Vendor-neutral; agentgateway already emits OTel |
| **Transport security** | mTLS (gateway↔agentgate), TLS (DB, SpiceDB) | — | The decision channel is the highest-value link |
| **Packaging** | Distroless container + Helm (OCI) | — | Non-root, minimal surface; agentgateway ships Helm/CRDs |
| **CI/CD** | GitHub Actions + `golangci-lint` + `govulncheck` | — | Test/lint/scan on every PR |
| **Supply chain** | SBOM (CycloneDX) + cosign-signed images | Apache-2.0 | Table stakes for a security product |

---

## 2. Component-by-component rationale

### 2.1 Language — Go 1.26+

Keep Go. It is genuinely the right language for this, independent of the POC:
- **`cedar-go` is a Go library** — the policy engine runs in-process, no sidecar, no extra network
  hop, and role checks keep working even if Postgres is down.
- Single static binary → trivial container, trivial single-binary quickstart (a project goal).
- First-class gRPC (the `ext_authz` interface) and strong OpenTelemetry support.
- The MCP/agent infra ecosystem (agentgateway itself, most MCP tooling) is Go/Rust — we stay
  close to it.

*Not chosen:* Rust (agentgateway's language) — we are not writing a proxy, so its main advantage
does not apply, and it would slow a small team. Python/Node — weaker for a latency-sensitive
in-path decision service.

### 2.2 MCP proxy — agentgateway (unchanged from D-01, now verified)

We do **not** build a proxy. agentgateway handles transport, routing, TLS, JWT signature
validation, tool discovery, and multiplexing. Confirmed capabilities we depend on:
- `ext_authz` gRPC callout per request, with request body included → our decision hook.
- `jwtAuth` (`mode: strict`) → signature validation before we are called (D-04). **Use
  `jwks.file` or `jwks.remote`, not `jwks.url`.**
- Auto-discovery of backend tools via `tools/list`; `prefixMode` multiplexing for multi-backend.
- Native OpenTelemetry / Prometheus / Grafana / Jaeger integration.

**Two risks to manage, not ignore:**
- *No published production-readiness claim and no HA/scaling guide.* We must do our own load and
  failover testing rather than assume it.
- *Timeout behaviour for long-held requests is undocumented.* This bounds any future real-time
  HITL work (D-07) and must be measured empirically before relying on it. For v1's detective
  model it does not matter (decisions are immediate).

### 2.3 Policy — Cedar (`cedar-go`), stored in Postgres

Cedar stays (D-02): deny-by-default by construction, statically analyzable, embeddable, readable
by non-engineers. The change from the POC is **where policy lives**:

- **Source of truth = a versioned Postgres table**, not a file. Cedar builds its policy set from
  bytes in memory; the bytes can come from a row as easily as a file.
  ```
  policies(id, workspace_id, version, source_text, hash,
           created_by, created_at, activated_at, state)
  ```
- The content-hash version scheme carries over — it is what a DB-backed, dry-runnable,
  rollbackable store needs.
- **Operators never hand-write Cedar.** A structured rule builder compiles to Cedar text; the
  generated policy is stored, hashed, and shown read-only, with a raw-Cedar escape hatch.
- **A file loader stays as the dev/bootstrap path** and for the single-binary quickstart.

*Cedar version and its policy-analysis tooling (for proving one policy version is more/less
permissive than another — directly useful for the dry-run feature) are still being confirmed;
this section will be pinned to specific versions once that lands.*

### 2.4 Identity — consume OIDC; on-behalf-of via ID-JAG (revised)

Unchanged: we are **not** an identity provider (D-04). agentgateway validates JWT signatures; we
read claims via a **per-deployment `claims-mapping.yaml`** (never hardcoded — the POC's mistake).

**Revised by verification:** the on-behalf-of problem has a spec-sanctioned answer that is *not*
RFC 8693. The MCP **Enterprise-Managed Authorization** extension issues an **ID-JAG** (Identity
Assertion JWT) carrying the actual employee's `sub`/`email` through a shared client credential.
That is how "agent X acting for Alice vs Bob" is expressed. Plan around ID-JAG, treat RFC 8693 as
a fallback only for IdPs that cannot do ID-JAG.

**Also revised — downstream identity (D-06):** the definition assumed we forward the caller's
identity (possibly the token) so the tool self-authorizes. **Token passthrough is forbidden by
the spec.** The compliant pattern: agentgate is its own resource server (validates the inbound
token's audience = itself), and to reach a downstream tool it **mints/obtains a separate
credential** rather than relaying. This does not kill D-06 (the tool still owns per-record
authorization), but the identity *transfer* mechanism must be redesigned. **This is now the most
important open technical question** — flagged in §5.

### 2.5 Audit — Postgres, hash-chained

Postgres, one row per decision (agent, on-behalf-of, roles, tool, arguments-per-redaction-policy,
decision, reason, **policy-version hash**, timestamp). Changes from the POC:
- **Tamper-evidence via hash chaining:** each row stores `prev_hash` and
  `row_hash = H(prev_hash ‖ canonical(row))`. This is what makes "immutable"/"provable" a real
  claim rather than a promise. Design reference: RFC 6962 (Certificate Transparency).
- **Revoke `UPDATE`/`DELETE` from the application role** — schema changes go through a separate
  migration role.
- **Argument redaction policy per tool** (`full` / `hash` / `omit`) — the audit log must not
  silently become a PII/secrets store; hashing needs a per-deployment salt.
- **Monthly partitioning + a retention policy** — unbounded growth otherwise.
- **Async write off the hot path**, into a durable local buffer, to keep decision latency low
  while preserving "no decision without a record."

### 2.6 Relationships — SpiceDB, optional

Per D-06, SpiceDB is demoted from required to optional — used only for the narrow set of tools
that cannot authorize their own records. Kept in the stack because when it *is* needed it is the
right tool (Zanzibar-style, `fully_consistent` reads), but it is **not in the default deploy** and
not a dependency of the core decision path.

### 2.7 Observability — OpenTelemetry + `slog`

The POC had none; production needs it from day one, and latency is the adoption make-or-break:
- **`log/slog`** structured JSON logs, correlated by a request id generated at entry (never log
  argument values).
- **OpenTelemetry** traces (propagate W3C `traceparent` across agentgateway → agentgate →
  Postgres → backend) and metrics (decision counts by outcome, **decision-latency histogram**,
  audit-write failures, policy reloads, panics recovered).
- agentgateway already emits OTel/Prometheus, so the two tiers share one telemetry backend.

### 2.8 Packaging & deployment

- **Distroless (or scratch) container**, non-root, read-only root filesystem.
- **Helm charts** — agentgateway already ships OCI Helm charts + CRDs and a control plane; we
  ship alongside that model.
- **Single-binary quickstart** for OSS adoption: one binary, SQLite instead of Postgres, no
  SpiceDB, accept any OIDC issuer instead of bundling Keycloak. Every heavy dependency
  **optional and upgradeable**.

### 2.9 CI/CD & supply chain (new — required because it is a security product, OSS)

- GitHub Actions: build, `go test` (incl. race), `golangci-lint`, `govulncheck` on every PR.
- **Fuzz** the request/body parser (untrusted input).
- **Integration tests** with testcontainers: real Postgres + gateway + backend, asserting a
  denied call never reaches the backend.
- SBOM (CycloneDX), cosign-signed images, pinned base images, `SECURITY.md` + disclosure process.

---

## 3. What changes vs the POC stack

| POC | Production | Reason |
|---|---|---|
| Policy in a mounted `.cedar` file | Policy in versioned Postgres rows (file = dev only) | Dry-run, rollback, UI editing |
| Claim names hardcoded in Go | `claims-mapping.yaml` per deployment | Works with any real IdP |
| Ownership seeded in SpiceDB, required | SpiceDB optional; tools self-authorize | D-06 + token-passthrough rules |
| Plaintext gRPC/HTTP/DB | mTLS + TLS everywhere | The decision channel must be authenticated |
| Audit = plain table | Hash-chained, role-restricted, partitioned, async | "Immutable" must be true; latency off hot path |
| `os.Getenv` defaults incl. credentials | Typed validated config, no secret defaults | Fail-fast, no accidental prod defaults |
| stdlib `log.Printf` | `slog` + OpenTelemetry | Observability, latency measurement |
| No migrations tool | `golang-migrate`/`goose` | Repeatable schema, zero-downtime upgrades |
| No CI, no scanning, no SBOM | Full CI + supply-chain integrity | Security-product credibility |
| MCP (older assumptions: sessions, `initialize`) | MCP `2026-07-28` (stateless, header-routed) | Current spec; POC targets a dead revision |

---

## 4. Decisions still open (block parts of the plan)

These need a decision before the affected component is built. None blocks *starting*.

| # | Question | Blocks | Note |
|---|---|---|---|
| O-1 | **Downstream identity mechanism** — how agentgate conveys caller identity to a tool without token passthrough (mint-per-call? ID-JAG re-assertion? gateway-injected header the tool trusts?) | D-06 enforcement, §2.4 | Highest priority. The spec forbids the obvious approach; the replacement is undesigned. |
| O-2 | **Which MCP revision(s) to support** — `2026-07-28` only, or also older revisions still common in deployed clients? | Backend compat, client testing | The POC's `mark3labs/mcp-go v0.32.0` likely predates `2026-07-28`; library choice depends on this. |
| O-3 | **Whether to trust `ToolAnnotations` as a classification seed** — pre-fill the operator's risk screen from `destructiveHint` etc., knowing they are untrusted hints | Tool classification UX | Safe as a *default suggestion the operator confirms*, never as authority. |
| O-4 | **License** — Apache-2.0 (max adoption) vs AGPL-3.0/BUSL (protect hosted offering) | OSS launch | Hard to reverse once contributions arrive. Management decision. |
| O-5 | **Cedar version + policy-analysis tooling** — needed for the dry-run "is this policy more permissive?" feature | Dry-run feature | Verification of cedar-go tooling still in progress. |
| O-6 | **Managed-vs-self-hosted deployment target for v1** — affects whether we build tenant isolation and a control plane now | Tenancy scope | Definition D-11 says single-tenant runtime, tenant-aware schema. Confirm. |

> **Status note (2026-09-13):** this table's own "O-N" numbering predates, and is a **different
> numbering scheme** from, `docs/DECISIONS/OPEN_DECISIONS.md`'s "O-00N" items — the two are not
> the same list; don't confuse `O-1` here with `O-001` there. `docs/DECISIONS/OPEN_DECISIONS.md`
> is the canonical, currently-maintained open-decisions registry going forward. Per-item status,
> without rewriting the analysis above:
> - **O-1** (downstream identity) — still open; tracked as `OPEN_DECISIONS.md` O-001.
> - **O-2** (MCP revision) — still open; tracked as `OPEN_DECISIONS.md` O-004.
> - **O-3** (trust `ToolAnnotations`?) — **resolved**, and more strongly than proposed here:
>   `docs/SECURITY/PRODUCTION-INVARIANTS.md §2` item 10 makes "never treated as authorization
>   authority" a hard invariant, not just a UX default.
> - **O-4** (license) — still open; tracked in `docs/DEVELOPMENT/OSS_READINESS.md`, not in
>   `OPEN_DECISIONS.md`.
> - **O-5** (Cedar version) — **resolved**: `cedar-go v1.8.0` is pinned in `agentgate/go.mod` and
>   in active production use since the Day-2/G1 decision core.
> - **O-6** (tenancy) — **resolved**: `docs/SECURITY/PRODUCTION-INVARIANTS.md §9` confirms
>   single-tenant-per-deployment, matching D-11 as proposed here.

---

## 5. Recommended next step

Once this stack is agreed, the implementation plan proceeds in this order (each phase a working,
testable increment — not the POC's demo phases):

1. **Skeleton + config + observability spine** — typed config, `slog`, OTel, health probes,
   graceful shutdown, CI. (Everything else is built on this; the POC never had it.)
2. **Decision core** — Cedar engine, Postgres policy store (versioned), the `ext_authz` server,
   fail-closed everywhere, hash-chained audit. Prove: a denied call never reaches the backend.
3. **Identity** — OIDC claim mapping as config; resolve O-1 and implement the downstream-identity
   mechanism; ID-JAG on-behalf-of.
4. **Operator product** — tool discovery + classification (deny-unknown-by-default), the policy
   rule builder, and the **dry-run + rollback** loop (the differentiator).
5. **Hardening** — mTLS, argument redaction, rate/quota limits, retention/partitioning, degraded-
   mode behaviour.
6. **OSS launch** — single-binary quickstart, license decision (O-4), SBOM + signed releases,
   docs.

I will write that full implementation plan once we have signed off on this stack.
