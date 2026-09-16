import { Icon } from "./Icon";

export function Topbar({ onLogout }: { onLogout: () => void }) {
  return (
    <header className="topbar">
      <div className="topbar-search">
        <Icon name="search" size={16} />
        <input placeholder="Search tools, policies, identities…" />
      </div>
      <div className="topbar-actions">
        <button className="icon-button" title="Notifications" aria-label="Notifications">
          <Icon name="bell" size={18} />
          <span className="notif-dot" />
        </button>
        <div className="user-chip">
          <div className="avatar">SA</div>
          <div className="user-meta">
            <div className="user-name">System Admin</div>
            <div className="user-role">Administrator</div>
          </div>
        </div>
        <button className="icon-button" title="Sign out" aria-label="Sign out" onClick={onLogout}>
          <Icon name="logout" size={18} />
        </button>
      </div>
    </header>
  );
}
