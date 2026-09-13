package identity_test

import (
	"errors"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/identity"
)

// ─── MapperConfig.Validate ───────────────────────────────────────────────────

func TestMapperConfigValidate(t *testing.T) {
	t.Run("valid full config", func(t *testing.T) {
		cfg := identity.MapperConfig{
			AgentIDClaim: "sub",
			RolesClaim:   "roles",
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid config: %v", err)
		}
	})
	t.Run("valid with optional OnBehalfOf", func(t *testing.T) {
		cfg := identity.MapperConfig{
			AgentIDClaim:    "sub",
			RolesClaim:      "roles",
			OnBehalfOfClaim: "obo",
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid config: %v", err)
		}
	})
	t.Run("missing AgentIDClaim", func(t *testing.T) {
		cfg := identity.MapperConfig{RolesClaim: "roles"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for missing AgentIDClaim")
		}
	})
	t.Run("blank AgentIDClaim", func(t *testing.T) {
		cfg := identity.MapperConfig{AgentIDClaim: "   ", RolesClaim: "roles"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for blank AgentIDClaim")
		}
	})
	t.Run("missing RolesClaim", func(t *testing.T) {
		cfg := identity.MapperConfig{AgentIDClaim: "sub"}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for missing RolesClaim")
		}
	})
}

// ─── NewMapper ───────────────────────────────────────────────────────────────

func TestNewMapper_InvalidConfig(t *testing.T) {
	_, err := identity.NewMapper(identity.MapperConfig{})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

// ─── Mapper.Map — success paths ──────────────────────────────────────────────

func validMapper(t *testing.T) *identity.Mapper {
	t.Helper()
	m, err := identity.NewMapper(identity.MapperConfig{
		AgentIDClaim:    "sub",
		RolesClaim:      "roles",
		OnBehalfOfClaim: "obo",
	})
	if err != nil {
		t.Fatalf("NewMapper: %v", err)
	}
	return m
}

func TestMap_ValidIdentity(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-123",
		"roles": "reader admin",
		"obo":   "user-456",
	}
	id, merr := m.Map(claims)
	if merr != nil {
		t.Fatalf("expected success, got: %v", merr)
	}
	if id.AgentID != "agent-123" {
		t.Errorf("AgentID: got %q, want %q", id.AgentID, "agent-123")
	}
	if id.OnBehalfOf != "user-456" {
		t.Errorf("OnBehalfOf: got %q, want %q", id.OnBehalfOf, "user-456")
	}
	if len(id.Roles) != 2 {
		t.Errorf("Roles length: got %d, want 2", len(id.Roles))
	}
}

func TestMap_NoOnBehalfOf_Optional(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-123",
		"roles": "reader",
		// "obo" absent
	}
	id, merr := m.Map(claims)
	if merr != nil {
		t.Fatalf("expected success: %v", merr)
	}
	if id.OnBehalfOf != "" {
		t.Errorf("OnBehalfOf should be empty when absent, got %q", id.OnBehalfOf)
	}
}

func TestMap_WithoutOptionalOnBehalfOfConfig(t *testing.T) {
	// When OnBehalfOfClaim is not configured, it is never read.
	m, err := identity.NewMapper(identity.MapperConfig{
		AgentIDClaim: "sub",
		RolesClaim:   "roles",
		// no OnBehalfOfClaim
	})
	if err != nil {
		t.Fatalf("NewMapper: %v", err)
	}
	claims := map[string]string{
		"sub":   "agent-x",
		"roles": "admin",
	}
	id, merr := m.Map(claims)
	if merr != nil {
		t.Fatalf("expected success: %v", merr)
	}
	if id.OnBehalfOf != "" {
		t.Errorf("expected empty OnBehalfOf, got %q", id.OnBehalfOf)
	}
}

func TestMap_DuplicateRolesDeduped(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-1",
		"roles": "reader reader admin reader",
	}
	id, merr := m.Map(claims)
	if merr != nil {
		t.Fatalf("expected success: %v", merr)
	}
	if len(id.Roles) != 2 {
		t.Errorf("expected 2 deduplicated roles, got %d: %v", len(id.Roles), id.Roles)
	}
}

func TestMap_RolesWithExtraWhitespace(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-1",
		"roles": "  reader   admin  ",
	}
	id, merr := m.Map(claims)
	if merr != nil {
		t.Fatalf("expected success: %v", merr)
	}
	if len(id.Roles) != 2 {
		t.Errorf("expected 2 roles after trim, got %d", len(id.Roles))
	}
}

