import { useState } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AppProvider } from "./context/AppContext";
import { AppShell } from "./components/AppShell";
import { LoginGate } from "./components/LoginGate";
import { DashboardPage } from "./pages/DashboardPage";
import { PoliciesListPage } from "./pages/PoliciesListPage";
import { PolicyWizardPage } from "./pages/PolicyWizardPage";
import { ToolsPage } from "./pages/ToolsPage";
import { ToolDetailPage } from "./pages/ToolDetailPage";
import { IdentitiesPage } from "./pages/IdentitiesPage";
import { AuditLogPage } from "./pages/AuditLogPage";
import { SettingsPage } from "./pages/SettingsPage";
import { DecisionTesterPanel } from "./components/DecisionTesterPanel";

export function App() {
  const [authed, setAuthed] = useState(false);

  if (!authed) {
    return <LoginGate onEnter={() => setAuthed(true)} />;
  }

  return (
    <AppProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell onLogout={() => setAuthed(false)} />}>
            <Route index element={<DashboardPage />} />
            <Route path="policies" element={<PoliciesListPage />} />
            <Route path="policies/new" element={<PolicyWizardPage />} />
            <Route path="tools" element={<ToolsPage />} />
            <Route path="tools/:id" element={<ToolDetailPage />} />
            <Route path="identities" element={<IdentitiesPage />} />
            <Route path="decisions" element={<DecisionTesterPanel />} />
            <Route path="audit" element={<AuditLogPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AppProvider>
  );
}
