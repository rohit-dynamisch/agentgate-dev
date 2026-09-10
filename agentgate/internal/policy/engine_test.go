package policy

import (
	"strings"
	"testing"
)

const smokePolicy = `
permit(
  principal in AgentGate::Role::"reader",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "read"
};
`

func TestLoadFromBytesRejectsMalformedPolicy(t *testing.T) {
	_, err := LoadFromBytes([]byte("this is not valid cedar policy text {{{"))
	if err == nil {
		t.Fatal("LoadFromBytes() error = nil, want error for malformed policy")
	}
}

func TestVersionIsContentAddressedAndDeterministic(t *testing.T) {
	e1, err := LoadFromBytes([]byte(smokePolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}
	e2, err := LoadFromBytes([]byte(smokePolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}
	if e1.Version() == "" {
		t.Fatal("Version() = empty, want non-empty hash")
	}
	if e1.Version() != e2.Version() {
		t.Errorf("Version() differs for identical bytes: %q vs %q", e1.Version(), e2.Version())
	}

	e3, err := LoadFromBytes([]byte(smokePolicy + "\n// trailing comment changes bytes\n"))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}
	if e3.Version() == e1.Version() {
		t.Error("Version() identical for different policy bytes, want different hash")
	}
}

func TestNilEngineVersionIsEmpty(t *testing.T) {
	var e *Engine
	if got := e.Version(); got != "" {
		t.Errorf("nil Engine.Version() = %q, want empty", got)
	}
}

func TestEvaluateExplicitAllow(t *testing.T) {
	e, err := LoadFromBytes([]byte(smokePolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	out := e.Evaluate(EvalInput{
		PrincipalID:    "agent-1",
		PrincipalRoles: []string{"reader"},
		ResourceID:     "backend/read_schedule",
		ResourceRisk:   "read",
	})

	if !out.Allowed {
		t.Errorf("Allowed = false, want true")
	}
	if !out.Matched {
		t.Errorf("Matched = false, want true (an explicit permit fired)")
	}
	if out.HadError {
		t.Errorf("HadError = true, want false")
	}
}

func TestEvaluateNoMatchingPolicyDenies(t *testing.T) {
	e, err := LoadFromBytes([]byte(smokePolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	out := e.Evaluate(EvalInput{
		PrincipalID:    "agent-1",
		PrincipalRoles: []string{"reader"},
		ResourceID:     "backend/delete_record",
		ResourceRisk:   "destructive",
	})

	if out.Allowed {
		t.Errorf("Allowed = true, want false")
	}
	if out.Matched {
		t.Errorf("Matched = true, want false (no policy should have matched)")
	}
	if out.HadError {
		t.Errorf("HadError = true, want false")
	}
}

func TestEvaluateMissingContextAttributeIsEvaluationError(t *testing.T) {
	// A policy that references a context attribute without a `has` guard
	// must surface as a Cedar diagnostic error when that attribute is
	// absent, not silently evaluate to false.
	const brokenPolicy = `
permit(
  principal in AgentGate::Role::"broken",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  context.amount > 10
};
`
	e, err := LoadFromBytes([]byte(brokenPolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	out := e.Evaluate(EvalInput{
		PrincipalID:    "agent-1",
		PrincipalRoles: []string{"broken"},
		ResourceID:     "backend/some_tool",
		ResourceRisk:   "write",
		Context:        nil, // "amount" deliberately absent
	})

	if !out.HadError {
		t.Fatalf("HadError = false, want true (context.amount was accessed without a `has` guard while absent)")
	}
	if out.Allowed {
		t.Errorf("Allowed = true, want false — an errored evaluation must never be trusted as Allow")
	}
}

func TestEvaluateArgumentDependentRule(t *testing.T) {
	const payerPolicy = `
permit(
  principal in AgentGate::Role::"payer",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "write" &&
  context has amount &&
  context.amount <= 1000
};
`
	e, err := LoadFromBytes([]byte(payerPolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	within := e.Evaluate(EvalInput{
		PrincipalID:    "agent-1",
		PrincipalRoles: []string{"payer"},
		ResourceID:     "backend/transfer",
		ResourceRisk:   "write",
		Context:        map[string]AttrValue{"amount": IntAttr(500)},
	})
	if !within.Allowed {
		t.Errorf("amount=500: Allowed = false, want true")
	}

	over := e.Evaluate(EvalInput{
		PrincipalID:    "agent-1",
		PrincipalRoles: []string{"payer"},
		ResourceID:     "backend/transfer",
		ResourceRisk:   "write",
		Context:        map[string]AttrValue{"amount": IntAttr(5000)},
	})
	if over.Allowed {
		t.Errorf("amount=5000: Allowed = true, want false")
	}
}

func TestEvaluateUndeclaredExtraContextKeyDoesNotInfluenceOutcome(t *testing.T) {
	const payerPolicy = `
permit(
  principal in AgentGate::Role::"payer",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "write" &&
  context has amount &&
  context.amount <= 1000
};
`
	e, err := LoadFromBytes([]byte(payerPolicy))
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	base := EvalInput{
		PrincipalID:    "agent-1",
		PrincipalRoles: []string{"payer"},
		ResourceID:     "backend/transfer",
		ResourceRisk:   "write",
		Context:        map[string]AttrValue{"amount": IntAttr(500)},
	}
	withExtra := base
	withExtra.Context = map[string]AttrValue{
		"amount": IntAttr(500),
		"note":   StringAttr("this key is not referenced by any policy"),
	}

	got1 := e.Evaluate(base)
	got2 := e.Evaluate(withExtra)

	if got1.Allowed != got2.Allowed || got1.Matched != got2.Matched || got1.HadError != got2.HadError {
		t.Errorf("outcome changed due to an undeclared/unreferenced context key: %+v vs %+v", got1, got2)
	}
	if !got1.Allowed {
		t.Fatal("baseline case should Allow")
	}
}

func TestLoadFromBytesErrorMentionsParse(t *testing.T) {
	_, err := LoadFromBytes([]byte("{{{ garbage"))
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("error = %v, want it to mention parse failure", err)
	}
}
