# AgentGate Admin UI (prototype)

A rendered UI on top of the existing framework-agnostic contract layer in
[`../frontend/src`](../frontend/src) (models, parsing, state machines, and
pure view-model functions built across G1–G4). That package intentionally
ships no rendering code — see its `package.json` description and the file
header in `frontend/src/view/decision-view.ts` — because the production UI
framework was left an open item. This app is a prototype/demo scaffold that
exercises that layer with React, **not** a resolution of that open item.

Built from a concept mockup (dashboard, policy lifecycle, tool governance,
identities, audit, settings, login). See
[`FLOW_AND_ARCHITECTURE.md`](FLOW_AND_ARCHITECTURE.md) for the screen-by-screen
mapping of that concept against what AgentGate's backend actually does today
versus what's documented-but-unbuilt versus what's been deliberately reframed
(the login and settings screens both diverge from the concept for reasons
explained there).

## Run it

```bash
cd admin-ui
npm install
npm run dev
```

Open http://localhost:5173 — click "Enter" on the token screen (blank is fine
in dummy-data mode). Defaults to dummy/mock data throughout; no backend needed.

## Pages

| Page | Real or mock? | Backed by |
|---|---|---|
| Dashboard | Mixed | Real: active-policy card (`GovernanceClient`). Mock: request-volume stats/chart/top-tools (no aggregation API exists yet). |
| Policies (list + create/validate/dry-run/activate wizard) | Real | `GovernanceClient` / `MockGovernanceClient` / `HttpGovernanceClient`, same as before. |
| Tools & Resources | Mock, grounded | No REST API exists for `internal/toolregistry` yet; shape mirrors it. Classification edits are client-side only. |
| Identities | Mock, reframed | AgentGate has no identity directory (`PROJECT_DEFINITION.md`: "not an identity provider") — framed as "seen in activity," not user management. |
| Decision Tester | Real | Unchanged from the original prototype — fixtures + optional live `cmd/g1-mock-authz` call. |
| Audit Logs | Mock, grounded | Durable audit storage is Gate G5 (not started); rows shaped to match the real decision/mutation-event fields. |
| Settings | Real (scoped down) | Workspace + connection mode/admin token — deliberately not an editable infra-config form (see flow doc). |

All new mock-only data lives under `src/mock/` — deliberately never mixed into
`../frontend/src/models` (the frozen G1 contract other workstreams read from);
see `FLOW_AND_ARCHITECTURE.md` §4.

## Login gate

Client-side only — not a security boundary. AgentGate has no built-in
login system; the real enforcement point is the backend's admin-token check,
which still applies if you switch a panel to "Live" mode. See the note on the
gate screen itself and `FLOW_AND_ARCHITECTURE.md` §3 row 13.

## Live backend modes

- Policies / Settings "Live cmd/agentgate": `HttpGovernanceClient` against a
  real `cmd/agentgate` on `:8090`, proxied via `/proxy/agentgate` (that server
  isn't CORS-enabled — it's server-to-server/CLI-facing). Needs
  `AGENTGATE_ADMIN_TOKEN`.
- Decision Tester "Live g1-mock-authz": POSTs to `go run ./cmd/g1-mock-authz`
  on `:8091`, proxied via `/proxy/mock-authz`.

**Not verified in this environment** — no Go toolchain was available here, so
neither live mode has been exercised end-to-end. The wire shapes match the
frozen G1 contract and reuse the existing tested client classes, but verify
yourself with Go installed before relying on them.

## Verified

- `npm run typecheck` / `npm run build` — clean.
- Full Playwright walkthrough of every page (login → dashboard → policies list
  → create/validate/dry-run/activate wizard → tools list/detail/classify →
  identities → decision tester (all 12 fixtures) → audit log/detail →
  settings). Zero browser console/page errors throughout.
