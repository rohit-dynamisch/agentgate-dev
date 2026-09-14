package config_test

import (
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// G2-DO-04: Negative security configuration validation tests.
// Verifies that invalid security configurations fail closed at validation time
// and can never become permissive or undefined.

func TestNegativeConfig_MissingAgentIDClaim(t *testing.T) {
	cfg := identity.MapperConfig{
		AgentIDClaim: "", // Missing/empty
		RolesClaim:   "roles",
	}
	_, err := identity.NewMapper(cfg)
	if err == nil {
		t.Fatal("expected error for empty AgentIDClaim, got nil (must fail closed)")
	}
}

func TestNegativeConfig_MissingRolesClaim(t *testing.T) {
	cfg := identity.MapperConfig{
		AgentIDClaim: "sub",
		RolesClaim:   "", // Missing/empty
	}
	_, err := identity.NewMapper(cfg)
	if err == nil {
		t.Fatal("expected error for empty RolesClaim, got nil (must fail closed)")
	}
}

func TestNegativeConfig_ConflictingClaims(t *testing.T) {
	cfg := identity.MapperConfig{
		AgentIDClaim:    "sub",
		RolesClaim:      "roles",
		OnBehalfOfClaim: "sub", // Conflict: same claim for agent and on_behalf_of
	}
	m, err := identity.NewMapper(cfg)
	if err != nil {
		return // rejected at config time is valid fail-closed
	}
	claims := map[string]string{
		"sub":   "agent-001",
		"roles": "reader",
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected error for conflicting AgentIDClaim and OnBehalfOfClaim, got nil (must fail closed)")
	}
	if merr.Class != identity.ErrAmbiguousIdentity {
		t.Fatalf("expected ErrAmbiguousIdentity, got %v", merr.Class)
	}
}

func TestNegativeConfig_MissingToolRisk(t *testing.T) {
	entries := []toolregistry.RegistryEntry{
		{
			ToolID: toolregistry.ToolID{
				BackendID: "backend-fixture",
				ToolName:  "list_files",
			},
			Risk:                  "", // Missing risk fails closed
			RegisteredFingerprint: "53582d9d7229826e75097ac478787ca403c94ce74e4611571be1b03c706952cc",
		},
	}
	_, err := toolregistry.NewRegistry(entries)
	if err == nil {
		t.Fatal("expected error registering tool with empty risk, got nil (must fail closed)")
	}
}

func TestNegativeConfig_EmptyFingerprint(t *testing.T) {
	entries := []toolregistry.RegistryEntry{
		{
			ToolID: toolregistry.ToolID{
				BackendID: "backend-fixture",
				ToolName:  "list_files",
			},
			Risk:                  toolregistry.RiskRead,
			RegisteredFingerprint: "", // Empty fingerprint fails closed
		},
	}
	_, err := toolregistry.NewRegistry(entries)
	if err == nil {
		t.Fatal("expected error registering tool with empty fingerprint, got nil (must fail closed)")
	}
}

func TestNegativeConfig_MalformedSchemaCannotFingerprint(t *testing.T) {
	malformed := []byte(`{"type": "object", "properties": { incomplete...`)
	_, err := toolregistry.FingerprintSchema(malformed)
	if err == nil {
		t.Fatal("expected error fingerprinting malformed schema JSON, got nil (must fail closed)")
	}
}

func TestNegativeConfig_InvalidArgType(t *testing.T) {
	decls := []argdecl.Declaration{
		{
			Name:     "threshold",
			Type:     argdecl.ArgType("float"), // Invalid type
			Required: true,
		},
	}
	_, err := argdecl.NewDeclarationSet(decls)
	if err == nil {
		t.Fatal("expected error for unsupported ArgType, got nil (must fail closed)")
	}
}

func TestNegativeConfig_DuplicateArgDeclarations(t *testing.T) {
	decls := []argdecl.Declaration{
		{
			Name:     "path",
			Type:     argdecl.ArgTypeString,
			Required: true,
		},
		{
			Name:     "path", // Duplicate
			Type:     argdecl.ArgTypeString,
			Required: false,
		},
	}
	_, err := argdecl.NewDeclarationSet(decls)
	if err == nil {
		t.Fatal("expected error for duplicate argument declaration, got nil (must fail closed)")
	}
}
