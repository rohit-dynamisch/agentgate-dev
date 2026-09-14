import { describe, it, expect } from "vitest";
import { MockGovernanceClient } from "../src/api/governanceClient.js";

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
