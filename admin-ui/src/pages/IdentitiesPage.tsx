import { mockIdentities } from "../mock/identities";
import { Badge } from "../components/Badge";

export function IdentitiesPage() {
  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <h1>Identities</h1>
          <p className="muted">Agents, humans, and service accounts seen in recent activity.</p>
        </div>
      </div>

      <div className="callout callout--stale">
        <strong>Not a user directory.</strong>
        <p>
          AgentGate is explicitly not an identity provider — it reads whatever claims arrive on each inbound JWT,
          issued by your OIDC provider (Okta/Entra/Keycloak). This list is derived from recent activity, not
          managed here.
        </p>
      </div>

      <section className="card">
        <table className="table">
          <thead>
            <tr>
              <th>Identity</th>
              <th>Type</th>
              <th>Roles</th>
              <th>On behalf of</th>
              <th>Status</th>
              <th>Last seen</th>
              <th>Auth method</th>
            </tr>
          </thead>
          <tbody>
            {mockIdentities.map((i) => (
              <tr key={i.id}>
                <td className="mono">{i.id}</td>
                <td>{i.kind}</td>
                <td>{i.roles.join(", ")}</td>
                <td className="muted">{i.onBehalfOf ?? "—"}</td>
                <td>
                  <Badge tone={i.status === "active" ? "allow" : "idle"}>{i.status}</Badge>
                </td>
                <td className="muted">{new Date(i.lastSeen).toLocaleString()}</td>
                <td className="muted">{i.authMethod}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
