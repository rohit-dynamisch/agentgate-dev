import { useParams, Link } from "react-router-dom";
import { useAppContext } from "../context/AppContext";
import type { ToolRisk } from "../mock/tools";
import { Badge } from "../components/Badge";

const RISKS: ToolRisk[] = ["read", "write", "destructive"];

export function ToolDetailPage() {
  const { id } = useParams();
  const { tools, classifyTool } = useAppContext();
  const tool = tools.find((t) => t.id === decodeURIComponent(id ?? ""));

  if (!tool) {
    return (
      <div className="stack">
        <p className="muted">Tool not found.</p>
        <Link to="/tools" className="link-more">
          ← Back to Tools & Resources
        </Link>
      </div>
    );
  }

  return (
    <div className="stack">
      <p className="breadcrumb">
        <Link to="/tools">Tools & Resources</Link> / {tool.name}
      </p>
      <div className="page-header">
        <div>
          <h1>{tool.name}</h1>
          <p className="muted">{tool.description}</p>
        </div>
        <Badge tone={tool.classification === "unclassified" ? "pending" : tool.classification === "destructive" ? "deny" : "allow"}>{tool.classification}</Badge>
      </div>

      <div className="grid-2">
        <section className="card">
          <h2>Basic information</h2>
          <dl className="detail-list">
            <dt>Backend</dt>
            <dd>{tool.backendId}</dd>
            <dt>Status</dt>
            <dd>
              <Badge tone={tool.status === "active" ? "allow" : "deny"}>{tool.status}</Badge>
            </dd>
            <dt>Fingerprint</dt>
            <dd className="mono">{tool.fingerprint}</dd>
            <dt>Last seen</dt>
            <dd>{new Date(tool.lastSeen).toLocaleString()}</dd>
          </dl>
        </section>

        <section className="card">
          <h2>Argument schema</h2>
          <pre className="code-block">{JSON.stringify(tool.schema, null, 2)}</pre>
        </section>
      </div>

      <section className="card">
        <h2>Risk classification</h2>
        <p className="muted">
          Determines which Cedar rules can match this tool (<code>resource.risk</code>). An unclassified tool is
          denied by default — see fixturepolicy's forbid-by-default behavior.
        </p>
        <div className="row">
          {RISKS.map((r) => (
            <button key={r} className={tool.classification === r ? "button-primary" : "button-secondary"} onClick={() => classifyTool(tool.id, r)}>
              {r}
            </button>
          ))}
        </div>
        <p className="muted small">
          Client-side only in this prototype — there's no REST endpoint yet to persist tool classification (see
          admin-ui/FLOW_AND_ARCHITECTURE.md §3 rows 8/9).
        </p>
      </section>
    </div>
  );
}
