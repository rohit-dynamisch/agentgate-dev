package authz

import (
	"context"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/protobuf/types/known/structpb"
)

func setupTestAdapter(t *testing.T) *Adapter {
	t.Helper()

	mapper, err := identity.NewMapper(identity.MapperConfig{
		AgentIDClaim:    "sub",
		RolesClaim:      "roles",
		OnBehalfOfClaim: "obo",
	})
	if err != nil {
		t.Fatalf("NewMapper failed: %v", err)
	}

	readStatusSchema := []byte(`{"type":"object","properties":{"verbose":{"type":"boolean"}}}`)
	readStatusFP, err := toolregistry.FingerprintSchema(readStatusSchema)
	if err != nil {
		t.Fatalf("FingerprintSchema failed: %v", err)
	}

	reg, err := toolregistry.NewRegistry([]toolregistry.RegistryEntry{
		{
			ToolID: toolregistry.ToolID{
				BackendID: "mcp-probe",
				ToolName:  "read_status",
			},
			Risk:                  toolregistry.RiskRead,
			RegisteredFingerprint: readStatusFP,
		},
		{
			ToolID: toolregistry.ToolID{
				BackendID: "mcp-probe",
				ToolName:  "transfer_funds",
			},
			Risk:                  toolregistry.RiskDestructive,
			RegisteredFingerprint: readStatusFP,
		},
	})
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	readStatusDecls, err := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{
			Name:     "verbose",
			Type:     argdecl.ArgTypeBool,
			Required: false,
		},
	})
	if err != nil {
		t.Fatalf("NewDeclarationSet failed: %v", err)
	}

	transferDecls, err := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{
			Name:     "amount",
			Type:     argdecl.ArgTypeInt,
			Required: true,
		},
	})
	if err != nil {
		t.Fatalf("NewDeclarationSet failed: %v", err)
	}

	return NewAdapter(AdapterConfig{
		DefaultWorkspaceID: "ws-test",
		DefaultBackendID:   "mcp-probe",
		IdentityMapper:     mapper,
		ToolRegistry:       reg,
		ArgDeclarations: map[string]*argdecl.DeclarationSet{
			"read_status":    readStatusDecls,
			"transfer_funds": transferDecls,
		},
	})
}

func TestAdapter_ValidToolCall_FromJWT(t *testing.T) {
	adapter := setupTestAdapter(t)

	jwtClaims, err := structpb.NewStruct(map[string]any{
		"sub":   "agent-007",
		"roles": "reader auditor",
		"obo":   "user-corp",
	})
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method: "POST",
					Path:   "/",
					Headers: map[string]string{
						":method":                   "POST",
						":path":                     "/",
						"x-agentgate-workspace-id": "ws-custom",
						"x-execution-id":            "exec-12345",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status","arguments":{"verbose":true}}}`,
				},
			},
			MetadataContext: &corev3.Metadata{
				FilterMetadata: map[string]*structpb.Struct{
					"envoy.filters.http.jwt_authn": jwtClaims,
				},
			},
		},
	}

	decisionReq, aerr := adapter.Adapt(context.Background(), req)
	if aerr != nil {
		t.Fatalf("unexpected adapter error: %v", aerr)
	}

	if decisionReq.WorkspaceID != "ws-custom" {
		t.Errorf("expected workspace 'ws-custom', got %q", decisionReq.WorkspaceID)
	}
	if decisionReq.ExecutionID != "exec-12345" {
		t.Errorf("expected execution ID 'exec-12345', got %q", decisionReq.ExecutionID)
	}
	if decisionReq.Identity.AgentID != "agent-007" {
		t.Errorf("expected agent 'agent-007', got %q", decisionReq.Identity.AgentID)
	}
	if decisionReq.Identity.OnBehalfOf != "user-corp" {
		t.Errorf("expected on-behalf-of 'user-corp', got %q", decisionReq.Identity.OnBehalfOf)
	}
	if len(decisionReq.Identity.Roles) != 2 || decisionReq.Identity.Roles[0] != "reader" || decisionReq.Identity.Roles[1] != "auditor" {
		t.Errorf("unexpected roles: %v", decisionReq.Identity.Roles)
	}
	if decisionReq.Tool.Name != "read_status" || decisionReq.Tool.BackendID != "mcp-probe" {
		t.Errorf("unexpected tool: %+v", decisionReq.Tool)
	}
	if !decisionReq.Classification.Known || decisionReq.Classification.Risk != "read" {
		t.Errorf("unexpected classification: %+v", decisionReq.Classification)
	}
	if val, ok := decisionReq.Arguments["verbose"]; !ok || val.String() != "true" {
		t.Errorf("expected argument verbose=true, got %v", decisionReq.Arguments)
	}
}

