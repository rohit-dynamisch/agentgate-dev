/**
 * Prototype-only mock data for the audit log screen. Durable, queryable
 * audit storage is Gate G5 (explicitly the next, not-yet-started checkpoint
 * per docs/DEVELOPMENT/CURRENT_STATUS.md's G4 handoff). G4 only has an
 * in-memory MutationListener for policy mutations (internal/auditevents).
 *
 * Field shapes here intentionally mirror two real, already-frozen contracts
 * so this mock is a faithful stand-in rather than an invented one:
 *   - decision/reason/policyVersion/executionId -> AuthorizationResult
 *     (frontend/src/models/decision.ts, frozen G1 contract)
 *   - action/version/workspaceId -> the shape described for MutationEvent
 *     in docs/PHASES/G4_WORKSTREAMS/CLOSURE_SUMMARY.md
 * Kept out of frontend/src/models regardless — see
 * admin-ui/FLOW_AND_ARCHITECTURE.md §4.
 */

import type { Decision, ReasonCode } from "../lib/contract";

export type AuditKind = "decision" | "policy_mutation";

export interface MockAuditEntry {
  id: string;
  time: string;
  kind: AuditKind;
  identity: string;
  onBehalfOf?: string;
  tool?: string;
  classification?: string;
  decision?: Decision;
  reason?: ReasonCode;
  action?: "create_candidate" | "activate" | "rollback";
  policyVersion: string;
  workspaceId: string;
  correlationId: string;
  arguments?: Record<string, string>;
}

export const mockAuditLog: MockAuditEntry[] = [
  { id: "evt-1042", time: "2026-09-16T10:24:31Z", kind: "decision", identity: "agent-support", tool: "get_customer", classification: "read", decision: "ALLOW", reason: "policy_allow", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9931" },
  { id: "evt-1041", time: "2026-09-16T10:24:12Z", kind: "decision", identity: "agent-hr-bot", onBehalfOf: "jane.doe@example.test", tool: "read_database", classification: "read", decision: "DENY", reason: "no_matching_policy", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9930" },
  { id: "evt-1040", time: "2026-09-16T10:23:55Z", kind: "decision", identity: "agent-support", tool: "update_customer", classification: "write", decision: "ALLOW", reason: "policy_allow", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9929", arguments: { customer_id: "cus_88f1", fields: "[redacted]" } },
  { id: "evt-1039", time: "2026-09-16T09:41:03Z", kind: "policy_mutation", identity: "user:jane.doe", action: "activate", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "mut-221" },
  { id: "evt-1038", time: "2026-09-16T09:23:44Z", kind: "decision", identity: "agent-finance", onBehalfOf: "john.smith@example.test", tool: "transfer_funds", classification: "unclassified", decision: "DENY", reason: "unknown_tool", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9902" },
  { id: "evt-1037", time: "2026-09-16T08:11:19Z", kind: "decision", identity: "agent-support", tool: "send_email", classification: "write", decision: "ALLOW", reason: "policy_allow", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9887" },
  { id: "evt-1036", time: "2026-09-16T07:02:51Z", kind: "decision", identity: "agent-hr-bot", tool: "read_database", classification: "read", decision: "ALLOW", reason: "policy_allow", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9871" },
  { id: "evt-1035", time: "2026-09-16T06:47:08Z", kind: "decision", identity: "agent-finance", tool: "delete_customer", classification: "destructive", decision: "DENY", reason: "policy_deny", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9855" },
  { id: "evt-1034", time: "2026-09-15T22:18:40Z", kind: "decision", identity: "svc-nightly-export", tool: "write_database", classification: "write", decision: "DENY", reason: "invalid_identity", policyVersion: "hash_2e244f41", workspaceId: "default-workspace", correlationId: "req-9801" },
  { id: "evt-1033", time: "2026-09-15T20:02:12Z", kind: "policy_mutation", identity: "user:jane.doe", action: "create_candidate", policyVersion: "hash_8d969ee", workspaceId: "default-workspace", correlationId: "mut-198" },
];
