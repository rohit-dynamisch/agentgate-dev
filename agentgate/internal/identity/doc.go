// Package identity will resolve verified request identity (agent,
// on-behalf-of subject, roles) from gateway-validated claims into the
// context AgentGate's policy engine evaluates against
// (docs/PROJECT_DEFINITION.md §2;
// docs/PHASES/PHASE-01-MINIMAL-E2E-ENFORCEMENT.md TASK-01-02).
//
// Claim names must come from a per-deployment mapping, not be hardcoded
// (docs/TECH_STACK.md §2.4) — not designed yet. The downstream
// credential/identity propagation mechanism is a separate, unresolved
// question (O-001 in docs/DECISIONS/OPEN_DECISIONS.md) and must not be
// guessed here.
//
// Not implemented yet.
package identity
