/**
 * Pure, framework-free "render" functions: given a domain model, produce a
 * plain display-safe view object (tone + headline + detail). No React, no
 * React Native, no DOM — deliberately so, since the production UI framework
 * is an open decision (see WS-C G1 report "Limitations"). These functions
 * are what a future component would call; they exist now so AG-UI-G1-03's
 * "each state can render/be exercised without a live backend" is provable
 * today, independent of that later choice.
 *
 * These functions perform no authorization logic of any kind — they only
 * format data the backend already decided.
 */

import type { ApiError } from "../models/api-error.js";
import type { AuthorizationResult } from "../models/decision.js";
import { policyWasReached } from "../models/decision.js";
import type { OperationState } from "../state/operation-state.js";

export type Tone = "allow" | "deny" | "error" | "stale" | "pending";

export interface DecisionView {
  tone: Tone;
  headline: string;
  detail: string;
  policyVersionLabel: string;
  executionId: string;
}

export function toDecisionView(result: AuthorizationResult): DecisionView {
  return {
    tone: result.decision === "ALLOW" ? "allow" : "deny",
    headline: result.decision === "ALLOW" ? "Allowed" : "Denied",
    detail: result.message !== "" ? result.message : result.reason,
    policyVersionLabel: policyWasReached(result) ? result.policyVersion : "not evaluated (policy never reached)",
    executionId: result.executionId,
  };
}

export interface ErrorView {
  tone: "error";
  headline: string;
  detail: string;
}

export function toApiErrorView(error: ApiError): ErrorView {
  switch (error.kind) {
    case "network":
      return { tone: "error", headline: "Could not reach AgentGate", detail: error.message };
    case "unexpected_status":
      return { tone: "error", headline: "Unexpected response", detail: `HTTP ${error.status}` };
    case "transport_rejected":
      return { tone: "error", headline: "Request rejected", detail: error.message };
    case "invalid_response":
      return { tone: "error", headline: "Response did not match the authorization contract", detail: error.message };
  }
}

export interface StaleView {
  tone: "stale";
  headline: string;
  detail: string;
  lastKnown?: DecisionView;
}

export function toStaleView(reason: string, lastKnown: AuthorizationResult | null): StaleView {
  const base: StaleView = {
    tone: "stale",
    headline: "Data may be out of date",
    detail: reason,
  };
  return lastKnown === null ? base : { ...base, lastKnown: toDecisionView(lastKnown) };
}

/**
 * Renders any OperationState<AuthorizationResult> into a single display-safe
 * view. This is the function that proves AG-UI-G1-04's invariant at the
 * view layer: an "apiError"/"stale" state can never produce a DecisionView
 * with tone "allow"/"deny" — it always renders as its own distinct tone.
 */
export type AnyView =
  | ({ status: "idle" | "loading" })
  | ({ status: "succeeded" } & DecisionView)
  | ({ status: "denied" } & { tone: "deny"; headline: string; detail: string })
  | ({ status: "apiError" } & ErrorView)
  | ({ status: "stale" } & StaleView);

export function renderOperationState(state: OperationState<AuthorizationResult>): AnyView {
  switch (state.status) {
    case "idle":
      return { status: "idle" };
    case "loading":
      return { status: "loading" };
    case "succeeded":
      return { status: "succeeded", ...toDecisionView(state.data) };
    case "denied":
      return { status: "denied", tone: "deny", headline: "Denied", detail: state.reason };
    case "apiError":
      return { status: "apiError", ...toApiErrorView(state.error) };
    case "stale":
      return { status: "stale", ...toStaleView(state.reason, state.lastKnown) };
  }
}
