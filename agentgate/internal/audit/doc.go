// Package audit will record every authorization decision — allowed or
// denied — with full request context and the exact policy version that
// produced it (docs/PROJECT_DEFINITION.md §6; docs/TECH_STACK.md §2.5).
//
// The production audit durability invariant (whether an ALLOW may be
// returned before its audit record is durably persisted) is unresolved
// (O-002 in docs/DECISIONS/OPEN_DECISIONS.md) and must not be guessed
// here.
//
// Not implemented yet.
package audit
