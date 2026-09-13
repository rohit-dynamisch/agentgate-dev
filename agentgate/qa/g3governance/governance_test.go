package g3governance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/govapi"
	"github.com/Dynamisch-LLC/agentgate/internal/policy"
	"github.com/Dynamisch-LLC/agentgate/internal/policymanager"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
)

const testAdminToken = "test-qa-admin-token"

type testEnv struct {
	server *httptest.Server
	mgr    *policymanager.Manager
	store  policystore.Store
}

func setupQAEnv() *testEnv {
	store := policystore.NewMemoryStore()
	mgr := policymanager.New(store)
	handler := govapi.NewHandler(mgr, testAdminToken)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	return &testEnv{
		server: ts,
		mgr:    mgr,
		store:  store,
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

// =============================================================================
// G3-QA-01: Lifecycle Invariant Verification
// =============================================================================

func TestLifecycleInvariants(t *testing.T) {
	env := setupQAEnv()
	defer env.close()

	ws := "qa-workspace"
	permitPolicy := `permit(principal, action, resource);`
	invalidPolicy := `broken syntax !!! not cedar`

	// Invariant 1: Validate does not alter store or active state
	valResp, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/validate", govapi.ValidateRequest{Content: permitPolicy})
	if err != nil || valResp.StatusCode != http.StatusOK {
		t.Fatalf("validate request failed: %v, code: %d", err, valResp.StatusCode)
	}
	valResp.Body.Close()

	listResp, _ := env.authedRequest("GET", "/api/v1/workspaces/"+ws+"/policies", nil)
	var list govapi.ListPoliciesResponse
	json.NewDecoder(listResp.Body).Decode(&list)
	listResp.Body.Close()
	if len(list.Policies) != 0 {
		t.Fatalf("Invariant violation: Validate persisted a policy record, expected 0, got %d", len(list.Policies))
	}

	// Invariant 2: Invalid candidate is rejected and not persisted
	createBadResp, _ := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies", govapi.CreateCandidateRequest{
		Content:     invalidPolicy,
		Description: "should fail",
	})
	if createBadResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Invariant violation: expected 400 for invalid candidate, got %d", createBadResp.StatusCode)
	}
	createBadResp.Body.Close()

	// Invariant 3: Valid candidate can be created and activated
	createGoodResp, _ := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies", govapi.CreateCandidateRequest{
		Content:     permitPolicy,
		Description: "v1 permit",
	})
	if createGoodResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", createGoodResp.StatusCode)
	}
	var createdRec policystore.PolicyRecord
	json.NewDecoder(createGoodResp.Body).Decode(&createdRec)
	createGoodResp.Body.Close()

	actResp, _ := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/"+createdRec.Version+"/activate", nil)
	if actResp.StatusCode != http.StatusOK {
		t.Fatalf("activate failed with %d", actResp.StatusCode)
	}
	actResp.Body.Close()

	// Invariant 4: Exactly one policy is active
	listResp2, _ := env.authedRequest("GET", "/api/v1/workspaces/"+ws+"/policies", nil)
	json.NewDecoder(listResp2.Body).Decode(&list)
	listResp2.Body.Close()

	activeCount := 0
	for _, p := range list.Policies {
		if p.State == policystore.StateActive {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("Invariant violation: expected exactly 1 active policy, found %d", activeCount)
	}

	// Invariant 5: Rollback to unknown version leaves active version unchanged
	rbBadResp, _ := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/rollback", govapi.RollbackRequest{
		TargetVersion: "unknown-target-version",
	})
	if rbBadResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown rollback, got %d", rbBadResp.StatusCode)
	}
	rbBadResp.Body.Close()

	activeEng, err := env.mgr.GetActiveEngine(ws)
	if err != nil || activeEng.Version() != createdRec.Version {
		t.Fatalf("Invariant violation: active engine mutated on failed rollback")
	}
}

// =============================================================================
// G3-QA-02: Concurrency & Atomicity Invariants
// =============================================================================

func TestConcurrencyInvariants(t *testing.T) {
	env := setupQAEnv()
	defer env.close()

	ws := "qa-concurrency"
	policy1 := `permit(principal, action, resource);`
	policy2 := `forbid(principal, action, resource);`

	c1, _ := env.mgr.CreateCandidate(context.Background(), ws, policy1, "v1")
	c2, _ := env.mgr.CreateCandidate(context.Background(), ws, policy2, "v2")

	env.mgr.Activate(context.Background(), ws, c1.Version)

	done := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, 1000)

	// Launch 25 reader threads
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					eng, err := env.mgr.GetActiveEngine(ws)
					if err != nil {
						errCh <- fmt.Errorf("reader %d saw error: %w", id, err)
						return
					}
					v := eng.Version()
					if v != c1.Version && v != c2.Version {
						errCh <- fmt.Errorf("reader %d observed torn/invalid version: %s", id, v)
						return
					}
					out := eng.Evaluate(policy.EvalInput{
						PrincipalID:    "agent-1",
						PrincipalRoles: []string{"reader"},
						ResourceID:     "tool-1",
						ResourceRisk:   "read",
					})
					if out.HadError {
						errCh <- fmt.Errorf("reader %d encountered evaluation error", id)
						return
					}
				}
			}
		}(i)
	}

	// Rapidly alternate active policies
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Millisecond)
		_, _ = env.mgr.Activate(context.Background(), ws, c2.Version)
		time.Sleep(1 * time.Millisecond)
		_, _ = env.mgr.Activate(context.Background(), ws, c1.Version)
	}

	close(done)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("Concurrency invariant violation: %v", err)
	}
}

