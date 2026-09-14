package audit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/audit"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
)

type mockEvaluator struct {
	result decision.Result
}

func (m *mockEvaluator) EvaluateWithActivePolicy(_ context.Context, _ string, req decision.Request) (decision.Result, error) {
	res := m.result
	res.ExecutionID = req.ExecutionID
	return res, nil
}

func TestAuditedDecisionService_Allow(t *testing.T) {
	evaluator := &mockEvaluator{
		result: decision.Result{
			Decision:      decision.Allow,
			Reason:        decision.ReasonPolicyAllow,
			PolicyVersion: "ver-cand-123",
		},
	}
	aStore := audit.NewMemoryStore()
	redactor := audit.NewRedactor("salt", nil)
	auditSvc := audit.NewAuditedDecisionService(evaluator, aStore, redactor)

	ctx := context.Background()
	ws := "ws-audited-svc"

	req := decision.Request{
		ExecutionID:    "exec-audit-1",
		WorkspaceID:    ws,
		Identity:       decision.Identity{AgentID: "agent-1", Roles: []string{"reader"}},
		Tool:           decision.ToolRef{BackendID: "b1", Name: "t1"},
		Classification: decision.ToolClassification{Known: true, Risk: "read"},
		Arguments:      map[string]decision.AttributeValue{"query": decision.StringAttr("SELECT 1")},
	}

	res, err := auditSvc.Evaluate(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != decision.Allow {
		t.Fatalf("expected ALLOW, got %s (reason: %s, message: %s)", res.Decision, res.Reason, res.Message)
	}

	latest, err := aStore.GetLatestRecord(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if latest.ExecutionID != "exec-audit-1" || latest.Decision != "ALLOW" {
		t.Fatalf("audit mismatch: %+v", latest)
	}
	if latest.PolicyVersion != "ver-cand-123" {
		t.Fatalf("expected policy version ver-cand-123, got %s", latest.PolicyVersion)
	}
	if latest.RedactedArguments["query"] != "SELECT 1" {
		t.Fatalf("expected query SELECT 1, got %v", latest.RedactedArguments)
	}
}

func TestAuditedDecisionService_Deny(t *testing.T) {
	evaluator := &mockEvaluator{
		result: decision.Result{
			Decision: decision.Deny,
			Reason:   decision.ReasonUnknownTool,
		},
	}
	aStore := audit.NewMemoryStore()
	redactor := audit.NewRedactor("salt", nil)
	auditSvc := audit.NewAuditedDecisionService(evaluator, aStore, redactor)

	ctx := context.Background()
	ws := "ws-deny-svc"

	req := decision.Request{
		ExecutionID:    "exec-2",
		WorkspaceID:    ws,
		Identity:       decision.Identity{AgentID: "agent-1", Roles: []string{"reader"}},
		Tool:           decision.ToolRef{BackendID: "b1", Name: "unlisted-tool"},
		Classification: decision.ToolClassification{Known: false},
	}

	res, err := auditSvc.Evaluate(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != decision.Deny {
		t.Fatalf("expected DENY, got %s", res.Decision)
	}

	latest, err := aStore.GetLatestRecord(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Decision != "DENY" || latest.Reason != string(decision.ReasonUnknownTool) {
		t.Fatalf("expected durable DENY record, got %+v", latest)
	}
}

func TestAuditedDecisionService_AuditOutageFailsClosed(t *testing.T) {
	evaluator := &mockEvaluator{
		result: decision.Result{
			Decision:      decision.Allow,
			Reason:        decision.ReasonPolicyAllow,
			PolicyVersion: "ver-123",
		},
	}
	aStore := audit.NewMemoryStore()
	aStore.SetFault(errors.New("audit store unavailable"))

	redactor := audit.NewRedactor("salt", nil)
	auditSvc := audit.NewAuditedDecisionService(evaluator, aStore, redactor)

	ctx := context.Background()
	ws := "ws-outage-test"

	req := decision.Request{
		ExecutionID:    "exec-outage",
		WorkspaceID:    ws,
		Identity:       decision.Identity{AgentID: "agent-1", Roles: []string{"reader"}},
		Tool:           decision.ToolRef{BackendID: "b1", Name: "t1"},
		Classification: decision.ToolClassification{Known: true, Risk: "read"},
	}

	res, err := auditSvc.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Decision != decision.Deny {
		t.Fatalf("invariant O-002: expected DENY, got %s", res.Decision)
	}
	if res.Reason != decision.ReasonEvaluationError {
		t.Fatalf("expected ReasonEvaluationError, got %s", res.Reason)
	}
}
