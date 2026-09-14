package g4integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/auditevents"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
	"github.com/Dynamisch-LLC/agentgate/internal/govapi"
	"github.com/Dynamisch-LLC/agentgate/internal/governanceintegration"
	"github.com/Dynamisch-LLC/agentgate/internal/policymanager"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
)

const testAdminToken = "test-g4-qa-token"

type testEnv struct {
	server         *httptest.Server
	mgr            *policymanager.Manager
	store          policystore.Store
	govIntegration *governanceintegration.GovernanceDecisionService
	events         *auditevents.RecordingListener
}

func setupQAEnv() *testEnv {
	store := policystore.NewMemoryStore()
	rec := &auditevents.RecordingListener{}
	mgr := policymanager.NewWithListener(store, rec)
	govIntegration := governanceintegration.NewGovernanceDecisionService(mgr)
	handler := govapi.NewHandler(mgr, testAdminToken, govIntegration)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	return &testEnv{
		server:         ts,
		mgr:            mgr,
		store:          store,
		govIntegration: govIntegration,
		events:         rec,
	}
}

func (e *testEnv) close() {
	e.server.Close()
	e.store.Close()
}

func (e *testEnv) authedRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, e.server.URL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func makeReaderRequest(ws string) decision.Request {
	return decision.Request{
		ExecutionID: "exec-g4-test",
		WorkspaceID: ws,
		Identity: decision.Identity{
			AgentID: "agent-qa-1",
			Roles:   []string{fixturepolicy.RoleReader},
		},
		Tool: decision.ToolRef{
			BackendID: "backend-qa",
			Name:      "read-tool",
		},
		Classification: decision.ToolClassification{
			Known: true,
			Risk:  fixturepolicy.RiskRead,
		},
	}
}

// =============================================================================
// G4-QA-01: Full Lifecycle Decision Propagation Proof
// Candidate -> Validate -> Dry-Run -> Activate -> Observe Changed -> Rollback -> Observe Restored
// =============================================================================

