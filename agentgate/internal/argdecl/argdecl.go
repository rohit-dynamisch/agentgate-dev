// Package argdecl defines per-tool typed argument policy-input declarations.
//
// # Purpose
//
// Not every tool argument is policy-relevant. This package defines an
// explicit whitelist of argument names and their expected types for each
// tool. Only declared arguments can enter policy evaluation (via
// [decision.Request.Arguments]); undeclared arguments are structurally
// invisible to Cedar.
//
// # Type enforcement
//
// Each declared argument has an [ArgType] (string, int64, or bool). When
// arguments are resolved, the caller must provide a raw value (from the
// tool invocation) that matches the declared type. Type mismatches fail
// closed — the argument (and the entire invocation context) is rejected.
//
// # Null and missing-optional handling
//
// JSON null is rejected for all declared argument types. It is never
// coerced to a zero value (a null integer is not 0; a null bool is not
// false). This closes the null-coercion bug documented in G1.
//
// A missing optional argument is distinct from an explicit null: missing
// optional → the argument is simply absent from [decision.Request.Arguments];
// explicit null → MappingError returned.
//
// # Duplicate and conflicting declarations
//
// A [DeclarationSet] with duplicate argument names is invalid and fails at
// construction time ([NewDeclarationSet] returns an error).
//
// # Schema mismatch
//
// Declaration/schema mismatch (a declared argument not present in the tool
// schema, or a schema field not declared) is detected at construction and
// reported as a validation warning; actual enforcement is fail-closed.
package argdecl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ─── Argument types ───────────────────────────────────────────────────────────

// ArgType is the closed set of policy-relevant argument value types.
// Matches the closed set in [decision.AttributeValue].
type ArgType string

const (
	ArgTypeString ArgType = "string"
	ArgTypeInt    ArgType = "int64"
	ArgTypeBool   ArgType = "bool"
)

// ValidArgType reports whether t is a recognized ArgType.
func ValidArgType(t ArgType) bool {
	switch t {
	case ArgTypeString, ArgTypeInt, ArgTypeBool:
		return true
	default:
		return false
	}
}

// ─── Declaration ─────────────────────────────────────────────────────────────

// Declaration specifies one policy-relevant argument for a tool.
type Declaration struct {
	// Name is the argument name as it appears in the tool's MCP input.
	// Required, non-empty.
	Name string

	// Type is the expected value type. Required.
	Type ArgType

	// Required, when true, means the argument MUST be present in the
	// tool invocation. When false (optional), it may be absent —
	// absent optional arguments simply do not appear in the resolved map.
	// An explicit JSON null is always rejected, whether required or optional.
	Required bool
}

// ─── Declaration set ─────────────────────────────────────────────────────────

// DeclarationSet is the validated, immutable set of argument declarations
// for one tool. Constructed via [NewDeclarationSet].
type DeclarationSet struct {
	decls map[string]Declaration // key: Declaration.Name
}

// NewDeclarationSet validates and constructs a DeclarationSet from a list
// of declarations. Returns an error for:
//   - duplicate argument names
//   - empty argument name
//   - unrecognized ArgType
func NewDeclarationSet(decls []Declaration) (*DeclarationSet, error) {
	seen := make(map[string]struct{}, len(decls))
	m := make(map[string]Declaration, len(decls))
	for _, d := range decls {
		if strings.TrimSpace(d.Name) == "" {
			return nil, fmt.Errorf("argdecl: declaration has empty name")
		}
		if !ValidArgType(d.Type) {
			return nil, fmt.Errorf("argdecl: declaration %q has unrecognized type %q", d.Name, d.Type)
		}
		if _, dup := seen[d.Name]; dup {
			return nil, fmt.Errorf("argdecl: duplicate declaration for argument %q", d.Name)
		}
		seen[d.Name] = struct{}{}
		m[d.Name] = d
	}
	return &DeclarationSet{decls: m}, nil
}

// Get returns the Declaration for name, and whether it exists.
func (ds *DeclarationSet) Get(name string) (Declaration, bool) {
	d, ok := ds.decls[name]
	return d, ok
}

