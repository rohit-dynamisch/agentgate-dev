/**
 * Generic async-operation state machine (AG-UI-G1-04).
 *
 * This models BOTH:
 *   (a) fetching/displaying an authorization decision (ALLOW/DENY/error), and
 *   (b) a hypothetical future backend mutation (e.g. G3's "activate policy"),
 *       modeled generically here since that API does not exist yet and must
 *       not be invented in G1 — only the shape of "what can a mutation's
 *       outcome be" is defined.
 *
 * The load-bearing property this module exists to prove (AG-UI-G1-04's DoD):
 *
 *   A failed operation must never transition to a "succeeded" state.
 *
 * This is enforced structurally: `reduce` only ever produces `succeeded`
 * from the `"succeed"` event, and that event carries the already-parsed,
 * already-validated success payload (see src/parsing). There is no code
 * path from a `"fail"` or `"deny"` event to a `"succeeded"` status — this is
 * verified in test/operation-state.test.ts by exhaustively driving every
 * (state, event) pair, not merely spot-checking a few.
 *
 * This module makes no authorization or policy decision itself — it only
 * tracks what a UI has been told by the backend.
 */

import type { ApiError } from "../models/api-error.js";

export type OperationState<TSuccess> =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "succeeded"; data: TSuccess; confirmedAt: string }
  | { status: "denied"; reason: string; confirmedAt: string }
  | { status: "apiError"; error: ApiError; occurredAt: string }
  | {
      status: "stale";
      /** The last data known to be good, if any was ever obtained. */
      lastKnown: TSuccess | null;
      reason: string;
      staleSince: string;
    };

export type OperationEvent<TSuccess> =
  | { type: "start" }
  | { type: "succeed"; data: TSuccess; at: string }
  | { type: "deny"; reason: string; at: string }
  | { type: "fail"; error: ApiError; at: string }
  | { type: "markStale"; reason: string; at: string };

/**
 * Pure reducer. Given any current state and any event, returns the next
 * state. No network/backend access, no timers, no side effects — a UI
 * layer drives this from its own fetch/mutation logic.
 */
export function reduce<TSuccess>(
  state: OperationState<TSuccess>,
  event: OperationEvent<TSuccess>
): OperationState<TSuccess> {
  switch (event.type) {
    case "start":
      return { status: "loading" };

    case "succeed":
      return { status: "succeeded", data: event.data, confirmedAt: event.at };

    case "deny":
      return { status: "denied", reason: event.reason, confirmedAt: event.at };

    case "fail":
      // A failure NEVER produces "succeeded". It also does not silently
      // keep showing whatever was previously displayed as if nothing
      // happened - stale-vs-error is a deliberate choice the caller makes
      // via markStale, not something "fail" does implicitly.
      return { status: "apiError", error: event.error, occurredAt: event.at };

    case "markStale": {
      const lastKnown = state.status === "succeeded" ? state.data : state.status === "stale" ? state.lastKnown : null;
      return { status: "stale", lastKnown, reason: event.reason, staleSince: event.at };
    }
  }
}

/** True for any state that must NOT be rendered as a confirmed success. */
export function isUnconfirmed<TSuccess>(state: OperationState<TSuccess>): boolean {
  return state.status !== "succeeded";
}

/** True only for a state the UI may present as a confirmed successful operation. */
export function isConfirmedSuccess<TSuccess>(state: OperationState<TSuccess>): state is Extract<OperationState<TSuccess>, { status: "succeeded" }> {
  return state.status === "succeeded";
}