func TestFullLifecycleDecisionPropagation(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()
	ws := "ws-lifecycle-proof"

	// 1. Initial state: no policy loaded -> Deny
	initRes, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	if initRes.Decision != decision.Deny || initRes.Reason != decision.ReasonNoPolicyLoaded {
		t.Fatalf("expected initial DENY/no_policy_loaded, got %s/%s", initRes.Decision, initRes.Reason)
	}

	// 2. Create Candidate A (reader can read)
	createRespA, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies", govapi.CreateCandidateRequest{
		Content:     fixturepolicy.CedarSource,
		Description: "policy A - reader permit",
	})
	if err != nil || createRespA.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create candidate A: status=%d err=%v", createRespA.StatusCode, err)
	}
	var recA policystore.PolicyRecord
	json.NewDecoder(createRespA.Body).Decode(&recA)
	createRespA.Body.Close()

	// 3. Activate Policy A
	actRespA, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/"+recA.Version+"/activate", nil)
	if err != nil || actRespA.StatusCode != http.StatusOK {
		t.Fatalf("failed to activate candidate A: status=%d err=%v", actRespA.StatusCode, err)
	}
	actRespA.Body.Close()

	// 4. Observe live evaluation with Policy A active -> ALLOW with recA provenance
	resA, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil {
		t.Fatalf("eval A failed: %v", err)
	}
	if resA.Decision != decision.Allow {
		t.Fatalf("expected ALLOW under policy A, got %s (reason: %s)", resA.Decision, resA.Reason)
	}
	if resA.PolicyVersion != recA.Version {
		t.Fatalf("provenance mismatch: expected %s, got %s", recA.Version, resA.PolicyVersion)
	}

	// 5. Create Candidate B (forbid all)
	restrictivePolicy := "forbid(principal, action, resource);"
	createRespB, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies", govapi.CreateCandidateRequest{
		Content:     restrictivePolicy,
		Description: "policy B - forbid all",
	})
	if err != nil || createRespB.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create candidate B: status=%d err=%v", createRespB.StatusCode, err)
	}
	var recB policystore.PolicyRecord
	json.NewDecoder(createRespB.Body).Decode(&recB)
	createRespB.Body.Close()

	// 6. Validate Policy B syntax
	valResp, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/validate", govapi.ValidateRequest{
		Content: restrictivePolicy,
	})
	if err != nil || valResp.StatusCode != http.StatusOK {
		t.Fatalf("validate failed: status=%d err=%v", valResp.StatusCode, err)
	}
	var valData govapi.ValidateResponse
	json.NewDecoder(valResp.Body).Decode(&valData)
	valResp.Body.Close()
	if !valData.Valid {
		t.Fatalf("expected candidate B to be valid, got invalid: %v", valData.Errors)
	}

	// 7. Dry-Run compare Candidate B against active Policy A
	dryResp, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/"+recB.Version+"/dryrun", govapi.DryRunCompareRequest{
		SampleRequests: []govapi.DryRunSampleRequest{
			{
				ExecutionID:    "exec-dry-proof",
				PrincipalID:    "agent-qa-1",
				PrincipalRoles: []string{fixturepolicy.RoleReader},
				BackendID:      "backend-qa",
				ToolName:       "read-tool",
				Risk:           fixturepolicy.RiskRead,
			},
		},
	})
	if err != nil || dryResp.StatusCode != http.StatusOK {
		t.Fatalf("dry-run compare failed: status=%d err=%v", dryResp.StatusCode, err)
	}
	var dryData govapi.DryRunCompareResponse
	json.NewDecoder(dryResp.Body).Decode(&dryData)
	dryResp.Body.Close()

	if dryData.CandidateVersion != recB.Version {
		t.Fatalf("dryrun candidate version mismatch: expected %s, got %s", recB.Version, dryData.CandidateVersion)
	}
	if len(dryData.Results) != 1 {
		t.Fatalf("expected 1 dryrun comparison result, got %d", len(dryData.Results))
	}
	if dryData.Results[0].ActiveDecision != "ALLOW" || dryData.Results[0].CandidateDecision != "DENY" {
		t.Fatalf("expected active=ALLOW, candidate=DENY; got active=%s, candidate=%s",
			dryData.Results[0].ActiveDecision, dryData.Results[0].CandidateDecision)
	}
	if !dryData.Results[0].Changed {
		t.Fatal("expected dryrun changed flag to be true")
	}

	// 8. Confirm active evaluation is still ALLOW before activation (dryrun isolation)
	resPreAct, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil || resPreAct.Decision != decision.Allow || resPreAct.PolicyVersion != recA.Version {
		t.Fatalf("dryrun mutated active state! decision=%s provenance=%s", resPreAct.Decision, resPreAct.PolicyVersion)
	}

	// 9. Activate Candidate B
	actRespB, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/"+recB.Version+"/activate", nil)
	if err != nil || actRespB.StatusCode != http.StatusOK {
		t.Fatalf("failed to activate candidate B: status=%d err=%v", actRespB.StatusCode, err)
	}
	actRespB.Body.Close()

	// 10. Observe live evaluation changed immediately -> DENY with recB provenance
	resB, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil {
		t.Fatalf("eval B failed: %v", err)
	}
	if resB.Decision != decision.Deny {
		t.Fatalf("expected DENY after activating B, got %s", resB.Decision)
	}
	if resB.PolicyVersion != recB.Version {
		t.Fatalf("provenance mismatch: expected %s, got %s", recB.Version, resB.PolicyVersion)
	}

	// 11. Rollback to Policy A
	rbResp, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/rollback", govapi.RollbackRequest{
		TargetVersion: recA.Version,
	})
	if err != nil || rbResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to rollback to A: status=%d err=%v", rbResp.StatusCode, err)
	}
	var rbData govapi.RollbackResponse
	json.NewDecoder(rbResp.Body).Decode(&rbData)
	rbResp.Body.Close()

	if rbData.ActiveVersion != recA.Version || rbData.RolledBackFrom != recB.Version {
		t.Fatalf("rollback metadata mismatch: active=%s from=%s", rbData.ActiveVersion, rbData.RolledBackFrom)
	}

	// 12. Observe restored live evaluation -> ALLOW with recA provenance
	resRestored, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil {
		t.Fatalf("eval restored failed: %v", err)
	}
	if resRestored.Decision != decision.Allow {
		t.Fatalf("expected restored ALLOW under policy A, got %s (reason: %s)", resRestored.Decision, resRestored.Reason)
	}
	if resRestored.PolicyVersion != recA.Version {
		t.Fatalf("restored provenance mismatch: expected %s, got %s", recA.Version, resRestored.PolicyVersion)
	}
}

// =============================================================================
// G4-QA-02: Dry-Run Isolation Proof
// =============================================================================

