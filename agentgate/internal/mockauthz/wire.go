package mockauthz

import (
	"encoding/json"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
)

// The wire types below are this package's transport-specific JSON shape.
// They exist so decision.Request/decision.Result — the frozen,
// transport-free domain contract — never has to grow JSON struct tags or
// otherwise know about HTTP
// (docs/PHASES/G1_WORKSTREAMS/01_GO_BACKEND_G1_DETAILED.md AG-GO-G1-02,
// step 5: "Keep gateway transport types out of internal/decision").
//
// Field names and the full request/response shape are documented in
// docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md.

type evaluateRequestWire struct {
	ExecutionID    string                   `json:"execution_id"`
	WorkspaceID    string                   `json:"workspace_id"`
	Identity       identityWire             `json:"identity"`
	Tool           toolWire                 `json:"tool"`
	Classification classificationWire       `json:"classification"`
	Arguments      map[string]attributeWire `json:"arguments,omitempty"`
}

type identityWire struct {
	AgentID    string   `json:"agent_id"`
	OnBehalfOf string   `json:"on_behalf_of,omitempty"`
	Roles      []string `json:"roles,omitempty"`
}

type toolWire struct {
	BackendID string `json:"backend_id"`
	Name      string `json:"name"`
}

type classificationWire struct {
	Known bool   `json:"known"`
	Risk  string `json:"risk,omitempty"`
}

// attributeWire is a declared argument attribute: {"type": "string" |
// "int" | "bool", "value": <matching JSON value>}.
type attributeWire struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

// toDomain converts one wire attribute into a decision.AttributeValue. An
// unknown type, a value that does not match the declared type, or a
// literal JSON null becomes the zero-value AttributeValue —
// decision.Engine already treats that deterministically as
// ReasonMalformedRequest, so this package does not duplicate that
// validation; it simply lets an invalid attribute flow through to the
// real, already-tested rule.
//
// The explicit null check matters: json.Unmarshal(null, &nonPointer) is a
// documented no-op — it returns a nil error and leaves the target at its
// zero value — so without this check a caller-supplied `null` would
// silently become a *valid* zero attribute (e.g. IntAttr(0)) instead of
// being rejected, which was capable of flipping a real decision from DENY
// to ALLOW (a request supplying `null` for a required numeric argument
// evaluated identically to supplying 0). Found independently by code
// review, QA/Security, and Frontend/UI during the G1 checkpoint; fixed
// here as a wire-decoding correctness issue, not deferred to the
// per-tool argument-declaration work tracked as O-006.
func (a attributeWire) toDomain() decision.AttributeValue {
	if len(a.Value) == 0 || string(a.Value) == "null" {
		return decision.AttributeValue{}
	}
	switch a.Type {
	case "string":
		var s string
		if json.Unmarshal(a.Value, &s) == nil {
			return decision.StringAttr(s)
		}
	case "int":
		var n int64
		if json.Unmarshal(a.Value, &n) == nil {
			return decision.IntAttr(n)
		}
	case "bool":
		var b bool
		if json.Unmarshal(a.Value, &b) == nil {
			return decision.BoolAttr(b)
		}
	}
	return decision.AttributeValue{}
}

func (w evaluateRequestWire) toDomain() decision.Request {
	var args map[string]decision.AttributeValue
	if len(w.Arguments) > 0 {
		args = make(map[string]decision.AttributeValue, len(w.Arguments))
		for k, v := range w.Arguments {
			args[k] = v.toDomain()
		}
	}

	return decision.Request{
		ExecutionID: w.ExecutionID,
		WorkspaceID: w.WorkspaceID,
		Identity: decision.Identity{
			AgentID:    w.Identity.AgentID,
			OnBehalfOf: w.Identity.OnBehalfOf,
			Roles:      w.Identity.Roles,
		},
		Tool: decision.ToolRef{
			BackendID: w.Tool.BackendID,
			Name:      w.Tool.Name,
		},
		Classification: decision.ToolClassification{
			Known: w.Classification.Known,
			Risk:  w.Classification.Risk,
		},
		Arguments: args,
	}
}

// resultWire is the exact decision.Result contract shape as JSON. Field
// names are stable after G1 freeze (AG-GO-G1-06) — changing any of them
// requires the process in docs/PHASES/G1_WORKSTREAMS/00_G1_CHECKPOINT_REFERENCE.md's
// "Freeze rule".
//
// All five fields are always emitted, including Message/PolicyVersion as
// "" when empty — deliberately NOT `omitempty`. The frozen contract doc's
// own worked examples always show all five keys present; `omitempty`
// silently dropped "message" from every ALLOW response and "policy_version"
// from every pre-Cedar denial, which three independent G1 workstreams
// (Go Backend's own review, QA/Security, Frontend/UI) each separately
// noticed as contract drift. A stable, fully-explicit shape is required so
// downstream consumers can rely on "key present, possibly empty" rather
// than "key present only sometimes."
type resultWire struct {
	Decision      string `json:"decision"`
	Reason        string `json:"reason"`
	Message       string `json:"message"`
	PolicyVersion string `json:"policy_version"`
	ExecutionID   string `json:"execution_id"`
}

func toWireResult(r decision.Result) resultWire {
	return resultWire{
		Decision:      string(r.Decision),
		Reason:        string(r.Reason),
		Message:       r.Message,
		PolicyVersion: r.PolicyVersion,
		ExecutionID:   r.ExecutionID,
	}
}
