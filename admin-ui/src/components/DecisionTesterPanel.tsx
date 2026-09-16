import { useState } from "react";
import {
  parseAuthorizationResult,
  parseTransportError,
  reduce,
  wireFixtures,
  type ApiError,
  type OperationState,
  type AuthorizationResult,
} from "../lib/contract";
import { renderOperationState } from "../lib/contract";
import { Badge } from "./Badge";

type Mode = "fixture" | "live";

const FIXTURE_KEYS = Object.keys(wireFixtures) as (keyof typeof wireFixtures)[];

function nowIso() {
  return new Date().toISOString();
}

function initialLiveForm() {
  return {
    executionId: `ui-${Date.now()}`,
    workspaceId: "default-workspace",
    agentId: "agent-1",
    onBehalfOf: "",
    roles: "reader",
    backendId: "crm",
    toolName: "read_record",
    known: true,
    risk: "read",
  };
}

export function DecisionTesterPanel() {
  const [mode, setMode] = useState<Mode>("fixture");
  const [fixtureKey, setFixtureKey] = useState<(typeof FIXTURE_KEYS)[number]>("allow");
  const [opState, setOpState] = useState<OperationState<AuthorizationResult>>({ status: "idle" });
  const [form, setForm] = useState(initialLiveForm());

  function set<K extends keyof ReturnType<typeof initialLiveForm>>(key: K, value: ReturnType<typeof initialLiveForm>[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function runFixture() {
    const raw: unknown = wireFixtures[fixtureKey];
    if (fixtureKey === "transportError") {
      const err = parseTransportError(raw, 400);
      setOpState((prev) => reduce(prev, { type: "fail", error: err, at: nowIso() }));
      return;
    }
    const parsed = parseAuthorizationResult(raw);
    if (parsed.ok) {
      setOpState((prev) => reduce(prev, { type: "succeed", data: parsed.value, at: nowIso() }));
    } else {
      setOpState((prev) => reduce(prev, { type: "fail", error: parsed.error, at: nowIso() }));
    }
  }

  async function runLive() {
    setOpState((prev) => reduce(prev, { type: "start" }));
    const body = {
      execution_id: form.executionId,
      workspace_id: form.workspaceId,
      identity: {
        agent_id: form.agentId,
        ...(form.onBehalfOf ? { on_behalf_of: form.onBehalfOf } : {}),
        roles: form.roles.split(",").map((r) => r.trim()).filter(Boolean),
      },
      tool: { backend_id: form.backendId, name: form.toolName },
      classification: { known: form.known, ...(form.known ? { risk: form.risk } : {}) },
    };

    try {
      const resp = await fetch("/proxy/mock-authz/evaluate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      let json: unknown;
      try {
        json = await resp.json();
      } catch {
        json = undefined;
      }
      if (!resp.ok) {
        const err: ApiError = json !== undefined ? parseTransportError(json, resp.status) : { kind: "unexpected_status", status: resp.status };
        setOpState((prev) => reduce(prev, { type: "fail", error: err, at: nowIso() }));
        return;
      }
      const parsed = parseAuthorizationResult(json);
      if (parsed.ok) {
        setOpState((prev) => reduce(prev, { type: "succeed", data: parsed.value, at: nowIso() }));
      } else {
        setOpState((prev) => reduce(prev, { type: "fail", error: parsed.error, at: nowIso() }));
      }
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      setOpState((prev) =>
        reduce(prev, {
          type: "fail",
          error: { kind: "network", message: `${message} — is g1-mock-authz running? (go run ./cmd/g1-mock-authz)` },
          at: nowIso(),
        })
      );
    }
  }

  const view = renderOperationState(opState);

  return (
    <div className="stack">
      <section className="card">
        <h2>Decision Tester</h2>
        <p className="muted">
          Exercises the frozen G1 authorization contract's rendering layer (frontend/src/view/decision-view.ts):
          every state below (allow, deny, transport error, malformed response) is rendered by the same code path.
        </p>
        <div className="row">
          <label>
            <input type="radio" checked={mode === "fixture"} onChange={() => setMode("fixture")} /> Dummy data (fixtures)
          </label>
          <label>
            <input type="radio" checked={mode === "live"} onChange={() => setMode("live")} /> Live g1-mock-authz
          </label>
        </div>

        {mode === "fixture" ? (
          <div className="row">
            <select value={fixtureKey} onChange={(e) => setFixtureKey(e.target.value as (typeof FIXTURE_KEYS)[number])}>
              {FIXTURE_KEYS.map((k) => (
                <option key={k} value={k}>
                  {k}
                </option>
              ))}
            </select>
            <button onClick={runFixture}>Run</button>
          </div>
        ) : (
          <div className="stack">
            <p className="muted">
              Requires <code>go run ./cmd/g1-mock-authz</code> running on :8091 (proxied via /proxy/mock-authz).
            </p>
            <div className="row">
              <input placeholder="agent_id" value={form.agentId} onChange={(e) => set("agentId", e.target.value)} />
              <input placeholder="on_behalf_of (optional)" value={form.onBehalfOf} onChange={(e) => set("onBehalfOf", e.target.value)} />
              <input placeholder="roles (comma-separated)" value={form.roles} onChange={(e) => set("roles", e.target.value)} />
            </div>
            <div className="row">
              <input placeholder="backend_id" value={form.backendId} onChange={(e) => set("backendId", e.target.value)} />
              <input placeholder="tool name" value={form.toolName} onChange={(e) => set("toolName", e.target.value)} />
              <label>
                <input type="checkbox" checked={form.known} onChange={(e) => set("known", e.target.checked)} /> known tool
              </label>
              <input placeholder="risk" value={form.risk} onChange={(e) => set("risk", e.target.value)} disabled={!form.known} />
            </div>
            <div className="row">
              <input placeholder="workspace_id" value={form.workspaceId} onChange={(e) => set("workspaceId", e.target.value)} />
              <button onClick={runLive}>Send to mock</button>
            </div>
          </div>
        )}
      </section>

      <section className="card">
        <h2>Result</h2>
        {view.status === "idle" && <p className="muted">No request run yet.</p>}
        {view.status === "loading" && <p className="muted">Loading…</p>}
        {view.status === "succeeded" && (
          <div className={`callout callout--${view.tone === "allow" ? "allow" : "deny"}`}>
            <Badge tone={view.tone}>{view.headline}</Badge>
            <p>{view.detail || <span className="muted">(no message)</span>}</p>
            <p className="muted mono">policy version: {view.policyVersionLabel}</p>
            <p className="muted mono">execution id: {view.executionId}</p>
          </div>
        )}
        {view.status === "denied" && (
          <div className="callout callout--deny">
            <Badge tone="deny">{view.headline}</Badge>
            <p>{view.detail}</p>
          </div>
        )}
        {view.status === "apiError" && (
          <div className="callout callout--error">
            <Badge tone="error">{view.headline}</Badge>
            <p>{view.detail}</p>
          </div>
        )}
        {view.status === "stale" && (
          <div className="callout callout--stale">
            <Badge tone="stale">{view.headline}</Badge>
            <p>{view.detail}</p>
          </div>
        )}
      </section>
    </div>
  );
}
