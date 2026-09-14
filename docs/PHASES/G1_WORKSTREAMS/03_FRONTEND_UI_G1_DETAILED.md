# WS-C — Frontend / Admin UI — G1 Detailed Execution Plan

**Owner:** Frontend/UI  
**Collaborators:** Go Backend, QA/Security  
**Execution:** Sequential tickets. G1 establishes the external model boundary; later G3 defines the complete policy lifecycle API.

## AG-UI-G1-01 — Inspect current frontend architecture

**Goal:** Identify the existing API client, authentication, state and component conventions.

**Steps:**
1. Inspect package/build configuration.
2. Locate API service/client code.
3. Locate auth handling.
4. Locate existing policy/tool/audit views.
5. Identify existing typed-model/error conventions.
6. Record the smallest implementation surface.

**DoD:** No duplicate API abstraction is created and no UI-only authorization path is introduced.

## AG-UI-G1-02 — Define the immediate external API models

**Goal:** Create typed models that represent only the backend information currently required by the UI.

**Required semantics where applicable:** decision state; denial/error state; policy version/hash; identity/tool information; correlation ID.

**Steps:**
1. Map model fields to the frozen external contract.
2. Match required/optional semantics.
3. Keep Cedar entities/attributes out of frontend models unless explicitly part of an external contract.
4. Add serialization fixtures.

**DoD:** Type-check/build passes; fixtures deserialize correctly; no model depends on private Go structures.

## AG-UI-G1-03 — Create deterministic UI fixtures

Create ALLOW, DENY, authorization-error and incomplete/stale-data fixtures needed for continued UI development.

**DoD:** Each state renders without a live backend; no real credential or secret appears in fixture data.

## AG-UI-G1-04 — Define failure/stale-state handling

**Goal:** Prevent the UI from implying an authorization or policy operation succeeded when the backend did not confirm it.

**Steps:** Define loading, successful, denied, API-error and stale/unavailable states. Ensure failed mutations do not transition to a success state.

**DoD:** A backend error cannot be displayed as successful authorization/policy state.

## AG-UI-G1-05 — Cross-check against Go contract

Compare field names, optionality, error semantics and provenance fields with the frozen Go contract.

**DoD:** No unresolved mismatch; future G3 lifecycle needs are recorded separately rather than invented in G1.

**Collaboration:** Backend owns backend contract; Frontend must not silently change it.
