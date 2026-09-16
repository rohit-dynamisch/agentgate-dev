package main

import (
	"context"
	"strings"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestProbeRedactsAuthorizationAndRecordsBodyFacts(t *testing.T) {
	probe := NewProbeServer("allow")

	metadataVal, err := structpb.NewStruct(map[string]any{
		"sub": "user-123",
		"iss": "https://dev-idp.example.invalid",
	})
	if err != nil {
		t.Fatalf("failed to create metadata struct: %v", err)
	}

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method: "POST",
					Path:   "/mcp",
					Headers: map[string]string{
						"authorization": "Bearer secret-token-do-not-log",
						"content-type":  "application/json",
						"x-request-id":  "req-456",
					},
					Body:    `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read_status"}}`,
					RawBody: []byte(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read_status"}}`),
				},
			},
			MetadataContext: &corev3.Metadata{
				FilterMetadata: map[string]*structpb.Struct{
					"envoy.filters.http.jwt_authn": metadataVal,
				},
			},
		},
	}

	resp, err := probe.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("expected Check to succeed in allow mode, got: %v", err)
	}
	if resp.GetStatus().GetCode() != int32(0) {
		t.Fatalf("expected status code 0 (OK), got: %d", resp.GetStatus().GetCode())
	}
	if resp.GetOkResponse() == nil {
		t.Fatalf("expected OkResponse to be non-nil")
	}

	records := probe.GetRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	record := records[0]

	// Check that authorization header value is redacted/hashed, not plaintext
	if authVal, ok := record.Headers["authorization"]; ok {
		if strings.Contains(authVal, "secret-token-do-not-log") {
			t.Errorf("expected authorization header to be redacted/hashed, but contained plaintext token")
		}
		if !strings.HasPrefix(authVal, "sha256:") && authVal != "[REDACTED]" {
			t.Errorf("expected authorization header to be sha256:... or [REDACTED], got %s", authVal)
		}
	} else {
		t.Errorf("expected authorization header to be recorded (in sanitized form)")
	}

	// Verify recorded body facts
	if !record.BodyPresent {
		t.Errorf("expected BodyPresent to be true")
	}
	if record.BodyLength != int64(len(req.Attributes.Request.Http.Body)) {
		t.Errorf("expected BodyLength %d, got %d", len(req.Attributes.Request.Http.Body), record.BodyLength)
	}
	if !strings.Contains(string(record.RawBody), "read_status") {
		t.Errorf("expected RawBody to contain 'read_status', got %s", string(record.RawBody))
	}

	// Verify metadata capture
	if record.Metadata["envoy.filters.http.jwt_authn"] == nil {
		t.Errorf("expected jwt_authn metadata to be recorded")
	}
}

func TestProbeNeverConvertsDenyOrMalformedModeToAllow(t *testing.T) {
	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method: "POST",
					Path:   "/mcp",
				},
			},
		},
	}

	// Test mode: deny
	denyProbe := NewProbeServer("deny")
	denyResp, err := denyProbe.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Check error in deny mode: %v", err)
	}
	if denyResp.GetStatus().GetCode() == 0 {
		t.Errorf("deny mode must never return status code 0 (OK)")
	}
	if denyResp.GetDeniedResponse() == nil {
		t.Errorf("deny mode must populate DeniedResponse")
	}
	if denyResp.GetDeniedResponse().GetStatus().GetCode() != typev3.StatusCode_Forbidden {
		t.Errorf("expected HTTP 403 Forbidden in DeniedResponse")
	}

	// Test mode: malformed
	malformedProbe := NewProbeServer("malformed")
	malformedResp, err := malformedProbe.Check(context.Background(), req)
	// Malformed mode should either return an error or a non-OK status
	if err == nil {
		if malformedResp != nil && malformedResp.GetStatus().GetCode() == 0 {
			t.Errorf("malformed mode must never return OK allow response")
		}
	}

	// Test mode: unavailable
	unavailableProbe := NewProbeServer("unavailable")
	_, err = unavailableProbe.Check(context.Background(), req)
	if err == nil {
		t.Errorf("unavailable mode must return an RPC error")
	}
}
