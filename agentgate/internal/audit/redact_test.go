package audit_test

import (
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/audit"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
)

func TestRedactor_SensitiveKeysNeverDisclosed(t *testing.T) {
	redactor := audit.NewRedactor("test-salt-secret", nil)

	raw := map[string]decision.AttributeValue{
		"safe_query":    decision.StringAttr("SELECT 1"),
		"password":      decision.StringAttr("super-secret-password-123"),
		"api_key":       decision.StringAttr("secret-key-abc"),
		"authorization": decision.StringAttr("Bearer eyJhbGci..."),
	}

	redacted := redactor.RedactArguments("db_tool", raw)

	if redacted["password"] == "super-secret-password-123" {
		t.Fatal("raw password leaked in redacted map!")
	}
	if redacted["api_key"] == "secret-key-abc" {
		t.Fatal("raw api_key leaked in redacted map!")
	}
	if redacted["authorization"] == "Bearer eyJhbGci..." {
		t.Fatal("raw authorization token leaked in redacted map!")
	}
	if redacted["safe_query"] != "SELECT 1" {
		t.Fatalf("expected safe_query preserved, got %s", redacted["safe_query"])
	}
}

func TestRedactor_SaltedHashModeIsDeterministic(t *testing.T) {
	rules := map[string]map[string]audit.RedactionMode{
		"user_tool": {"user_ssn": audit.RedactHash},
	}
	redactor1 := audit.NewRedactor("salt-xyz", rules)
	redactor2 := audit.NewRedactor("salt-xyz", rules)
	redactorDiffSalt := audit.NewRedactor("different-salt", rules)

	raw := map[string]decision.AttributeValue{
		"user_ssn": decision.StringAttr("123-45-6789"),
	}

	h1 := redactor1.RedactArguments("user_tool", raw)["user_ssn"]
	h2 := redactor2.RedactArguments("user_tool", raw)["user_ssn"]
	hDiff := redactorDiffSalt.RedactArguments("user_tool", raw)["user_ssn"]

	if h1 == "" || h1 == "123-45-6789" {
		t.Fatalf("hash was not computed: %s", h1)
	}
	if h1 != h2 {
		t.Fatal("same salt must produce identical hash")
	}
	if h1 == hDiff {
		t.Fatal("different salt must produce different hash")
	}
}

func TestRedactor_OmitMode(t *testing.T) {
	rules := map[string]map[string]audit.RedactionMode{
		"admin_tool": {"session_token": audit.RedactOmit},
	}
	redactor := audit.NewRedactor("salt", rules)

	raw := map[string]decision.AttributeValue{
		"session_token": decision.StringAttr("sess_abc123"),
	}

	res := redactor.RedactArguments("admin_tool", raw)
	if res["session_token"] == "sess_abc123" {
		t.Fatal("session_token leaked raw in omit mode")
	}
	if res["session_token"] != "[REDACTED]" {
		t.Fatalf("expected [REDACTED], got %s", res["session_token"])
	}
}

func TestRedactor_BearerPatternDetectedAndRedacted(t *testing.T) {
	// Even if rule is full, bearer tokens must never be disclosed raw
	rules := map[string]map[string]audit.RedactionMode{
		"custom_tool": {"header": audit.RedactFull},
	}
	redactor := audit.NewRedactor("salt", rules)

	raw := map[string]decision.AttributeValue{
		"header": decision.StringAttr("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."),
	}

	res := redactor.RedactArguments("custom_tool", raw)
	if res["header"] == "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." {
		t.Fatal("bearer token was leaked raw despite safety keyword scan")
	}
}
