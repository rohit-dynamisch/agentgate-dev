package contextassembly_test

import (
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/contextassembly"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func validFP(t *testing.T) toolregistry.SchemaFingerprint {
	t.Helper()
	fp, err := toolregistry.FingerprintSchema([]byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	return fp
}

func validGovernance(t *testing.T) toolregistry.GovernanceRecord {
	t.Helper()
	fp := validFP(t)
	return toolregistry.GovernanceRecord{
		ToolID:                toolregistry.ToolID{BackendID: "backend-a", ToolName: "list_files"},
		Known:                 true,
		Risk:                  toolregistry.RiskRead,
		RegisteredFingerprint: fp,
		DriftStatus:           toolregistry.DriftNone,
	}
}

func validIdentity() identity.MappedIdentity {
	return identity.MappedIdentity{
		AgentID:    "agent-123",
		OnBehalfOf: "user-456",
		Roles:      []string{"reader"},
	}
}

func validInput(t *testing.T) contextassembly.AssemblyInput {
	t.Helper()
	return contextassembly.AssemblyInput{
		ExecutionID:      "exec-001",
		WorkspaceID:      "ws-001",
		MappedIdentity:   validIdentity(),
		GovernanceRecord: validGovernance(t),
		ResolvedArgs:     nil,
	}
}

// ─── Success path ─────────────────────────────────────────────────────────────

func TestAssemble_ValidInput(t *testing.T) {
	req, aerr := contextassembly.Assemble(validInput(t))
	if aerr != nil {
		t.Fatalf("expected success: %v", aerr)
	}
	if req.ExecutionID != "exec-001" {
		t.Errorf("ExecutionID: got %q", req.ExecutionID)
	}
	if req.WorkspaceID != "ws-001" {
		t.Errorf("WorkspaceID: got %q", req.WorkspaceID)
	}
	if req.Identity.AgentID != "agent-123" {
		t.Errorf("Identity.AgentID: got %q", req.Identity.AgentID)
	}
	if req.Identity.OnBehalfOf != "user-456" {
		t.Errorf("Identity.OnBehalfOf: got %q", req.Identity.OnBehalfOf)
	}
	if len(req.Identity.Roles) != 1 || req.Identity.Roles[0] != "reader" {
		t.Errorf("Identity.Roles: got %v", req.Identity.Roles)
	}
	if req.Tool.BackendID != "backend-a" {
		t.Errorf("Tool.BackendID: got %q", req.Tool.BackendID)
	}
	if req.Tool.Name != "list_files" {
		t.Errorf("Tool.Name: got %q", req.Tool.Name)
	}
	if !req.Classification.Known {
		t.Error("Classification.Known must be true")
	}
	if req.Classification.Risk != "read" {
		t.Errorf("Classification.Risk: got %q", req.Classification.Risk)
	}
}

func TestAssemble_WithResolvedArgs(t *testing.T) {
	in := validInput(t)
	in.ResolvedArgs = map[string]argdecl.ResolvedArg{
		"amount": {Type: argdecl.ArgTypeInt, IntVal: 500},
		"label":  {Type: argdecl.ArgTypeString, StrVal: "invoice"},
		"dry":    {Type: argdecl.ArgTypeBool, BoolVal: true},
	}
	req, aerr := contextassembly.Assemble(in)
	if aerr != nil {
		t.Fatalf("expected success: %v", aerr)
	}
	// Each resolved arg must appear in decision.Arguments.
	if len(req.Arguments) != 3 {
		t.Errorf("expected 3 arguments, got %d", len(req.Arguments))
	}
	// Verify the decision engine can evaluate it without malformed-request error.
	engine, err := decision.NewEngineWithoutPolicy(), error(nil)
	_ = err
	result := decision.NewEngineWithoutPolicy().Evaluate(req)
	// No policy → NoPolicyLoaded, but NOT MalformedRequest (args are valid).
	if result.Reason == decision.ReasonMalformedRequest {
		t.Errorf("assembled request should not be malformed, got: %s", result.Message)
	}
	_ = engine
}

func TestAssemble_RolesSliceIsCopied(t *testing.T) {
	// Mutations to the original slice must not affect the assembled request.
	in := validInput(t)
	original := in.MappedIdentity.Roles
	req, _ := contextassembly.Assemble(in)
	original[0] = "mutated"
	if req.Identity.Roles[0] == "mutated" {
		t.Error("roles slice was not copied; original mutation leaked into request")
	}
}

// ─── Missing ExecutionID / WorkspaceID ───────────────────────────────────────

func TestAssemble_MissingExecutionID(t *testing.T) {
	in := validInput(t)
	in.ExecutionID = ""
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for empty ExecutionID")
	}
}

