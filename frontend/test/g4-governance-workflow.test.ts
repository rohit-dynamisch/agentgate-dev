import { describe, it, expect } from "vitest";
import { MockGovernanceClient } from "../src/api/governanceClient.js";
import { PolicyLifecycleStore } from "../src/state/policyLifecycleState.js";
import {
  toDryRunComparisonView,
  toOperationStatusView,
} from "../src/view/policy-lifecycle-view.js";

describe("G4 governance workflow", () => {
  it("dry-run compare returns paired results", async () => {
    const client = new MockGovernanceClient();
    const result = await client.dryRunCompare("ws-1", "v1", [
      { principal_id: "a1", principal_roles: ["reader"], resource_id: "b1/t1", resource_risk: "read" },
    ]);
    expect(result.candidate_version).toBe("v1");
    expect(result.results.length).toBe(1);
    expect(result.results[0]).toHaveProperty("active_decision");
    expect(result.results[0]).toHaveProperty("candidate_decision");
    expect(result.results[0]).toHaveProperty("changed");
  });
});

describe("G4 lifecycle store workflow", () => {
  it("dry-run does not change active version", async () => {
    const client = new MockGovernanceClient();
    const store = new PolicyLifecycleStore(client, "ws-1");
    // Create and activate policy A
    const candA = await store.submitCandidate("permit(principal, action, resource);", "policy A");
    await store.activate(candA.version);
    const stateBeforeDryRun = store.getState();
    // Create candidate B
    const candB = await store.submitCandidate("forbid(principal, action, resource);", "policy B");
    // Dry-run B
    await store.dryRunCompare(candB.version, [
      { principal_id: "a1", principal_roles: ["reader"], resource_id: "b1/t1", resource_risk: "read" },
    ]);
    const stateAfterDryRun = store.getState();
    expect(stateAfterDryRun.activeVersion).toBe(stateBeforeDryRun.activeVersion);
    expect(stateAfterDryRun.dryRunResult).toBeDefined();
  });

  it("rollback tracks status correctly", async () => {
    const client = new MockGovernanceClient();
    const store = new PolicyLifecycleStore(client, "ws-1");
    const candA = await store.submitCandidate("permit(principal, action, resource);", "A");
    await store.activate(candA.version);
    const candB = await store.submitCandidate("forbid(principal, action, resource);", "B");
    await store.activate(candB.version);
    await store.rollback(candA.version);
    const state = store.getState();
    expect(state.activeVersion).toBe(candA.version);
    expect(state.rollbackStatus).toBe("confirmed");
  });
});

describe("G4 view renderers", () => {
  it("toDryRunComparisonView renders changed status", () => {
    const view = toDryRunComparisonView({
      active_decision: "ALLOW",
      active_reason: "policy_allow",
      active_policy_version: "aaa",
      candidate_decision: "DENY",
      candidate_reason: "policy_deny",
      candidate_policy_version: "bbb",
      changed: true,
    });
    expect(view.changed).toBe(true);
    expect(view.headline).toContain("Changed");
  });

  it("toOperationStatusView renders pending and confirmed", () => {
    expect(toOperationStatusView("rolling_back").tone).toBe("pending");
    expect(toOperationStatusView("confirmed").tone).toBe("success");
  });
});
