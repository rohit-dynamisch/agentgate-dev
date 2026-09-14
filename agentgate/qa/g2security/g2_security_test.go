package g2security_test

import (
	"encoding/json"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/contextassembly"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/fixturepolicy"
	"github.com/Dynamisch-LLC/agentgate/internal/g2fixtures"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// =============================================================================
// G2-QA-01: Authoritative Test Matrix
// =============================================================================

func TestAuthoritativeMatrix_Identity(t *testing.T) {
	mapper, err := identity.NewMapper(g2fixtures.DefaultMapperConfig)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	matrix := []struct {
		name          string
		claims        map[string]string
		wantErr       bool
		expectedClass identity.FailureClass
	}{
		{
			name:    "valid minimal",
			claims:  g2fixtures.ValidClaims,
			wantErr: false,
		},
		{
			name:    "valid with delegate",
			claims:  g2fixtures.ValidClaimsWithDelegate,
			wantErr: false,
		},
		{
			name:          "missing agent ID",
			claims:        g2fixtures.MissingAgentIDClaims,
			wantErr:       true,
			expectedClass: identity.ErrMissingClaim,
		},
		{
			name:          "malformed agent ID (blank spaces)",
			claims:        g2fixtures.MalformedAgentIDClaims,
			wantErr:       true,
			expectedClass: identity.ErrMalformedClaim,
		},
		{
			name:          "empty roles",
			claims:        g2fixtures.EmptyRolesClaims,
			wantErr:       true,
			expectedClass: identity.ErrMalformedClaim,
		},
		{
			name:          "ambiguous identity (obo == agent_id)",
			claims:        g2fixtures.AmbiguousClaims,
			wantErr:       true,
			expectedClass: identity.ErrAmbiguousIdentity,
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			id, merr := mapper.Map(tc.claims)
			if tc.wantErr {
				if merr == nil {
					t.Fatalf("expected error, got success: %+v", id)
				}
				if merr.Class != tc.expectedClass {
					t.Errorf("got failure class %q, want %q", merr.Class, tc.expectedClass)
				}
			} else {
				if merr != nil {
					t.Fatalf("expected success, got error: %v", merr)
				}
				if id.AgentID == "" {
					t.Error("expected non-empty AgentID")
				}
				if len(id.Roles) == 0 {
					t.Error("expected non-empty Roles")
				}
			}
		})
	}
}

// =============================================================================
// G2-QA-02: Identity Abuse Tests
// =============================================================================

func TestIdentityAbuse_NoDefaultIdentityFallback(t *testing.T) {
	mapper, _ := identity.NewMapper(g2fixtures.DefaultMapperConfig)

	// An empty claim map must never default to an anonymous or guest identity
	emptyClaims := map[string]string{}
	_, merr := mapper.Map(emptyClaims)
	if merr == nil {
		t.Fatal("security bypass: empty claims mapped to default identity instead of failing closed")
	}
	if merr.Class != identity.ErrMissingClaim {
		t.Fatalf("expected ErrMissingClaim, got %v", merr.Class)
	}
}

func TestIdentityAbuse_UntrustedClaimsCannotOverride(t *testing.T) {
	// Attacker tries to inject admin role via non-configured claim "super_roles"
	mapper, _ := identity.NewMapper(g2fixtures.DefaultMapperConfig) // config uses "roles"

	claims := map[string]string{
		"sub":         "agent-001",
		"roles":       "reader",
		"super_roles": "admin superuser root",
	}

	id, merr := mapper.Map(claims)
	if merr != nil {
		t.Fatalf("unexpected error: %v", merr)
	}
	for _, r := range id.Roles {
		if r == "admin" || r == "superuser" || r == "root" {
			t.Fatalf("security violation: unconfigured claim injected into roles: %v", id.Roles)
		}
	}
}

func TestIdentityAbuse_ConflictingDelegateFailsClosed(t *testing.T) {
	mapper, _ := identity.NewMapper(g2fixtures.DefaultMapperConfig)

	// Agent claims to be acting on behalf of itself to bypass delegation audit
	claims := map[string]string{
		"sub":   "agent-x",
		"roles": "reader",
		"obo":   "agent-x",
	}

	_, merr := mapper.Map(claims)
	if merr == nil {
		t.Fatal("expected ErrAmbiguousIdentity when obo == agent_id, got nil")
	}
	if merr.Class != identity.ErrAmbiguousIdentity {
		t.Fatalf("expected ErrAmbiguousIdentity, got %v", merr.Class)
	}
}

