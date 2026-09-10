package decision

import (
	"fmt"
	"strings"

	"github.com/Dynamisch-LLC/agentgate/internal/policy"
)

// Engine is the decision core. It depends only on internal/policy's
// in-process Cedar boundary — no agentgateway, PostgreSQL, or network I/O
// — so it is independently testable (docs/PHASES/DAY-02-TASK-02.md).
type Engine struct {
	policy *policy.Engine // nil means no policy loaded
}

// NewEngine constructs a decision-core Engine from raw Cedar policy bytes.
// Policy bytes are a test/dev fixture for Day 2 — Day 4 replaces the
// source with versioned PostgreSQL rows without changing this function's
// contract or this package's evaluation boundary
// (docs/PHASES/AGENTGATE_V1_15_DAY_PRODUCTION_PLAN.md Day 4).
func NewEngine(policyBytes []byte) (*Engine, error) {
	pe, err := policy.LoadFromBytes(policyBytes)
	if err != nil {
		return nil, fmt.Errorf("decision: load policy: %w", err)
	}
	return &Engine{policy: pe}, nil
}

// NewEngineWithoutPolicy constructs an Engine with no policy loaded. Every
// request against it denies with ReasonNoPolicyLoaded — the explicit
// representation of "startup with no valid policy must not authorize
// requests" (docs/SECURITY/PRODUCTION-INVARIANTS.md §4).
func NewEngineWithoutPolicy() *Engine {
	return &Engine{}
}

// Evaluate runs one authorization decision. It is deterministic for
// identical inputs and policy, and never returns Allow on any failure
// path.
func (e *Engine) Evaluate(req Request) Result {
	if reason, msg, ok := validateRequestShape(req); !ok {
		return deny(req, reason, msg, "")
	}
	if reason, msg, ok := validateIdentity(req.Identity); !ok {
		return deny(req, reason, msg, "")
	}
	if !req.Classification.Known {
		return deny(req, ReasonUnknownTool, "tool is not classified", "")
	}
	if e == nil || e.policy == nil {
		return deny(req, ReasonNoPolicyLoaded, "no policy loaded", "")
	}

	evalIn, err := buildEvalInput(req)
	if err != nil {
		// An invalid/zero-value AttributeValue slipped into the request.
		// Treat it as malformed rather than letting it reach Cedar.
		return deny(req, ReasonMalformedRequest, err.Error(), "")
	}

	out := e.policy.Evaluate(evalIn)
	version := e.policy.Version()

	if out.HadError {
		return Result{
			Decision:      Deny,
			Reason:        ReasonEvaluationError,
			Message:       "cedar policy evaluation reported an error",
			PolicyVersion: version,
			ExecutionID:   req.ExecutionID,
		}
	}
	if out.Allowed {
		return Result{
			Decision:      Allow,
			Reason:        ReasonPolicyAllow,
			PolicyVersion: version,
			ExecutionID:   req.ExecutionID,
		}
	}
	if out.Matched {
		return Result{
			Decision:      Deny,
			Reason:        ReasonPolicyDeny,
			Message:       "an explicit policy denied this request",
			PolicyVersion: version,
			ExecutionID:   req.ExecutionID,
		}
	}
	return Result{
		Decision:      Deny,
		Reason:        ReasonNoMatchingPolicy,
		Message:       "no policy matched this request; deny-by-default",
		PolicyVersion: version,
		ExecutionID:   req.ExecutionID,
	}
}

// deny builds a Deny Result while always preserving ExecutionID, including
// on every pre-Cedar structural failure path.
func deny(req Request, reason ReasonCode, msg, version string) Result {
	return Result{
		Decision:      Deny,
		Reason:        reason,
		Message:       msg,
		PolicyVersion: version,
		ExecutionID:   req.ExecutionID,
	}
}

func validateRequestShape(req Request) (ReasonCode, string, bool) {
	switch {
	case req.ExecutionID == "":
		return ReasonMalformedRequest, "execution id is required", false
	case req.WorkspaceID == "":
		return ReasonMalformedRequest, "workspace id is required", false
	case req.Tool.BackendID == "":
		return ReasonMalformedRequest, "tool backend id is required", false
	case req.Tool.Name == "":
		return ReasonMalformedRequest, "tool name is required", false
	}
	return "", "", true
}

func validateIdentity(id Identity) (ReasonCode, string, bool) {
	switch {
	case id.AgentID == "":
		return ReasonInvalidIdentity, "agent id is missing", false
	case len(id.Roles) == 0:
		return ReasonInvalidIdentity, "identity has no roles", false
	}
	for _, r := range id.Roles {
		if strings.TrimSpace(r) == "" {
			return ReasonInvalidIdentity, "identity contains an empty role", false
		}
	}
	if id.OnBehalfOf != "" && id.OnBehalfOf == id.AgentID {
		return ReasonInvalidIdentity, "on_behalf_of must not equal agent id", false
	}
	return "", "", true
}

// buildEvalInput translates the Request's declared Arguments into the
// policy package's primitive attribute vocabulary. This is the one place
// this package touches internal/policy's types.
func buildEvalInput(req Request) (policy.EvalInput, error) {
	args := make(map[string]policy.AttrValue, len(req.Arguments))
	for k, v := range req.Arguments {
		if !v.valid() {
			return policy.EvalInput{}, fmt.Errorf("argument %q: invalid or zero-value attribute", k)
		}
		args[k] = toPolicyAttr(v)
	}

	return policy.EvalInput{
		PrincipalID:    req.Identity.AgentID,
		PrincipalRoles: append([]string(nil), req.Identity.Roles...),
		OnBehalfOf:     req.Identity.OnBehalfOf,
		ResourceID:     req.Tool.BackendID + "/" + req.Tool.Name,
		ResourceRisk:   req.Classification.Risk,
		Context:        args,
	}, nil
}

func toPolicyAttr(a AttributeValue) policy.AttrValue {
	switch a.kind {
	case attributeKindString:
		return policy.StringAttr(a.str)
	case attributeKindInt:
		return policy.IntAttr(a.num)
	case attributeKindBool:
		return policy.BoolAttr(a.b)
	default:
		// Unreachable: callers only reach here after v.valid() passed.
		return policy.AttrValue{}
	}
}
