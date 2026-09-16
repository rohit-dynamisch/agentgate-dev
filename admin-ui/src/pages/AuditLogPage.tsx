import { useState } from "react";
import { mockAuditLog, type MockAuditEntry } from "../mock/auditLog";
import { Badge } from "../components/Badge";

type Filter = "all" | "ALLOW" | "DENY";

export function AuditLogPage() {
  const [filter, setFilter] = useState<Filter>("all");
  const [selected, setSelected] = useState<MockAuditEntry | null>(mockAuditLog[0] ?? null);

  const rows = mockAuditLog.filter((e) => (filter === "all" ? true : e.decision === filter));

  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <h1>Audit logs</h1>
          <p className="muted">View all authorization decisions and policy mutations.</p>
        </div>
      </div>

      <div className="callout callout--stale">
        <strong>Prototype data.</strong>
        <p>
          Durable, queryable audit storage is Gate G5 — explicitly the next, not-yet-started checkpoint. G4 only
          keeps an in-memory listener for policy mutations. These rows are illustrative, shaped to match the real
          decision/mutation-event fields.
        </p>
      </div>

      <div className="tab-row">
        {(["all", "ALLOW", "DENY"] as Filter[]).map((f) => (
          <button key={f} className={filter === f ? "tab-chip tab-chip--active" : "tab-chip"} onClick={() => setFilter(f)}>
            {f === "all" ? "All decisions" : f}
          </button>
        ))}
      </div>

      <div className="audit-layout">
        <section className="card audit-list">
          <table className="table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Identity</th>
                <th>Tool / Action</th>
                <th>Decision</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((e) => (
                <tr key={e.id} className={selected?.id === e.id ? "row-selected" : ""} onClick={() => setSelected(e)}>
                  <td className="muted mono">{new Date(e.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</td>
                  <td>{e.identity}</td>
                  <td className="mono">{e.tool ?? e.action}</td>
                  <td>{e.decision ? <Badge tone={e.decision === "ALLOW" ? "allow" : "deny"}>{e.decision}</Badge> : <Badge tone="idle">mutation</Badge>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>

        <section className="card audit-detail">
          <h2>Event details</h2>
          {selected ? (
            <dl className="detail-list">
              <dt>Time</dt>
              <dd>{new Date(selected.time).toLocaleString()}</dd>
              <dt>Identity</dt>
              <dd className="mono">{selected.identity}</dd>
              {selected.onBehalfOf && (
                <>
                  <dt>Acting for</dt>
                  <dd>{selected.onBehalfOf}</dd>
                </>
              )}
              {selected.tool && (
                <>
                  <dt>Tool</dt>
                  <dd className="mono">{selected.tool}</dd>
                  <dt>Classification</dt>
                  <dd>{selected.classification}</dd>
                </>
              )}
              {selected.decision && (
                <>
                  <dt>Decision</dt>
                  <dd>
                    <Badge tone={selected.decision === "ALLOW" ? "allow" : "deny"}>{selected.decision}</Badge>
                  </dd>
                  <dt>Reason</dt>
                  <dd className="mono">{selected.reason}</dd>
                </>
              )}
              {selected.action && (
                <>
                  <dt>Mutation</dt>
                  <dd className="mono">{selected.action}</dd>
                </>
              )}
              <dt>Policy version</dt>
              <dd className="mono">{selected.policyVersion}</dd>
              <dt>Correlation ID</dt>
              <dd className="mono">{selected.correlationId}</dd>
              {selected.arguments && (
                <>
                  <dt>Arguments</dt>
                  <dd className="mono">{JSON.stringify(selected.arguments)}</dd>
                </>
              )}
            </dl>
          ) : (
            <p className="muted">Select a row to see details.</p>
          )}
        </section>
      </div>
    </div>
  );
}
