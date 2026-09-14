/**
 * G2 Identity Governance Models
 *
 * Framework-agnostic TypeScript types mirroring the Go backend's
 * internal/identity package (G2-GO-02, commit c6a5acf).
 *
 * Source of truth: agentgate/internal/identity/mapper.go
 *
 * IMPORTANT: These types model identity MAPPING state only.
 * Raw bearer tokens or JWT signatures MUST NEVER appear here.
 * This package only models the output of gateway-validated claim mapping.
 *
 * No authorization logic lives here — all authorization is performed by
 * the Go backend's decision.Engine. These types are for operator-facing
 * UI display and state tracking only.
 */

// ---------------------------------------------------------------------------
// Claim mapping configuration (mirrors MapperConfig in Go)
// ---------------------------------------------------------------------------

/**
 * Operator-visible identity mapping configuration.
 * Mirrors internal/identity.MapperConfig.
 * Claim names are deployment-specific — never hardcoded.
 */
export interface IdentityMappingConfig {
  /** Claim key whose value becomes the agent's ID. Required. */
  agentIdClaim: string;
  /** Claim key for the space-separated role list. Required. */
  rolesClaim: string;
  /**
   * Claim key for delegated human identity. Optional.
   * When absent/empty, on_behalf_of is never read.
   */
  onBehalfOfClaim?: string;
}

// ---------------------------------------------------------------------------
// Mapping status (mirrors FailureClass in Go)
// ---------------------------------------------------------------------------

/**
 * Outcome status of an identity claim mapping operation.
 * Mirrors the FailureClass enum in internal/identity/mapper.go.
 * "ok" means MappedIdentity was produced; all others mean fail-closed.
 */
export type IdentityMappingStatus =
  | "ok"
  /** Required claim key was absent from the claims map. */
  | "missing_claim"
  /** Claim key was present but its value was blank/unusable. */
  | "malformed_claim"
  /** Conflicting identity signals (e.g. on_behalf_of == agent_id). */
  | "ambiguous_identity"
  /** Roles claim present but resolved to an empty set. */
  | "missing_roles"
  /** Unexpected/internal error (should not occur in production). */
  | "error";

/** Status values that represent a failed (fail-closed) mapping. */
export const FAILED_IDENTITY_MAPPING_STATUSES: readonly IdentityMappingStatus[] = [
  "missing_claim",
  "malformed_claim",
  "ambiguous_identity",
  "missing_roles",
  "error",
];

export function isIdentityMappingFailed(status: IdentityMappingStatus): boolean {
  return FAILED_IDENTITY_MAPPING_STATUSES.includes(status);
}

// ---------------------------------------------------------------------------
// Mapped identity view (operator-facing, no raw credentials)
// ---------------------------------------------------------------------------

/**
 * Operator-visible view of a successfully mapped identity.
 * Mirrors internal/identity.MappedIdentity.
 *
 * NEVER expose raw JWT, bearer token, or signature material here.
 * This type represents post-mapping, post-validation identity only.
 */
export interface MappedIdentityView {
  /** The authoritative identifier for the calling agent. */
  agentId: string;
  /**
   * The human the agent is acting for.
   * Empty string means no delegated identity for this call.
   */
  onBehalfOf: string;
  /**
   * Non-empty set of role assignments.
   * Guaranteed non-empty on successful mapping.
   */
  roles: readonly string[];
}

// ---------------------------------------------------------------------------
// Mapping result (discriminated union: success or failure)
// ---------------------------------------------------------------------------

export interface IdentityMappingSuccess {
  status: "ok";
  identity: MappedIdentityView;
}

export interface IdentityMappingFailure {
  status: Exclude<IdentityMappingStatus, "ok">;
  /** Human-readable reason, for operator display. Never parse programmatically. */
  message: string;
  /**
   * The specific claim key that caused the failure, when applicable.
   * Undefined for ambiguous_identity (involves two claims).
   */
  claimKey?: string;
}

export type IdentityMappingResult = IdentityMappingSuccess | IdentityMappingFailure;

export function isIdentityMappingSuccess(
  result: IdentityMappingResult
): result is IdentityMappingSuccess {
  return result.status === "ok";
}
