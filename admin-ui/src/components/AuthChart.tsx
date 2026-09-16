import { useState } from "react";
import type { HourBucket } from "../mock/dashboard";

/**
 * 24-hour stacked bar chart of authorization outcomes. Follows the dataviz
 * skill's status-palette job (ALLOW/DENY/UNKNOWN are states, not identities)
 * and mark specs: <=24px bars, 4px rounded top on the topmost segment only,
 * 2px surface-color gap between stacked segments, legend + icon pairing
 * (status hues are sub-3:1 on light surfaces by design, mitigated by the
 * label, never color alone), and a per-bar hover/focus tooltip.
 */

const COLORS = {
  allowed: "#0ca30c", // status: good
  denied: "#d03b3b", // status: critical
  unknown: "#fab219", // status: warning
};

const GAP = 2;
const MAX_BAR = 24;

function roundedTopRectPath(x: number, y: number, w: number, h: number, r: number): string {
  const radius = Math.min(r, w / 2, h);
  return `M${x},${y + h} L${x},${y + radius} Q${x},${y} ${x + radius},${y} L${x + w - radius},${y} Q${x + w},${y} ${x + w},${y + radius} L${x + w},${y + h} Z`;
}

export function AuthChart({ data }: { data: HourBucket[] }) {
  const [hover, setHover] = useState<number | null>(null);

  const width = 720;
  const height = 160;
  const baseline = height - 4;
  const maxTotal = Math.max(...data.map((d) => d.allowed + d.denied + d.unknown));
  const slot = width / data.length;
  const barW = Math.min(MAX_BAR, slot - 4);

  const active = hover !== null ? data[hover] : null;

  return (
    <div className="auth-chart">
      <div className="auth-chart-header">
        <div className="auth-chart-readout">
          {active ? (
            <span>
              <strong>{active.hour}</strong> — Allowed <strong>{active.allowed.toLocaleString()}</strong> · Denied{" "}
              <strong>{active.denied.toLocaleString()}</strong> · Unknown <strong>{active.unknown.toLocaleString()}</strong>
            </span>
          ) : (
            <span className="muted">Hover or focus a bar for hourly detail</span>
          )}
        </div>
        <ul className="chart-legend">
          <li>
            <span className="legend-swatch" style={{ background: COLORS.allowed }} /> Allowed
          </li>
          <li>
            <span className="legend-swatch" style={{ background: COLORS.denied }} /> Denied
          </li>
          <li>
            <span className="legend-swatch" style={{ background: COLORS.unknown }} /> Unknown tool
          </li>
        </ul>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="auth-chart-svg" role="img" aria-label="Authorization requests by hour, allowed vs denied vs unknown">
        <line x1={0} y1={baseline} x2={width} y2={baseline} stroke="#c3c2b7" strokeWidth={1} />
        {data.map((d, i) => {
          const total = d.allowed + d.denied + d.unknown;
          const scale = (v: number) => (v / maxTotal) * (height - 20);
          const x = i * slot + (slot - barW) / 2;

          const allowedH = scale(d.allowed);
          const deniedH = scale(d.denied) - GAP;
          const unknownH = scale(d.unknown) - GAP;

          let cursorY = baseline;
          const segments: { y: number; h: number; color: string; roundTop: boolean }[] = [];

          cursorY -= allowedH;
          const allowedY = cursorY;
          cursorY -= GAP;

          cursorY -= Math.max(deniedH, 0);
          const deniedY = cursorY;
          cursorY -= GAP;

          cursorY -= Math.max(unknownH, 0);
          const unknownY = cursorY;

          if (d.unknown > 0) {
            segments.push({ y: unknownY, h: Math.max(unknownH, 1), color: COLORS.unknown, roundTop: true });
            segments.push({ y: deniedY, h: Math.max(deniedH, 1), color: COLORS.denied, roundTop: false });
          } else {
            segments.push({ y: deniedY, h: Math.max(deniedH, 1), color: COLORS.denied, roundTop: true });
          }
          segments.push({ y: allowedY, h: allowedH, color: COLORS.allowed, roundTop: false });

          const label = `${d.hour}: ${total.toLocaleString()} requests, ${d.allowed.toLocaleString()} allowed, ${d.denied.toLocaleString()} denied, ${d.unknown.toLocaleString()} unknown`;

          return (
            <g
              key={d.hour}
              tabIndex={0}
              role="img"
              aria-label={label}
              onMouseEnter={() => setHover(i)}
              onMouseLeave={() => setHover(null)}
              onFocus={() => setHover(i)}
              onBlur={() => setHover(null)}
              opacity={hover === null || hover === i ? 1 : 0.55}
            >
              <rect x={x} y={0} width={barW} height={height} fill="transparent" />
              {segments.map((s, si) =>
                s.roundTop ? (
                  <path key={si} d={roundedTopRectPath(x, s.y, barW, s.h, 4)} fill={s.color} />
                ) : (
                  <rect key={si} x={x} y={s.y} width={barW} height={s.h} fill={s.color} />
                )
              )}
            </g>
          );
        })}
      </svg>
    </div>
  );
}
