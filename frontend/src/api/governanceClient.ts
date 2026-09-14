import {
  ActivateResponse,
  DryRunCompareResponse,
  DryRunCompareResult,
  DryRunSample,
  GovernanceApiError,
  ListPoliciesResponse,
  PolicyRecord,
  PreviewResponse,
  PreviewSample,
  RollbackResponse,
  ValidateResponse,
} from "../models/governance.js";

export interface GovernanceClient {
  listPolicies(workspaceId: string): Promise<PolicyRecord[]>;
  createCandidate(workspaceId: string, content: string, description: string): Promise<PolicyRecord>;
  validatePolicy(workspaceId: string, content: string): Promise<ValidateResponse>;
  activatePolicy(workspaceId: string, version: string): Promise<ActivateResponse>;
  rollbackPolicy(workspaceId: string, targetVersion: string): Promise<RollbackResponse>;
  previewPolicy(workspaceId: string, version: string, sampleRequests: PreviewSample[]): Promise<PreviewResponse>;
  dryRunCompare(workspaceId: string, version: string, sampleRequests: (PreviewSample | DryRunSample)[]): Promise<DryRunCompareResponse>;
}

// Simple hash simulation for MockGovernanceClient
function mockHash(content: string): string {
  let hash = 0;
  for (let i = 0; i < content.length; i++) {
    const char = content.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash |= 0;
  }
  return "hash_" + Math.abs(hash).toString(16).padStart(8, "0");
}

export class MockGovernanceClient implements GovernanceClient {
  private data: Map<string, Map<string, PolicyRecord>> = new Map();

  private getWorkspaceMap(ws: string): Map<string, PolicyRecord> {
    if (!this.data.has(ws)) {
      this.data.set(ws, new Map());
    }
    return this.data.get(ws)!;
  }

  async listPolicies(workspaceId: string): Promise<PolicyRecord[]> {
    const ws = this.getWorkspaceMap(workspaceId);
    return Array.from(ws.values()).sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  }

  async validatePolicy(_workspaceId: string, content: string): Promise<ValidateResponse> {
    if (content.includes("error") || !content.includes(";")) {
      return {
        valid: false,
        errors: ["syntax error: invalid Cedar policy declaration"],
      };
    }
    return {
      valid: true,
      version: mockHash(content),
    };
  }

  async createCandidate(workspaceId: string, content: string, description: string): Promise<PolicyRecord> {
    const val = await this.validatePolicy(workspaceId, content);
    if (!val.valid || !val.version) {
      throw new GovernanceApiError(400, "INVALID_POLICY", "invalid Cedar syntax");
    }

    const ws = this.getWorkspaceMap(workspaceId);
    if (ws.has(val.version)) {
      return ws.get(val.version)!;
    }

    const rec: PolicyRecord = {
      workspace_id: workspaceId,
      version: val.version,
      content,
      state: "candidate",
      description,
      created_at: new Date().toISOString(),
    };

    ws.set(val.version, rec);
    return rec;
  }

  async activatePolicy(workspaceId: string, version: string): Promise<ActivateResponse> {
    const ws = this.getWorkspaceMap(workspaceId);
    const target = ws.get(version);
    if (!target) {
      throw new GovernanceApiError(404, "NOT_FOUND", "policy version not found");
    }

    let previousVersion: string | undefined;
    for (const [v, r] of ws.entries()) {
      if (r.state === "active") {
        previousVersion = v;
        r.state = "historical";
      }
    }

    const activatedAt = new Date().toISOString();
    target.state = "active";
    target.activated_at = activatedAt;

    return {
      workspace_id: workspaceId,
      active_version: version,
      previous_version: previousVersion,
      activated_at: activatedAt,
    };
  }

  async rollbackPolicy(workspaceId: string, targetVersion: string): Promise<RollbackResponse> {
    const ws = this.getWorkspaceMap(workspaceId);
    const target = ws.get(targetVersion);
    if (!target) {
      throw new GovernanceApiError(404, "NOT_FOUND", "target policy version not found");
    }

    let currentActive = "";
    for (const [v, r] of ws.entries()) {
      if (r.state === "active") {
        currentActive = v;
        r.state = "historical";
      }
    }

    const activatedAt = new Date().toISOString();
    target.state = "active";
    target.activated_at = activatedAt;

    return {
      workspace_id: workspaceId,
      active_version: targetVersion,
      rolled_back_from: currentActive,
      activated_at: activatedAt,
    };
  }

  async previewPolicy(_workspaceId: string, version: string, sampleRequests: PreviewSample[]): Promise<PreviewResponse> {
    return {
      version,
      results: sampleRequests.map(() => ({
        allowed: !version.includes("forbid"),
        matched: true,
        had_error: false,
      })),
    };
  }