// Names returns all declared argument names (unordered).
func (ds *DeclarationSet) Names() []string {
	names := make([]string, 0, len(ds.decls))
	for n := range ds.decls {
		names = append(names, n)
	}
	return names
}

// ─── Raw argument value ───────────────────────────────────────────────────────

// RawArgValue is an untyped argument value from a tool invocation. It is
// typically decoded from MCP request JSON. Use [json.RawMessage] so null
// and missing can be distinguished by callers.
type RawArgValue = json.RawMessage

// ─── Resolution result ────────────────────────────────────────────────────────

// ResolvedArg is a successfully typed, validated argument value. The
// field matching the declaration's Type is populated; others are zero.
type ResolvedArg struct {
	Type    ArgType
	StrVal  string
	IntVal  int64
	BoolVal bool
}

// ResolutionError describes why an argument value could not be resolved.
type ResolutionError struct {
	ArgName string
	Reason  string
}

func (e *ResolutionError) Error() string {
	return fmt.Sprintf("argdecl: argument %q: %s", e.ArgName, e.Reason)
}

// ─── Resolution ───────────────────────────────────────────────────────────────

// Resolve maps raw argument values (from a tool invocation) through the
// declaration set, producing a type-safe map suitable for populating
// [decision.Request.Arguments].
//
// Rules:
//   - Undeclared arguments in raw are silently ignored (never reach policy).
//   - Required declared arguments missing from raw produce a ResolutionError.
//   - Explicit JSON null for any declared argument produces a ResolutionError.
//   - Type mismatches produce a ResolutionError.
//   - Optional absent arguments are simply absent from the result map.
func (ds *DeclarationSet) Resolve(raw map[string]RawArgValue) (map[string]ResolvedArg, *ResolutionError) {
	result := make(map[string]ResolvedArg, len(ds.decls))

	for name, decl := range ds.decls {
		rawVal, present := raw[name]

		if !present {
			if decl.Required {
				return nil, &ResolutionError{
					ArgName: name,
					Reason:  "required argument is missing",
				}
			}
			// Optional and absent — simply omit from result.
			continue
		}

		// Explicit null check — reject regardless of required/optional.
		if isNull(rawVal) {
			return nil, &ResolutionError{
				ArgName: name,
				Reason:  "explicit JSON null is not allowed (use absent for optional arguments)",
			}
		}

		resolved, rerr := decodeTyped(name, decl.Type, rawVal)
		if rerr != nil {
			return nil, rerr
		}
		result[name] = resolved
	}

	return result, nil
}

// isNull reports whether raw is a JSON null literal.
func isNull(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed == "null"
}

// decodeTyped decodes a raw JSON value into the expected ArgType.
func decodeTyped(name string, want ArgType, raw json.RawMessage) (ResolvedArg, *ResolutionError) {
	switch want {
	case ArgTypeString:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ResolvedArg{}, &ResolutionError{
				ArgName: name,
				Reason:  fmt.Sprintf("expected string, got: %s", safeRawPreview(raw)),
			}
		}
		return ResolvedArg{Type: ArgTypeString, StrVal: s}, nil

	case ArgTypeInt:
		var n int64
		if err := json.Unmarshal(raw, &n); err != nil {
			return ResolvedArg{}, &ResolutionError{
				ArgName: name,
				Reason:  fmt.Sprintf("expected int64, got: %s", safeRawPreview(raw)),
			}
		}
		return ResolvedArg{Type: ArgTypeInt, IntVal: n}, nil

	case ArgTypeBool:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return ResolvedArg{}, &ResolutionError{
				ArgName: name,
				Reason:  fmt.Sprintf("expected bool, got: %s", safeRawPreview(raw)),
			}
		}
		return ResolvedArg{Type: ArgTypeBool, BoolVal: b}, nil

	default:
		// Unreachable: NewDeclarationSet validates all types at construction.
		return ResolvedArg{}, &ResolutionError{ArgName: name, Reason: "internal: unrecognized type"}
	}
}

func safeRawPreview(raw json.RawMessage) string {
	s := string(raw)
	if len(s) > 32 {
		return s[:32] + "…"
	}
	return s
}
