import { HttpGovernanceClient, MockGovernanceClient, type GovernanceClient } from "./contract";

export type GovernanceMode = "mock" | "live";

/**
 * mock: MockGovernanceClient (frontend/src/api/governanceClient.ts) — an
 * in-memory, deterministic implementation of the same GovernanceClient
 * interface. No network calls, nothing to run locally.
 *
 * live: HttpGovernanceClient against a real cmd/agentgate instance, routed
 * through the Vite dev-server proxy (/proxy/agentgate -> localhost:8090) to
 * sidestep CORS. Requires AGENTGATE_ADMIN_TOKEN to match what that instance
 * was started with.
 */
export function createGovernanceClient(mode: GovernanceMode, adminToken: string): GovernanceClient {
  if (mode === "mock") {
    return new MockGovernanceClient();
  }
  return new HttpGovernanceClient("/proxy/agentgate", adminToken);
}
