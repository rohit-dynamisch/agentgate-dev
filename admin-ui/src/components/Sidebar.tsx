import { NavLink } from "react-router-dom";
import { Icon } from "./Icon";

const NAV = [
  { to: "/", label: "Dashboard", icon: "dashboard" as const, end: true },
  { to: "/policies", label: "Policies", icon: "policies" as const },
  { to: "/tools", label: "Tools & Resources", icon: "tools" as const },
  { to: "/identities", label: "Identities", icon: "identities" as const },
  { to: "/decisions", label: "Decision Tester", icon: "decision" as const },
  { to: "/audit", label: "Audit Logs", icon: "audit" as const },
  { to: "/settings", label: "Settings", icon: "settings" as const },
];

export function Sidebar() {
  return (
    <aside className="sidebar">
      <div className="sidebar-brand">
        <div className="sidebar-logo">
          <Icon name="shield" size={20} />
        </div>
        <div>
          <div className="sidebar-title">AgentGate</div>
          <div className="sidebar-subtitle">Secure AI Actions</div>
        </div>
      </div>
      <nav className="sidebar-nav">
        {NAV.map((item) => (
          <NavLink key={item.to} to={item.to} end={item.end ?? false} className={({ isActive }) => "sidebar-link" + (isActive ? " sidebar-link--active" : "")}>
            <Icon name={item.icon} size={18} />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
