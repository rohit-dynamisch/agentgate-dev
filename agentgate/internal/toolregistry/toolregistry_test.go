package toolregistry_test

import (
	"strings"
	"testing"

	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// ─── FingerprintSchema ────────────────────────────────────────────────────────

func TestFingerprintSchema_Deterministic(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"amount":{"type":"integer"}}}`)
	fp1, err := toolregistry.FingerprintSchema(schema)
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}
	fp2, err := toolregistry.FingerprintSchema(schema)
	if err != nil {
		t.Fatalf("second fingerprint: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("fingerprint not deterministic: %q vs %q", fp1, fp2)
	}
}

func TestFingerprintSchema_KeyOrderIndependent(t *testing.T) {
	// Same schema, different key order — must produce same fingerprint.
	schema1 := []byte(`{"properties":{"b":{"type":"string"},"a":{"type":"integer"}},"type":"object"}`)
	schema2 := []byte(`{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"string"}}}`)
	fp1, err := toolregistry.FingerprintSchema(schema1)
	if err != nil {
		t.Fatalf("schema1: %v", err)
	}
	fp2, err := toolregistry.FingerprintSchema(schema2)
	if err != nil {
		t.Fatalf("schema2: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("key order changed fingerprint: %q vs %q", fp1, fp2)
	}
}

func TestFingerprintSchema_MeaningfulChangeAltersFingerprint(t *testing.T) {
	schema1 := []byte(`{"type":"object","properties":{"amount":{"type":"integer"}}}`)
	schema2 := []byte(`{"type":"object","properties":{"amount":{"type":"string"}}}`)
	fp1, _ := toolregistry.FingerprintSchema(schema1)
	fp2, _ := toolregistry.FingerprintSchema(schema2)
	if fp1 == fp2 {
		t.Error("different schemas must produce different fingerprints")
	}
}

func TestFingerprintSchema_EmptyInput_FailsClosed(t *testing.T) {
	_, err := toolregistry.FingerprintSchema(nil)
	if err == nil {
		t.Fatal("expected error for nil schema")
	}
	_, err = toolregistry.FingerprintSchema([]byte{})
	if err == nil {
		t.Fatal("expected error for empty schema")
	}
}

func TestFingerprintSchema_InvalidJSON_FailsClosed(t *testing.T) {
	_, err := toolregistry.FingerprintSchema([]byte(`{not json}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON schema")
	}
}

func TestFingerprintSchema_IsHex(t *testing.T) {
	fp, err := toolregistry.FingerprintSchema([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fp) != 64 {
		t.Errorf("expected 64-char hex fingerprint, got len=%d", len(fp))
	}
	for _, c := range string(fp) {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("fingerprint contains non-hex char: %q", c)
			break
		}
	}
}

// ─── FingerprintProperties ────────────────────────────────────────────────────

func TestFingerprintProperties_Deterministic(t *testing.T) {
	props := map[string]string{"amount": "integer", "name": "string"}
	fp1, err := toolregistry.FingerprintProperties(props)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	fp2, err := toolregistry.FingerprintProperties(props)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("not deterministic: %q vs %q", fp1, fp2)
	}
}

func TestFingerprintProperties_NilFailsClosed(t *testing.T) {
	_, err := toolregistry.FingerprintProperties(nil)
	if err == nil {
		t.Fatal("expected error for nil properties")
	}
}

// ─── CheckDrift ───────────────────────────────────────────────────────────────

func TestCheckDrift_None(t *testing.T) {
	fp, _ := toolregistry.FingerprintSchema([]byte(`{"type":"object"}`))
	if toolregistry.CheckDrift(fp, fp) != toolregistry.DriftNone {
		t.Error("same fingerprint should be DriftNone")
	}
}

func TestCheckDrift_Detected(t *testing.T) {
	fp1, _ := toolregistry.FingerprintSchema([]byte(`{"type":"object"}`))
	fp2, _ := toolregistry.FingerprintSchema([]byte(`{"type":"string"}`))
	if toolregistry.CheckDrift(fp1, fp2) != toolregistry.DriftDetected {
		t.Error("different fingerprints should be DriftDetected")
	}
}

// ─── ToolID ───────────────────────────────────────────────────────────────────

func TestToolID_SameName_DifferentBackend_IsDistinct(t *testing.T) {
	id1 := toolregistry.ToolID{BackendID: "backend-a", ToolName: "list"}
	id2 := toolregistry.ToolID{BackendID: "backend-b", ToolName: "list"}
	if id1.String() == id2.String() {
		t.Error("same tool name on different backends must be distinct")
	}
}

func TestToolID_Valid(t *testing.T) {
	if (toolregistry.ToolID{BackendID: "b", ToolName: "t"}).Valid() == false {
		t.Error("should be valid")
	}
	if (toolregistry.ToolID{BackendID: "", ToolName: "t"}).Valid() == true {
		t.Error("empty BackendID should be invalid")
	}
	if (toolregistry.ToolID{BackendID: "b", ToolName: ""}).Valid() == true {
		t.Error("empty ToolName should be invalid")
	}
}

// ─── Registry ─────────────────────────────────────────────────────────────────

