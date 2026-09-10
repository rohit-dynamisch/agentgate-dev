// Package policy is AgentGate's narrow, application-owned boundary around
// Cedar (docs/PHASES/DAY-02-TASK-02.md). It is the only package in this
// module that imports github.com/cedar-policy/cedar-go — everything above
// it (internal/decision and, later, internal/authz) talks to Cedar only
// through the primitive, Cedar-free types exported here (AttrValue,
// EvalInput, EvalOutput), never through cedar-go's own types.
//
// Policy source is a raw byte slice for now — Day 4 moves the source of
// truth to versioned PostgreSQL rows; Cedar takes bytes regardless of
// where they came from, so only the loader changes, not this package's
// evaluation contract (docs/TECH_STACK.md §2.3,
// docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md Day 4).
package policy
