package policymanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/policy"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
)

func TestManager_Validate(t *testing.T) {
	store := policystore.NewMemoryStore()
	defer store.Close()
	mgr := New(store)

	validPolicy := `permit(principal, action, resource);`
	v, err := mgr.Validate(validPolicy)
	if err != nil {
		t.Fatalf("Validate failed for valid policy: %v", err)
	}
	expectedV := policystore.ComputeVersion(validPolicy)
	if v != expectedV {
		t.Fatalf("expected version %s, got %s", expectedV, v)
	}

	invalidPolicy := `this is not a valid cedar syntax;`
	_, err = mgr.Validate(invalidPolicy)
	if err == nil {
		t.Fatal("expected Validate to fail for invalid syntax, but got nil error")
	}
}

func TestManager_CreateCandidate(t *testing.T) {
	ctx := context.Background()
	store := policystore.NewMemoryStore()
	defer store.Close()
	mgr := New(store)

	ws := "ws-test"

	// 1. Valid candidate
	validPolicy := `permit(principal, action, resource);`
	rec, err := mgr.CreateCandidate(ctx, ws, validPolicy, "test candidate")
	if err != nil {
		t.Fatalf("CreateCandidate failed: %v", err)
	}
	if rec.State != policystore.StateCandidate {
		t.Fatalf("expected state candidate, got %s", rec.State)
	}

	// 2. Invalid candidate must be rejected before saving to store
	invalidPolicy := `not cedar`
	_, err = mgr.CreateCandidate(ctx, ws, invalidPolicy, "invalid")
	if err == nil {
		t.Fatal("expected error creating invalid candidate, got nil")
	}

	// Verify invalid policy was not stored
	list, err := store.ListPolicies(ctx, ws)
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 policy in store, got %d", len(list))
	}
}

func TestManager_Activate_And_Rollback(t *testing.T) {
	ctx := context.Background()
	store := policystore.NewMemoryStore()
	defer store.Close()
	mgr := New(store)

	ws := "ws-lifecycle"

	policy1 := `permit(principal, action, resource);`
	policy2 := `forbid(principal, action, resource);`

	c1, err := mgr.CreateCandidate(ctx, ws, policy1, "v1 permit")
	if err != nil {
		t.Fatalf("create c1: %v", err)
	}
	c2, err := mgr.CreateCandidate(ctx, ws, policy2, "v2 forbid")
	if err != nil {
		t.Fatalf("create c2: %v", err)
	}

	// Active initially none
	_, err = mgr.GetActiveEngine(ws)
	if !errors.Is(err, ErrNoActiveEngine) {
		t.Fatalf("expected ErrNoActiveEngine, got: %v", err)
	}

	// Activate c1
	rec1, err := mgr.Activate(ctx, ws, c1.Version)
	if err != nil {
		t.Fatalf("Activate c1 failed: %v", err)
	}
	if rec1.Version != c1.Version || rec1.State != policystore.StateActive {
		t.Fatalf("unexpected record on activate: %+v", rec1)
	}

	engine1, err := mgr.GetActiveEngine(ws)
	if err != nil {
		t.Fatalf("GetActiveEngine failed: %v", err)
	}
	if engine1.Version() != c1.Version {
		t.Fatalf("engine version mismatch: got %s, want %s", engine1.Version(), c1.Version)
	}

	// Activate c2
	rec2, err := mgr.Activate(ctx, ws, c2.Version)
	if err != nil {
		t.Fatalf("Activate c2 failed: %v", err)
	}
	if rec2.Version != c2.Version {
		t.Fatalf("unexpected record on activate c2: %+v", rec2)
	}

	engine2, err := mgr.GetActiveEngine(ws)
	if err != nil {
		t.Fatalf("GetActiveEngine failed: %v", err)
	}
	if engine2.Version() != c2.Version {
		t.Fatalf("engine version mismatch: got %s, want %s", engine2.Version(), c2.Version)
	}

	// Activate unknown version should fail and leave active engine at c2
	_, err = mgr.Activate(ctx, ws, "unknown-hash")
	if err == nil {
		t.Fatal("expected error activating unknown version")
	}

	engineAfterBadActivate, err := mgr.GetActiveEngine(ws)
	if err != nil || engineAfterBadActivate.Version() != c2.Version {
		t.Fatalf("active engine corrupted after bad activate: %v", err)
	}

	// Rollback to c1
	recRollback, err := mgr.Rollback(ctx, ws, c1.Version)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if recRollback.Version != c1.Version {
		t.Fatalf("expected rolled back version %s, got %s", c1.Version, recRollback.Version)
	}

	engineAfterRollback, err := mgr.GetActiveEngine(ws)
	if err != nil || engineAfterRollback.Version() != c1.Version {
		t.Fatalf("active engine version mismatch after rollback: got %s, want %s", engineAfterRollback.Version(), c1.Version)
	}

	// Rollback to unknown should fail and leave active at c1
	_, err = mgr.Rollback(ctx, ws, "non-existent")
	if err == nil {
		t.Fatal("expected error rolling back to unknown version")
	}

	engineAfterBadRollback, err := mgr.GetActiveEngine(ws)
	if err != nil || engineAfterBadRollback.Version() != c1.Version {
		t.Fatalf("active engine changed after bad rollback: %v", err)
	}
}

