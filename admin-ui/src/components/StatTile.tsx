import type { ReactNode } from "react";

export type StatTone = "good" | "critical" | "warning" | "neutral";

export function StatTile({ label, value, sub, tone = "neutral", icon }: { label: string; value: string; sub?: string; tone?: StatTone; icon?: ReactNode }) {
  return (
    <div className="stat-tile">
      <div className={`stat-tile-icon stat-tile-icon--${tone}`}>{icon}</div>
      <div>
        <div className="stat-tile-value">{value}</div>
        <div className="stat-tile-label">{label}</div>
        {sub && <div className={`stat-tile-sub stat-tile-sub--${tone}`}>{sub}</div>}
      </div>
    </div>
  );
}
