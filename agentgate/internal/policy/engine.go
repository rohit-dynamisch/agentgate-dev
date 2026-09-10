package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	cedar "github.com/cedar-policy/cedar-go"
)

// Fixed Cedar entity-model constants for v1. AgentGate models exactly one
// action ("InvokeTool") — the single operation this product governs — with
// the actual backend/tool identity carried on the resource entity. This
// matches the rule shape in docs/PROJECT_DEFINITION.md §6
// ("role=reader → tools where risk=read"): role membership is principal
// group membership, and risk is a resource attribute.
const (
	principalEntityType cedar.EntityType = "AgentGate::Agent"
	roleEntityType      cedar.EntityType = "AgentGate::Role"
	actionEntityType    cedar.EntityType = "AgentGate::Action"
	resourceEntityType  cedar.EntityType = "AgentGate::Tool"

	invokeToolActionID cedar.String = "InvokeTool"
)

// attrKind is the closed set of value kinds AttrValue can hold.
type attrKind uint8

const (
	attrKindInvalid attrKind = iota
	attrKindString
	attrKindInt
	attrKindBool
)

// AttrValue is the small, closed set of value kinds this boundary accepts
// into a Cedar evaluation context (resource risk aside, which is always a
// plain string). It exists so callers above this package never need to
// import cedar-go's own value types — the "narrow application-owned
// boundary" required by docs/PHASES/DAY-02-TASK-02.md. A zero-value
// AttrValue (unset kind) is never valid and is rejected by Evaluate.
type AttrValue struct {
	kind attrKind
	str  string
	num  int64
	b    bool
}

// StringAttr, IntAttr and BoolAttr are the only ways to construct a valid
// AttrValue.
func StringAttr(v string) AttrValue { return AttrValue{kind: attrKindString, str: v} }
func IntAttr(v int64) AttrValue     { return AttrValue{kind: attrKindInt, num: v} }
func BoolAttr(v bool) AttrValue     { return AttrValue{kind: attrKindBool, b: v} }

func (a AttrValue) cedarValue() (cedar.Value, bool) {
	switch a.kind {
	case attrKindString:
		return cedar.String(a.str), true
	case attrKindInt:
		return cedar.Long(a.num), true
	case attrKindBool:
		return cedar.Boolean(a.b), true
	default:
		return nil, false
	}
}

// EvalInput is the primitive, Cedar-free description of one authorization
// evaluation. The caller (internal/decision) is responsible for every
// invariant this package assumes: ResourceRisk non-empty, PrincipalID
// non-empty, PrincipalRoles non-empty. This package does not re-validate
// those — it is the Cedar boundary, not the decision core's identity/tool
// validation.
type EvalInput struct {
	PrincipalID    string
	PrincipalRoles []string
	OnBehalfOf     string // "" means absent
	ResourceID     string
	ResourceRisk   string
	Context        map[string]AttrValue
}

// EvalOutput is the primitive result of one evaluation.
type EvalOutput struct {
	// Allowed is Cedar's own decision (Allow/Deny).
	Allowed bool

	// Matched is true when at least one policy contributed to the
	// decision (Cedar's diagnostic reasons were non-empty) — this
	// distinguishes an explicit forbid/permit match from Cedar's
	// structural default-deny (no policy matched at all).
	Matched bool

	// HadError is true when Cedar reported an internal evaluation error
	// for one or more policies while producing this decision (e.g. a
	// policy referenced a context attribute that was not present without
	// guarding it with `has`). The caller must never trust Allowed when
	// HadError is true (docs/SECURITY/PRODUCTION-INVARIANTS.md §2 item 4,
	// §8).
	HadError bool
}

// Engine holds one loaded, validated Cedar policy set and its
// content-addressed version.
type Engine struct {
	set     *cedar.PolicySet
	version string
}

// LoadFromBytes parses and validates raw Cedar policy source, returning an
// error for syntactically invalid policy. The version is the hex-encoded
// SHA-256 of the exact source bytes — content-addressed, so it is stable
// regardless of where the bytes came from (file today, a PostgreSQL row
// from Day 4 onward) (docs/PROJECT_DEFINITION.md §6, §12).
func LoadFromBytes(src []byte) (*Engine, error) {
	set, err := cedar.NewPolicySetFromBytes("policy.cedar", src)
	if err != nil {
		return nil, fmt.Errorf("policy: parse: %w", err)
	}
	sum := sha256.Sum256(src)
	return &Engine{set: set, version: hex.EncodeToString(sum[:])}, nil
}

// Version is the content-hash identity of the loaded policy — the exact
// value every decision evaluated against this Engine must carry. Calling
// Version on a nil Engine returns "" (no policy loaded).
func (e *Engine) Version() string {
	if e == nil {
		return ""
	}
	return e.version
}

// Evaluate runs one authorization decision against the loaded policy set.
// Evaluate on a nil Engine or an Engine with no policy set is a caller
// error (internal/decision never calls it in that state — see
// docs/SECURITY/PRODUCTION-INVARIANTS.md §4's "no valid policy => no
// authorization" invariant, enforced one layer up).
func (e *Engine) Evaluate(in EvalInput) EvalOutput {
	principal := cedar.NewEntityUID(principalEntityType, cedar.String(in.PrincipalID))
	resource := cedar.NewEntityUID(resourceEntityType, cedar.String(in.ResourceID))

	parents := make([]cedar.EntityUID, 0, len(in.PrincipalRoles))
	for _, role := range in.PrincipalRoles {
		parents = append(parents, cedar.NewEntityUID(roleEntityType, cedar.String(role)))
	}

	principalAttrs := cedar.RecordMap{}
	if in.OnBehalfOf != "" {
		principalAttrs["on_behalf_of"] = cedar.String(in.OnBehalfOf)
	}

	resourceAttrs := cedar.RecordMap{
		"risk": cedar.String(in.ResourceRisk),
	}

	entities := cedar.EntityMap{
		principal: {
			UID:        principal,
			Parents:    cedar.NewEntityUIDSet(parents...),
			Attributes: cedar.NewRecord(principalAttrs),
		},
		resource: {
			UID:        resource,
			Attributes: cedar.NewRecord(resourceAttrs),
		},
	}

	ctxMap := cedar.RecordMap{}
	for k, v := range in.Context {
		cv, ok := v.cedarValue()
		if !ok {
			// The decision package guarantees only valid AttrValues reach
			// here; treating an impossible zero-value as an evaluation
			// error (rather than silently dropping it) keeps this
			// package fail-closed even against its own caller's bugs.
			return EvalOutput{HadError: true}
		}
		ctxMap[cedar.String(k)] = cv
	}

	req := cedar.Request{
		Principal: principal,
		Action:    cedar.NewEntityUID(actionEntityType, invokeToolActionID),
		Resource:  resource,
		Context:   cedar.NewRecord(ctxMap),
	}

	decision, diag := e.set.IsAuthorized(entities, req)

	return EvalOutput{
		Allowed:  decision == cedar.Allow,
		Matched:  len(diag.Reasons) > 0,
		HadError: len(diag.Errors) > 0,
	}
}
