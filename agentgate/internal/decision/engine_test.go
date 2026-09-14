package decision

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
)

// fixturePolicy is the canonical G1 fixture policy
// (internal/fixturepolicy), also used by the G1 mock (internal/mockauthz)
// so both exercise identical, well-understood policy behavior. Test/dev
// policy fixtures are acceptable for Day 2 (docs/PHASES/DAY-02-TASK-02.md);
// PostgreSQL-backed policy is Day 4.
const fixturePolicy = fixturepolicy.CedarSource

func mustEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine([]byte(fixturePolicy))
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	return e
}

func baseRequest() Request {
	return Request{
		ExecutionID: "exec-1",
		WorkspaceID: "ws-1",
		Identity: Identity{
			AgentID: "agent-1",
			Roles:   []string{"reader"},
		},
		Tool: ToolRef{
			BackendID: "backend-a",
			Name:      "read_schedule",
		},
		Classification: ToolClassification{Known: true, Risk: "read"},
	}
}

func TestEvaluate_ExplicitAllow(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()

	got := e.Evaluate(req)

	if got.Decision != Allow {
		t.Errorf("Decision = %v, want %v", got.Decision, Allow)
	}
	if got.Reason != ReasonPolicyAllow {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonPolicyAllow)
	}
	if got.PolicyVersion == "" {
		t.Error("PolicyVersion is empty, want the evaluated version")
	}
	if got.ExecutionID != req.ExecutionID {
		t.Errorf("ExecutionID = %q, want %q", got.ExecutionID, req.ExecutionID)
	}
}

func TestEvaluate_ExplicitDeny(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Tool = ToolRef{BackendID: "backend-a", Name: "delete_everything"}
	req.Classification = ToolClassification{Known: true, Risk: "destructive"}

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonPolicyDeny {
		t.Errorf("Reason = %v, want %v (explicit forbid should have matched)", got.Reason, ReasonPolicyDeny)
	}
	if got.PolicyVersion == "" {
		t.Error("PolicyVersion is empty, want the evaluated version even on DENY")
	}
	if got.ExecutionID != req.ExecutionID {
		t.Errorf("ExecutionID = %q, want %q", got.ExecutionID, req.ExecutionID)
	}
}

func TestEvaluate_NoMatchingPolicy(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest() // reader role
	req.Tool = ToolRef{BackendID: "backend-a", Name: "send_email"}
	req.Classification = ToolClassification{Known: true, Risk: "write"} // reader has no write permit

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonNoMatchingPolicy {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonNoMatchingPolicy)
	}
	if got.PolicyVersion == "" {
		t.Error("PolicyVersion is empty, want the evaluated version even on DENY")
	}
}

func TestEvaluate_MissingIdentity(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Identity = Identity{Roles: []string{"reader"}} // AgentID empty

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonInvalidIdentity {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonInvalidIdentity)
	}
	if got.PolicyVersion != "" {
		t.Errorf("PolicyVersion = %q, want empty (Cedar must never be reached)", got.PolicyVersion)
	}
	if got.ExecutionID != req.ExecutionID {
		t.Errorf("ExecutionID = %q, want %q (must be preserved even on early denial)", got.ExecutionID, req.ExecutionID)
	}
}

func TestEvaluate_AmbiguousIdentity(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Identity = Identity{AgentID: "agent-1", OnBehalfOf: "agent-1", Roles: []string{"reader"}}

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonInvalidIdentity {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonInvalidIdentity)
	}
	if got.PolicyVersion != "" {
		t.Errorf("PolicyVersion = %q, want empty", got.PolicyVersion)
	}
}

func TestEvaluate_UnusableIdentityNoRoles(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Identity = Identity{AgentID: "agent-1"} // no roles

	got := e.Evaluate(req)

	if got.Decision != Deny || got.Reason != ReasonInvalidIdentity {
		t.Errorf("got Decision=%v Reason=%v, want Deny/%v", got.Decision, got.Reason, ReasonInvalidIdentity)
	}
}

