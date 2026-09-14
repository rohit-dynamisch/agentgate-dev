/**
 * G2 Tool Inventory Models
 *
 * Framework-agnostic TypeScript types mirroring the Go backend's
 * internal/toolregistry package (G2-GO-03/04, commit c6a5acf).
 *
 * Source of truth: agentgate/internal/toolregistry/toolregistry.go
 *
 * Key invariants preserved from Go:
 * - Unknown tools NEVER inherit classification from a same-named tool on
 *   another backend. ToolIDView is the full identity key.
 * - DriftStatus "detected" means governance review required before
 *   authorization can proceed.
 * - ToolGovernanceState preserves all fail-closed semantics —
 *   "unknown", "drifted", "missing_risk" are distinct from "authorized".
 */

// ---------------------------------------------------------------------------
// Tool identity (mirrors toolregistry.ToolID in Go)
// ---------------------------------------------------------------------------

/**
 * Canonical tool identity. The combination (backendId + toolName) is unique
 * within a workspace. Same name on a different backend is a DIFFERENT tool.
 * Mirrors internal/toolregistry.ToolID.
 */
export interface ToolIDView {
  /** Identifies the MCP backend/resource serving this tool. */
  backendId: string;
  /** Name the tool is registered under on this backend. */
  toolName: string;
}

export function toolIDString(id: ToolIDView): string {
  return `${id.backendId}/${id.toolName}`;
}

// ---------------------------------------------------------------------------
// Risk classification (mirrors toolregistry.RiskLevel in Go)
// ---------------------------------------------------------------------------

/**
 * Closed vocabulary of tool risk levels.
 * Mirrors internal/toolregistry.RiskLevel.
 * Must stay in sync with fixturepolicy.CedarSource risk values.
 */
export type RiskLevel = "read" | "write" | "destructive";

export const ALL_RISK_LEVELS: readonly RiskLevel[] = ["read", "write", "destructive"];

export function isValidRiskLevel(s: string): s is RiskLevel {
  return ALL_RISK_LEVELS.includes(s as RiskLevel);
}

// ---------------------------------------------------------------------------
// Schema fingerprint (mirrors toolregistry.SchemaFingerprint in Go)
// ---------------------------------------------------------------------------

/**
 * Opaque hex string representing the SHA-256 of the tool's canonical
 * input schema JSON (keys sorted). 64 hex characters.
 *
 * Algorithm note: SHA-256 of canonical JSON is used in G2 but is NOT frozen
 * as a repository decision (O-005). If O-005 resolves differently, stored
 * fingerprints must be recomputed.
 *
 * Mirrors internal/toolregistry.SchemaFingerprint.
 */
export type SchemaFingerprint = string;

// ---------------------------------------------------------------------------
// Drift status (mirrors toolregistry.DriftStatus in Go)
// ---------------------------------------------------------------------------

/**
 * Schema drift detection result.
 * Mirrors internal/toolregistry.DriftStatus.
 */
export type DriftStatus =
  /** Live fingerprint matches governance record. Tool eligible for authorization. */
  | "none"
  /** Live fingerprint differs from governance record. Governance review required. */
  | "detected"
  /** No governance record exists. Tool cannot inherit classification. */
  | "unknown_tool";

export function isDriftFailing(status: DriftStatus): boolean {
  return status !== "none";
}

// ---------------------------------------------------------------------------
// Governance record view (mirrors toolregistry.GovernanceRecord in Go)
// ---------------------------------------------------------------------------

/**
 * Authoritative governance state for one tool, as seen by the operator UI.
 * Mirrors internal/toolregistry.GovernanceRecord.
 *
 * Semantics:
 * - known=false → tool has no governance record → fails closed.
 * - driftStatus="detected" → schema changed → fails closed.
 * - risk undefined/missing → fails closed.
 */
export interface ToolGovernanceView {
  toolId: ToolIDView;
  /** False for any tool without a governance record. */
  known: boolean;
  /**
   * Meaningful only when known=true.
   * Undefined when known=false or risk is missing (both fail-closed).
   */
  risk?: RiskLevel;
  /** The registered schema fingerprint. Undefined when unknown. */
  registeredFingerprint?: SchemaFingerprint;
  /** Drift status between live schema and registered fingerprint. */
  driftStatus: DriftStatus;
}

// ---------------------------------------------------------------------------
// Operator-facing governance state (derived summary)
// ---------------------------------------------------------------------------

/**
 * High-level operator-facing state summarizing a tool's governance health.
 * This is a derived view for display — the authoritative check is always
 * GovernanceRecord.Validate() on the Go backend.
 */
export type ToolGovernanceState =
  /** Tool is known, no drift, valid risk — eligible for authorization. */
  | "authorized"
  /** Tool has no governance record. Authorization denied. */
  | "unknown"
  /** Schema has drifted since registration. Authorization denied. */
  | "drifted"
  /** Tool is known but risk field is missing or unrecognized. */
  | "missing_risk"
  /** Governance record exists but is stale (timestamp-based, future use). */
  | "stale"
  /** Unexpected error fetching governance state. */
  | "error";

export const FAILING_GOVERNANCE_STATES: readonly ToolGovernanceState[] = [
  "unknown",
  "drifted",
  "missing_risk",
  "stale",
  "error",
];

export function isGovernanceFailing(state: ToolGovernanceState): boolean {
  return FAILING_GOVERNANCE_STATES.includes(state);
}

/**
 * Derives the operator-facing ToolGovernanceState from a GovernanceView.
 * Does NOT perform authorization — that is the backend's job.
 */
export function deriveGovernanceState(gov: ToolGovernanceView): ToolGovernanceState {
  if (!gov.known) return "unknown";
  if (gov.driftStatus === "detected") return "drifted";
  if (!gov.risk || !isValidRiskLevel(gov.risk)) return "missing_risk";
  return "authorized";
}