// =============================================================================
// G2-QA-03: Tool Governance Abuse Tests
// =============================================================================

func TestToolGovernanceAbuse_UnknownToolsNeverInheritClassification(t *testing.T) {
	readFP, err := toolregistry.FingerprintSchema(g2fixtures.ReadToolSchemaJSON)
	if err != nil {
		t.Fatalf("fingerprint error: %v", err)
	}

	// Register ReadToolID on "backend-fixture"
	reg, err := toolregistry.NewRegistry([]toolregistry.RegistryEntry{
		{
			ToolID:                g2fixtures.ReadToolID,
			Risk:                  toolregistry.RiskRead,
			RegisteredFingerprint: readFP,
		},
	})
	if err != nil {
		t.Fatalf("registry setup: %v", err)
	}

	// Attacker attempts to invoke a same-named tool on "attacker-backend"
	gov := reg.Lookup(g2fixtures.SameNameDifferentBackendID, readFP)
	if gov.Known {
		t.Fatal("security violation: attacker backend inherited known status from trusted backend")
	}
	if err := gov.Validate(); err == nil {
		t.Fatal("security violation: unvalidated tool on untrusted backend passed validation")
	}
}

func TestToolGovernanceAbuse_SchemaDriftFailsClosed(t *testing.T) {
	readFP, _ := toolregistry.FingerprintSchema(g2fixtures.ReadToolSchemaJSON)
	reg, _ := toolregistry.NewRegistry([]toolregistry.RegistryEntry{
		{
			ToolID:                g2fixtures.ReadToolID,
			Risk:                  toolregistry.RiskRead,
			RegisteredFingerprint: readFP,
		},
	})

	// Live schema has drifted
	driftedFP, _ := toolregistry.FingerprintSchema(g2fixtures.DriftedSchemaJSON)
	gov := reg.Lookup(g2fixtures.ReadToolID, driftedFP)

	if gov.DriftStatus != toolregistry.DriftDetected {
		t.Fatalf("expected DriftDetected, got %v", gov.DriftStatus)
	}
	if err := gov.Validate(); err == nil {
		t.Fatal("security violation: drifted tool passed governance validation")
	}
}

func TestToolGovernanceAbuse_MalformedSchemaCannotFingerprint(t *testing.T) {
	malformedJSON := []byte(`{"properties": { unclosed...`)
	_, err := toolregistry.FingerprintSchema(malformedJSON)
	if err == nil {
		t.Fatal("security violation: malformed schema received a valid fingerprint")
	}
}

// =============================================================================
// G2-QA-04: Argument Authorization Tests
// =============================================================================

func TestArgumentAuthorization_UndeclaredArgsNeverReachPolicy(t *testing.T) {
	decls, err := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{
			Name:     "path",
			Type:     argdecl.ArgTypeString,
			Required: true,
		},
	})
	if err != nil {
		t.Fatalf("argdecl setup: %v", err)
	}

	// Caller sends declared "path" plus undeclared malicious parameters
	raw := map[string]argdecl.RawArgValue{
		"path":           json.RawMessage(`"/var/log/app.log"`),
		"override_admin": json.RawMessage(`"true"`),
		"bypass_policy":  json.RawMessage(`12345`),
	}

	resolved, rerr := decls.Resolve(raw)
	if rerr != nil {
		t.Fatalf("unexpected error: %v", rerr)
	}

	if _, exists := resolved["override_admin"]; exists {
		t.Fatal("security violation: undeclared argument 'override_admin' leaked into resolved map")
	}
	if _, exists := resolved["bypass_policy"]; exists {
		t.Fatal("security violation: undeclared argument 'bypass_policy' leaked into resolved map")
	}
}

func TestArgumentAuthorization_ExplicitNullRejectedForRequiredAndOptional(t *testing.T) {
	decls, _ := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{Name: "req_arg", Type: argdecl.ArgTypeString, Required: true},
		{Name: "opt_arg", Type: argdecl.ArgTypeInt, Required: false},
	})

	// Required is null
	rawReqNull := map[string]argdecl.RawArgValue{
		"req_arg": json.RawMessage(`null`),
		"opt_arg": json.RawMessage(`42`),
	}
	_, errReq := decls.Resolve(rawReqNull)
	if errReq == nil {
		t.Fatal("security violation: explicit null for required argument was accepted")
	}

	// Optional is null
	rawOptNull := map[string]argdecl.RawArgValue{
		"req_arg": json.RawMessage(`"valid"`),
		"opt_arg": json.RawMessage(`null`),
	}
	_, errOpt := decls.Resolve(rawOptNull)
	if errOpt == nil {
		t.Fatal("security violation: explicit null for optional argument was accepted (null is not absent)")
	}
}

