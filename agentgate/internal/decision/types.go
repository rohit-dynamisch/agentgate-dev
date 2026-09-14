package decision

import "strconv"

// Decision is the outcome of an authorization request. There are exactly
// two values — there is no third "error" decision. Every failure path
// (invalid identity, unknown tool, malformed input, a Cedar evaluation
// error, no policy loaded) resolves to Deny
// (docs/SECURITY/PRODUCTION-INVARIANTS.md §2, §8).
type Decision string

const (
	Allow Decision = "ALLOW"
	Deny  Decision = "DENY"
)

// ReasonCode is a stable, deterministic category for why a Decision was
// reached. It exists so audit records and callers never have to parse a
// free-text Message to know what happened.
type ReasonCode string

const (
	// ReasonPolicyAllow: Cedar evaluated the request and an explicit
	// permit matched.
	ReasonPolicyAllow ReasonCode = "policy_allow"

	// ReasonPolicyDeny: Cedar evaluated the request and an explicit
	// forbid matched (or a permit's conditions failed in a way Cedar
	// reports as a matched-but-denying policy).
	ReasonPolicyDeny ReasonCode = "policy_deny"

	// ReasonNoMatchingPolicy: Cedar evaluated the request and no policy
	// matched at all — deny-by-default.
	ReasonNoMatchingPolicy ReasonCode = "no_matching_policy"

	// ReasonInvalidIdentity: the request's Identity is missing, ambiguous
	// or otherwise unusable. Cedar was never reached.
	ReasonInvalidIdentity ReasonCode = "invalid_identity"

	// ReasonUnknownTool: the request's tool is not classified. Cedar was
	// never reached.
	ReasonUnknownTool ReasonCode = "unknown_tool"

	// ReasonMalformedRequest: the request is structurally incomplete or
	// contains an invalid value (e.g. a zero-value AttributeValue).
	// Cedar was never reached.
	ReasonMalformedRequest ReasonCode = "malformed_request"

	// ReasonEvaluationError: Cedar reported an internal evaluation error
	// while producing a decision. The result is never trusted as Allow
	// when this reason applies.
	ReasonEvaluationError ReasonCode = "evaluation_error"

	// ReasonNoPolicyLoaded: there is no valid policy loaded in this
	// Engine. Cedar was never reached.
	ReasonNoPolicyLoaded ReasonCode = "no_policy_loaded"
)

// Identity is the caller identity presented to the decision core. It is
// assumed already resolved and verified upstream — extracting it from
// gateway-validated JWT claims via a claims-mapping configuration is Day 3
// work (internal/identity). This package only validates that the shape it
// received is usable; it does not verify signatures or parse tokens.
type Identity struct {
	// AgentID identifies the calling agent. Required.
	AgentID string

	// OnBehalfOf identifies the human the agent is acting for, when
	// present. "" means no delegated human identity for this call.
	OnBehalfOf string

	// Roles are the identity's role assignments, evaluated against Cedar
	// principal-group membership. Required to be non-empty — an identity
	// with no roles cannot be authorized against any role-based policy
	// and is treated as unusable rather than silently falling through to
	// Cedar's own default-deny (docs/PHASES/DAY-02-TASK-02.md's
	// "missing/ambiguous/unusable identity" invariant).
	Roles []string
}

// ToolRef identifies the backend and tool a request targets.
type ToolRef struct {
	BackendID string
	Name      string
}

// ToolClassification is the tool-governance state input the decision core
// needs today: whether the tool is known/classified, and its risk level
// if so. The complete fingerprinting/drift-detection system that produces
// this is Day 3 work (docs/DECISIONS/OPEN_DECISIONS.md O-005); this is
// only the typed boundary Day 2 needs to deny unknown tools and support
// risk-based policy.
type ToolClassification struct {
	// Known is false for an unclassified/unrecognized tool. A request
	// with Known == false always denies with ReasonUnknownTool —
	// unconditionally, before Cedar is ever consulted
	// (docs/SECURITY/PRODUCTION-INVARIANTS.md §2 item 2).
	Known bool

	// Risk is meaningful only when Known is true (e.g. "read", "write",
	// "destructive"). It becomes the resource's "risk" attribute for
	// Cedar policy evaluation.
	Risk string
}

// attributeKind is the closed set of value kinds AttributeValue can hold.
type attributeKind uint8

const (
	attributeKindInvalid attributeKind = iota
	attributeKindString
	attributeKindInt
	attributeKindBool
)

