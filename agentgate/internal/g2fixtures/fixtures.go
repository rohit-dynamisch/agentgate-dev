// Package g2fixtures provides durable fixture definitions for G2
// cross-workstream coordination. These fixtures are shared between the
// Go Backend, Gateway/MCP, QA/Security, and Frontend/UI workstreams so
// all streams test the same semantic cases and produce comparable results.
//
// Fixtures are defined as typed Go values — not raw JSON — to ensure the
// Go compiler catches stale references when types evolve.
//
// The fixture set covers:
//   - Valid identity (with and without on_behalf_of)
//   - Missing identity (each failure class)
//   - Unknown tool
//   - Fingerprint drift
//   - Type mismatch argument
//   - Undeclared argument (not reaching policy)
//   - Null argument (rejected)
//   - Missing risk
//   - Valid allow context (reader + read-risk)
//   - Valid deny context (reader + write-risk → no matching policy)
package g2fixtures

import (
	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/contextassembly"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// ─── Fingerprints ─────────────────────────────────────────────────────────────

// ReadToolSchemaJSON is the canonical schema JSON for the "list_files" read tool.
// Used to compute consistent fingerprints across workstreams.
var ReadToolSchemaJSON = []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`)

// WriteToolSchemaJSON is the canonical schema JSON for the "write_file" write tool.
var WriteToolSchemaJSON = []byte(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}}}`)

// DriftedSchemaJSON represents a schema that has changed from the registered one.
// When used as a liveFingerprint against ReadToolSchemaJSON, it triggers DriftDetected.
var DriftedSchemaJSON = []byte(`{"type":"object","properties":{"path":{"type":"string"},"new_field":{"type":"integer"}}}`)

// ─── Claim maps (raw input to identity.Mapper) ────────────────────────────────

// ValidClaims is a set of gateway-validated claims that produce a valid identity.
var ValidClaims = map[string]string{
	"sub":   "agent-fixture-001",
	"roles": "reader",
}

// ValidClaimsWithDelegate includes an on_behalf_of claim.
var ValidClaimsWithDelegate = map[string]string{
	"sub":   "agent-fixture-001",
	"roles": "reader",
	"obo":   "user-fixture-001",
}

// MissingAgentIDClaims has no "sub" claim — triggers ErrMissingClaim.
var MissingAgentIDClaims = map[string]string{
	"roles": "reader",
}

// MalformedAgentIDClaims has a blank "sub" — triggers ErrMalformedClaim.
var MalformedAgentIDClaims = map[string]string{
	"sub":   "   ",
	"roles": "reader",
}

// EmptyRolesClaims has a blank roles string — triggers ErrMalformedClaim.
var EmptyRolesClaims = map[string]string{
	"sub":   "agent-fixture-001",
	"roles": "",
}

// AmbiguousClaims has on_behalf_of == agent_id — triggers ErrAmbiguousIdentity.
var AmbiguousClaims = map[string]string{
	"sub":   "agent-fixture-001",
	"roles": "reader",
	"obo":   "agent-fixture-001",
}

// DefaultMapperConfig is the standard fixture mapper config for use across workstreams.
var DefaultMapperConfig = identity.MapperConfig{
	AgentIDClaim:    "sub",
	RolesClaim:      "roles",
	OnBehalfOfClaim: "obo",
}

// ─── Mapped identities ────────────────────────────────────────────────────────

// ValidReaderIdentity is the mapped identity for a reader agent with no delegate.
var ValidReaderIdentity = identity.MappedIdentity{
	AgentID: "agent-fixture-001",
	Roles:   []string{"reader"},
}

// ValidAdminIdentity is the mapped identity for an admin agent.
var ValidAdminIdentity = identity.MappedIdentity{
	AgentID: "agent-fixture-002",
	Roles:   []string{"admin"},
}

// ValidDelegateIdentity is a reader acting on behalf of a user.
var ValidDelegateIdentity = identity.MappedIdentity{
	AgentID:    "agent-fixture-001",
	OnBehalfOf: "user-fixture-001",
	Roles:      []string{"reader"},
}

// ─── Tool IDs ─────────────────────────────────────────────────────────────────

// ReadToolID identifies the canonical "list_files" tool on "backend-fixture".
var ReadToolID = toolregistry.ToolID{
	BackendID: "backend-fixture",
	ToolName:  "list_files",
}

// WriteToolID identifies the canonical "write_file" tool on "backend-fixture".
var WriteToolID = toolregistry.ToolID{
	BackendID: "backend-fixture",
	ToolName:  "write_file",
}

