package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/audit"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/grpc/codes"
)

type mockEvaluator struct {
	decision decision.Decision
	reason   decision.ReasonCode
	err      error
}

func (m *mockEvaluator) EvaluateWithActivePolicy(_ context.Context, _ string, req decision.Request) (decision.Result, error) {
	if m.err != nil {
		return decision.Result{}, m.err
	}
	return decision.Result{
		Decision:      m.decision,
		Reason:        m.reason,
		ExecutionID:   req.ExecutionID,
		PolicyVersion: "v1.0.0",
	}, nil
}

type failingAuditStore struct {
	audit.Store
}

func (f *failingAuditStore) AppendDecision(_ context.Context, _ audit.DecisionRecord) (*audit.StoredRecord, error) {
	return nil, errors.New("simulated disk write failure")
}
func (f *failingAuditStore) GetLatestRecord(_ context.Context, _ string) (*audit.StoredRecord, error) {
	return nil, nil
}

func setupTestServer(t *testing.T, dec decision.Decision, reason decision.ReasonCode) *Server {
	t.Helper()
	adapter := setupTestAdapter(t)
	mockEval := &mockEvaluator{decision: dec, reason: reason}
	memStore := audit.NewMemoryStore()
	auditedSvc := audit.NewAuditedDecisionService(mockEval, memStore, audit.NewRedactor("", nil))
	return NewServer(adapter, auditedSvc)
}

func TestAuthzServer_Allow(t *testing.T) {
	srv := setupTestServer(t, decision.Allow, decision.ReasonPolicyAllow)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method: "POST",
					Path:   "/",
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status","arguments":{"verbose":true}}}`,
				},
			},
		},
	}

	resp, err := srv.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}
	if resp.Status.Code != int32(codes.OK) {
		t.Errorf("expected Status.Code == OK (0), got %d", resp.Status.Code)
	}
	if resp.GetOkResponse() == nil {
		t.Fatal("expected OkResponse, got nil")
	}

	hasAllowHeader := false
	for _, h := range resp.GetOkResponse().Headers {
		if h.Header.Key == "x-agentgate-decision" && h.Header.Value == "allow" {
			hasAllowHeader = true
		}
	}
	if !hasAllowHeader {
		t.Errorf("expected x-agentgate-decision: allow header in OkResponse")
	}
}

func TestAuthzServer_Deny(t *testing.T) {
	srv := setupTestServer(t, decision.Deny, decision.ReasonPolicyDeny)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method: "POST",
					Path:   "/",
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status","arguments":{"verbose":true}}}`,
				},
			},
		},
	}

	resp, err := srv.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}
	if resp.Status.Code != int32(codes.PermissionDenied) {
		t.Errorf("expected Status.Code == PermissionDenied (7), got %d", resp.Status.Code)
	}
	deniedResp := resp.GetDeniedResponse()
	if deniedResp == nil {
		t.Fatal("expected DeniedResponse, got nil")
	}
	if deniedResp.Status.Code != typev3.StatusCode_Forbidden {
		t.Errorf("expected HTTP status Forbidden (403), got %v", deniedResp.Status.Code)
	}

	hasDenyHeader := false
	for _, h := range deniedResp.Headers {
		if h.Header.Key == "x-agentgate-decision" && h.Header.Value == "deny" {
			hasDenyHeader = true
		}
	}
	if !hasDenyHeader {
		t.Errorf("expected x-agentgate-decision: deny header in DeniedResponse")
	}
}

func TestAuthzServer_UnknownTool(t *testing.T) {
	srv := setupTestServer(t, decision.Allow, decision.ReasonPolicyAllow)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"unknown_tool"}}`,
				},
			},
		},
	}

	resp, err := srv.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}
	if resp.Status.Code != int32(codes.PermissionDenied) {
		t.Errorf("expected PermissionDenied for unknown tool, got %d", resp.Status.Code)
	}
	if resp.GetDeniedResponse() == nil {
		t.Fatal("expected DeniedResponse for unknown tool, got nil")
	}
}

func TestAuthzServer_MissingIdentity(t *testing.T) {
	srv := setupTestServer(t, decision.Allow, decision.ReasonPolicyAllow)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status"}}`,
				},
			},
		},
	}

	resp, err := srv.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}
	if resp.Status.Code != int32(codes.PermissionDenied) {
		t.Errorf("expected PermissionDenied for missing identity, got %d", resp.Status.Code)
	}
	if resp.GetDeniedResponse() == nil {
		t.Fatal("expected DeniedResponse for missing identity, got nil")
	}
}

func TestAuthzServer_AuditFailureFailsClosed_O002(t *testing.T) {
	adapter := setupTestAdapter(t)
	// Evaluator would allow
	mockEval := &mockEvaluator{decision: decision.Allow, reason: decision.ReasonPolicyAllow}
	// Audit store will fail on append
	failingStore := &failingAuditStore{Store: audit.NewMemoryStore()}
	auditedSvc := audit.NewAuditedDecisionService(mockEval, failingStore, audit.NewRedactor("", nil))
	srv := NewServer(adapter, auditedSvc)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status","arguments":{"verbose":true}}}`,
				},
			},
		},
	}

	resp, err := srv.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}
	if resp.Status.Code != int32(codes.PermissionDenied) {
		t.Errorf("O-002 invariant violated: expected PermissionDenied on audit failure, got %d", resp.Status.Code)
	}
	if resp.GetDeniedResponse() == nil {
		t.Fatal("expected DeniedResponse on audit failure, got nil")
	}
}

func TestAuthzServer_MalformedRequest(t *testing.T) {
	srv := setupTestServer(t, decision.Allow, decision.ReasonPolicyAllow)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{NOT VALID JSON`,
				},
			},
		},
	}

	resp, err := srv.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}
	if resp.Status.Code != int32(codes.PermissionDenied) {
		t.Errorf("expected PermissionDenied on malformed JSON, got %d", resp.Status.Code)
	}
	if resp.GetDeniedResponse() == nil {
		t.Fatal("expected DeniedResponse on malformed JSON, got nil")
	}
}
