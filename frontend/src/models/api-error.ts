/**
 * Typed model for a failure to obtain an AuthorizationResult at all.
 *
 * This is deliberately NOT part of `Decision` (decision.ts): the frozen Go
 * contract has exactly two decision values, ALLOW and DENY
 * (docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md, AG-GO-G1-03 —
 * "There is no third 'error' decision"). An ApiError represents the
 * frontend never having received a decision.Result to interpret in the
 * first place — a transport/network failure, an unexpected HTTP status, or
 * a response body that fails to satisfy the contract shape. It must never
 * be coerced into, or displayed as, ALLOW or DENY.
 */

export type ApiError =
  /** fetch()/network failure — no HTTP response was received at all. */
  | { kind: "network"; message: string }
  /** An HTTP response was received but with an unexpected status the caller cannot interpret as either an evaluate response or a documented transport error. */
  | { kind: "unexpected_status"; status: number; body?: string }
  /**
   * The mock's documented transport-level rejection: HTTP 400 with a
   * `{"error": "..."}` body (agentgate/internal/mockauthz/handler.go) — the
   * request body never became a decision.Request, so there is no
   * decision.Result to report.
   */
  | { kind: "transport_rejected"; status: number; message: string }
  /**
   * An HTTP 200 response was received, but its body does not satisfy the
   * AuthorizationResultWire contract (missing required field, unknown
   * decision/reason value, wrong type, etc.). This is the "incomplete
   * data" case from AG-UI-G1-03/04: never guess a decision from a response
   * that doesn't parse.
   */
  | { kind: "invalid_response"; message: string };

export function apiErrorMessage(error: ApiError): string {
  switch (error.kind) {
    case "network":
      return `network error: ${error.message}`;
    case "unexpected_status":
      return `unexpected HTTP status ${error.status}`;
    case "transport_rejected":
      return `request rejected before evaluation (HTTP ${error.status}): ${error.message}`;
    case "invalid_response":
      return `response did not match the authorization result contract: ${error.message}`;
  }
}
