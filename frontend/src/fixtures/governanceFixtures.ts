import { PolicyRecord } from "../models/governance.js";

export const g3Fixtures = {
  validPolicy1: `permit(principal, action, resource);`,
  validPolicy2: `forbid(principal, action, resource);`,
  invalidPolicy: `invalid cedar syntax error {`,

  sampleRecords: [
    {
      workspace_id: "default-workspace",
      version: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      content: `permit(principal, action, resource);`,
      state: "active",
      description: "Default permissive policy",
      created_at: "2026-09-13T10:00:00Z",
      activated_at: "2026-09-13T10:01:00Z",
    },
    {
      workspace_id: "default-workspace",
      version: "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92",
      content: `forbid(principal, action, resource);`,
      state: "candidate",
      description: "Candidate restrictive policy",
      created_at: "2026-09-13T11:00:00Z",
    },
  ] as PolicyRecord[],
};
