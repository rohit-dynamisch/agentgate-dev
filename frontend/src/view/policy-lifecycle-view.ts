import { PolicyRecord, PolicyState, ValidateResponse } from "../models/governance.js";

export type BadgeTone = "active" | "candidate" | "historical";

export interface PolicyBadgeView {
  tone: BadgeTone;
  label: string;
}

export function toPolicyBadgeView(state: PolicyState): PolicyBadgeView {
  switch (state) {
    case "active":
      return { tone: "active", label: "Active" };
    case "candidate":
      return { tone: "candidate", label: "Candidate" };
    case "historical":
      return { tone: "historical", label: "Historical" };
  }
}

export interface LifecycleSummaryView {
  workspaceId: string;
  activeVersion?: string | undefined;
  candidateCount: number;
  historicalCount: number;
  totalPolicies: number;
}

export function toLifecycleSummaryView(
  workspaceId: string,
  policies: PolicyRecord[]
): LifecycleSummaryView {
  const active = policies.find((p) => p.state === "active");
  const candidates = policies.filter((p) => p.state === "candidate");
  const historical = policies.filter((p) => p.state === "historical");

  return {
    workspaceId,
    activeVersion: active?.version,
    candidateCount: candidates.length,
    historicalCount: historical.length,
    totalPolicies: policies.length,
  };
}

export interface ValidationDisplayView {
  tone: "valid" | "invalid";
  headline: string;
  versionLabel?: string | undefined;
  errorMessages?: string[] | undefined;
}

export function toValidationDisplayView(result: ValidateResponse): ValidationDisplayView {
  if (result.valid) {
    return {
      tone: "valid",
      headline: "Policy Syntax Valid",
      versionLabel: result.version,
    };
  }
  return {
    tone: "invalid",
    headline: "Policy Syntax Invalid",
    errorMessages: result.errors || ["Unknown syntax error"],
  };
}
