/**
 * Single seam through which this app touches AgentGate's framework-agnostic
 * frontend contract layer (../../frontend/src). That package deliberately
 * ships no rendering code (see frontend/package.json and
 * frontend/src/view/decision-view.ts's file header) because the production
 * UI framework was an open item as of G1-G4. This app is the prototype that
 * exercises that layer with real React rendering; everything it imports
 * from ../../frontend/src is re-exported here so no other file in this app
 * needs to know the cross-package relative path.
 */

export type {
  GovernanceClient,
  FetchFunction,
  FetchRequestOptions,
  FetchResponse,
} from "../../../frontend/src/api/governanceClient.js";
export { MockGovernanceClient, HttpGovernanceClient } from "../../../frontend/src/api/governanceClient.js";

export type {
  PolicyState,
  PolicyRecord,
  ValidateResponse,
  ActivateResponse,
  RollbackResponse,
  ListPoliciesResponse,
  PreviewSample,
  PreviewResult,
  PreviewResponse,
  DryRunCompareResult,
  DryRunCompareResponse,
  DryRunSample,
} from "../../../frontend/src/models/governance.js";
export { GovernanceApiError } from "../../../frontend/src/models/governance.js";

export type { PolicyLifecycleState } from "../../../frontend/src/state/policyLifecycleState.js";
export { PolicyLifecycleStore } from "../../../frontend/src/state/policyLifecycleState.js";

export type {
  BadgeTone,
  PolicyBadgeView,
  LifecycleSummaryView,
  ValidationDisplayView,
  DryRunComparisonView,
  OperationStatusView,
} from "../../../frontend/src/view/policy-lifecycle-view.js";
export {
  toPolicyBadgeView,
  toLifecycleSummaryView,
  toValidationDisplayView,
  toDryRunComparisonView,
  toOperationStatusView,
} from "../../../frontend/src/view/policy-lifecycle-view.js";

export { g3Fixtures } from "../../../frontend/src/fixtures/governanceFixtures.js";

export type {
  Decision,
  ReasonCode,
  Identity,
  ToolRef,
  ToolClassification,
  AttributeValue,
  AuthorizationRequest,
  AuthorizationRequestWire,
  AuthorizationResult,
} from "../../../frontend/src/models/decision.js";
export { policyWasReached } from "../../../frontend/src/models/decision.js";

export type { ApiError } from "../../../frontend/src/models/api-error.js";
export { apiErrorMessage } from "../../../frontend/src/models/api-error.js";

export type { OperationState, OperationEvent } from "../../../frontend/src/state/operation-state.js";
export { reduce } from "../../../frontend/src/state/operation-state.js";

export type { Tone, DecisionView, ErrorView, StaleView, AnyView } from "../../../frontend/src/view/decision-view.js";
export { toDecisionView, toApiErrorView, toStaleView, renderOperationState } from "../../../frontend/src/view/decision-view.js";

export type { ParseResult } from "../../../frontend/src/parsing/parse-authorization-result.js";
export { parseAuthorizationResult, parseTransportError } from "../../../frontend/src/parsing/parse-authorization-result.js";

export { wireFixtures, uiFixtures } from "../../../frontend/src/fixtures/index.js";