func TestAssemble_MissingWorkspaceID(t *testing.T) {
	in := validInput(t)
	in.WorkspaceID = ""
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for empty WorkspaceID")
	}
}

// ─── Invalid identity ─────────────────────────────────────────────────────────

func TestAssemble_EmptyAgentID(t *testing.T) {
	in := validInput(t)
	in.MappedIdentity.AgentID = ""
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for empty AgentID")
	}
}

func TestAssemble_EmptyRoles(t *testing.T) {
	in := validInput(t)
	in.MappedIdentity.Roles = nil
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for empty Roles")
	}
}

// ─── Governance failures ──────────────────────────────────────────────────────

func TestAssemble_UnknownTool_FailsClosed(t *testing.T) {
	in := validInput(t)
	in.GovernanceRecord = toolregistry.GovernanceRecord{
		ToolID:      toolregistry.ToolID{BackendID: "b", ToolName: "x"},
		Known:       false,
		DriftStatus: toolregistry.DriftUnknownTool,
	}
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestAssemble_DriftedTool_FailsClosed(t *testing.T) {
	in := validInput(t)
	gov := validGovernance(t)
	gov.DriftStatus = toolregistry.DriftDetected
	in.GovernanceRecord = gov
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for drifted tool")
	}
}

func TestAssemble_MissingRisk_FailsClosed(t *testing.T) {
	in := validInput(t)
	gov := validGovernance(t)
	gov.Risk = ""
	in.GovernanceRecord = gov
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for missing risk")
	}
}

func TestAssemble_InvalidRisk_FailsClosed(t *testing.T) {
	in := validInput(t)
	gov := validGovernance(t)
	gov.Risk = "nuclear"
	in.GovernanceRecord = gov
	_, aerr := contextassembly.Assemble(in)
	if aerr == nil {
		t.Fatal("expected error for unrecognized risk")
	}
}

// ─── No extra fields invented ─────────────────────────────────────────────────

func TestAssemble_NilArgs_ProducesEmptyArguments(t *testing.T) {
	in := validInput(t)
	in.ResolvedArgs = nil
	req, aerr := contextassembly.Assemble(in)
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	// Arguments map should be present but empty (not nil panic risk).
	if req.Arguments == nil {
		// nil is fine for decision.Engine — no args declared
	}
	if len(req.Arguments) != 0 {
		t.Errorf("expected 0 arguments for nil ResolvedArgs, got %d", len(req.Arguments))
	}
}

// ─── End-to-end: assembled request feeds the frozen G1 decision engine ────────

func TestAssemble_FeedsDecisionEngine(t *testing.T) {
	// This test verifies the assembled request is accepted by the frozen
	// G1 decision engine without modification — the key integration point.
	in := validInput(t)
	req, aerr := contextassembly.Assemble(in)
	if aerr != nil {
		t.Fatalf("assemble: %v", aerr)
	}

	// Load the G1 fixture policy to get a real engine.
	const fixturePolicy = `
permit(
  principal in AgentGate::Role::"reader",
  action == AgentGate::Action::"InvokeTool",
  resource
) when {
  resource.risk == "read"
};`

	engine, err := decision.NewEngine([]byte(fixturePolicy))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	result := engine.Evaluate(req)
	if result.Decision != decision.Allow {
		t.Errorf("expected ALLOW for reader+read-risk tool, got %s: %s", result.Decision, result.Message)
	}
}
