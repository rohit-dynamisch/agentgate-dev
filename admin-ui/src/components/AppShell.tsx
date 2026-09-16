import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

export function AppShell({ onLogout }: { onLogout: () => void }) {
  return (
    <div className="shell">
      <Sidebar />
      <div className="shell-main">
        <Topbar onLogout={onLogout} />
        <main className="page">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