  async dryRunCompare(_workspaceId: string, version: string, sampleRequests: (PreviewSample | DryRunSample)[]): Promise<DryRunCompareResponse> {
    const isForbid = version.includes("forbid");
    const activeDecision = "ALLOW";
    const candidateDecision = isForbid ? "DENY" : "ALLOW";
    return {
      candidate_version: version,
      results: sampleRequests.map(() => ({
        active_decision: activeDecision,
        active_reason: "policy_allow",
        active_policy_version: "v_active",
        candidate_decision: candidateDecision,
        candidate_reason: isForbid ? "policy_deny" : "policy_allow",
        candidate_policy_version: version,
        changed: activeDecision !== candidateDecision,
      })),
    };
  }
}

export interface FetchRequestOptions {
  method?: string;
  headers?: Record<string, string>;
  body?: string;
}

export interface FetchResponse {
  ok: boolean;
  status: number;
  json(): Promise<any>;
}

export type FetchFunction = (url: string, options?: FetchRequestOptions) => Promise<FetchResponse>;

export class HttpGovernanceClient implements GovernanceClient {
  private baseUrl: string;
  private adminToken: string;
  private fetchFn: FetchFunction;

  constructor(
    baseUrl: string,
    adminToken: string,
    fetchFn: FetchFunction = (globalThis as any).fetch
  ) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.adminToken = adminToken;
    this.fetchFn = fetchFn;
  }

  private async request<T>(path: string, options: FetchRequestOptions = {}): Promise<T> {
    const headers: Record<string, string> = {
      Authorization: `Bearer ${this.adminToken}`,
      "Content-Type": "application/json",
      ...(options.headers || {}),
    };

    const resp = await this.fetchFn(`${this.baseUrl}${path}`, {
      ...options,
      headers,
    });

    if (!resp.ok) {
      let code = "UNKNOWN_ERROR";
      let message = `Request failed with status ${resp.status}`;
      try {
        const errJson = await resp.json();
        if (errJson.error) {
          code = errJson.error.code || code;
          message = errJson.error.message || message;
        }
      } catch {
        // non-JSON response body
      }
      throw new GovernanceApiError(resp.status, code, message);
    }

    return (await resp.json()) as T;
  }

  async listPolicies(workspaceId: string): Promise<PolicyRecord[]> {
    const res = await this.request<ListPoliciesResponse>(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies`);
    return res.policies;
  }

  async createCandidate(workspaceId: string, content: string, description: string): Promise<PolicyRecord> {
    return this.request<PolicyRecord>(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies`, {
      method: "POST",
      body: JSON.stringify({ content, description }),
    });
  }

  async validatePolicy(workspaceId: string, content: string): Promise<ValidateResponse> {
    return this.request<ValidateResponse>(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies/validate`, {
      method: "POST",
      body: JSON.stringify({ content }),
    });
  }

  async activatePolicy(workspaceId: string, version: string): Promise<ActivateResponse> {
    return this.request<ActivateResponse>(
      `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies/${encodeURIComponent(version)}/activate`,
      { method: "POST" }
    );
  }

  async rollbackPolicy(workspaceId: string, targetVersion: string): Promise<RollbackResponse> {
    return this.request<RollbackResponse>(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies/rollback`, {
      method: "POST",
      body: JSON.stringify({ target_version: targetVersion }),
    });
  }

  async previewPolicy(workspaceId: string, version: string, sampleRequests: PreviewSample[]): Promise<PreviewResponse> {
    return this.request<PreviewResponse>(
      `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies/${encodeURIComponent(version)}/preview`,
      {
        method: "POST",
        body: JSON.stringify({ sample_requests: sampleRequests }),
      }
    );
  }

  async dryRunCompare(workspaceId: string, version: string, sampleRequests: (PreviewSample | DryRunSample)[]): Promise<DryRunCompareResponse> {
    const samples = sampleRequests.map((s, idx) => {
      let backendId = (s as DryRunSample).backend_id;
      let toolName = (s as DryRunSample).tool_name;
      if (!backendId && s.resource_id) {
        const parts = s.resource_id.split("/");
        if (parts.length > 1) {
          backendId = parts[0]!;
          toolName = parts.slice(1).join("/");
        } else {
          backendId = "default";
          toolName = parts[0]!;
        }
      }
      const sampleObj: Record<string, unknown> = {
        execution_id: (s as DryRunSample).execution_id || `dryrun-${idx + 1}`,
        principal_id: s.principal_id,
        principal_roles: s.principal_roles,
        backend_id: backendId || "default",
        tool_name: toolName || "tool",
        risk: (s as DryRunSample).risk || s.resource_risk || "read",
      };
      if ((s as DryRunSample).on_behalf_of) {
        sampleObj["on_behalf_of"] = (s as DryRunSample).on_behalf_of;
      }
      return sampleObj;
    });

    return this.request<DryRunCompareResponse>(
      `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/policies/${encodeURIComponent(version)}/dryrun`,
      {
        method: "POST",
        body: JSON.stringify({ sample_requests: samples }),
      }
    );
  }
}
