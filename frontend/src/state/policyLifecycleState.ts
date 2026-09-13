import { GovernanceClient } from "../api/governanceClient.js";
import {
  ActivateResponse,
  PolicyRecord,
  RollbackResponse,
  ValidateResponse,
} from "../models/governance.js";

export interface PolicyLifecycleState {
  workspaceId: string;
  policies: PolicyRecord[];
  activeVersion?: string | undefined;
  validation?: ValidateResponse | undefined;
  activationStatus: "idle" | "activating" | "confirmed" | "error";
  lastError?: string | undefined;
}

export class PolicyLifecycleStore {
  private client: GovernanceClient;
  private state: PolicyLifecycleState;

  constructor(client: GovernanceClient, workspaceId: string) {
    this.client = client;
    this.state = {
      workspaceId,
      policies: [],
      activationStatus: "idle",
    };
  }

  getState(): PolicyLifecycleState {
    return { ...this.state, policies: [...this.state.policies] };
  }

  async loadPolicies(): Promise<void> {
    try {
      const policies = await this.client.listPolicies(this.state.workspaceId);
      const active = policies.find((p) => p.state === "active");
      this.state.policies = policies;
      this.state.activeVersion = active ? active.version : undefined;
      this.state.lastError = undefined;
    } catch (err: any) {
      this.state.lastError = err.message || "failed to load policies";
      throw err;
    }
  }

  async validateDraft(content: string): Promise<ValidateResponse> {
    try {
      const val = await this.client.validatePolicy(this.state.workspaceId, content);
      this.state.validation = val;
      return val;
    } catch (err: any) {
      const res: ValidateResponse = {
        valid: false,
        errors: [err.message || "validation error"],
      };
      this.state.validation = res;
      return res;
    }
  }

  async submitCandidate(content: string, description: string): Promise<PolicyRecord> {
    try {
      const rec = await this.client.createCandidate(this.state.workspaceId, content, description);
      await this.loadPolicies();
      return rec;
    } catch (err: any) {
      this.state.lastError = err.message || "failed to create candidate";
      throw err;
    }
  }

  async activate(version: string): Promise<ActivateResponse> {
    this.state.activationStatus = "activating";
    try {
      const resp = await this.client.activatePolicy(this.state.workspaceId, version);
      this.state.activeVersion = resp.active_version;
      this.state.activationStatus = "confirmed";
      await this.loadPolicies();
      return resp;
    } catch (err: any) {
      this.state.activationStatus = "error";
      this.state.lastError = err.message || "activation failed";
      throw err;
    }
  }

  async rollback(targetVersion: string): Promise<RollbackResponse> {
    try {
      const resp = await this.client.rollbackPolicy(this.state.workspaceId, targetVersion);
      this.state.activeVersion = resp.active_version;
      await this.loadPolicies();
      return resp;
    } catch (err: any) {
      this.state.lastError = err.message || "rollback failed";
      throw err;
    }
  }
}
