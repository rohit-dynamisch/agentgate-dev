// Package toolregistry defines the authoritative tool identity, inventory,
// and governance boundary for AgentGate.
//
// # Tool identity
//
// A tool's canonical identity is the triple (BackendID, ToolName,
// SchemaFingerprint). Fingerprinting is deterministic: the same schema
// always produces the same fingerprint. A schema change produces a
// different fingerprint, which triggers fail-closed governance review.
//
// # Algorithm note (O-005)
//
// The schema fingerprint algorithm used here (SHA-256 of canonical JSON)
// is NOT frozen as a repository-level decision. It is an implementation
// choice that is explicitly marked as such in code comments. If O-005
// is later resolved differently, the fingerprint algorithm must be
// updated and all stored fingerprints recomputed.
// See docs/DECISIONS/OPEN_DECISIONS.md O-005.
//
// # Governance
//
// [GovernanceRecord] is the authoritative governance state for one tool.
// Unknown tools cannot inherit classification by name similarity.
// Fingerprint mismatch fails closed. No resource-level authorization is
// added here (that is a later phase).
//
// # Argument declarations
//
// Per-tool typed argument declarations live in [internal/argdecl].
// [GovernanceRecord] holds only a reference to their names; the
// declaration registry is a separate boundary.
package toolregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// ─── Tool identity ────────────────────────────────────────────────────────────

// ToolID is the canonical, immutable identity for one tool served by one
// backend. The combination (BackendID + ToolName) is unique within a
// workspace; the same tool name on a different backend is a different tool.
type ToolID struct {
	// BackendID identifies the MCP backend/resource serving this tool.
	// Required, non-empty.
	BackendID string

	// ToolName is the name the tool is registered under on this backend.
	// Required, non-empty.
	ToolName string
}

// String returns a human-readable representation for logs/audit.
func (t ToolID) String() string {
	return t.BackendID + "/" + t.ToolName
}

// Valid reports whether both required fields are non-empty.
func (t ToolID) Valid() bool {
	return t.BackendID != "" && t.ToolName != ""
}

// ─── Risk classification ──────────────────────────────────────────────────────

// RiskLevel is the closed vocabulary of tool risk values understood by
// the Cedar policy layer. The set must stay in sync with fixturepolicy
// and any Cedar policies that reference them.
type RiskLevel string

const (
	RiskRead        RiskLevel = "read"
	RiskWrite       RiskLevel = "write"
	RiskDestructive RiskLevel = "destructive"
)

// ValidRisk reports whether r is a non-empty, recognized risk level.
// An empty or unrecognized risk fails closed (same as unknown tool).
func ValidRisk(r RiskLevel) bool {
	switch r {
	case RiskRead, RiskWrite, RiskDestructive:
		return true
	default:
		return false
	}
}

// ─── Schema fingerprint ───────────────────────────────────────────────────────

// SchemaFingerprint is the hex-encoded SHA-256 of the canonical JSON
// serialization of a tool's MCP input schema.
//
// Algorithm note: SHA-256 of canonical JSON is the chosen algorithm.
// This is NOT a frozen repository decision (O-005). If the algorithm
// changes, stored fingerprints must be recomputed.
type SchemaFingerprint string

// FingerprintSchema computes a deterministic [SchemaFingerprint] from a
// raw JSON schema. The schema is canonicalized before hashing: object
// keys are sorted, whitespace is removed. This ensures that semantically
// identical schemas (differing only in key order or whitespace) produce
// the same fingerprint.
//
// Returns an error for nil, empty, or non-JSON input. Invalid JSON fails
// closed: it will not receive a valid governance fingerprint.
func FingerprintSchema(rawSchemaJSON []byte) (SchemaFingerprint, error) {
	if len(rawSchemaJSON) == 0 {
		return "", errors.New("toolregistry: schema fingerprint: schema is empty")
	}

	// Unmarshal then re-marshal with sorted keys (encoding/json sorts map
	// keys) to produce canonical JSON.
	var v interface{}
	if err := json.Unmarshal(rawSchemaJSON, &v); err != nil {
		return "", fmt.Errorf("toolregistry: schema fingerprint: invalid JSON: %w", err)
	}

	canonical, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("toolregistry: schema fingerprint: re-marshal: %w", err)
	}

	sum := sha256.Sum256(canonical)
	return SchemaFingerprint(hex.EncodeToString(sum[:])), nil
}

