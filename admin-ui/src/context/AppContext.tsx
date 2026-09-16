import { createContext, useContext, useMemo, useState, type ReactNode } from "react";
import { createGovernanceClient, type GovernanceMode } from "../lib/client";
import type { GovernanceClient } from "../lib/contract";
import { mockTools, type MockTool, type ToolRisk } from "../mock/tools";

interface AppState {
  workspaceId: string;
  setWorkspaceId: (id: string) => void;
  govMode: GovernanceMode;
  setGovMode: (m: GovernanceMode) => void;
  adminToken: string;
  setAdminToken: (t: string) => void;
  client: GovernanceClient;
  tools: MockTool[];
  classifyTool: (id: string, risk: ToolRisk) => void;
}

const AppContext = createContext<AppState | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  const [workspaceId, setWorkspaceId] = useState("default-workspace");
  const [govMode, setGovMode] = useState<GovernanceMode>("mock");
  const [adminToken, setAdminToken] = useState("");
  const [tools, setTools] = useState<MockTool[]>(mockTools);

  const client = useMemo(() => createGovernanceClient(govMode, adminToken), [govMode, adminToken]);

  function classifyTool(id: string, risk: ToolRisk) {
    setTools((prev) => prev.map((t) => (t.id === id ? { ...t, classification: risk, status: "active" } : t)));
  }

  const value: AppState = {
    workspaceId,
    setWorkspaceId,
    govMode,
    setGovMode,
    adminToken,
    setAdminToken,
    client,
    tools,
    classifyTool,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useAppContext(): AppState {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppContext must be used within AppProvider");
  return ctx;
}
