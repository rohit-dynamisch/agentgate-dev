import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAppContext } from "../context/AppContext";
import { computeDashboardStats } from "../mock/dashboard";
import { mockAuditLog } from "../mock/auditLog";
import { AuthChart } from "../components/AuthChart";
import { StatTile } from "../components/StatTile";
import { Badge } from "../components/Badge";
import { Icon } from "../components/Icon";
import { formatCompact } from "../lib/format";
import type { PolicyRecord } from "../lib/contract";

const SYSTEM_STATUS = [
  { name: "AgentGate", ok: true },
  { name: "agentgateway", ok: true },
  { name: "PostgreSQL", ok: true },
  { name: "MCP backends", ok: true },
];

export function DashboardPage() {
  const { client, workspaceId } = useAppContext();
  const [policies, setPolicies] = useState<PolicyRecord[]>([]);
  const stats = computeDashboardStats();
  const active = policies.find((p) => p.state === "active");

  useEffect(() => {
    client.listPolicies(workspaceId).then(setPolicies).catch(() => setPolicies([]));
  }, [client, workspaceId]);

  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <h1>Welcome to AgentGate</h1>
          <p className="muted">Monitor, manage, and secure AI agent access to your tools and resources.</p>
        </div>
      </div>

      <div className="stat-grid">
        <StatTile label="Total requests (24h)" value={formatCompact(stats.totalRequests)} tone="neutral" icon={<Icon name="dashboard" size={18} />} />
        <StatTile label="Allowed" value={formatCompact(stats.allowed)} sub={`${stats.allowedPct}%`} tone="good" icon={<Icon name="shield" size={18} />} />
        <StatTile label="Denied" value={formatCompact(stats.denied)} sub={`${stats.deniedPct}%`} tone="critical" icon={<Icon name="policies" size={18} />} />
        <StatTile label="Unknown tool" value={formatCompact(stats.unknown)} sub={`${stats.unknownPct}%`} tone="warning" icon={<Icon name="tools" size={18} />} />
      </div>

      <div className="grid-2">
        <section className="card">
          <h2>Authorization requests</h2>
          <AuthChart data={stats.timeline} />
        </section>

        <section className="card">
          <h2>Tool usage (top 5)</h2>
          <ul className="bar-list">
            {stats.topTools.map((t) => {
              const max = stats.topTools[0]?.count ?? 1;
              return (
                <li key={t.name}>
                  <div className="bar-list-row">
                    <span className="mono">{t.name}</span>
                    <span className="muted">{formatCompact(t.count)}</span>
                  </div>
                  <div className="bar-track">
                    <div className="bar-fill" style={{ width: `${(t.count / max) * 100}%` }} />
                  </div>
                </li>
              );
            })}
          </ul>
          <Link to="/tools" className="link-more">
            View all tools →
          </Link>
        </section>
      </div>

      <div className="grid-3">
        <section className="card">
          <h2>System status</h2>
          <ul className="status-list">
            {SYSTEM_STATUS.map((s) => (
              <li key={s.name}>
                <span className={`dot ${s.ok ? "dot--good" : "dot--critical"}`} />
                {s.name}
              </li>
            ))}
          </ul>
          <p className="muted small">Illustrative — real health/readiness endpoints exist on cmd/agentgate but aren't polled from this prototype.</p>
        </section>

        <section className="card">
          <h2>Active policy</h2>
          {active ? (
            <>
              <p className="mono">{active.version.slice(0, 20)}…</p>
              <p className="muted">{active.description}</p>
              <Link to="/policies" className="link-more">
                View →
              </Link>
            </>
          ) : (
            <p className="muted">No active policy loaded for this workspace yet.</p>
          )}
        </section>

        <section className="card">
          <h2>Recent audit logs</h2>
          <ul className="mini-log">
            {mockAuditLog.slice(0, 4).map((e) => (
              <li key={e.id}>
                <span className="muted mono">{new Date(e.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span>
                <span>{e.identity}</span>
                <span className="muted">{e.tool ?? e.action}</span>
                {e.decision && <Badge tone={e.decision === "ALLOW" ? "allow" : "deny"}>{e.decision}</Badge>}
              </li>
            ))}
          </ul>
          <Link to="/audit" className="link-more">
            View all →
          </Link>
        </section>
      </div>
    </div>
  );
}
