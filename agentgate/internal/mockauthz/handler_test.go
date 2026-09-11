package mockauthz

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
)

// These are the G1 contract tests (AG-GO-G1-05): they exercise the actual
// wire boundary (JSON over HTTP) that Gateway/MCP and QA/Security
// integrate against, not just the underlying decision.Engine (which
// already has its own thorough tests in internal/decision). A change here
// is a change to the frozen contract and requires the G1 freeze-rule
// review (docs/PHASES/G1_WORKSTREAMS/00_G1_CHECKPOINT_REFERENCE.md).

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	engine, err := decision.NewEngine([]byte(fixturepolicy.CedarSource))
	if err != nil {
		t.Fatalf("decision.NewEngine() error = %v", err)
	}
	return NewHandler(engine)
}

func post(t *testing.T, h *Handler, body string) (*http.Response, resultWire) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/evaluate", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	resp := rec.Result()

	var out resultWire
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decoding response body: %v", err)
		}
	}
	return resp, out
}

func expectedVersion(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(fixturepolicy.CedarSource))
	return hex.EncodeToString(sum[:])
}

func TestHandler_ValidAllow(t *testing.T) {
	h := newTestHandler(t)
	resp, got := post(t, h, `{
		"execution_id":"e1","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["reader"]},
		"tool":{"backend_id":"b","name":"read_schedule"},
		"classification":{"known":true,"risk":"read"}
	}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got.Decision != "ALLOW" {
		t.Errorf("decision = %q, want ALLOW", got.Decision)
	}
	if got.Reason != "policy_allow" {
		t.Errorf("reason = %q, want policy_allow", got.Reason)
	}
	if got.PolicyVersion != expectedVersion(t) {
		t.Errorf("policy_version = %q, want %q", got.PolicyVersion, expectedVersion(t))
	}
	if got.ExecutionID != "e1" {
		t.Errorf("execution_id = %q, want e1", got.ExecutionID)
	}
}

func TestHandler_ValidDenyExplicitForbid(t *testing.T) {
	h := newTestHandler(t)
	resp, got := post(t, h, `{
		"execution_id":"e2","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["reader"]},
		"tool":{"backend_id":"b","name":"delete_all"},
		"classification":{"known":true,"risk":"destructive"}
	}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got.Decision != "DENY" {
		t.Errorf("decision = %q, want DENY", got.Decision)
	}
	if got.Reason != "policy_deny" {
		t.Errorf("reason = %q, want policy_deny", got.Reason)
	}
	if got.PolicyVersion != expectedVersion(t) {
		t.Errorf("policy_version = %q, want %q (Cedar was reached)", got.PolicyVersion, expectedVersion(t))
	}
}

func TestHandler_ValidDenyNoMatchingPolicy(t *testing.T) {
	h := newTestHandler(t)
	_, got := post(t, h, `{
		"execution_id":"e3","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["reader"]},
		"tool":{"backend_id":"b","name":"send_email"},
		"classification":{"known":true,"risk":"write"}
	}`)

	if got.Decision != "DENY" || got.Reason != "no_matching_policy" {
		t.Errorf("got decision=%q reason=%q, want DENY/no_matching_policy", got.Decision, got.Reason)
	}
}

func TestHandler_MissingIdentity(t *testing.T) {
	h := newTestHandler(t)
	resp, got := post(t, h, `{
		"execution_id":"e4","workspace_id":"w1",
		"identity":{"roles":["reader"]},
		"tool":{"backend_id":"b","name":"read_schedule"},
		"classification":{"known":true,"risk":"read"}
	}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (this reaches the decision core and denies, it is not a transport error)", resp.StatusCode)
	}
	if got.Decision != "DENY" || got.Reason != "invalid_identity" {
		t.Errorf("got decision=%q reason=%q, want DENY/invalid_identity", got.Decision, got.Reason)
	}
	if got.PolicyVersion != "" {
		t.Errorf("policy_version = %q, want empty (Cedar must never be reached)", got.PolicyVersion)
	}
	if got.ExecutionID != "e4" {
		t.Errorf("execution_id = %q, want e4 (preserved even on early denial)", got.ExecutionID)
	}
}

func TestHandler_UnknownTool(t *testing.T) {
	h := newTestHandler(t)
	_, got := post(t, h, `{
		"execution_id":"e5","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["reader"]},
		"tool":{"backend_id":"b","name":"mystery"},
		"classification":{"known":false}
	}`)

	if got.Decision != "DENY" || got.Reason != "unknown_tool" {
		t.Errorf("got decision=%q reason=%q, want DENY/unknown_tool", got.Decision, got.Reason)
	}
}

func TestHandler_MalformedTransportBody(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/evaluate", bytes.NewBufferString("not json{{{"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a body that never became a decision.Request", rec.Code)
	}
}

func TestHandler_UnknownWireFieldIsRejectedAtTransport(t *testing.T) {
	// DisallowUnknownFields guards the frozen contract: a caller sending a
	// field outside the documented shape gets an explicit transport
	// error, not silent ignoring, so contract drift is caught immediately
	// rather than producing a confusing decision-core denial.
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/evaluate", bytes.NewBufferString(`{
		"execution_id":"e1","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["reader"]},
		"tool":{"backend_id":"b","name":"read_schedule"},
		"classification":{"known":true,"risk":"read"},
		"unexpected_field": true
	}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for an undocumented field", rec.Code)
	}
}

func TestHandler_EvaluationError(t *testing.T) {
	h := newTestHandler(t)
	resp, got := post(t, h, `{
		"execution_id":"e6","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["broken"]},
		"tool":{"backend_id":"b","name":"whatever"},
		"classification":{"known":true,"risk":"write"}
	}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got.Decision != "DENY" {
		t.Errorf("decision = %q, want DENY (an evaluation error must never become ALLOW)", got.Decision)
	}
	if got.Reason != "evaluation_error" {
		t.Errorf("reason = %q, want evaluation_error", got.Reason)
	}
	if got.PolicyVersion != expectedVersion(t) {
		t.Errorf("policy_version = %q, want %q (Cedar was reached)", got.PolicyVersion, expectedVersion(t))
	}
}

func TestHandler_ArgumentDependentRule(t *testing.T) {
	h := newTestHandler(t)

	_, within := post(t, h, `{
		"execution_id":"e7","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["payer"]},
		"tool":{"backend_id":"b","name":"transfer"},
		"classification":{"known":true,"risk":"write"},
		"arguments":{"amount":{"type":"int","value":500}}
	}`)
	if within.Decision != "ALLOW" {
		t.Errorf("amount=500: decision = %q, want ALLOW", within.Decision)
	}

	_, over := post(t, h, `{
		"execution_id":"e8","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["payer"]},
		"tool":{"backend_id":"b","name":"transfer"},
		"classification":{"known":true,"risk":"write"},
		"arguments":{"amount":{"type":"int","value":5000}}
	}`)
	if over.Decision != "DENY" || over.Reason != "no_matching_policy" {
		t.Errorf("amount=5000: got decision=%q reason=%q, want DENY/no_matching_policy", over.Decision, over.Reason)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/evaluate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandler_UnknownPath(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/does-not-exist", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandler_DeterministicAcrossRequests(t *testing.T) {
	h := newTestHandler(t)
	body := `{
		"execution_id":"e1","workspace_id":"w1",
		"identity":{"agent_id":"a1","roles":["reader"]},
		"tool":{"backend_id":"b","name":"read_schedule"},
		"classification":{"known":true,"risk":"read"}
	}`

	_, first := post(t, h, body)
	_, second := post(t, h, body)

	if first != second {
		t.Errorf("handler not deterministic for identical input: %+v vs %+v", first, second)
	}
}