func TestDryRunIsolation(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()
	ws := "ws-dryrun-isolation"

	// Create and activate policy A
	candA, err := env.mgr.CreateCandidate(ctx, ws, fixturepolicy.CedarSource, "policy A")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.mgr.Activate(ctx, ws, candA.Version); err != nil {
		t.Fatal(err)
	}

	// Create candidate B
	candB, err := env.mgr.CreateCandidate(ctx, ws, "forbid(principal, action, resource);", "policy B")
	if err != nil {
		t.Fatal(err)
	}

	// Perform dry-run comparison multiple times
	for i := 0; i < 3; i++ {
		comp, err := env.govIntegration.DryRunCompare(ctx, ws, candB.Version, []decision.Request{makeReaderRequest(ws)})
		if err != nil {
			t.Fatalf("dry-run compare %d failed: %v", i, err)
		}
		if len(comp) != 1 || comp[0].CandidateResult.Decision != decision.Deny {
			t.Fatalf("unexpected dry-run output at %d: %+v", i, comp)
		}
	}

	// Verify active engine is unchanged and untouched
	activePol, err := env.store.GetActivePolicy(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if activePol.Version != candA.Version {
		t.Fatalf("active version changed: expected %s, got %s", candA.Version, activePol.Version)
	}

	liveRes, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil || liveRes.Decision != decision.Allow || liveRes.PolicyVersion != candA.Version {
		t.Fatalf("live evaluation affected by dry-run: decision=%s version=%s", liveRes.Decision, liveRes.PolicyVersion)
	}
}

// =============================================================================
// G4-QA-03: Failed Activation Leaves Previous Active Effective
// =============================================================================

func TestFailedActivationLeavesActiveEffective(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()
	ws := "ws-failed-activation"

	candA, err := env.mgr.CreateCandidate(ctx, ws, fixturepolicy.CedarSource, "policy A")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.mgr.Activate(ctx, ws, candA.Version); err != nil {
		t.Fatal(err)
	}

	// Attempt activating non-existent version via REST
	resp, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/nonexistent-hash/activate", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for bad activation, got %d", resp.StatusCode)
	}

	// Active policy and evaluation must still be A
	activePol, err := env.store.GetActivePolicy(ctx, ws)
	if err != nil || activePol.Version != candA.Version {
		t.Fatalf("active policy corrupted: %+v, err=%v", activePol, err)
	}

	res, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil || res.Decision != decision.Allow || res.PolicyVersion != candA.Version {
		t.Fatalf("live eval corrupted after failed activate: decision=%s version=%s", res.Decision, res.PolicyVersion)
	}
}

// =============================================================================
// G4-QA-04: Immediate Cache Coherence (No Stale Policy Served)
// =============================================================================

func TestImmediateCacheCoherence(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()
	ws := "ws-cache-coherence"

	candA, _ := env.mgr.CreateCandidate(ctx, ws, fixturepolicy.CedarSource, "A")
	env.mgr.Activate(ctx, ws, candA.Version)

	candB, _ := env.mgr.CreateCandidate(ctx, ws, "forbid(principal, action, resource);", "B")

	// Pre-condition: eval uses A
	resPre, _ := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if resPre.Decision != decision.Allow {
		t.Fatalf("expected ALLOW before activate B")
	}

	// Activate B
	if _, err := env.mgr.Activate(ctx, ws, candB.Version); err != nil {
		t.Fatal(err)
	}

	// Immediate next call MUST see B
	resPost, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws, makeReaderRequest(ws))
	if err != nil {
		t.Fatal(err)
	}
	if resPost.Decision != decision.Deny {
		t.Fatalf("stale cache served! expected DENY, got %s", resPost.Decision)
	}
	if resPost.PolicyVersion != candB.Version {
		t.Fatalf("stale provenance! expected %s, got %s", candB.Version, resPost.PolicyVersion)
	}
}

// =============================================================================
// G4-QA-05: Multi-Workspace Isolation
// =============================================================================

func TestWorkspaceIsolation(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()
	ws1 := "ws-isolation-1"
	ws2 := "ws-isolation-2"

	// Activate permit in ws1
	cand1, _ := env.mgr.CreateCandidate(ctx, ws1, fixturepolicy.CedarSource, "ws1 policy")
	env.mgr.Activate(ctx, ws1, cand1.Version)

	// ws2 has no policy loaded
	res2, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws2, makeReaderRequest(ws2))
	if err != nil {
		t.Fatal(err)
	}
	if res2.Decision != decision.Deny || res2.Reason != decision.ReasonNoPolicyLoaded {
		t.Fatalf("ws2 should have no_policy_loaded, got %s/%s", res2.Decision, res2.Reason)
	}

	// ws1 returns ALLOW
	res1, err := env.govIntegration.EvaluateWithActivePolicy(ctx, ws1, makeReaderRequest(ws1))
	if err != nil || res1.Decision != decision.Allow {
		t.Fatalf("ws1 should allow, got %s", res1.Decision)
	}

	// Activate forbid in ws2
	cand2, _ := env.mgr.CreateCandidate(ctx, ws2, "forbid(principal, action, resource);", "ws2 policy")
	env.mgr.Activate(ctx, ws2, cand2.Version)

	// ws1 is still ALLOW, ws2 is now DENY
	res1Post, _ := env.govIntegration.EvaluateWithActivePolicy(ctx, ws1, makeReaderRequest(ws1))
	res2Post, _ := env.govIntegration.EvaluateWithActivePolicy(ctx, ws2, makeReaderRequest(ws2))

	if res1Post.Decision != decision.Allow || res1Post.PolicyVersion != cand1.Version {
		t.Fatalf("ws1 was affected by ws2! decision=%s version=%s", res1Post.Decision, res1Post.PolicyVersion)
	}
	if res2Post.Decision != decision.Deny || res2Post.PolicyVersion != cand2.Version {
		t.Fatalf("ws2 decision incorrect! decision=%s version=%s", res2Post.Decision, res2Post.PolicyVersion)
	}
}

