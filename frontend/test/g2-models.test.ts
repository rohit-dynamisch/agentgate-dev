import { describe, expect, it } from "vitest";
import {
  isIdentityMappingFailed,
  isIdentityMappingSuccess,
  IdentityMappingResult,
} from "../src/models/identity-governance.js";
import {
  deriveGovernanceState,
  isDriftFailing,
  isGovernanceFailing,
  toolIDString,
  ToolGovernanceView,
} from "../src/models/tool-inventory.js";
import {
  resolveArguments,
  validateDeclarationSet,
  ArgumentDeclarationView,
} from "../src/models/argument-declaration.js";
import { g2Fixtures } from "../src/fixtures/g2/index.js";

describe("G2 Identity Governance Models", () => {
  it("recognizes successful identity mapping", () => {
    const successResult: IdentityMappingResult = {
      status: "ok",
      identity: {
        agentId: "agent-fixture-001",
        onBehalfOf: "",
        roles: ["reader"],
      },
    };
    expect(isIdentityMappingSuccess(successResult)).toBe(true);
    expect(isIdentityMappingFailed(successResult.status)).toBe(false);
  });

  it("recognizes all failure classes as fail-closed", () => {
    const failures = [
      "missing_claim",
      "malformed_claim",
      "ambiguous_identity",
      "missing_roles",
      "error",
    ] as const;

    for (const status of failures) {
      expect(isIdentityMappingFailed(status)).toBe(true);
      const res: IdentityMappingResult = {
        status,
        message: `Failure: ${status}`,
      };
      expect(isIdentityMappingSuccess(res)).toBe(false);
    }
  });

  it("verifies identity fixtures match expected statuses", () => {
    const idFix = g2Fixtures.identity;
    expect(idFix.validMinimal.expectedStatus).toBe("ok");
    expect(idFix.validWithDelegate.expectedStatus).toBe("ok");
    expect(idFix.missingAgentId.expectedStatus).toBe("missing_claim");
    expect(idFix.malformedAgentId.expectedStatus).toBe("malformed_claim");
    expect(idFix.ambiguousIdentity.expectedStatus).toBe("ambiguous_identity");
  });
});

describe("G2 Tool Inventory Models", () => {
  it("formats tool ID correctly as backendId/toolName", () => {
    expect(
      toolIDString({ backendId: "backend-fixture", toolName: "list_files" })
    ).toBe("backend-fixture/list_files");
  });

  it("derives authorized state for known valid tools", () => {
    const validView: ToolGovernanceView = {
      toolId: { backendId: "backend-fixture", toolName: "list_files" },
      known: true,
      risk: "read",
      registeredFingerprint: "53582d9d7229826e75097ac478787ca403c94ce74e4611571be1b03c706952cc",
      driftStatus: "none",
    };
    expect(deriveGovernanceState(validView)).toBe("authorized");
    expect(isGovernanceFailing("authorized")).toBe(false);
    expect(isDriftFailing(validView.driftStatus)).toBe(false);
  });

  it("derives unknown state for unregistered tools (fails closed)", () => {
    const unknownView: ToolGovernanceView = {
      toolId: { backendId: "backend-fixture", toolName: "unregistered_op" },
      known: false,
      driftStatus: "unknown_tool",
    };
    expect(deriveGovernanceState(unknownView)).toBe("unknown");
    expect(isGovernanceFailing("unknown")).toBe(true);
    expect(isDriftFailing(unknownView.driftStatus)).toBe(true);
  });

  it("derives drifted state when schema drift is detected (fails closed)", () => {
    const driftedView: ToolGovernanceView = {
      toolId: { backendId: "backend-fixture", toolName: "list_files" },
      known: true,
      risk: "read",
      registeredFingerprint: "53582d9d7229826e75097ac478787ca403c94ce74e4611571be1b03c706952cc",
      driftStatus: "detected",
    };
    expect(deriveGovernanceState(driftedView)).toBe("drifted");
    expect(isGovernanceFailing("drifted")).toBe(true);
    expect(isDriftFailing(driftedView.driftStatus)).toBe(true);
  });

  it("derives missing_risk when tool has empty/invalid risk (fails closed)", () => {
    const missingRiskView: ToolGovernanceView = {
      toolId: { backendId: "backend-fixture", toolName: "list_files" },
      known: true,
      driftStatus: "none",
    };
    expect(deriveGovernanceState(missingRiskView)).toBe("missing_risk");
    expect(isGovernanceFailing("missing_risk")).toBe(true);
  });

  it("proves same tool name on different backend has distinct identity", () => {
    const toolA = { backendId: "backend-1", toolName: "list_files" };
    const toolB = { backendId: "backend-2", toolName: "list_files" };
    expect(toolIDString(toolA)).not.toBe(toolIDString(toolB));
  });
});

