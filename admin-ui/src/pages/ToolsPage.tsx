import { Link } from "react-router-dom";
import { useAppContext } from "../context/AppContext";
import { Badge } from "../components/Badge";

export function ToolsPage() {
  const { tools } = useAppContext();
  const unclassified = tools.filter((t) => t.classification === "unclassified").length;

  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <h1>Tools & Resources</h1>
          <p className="muted">Manage tool classification and governance. Discovered automatically from connected MCP backends.</p>
        </div>
      </div>

      {unclassified > 0 && (
        <div className="callout callout--stale">
          <strong>{unclassified} unclassified tool{unclassified > 1 ? "s" : ""} detected.</strong>
          <p>Unclassified tools are denied by default until an operator assigns a risk level.</p>
        </div>
      )}

      <section className="card">
        <table className="table">
          <thead>
            <tr>
              <th>Tool name</th>
              <th>Backend</th>
              <th>Classification</th>
              <th>Status</th>
              <th>Last seen</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {tools.map((t) => (
              <tr key={t.id}>
                <td className="mono">{t.name}</td>
                <td>{t.backendId}</td>
                <td>
                  <Badge tone={t.classification === "unclassified" ? "pending" : t.classification === "destructive" ? "deny" : "allow"}>{t.classification}</Badge>
                </td>
                <td>
                  <Badge tone={t.status === "active" ? "allow" : "deny"}>{t.status}</Badge>
                </td>
                <td className="muted">{new Date(t.lastSeen).toLocaleString()}</td>
                <td>
                  <Link to={`/tools/${encodeURIComponent(t.id)}`} className="link-more">
                    Details →
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
