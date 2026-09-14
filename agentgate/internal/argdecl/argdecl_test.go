package argdecl_test

import (
	"encoding/json"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
)

// ─── NewDeclarationSet ────────────────────────────────────────────────────────

func TestNewDeclarationSet_Valid(t *testing.T) {
	decls := []argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt, Required: true},
		{Name: "label", Type: argdecl.ArgTypeString, Required: false},
		{Name: "dry_run", Type: argdecl.ArgTypeBool, Required: false},
	}
	ds, err := argdecl.NewDeclarationSet(decls)
	if err != nil {
		t.Fatalf("expected valid set: %v", err)
	}
	if _, ok := ds.Get("amount"); !ok {
		t.Error("expected 'amount' in declaration set")
	}
}

func TestNewDeclarationSet_DuplicateName(t *testing.T) {
	decls := []argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt},
		{Name: "amount", Type: argdecl.ArgTypeString}, // duplicate
	}
	_, err := argdecl.NewDeclarationSet(decls)
	if err == nil {
		t.Fatal("expected error for duplicate declaration name")
	}
}

func TestNewDeclarationSet_EmptyName(t *testing.T) {
	decls := []argdecl.Declaration{
		{Name: "", Type: argdecl.ArgTypeInt},
	}
	_, err := argdecl.NewDeclarationSet(decls)
	if err == nil {
		t.Fatal("expected error for empty argument name")
	}
}

func TestNewDeclarationSet_UnrecognizedType(t *testing.T) {
	decls := []argdecl.Declaration{
		{Name: "x", Type: argdecl.ArgType("float64")},
	}
	_, err := argdecl.NewDeclarationSet(decls)
	if err == nil {
		t.Fatal("expected error for unrecognized type")
	}
}

func TestNewDeclarationSet_Empty(t *testing.T) {
	// An empty declaration set is valid (no args declared for this tool).
	ds, err := argdecl.NewDeclarationSet(nil)
	if err != nil {
		t.Fatalf("empty set should be valid: %v", err)
	}
	if len(ds.Names()) != 0 {
		t.Error("expected 0 names in empty set")
	}
}

// ─── Resolve — success ────────────────────────────────────────────────────────

func rawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("rawJSON: %v", err)
	}
	return b
}

func makeDS(t *testing.T, decls []argdecl.Declaration) *argdecl.DeclarationSet {
	t.Helper()
	ds, err := argdecl.NewDeclarationSet(decls)
	if err != nil {
		t.Fatalf("NewDeclarationSet: %v", err)
	}
	return ds
}

func TestResolve_AllTypesPresent(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt, Required: true},
		{Name: "label", Type: argdecl.ArgTypeString, Required: true},
		{Name: "dry_run", Type: argdecl.ArgTypeBool, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{
		"amount":  rawJSON(t, int64(42)),
		"label":   rawJSON(t, "invoice-7"),
		"dry_run": rawJSON(t, false),
	}
	result, rerr := ds.Resolve(raw)
	if rerr != nil {
		t.Fatalf("expected success: %v", rerr)
	}
	if result["amount"].IntVal != 42 {
		t.Errorf("amount: got %d, want 42", result["amount"].IntVal)
	}
	if result["label"].StrVal != "invoice-7" {
		t.Errorf("label: got %q", result["label"].StrVal)
	}
	if result["dry_run"].BoolVal != false {
		t.Errorf("dry_run: got %v", result["dry_run"].BoolVal)
	}
}

func TestResolve_OptionalAbsent_OmittedFromResult(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "required_arg", Type: argdecl.ArgTypeString, Required: true},
		{Name: "optional_arg", Type: argdecl.ArgTypeInt, Required: false},
	})
	raw := map[string]argdecl.RawArgValue{
		"required_arg": rawJSON(t, "hello"),
		// optional_arg absent
	}
	result, rerr := ds.Resolve(raw)
	if rerr != nil {
		t.Fatalf("expected success: %v", rerr)
	}
	if _, ok := result["optional_arg"]; ok {
		t.Error("absent optional arg should not appear in result")
	}
}