func TestEvaluate_UnknownTool(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Classification = ToolClassification{Known: false}

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonUnknownTool {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonUnknownTool)
	}
	if got.PolicyVersion != "" {
		t.Errorf("PolicyVersion = %q, want empty (Cedar must never be reached)", got.PolicyVersion)
	}
}

func TestEvaluate_KnownToolWithoutRiskIsMalformed(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Classification = ToolClassification{Known: true, Risk: ""}

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonMalformedRequest {
		t.Errorf("Reason = %v, want %v (risk is required when known is true)", got.Reason, ReasonMalformedRequest)
	}
	if got.PolicyVersion != "" {
		t.Errorf("PolicyVersion = %q, want empty (Cedar must never be reached)", got.PolicyVersion)
	}
}

func TestEvaluate_KnownToolWithBlankRiskIsMalformed(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Classification = ToolClassification{Known: true, Risk: "   "}

	got := e.Evaluate(req)

	if got.Decision != Deny || got.Reason != ReasonMalformedRequest {
		t.Errorf("got Decision=%v Reason=%v, want Deny/%v", got.Decision, got.Reason, ReasonMalformedRequest)
	}
}

func TestEvaluate_MalformedRequest(t *testing.T) {
	tests := map[string]func(*Request){
		"missing execution id": func(r *Request) { r.ExecutionID = "" },
		"missing workspace id": func(r *Request) { r.WorkspaceID = "" },
		"missing backend id":   func(r *Request) { r.Tool.BackendID = "" },
		"missing tool name":    func(r *Request) { r.Tool.Name = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := mustEngine(t)
			req := baseRequest()
			mutate(&req)

			got := e.Evaluate(req)

			if got.Decision != Deny {
				t.Errorf("Decision = %v, want %v", got.Decision, Deny)
			}
			if got.Reason != ReasonMalformedRequest {
				t.Errorf("Reason = %v, want %v", got.Reason, ReasonMalformedRequest)
			}
			if got.PolicyVersion != "" {
				t.Errorf("PolicyVersion = %q, want empty", got.PolicyVersion)
			}
		})
	}
}

func TestEvaluate_ZeroValueAttributeIsMalformed(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Arguments = map[string]AttributeValue{"amount": {}} // zero-value, never constructed via a valid constructor

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonMalformedRequest {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonMalformedRequest)
	}
}

func TestEvaluate_CedarEvaluationError(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()
	req.Identity = Identity{AgentID: "agent-1", Roles: []string{"broken"}}
	req.Tool = ToolRef{BackendID: "backend-a", Name: "some_tool"}
	req.Classification = ToolClassification{Known: true, Risk: "write"}
	req.Arguments = nil // "amount" deliberately absent; fixturePolicy's "broken" rule references it unguarded

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v (an evaluation error must never become ALLOW)", got.Decision, Deny)
	}
	if got.Reason != ReasonEvaluationError {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonEvaluationError)
	}
	if got.PolicyVersion == "" {
		t.Error("PolicyVersion is empty, want the evaluated version (Cedar was reached)")
	}
}

func TestEvaluate_NoPolicyLoaded(t *testing.T) {
	e := NewEngineWithoutPolicy()
	req := baseRequest()

	got := e.Evaluate(req)

	if got.Decision != Deny {
		t.Errorf("Decision = %v, want %v", got.Decision, Deny)
	}
	if got.Reason != ReasonNoPolicyLoaded {
		t.Errorf("Reason = %v, want %v", got.Reason, ReasonNoPolicyLoaded)
	}
	if got.PolicyVersion != "" {
		t.Errorf("PolicyVersion = %q, want empty", got.PolicyVersion)
	}
	if got.ExecutionID != req.ExecutionID {
		t.Errorf("ExecutionID = %q, want %q", got.ExecutionID, req.ExecutionID)
	}
}

