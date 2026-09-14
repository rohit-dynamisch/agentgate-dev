package auditevents_test

import (
	"context"
	"testing"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/auditevents"
)

func TestMutationEventFields(t *testing.T) {
	ev := auditevents.MutationEvent{
		WorkspaceID:     "ws-1",
		Action:          auditevents.ActionActivate,
		PreviousVersion: "aaa",
		NewVersion:      "bbb",
		CorrelationID:   "corr-1",
		Timestamp:       time.Now(),
		OperatorID:      "admin",
	}
	if ev.WorkspaceID != "ws-1" {
		t.Fatal("workspace mismatch")
	}
	if ev.Action != auditevents.ActionActivate {
		t.Fatal("action mismatch")
	}
	if ev.PreviousVersion != "aaa" {
		t.Fatal("previous version mismatch")
	}
	if ev.NewVersion != "bbb" {
		t.Fatal("new version mismatch")
	}
}

func TestMutationActionConstants(t *testing.T) {
	if auditevents.ActionActivate != "activate" {
		t.Fatal("ActionActivate value mismatch")
	}
	if auditevents.ActionRollback != "rollback" {
		t.Fatal("ActionRollback value mismatch")
	}
	if auditevents.ActionCreateCandidate != "create_candidate" {
		t.Fatal("ActionCreateCandidate value mismatch")
	}
}

func TestNoopListenerDoesNotPanic(t *testing.T) {
	listener := auditevents.NewNoopListener()
	listener.OnMutation(context.Background(), auditevents.MutationEvent{
		WorkspaceID:   "ws-1",
		Action:        auditevents.ActionActivate,
		CorrelationID: "corr-1",
		Timestamp:     time.Now(),
	})
}

func TestRecordingListenerCapturesEvents(t *testing.T) {
	rec := &auditevents.RecordingListener{}
	ev := auditevents.MutationEvent{
		WorkspaceID:   "ws-1",
		Action:        auditevents.ActionRollback,
		CorrelationID: "corr-2",
		Timestamp:     time.Now(),
	}
	rec.OnMutation(context.Background(), ev)
	if len(rec.Events()) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rec.Events()))
	}
	if rec.Events()[0].Action != auditevents.ActionRollback {
		t.Fatal("event action mismatch")
	}
}

func TestRecordingListenerMultipleEvents(t *testing.T) {
	rec := &auditevents.RecordingListener{}
	for i := 0; i < 5; i++ {
		rec.OnMutation(context.Background(), auditevents.MutationEvent{
			WorkspaceID: "ws-1",
			Action:      auditevents.ActionActivate,
			Timestamp:   time.Now(),
		})
	}
	if len(rec.Events()) != 5 {
		t.Fatalf("expected 5 events, got %d", len(rec.Events()))
	}
}

func TestRecordingListenerReturnsCopy(t *testing.T) {
	rec := &auditevents.RecordingListener{}
	rec.OnMutation(context.Background(), auditevents.MutationEvent{
		WorkspaceID: "ws-1",
		Action:      auditevents.ActionActivate,
		Timestamp:   time.Now(),
	})
	events1 := rec.Events()
	events1[0].WorkspaceID = "modified"

	events2 := rec.Events()
	if events2[0].WorkspaceID != "ws-1" {
		t.Fatal("Events() should return a copy, not a reference")
	}
}
