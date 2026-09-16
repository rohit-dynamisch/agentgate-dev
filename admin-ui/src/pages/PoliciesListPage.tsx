import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAppContext } from "../context/AppContext";
import { usePolicyLifecycle } from "../hooks/usePolicyLifecycle";
import { toLifecycleSummaryView, toOperationStatusView, toPolicyBadgeView, type PolicyState } from "../lib/contract";
import { Badge } from "../components/Badge";

type Tab = "all" | "active" | "candidate" | "historical";

const TABS: { key: Tab; label: string }[] = [
  { key: "all", label: "All Policies" },
  { key: "active", label: "Active" },
  { key: "candidate", label: "Candidates" },
  { key: "historical", label: "History" },
];

export function PoliciesListPage() {
  const { client, workspaceId } = useAppContext();
  const { state, run } = usePolicyLifecycle(client, workspaceId);
  const [tab, setTab] = useState<Tab>("all");

  useEffect(() => {
    run((s) => s.loadPolicies());
  }, [client, workspaceId]);

  const summary = toLifecycleSummaryView(workspaceId, state.policies);
  const visible = state.policies.filter((p) => (tab === "all" ? true : (p.state as PolicyState) === tab));

  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <h1>Policies</h1>
          <p className="muted">Manage authorization policies for your organization.</p>
        </div>
        <Link to="/policies/new" className="button-primary">
          + Create Policy
        </Link>
      </div>

      <div className="stat-grid stat-grid--compact">
        <div className="mini-stat">
          <span className="mini-stat-value">{summary.totalPolicies}</span>
          <span className="mini-stat-label">Total</span>
        </div>
        <div className="mini-stat">
          <span className="mini-stat-value">{summary.activeVersion ? 1 : 0}</span>
          <span className="mini-stat-label">Active</span>
        </div>
        <div className="mini-stat">
          <span className="mini-stat-value">{summary.candidateCount}</span>
          <span className="mini-stat-label">Candidates</span>
        </div>
        <div className="mini-stat">
          <span className="mini-stat-value">{summary.historicalCount}</span>
          <span className="mini-stat-label">History</span>
        </div>
      </div>

      <section className="card">
        <div className="tab-row">
          {TABS.map((t) => (
            <button key={t.key} className={tab === t.key ? "tab-chip tab-chip--active" : "tab-chip"} onClick={() => setTab(t.key)}>
              {t.label}
            </button>
          ))}
        </div>

        {state.lastError && <p className="error-text">{state.lastError}</p>}

        {visible.length === 0 ? (
          <p className="muted">No policies in this view yet.</p>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Version</th>
                <th>State</th>
                <th>Description</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((p) => {
                const badge = toPolicyBadgeView(p.state);
                return (
                  <tr key={p.version}>
                    <td className="mono">{p.version.slice(0, 16)}…</td>
                    <td>
                      <Badge tone={badge.tone}>{badge.label}</Badge>
                    </td>
                    <td>{p.description}</td>
                    <td className="muted">{new Date(p.created_at).toLocaleString()}</td>
                    <td className="row">
                      {p.state !== "active" && (
                        <button className="button-secondary" onClick={() => run((s) => s.activate(p.version))}>
                          {p.state === "historical" ? "Roll back to this" : "Activate"}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
        <div className="row">
          <span className="muted">Activation:</span>
          <Badge tone={toOperationStatusView(state.activationStatus).tone}>{toOperationStatusView(state.activationStatus).label}</Badge>
          <span className="muted">Rollback:</span>
          <Badge tone={toOperationStatusView(state.rollbackStatus).tone}>{toOperationStatusView(state.rollbackStatus).label}</Badge>
        </div>
      </section>
    </div>
  );
}
