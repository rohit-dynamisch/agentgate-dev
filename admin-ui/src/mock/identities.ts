/**
 * Prototype-only mock data for "identities seen in activity."
 *
 * AgentGate is explicitly not an identity provider (docs/PROJECT_DEFINITION.md
 * §1: "we consume the customer's existing OIDC"). There is no AgentGate-owned
 * user/agent directory to list — internal/identity only reads claims already
 * present on each inbound JWT, per call. This is deliberately framed as a
 * derived "seen recently" view, not a user-management screen. See
 * admin-ui/FLOW_AND_ARCHITECTURE.md §3 row 11.
 */

export type IdentityKind = "Agent" | "Human" | "Service";

export interface MockIdentity {
  id: string;
  kind: IdentityKind;
  roles: string[];
  onBehalfOf?: string;
  status: "active" | "inactive";
  lastSeen: string;
  authMethod: string;
}

export const mockIdentities: MockIdentity[] = [
  { id: "agent-support", kind: "Agent", roles: ["reader"], status: "active", lastSeen: "2026-09-16T10:24:00Z", authMethod: "JWT (OIDC)" },
  { id: "agent-hr-bot", kind: "Agent", roles: ["reader", "admin"], onBehalfOf: "jane.doe@example.test", status: "active", lastSeen: "2026-09-16T09:18:00Z", authMethod: "JWT (OIDC)" },
  { id: "agent-finance", kind: "Agent", roles: ["payer"], onBehalfOf: "john.smith@example.test", status: "active", lastSeen: "2026-09-16T06:47:00Z", authMethod: "JWT (OIDC)" },
  { id: "user:jane.doe", kind: "Human", roles: ["admin"], status: "active", lastSeen: "2026-09-16T09:18:00Z", authMethod: "OIDC (delegated)" },
  { id: "user:mike.wilson", kind: "Human", roles: ["reader"], status: "inactive", lastSeen: "2026-09-10T14:02:00Z", authMethod: "OIDC (delegated)" },
  { id: "svc-nightly-export", kind: "Service", roles: ["reader"], status: "active", lastSeen: "2026-09-16T02:00:00Z", authMethod: "JWT (service account)" },
];
