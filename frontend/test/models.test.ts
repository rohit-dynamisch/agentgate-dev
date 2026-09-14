import { describe, expect, it } from "vitest";
import { parseAuthorizationResult } from "../src/parsing/parse-authorization-result.js";
import { parseAuthorizationRequest } from "../src/parsing/parse-authorization-request.js";
import { wireFixtures } from "../src/fixtures/index.js";
import { policyWasReached } from "../src/models/decision.js";

// AG-UI-G1-02: fixtures deserialize correctly into typed models; field
// mapping matches the frozen Go contract exactly.

describe("parseAuthorizationResult — valid fixtures", () => {
  it("parses ALLOW", () => {
    const r = parseAuthorizationResult(wireFixtures.allow);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.decision).toBe("ALLOW");
    expect(r.value.reason).toBe("policy_allow");
    expect(r.value.message).toBe(""); // absent on the wire -> defaults to ""
    expect(r.value.policyVersion).toBe(
      "504c632d1057fc09060a8263928618e51e6b2a8b8bc6350d253d3c3e37e89e19"
    );
    expect(r.value.executionId).toBe("exec-123");
    expect(policyWasReached(r.value)).toBe(true);
  });

  it("parses DENY / policy_deny (explicit forbid)", () => {
    const r = parseAuthorizationResult(wireFixtures.denyExplicitForbid);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.decision).toBe("DENY");
    expect(r.value.reason).toBe("policy_deny");
    expect(policyWasReached(r.value)).toBe(true);
  });

  it("parses DENY / no_matching_policy", () => {
    const r = parseAuthorizationResult(wireFixtures.denyNoMatchingPolicy);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.reason).toBe("no_matching_policy");
  });

  it("parses DENY / invalid_identity with policy_version empty (Cedar never reached)", () => {
    const r = parseAuthorizationResult(wireFixtures.denyInvalidIdentity);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.reason).toBe("invalid_identity");
    expect(r.value.policyVersion).toBe("");
    expect(policyWasReached(r.value)).toBe(false);
    expect(r.value.message).toBe("agent id is missing");
  });

  it("parses DENY / unknown_tool", () => {
    const r = parseAuthorizationResult(wireFixtures.denyUnknownTool);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.reason).toBe("unknown_tool");
    expect(policyWasReached(r.value)).toBe(false);
  });

  it("parses DENY / evaluation_error, with policy_version present (Cedar WAS reached)", () => {
    const r = parseAuthorizationResult(wireFixtures.denyEvaluationError);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.decision).toBe("DENY");
    expect(r.value.reason).toBe("evaluation_error");
    expect(policyWasReached(r.value)).toBe(true);
  });

  it("parses DENY / malformed_request", () => {
    const r = parseAuthorizationResult(wireFixtures.denyMalformedRequest);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.reason).toBe("malformed_request");
    expect(policyWasReached(r.value)).toBe(false);
  });

  it("parses DENY / no_policy_loaded", () => {
    const r = parseAuthorizationResult(wireFixtures.denyNoPolicyLoaded);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.reason).toBe("no_policy_loaded");
    expect(policyWasReached(r.value)).toBe(false);
  });
});

describe("parseAuthorizationResult — invalid/incomplete fixtures never become a decision", () => {
  it("rejects a response missing required fields rather than guessing", () => {
    const r = parseAuthorizationResult(wireFixtures.incompleteResponse);
    expect(r.ok).toBe(false);
    if (r.ok) return;
    expect(r.error.kind).toBe("invalid_response");
  });

  it("rejects an unrecognized decision value", () => {
    const r = parseAuthorizationResult(wireFixtures.invalidDecisionValue);
    expect(r.ok).toBe(false);
  });

  it("rejects a decision/reason combination that contradicts the fail-closed matrix", () => {
    const r = parseAuthorizationResult(wireFixtures.inconsistentDecisionReason);
    expect(r.ok).toBe(false);
  });

  it("never parses the transport-error body as an AuthorizationResult", () => {
    const r = parseAuthorizationResult(wireFixtures.transportError);
    expect(r.ok).toBe(false);
  });
});

describe("parseAuthorizationRequest — identity/tool/correlation fields", () => {
  it("parses a request with no arguments", () => {
    const r = parseAuthorizationRequest(wireFixtures.requestAllow);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.executionId).toBe("exec-123");
    expect(r.value.workspaceId).toBe("ws-1");
    expect(r.value.identity.agentId).toBe("agent-1");
    expect(r.value.identity.onBehalfOf).toBe("");
    expect(r.value.identity.roles).toEqual(["reader"]);
    expect(r.value.tool).toEqual({ backendId: "backend-a", name: "read_schedule" });
    expect(r.value.classification).toEqual({ known: true, risk: "read" });
  });

  it("parses a request with declared arguments and on_behalf_of", () => {
    const r = parseAuthorizationRequest(wireFixtures.requestWithArguments);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.value.identity.onBehalfOf).toBe("alice@example.test");
    expect(r.value.arguments.amount).toEqual({ kind: "int", value: 500 });
  });
});
