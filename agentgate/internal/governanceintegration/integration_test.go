package governanceintegration_test

import (
	"context"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
	"github.com/Dynamisch-LLC/agentgate/internal/governanceintegration"
	"github.com/Dynamisch-LLC/agentgate/internal/policymanager"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
)

func setupService(t *testing.T) (*governanceintegration.GovernanceDecisionService, *policymanager.Manager) {
	t.Helper()
	store := policystore.NewMemoryStore()
	mgr := policymanager.New(store)
	svc := governanceintegration.NewGovernanceDecisionService(mgr)
	return svc, mgr
}

func readerReadReq(ws, execID string) decision.Request {
	return decision.Request{
		ExecutionID:    execID,
		WorkspaceID:    ws,
		Identity:       decision.Identity{AgentID: "agent-1", Roles: []string{fixturepolicy.RoleReader}},
		Tool:           decision.ToolRef{BackendID: "backend-1", Name: "read-tool"},
		Classification: decision.ToolClassification{Known: true, Risk: fixturepolicy.RiskRead},
	}
}

func TestEvaluateWithActivePolicyReturnsCorrectDecision(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	cand, err := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Activate(ctx, "ws-1", cand.Version); err != nil {
		t.Fatal(err)
	}

	result, err := svc.EvaluateWithActivePolicy(ctx, "ws-1", readerReadReq("ws-1", "exec-1"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != decision.Allow {
		t.Fatalf("expected ALLOW, got %s: %s", result.Decision, result.Message)
	}
	if result.PolicyVersion != cand.Version {
		t.Fatalf("provenance mismatch: expected %s, got %s", cand.Version, result.PolicyVersion)
	}
}

func TestEvaluateWithNoActivePolicyDenies(t *testing.T) {
	svc, _ := setupService(t)
	ctx := context.Background()

	result, err := svc.EvaluateWithActivePolicy(ctx, "ws-1", readerReadReq("ws-1", "exec-2"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != decision.Deny {
		t.Fatal("expected DENY with no active policy")
	}
	if result.Reason != decision.ReasonNoPolicyLoaded {
		t.Fatalf("expected reason no_policy_loaded, got %s", result.Reason)
	}
}

func TestDryRunCompareShowsDifference(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	// Activate policy A: reader can read
	candA, _ := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "policy A")
	mgr.Activate(ctx, "ws-1", candA.Version)

	// Create candidate B: restrictive — forbids everything
	restrictive := `forbid(principal, action, resource);`
	candB, _ := mgr.CreateCandidate(ctx, "ws-1", restrictive, "policy B")

	comparisons, err := svc.DryRunCompare(ctx, "ws-1", candB.Version, []decision.Request{
		readerReadReq("ws-1", "exec-3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(comparisons) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(comparisons))
	}
	if comparisons[0].ActiveResult.Decision != decision.Allow {
		t.Fatal("active should ALLOW reader+read")
	}
	if comparisons[0].CandidateResult.Decision != decision.Deny {
		t.Fatal("candidate should DENY (forbid all)")
	}
}

func TestDryRunCompareDoesNotMutateActive(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	candA, _ := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "policy A")
	mgr.Activate(ctx, "ws-1", candA.Version)

	restrictive := `forbid(principal, action, resource);`
	candB, _ := mgr.CreateCandidate(ctx, "ws-1", restrictive, "policy B")

	// Dry-run B
	_, err := svc.DryRunCompare(ctx, "ws-1", candB.Version, []decision.Request{
		readerReadReq("ws-1", "exec-4"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Active should still be A and reader+read should still ALLOW
	result, err := svc.EvaluateWithActivePolicy(ctx, "ws-1", readerReadReq("ws-1", "exec-5"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != decision.Allow {
		t.Fatal("dry-run must not mutate active policy — reader+read should still ALLOW")
	}
	if result.PolicyVersion != candA.Version {
		t.Fatalf("active provenance should still be A (%s), got %s", candA.Version, result.PolicyVersion)
	}
}

func TestActivationChangesLiveDecision(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	// Activate A
	candA, _ := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "policy A")
	mgr.Activate(ctx, "ws-1", candA.Version)

	result1, _ := svc.EvaluateWithActivePolicy(ctx, "ws-1", readerReadReq("ws-1", "exec-6"))
	if result1.Decision != decision.Allow {
		t.Fatal("A should ALLOW reader+read")
	}

	// Activate B (forbid all)
	restrictive := `forbid(principal, action, resource);`
	candB, _ := mgr.CreateCandidate(ctx, "ws-1", restrictive, "policy B")
	mgr.Activate(ctx, "ws-1", candB.Version)

	result2, _ := svc.EvaluateWithActivePolicy(ctx, "ws-1", readerReadReq("ws-1", "exec-7"))
	if result2.Decision != decision.Deny {
		t.Fatal("B should DENY reader+read")
	}
	if result2.PolicyVersion != candB.Version {
		t.Fatalf("provenance should be B (%s), got %s", candB.Version, result2.PolicyVersion)
	}
}

func TestRollbackRestoresDecision(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	// Activate A, then B, then rollback to A
	candA, _ := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "policy A")
	mgr.Activate(ctx, "ws-1", candA.Version)

	restrictive := `forbid(principal, action, resource);`
	candB, _ := mgr.CreateCandidate(ctx, "ws-1", restrictive, "policy B")
	mgr.Activate(ctx, "ws-1", candB.Version)

	// Rollback to A
	mgr.Rollback(ctx, "ws-1", candA.Version)

	result, _ := svc.EvaluateWithActivePolicy(ctx, "ws-1", readerReadReq("ws-1", "exec-8"))
	if result.Decision != decision.Allow {
		t.Fatal("rollback to A should restore ALLOW for reader+read")
	}
	if result.PolicyVersion != candA.Version {
		t.Fatalf("provenance should be A (%s), got %s", candA.Version, result.PolicyVersion)
	}
}

func TestDryRunWithNoActivePolicy(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	// Only create candidate, don't activate
	cand, _ := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "test")

	comparisons, err := svc.DryRunCompare(ctx, "ws-1", cand.Version, []decision.Request{
		readerReadReq("ws-1", "exec-9"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(comparisons) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(comparisons))
	}
	if comparisons[0].ActiveResult.Decision != decision.Deny {
		t.Fatal("with no active policy, active result should be DENY")
	}
	if comparisons[0].ActiveResult.Reason != decision.ReasonNoPolicyLoaded {
		t.Fatalf("expected no_policy_loaded, got %s", comparisons[0].ActiveResult.Reason)
	}
	if comparisons[0].CandidateResult.Decision != decision.Allow {
		t.Fatal("candidate (fixturepolicy) should ALLOW reader+read")
	}
}

func TestDryRunCompareMultipleRequests(t *testing.T) {
	svc, mgr := setupService(t)
	ctx := context.Background()

	candA, _ := mgr.CreateCandidate(ctx, "ws-1", fixturepolicy.CedarSource, "A")
	mgr.Activate(ctx, "ws-1", candA.Version)

	restrictive := `forbid(principal, action, resource);`
	candB, _ := mgr.CreateCandidate(ctx, "ws-1", restrictive, "B")

	requests := []decision.Request{
		readerReadReq("ws-1", "exec-10"),
		{
			ExecutionID:    "exec-11",
			WorkspaceID:    "ws-1",
			Identity:       decision.Identity{AgentID: "agent-2", Roles: []string{fixturepolicy.RoleAdmin}},
			Tool:           decision.ToolRef{BackendID: "b1", Name: "write-tool"},
			Classification: decision.ToolClassification{Known: true, Risk: fixturepolicy.RiskWrite},
		},
	}

	comparisons, err := svc.DryRunCompare(ctx, "ws-1", candB.Version, requests)
	if err != nil {
		t.Fatal(err)
	}
	if len(comparisons) != 2 {
		t.Fatalf("expected 2 comparisons, got %d", len(comparisons))
	}
	// Both should show difference: A allows, B forbids
	for i, comp := range comparisons {
		if comp.ActiveResult.Decision != decision.Allow {
			t.Fatalf("comparison[%d]: active should ALLOW", i)
		}
		if comp.CandidateResult.Decision != decision.Deny {
			t.Fatalf("comparison[%d]: candidate should DENY", i)
		}
	}
}
