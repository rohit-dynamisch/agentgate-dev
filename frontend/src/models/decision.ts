/**
 * Typed external models for AgentGate's frozen G1 authorization contract.
 *
 * Source of truth: docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md
 * (AG-GO-G1-02, AG-GO-G1-03), cross-checked directly against
 * agentgate/internal/decision/types.go and agentgate/internal/mockauthz/wire.go.
 *
 * These types describe ONLY what the frozen contract provides:
 *   - decision.Request  -> AuthorizationRequestWire / AuthorizationRequest
 *   - decision.Result   -> AuthorizationResultWire  / AuthorizationResult
 *
 * They intentionally do NOT expose Cedar internals (principal/resource/action/
 * context entities). The frontend only ever sees the already-abstracted
 * decision.Result / Request shape that the Go decision core produces — never
 * a Cedar policy, entity, or diagnostic. See AG-UI-G1-02 and the "no raw
 * Cedar diagnostics" guarantee documented in the Go contract.
 *
 * Two layers, mirroring the split the Go side already made
 * (internal/decision vs internal/mockauthz/wire.go):
 *   - "*Wire" types: exact JSON shape (snake_case keys), used only at the
 *     parsing boundary (src/parsing).
 *   - Plain types (no suffix): frontend-idiomatic camelCase domain models,
 *     used everywhere else in the UI layer.
 *
 * This module makes no authorization decision and calls no network/backend
 * code — it only types data the backend has already decided.
 */

/** Exactly two values. There is no third "error" decision (AG-GO-G1-03). */
export type Decision = "ALLOW" | "DENY";

/**
 * Stable, deterministic reason category. Mirrors decision.ReasonCode
 * (agentgate/internal/decision/types.go) exactly, string-for-string.
 */
export type ReasonCode =
  | "policy_allow"
  | "policy_deny"
  | "no_matching_policy"
  | "invalid_identity"
  | "unknown_tool"
  | "malformed_request"
  | "evaluation_error"
  | "no_policy_loaded";

/** All reason codes that resolve to `Decision: "DENY"` in the fail-closed matrix. */
export const DENY_REASON_CODES: readonly ReasonCode[] = [
  "policy_deny",
  "no_matching_policy",
  "invalid_identity",
  "unknown_tool",
  "malformed_request",
  "evaluation_error",
  "no_policy_loaded",
];

/** Reason codes that are structural denials — Cedar was never reached, so PolicyVersion is always "". */
export const PRE_CEDAR_DENY_REASON_CODES: readonly ReasonCode[] = [
  "invalid_identity",
  "unknown_tool",
  "malformed_request",
  "no_policy_loaded",
];

// ---------------------------------------------------------------------------
// Wire shapes — exact JSON contract from docs/.../GO_BACKEND_G1_CONTRACT.md
// and agentgate/internal/mockauthz/wire.go. Field names/casing/optionality
// here MUST match those files exactly; do not "improve" them independently.
// ---------------------------------------------------------------------------

export interface IdentityWire {
  agent_id: string;
  /** Optional; omitted or "" both mean "no delegated human identity". */
  on_behalf_of?: string;
  roles: string[];
}

export interface ToolWire {
  backend_id: string;
  name: string;
}

export interface ClassificationWire {
  known: boolean;
  /** Only meaningful when known === true. */
  risk?: string;
}

export type AttributeValueWire =
  | { type: "string"; value: string }
  | { type: "int"; value: number }
  | { type: "bool"; value: boolean };

/** decision.Request over the wire (agentgate/internal/mockauthz/wire.go: evaluateRequestWire). */
export interface AuthorizationRequestWire {
  execution_id: string;
  workspace_id: string;
  identity: IdentityWire;
  tool: ToolWire;
  classification: ClassificationWire;
  arguments?: Record<string, AttributeValueWire>;
}

/** decision.Result over the wire (agentgate/internal/mockauthz/wire.go: resultWire). */
export interface AuthorizationResultWire {
  decision: string; // validated/narrowed to Decision during parsing
  reason: string; // validated/narrowed to ReasonCode during parsing
  /** May be absent (omitempty) on the wire; always "" or a message once parsed. */
  message?: string;
  /** Absent (omitempty) exactly when Cedar was never reached. */
  policy_version?: string;
  execution_id: string;
}

/** The mock's transport-level error body: `{"error": "..."}"` on HTTP 400 (never a decision.Result). */
export interface TransportErrorWire {
  error: string;
}

// ---------------------------------------------------------------------------
// Domain models — camelCase, used by everything above the parsing boundary.
// ---------------------------------------------------------------------------

export interface Identity {
  agentId: string;
  /** "" means no delegated human identity for this call. */
  onBehalfOf: string;
  roles: string[];
}

export interface ToolRef {
  backendId: string;
  name: string;
}

export interface ToolClassification {
  known: boolean;
  /** "" when known === false (irrelevant in that case). */
  risk: string;
}

export type AttributeValue =
  | { kind: "string"; value: string }
  | { kind: "int"; value: number }
  | { kind: "bool"; value: boolean };

/** decision.Request as consumed by the frontend. */
export interface AuthorizationRequest {
  executionId: string;
  workspaceId: string;
  identity: Identity;
  tool: ToolRef;
  classification: ToolClassification;
  arguments: Record<string, AttributeValue>;
}

/**
 * decision.Result as consumed by the frontend — the immediate external
 * model this ticket set exists to define. Field-by-field mapping to the Go
 * contract is documented in the WS-C G1 report.
 */
export interface AuthorizationResult {
  decision: Decision;
  reason: ReasonCode;
  /** Human-readable detail, always AgentGate-authored. May be "" on ALLOW. */
  message: string;
  /**
   * The exact policy version/hash Cedar evaluated. "" only when Cedar was
   * never reached (see PRE_CEDAR_DENY_REASON_CODES). Never treat "" as "no
   * version known" in the sense of a display gap — it is a meaningful,
   * documented value: "Cedar was not consulted for this decision".
   */
  policyVersion: string;
  executionId: string;
}

/** True exactly when policyVersion is guaranteed empty by the fail-closed matrix. */
export function policyWasReached(result: Pick<AuthorizationResult, "policyVersion">): boolean {
  return result.policyVersion !== "";
}
