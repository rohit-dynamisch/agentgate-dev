// Package fixturepolicy provides the canonical G1 contract-freeze test/dev
// Cedar policy fixture, shared between internal/decision's own unit tests
// and the G1 AgentGate mock (cmd/g1-mock-authz, internal/mockauthz) — so
// both exercise identical, well-understood policy behavior rather than
// two independently-drifting copies.
//
// This is a test/dev fixture, not a production policy source. Production
// policy is versioned PostgreSQL rows (Day 4,
// docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md).
package fixturepolicy

// Role, tool-risk and action identifiers used by CedarSource, exported so
// dependent streams (Gateway/MCP, QA/Security, Frontend/UI) can construct
// correct fixture requests without reading Cedar syntax.
const (
	RoleReader = "reader"
	RoleAdmin  = "admin"
	RolePayer  = "payer"

	// RoleBroken deliberately triggers a genuine Cedar evaluation error
	// (not a simulated one) — see CedarSource.
	RoleBroken = "broken"

	RiskRead        = "read"
	RiskWrite       = "write"
	RiskDestructive = "destructive"

	// ArgAmount is the declared argument attribute name the "payer" and
	// "broken" rules below read from the request context.
	ArgAmount = "amount"
)

// CedarSource is the canonical G1 fixture policy set:
//
//   - reader: may invoke any tool classified risk=read.
//   - admin: may invoke any tool classified risk=read or risk=write.
//   - payer: may invoke a risk=write tool only when the declared "amount"
//     argument attribute is present and <= 1000 (the argument-dependent
//     rule fixture).
//   - any role: explicitly forbidden from invoking a risk=destructive
//     tool (the explicit-deny fixture, distinct from Cedar's structural
//     default-deny for "no matching policy").
//   - broken: deliberately references context.amount without a `has`
//     guard, so a request missing "amount" produces a genuine Cedar
//     evaluation error (the evaluation-error/"simulated failure" fixture)
//     rather than a hand-rolled fake one.
const CedarSource = `
permit(
  principal in AgentGate::Role::"reader",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "read"
};

permit(
  principal in AgentGate::Role::"admin",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "read" || resource.risk == "write"
};

permit(
  principal in AgentGate::Role::"payer",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "write" &&
  context has amount &&
  context.amount <= 1000
};

forbid(
  principal,
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "destructive"
};

permit(
  principal in AgentGate::Role::"broken",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  context.amount > 10
};
`