func TestResolve_UndeclaredArgIgnored(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "declared", Type: argdecl.ArgTypeString, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{
		"declared":   rawJSON(t, "value"),
		"undeclared": rawJSON(t, "should be ignored"),
		"another":    rawJSON(t, 999),
	}
	result, rerr := ds.Resolve(raw)
	if rerr != nil {
		t.Fatalf("expected success: %v", rerr)
	}
	if _, ok := result["undeclared"]; ok {
		t.Error("undeclared arg must NOT appear in result")
	}
	if _, ok := result["another"]; ok {
		t.Error("undeclared arg must NOT appear in result")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 resolved arg, got %d", len(result))
	}
}

// ─── Resolve — null rejection (G1 bug fix regression) ────────────────────────

func TestResolve_NullRequired_Rejected(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{
		"amount": []byte("null"),
	}
	_, rerr := ds.Resolve(raw)
	if rerr == nil {
		t.Fatal("explicit null must be rejected for required arg")
	}
	if rerr.ArgName != "amount" {
		t.Errorf("expected error on 'amount', got %q", rerr.ArgName)
	}
}

func TestResolve_NullOptional_Rejected(t *testing.T) {
	// Null is never coerced to zero value, even for optional args.
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "label", Type: argdecl.ArgTypeString, Required: false},
	})
	raw := map[string]argdecl.RawArgValue{
		"label": []byte("null"),
	}
	_, rerr := ds.Resolve(raw)
	if rerr == nil {
		t.Fatal("explicit null must be rejected even for optional arg")
	}
}

// ─── Resolve — missing required ──────────────────────────────────────────────

func TestResolve_MissingRequired_Rejected(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{} // amount absent
	_, rerr := ds.Resolve(raw)
	if rerr == nil {
		t.Fatal("missing required arg must be rejected")
	}
	if rerr.ArgName != "amount" {
		t.Errorf("expected error on 'amount', got %q", rerr.ArgName)
	}
}

// Distinguish missing optional (absent) from explicit null.
func TestResolve_MissingOptional_vs_ExplicitNull(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "tag", Type: argdecl.ArgTypeString, Required: false},
	})

	// Missing optional — should succeed with tag absent from result.
	result, rerr := ds.Resolve(map[string]argdecl.RawArgValue{})
	if rerr != nil {
		t.Fatalf("missing optional should succeed: %v", rerr)
	}
	if _, ok := result["tag"]; ok {
		t.Error("absent optional must not be in result")
	}

	// Explicit null — must fail.
	_, rerr = ds.Resolve(map[string]argdecl.RawArgValue{"tag": []byte("null")})
	if rerr == nil {
		t.Fatal("explicit null must be rejected")
	}
}

// ─── Resolve — type mismatch ─────────────────────────────────────────────────

func TestResolve_TypeMismatch_String_GetsInt(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "label", Type: argdecl.ArgTypeString, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{
		"label": rawJSON(t, 42), // int, not string
	}
	_, rerr := ds.Resolve(raw)
	if rerr == nil {
		t.Fatal("type mismatch must be rejected")
	}
}

func TestResolve_TypeMismatch_Int_GetsBool(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "amount", Type: argdecl.ArgTypeInt, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{
		"amount": rawJSON(t, true), // bool, not int
	}
	_, rerr := ds.Resolve(raw)
	if rerr == nil {
		t.Fatal("type mismatch must be rejected")
	}
}

func TestResolve_TypeMismatch_Bool_GetsString(t *testing.T) {
	ds := makeDS(t, []argdecl.Declaration{
		{Name: "dry_run", Type: argdecl.ArgTypeBool, Required: true},
	})
	raw := map[string]argdecl.RawArgValue{
		"dry_run": rawJSON(t, "yes"), // string, not bool
	}
	_, rerr := ds.Resolve(raw)
	if rerr == nil {
		t.Fatal("type mismatch must be rejected")
	}
}

// ─── Empty declaration set ────────────────────────────────────────────────────

func TestResolve_EmptyDeclarationSet_NoArgReachesPolicy(t *testing.T) {
	ds := makeDS(t, nil) // no declared args
	raw := map[string]argdecl.RawArgValue{
		"anything": rawJSON(t, "value"),
	}
	result, rerr := ds.Resolve(raw)
	if rerr != nil {
		t.Fatalf("expected success: %v", rerr)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result when no args declared, got %d", len(result))
	}
}
