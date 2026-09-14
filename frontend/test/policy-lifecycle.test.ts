import { describe, expect, it } from "vitest";
import { MockGovernanceClient } from "../src/api/governanceClient.js";
import { PolicyLifecycleStore } from "../src/state/policyLifecycleState.js";
import {
  toPolicyBadgeView,
  toLifecycleSummaryView,
  toValidationDisplayView,
} from "../src/view/policy-lifecycle-view.js";
import { g3Fixtures } from "../src/fixtures/governanceFixtures.js";

describe("Frontend Policy Lifecycle State & Views", () => {
  it("manages policy lifecycle transitions and state confirmation", async () => {
    const client = new MockGovernanceClient();
    const ws = "ws-ui-lifecycle";
    const store = new PolicyLifecycleStore(client, ws);

    // Initial state
    expect(store.getState().activeVersion).toBeUndefined();
    expect(store.getState().policies).toHaveLength(0);

    // 1. Validate invalid policy
    const valBad = await store.validateDraft(g3Fixtures.invalidPolicy);
    expect(valBad.valid).toBe(false);
    expect(store.getState().validation?.valid).toBe(false);

    // 2. Validate valid policy
    const valGood = await store.validateDraft(g3Fixtures.validPolicy1);
    expect(valGood.valid).toBe(true);
    expect(store.getState().validation?.valid).toBe(true);

    // 3. Create candidate
    const c1 = await store.submitCandidate(g3Fixtures.validPolicy1, "v1 draft");
    expect(c1.state).toBe("candidate");
    expect(store.getState().policies).toHaveLength(1);

    // 4. Activate candidate
    const actResp = await store.activate(c1.version);
    expect(actResp.active_version).toBe(c1.version);
    expect(store.getState().activeVersion).toBe(c1.version);
    expect(store.getState().activationStatus).toBe("confirmed");

    // 5. Create second candidate
    const c2 = await store.submitCandidate(g3Fixtures.validPolicy2, "v2 draft");
    expect(c2.state).toBe("candidate");

    // 6. Failed activation leaves active unchanged
    await expect(store.activate("unknown-hash")).rejects.toThrow();
    expect(store.getState().activeVersion).toBe(c1.version);

    // 7. Activate c2
    await store.activate(c2.version);
    expect(store.getState().activeVersion).toBe(c2.version);

    // 8. Rollback to c1
    const rbResp = await store.rollback(c1.version);
    expect(rbResp.active_version).toBe(c1.version);
    expect(store.getState().activeVersion).toBe(c1.version);

    // 9. Rollback to unknown leaves active at c1
    await expect(store.rollback("non-existent")).rejects.toThrow();
    expect(store.getState().activeVersion).toBe(c1.version);
  });

  it("renders pure display views accurately", () => {
    // 1. Badges
    const badgeActive = toPolicyBadgeView("active");
    expect(badgeActive.tone).toBe("active");
    expect(badgeActive.label).toBe("Active");

    const badgeCandidate = toPolicyBadgeView("candidate");
    expect(badgeCandidate.tone).toBe("candidate");
    expect(badgeCandidate.label).toBe("Candidate");

    const badgeHistorical = toPolicyBadgeView("historical");
    expect(badgeHistorical.tone).toBe("historical");
    expect(badgeHistorical.label).toBe("Historical");

    // 2. Summary view
    const summary = toLifecycleSummaryView("ws-summary", [
      {
        workspace_id: "ws-summary",
        version: "v1",
        content: "...",
        state: "historical",
        description: "old",
        created_at: "2026-09-13T10:00:00Z",
      },
      {
        workspace_id: "ws-summary",
        version: "v2",
        content: "...",
        state: "active",
        description: "current",
        created_at: "2026-09-13T10:05:00Z",
      },
      {
        workspace_id: "ws-summary",
        version: "v3",
        content: "...",
        state: "candidate",
        description: "draft",
        created_at: "2026-09-13T10:10:00Z",
      },
    ]);

    expect(summary.workspaceId).toBe("ws-summary");
    expect(summary.activeVersion).toBe("v2");
    expect(summary.candidateCount).toBe(1);
    expect(summary.historicalCount).toBe(1);

    // 3. Validation display view
    const valViewOk = toValidationDisplayView({ valid: true, version: "hash-abc" });
    expect(valViewOk.tone).toBe("valid");
    expect(valViewOk.headline).toContain("Valid");

    const valViewErr = toValidationDisplayView({
      valid: false,
      errors: ["unexpected token 'foo'"],
    });
    expect(valViewErr.tone).toBe("invalid");
    expect(valViewErr.headline).toContain("Invalid");
    expect(valViewErr.errorMessages).toHaveLength(1);
  });
});
