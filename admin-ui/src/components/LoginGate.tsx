import { useState } from "react";
import { Icon } from "./Icon";

/**
 * Reframed from the concept's email/password login — see
 * FLOW_AND_ARCHITECTURE.md §3 row 13. AgentGate is explicitly not an
 * identity provider (docs/PROJECT_DEFINITION.md §1) and has no user/password
 * store; the only real admin-facing auth mechanism today is a single shared
 * bearer token checked by the governance REST API (G3/G4). This screen is a
 * client-side UX gate only — it does not itself enforce anything; the real
 * boundary is still the backend's token check on live-mode requests.
 */
export function LoginGate({ onEnter }: { onEnter: (token: string) => void }) {
  const [token, setToken] = useState("");

  return (
    <div className="login-screen">
      <form
        className="login-card"
        onSubmit={(e) => {
          e.preventDefault();
          onEnter(token);
        }}
      >
        <div className="login-logo">
          <Icon name="shield" size={28} />
        </div>
        <h1>AgentGate</h1>
        <p className="login-subtitle">Secure AI Actions</p>

        <label className="login-field">
          <span>Admin token</span>
          <input
            type="password"
            placeholder="AGENTGATE_ADMIN_TOKEN (optional in dummy-data mode)"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            autoFocus
          />
        </label>

        <button type="submit" className="login-submit">
          Enter
        </button>

        <p className="login-note">
          This is a UX gate for the prototype, not a security boundary — AgentGate has no
          built-in login system (it's explicitly not an identity provider; production admin
          auth would federate to your OIDC provider). The real enforcement point is the
          backend's admin-token check, which still applies if you switch any panel to "Live" mode.
        </p>
      </form>
    </div>
  );
}