func TestArgumentAuthorization_TypeMismatchFailsClosed(t *testing.T) {
	decls, _ := argdecl.NewDeclarationSet([]argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt, Required: true},
	})

	// String passed instead of int64
	rawString := map[string]argdecl.RawArgValue{
		"amount": json.RawMessage(`"500"`),
	}
	_, err := decls.Resolve(rawString)
	if err == nil {
		t.Fatal("security violation: string was coerced/accepted for int64 declaration")
	}
}

// =============================================================================
// G2-QA-05: Regression & Determinism Tests
// =============================================================================

func TestDeterminism_SchemaFingerprintCanonicalization(t *testing.T) {
	// Schemas differing only in whitespace, key order, and indentation
	schema1 := []byte(`{"type":"object","properties":{"path":{"type":"string"},"limit":{"type":"integer"}}}`)
	schema2 := []byte(`{
		"properties": {
			"limit": {
				"type": "integer"
			},
			"path": {
				"type": "string"
			}
		},
		"type": "object"
	}`)

	fp1, err1 := toolregistry.FingerprintSchema(schema1)
	if err1 != nil {
		t.Fatalf("fp1: %v", err1)
	}

	fp2, err2 := toolregistry.FingerprintSchema(schema2)
	if err2 != nil {
		t.Fatalf("fp2: %v", err2)
	}

	if fp1 != fp2 {
		t.Fatalf("canonicalization non-determinism: %q != %q", fp1, fp2)
	}
}

func TestRegression_G1EngineWithG2Assembly(t *testing.T) {
	// Test full end-to-end integration: Assemble -> Evaluate with G1 engine
	engine, err := decision.NewEngine([]byte(fixturepolicy.CedarSource))
	if err != nil {
		t.Fatalf("engine setup: %v", err)
	}

	fp, _ := toolregistry.FingerprintSchema(g2fixtures.ReadToolSchemaJSON)
	input := g2fixtures.ValidAllowInput(fp)

	req, aerr := contextassembly.Assemble(input)
	if aerr != nil {
		t.Fatalf("assemble: %v", aerr)
	}

	res := engine.Evaluate(req)
	if res.Decision != decision.Allow {
		t.Fatalf("expected ALLOW, got %v (reason: %v)", res.Decision, res.Reason)
	}

	// Now try deny context
	denyInput := g2fixtures.ValidDenyInput(fp)
	denyReq, aerr := contextassembly.Assemble(denyInput)
	if aerr != nil {
		t.Fatalf("assemble deny: %v", aerr)
	}

	denyRes := engine.Evaluate(denyReq)
	if denyRes.Decision != decision.Deny {
		t.Fatalf("expected DENY, got %v", denyRes.Decision)
	}
	if denyRes.Reason != decision.ReasonNoMatchingPolicy {
		t.Fatalf("expected ReasonNoMatchingPolicy, got %v", denyRes.Reason)
	}
}

// =============================================================================
// G2-QA-06: Trust Boundary Review
// =============================================================================

func TestTrustBoundary_ContextAssemblyRejectsUnvalidatedGovernance(t *testing.T) {
	// Attempt to assemble request with an invalid/drifted governance record
	fp, _ := toolregistry.FingerprintSchema(g2fixtures.ReadToolSchemaJSON)
	input := g2fixtures.ValidAllowInput(fp)

	// Corrupt the governance record to drifted
	input.GovernanceRecord.DriftStatus = toolregistry.DriftDetected

	_, err := contextassembly.Assemble(input)
	if err == nil {
		t.Fatal("trust boundary violation: Assemble accepted drifted governance record")
	}
}

func TestTrustBoundary_ContextAssemblyRejectsEmptyIdentity(t *testing.T) {
	fp, _ := toolregistry.FingerprintSchema(g2fixtures.ReadToolSchemaJSON)
	input := g2fixtures.ValidAllowInput(fp)

	// Blank agent ID
	input.MappedIdentity.AgentID = ""

	_, err := contextassembly.Assemble(input)
	if err == nil {
		t.Fatal("trust boundary violation: Assemble accepted empty AgentID")
	}
}