func TestNewEngineRejectsMalformedPolicy(t *testing.T) {
	_, err := NewEngine([]byte("not valid cedar {{{"))
	if err == nil {
		t.Fatal("NewEngine() error = nil, want error for malformed policy")
	}
}

func TestEvaluate_ArgumentDependentRule(t *testing.T) {
	e := mustEngine(t)

	within := baseRequest()
	within.Identity = Identity{AgentID: "agent-1", Roles: []string{"payer"}}
	within.Tool = ToolRef{BackendID: "backend-a", Name: "transfer"}
	within.Classification = ToolClassification{Known: true, Risk: "write"}
	within.Arguments = map[string]AttributeValue{"amount": IntAttr(500)}

	gotWithin := e.Evaluate(within)
	if gotWithin.Decision != Allow {
		t.Errorf("amount=500: Decision = %v, want %v", gotWithin.Decision, Allow)
	}

	over := within
	over.Arguments = map[string]AttributeValue{"amount": IntAttr(5000)}

	gotOver := e.Evaluate(over)
	if gotOver.Decision != Deny {
		t.Errorf("amount=5000: Decision = %v, want %v", gotOver.Decision, Deny)
	}
	if gotOver.Reason != ReasonNoMatchingPolicy {
		t.Errorf("amount=5000: Reason = %v, want %v", gotOver.Reason, ReasonNoMatchingPolicy)
	}
}

func TestEvaluate_UndeclaredArgumentCannotInfluenceOutcome(t *testing.T) {
	e := mustEngine(t)

	req := baseRequest()
	req.Identity = Identity{AgentID: "agent-1", Roles: []string{"payer"}}
	req.Tool = ToolRef{BackendID: "backend-a", Name: "transfer"}
	req.Classification = ToolClassification{Known: true, Risk: "write"}
	req.Arguments = map[string]AttributeValue{"amount": IntAttr(500)}

	withExtra := req
	withExtra.Arguments = map[string]AttributeValue{
		"amount": IntAttr(500),
		"note":   StringAttr("no policy in fixturePolicy references this key"),
	}

	got := e.Evaluate(req)
	gotExtra := e.Evaluate(withExtra)

	if got.Decision != Allow {
		t.Fatalf("baseline Decision = %v, want %v", got.Decision, Allow)
	}
	if got.Decision != gotExtra.Decision || got.Reason != gotExtra.Reason {
		t.Errorf("an unreferenced/undeclared argument changed the outcome: %+v vs %+v", got, gotExtra)
	}
}

func TestEvaluate_PolicyVersionExactAndDeterministic(t *testing.T) {
	e := mustEngine(t)
	req := baseRequest()

	first := e.Evaluate(req)
	second := e.Evaluate(req)

	if first != second {
		t.Errorf("Evaluate() not deterministic for identical input: %+v vs %+v", first, second)
	}

	wantVersion := sha256Hex([]byte(fixturePolicy))
	if first.PolicyVersion != wantVersion {
		t.Errorf("PolicyVersion = %q, want the exact content hash of the evaluated policy %q", first.PolicyVersion, wantVersion)
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestEvaluate_DeterministicAcrossIndependentEngineInstances(t *testing.T) {
	e1, err := NewEngine([]byte(fixturePolicy))
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	e2, err := NewEngine([]byte(fixturePolicy))
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	req := baseRequest()
	got1 := e1.Evaluate(req)
	got2 := e2.Evaluate(req)

	if got1 != got2 {
		t.Errorf("two engines loaded from identical policy bytes produced different results: %+v vs %+v", got1, got2)
	}
	if got1.PolicyVersion != got2.PolicyVersion {
		t.Errorf("PolicyVersion differs across independently loaded engines with identical bytes: %q vs %q", got1.PolicyVersion, got2.PolicyVersion)
	}
}
