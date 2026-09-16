import { useCallback, useMemo, useState } from "react";
import { PolicyLifecycleStore, type GovernanceClient, type PolicyLifecycleState } from "../lib/contract";

/**
 * PolicyLifecycleStore (frontend/src/state/policyLifecycleState.ts) is a
 * plain class that mutates its own internal state — it has no subscription
 * mechanism of its own, by design (it's framework-agnostic). This hook is
 * the thin React adapter: it re-reads getState() after every store call and
 * pushes it into React state so components re-render.
 */
export function usePolicyLifecycle(client: GovernanceClient, workspaceId: string) {
  const store = useMemo(() => new PolicyLifecycleStore(client, workspaceId), [client, workspaceId]);
  const [state, setState] = useState<PolicyLifecycleState>(() => store.getState());

  const run = useCallback(
    async <T,>(fn: (s: PolicyLifecycleStore) => Promise<T>): Promise<T | undefined> => {
      try {
        const result = await fn(store);
        setState(store.getState());
        return result;
      } catch {
        // Store already recorded lastError internally; surface it via state.
        setState(store.getState());
        return undefined;
      }
    },
    [store]
  );

  return { state, run };
}