// =============================================================================
// G3-QA-03: Multi-tenant Workspace Isolation
// =============================================================================

func TestWorkspaceIsolation(t *testing.T) {
	env := setupQAEnv()
	defer env.close()

	wsA := "tenant-alpha"
	wsB := "tenant-bravo"

	policyA := `permit(principal, action, resource);`
	policyB := `forbid(principal, action, resource);`

	cA, err := env.mgr.CreateCandidate(context.Background(), wsA, policyA, "A")
	if err != nil {
		t.Fatalf("create cA: %v", err)
	}
	cB, err := env.mgr.CreateCandidate(context.Background(), wsB, policyB, "B")
	if err != nil {
		t.Fatalf("create cB: %v", err)
	}

	env.mgr.Activate(context.Background(), wsA, cA.Version)
	env.mgr.Activate(context.Background(), wsB, cB.Version)

	// Workspace A cannot retrieve Workspace B's policy
	respCross, err := env.authedRequest("GET", "/api/v1/workspaces/"+wsA+"/policies/"+cB.Version, nil)
	if err != nil {
		t.Fatalf("cross-get request: %v", err)
	}
	defer respCross.Body.Close()

	if respCross.StatusCode != http.StatusNotFound {
		t.Fatalf("Isolation violation: wsA accessed wsB policy version, got status %d", respCross.StatusCode)
	}

	// Workspace A cannot activate Workspace B's version
	respActCross, err := env.authedRequest("POST", "/api/v1/workspaces/"+wsA+"/policies/"+cB.Version+"/activate", nil)
	if err != nil {
		t.Fatalf("cross-activate request: %v", err)
	}
	defer respActCross.Body.Close()

	if respActCross.StatusCode != http.StatusNotFound {
		t.Fatalf("Isolation violation: wsA activated wsB policy version, got status %d", respActCross.StatusCode)
	}
}

// =============================================================================
// G3-QA-04: Security & Authentication Boundaries
// =============================================================================

func TestSecurityBoundaries(t *testing.T) {
	env := setupQAEnv()
	defer env.close()

	ws := "qa-sec"

	// 1. Missing authentication fails with 401
	unauthReq, _ := http.NewRequest("POST", env.server.URL+"/api/v1/workspaces/"+ws+"/policies/validate", bytes.NewReader([]byte("{}")))
	unauthResp, err := http.DefaultClient.Do(unauthReq)
	if err != nil {
		t.Fatalf("unauth req: %v", err)
	}
	unauthResp.Body.Close()
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", unauthResp.StatusCode)
	}

	// 2. Wrong credentials fails with 401
	badAuthReq, _ := http.NewRequest("POST", env.server.URL+"/api/v1/workspaces/"+ws+"/policies/validate", bytes.NewReader([]byte("{}")))
	badAuthReq.Header.Set("Authorization", "Bearer invalid-token")
	badAuthResp, err := http.DefaultClient.Do(badAuthReq)
	if err != nil {
		t.Fatalf("bad auth req: %v", err)
	}
	badAuthResp.Body.Close()
	if badAuthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", badAuthResp.StatusCode)
	}

	// 3. Malformed JSON payload fails with 400
	malformedResp, err := env.authedRequest("POST", "/api/v1/workspaces/"+ws+"/policies/validate", "not-a-json-object")
	if err != nil {
		t.Fatalf("malformed req: %v", err)
	}
	defer malformedResp.Body.Close()
	if malformedResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed payload, got %d", malformedResp.StatusCode)
	}
}

// =============================================================================
// G3-QA-05: Decision Engine Exact Policy Provenance
// =============================================================================

func TestDecisionEnginePolicyProvenance(t *testing.T) {
	env := setupQAEnv()
	defer env.close()

	ws := "qa-provenance"
	policySrc := `permit(principal, action, resource);`

	c1, err := env.mgr.CreateCandidate(context.Background(), ws, policySrc, "v1")
	if err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	_, err = env.mgr.Activate(context.Background(), ws, c1.Version)
	if err != nil {
		t.Fatalf("activate candidate: %v", err)
	}

	activeEng, err := env.mgr.GetActiveEngine(ws)
	if err != nil {
		t.Fatalf("get active engine: %v", err)
	}

	// Decision core evaluation carries exact active version
	decEngine := decision.NewEngineWithPolicy(activeEng)
	req := decision.Request{
		ExecutionID: "exec-test-001",
		WorkspaceID: ws,
		Identity: decision.Identity{
			AgentID: "agent-007",
			Roles:   []string{"operator"},
		},
		Tool: decision.ToolRef{
			BackendID: "default",
			Name:      "database-query",
		},
		Classification: decision.ToolClassification{
			Known: true,
			Risk:  "read",
		},
	}

	res := decEngine.Evaluate(req)
	if res.PolicyVersion != c1.Version {
		t.Fatalf("Provenance violation: decision carried version %s, want exact active version %s", res.PolicyVersion, c1.Version)
	}
}
