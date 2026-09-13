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
