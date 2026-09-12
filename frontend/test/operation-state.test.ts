import { describe, expect, it } from "vitest";
import { reduce, isConfirmedSuccess, type OperationState } from "../src/state/operation-state.js";
import type { ApiError } from "../src/models/api-error.js";
import { parseAuthorizationResult } from "../src/parsing/parse-authorization-result.js";
import { wireFixtures } from "../src/fixtures/index.js";

// AG-UI-G1-04: prove that a backend error can never be displayed/represented
// as a successful authorization/policy state. This is exercised against the
// generic OperationState<T> machine (used both for displaying a decision and
// for a hypothetical future policy mutation) by exhaustively driving every
// (state, event) pair, not by spot-checking one happy path.

type Payload = { marker: string };
const SAMPLE: Payload = { marker: "sample-success-payload" };
const SOME_ERROR: ApiError = { kind: "network", message: "connection refused" };

function allStates(): OperationState<Payload>[] {
  return [
    { status: "idle" },
    { status: "loading" },
    { status: "succeeded", data: SAMPLE, confirmedAt: "t0" },
    { status: "denied", reason: "unauthorized", confirmedAt: "t0" },
    { status: "apiError", error: SOME_ERROR, occurredAt: "t0" },
    { status: "stale", lastKnown: SAMPLE, reason: "poll failed", staleSince: "t0" },
    { status: "stale", lastKnown: null, reason: "never fetched", staleSince: "t0" },
  ];
}

describe("operation-state: a failed/denied operation never becomes succeeded", () => {
  it("a 'fail' event from every possible prior state never yields 'succeeded'", () => {
    for (const prior of allStates()) {
      const next = reduce(prior, { type: "fail", error: SOME_ERROR, at: "t1" });
      expect(next.status).toBe("apiError");
      expect(isConfirmedSuccess(next)).toBe(false);
    }
  });

  it("a 'deny' event from every possible prior state never yields 'succeeded'", () => {
    for (const prior of allStates()) {
      const next = reduce(prior, { type: "deny", reason: "forbidden", at: "t1" });
      expect(next.status).toBe("denied");
      expect(isConfirmedSuccess(next)).toBe(false);
    }
  });

  it("'markStale' from every possible prior state never yields 'succeeded'", () => {
    for (const prior of allStates()) {
      const next = reduce(prior, { type: "markStale", reason: "connection lost", at: "t1" });
      expect(next.status).toBe("stale");
      expect(isConfirmedSuccess(next)).toBe(false);
    }
  });

  it("'start' never yields 'succeeded' directly", () => {
    for (const prior of allStates()) {
      const next = reduce(prior, { type: "start" });
      expect(next.status).toBe("loading");
    }
  });

  it("only an explicit 'succeed' event (carrying confirmed data) produces 'succeeded'", () => {
    for (const prior of allStates()) {
      const next = reduce(prior, { type: "succeed", data: SAMPLE, at: "t1" });
      expect(next.status).toBe("succeeded");
      expect(isConfirmedSuccess(next)).toBe(true);
      if (next.status === "succeeded") {
        expect(next.data).toEqual(SAMPLE);
      }
    }
  });

  it("'markStale' after a prior success carries the last-known data forward, not a fabricated success", () => {
    const succeeded: OperationState<Payload> = { status: "succeeded", data: SAMPLE, confirmedAt: "t0" };
    const stale = reduce(succeeded, { type: "markStale", reason: "refresh failed", at: "t1" });
    expect(stale.status).toBe("stale");
    if (stale.status === "stale") {
      expect(stale.lastKnown).toEqual(SAMPLE);
    }
  });

  it("'markStale' with no prior success carries lastKnown = null, never invents data", () => {
    const stale = reduce({ status: "idle" }, { type: "markStale", reason: "never connected", at: "t1" });
    expect(stale.status).toBe("stale");
    if (stale.status === "stale") {
      expect(stale.lastKnown).toBeNull();
    }
  });
});

describe("operation-state driven by real parsed backend fixtures (not hand-typed data)", () => {
  it("an ALLOW fixture, once parsed, drives the state machine to 'succeeded'", () => {
    const parsed = parseAuthorizationResult(wireFixtures.allow);
    expect(parsed.ok).toBe(true);
    if (!parsed.ok) return;
    const next = reduce({ status: "loading" }, { type: "succeed", data: parsed.value, at: "t1" });
    expect(next.status).toBe("succeeded");
  });

  it("a DENY fixture, once parsed, is a 'succeeded' fetch of a DENY decision — not the generic 'denied' mutation-rejection state", () => {
    // Distinguishing these two is deliberate: decision.Result{Decision: DENY}
    // is data the backend successfully returned (a real, confirmed
    // authorization answer) — it is not a failed API call. The generic
    // "denied" status in this state machine is reserved for a hypothetical
    // future mutation being rejected by the backend (e.g. "activate policy:
    // 403 unauthorized"), which is a different situation from "the
    // authorization decision itself was DENY".
    const parsed = parseAuthorizationResult(wireFixtures.denyExplicitForbid);
    expect(parsed.ok).toBe(true);
    if (!parsed.ok) return;
    const next = reduce({ status: "loading" }, { type: "succeed", data: parsed.value, at: "t1" });
    expect(next.status).toBe("succeeded");
    if (next.status === "succeeded") {
      expect(next.data.decision).toBe("DENY");
    }
  });

  it("an incomplete/invalid response fixture cannot reach 'succeeded' at all — it must be routed through 'fail', not 'succeed'", () => {
    const parsed = parseAuthorizationResult(wireFixtures.incompleteResponse);
    expect(parsed.ok).toBe(false);
    if (parsed.ok) return;
    // The only legitimate transition available to a caller holding a parse
    // failure is "fail" - there is no way to construct a "succeed" event
    // without a valid AuthorizationResult, because "succeed" requires
    // `data: AuthorizationResult` and none exists here.
    const next = reduce({ status: "loading" }, { type: "fail", error: parsed.error, at: "t1" });
    expect(next.status).toBe("apiError");
  });
});