// UnknownToolID is a tool not in the governance registry.
var UnknownToolID = toolregistry.ToolID{
	BackendID: "backend-fixture",
	ToolName:  "unregistered_op",
}

// SameNameDifferentBackendID is the same tool name as ReadToolID but on
// a different backend — must NOT inherit ReadToolID's classification.
var SameNameDifferentBackendID = toolregistry.ToolID{
	BackendID: "attacker-backend",
	ToolName:  "list_files",
}

// ─── Governance records ───────────────────────────────────────────────────────

// ValidReadGovernance is the authoritative governance record for ReadToolID.
// Callers must compute the fingerprint from ReadToolSchemaJSON to use this.
func ValidReadGovernance(fp toolregistry.SchemaFingerprint) toolregistry.GovernanceRecord {
	return toolregistry.GovernanceRecord{
		ToolID:                ReadToolID,
		Known:                 true,
		Risk:                  toolregistry.RiskRead,
		RegisteredFingerprint: fp,
		DriftStatus:           toolregistry.DriftNone,
	}
}

// ValidWriteGovernance is the authoritative governance record for WriteToolID.
func ValidWriteGovernance(fp toolregistry.SchemaFingerprint) toolregistry.GovernanceRecord {
	return toolregistry.GovernanceRecord{
		ToolID:                WriteToolID,
		Known:                 true,
		Risk:                  toolregistry.RiskWrite,
		RegisteredFingerprint: fp,
		DriftStatus:           toolregistry.DriftNone,
	}
}

// UnknownToolGovernance is the governance record for an unregistered tool.
var UnknownToolGovernance = toolregistry.GovernanceRecord{
	ToolID:      UnknownToolID,
	Known:       false,
	DriftStatus: toolregistry.DriftUnknownTool,
}

// DriftedGovernance represents a tool whose live schema no longer matches
// the registered fingerprint.
func DriftedGovernance(registeredFP toolregistry.SchemaFingerprint) toolregistry.GovernanceRecord {
	return toolregistry.GovernanceRecord{
		ToolID:                ReadToolID,
		Known:                 true,
		Risk:                  toolregistry.RiskRead,
		RegisteredFingerprint: registeredFP,
		DriftStatus:           toolregistry.DriftDetected,
	}
}

// MissingRiskGovernance is a Known tool with no risk (invalid).
func MissingRiskGovernance(fp toolregistry.SchemaFingerprint) toolregistry.GovernanceRecord {
	return toolregistry.GovernanceRecord{
		ToolID:                ReadToolID,
		Known:                 true,
		Risk:                  "",
		RegisteredFingerprint: fp,
		DriftStatus:           toolregistry.DriftNone,
	}
}

// ─── Argument fixtures ────────────────────────────────────────────────────────

// AmountDeclaration is the standard int64 "amount" declaration for write tools.
var AmountDeclaration = argdecl.Declaration{
	Name:     "amount",
	Type:     argdecl.ArgTypeInt,
	Required: true,
}

// ResolvedAmountArg is a successfully resolved amount=500.
var ResolvedAmountArg = map[string]argdecl.ResolvedArg{
	"amount": {Type: argdecl.ArgTypeInt, IntVal: 500},
}

// ─── Assembly inputs ──────────────────────────────────────────────────────────

// ValidAllowInput is a complete, valid allow context for the G1 fixture policy
// (reader + read-risk tool → ALLOW).
func ValidAllowInput(fp toolregistry.SchemaFingerprint) contextassembly.AssemblyInput {
	return contextassembly.AssemblyInput{
		ExecutionID:      "exec-fixture-allow",
		WorkspaceID:      "ws-fixture",
		MappedIdentity:   ValidReaderIdentity,
		GovernanceRecord: ValidReadGovernance(fp),
		ResolvedArgs:     nil,
	}
}

// ValidDenyInput is a complete, valid deny context (reader + write-risk →
// no matching policy → DENY with ReasonNoMatchingPolicy).
func ValidDenyInput(fp toolregistry.SchemaFingerprint) contextassembly.AssemblyInput {
	return contextassembly.AssemblyInput{
		ExecutionID:      "exec-fixture-deny",
		WorkspaceID:      "ws-fixture",
		MappedIdentity:   ValidReaderIdentity,
		GovernanceRecord: ValidWriteGovernance(fp),
		ResolvedArgs:     nil,
	}
}
