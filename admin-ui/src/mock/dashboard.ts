/**
 * Prototype-only dashboard aggregates. No aggregation endpoint exists on the
 * real backend yet (see FLOW_AND_ARCHITECTURE.md §3 row 1) — this generates
 * a deterministic, plausibly-scaled synthetic dataset (thousands of
 * decisions) to show what a populated dashboard looks like, distinct from
 * the small set of specific, readable rows in mock/auditLog.ts used for the
 * Audit Logs screen itself. Categories used (ALLOW/DENY/reason codes, tool
 * names) are drawn from the real frozen contract and mock/tools.ts, not
 * invented ones.
 */

import { mockTools } from "./tools";

// Deterministic PRNG (mulberry32) so the dashboard renders identically on
// every load instead of jittering between runs/screenshots.
function mulberry32(seed: number) {
  return function () {
    seed |= 0;
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export interface HourBucket {
  hour: string;
  allowed: number;
  denied: number;
  unknown: number;
}

export interface ToolUsage {
  name: string;
  count: number;
  classification: string;
}

export interface DashboardStats {
  totalRequests: number;
  allowed: number;
  denied: number;
  unknown: number;
  allowedPct: number;
  deniedPct: number;
  unknownPct: number;
  topTools: ToolUsage[];
  timeline: HourBucket[];
}

export function computeDashboardStats(): DashboardStats {
  const rnd = mulberry32(42);
  const timeline: HourBucket[] = [];
  let allowed = 0;
  let denied = 0;
  let unknown = 0;

  for (let h = 0; h < 24; h++) {
    const base = 600 + Math.floor(rnd() * 500);
    const unk = Math.floor(base * (0.02 + rnd() * 0.05));
    const den = Math.floor(base * (0.12 + rnd() * 0.1));
    const alw = base - unk - den;
    allowed += alw;
    denied += den;
    unknown += unk;
    timeline.push({ hour: `${String(h).padStart(2, "0")}:00`, allowed: alw, denied: den, unknown: unk });
  }

  const total = allowed + denied + unknown;

  const topTools: ToolUsage[] = mockTools
    .filter((t) => t.status === "active")
    .map((t) => ({ name: t.name, classification: t.classification, count: 300 + Math.floor(rnd() * 8000) }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 5);

  return {
    totalRequests: total,
    allowed,
    denied,
    unknown,
    allowedPct: Math.round((allowed / total) * 1000) / 10,
    deniedPct: Math.round((denied / total) * 1000) / 10,
    unknownPct: Math.round((unknown / total) * 1000) / 10,
    topTools,
    timeline,
  };
}