func validEntry(t *testing.T) toolregistry.RegistryEntry {
	t.Helper()
	fp, err := toolregistry.FingerprintSchema([]byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	return toolregistry.RegistryEntry{
		ToolID:                toolregistry.ToolID{BackendID: "backend-a", ToolName: "list_files"},
		Risk:                  toolregistry.RiskRead,
		RegisteredFingerprint: fp,
	}
}

func TestRegistry_Lookup_KnownTool(t *testing.T) {
	entry := validEntry(t)
	reg, err := toolregistry.NewRegistry([]toolregistry.RegistryEntry{entry})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	rec := reg.Lookup(entry.ToolID, entry.RegisteredFingerprint)
	if !rec.Known {
		t.Error("expected Known=true for registered tool")
	}
	if rec.DriftStatus != toolregistry.DriftNone {
		t.Errorf("expected DriftNone, got %q", rec.DriftStatus)
	}
	if rec.Risk != toolregistry.RiskRead {
		t.Errorf("expected RiskRead, got %q", rec.Risk)
	}
}

func TestRegistry_Lookup_UnknownTool(t *testing.T) {
	reg, _ := toolregistry.NewRegistry(nil)
	unknown := toolregistry.ToolID{BackendID: "backend-x", ToolName: "unknown"}
	rec := reg.Lookup(unknown, "")
	if rec.Known {
		t.Error("expected Known=false for unregistered tool")
	}
	if rec.DriftStatus != toolregistry.DriftUnknownTool {
		t.Errorf("expected DriftUnknownTool, got %q", rec.DriftStatus)
	}
}

func TestRegistry_UnknownTool_CannotInheritBySameName(t *testing.T) {
	entry := validEntry(t)
	reg, _ := toolregistry.NewRegistry([]toolregistry.RegistryEntry{entry})

	// Same tool name, different backend — must NOT inherit classification.
	otherBackend := toolregistry.ToolID{BackendID: "attacker-backend", ToolName: entry.ToolID.ToolName}
	rec := reg.Lookup(otherBackend, "")
	if rec.Known {
		t.Error("unknown tool must NOT inherit classification from same-named tool on another backend")
	}
}

func TestRegistry_DriftDetected_FailsClosed(t *testing.T) {
	entry := validEntry(t)
	reg, _ := toolregistry.NewRegistry([]toolregistry.RegistryEntry{entry})

	// Different live fingerprint simulates schema drift.
	liveNew, _ := toolregistry.FingerprintSchema([]byte(`{"type":"string"}`))
	rec := reg.Lookup(entry.ToolID, liveNew)
	if rec.DriftStatus != toolregistry.DriftDetected {
		t.Errorf("expected DriftDetected, got %q", rec.DriftStatus)
	}
	// Validate should fail closed on drift.
	if err := rec.Validate(); err == nil {
		t.Error("GovernanceRecord.Validate should fail closed on drift")
	}
}

func TestNewRegistry_InvalidEntries(t *testing.T) {
	fp, _ := toolregistry.FingerprintSchema([]byte(`{}`))
	tests := []struct {
		name  string
		entry toolregistry.RegistryEntry
	}{
		{
			name:  "empty BackendID",
			entry: toolregistry.RegistryEntry{ToolID: toolregistry.ToolID{ToolName: "t"}, Risk: toolregistry.RiskRead, RegisteredFingerprint: fp},
		},
		{
			name:  "empty ToolName",
			entry: toolregistry.RegistryEntry{ToolID: toolregistry.ToolID{BackendID: "b"}, Risk: toolregistry.RiskRead, RegisteredFingerprint: fp},
		},
		{
			name:  "invalid risk",
			entry: toolregistry.RegistryEntry{ToolID: toolregistry.ToolID{BackendID: "b", ToolName: "t"}, Risk: "superadmin", RegisteredFingerprint: fp},
		},
		{
			name:  "empty fingerprint",
			entry: toolregistry.RegistryEntry{ToolID: toolregistry.ToolID{BackendID: "b", ToolName: "t"}, Risk: toolregistry.RiskRead},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := toolregistry.NewRegistry([]toolregistry.RegistryEntry{tc.entry})
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestNewRegistry_DuplicateEntry(t *testing.T) {
	entry := validEntry(t)
	_, err := toolregistry.NewRegistry([]toolregistry.RegistryEntry{entry, entry})
	if err == nil {
		t.Error("expected error for duplicate registry entry")
	}
}

func TestGovernanceRecord_Validate(t *testing.T) {
	fp, _ := toolregistry.FingerprintSchema([]byte(`{}`))
	good := toolregistry.GovernanceRecord{
		ToolID:                toolregistry.ToolID{BackendID: "b", ToolName: "t"},
		Known:                 true,
		Risk:                  toolregistry.RiskRead,
		RegisteredFingerprint: fp,
		DriftStatus:           toolregistry.DriftNone,
	}
	if err := good.Validate(); err != nil {
		t.Errorf("expected valid record: %v", err)
	}

	// Unknown tool fails closed.
	unknown := good
	unknown.Known = false
	if toolregistry.GovernanceRecord(unknown).Validate() == nil {
		t.Error("Unknown record should fail Validate")
	}

	// Drifted tool fails closed.
	drifted := good
	drifted.DriftStatus = toolregistry.DriftDetected
	if toolregistry.GovernanceRecord(drifted).Validate() == nil {
		t.Error("Drifted record should fail Validate")
	}

	// Missing risk fails closed.
	noRisk := good
	noRisk.Risk = ""
	if toolregistry.GovernanceRecord(noRisk).Validate() == nil {
		t.Error("Empty risk should fail Validate")
	}

	// Unrecognized risk fails closed.
	badRisk := good
	badRisk.Risk = "nuclear"
	if toolregistry.GovernanceRecord(badRisk).Validate() == nil {
		t.Error("Unrecognized risk should fail Validate")
	}
}