// FingerprintProperties computes a [SchemaFingerprint] from a pre-parsed
// map of property name → JSON type string. This is a convenience for
// callers that have already parsed the schema's "properties" field.
// Properties are sorted deterministically before hashing.
func FingerprintProperties(props map[string]string) (SchemaFingerprint, error) {
	if props == nil {
		return "", errors.New("toolregistry: schema fingerprint: nil properties")
	}
	// Sort keys for determinism.
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]any, 0, len(keys))
	for _, k := range keys {
		ordered = append(ordered, map[string]string{"name": k, "type": props[k]})
	}

	canonical, err := json.Marshal(ordered)
	if err != nil {
		return "", fmt.Errorf("toolregistry: schema fingerprint: marshal: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return SchemaFingerprint(hex.EncodeToString(sum[:])), nil
}

// ─── Drift status ─────────────────────────────────────────────────────────────

// DriftStatus is the result of comparing a live schema fingerprint against
// the governance-registered fingerprint for a tool.
type DriftStatus string

const (
	// DriftNone: live fingerprint matches governance record. Tool is
	// eligible for policy evaluation.
	DriftNone DriftStatus = "none"

	// DriftDetected: live fingerprint differs from governance record.
	// The tool must fail closed until governance is updated.
	DriftDetected DriftStatus = "detected"

	// DriftUnknownTool: the tool has no governance record. It cannot
	// inherit classification by name similarity and fails closed.
	DriftUnknownTool DriftStatus = "unknown_tool"
)

// CheckDrift compares live against registered. Returns [DriftDetected]
// when they differ and [DriftNone] when they match. Does NOT handle the
// unknown-tool case (no registered record) — callers use [GovernanceRecord.Known].
func CheckDrift(registered, live SchemaFingerprint) DriftStatus {
	if registered == live {
		return DriftNone
	}
	return DriftDetected
}

// ─── Governance record ────────────────────────────────────────────────────────

// GovernanceRecord is the authoritative governance state for one tool.
// It is produced by the tool registry and consumed by the context assembly
// adapter (internal/contextassembly).
//
// A GovernanceRecord is only valid when [GovernanceRecord.Known] is true
// AND [GovernanceRecord.DriftStatus] is [DriftNone] AND [GovernanceRecord.Risk]
// is a recognized risk level. Any other combination fails closed.
type GovernanceRecord struct {
	// ToolID is the canonical identity of this tool.
	ToolID ToolID

	// Known is false for any tool without a governance record. A request
	// for an unknown tool always fails closed; unknown tools cannot inherit
	// classification from a same-named tool on another backend.
	Known bool

	// Risk is meaningful only when Known is true. Must be a recognized
	// [RiskLevel] value; empty or unrecognized risk fails closed.
	Risk RiskLevel

	// RegisteredFingerprint is the schema fingerprint at time of governance
	// registration. Used by callers to detect drift against the live schema.
	RegisteredFingerprint SchemaFingerprint

	// DriftStatus is pre-computed by the registry when the live schema
	// fingerprint is available. Callers that have the live schema must
	// verify this field before treating the record as authoritative.
	DriftStatus DriftStatus
}

// GovernanceError describes why a [GovernanceRecord] cannot be used for
// authorization.
type GovernanceError struct {
	Reason string
}

func (e *GovernanceError) Error() string {
	return "toolregistry: " + e.Reason
}

// Validate checks that a GovernanceRecord is fully authoritative:
// Known, non-drifted, valid risk. Returns a [GovernanceError] if not.
func (r GovernanceRecord) Validate() error {
	if !r.ToolID.Valid() {
		return &GovernanceError{Reason: "tool id is incomplete (BackendID or ToolName empty)"}
	}
	if !r.Known {
		return &GovernanceError{Reason: fmt.Sprintf("tool %s is not in the governance registry", r.ToolID)}
	}
	if r.DriftStatus == DriftDetected {
		return &GovernanceError{Reason: fmt.Sprintf("tool %s schema drift detected; governance review required", r.ToolID)}
	}
	if !ValidRisk(r.Risk) {
		return &GovernanceError{Reason: fmt.Sprintf("tool %s has missing or unrecognized risk %q", r.ToolID, r.Risk)}
	}
	return nil
}

// ─── Registry ─────────────────────────────────────────────────────────────────

// RegistryEntry is one registered tool entry. It holds the governance
// metadata at time of registration. Schema fingerprints are computed
// from the registered schema at registration time.
type RegistryEntry struct {
	ToolID                ToolID
	Risk                  RiskLevel
	RegisteredFingerprint SchemaFingerprint
}

// Registry is the authoritative in-memory tool governance registry.
// In G2 it is loaded from fixture files; later phases (G3+) will load
// from a persistent store.
type Registry struct {
	entries map[string]RegistryEntry // key: ToolID.String()
}

// NewRegistry constructs a Registry from a list of entries. Returns an
// error if any entry has an invalid ToolID, invalid risk, or empty
// fingerprint.
func NewRegistry(entries []RegistryEntry) (*Registry, error) {
	r := &Registry{entries: make(map[string]RegistryEntry, len(entries))}
	for _, e := range entries {
		if !e.ToolID.Valid() {
			return nil, fmt.Errorf("toolregistry: entry with incomplete ToolID: %+v", e.ToolID)
		}
		if !ValidRisk(e.Risk) {
			return nil, fmt.Errorf("toolregistry: entry for %s has invalid risk %q", e.ToolID, e.Risk)
		}
		if e.RegisteredFingerprint == "" {
			return nil, fmt.Errorf("toolregistry: entry for %s has empty fingerprint", e.ToolID)
		}
		key := e.ToolID.String()
		if _, dup := r.entries[key]; dup {
			return nil, fmt.Errorf("toolregistry: duplicate entry for %s", e.ToolID)
		}
		r.entries[key] = e
	}
	return r, nil
}

// Lookup returns the [GovernanceRecord] for the given tool, optionally
// checking for schema drift when liveFingerprint is non-empty.
//
// For unknown tools, returns a GovernanceRecord with Known=false.
// Unknown tools cannot inherit classification by name similarity — only
// an exact (BackendID, ToolName) match can return Known=true.
func (r *Registry) Lookup(id ToolID, liveFingerprint SchemaFingerprint) GovernanceRecord {
	entry, ok := r.entries[id.String()]
	if !ok {
		return GovernanceRecord{
			ToolID:      id,
			Known:       false,
			DriftStatus: DriftUnknownTool,
		}
	}

	drift := DriftNone
	if liveFingerprint != "" {
		drift = CheckDrift(entry.RegisteredFingerprint, liveFingerprint)
	}

	return GovernanceRecord{
		ToolID:                id,
		Known:                 true,
		Risk:                  entry.Risk,
		RegisteredFingerprint: entry.RegisteredFingerprint,
		DriftStatus:           drift,
	}
}
