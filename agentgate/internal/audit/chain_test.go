package audit_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/audit"
)

func TestCanonicalPayload_DeterministicOrdering(t *testing.T) {
	ts := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	rec1 := audit.DecisionRecord{
		WorkspaceID:       "ws-test",
		ExecutionID:       "exec-1",
		Timestamp:         ts,
		EventType:         audit.EventTypeDecision,
		Decision:          "ALLOW",
		Reason:            "policy_allow",
		PrincipalAgentID:  "agent-1",
		PrincipalRoles:    []string{"writer", "admin", "reader"},
		ToolBackendID:     "backend-sql",
		ToolName:          "query",
		ToolRisk:          "write",
		PolicyVersion:     "ver-1",
		PolicyHash:        "hash-1",
		RedactedArguments: map[string]string{"z": "last", "a": "first", "m": "middle"},
		PrevHash:          audit.GenesisHash,
	}

	rec2 := audit.DecisionRecord{
		WorkspaceID:       "ws-test",
		ExecutionID:       "exec-1",
		Timestamp:         ts,
		EventType:         audit.EventTypeDecision,
		Decision:          "ALLOW",
		Reason:            "policy_allow",
		PrincipalAgentID:  "agent-1",
		PrincipalRoles:    []string{"admin", "reader", "writer"}, // differently ordered roles
		ToolBackendID:     "backend-sql",
		ToolName:          "query",
		ToolRisk:          "write",
		PolicyVersion:     "ver-1",
		PolicyHash:        "hash-1",
		RedactedArguments: map[string]string{"m": "middle", "z": "last", "a": "first"}, // differently inserted map keys
		PrevHash:          audit.GenesisHash,
	}

	c1, err := audit.ComputeCanonicalPayload(rec1)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := audit.ComputeCanonicalPayload(rec2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(c1, c2) {
		t.Fatalf("canonical payloads must be identical despite map/slice input ordering:\nc1: %s\nc2: %s", c1, c2)
	}

	h1 := audit.ComputeRowHash(rec1.PrevHash, c1)
	h2 := audit.ComputeRowHash(rec2.PrevHash, c2)
	if h1 != h2 {
		t.Fatalf("row hashes must be identical:\nh1: %s\nh2: %s", h1, h2)
	}
}

func TestComputeRowHash_TamperChangesHash(t *testing.T) {
	ts := time.Now().UTC()
	rec := audit.DecisionRecord{
		WorkspaceID: "ws-1",
		ExecutionID: "ex-1",
		Timestamp:   ts,
		Decision:    "ALLOW",
		Reason:      "policy_allow",
	}

	c, _ := audit.ComputeCanonicalPayload(rec)
	hOrig := audit.ComputeRowHash(audit.GenesisHash, c)

	// Tampered reason
	rec.Reason = "tampered_reason"
	cTampered, _ := audit.ComputeCanonicalPayload(rec)
	hTampered := audit.ComputeRowHash(audit.GenesisHash, cTampered)

	if hOrig == hTampered {
		t.Fatal("altering canonical payload must alter row hash")
	}
}
