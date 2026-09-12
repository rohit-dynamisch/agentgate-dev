/**
 * Parses raw JSON (as received over the wire from the AgentGate mock /
 * eventual ext_authz-fronting API) into the typed AuthorizationResult
 * domain model — or fails explicitly.
 *
 * This is the frontend's contract-enforcement boundary: it never invents a
 * value, never defaults an unrecognized `decision`/`reason` string to
 * something plausible, and never returns a partially-valid result. A
 * response that does not exactly satisfy the frozen wire shape is
 * `invalid_response`, not a best-effort guess — this is what lets
 * AG-UI-G1-04's "a backend error cannot be displayed as a successful
 * authorization state" hold even against a malformed/incomplete backend
 * response, not just an HTTP-level error.
 */

import type { ApiError } from "../models/api-error.js";
import {
  DENY_REASON_CODES,
  type AuthorizationResult,
  type AuthorizationResultWire,
  type Decision,
  type ReasonCode,
} from "../models/decision.js";

const VALID_DECISIONS: readonly Decision[] = ["ALLOW", "DENY"];
const VALID_REASONS: readonly ReasonCode[] = [
  "policy_allow",
  "policy_deny",
  "no_matching_policy",
  "invalid_identity",
  "unknown_tool",
  "malformed_request",
  "evaluation_error",
  "no_policy_loaded",
];

export type ParseResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: ApiError };

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

/**
 * Validates and converts a raw JSON value into an AuthorizationResult.
 * Never throws; returns a discriminated ParseResult so callers cannot
 * forget to handle the failure path.
 */
export function parseAuthorizationResult(raw: unknown): ParseResult<AuthorizationResult> {
  if (!isRecord(raw)) {
    return { ok: false, error: { kind: "invalid_response", message: "response body is not a JSON object" } };
  }

  const wire = raw as Partial<AuthorizationResultWire>;

  if (typeof wire.decision !== "string" || !VALID_DECISIONS.includes(wire.decision as Decision)) {
    return {
      ok: false,
      error: { kind: "invalid_response", message: `"decision" is missing or not one of ${VALID_DECISIONS.join("/")}` },
    };
  }
  if (typeof wire.reason !== "string" || !VALID_REASONS.includes(wire.reason as ReasonCode)) {
    return {
      ok: false,
      error: { kind: "invalid_response", message: `"reason" is missing or not a recognized reason code` },
    };
  }
  if (wire.message !== undefined && typeof wire.message !== "string") {
    return { ok: false, error: { kind: "invalid_response", message: `"message" must be a string when present` } };
  }
  if (wire.policy_version !== undefined && typeof wire.policy_version !== "string") {
    return { ok: false, error: { kind: "invalid_response", message: `"policy_version" must be a string when present` } };
  }
  if (typeof wire.execution_id !== "string" || wire.execution_id === "") {
    return { ok: false, error: { kind: "invalid_response", message: `"execution_id" is missing or empty` } };
  }

  const decision = wire.decision as Decision;
  const reason = wire.reason as ReasonCode;

  // Cross-field contract check: DENY reasons must never carry decision ALLOW
  // and vice versa. A response violating this is internally inconsistent —
  // treat it as invalid rather than trusting one field over the other.
  const reasonImpliesDeny = (DENY_REASON_CODES as readonly string[]).includes(reason);
  if (decision === "ALLOW" && reasonImpliesDeny) {
    return {
      ok: false,
      error: { kind: "invalid_response", message: `decision "ALLOW" is inconsistent with deny reason "${reason}"` },
    };
  }
  if (decision === "DENY" && reason === "policy_allow") {
    return {
      ok: false,
      error: { kind: "invalid_response", message: `decision "DENY" is inconsistent with reason "policy_allow"` },
    };
  }

  return {
    ok: true,
    value: {
      decision,
      reason,
      message: wire.message ?? "",
      policyVersion: wire.policy_version ?? "",
      executionId: wire.execution_id,
    },
  };
}

/** Parses the mock's documented transport-error body: `{"error": "..."}`. */
export function parseTransportError(raw: unknown, status: number): ApiError {
  if (isRecord(raw) && typeof raw.error === "string") {
    return { kind: "transport_rejected", status, message: raw.error };
  }
  return { kind: "invalid_response", message: "expected a transport error body ({ error: string }) but got something else" };
}
