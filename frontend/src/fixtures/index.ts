/**
 * Deterministic fixtures (AG-UI-G1-03).
 *
 * "wire/*" fixtures are raw JSON exactly as it appears on the wire between
 * the G1 mock (agentgate/internal/mockauthz) and a caller — verified
 * byte-for-byte against the real handler for allow.json and
 * deny-invalid-identity.json (see the WS-C G1 report's "Compatibility
 * issues" section for the one documentation-vs-implementation discrepancy
 * found: `message`/`policy_version` use Go's `omitempty` and are omitted
 * entirely when empty, not sent as `""`, which the contract doc's
 * illustrative example does not show).
 *
 * "ui/*" fixtures are frontend-only view-state data (there is no backend
 * wire shape for "stale") representing AG-UI-G1-04's stale/unavailable
 * state.
 *
 * None of these contain a real credential, token, or secret — `on_behalf_of`
 * values are synthetic test identities (`*@example.test`).
 *
 * Deliberately typed `unknown` here (not AuthorizationResult /
 * AuthorizationRequest): every fixture must be pushed through
 * src/parsing/* in tests, proving the parser — not a cast — is what
 * produces the typed model.
 */

import allow from "./wire/allow.json" with { type: "json" };
import denyExplicitForbid from "./wire/deny-explicit-forbid.json" with { type: "json" };
import denyNoMatchingPolicy from "./wire/deny-no-matching-policy.json" with { type: "json" };
import denyInvalidIdentity from "./wire/deny-invalid-identity.json" with { type: "json" };
import denyUnknownTool from "./wire/deny-unknown-tool.json" with { type: "json" };
import denyEvaluationError from "./wire/deny-evaluation-error.json" with { type: "json" };
import denyMalformedRequest from "./wire/deny-malformed-request.json" with { type: "json" };
import denyNoPolicyLoaded from "./wire/deny-no-policy-loaded.json" with { type: "json" };
import transportError from "./wire/transport-error.json" with { type: "json" };
import incompleteResponse from "./wire/incomplete-response.json" with { type: "json" };
import invalidDecisionValue from "./wire/invalid-decision-value.json" with { type: "json" };
import inconsistentDecisionReason from "./wire/inconsistent-decision-reason.json" with { type: "json" };
import requestAllow from "./wire/request-allow.json" with { type: "json" };
import requestWithArguments from "./wire/request-with-arguments.json" with { type: "json" };

import staleWithLastKnownAllow from "./ui/stale-with-last-known-allow.json" with { type: "json" };
import staleNoPriorData from "./ui/stale-no-prior-data.json" with { type: "json" };

export const wireFixtures = {
  /** Valid AuthorizationResult wire bodies — every documented reason code from the fail-closed matrix. */
  allow,
  denyExplicitForbid,
  denyNoMatchingPolicy,
  denyInvalidIdentity,
  denyUnknownTool,
  denyEvaluationError,
  denyMalformedRequest,
  denyNoPolicyLoaded,
  /** Not an AuthorizationResult at all — the mock's transport-level rejection body. */
  transportError,
  /** Deliberately invalid/incomplete AuthorizationResult bodies — must fail to parse. */
  incompleteResponse,
  invalidDecisionValue,
  inconsistentDecisionReason,
  /** Valid AuthorizationRequest wire bodies. */
  requestAllow,
  requestWithArguments,
} as const;

export const uiFixtures = {
  staleWithLastKnownAllow,
  staleNoPriorData,
} as const;