// ─── Failure class: ErrMissingClaim ──────────────────────────────────────────

func TestMap_MissingAgentID(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		// "sub" absent
		"roles": "reader",
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for missing agent id claim")
	}
	if merr.Class != identity.ErrMissingClaim {
		t.Errorf("got class %q, want ErrMissingClaim", merr.Class)
	}
}

func TestMap_MissingRolesClaim(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub": "agent-1",
		// "roles" absent
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for missing roles claim")
	}
	if merr.Class != identity.ErrMissingClaim {
		t.Errorf("got class %q, want ErrMissingClaim", merr.Class)
	}
}

// ─── Failure class: ErrMalformedClaim ────────────────────────────────────────

func TestMap_BlankAgentID(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "   ",
		"roles": "reader",
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for blank agent id")
	}
	if merr.Class != identity.ErrMalformedClaim {
		t.Errorf("got class %q, want ErrMalformedClaim", merr.Class)
	}
}

func TestMap_BlankOnBehalfOf_WhenPresent(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-1",
		"roles": "reader",
		"obo":   "   ", // present but blank
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for present-but-blank on_behalf_of")
	}
	if merr.Class != identity.ErrMalformedClaim {
		t.Errorf("got class %q, want ErrMalformedClaim", merr.Class)
	}
}

func TestMap_BlankRolesString(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-1",
		"roles": "   ",
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for blank roles claim")
	}
	if merr.Class != identity.ErrMalformedClaim {
		t.Errorf("got class %q, want ErrMalformedClaim", merr.Class)
	}
}

// ─── Failure class: ErrMissingRoles ──────────────────────────────────────────

func TestMap_RolesClaimAllBlankParts(t *testing.T) {
	m := validMapper(t)
	// roles present, splits to only blank parts
	claims := map[string]string{
		"sub":   "agent-1",
		"roles": " ",
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for all-blank role parts")
	}
	// A single-space string is blank → ErrMalformedClaim (blank claim value)
	if merr.Class != identity.ErrMalformedClaim && merr.Class != identity.ErrMissingRoles {
		t.Errorf("got class %q, want ErrMalformedClaim or ErrMissingRoles", merr.Class)
	}
}

// ─── Failure class: ErrAmbiguousIdentity ─────────────────────────────────────

func TestMap_OnBehalfOf_EqualToAgentID(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{
		"sub":   "agent-1",
		"roles": "reader",
		"obo":   "agent-1", // same as agent id
	}
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected MappingError for on_behalf_of == agent_id")
	}
	if merr.Class != identity.ErrAmbiguousIdentity {
		t.Errorf("got class %q, want ErrAmbiguousIdentity", merr.Class)
	}
}

// ─── errors.Is via AsClass ────────────────────────────────────────────────────

func TestMappingError_Is(t *testing.T) {
	m := validMapper(t)
	claims := map[string]string{"roles": "reader"} // missing "sub"
	_, merr := m.Map(claims)
	if merr == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(merr, identity.AsClass(identity.ErrMissingClaim)) {
		t.Errorf("errors.Is should match ErrMissingClaim sentinel")
	}
	if errors.Is(merr, identity.AsClass(identity.ErrMalformedClaim)) {
		t.Errorf("errors.Is should NOT match ErrMalformedClaim sentinel")
	}
}

// ─── No hardcoded claim names ─────────────────────────────────────────────────

func TestMap_CustomClaimNames(t *testing.T) {
	// Verify that non-standard claim key names work — no hardcoded "sub"/etc.
	m, err := identity.NewMapper(identity.MapperConfig{
		AgentIDClaim:    "x-agent-id",
		RolesClaim:      "x-permissions",
		OnBehalfOfClaim: "x-delegate",
	})
	if err != nil {
		t.Fatalf("NewMapper: %v", err)
	}
	claims := map[string]string{
		"x-agent-id":    "my-agent",
		"x-permissions": "admin",
		"x-delegate":    "human-user",
	}
	id, merr := m.Map(claims)
	if merr != nil {
		t.Fatalf("Map: %v", merr)
	}
	if id.AgentID != "my-agent" {
		t.Errorf("AgentID: got %q", id.AgentID)
	}
	if id.OnBehalfOf != "human-user" {
		t.Errorf("OnBehalfOf: got %q", id.OnBehalfOf)
	}
	if len(id.Roles) != 1 || id.Roles[0] != "admin" {
		t.Errorf("Roles: got %v", id.Roles)
	}
}
