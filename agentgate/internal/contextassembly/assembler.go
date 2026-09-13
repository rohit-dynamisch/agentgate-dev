// Package contextassembly provides the narrow adapter from validated
// identity, authoritative tool governance metadata, and resolved typed
// argument declarations into the frozen G1 [decision.Request].
//
// # Scope
//
// This package only populates fields already represented in the frozen
// G1 contract. It never invents missing security context. If any input
// is invalid (identity mapping failed, governance record is drifted/unknown,
// argument resolution failed), the adapter returns an error and the caller
// must deny.
//
// # O-008 gap (explicitly documented)
//
// This adapter does NOT solve O-008 (how agentgateway's ext_authz callout
// becomes a decision.Request). The production mechanism for translating a
// real ext_authz callout through internal/authz into a decision.Request
// is undesigned and must not be invented here.
// See docs/DECISIONS/OPEN_DECISIONS.md O-008.
//
// # What this package does
//
// The adapter receives pre-resolved inputs (after identity mapping,
// tool registry lookup, and argument declaration resolution are complete)
// and assembles them into the exact decision.Request shape. It does not
// re-validate inputs beyond checking that they are non-zero/coherent.
package contextassembly

import (
	"fmt"

	"github.com/Dynamisch-LLC/agentgate/internal/argdecl"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/identity"
	"github.com/Dynamisch-LLC/agentgate/internal/toolregistry"
)

// AssemblyInput holds the pre-resolved, pre-validated inputs for one
// authorization context assembly. All inputs must already be validated
// by their respective packages before being passed here.
type AssemblyInput struct {
	// ExecutionID correlates this request for audit. Required non-empty.
	ExecutionID string

	// WorkspaceID scopes the request. Required non-empty.
	WorkspaceID string

	// MappedIdentity is the result of a successful identity mapping.
	// The caller must NOT pass an identity that failed mapping.
	MappedIdentity identity.MappedIdentity

	// GovernanceRecord is the authoritative tool governance state.
	// The caller must NOT pass a record that fails GovernanceRecord.Validate().
	GovernanceRecord toolregistry.GovernanceRecord

	// ResolvedArgs is the output of argdecl.DeclarationSet.Resolve.
	// May be nil/empty when no policy-relevant arguments are declared.
	// The caller must NOT pass unresolved or undeclared arguments here.
	ResolvedArgs map[string]argdecl.ResolvedArg
}

// AssemblyError is returned when the adapter cannot produce a valid
// decision.Request from the given inputs.
type AssemblyError struct {
	Field  string
	Reason string
}

func (e *AssemblyError) Error() string {
	return fmt.Sprintf("contextassembly: field %q: %s", e.Field, e.Reason)
}

// Assemble converts validated inputs into a [decision.Request].
//
// Fails with [AssemblyError] if:
//   - ExecutionID or WorkspaceID is empty
//   - MappedIdentity has empty AgentID or no roles
//   - GovernanceRecord.Validate() fails (unknown/drifted/bad risk)
//   - Any ResolvedArg has an unrecognized type (should be impossible after
//     argdecl.Resolve, but fail-closed if it somehow occurs)
//
// On success, the returned [decision.Request] contains only fields from
// the frozen G1 contract. No extra fields are invented.
func Assemble(in AssemblyInput) (decision.Request, *AssemblyError) {
	// --- Required correlation fields ---
	if in.ExecutionID == "" {
		return decision.Request{}, &AssemblyError{Field: "ExecutionID", Reason: "must not be empty"}
	}
	if in.WorkspaceID == "" {
		return decision.Request{}, &AssemblyError{Field: "WorkspaceID", Reason: "must not be empty"}
	}

	// --- Identity ---
	if in.MappedIdentity.AgentID == "" {
		return decision.Request{}, &AssemblyError{
			Field:  "MappedIdentity.AgentID",
			Reason: "must not be empty (identity mapping should have failed before reaching here)",
		}
	}
	if len(in.MappedIdentity.Roles) == 0 {
		return decision.Request{}, &AssemblyError{
			Field:  "MappedIdentity.Roles",
			Reason: "must not be empty (identity mapping should have failed before reaching here)",
		}
	}

	// --- Tool governance ---
	if err := in.GovernanceRecord.Validate(); err != nil {
		return decision.Request{}, &AssemblyError{
			Field:  "GovernanceRecord",
			Reason: err.Error(),
		}
	}

	// --- Argument translation ---
	// ResolvedArgs → decision.Arguments (typed AttributeValues)
	// Only the three types declared in decision.AttributeValue are supported.
	args := make(map[string]decision.AttributeValue, len(in.ResolvedArgs))
	for name, resolved := range in.ResolvedArgs {
		attr, err := toAttributeValue(name, resolved)
		if err != nil {
			return decision.Request{}, err
		}
		args[name] = attr
	}

	return decision.Request{
		ExecutionID: in.ExecutionID,
		WorkspaceID: in.WorkspaceID,
		Identity: decision.Identity{
			AgentID:    in.MappedIdentity.AgentID,
			OnBehalfOf: in.MappedIdentity.OnBehalfOf,
			Roles:      append([]string(nil), in.MappedIdentity.Roles...),
		},
		Tool: decision.ToolRef{
			BackendID: in.GovernanceRecord.ToolID.BackendID,
			Name:      in.GovernanceRecord.ToolID.ToolName,
		},
		Classification: decision.ToolClassification{
			Known: true,
			Risk:  string(in.GovernanceRecord.Risk),
		},
		Arguments: args,
	}, nil
}

// toAttributeValue converts a [argdecl.ResolvedArg] to a [decision.AttributeValue].
// Returns an AssemblyError if the type is somehow unrecognized (defensive;
// argdecl.Resolve already enforces the type at resolution time).
func toAttributeValue(name string, r argdecl.ResolvedArg) (decision.AttributeValue, *AssemblyError) {
	switch r.Type {
	case argdecl.ArgTypeString:
		return decision.StringAttr(r.StrVal), nil
	case argdecl.ArgTypeInt:
		return decision.IntAttr(r.IntVal), nil
	case argdecl.ArgTypeBool:
		return decision.BoolAttr(r.BoolVal), nil
	default:
		return decision.AttributeValue{}, &AssemblyError{
			Field:  "ResolvedArgs[" + name + "]",
			Reason: fmt.Sprintf("unrecognized ArgType %q — this is a programming error", r.Type),
		}
	}
}