func TestManager_Concurrency(t *testing.T) {
	ctx := context.Background()
	store := policystore.NewMemoryStore()
	defer store.Close()
	mgr := New(store)

	ws := "ws-concurrency"

	policy1 := `permit(principal, action, resource);`
	policy2 := `forbid(principal, action, resource);`

	c1, err := mgr.CreateCandidate(ctx, ws, policy1, "v1")
	if err != nil {
		t.Fatalf("create c1: %v", err)
	}
	c2, err := mgr.CreateCandidate(ctx, ws, policy2, "v2")
	if err != nil {
		t.Fatalf("create c2: %v", err)
	}

	// Activate c1 first
	if _, err := mgr.Activate(ctx, ws, c1.Version); err != nil {
		t.Fatalf("initial activate: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Start 30 concurrent readers
	readersCount := 30
	readErrors := make(chan error, readersCount*100)

	for i := 0; i < readersCount; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					engine, err := mgr.GetActiveEngine(ws)
					if err != nil {
						readErrors <- fmt.Errorf("reader %d got error: %w", readerID, err)
						return
					}
					v := engine.Version()
					if v != c1.Version && v != c2.Version {
						readErrors <- fmt.Errorf("reader %d saw invalid version: %s", readerID, v)
						return
					}
					// Verify evaluation works safely
					evalOut := engine.Evaluate(policy.EvalInput{
						PrincipalID:    "user",
						PrincipalRoles: []string{"role"},
						ResourceID:     "tool",
						ResourceRisk:   "read",
					})
					_ = evalOut
				}
			}
		}(i)
	}

	// Perform 10 back-and-forth activations
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Millisecond)
		if _, err := mgr.Activate(ctx, ws, c2.Version); err != nil {
			t.Errorf("activate c2 in loop %d: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
		if _, err := mgr.Activate(ctx, ws, c1.Version); err != nil {
			t.Errorf("activate c1 in loop %d: %v", i, err)
		}
	}

	close(done)
	wg.Wait()
	close(readErrors)

	for err := range readErrors {
		t.Fatal(err)
	}
}

func TestManager_Preview(t *testing.T) {
	ctx := context.Background()
	store := policystore.NewMemoryStore()
	defer store.Close()
	mgr := New(store)

	ws := "ws-preview"

	policyV1 := `permit(principal == AgentGate::Agent::"alice", action, resource);`
	policyV2 := `forbid(principal == AgentGate::Agent::"alice", action, resource);`

	c1, _ := mgr.CreateCandidate(ctx, ws, policyV1, "v1")
	c2, _ := mgr.CreateCandidate(ctx, ws, policyV2, "v2")

	// Activate V1
	mgr.Activate(ctx, ws, c1.Version)

	input := policy.EvalInput{
		PrincipalID:    "alice",
		PrincipalRoles: []string{"role"},
		ResourceID:     "tool-1",
		ResourceRisk:   "read",
	}

	// Preview against candidate V2
	results, err := mgr.Preview(ctx, ws, c2.Version, []policy.EvalInput{input})
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Allowed {
		t.Fatal("expected preview of forbid policy to return Allowed=false")
	}

	// Verify active engine is STILL V1 (permit)
	activeEngine, err := mgr.GetActiveEngine(ws)
	if err != nil {
		t.Fatalf("get active engine: %v", err)
	}
	if activeEngine.Version() != c1.Version {
		t.Fatalf("active engine version changed unexpectedly: got %s, want %s", activeEngine.Version(), c1.Version)
	}
	activeResult := activeEngine.Evaluate(input)
	if !activeResult.Allowed {
		t.Fatal("expected active engine to still permit alice")
	}
}
