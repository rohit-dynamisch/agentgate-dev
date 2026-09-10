// Package decision is AgentGate's authorization decision core
// (docs/PHASES/DAY-02-TASK-02.md): a typed, deterministic, fail-closed
// contract that later gateway integration (internal/authz), identity
// resolution (internal/identity), tool governance, policy persistence and
// audit implementations are built on top of.
//
// It is independently testable — it depends on nothing but
// internal/policy's in-process Cedar boundary; no agentgateway,
// PostgreSQL, network I/O, or UI. Cedar itself never leaks into this
// package's exported surface: internal/policy is the only place cedar-go
// is imported (docs/SECURITY/PRODUCTION-INVARIANTS.md, "Cedar behind a
// narrow application-owned boundary").
//
// Scope note: this package defines the Identity shape the decision core
// needs, but does not resolve it from JWT claims — that is Day 3 work,
// owned by internal/identity, which will produce values in this shape (or
// convert into it). Likewise ToolClassification is a typed input here;
// the full fingerprinting/drift-detection system that produces it is Day
// 3 work (docs/DECISIONS/OPEN_DECISIONS.md O-005), and the per-tool
// argument-attribute declaration registry that governs what may populate
// Request.Arguments is also Day 3 work (O-006) — Day 2 establishes only
// the typed boundary, not either governance system.
package decision
