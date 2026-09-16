import { useAppContext } from "../context/AppContext";

export function SettingsPage() {
  const { workspaceId, setWorkspaceId, govMode, setGovMode, adminToken, setAdminToken } = useAppContext();

  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <h1>Settings</h1>
          <p className="muted">Governance connection and workspace configuration.</p>
        </div>
      </div>

      <div className="callout callout--stale">
        <strong>Not a live infrastructure editor.</strong>
        <p>
          AgentGate's runtime configuration (DB connection, TLS, gateway endpoints — <code>internal/config</code>)
          is injected via environment variables at process start, not edited at runtime. Showing an editable form
          for that here would misrepresent the security model — see admin-ui/FLOW_AND_ARCHITECTURE.md §3 row 12.
          What's below is what a governance UI can actually control: which workspace and backend this session
          talks to.
        </p>
      </div>

      <section className="card">
        <h2>Workspace</h2>
        <label className="field">
          <span>Workspace ID</span>
          <input value={workspaceId} onChange={(e) => setWorkspaceId(e.target.value)} />
        </label>
      </section>

      <section className="card">
        <h2>Governance API connection</h2>
        <div className="row">
          <label>
            <input type="radio" checked={govMode === "mock"} onChange={() => setGovMode("mock")} /> Dummy data (in-memory mock)
          </label>
          <label>
            <input type="radio" checked={govMode === "live"} onChange={() => setGovMode("live")} /> Live cmd/agentgate (:8090)
          </label>
        </div>
        {govMode === "live" && (
          <label className="field">
            <span>AGENTGATE_ADMIN_TOKEN</span>
            <input type="password" value={adminToken} onChange={(e) => setAdminToken(e.target.value)} />
          </label>
        )}
      </section>

      <section className="card">
        <h2>Instance info (read-only)</h2>
        <dl className="detail-list">
          <dt>Cedar policy engine</dt>
          <dd className="mono">internal/policy (in-process, no sidecar)</dd>
          <dt>Governance API</dt>
          <dd className="mono">/proxy/agentgate → http://localhost:8090</dd>
          <dt>Decision mock</dt>
          <dd className="mono">/proxy/mock-authz → http://localhost:8091</dd>
        </dl>
      </section>
    </div>
  );
}
