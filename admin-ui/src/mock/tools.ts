/**
 * Prototype-only mock data for the "tool inventory" concept — explicitly
 * listed as required-but-not-built in docs/PROJECT_DEFINITION.md §5a
 * ("Tool inventory UI ... not built") and §8 ("Tool risk classification").
 *
 * No real backend contract exists for this yet (internal/toolregistry is a
 * Go-internal package with no REST surface). Deliberately kept out of
 * frontend/src/models — see admin-ui/FLOW_AND_ARCHITECTURE.md §4 — so this
 * shape is never mistaken for a proposed or frozen contract.
 */

export type ToolRisk = "read" | "write" | "destructive" | "unclassified";
export type ToolStatus = "active" | "blocked";

export interface MockTool {
  id: string;
  name: string;
  backendId: string;
  classification: ToolRisk;
  status: ToolStatus;
  lastSeen: string;
  description: string;
  fingerprint: string;
  schema: Record<string, string>;
}

export const mockTools: MockTool[] = [
  {
    id: "crm/get_customer",
    name: "get_customer",
    backendId: "crm",
    classification: "read",
    status: "active",
    lastSeen: "2026-09-16T10:24:00Z",
    description: "Fetch a customer record by id.",
    fingerprint: "sha256:a1b8c...e0f2",
    schema: { customer_id: "string" },
  },
  {
    id: "crm/update_customer",
    name: "update_customer",
    backendId: "crm",
    classification: "write",
    status: "active",
    lastSeen: "2026-09-16T10:23:00Z",
    description: "Update mutable fields on a customer record.",
    fingerprint: "sha256:d94e2...11ab",
    schema: { customer_id: "string", fields: "object" },
  },
  {
    id: "crm/delete_customer",
    name: "delete_customer",
    backendId: "crm",
    classification: "destructive",
    status: "active",
    lastSeen: "2026-09-16T09:16:00Z",
    description: "Permanently deletes a customer record.",
    fingerprint: "sha256:1b8c0...e6fd",
    schema: { customer_id: "string" },
  },
  {
    id: "docs/search_documents",
    name: "search_documents",
    backendId: "docs",
    classification: "read",
    status: "active",
    lastSeen: "2026-09-16T10:22:00Z",
    description: "Full-text search over the document store.",
    fingerprint: "sha256:77aa1...9c3d",
    schema: { query: "string", limit: "int" },
  },
  {
    id: "ticketing/create_ticket",
    name: "create_ticket",
    backendId: "ticketing",
    classification: "write",
    status: "active",
    lastSeen: "2026-09-16T09:40:00Z",
    description: "Opens a new support ticket.",
    fingerprint: "sha256:5f0d3...7b1a",
    schema: { subject: "string", body: "string", priority: "string" },
  },
  {
    id: "email/send_email",
    name: "send_email",
    backendId: "email",
    classification: "write",
    status: "active",
    lastSeen: "2026-09-16T08:11:00Z",
    description: "Sends an email on behalf of the calling agent.",
    fingerprint: "sha256:c2e91...4a08",
    schema: { to: "string", subject: "string", body: "string" },
  },
  {
    id: "hr/read_database",
    name: "read_database",
    backendId: "hr",
    classification: "read",
    status: "active",
    lastSeen: "2026-09-16T07:02:00Z",
    description: "Direct read access to the HR reporting database.",
    fingerprint: "sha256:9e0a2...23fc",
    schema: { query: "string" },
  },
  {
    id: "hr/write_database",
    name: "write_database",
    backendId: "hr",
    classification: "write",
    status: "active",
    lastSeen: "2026-09-15T22:18:00Z",
    description: "Direct write access to the HR reporting database.",
    fingerprint: "sha256:44b7f...d901",
    schema: { table: "string", values: "object" },
  },
  {
    id: "finance/transfer_funds",
    name: "transfer_funds",
    backendId: "finance",
    classification: "unclassified",
    status: "blocked",
    lastSeen: "2026-09-16T06:47:00Z",
    description: "Newly discovered by tool discovery — not yet classified. Denied by default until an operator assigns a risk level.",
    fingerprint: "sha256:0c1de...88a2",
    schema: { from: "string", to: "string", amount: "int" },
  },
];
