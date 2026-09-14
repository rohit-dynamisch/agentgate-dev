package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/audit"
	"github.com/Dynamisch-LLC/agentgate/internal/auditevents"
)

func TestMemoryStore_AppendAndRetrieve(t *testing.T) {
	store := audit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()
	ws := "ws-audit-test"

	rec := audit.DecisionRecord{
		WorkspaceID:       ws,
		ExecutionID:       "exec-1",
		Timestamp:         time.Now().UTC(),
		Decision:          "ALLOW",
		Reason:            "policy_allow",
		PrincipalAgentID:  "agent-1",
		PrincipalRoles:    []string{"reader"},
		ToolBackendID:     "backend-1",
		ToolName:          "read-tool",
		ToolRisk:          "read",
		PolicyVersion:     "ver-123",
		PolicyHash:        "hash-123",
		RedactedArguments: map[string]string{"query": "hello"},
		PrevHash:          audit.GenesisHash,
		RowHash:           "hash-placeholder-1",
	}

	stored, err := store.AppendDecision(ctx, rec)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if stored.SequenceNumber != 1 {
		t.Fatalf("expected sequence 1, got %d", stored.SequenceNumber)
	}
	if stored.EventType != audit.EventTypeDecision {
		t.Fatalf("expected event type decision, got %s", stored.EventType)
	}

	latest, err := store.GetLatestRecord(ctx, ws)
	if err != nil {
		t.Fatalf("get latest failed: %v", err)
	}
	if latest.ExecutionID != "exec-1" {
		t.Fatalf("expected exec-1, got %s", latest.ExecutionID)
	}
	if latest.PrevHash != audit.GenesisHash {
		t.Fatalf("expected genesis hash, got %s", latest.PrevHash)
	}
}

func TestMemoryStore_SequenceMonotonicPerWorkspace(t *testing.T) {
	store := audit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	ws1 := "ws-seq-1"
	ws2 := "ws-seq-2"

	for i := 1; i <= 3; i++ {
		rec1 := audit.DecisionRecord{
			WorkspaceID:      ws1,
			ExecutionID:      "exec-ws1",
			Decision:         "DENY",
			Reason:           "policy_deny",
			PrincipalAgentID: "agent-1",
			ToolBackendID:    "b1",
			ToolName:         "t1",
			ToolRisk:         "write",
			PrevHash:         "prev",
			RowHash:          "row",
		}
		stored1, err := store.AppendDecision(ctx, rec1)
		if err != nil {
			t.Fatal(err)
		}
		if stored1.SequenceNumber != int64(i) {
			t.Fatalf("expected ws1 sequence %d, got %d", i, stored1.SequenceNumber)
		}

		rec2 := audit.DecisionRecord{
			WorkspaceID:      ws2,
			ExecutionID:      "exec-ws2",
			Decision:         "ALLOW",
			Reason:           "policy_allow",
			PrincipalAgentID: "agent-2",
			ToolBackendID:    "b1",
			ToolName:         "t1",
			ToolRisk:         "read",
			PrevHash:         "prev",
			RowHash:          "row",
		}
		stored2, err := store.AppendDecision(ctx, rec2)
		if err != nil {
			t.Fatal(err)
		}
		if stored2.SequenceNumber != int64(i) {
			t.Fatalf("expected ws2 sequence %d, got %d", i, stored2.SequenceNumber)
		}
	}
}

func TestMemoryStore_ListAndGetBySequence(t *testing.T) {
	store := audit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()
	ws := "ws-list-test"

	for i := 1; i <= 5; i++ {
		rec := audit.DecisionRecord{
			WorkspaceID:      ws,
			ExecutionID:      "exec",
			Decision:         "ALLOW",
			Reason:           "policy_allow",
			PrincipalAgentID: "agent-1",
			ToolBackendID:    "b1",
			ToolName:         "t1",
			ToolRisk:         "read",
			PrevHash:         "prev",
			RowHash:          "row",
		}
		_, err := store.AppendDecision(ctx, rec)
		if err != nil {
			t.Fatal(err)
		}
	}

	records, err := store.ListRecords(ctx, ws, 3)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	rec3, err := store.GetRecordBySequence(ctx, ws, 3)
	if err != nil {
		t.Fatalf("get by sequence failed: %v", err)
	}
	if rec3.SequenceNumber != 3 {
		t.Fatalf("expected sequence 3, got %d", rec3.SequenceNumber)
	}

	_, err = store.GetRecordBySequence(ctx, ws, 99)
	if !errors.Is(err, audit.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_AppendMutation(t *testing.T) {
	store := audit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()
	ws := "ws-mut-test"

	ev := auditevents.MutationEvent{
		WorkspaceID:     ws,
		Action:          auditevents.ActionActivate,
		PreviousVersion: "v1",
		NewVersion:      "v2",
		CorrelationID:   "corr-123",
		Timestamp:       time.Now().UTC(),
		OperatorID:      "admin-user",
	}

	stored, err := store.AppendMutation(ctx, ev, audit.GenesisHash)
	if err != nil {
		t.Fatalf("append mutation failed: %v", err)
	}
	if stored.SequenceNumber != 1 {
		t.Fatalf("expected sequence 1, got %d", stored.SequenceNumber)
	}
	if stored.EventType != audit.EventTypeMutation {
		t.Fatalf("expected event type mutation, got %s", stored.EventType)
	}
	if stored.PolicyVersion != "v2" {
		t.Fatalf("expected version v2, got %s", stored.PolicyVersion)
	}
}

func TestMemoryStore_FaultInjection(t *testing.T) {
	store := audit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	faultErr := errors.New("simulated postgres failure")
	store.SetFault(faultErr)

	rec := audit.DecisionRecord{
		WorkspaceID:      "ws-fault",
		ExecutionID:      "exec-fault",
		Decision:         "ALLOW",
		Reason:           "policy_allow",
		PrincipalAgentID: "agent-1",
		ToolBackendID:    "b1",
		ToolName:         "t1",
		ToolRisk:         "read",
		PrevHash:         audit.GenesisHash,
		RowHash:          "row",
	}

	_, err := store.AppendDecision(ctx, rec)
	if !errors.Is(err, faultErr) {
		t.Fatalf("expected injected fault error, got %v", err)
	}
}
