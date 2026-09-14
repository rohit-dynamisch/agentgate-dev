import { describe, expect, it } from "vitest";
import { wireFixtures, uiFixtures } from "../src/fixtures/index.js";
import { parseAuthorizationResult } from "../src/parsing/parse-authorization-result.js";
import { parseTransportError } from "../src/parsing/parse-authorization-result.js";
import { renderOperationState, toStaleView } from "../src/view/decision-view.js";
import type { OperationState } from "../src/state/operation-state.js";
import type { AuthorizationResult } from "../src/models/decision.js";

// AG-UI-G1-03 DoD: "each state can render/be exercised without a live
// backend". No network access anywhere in this file — every input is a
// fixture already committed to the repo.

function stateFromWireFixture(fixture: unknown): OperationState<AuthorizationResult> {
  const parsed = parseAuthorizationResult(fixture);
  if (parsed.ok) {
    return { status: "succeeded", data: parsed.value, confirmedAt: "2026-09-12T00:00:00Z" };
  }
  return { status: "apiError", error: parsed.error, occurredAt: "2026-09-12T00:00:00Z" };
}

describe("every ALLOW/DENY fixture renders to a distinct, correct tone with no backend", () => {
  it("ALLOW renders tone 'allow'", () => {
    const view = renderOperationState(stateFromWireFixture(wireFixtures.allow));
    expect(view.status).toBe("succeeded");
    if (view.status === "succeeded") {
      expect(view.tone).toBe("allow");
      expect(view.policyVersionLabel).not.toBe("not evaluated (policy never reached)");
    }
  });

  for (const [name, fixture] of [
    ["denyExplicitForbid", wireFixtures.denyExplicitForbid],
    ["denyNoMatchingPolicy", wireFixtures.denyNoMatchingPolicy],
    ["denyInvalidIdentity", wireFixtures.denyInvalidIdentity],
    ["denyUnknownTool", wireFixtures.denyUnknownTool],
    ["denyEvaluationError", wireFixtures.denyEvaluationError],
    ["denyMalformedRequest", wireFixtures.denyMalformedRequest],
    ["denyNoPolicyLoaded", wireFixtures.denyNoPolicyLoaded],
  ] as const) {
    it(`${name} renders tone 'deny', never 'allow'`, () => {
      const view = renderOperationState(stateFromWireFixture(fixture));
      expect(view.status).toBe("succeeded");
      if (view.status === "succeeded") {
        expect(view.tone).toBe("deny");
      }
    });
  }
});

describe("authorization-error fixtures render as errors, never as a decision", () => {
  it("the mock's transport-rejection body renders as an error view", () => {
    const err = parseTransportError(wireFixtures.transportError, 400);
    expect(err.kind).toBe("transport_rejected");
    const view = renderOperationState({ status: "apiError", error: err, occurredAt: "t" });
    expect(view.status).toBe("apiError");
    if (view.status === "apiError") {
      expect(view.tone).toBe("error");
    }
  });

  it("an incomplete response renders as an error view, never as ALLOW/DENY", () => {
    const state = stateFromWireFixture(wireFixtures.incompleteResponse);
    expect(state.status).toBe("apiError");
    const view = renderOperationState(state);
    expect(view.status).toBe("apiError");
    if (view.status === "apiError") {
      expect(view.tone).toBe("error");
    }
  });

  it("an inconsistent decision/reason response renders as an error view", () => {
    const state = stateFromWireFixture(wireFixtures.inconsistentDecisionReason);
    expect(state.status).toBe("apiError");
  });
});

describe("stale/unavailable UI fixtures render without a live backend", () => {
  it("stale-with-last-known-allow renders 'stale' tone but keeps the last-known decision visible and clearly labeled", () => {
    const fixture = uiFixtures.staleWithLastKnownAllow;
    const parsedLastKnown = parseAuthorizationResult(fixture.lastKnown);
    expect(parsedLastKnown.ok).toBe(true);
    if (!parsedLastKnown.ok) return;

    const view = toStaleView(fixture.reason, parsedLastKnown.value);
    expect(view.tone).toBe("stale");
    expect(view.lastKnown).toBeDefined();
    expect(view.lastKnown?.tone).toBe("allow");
  });

  it("stale-no-prior-data renders 'stale' tone with no fabricated decision", () => {
    const fixture = uiFixtures.staleNoPriorData;
    expect(fixture.lastKnown).toBeNull();
    const view = toStaleView(fixture.reason, null);
    expect(view.tone).toBe("stale");
    expect(view.lastKnown).toBeUndefined();
  });
});
