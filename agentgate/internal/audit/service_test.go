package audit_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type mockEvaluatorWithProvenance struct {
	result  decision.Result
	version string
	hash    string
}

func (m *mockEvaluatorWithProvenance) EvaluateWithActivePolicy(_ context.Context, _ string, req decision.Request) (decision.Result, error) {
	res := m.result
	res.ExecutionID = req.ExecutionID
	return res, nil
}

func (m *mockEvaluatorWithProvenance) GetActiveProvenance(_ string) (string, string, error) {
	return m.version, m.hash, nil
}

func TestAuditedDecisionService_ExactPolicyHashProvenance(t *testing.T) {
	policyBytes := []byte("permit(principal, action, resource);")
	h := sha256.Sum256(policyBytes)
	expectedHash := hex.EncodeToString(h[:])
	normalVersion := "v1.0.0"

	evaluator := &mockEvaluatorWithProvenance{
		result: decision.Result{
			Decision:      decision.Allow,
			Reason:        decision.ReasonPolicyAllow,
			PolicyVersion: expectedHash, // G1 decision.Engine returns SHA-256 in PolicyVersion
		},
		version: normalVersion,
		hash:    expectedHash,
	}

	aStore := audit.NewMemoryStore()
	redactor := audit.NewRedactor("salt", nil)
	auditSvc := audit.NewAuditedDecisionService(evaluator, aStore, redactor)

	ctx := context.Background()
	ws := "ws-provenance-test"

	req := decision.Request{
		ExecutionID:    "exec-prov-1",
		WorkspaceID:    ws,
		Identity:       decision.Identity{AgentID: "agent-1", Roles: []string{"reader"}},
		Tool:           decision.ToolRef{BackendID: "b1", Name: "t1"},
		Classification: decision.ToolClassification{Known: true, Risk: "read"},
	}

	res, err := auditSvc.Evaluate(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != decision.Allow {
		t.Fatalf("expected ALLOW, got %s", res.Decision)
	}

	record, err := aStore.GetLatestRecord(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Assert exact policy version preserved
	if record.PolicyVersion != normalVersion {
		t.Fatalf("expected policy_version %s, got %s", normalVersion, record.PolicyVersion)
	}

	// 2. Assert exact policy hash equals SHA-256 of evaluated policy bytes
	if record.PolicyHash != expectedHash {
		t.Fatalf("expected policy_hash == SHA-256 of evaluated bytes (%s), got %s", expectedHash, record.PolicyHash)
	}

	// 3. Assert policy_hash != policy_version for normal version identifier
	if record.PolicyHash == record.PolicyVersion {
		t.Fatalf("regression: policy_hash must not equal policy_version (%s == %s)", record.PolicyHash, record.PolicyVersion)
	}

	// 4. Pre-Cedar denial must leave both policy_version and policy_hash empty
	denyEvaluator := &mockEvaluatorWithProvenance{
		result: decision.Result{
			Decision:      decision.Deny,
			Reason:        decision.ReasonUnknownTool,
			PolicyVersion: "", // Cedar not reached
		},
		version: normalVersion,
		hash:    expectedHash,
	}
	auditSvc.SetEvaluator(denyEvaluator)

	reqDeny := decision.Request{
		ExecutionID:    "exec-prov-2",
		WorkspaceID:    ws,
		Identity:       decision.Identity{AgentID: "agent-1", Roles: []string{"reader"}},
		Tool:           decision.ToolRef{BackendID: "b1", Name: "unregistered"},
		Classification: decision.ToolClassification{Known: false},
	}
	_, err = auditSvc.Evaluate(ctx, reqDeny)
	if err != nil {
		t.Fatal(err)
	}
	recordDeny, err := aStore.GetLatestRecord(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if recordDeny.PolicyVersion != "" || recordDeny.PolicyHash != "" {
		t.Fatalf("pre-Cedar denial must not carry policy provenance: version=%q hash=%q", recordDeny.PolicyVersion, recordDeny.PolicyHash)
	}
}