describe("G2 Argument Declaration Models", () => {
  const decls = g2Fixtures.argumentDeclaration
    .validDeclarations as readonly ArgumentDeclarationView[];

  it("validates declaration sets properly", () => {
    const ok = validateDeclarationSet(decls);
    expect(ok.valid).toBe(true);

    const dup = validateDeclarationSet([
      { name: "path", type: "string", required: true },
      { name: "path", type: "string", required: false },
    ]);
    expect(dup.valid).toBe(false);
    expect(dup.error).toContain("duplicate");

    const empty = validateDeclarationSet([
      { name: "", type: "string", required: true },
    ]);
    expect(empty.valid).toBe(false);
  });

  it("resolves complete valid arguments", () => {
    const res = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.validArgsComplete
    );
    expect(res.ok).toBe(true);
    if (!res.ok) return;
    expect(res.resolved["path"]?.strVal).toBe("/var/log/app.log");
    expect(res.resolved["count"]?.intVal).toBe(10);
    expect(res.resolved["recursive"]?.boolVal).toBe(true);
  });

  it("omits missing optional arguments without error", () => {
    const res = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.validArgsMissingOptional
    );
    expect(res.ok).toBe(true);
    if (!res.ok) return;
    expect(res.resolved["path"]?.strVal).toBe("/var/log/app.log");
    expect(res.resolved["count"]).toBeUndefined();
    expect(res.resolved["recursive"]).toBeUndefined();
  });

  it("fails when required argument is missing", () => {
    const res = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.missingRequired
    );
    expect(res.ok).toBe(false);
    if (res.ok) return;
    expect(res.error.argName).toBe("path");
    expect(res.error.reason).toContain("required");
  });

  it("rejects explicit JSON null for required arguments", () => {
    const res = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.explicitNullRequired
    );
    expect(res.ok).toBe(false);
    if (res.ok) return;
    expect(res.error.argName).toBe("path");
    expect(res.error.reason).toContain("null");
  });

  it("rejects explicit JSON null for optional arguments (null is NOT absent)", () => {
    const res = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.explicitNullOptional
    );
    expect(res.ok).toBe(false);
    if (res.ok) return;
    expect(res.error.argName).toBe("count");
    expect(res.error.reason).toContain("null");
  });

  it("rejects type mismatches", () => {
    const resStr = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.typeMismatchString
    );
    expect(resStr.ok).toBe(false);
    if (resStr.ok) return;
    expect(resStr.error.argName).toBe("path");

    const resInt = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.typeMismatchInt
    );
    expect(resInt.ok).toBe(false);
    if (resInt.ok) return;
    expect(resInt.error.argName).toBe("count");

    const resBool = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.typeMismatchBool
    );
    expect(resBool.ok).toBe(false);
    if (resBool.ok) return;
    expect(resBool.error.argName).toBe("recursive");
  });

  it("ignores undeclared arguments so they never enter policy", () => {
    const res = resolveArguments(
      decls,
      g2Fixtures.argumentDeclaration.undeclaredArgs
    );
    expect(res.ok).toBe(true);
    if (!res.ok) return;
    expect(res.resolved["path"]?.strVal).toBe("/var/log/app.log");
    expect((res.resolved as Record<string, unknown>)["extraParam"]).toBeUndefined();
    expect((res.resolved as Record<string, unknown>)["secretToken"]).toBeUndefined();
  });
});
