export type PolicyState = "candidate" | "active" | "historical";

export interface PolicyRecord {
  id?: number | undefined;
  workspace_id: string;
  version: string;
  content: string;
  state: PolicyState;
  description: string;
  created_at: string;
  activated_at?: string | undefined;
}

export interface ValidateResponse {
  valid: boolean;
  version?: string | undefined;
  errors?: string[] | undefined;
}

export interface ActivateResponse {
  workspace_id: string;
  active_version: string;
  previous_version?: string | undefined;
  activated_at: string;
}

export interface RollbackResponse {
  workspace_id: string;
  active_version: string;
  rolled_back_from: string;
  activated_at: string;
}

export interface ListPoliciesResponse {
  workspace_id: string;
  policies: PolicyRecord[];
}

export interface PreviewSample {
  principal_id: string;
  principal_roles: string[];
  resource_id: string;
  resource_risk: string;
}

export interface PreviewResult {
  allowed: boolean;
  matched: boolean;
  had_error: boolean;
}

export interface PreviewResponse {
  version: string;
  results: PreviewResult[];
}

export interface DryRunCompareResult {
  active_decision: string;
  active_reason: string;
  active_policy_version: string;
  candidate_decision: string;
  candidate_reason: string;
  candidate_policy_version: string;
  changed: boolean;
}

export interface DryRunCompareResponse {
  candidate_version: string;
  results: DryRunCompareResult[];
}

export interface DryRunSample {
  principal_id: string;
  principal_roles: string[];
  resource_id?: string | undefined;
  resource_risk?: string | undefined;
  backend_id?: string | undefined;
  tool_name?: string | undefined;
  risk?: string | undefined;
  execution_id?: string | undefined;
  on_behalf_of?: string | undefined;
}

export class GovernanceApiError extends Error {
  public readonly code: string;
  public readonly status: number;

  constructor(status: number, code: string, message: string) {
    super(`[${code}] ${message}`);
    this.name = "GovernanceApiError";
    this.status = status;
    this.code = code;
  }
}

