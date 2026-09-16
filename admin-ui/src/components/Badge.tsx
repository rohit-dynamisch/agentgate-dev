import type { ReactNode } from "react";

export type Tone = "active" | "candidate" | "historical" | "allow" | "deny" | "error" | "stale" | "pending" | "valid" | "invalid" | "changed" | "unchanged" | "idle" | "success";

const TONE_CLASS: Record<Tone, string> = {
  active: "badge badge--allow",
  allow: "badge badge--allow",
  valid: "badge badge--allow",
  success: "badge badge--allow",
  unchanged: "badge badge--allow",
  candidate: "badge badge--pending",
  pending: "badge badge--pending",
  historical: "badge badge--idle",
  idle: "badge badge--idle",
  deny: "badge badge--deny",
  invalid: "badge badge--deny",
  changed: "badge badge--deny",
  error: "badge badge--error",
  stale: "badge badge--stale",
};

export function Badge({ tone, children }: { tone: Tone; children: ReactNode }) {
  return <span className={TONE_CLASS[tone]}>{children}</span>;
}