// =============================================================================
// G4-QA-06: Mutation Event Correlation & Audit Hooks
// =============================================================================

func TestMutationEventAuditHooks(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()
	ws := "ws-mutation-events"

	candA, _ := env.mgr.CreateCandidate(ctx, ws, fixturepolicy.CedarSource, "A")
	env.mgr.Activate(ctx, ws, candA.Version)

	candB, _ := env.mgr.CreateCandidate(ctx, ws, "forbid(principal, action, resource);", "B")
	env.mgr.Activate(ctx, ws, candB.Version)

	env.mgr.Rollback(ctx, ws, candA.Version)

	events := env.events.Events()
	if len(events) < 5 {
		t.Fatalf("expected at least 5 mutation events, got %d", len(events))
	}

	// Verify event fields
	actions := make([]auditevents.MutationAction, 0, len(events))
	for _, ev := range events {
		if ev.WorkspaceID != ws {
			t.Fatalf("workspace mismatch in event: %s", ev.WorkspaceID)
		}
		if ev.Timestamp.IsZero() {
			t.Fatal("event timestamp is zero")
		}
		actions = append(actions, ev.Action)
	}

	expectedActions := []auditevents.MutationAction{
		auditevents.ActionCreateCandidate,
		auditevents.ActionActivate,
		auditevents.ActionCreateCandidate,
		auditevents.ActionActivate,
		auditevents.ActionRollback,
	}

	for i, exp := range expectedActions {
		if actions[i] != exp {
			t.Fatalf("event %d action mismatch: expected %s, got %s", i, exp, actions[i])
		}
	}
}

// =============================================================================
// G4-QA-07: Admin Authentication Boundary Protection
// =============================================================================

func TestAdminAuthBoundary(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ws := "ws-auth-boundary"

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/workspaces/" + ws + "/policies"},
		{"POST", "/api/v1/workspaces/" + ws + "/policies"},
		{"POST", "/api/v1/workspaces/" + ws + "/policies/validate"},
		{"POST", "/api/v1/workspaces/" + ws + "/policies/rollback"},
		{"POST", "/api/v1/workspaces/" + ws + "/policies/v1/activate"},
		{"POST", "/api/v1/workspaces/" + ws + "/policies/v1/preview"},
		{"POST", "/api/v1/workspaces/" + ws + "/policies/v1/dryrun"},
	}

	for _, ep := range endpoints {
		// 1. Missing Authorization header -> 401
		req, _ := http.NewRequest(ep.method, env.server.URL+ep.path, bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s failed: %v", ep.method, ep.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s missing auth: expected 401, got %d", ep.method, ep.path, resp.StatusCode)
		}

		// 2. Invalid Authorization header -> 401
		reqBad, _ := http.NewRequest(ep.method, env.server.URL+ep.path, bytes.NewReader([]byte("{}")))
		reqBad.Header.Set("Authorization", "Bearer wrong-token")
		reqBad.Header.Set("Content-Type", "application/json")
		respBad, err := http.DefaultClient.Do(reqBad)
		if err != nil {
			t.Fatalf("%s %s bad token failed: %v", ep.method, ep.path, err)
		}
		respBad.Body.Close()
		if respBad.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s bad token: expected 401, got %d", ep.method, ep.path, respBad.StatusCode)
		}
	}
}

// =============================================================================
// G4-QA-08: Fail Closed on Uninitialized / Missing Policy
// =============================================================================

func TestFailClosedOnMissingPolicy(t *testing.T) {
	env := setupQAEnv()
	defer env.close()
	ctx := context.Background()

	res, err := env.govIntegration.EvaluateWithActivePolicy(ctx, "nonexistent-ws", makeReaderRequest("nonexistent-ws"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Decision != decision.Deny {
		t.Fatalf("expected DENY, got %s", res.Decision)
	}
	if res.Reason != decision.ReasonNoPolicyLoaded {
		t.Fatalf("expected ReasonNoPolicyLoaded, got %s", res.Reason)
	}
	if res.PolicyVersion != "" {
		t.Fatalf("expected empty policy version on fail-closed, got %s", res.PolicyVersion)
	}
}