// AttributeValue is a declared, policy-relevant tool-argument attribute.
// It is intentionally a small closed set (string/int64/bool). Day 2
// establishes only this typed boundary — there is no channel for
// undeclared data to reach policy evaluation: whatever the caller does
// not place into Request.Arguments is structurally invisible to Cedar.
// The complete per-tool declaration registry (which attribute names are
// allowed for which tool) is Day 3 work
// (docs/DECISIONS/OPEN_DECISIONS.md O-006).
//
// An AttributeValue can only be constructed through StringAttr, IntAttr or
// BoolAttr, so a zero-value AttributeValue (unset kind) is always
// detectably invalid — Evaluate treats it as a malformed request rather
// than silently ignoring or passing it through.
type AttributeValue struct {
	kind attributeKind
	str  string
	num  int64
	b    bool
}

// StringAttr, IntAttr and BoolAttr construct a valid AttributeValue.
func StringAttr(v string) AttributeValue { return AttributeValue{kind: attributeKindString, str: v} }
func IntAttr(v int64) AttributeValue     { return AttributeValue{kind: attributeKindInt, num: v} }
func BoolAttr(v bool) AttributeValue     { return AttributeValue{kind: attributeKindBool, b: v} }

func (a AttributeValue) valid() bool { return a.kind != attributeKindInvalid }

// String returns the string representation of the AttributeValue, implementing fmt.Stringer.
func (a AttributeValue) String() string {
	switch a.kind {
	case attributeKindString:
		return a.str
	case attributeKindInt:
		return strconv.FormatInt(a.num, 10)
	case attributeKindBool:
		return strconv.FormatBool(a.b)
	default:
		return ""
	}
}

// Request is AgentGate's typed authorization request — the stable domain
// contract the decision core, and everything built on top of it later,
// depends on. Every field has a concrete Day 2 purpose; see
// docs/PHASES/DAY-02-TASK-02.md, "Required Authorization Context".
//
// Trust note (G1, docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md):
// Evaluate performs no signature or token verification of its own — every
// field here is trusted exactly as given. Trust provenance is structural,
// not a field on this type: the only legitimate production caller is the
// ext_authz layer (internal/authz), which must populate Identity only
// from claims agentgateway has already validated
// (docs/SECURITY/PRODUCTION-INVARIANTS.md §1, §3). A Request built any
// other way (including via the G1 mock) is a test fixture, not an
// authenticated request.
type Request struct {
	// ExecutionID correlates this request through the decision and
	// (later) audit. It is preserved unchanged into the Result on every
	// path, including every failure path.
	ExecutionID string

	// WorkspaceID scopes the request. v1 is a single-tenant deployment
	// (docs/SECURITY/PRODUCTION-INVARIANTS.md §9), but every decision
	// still carries it for audit correlation and forward compatibility
	// with the latent tenant-aware schema (docs/PROJECT_DEFINITION.md §7).
	// It is not yet consumed by Cedar evaluation itself — no v1 policy
	// dimension depends on it.
	WorkspaceID string

	Identity       Identity
	Tool           ToolRef
	Classification ToolClassification

	// Arguments holds only explicitly declared, policy-relevant tool
	// argument attributes. There is no field on this struct through
	// which undeclared data can reach policy evaluation
	// (docs/SECURITY/PRODUCTION-INVARIANTS.md §2 item 6).
	Arguments map[string]AttributeValue
}

// Result is the outcome of evaluating a Request. Every field is populated
// deterministically for identical inputs and policy.
type Result struct {
	Decision Decision
	Reason   ReasonCode

	// Message is a human-readable detail, always authored by this
	// package. It must never be populated from cedar-go's own
	// Diagnostic.Errors/Reasons text or any other Cedar-internal string
	// — Cedar's diagnostic detail is not authorization semantics
	// (docs/PHASES/G1_WORKSTREAMS/01_GO_BACKEND_G1_DETAILED.md
	// AG-GO-G1-03). May be empty on ALLOW.
	Message string

	// PolicyVersion is the exact policy version/hash Cedar evaluated for
	// this decision. It is empty only when Cedar was never reached at
	// all (any pre-Cedar structural denial, or no policy is loaded) —
	// this package never fabricates or backfills it from "whatever is
	// currently active" (docs/SECURITY/PRODUCTION-INVARIANTS.md §4, §2
	// item 7).
	PolicyVersion string

	// ExecutionID is copied unchanged from the Request, on every path.
	ExecutionID string
}