func TestAdapter_ValidToolCall_FromHeaders(t *testing.T) {
	adapter := setupTestAdapter(t)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method: "POST",
					Path:   "/",
					Headers: map[string]string{
						"x-agent-id": "agent-hdr",
						"x-roles":    "admin",
					},
					RawBody: []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"transfer_funds","arguments":{"amount":100}}}`),
				},
			},
		},
	}

	decisionReq, aerr := adapter.Adapt(context.Background(), req)
	if aerr != nil {
		t.Fatalf("unexpected adapter error: %v", aerr)
	}

	if decisionReq.Identity.AgentID != "agent-hdr" {
		t.Errorf("expected agent 'agent-hdr', got %q", decisionReq.Identity.AgentID)
	}
	if len(decisionReq.Identity.Roles) != 1 || decisionReq.Identity.Roles[0] != "admin" {
		t.Errorf("unexpected roles: %v", decisionReq.Identity.Roles)
	}
	if decisionReq.Tool.Name != "transfer_funds" {
		t.Errorf("expected tool transfer_funds, got %q", decisionReq.Tool.Name)
	}
	if val, ok := decisionReq.Arguments["amount"]; !ok || val.String() != "100" {
		t.Errorf("expected argument amount=100, got %v", decisionReq.Arguments)
	}
}

func TestAdapter_MissingIdentity(t *testing.T) {
	adapter := setupTestAdapter(t)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status"}}`,
				},
			},
		},
	}

	_, aerr := adapter.Adapt(context.Background(), req)
	if aerr == nil {
		t.Fatal("expected adapter error for missing identity, got nil")
	}
	if aerr.ReasonCode != decision.ReasonInvalidIdentity {
		t.Errorf("expected ReasonInvalidIdentity, got %s", aerr.ReasonCode)
	}
}

func TestAdapter_AmbiguousIdentity(t *testing.T) {
	adapter := setupTestAdapter(t)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id":       "agent-same",
						"x-on-behalf-of":   "agent-same",
						"x-roles":          "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status"}}`,
				},
			},
		},
	}

	_, aerr := adapter.Adapt(context.Background(), req)
	if aerr == nil {
		t.Fatal("expected error for ambiguous identity (on_behalf_of == agent_id), got nil")
	}
	if aerr.ReasonCode != decision.ReasonInvalidIdentity {
		t.Errorf("expected ReasonInvalidIdentity, got %s", aerr.ReasonCode)
	}
}

func TestAdapter_NonToolCall(t *testing.T) {
	adapter := setupTestAdapter(t)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2026-07-28"}}`,
				},
			},
		},
	}

	_, aerr := adapter.Adapt(context.Background(), req)
	if aerr == nil {
		t.Fatal("expected error for non-tool call, got nil")
	}
	if aerr.ReasonCode != decision.ReasonMalformedRequest {
		t.Errorf("expected ReasonMalformedRequest, got %s", aerr.ReasonCode)
	}
}

func TestAdapter_MalformedJSON(t *testing.T) {
	adapter := setupTestAdapter(t)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params": BROKEN JSON`,
				},
			},
		},
	}

	_, aerr := adapter.Adapt(context.Background(), req)
	if aerr == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if aerr.ReasonCode != decision.ReasonMalformedRequest {
		t.Errorf("expected ReasonMalformedRequest, got %s", aerr.ReasonCode)
	}
}

func TestAdapter_UnknownTool(t *testing.T) {
	adapter := setupTestAdapter(t)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"delete_everything"}}`,
				},
			},
		},
	}

	_, aerr := adapter.Adapt(context.Background(), req)
	if aerr == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
	if aerr.ReasonCode != decision.ReasonUnknownTool {
		t.Errorf("expected ReasonUnknownTool, got %s", aerr.ReasonCode)
	}
}

func TestAdapter_InvalidArguments(t *testing.T) {
	adapter := setupTestAdapter(t)

	// transfer_funds requires amount as int64. Passing string "one_million" should fail.
	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "admin",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"transfer_funds","arguments":{"amount":"one_million"}}}`,
				},
			},
		},
	}

	_, aerr := adapter.Adapt(context.Background(), req)
	if aerr == nil {
		t.Fatal("expected error for invalid argument type, got nil")
	}
	if aerr.ReasonCode != decision.ReasonMalformedRequest {
		t.Errorf("expected ReasonMalformedRequest, got %s", aerr.ReasonCode)
	}
}

func TestAdapter_UndeclaredArgumentsIgnored(t *testing.T) {
	adapter := setupTestAdapter(t)

	// read_status only declares verbose. Passing extra undeclared argument should not reach decision.Arguments.
	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						"x-agent-id": "agent-001",
						"x-roles":    "reader",
					},
					Body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_status","arguments":{"verbose":true,"exploit_payload":"malicious"}}}`,
				},
			},
		},
	}

	decisionReq, aerr := adapter.Adapt(context.Background(), req)
	if aerr != nil {
		t.Fatalf("unexpected adapter error: %v", aerr)
	}

	if _, ok := decisionReq.Arguments["exploit_payload"]; ok {
		t.Errorf("security defect: undeclared argument 'exploit_payload' leaked to decision.Arguments: %v", decisionReq.Arguments)
	}
	if val, ok := decisionReq.Arguments["verbose"]; !ok || val.String() != "true" {
		t.Errorf("expected verbose=true, got %v", decisionReq.Arguments)
	}
}
