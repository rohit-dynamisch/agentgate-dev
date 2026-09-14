// Package mockauthz exposes AgentGate's frozen G1 decision-core contract
// (internal/decision) over a minimal JSON/HTTP transport, so Gateway/MCP
// and QA/Security can integrate against the real contract before the
// production ext_authz gRPC service exists
// (docs/PHASES/G1_WORKSTREAMS/01_GO_BACKEND_G1_DETAILED.md, AG-GO-G1-04).
//
// This is not a fake/stub decision engine: it wraps the real
// internal/decision.Engine (and therefore the real internal/policy Cedar
// boundary), so every response is a genuine, deterministic Cedar
// decision — never hand-rolled mock logic
// (docs/PHASES/G1_WORKSTREAMS/01_GO_BACKEND_G1_DETAILED.md AG-GO-G1-04,
// step 3: "Do not duplicate Cedar evaluation in the mock").
//
// This package (and cmd/g1-mock-authz, which runs it) must never be
// reached from cmd/agentgate's production startup path. It is a
// temporary parallelization tool
// (docs/PHASES/AGENTGATE_V1_10_DAY_PARALLEL_TEAM_EXECUTION_PLAN.md,
// Integration Rule 2) — every production caller of the decision core is
// the real ext_authz layer (internal/authz, Day 3/8), never this package.
package mockauthz

import (
	"encoding/json"
	"net/http"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
)

// Handler serves POST /evaluate: decode a JSON request in the wire shape
// documented in docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md, run
// it through the real decision core, and respond with the exact
// decision.Result contract shape as JSON.
//
// A request body that fails to decode at all is a transport-level error
// (HTTP 400) — it never became a decision.Request, so there is no
// decision.Result to report. A body that decodes but is semantically
// incomplete (e.g. a missing agent id) reaches the real decision core and
// comes back as an ordinary HTTP 200 response whose body is a DENY
// Result — exactly like every other decision, malformed or not. This
// mirrors the real ext_authz boundary, where the decision is data in the
// response, not conveyed by the transport status code.
type Handler struct {
	engine *decision.Engine
}

// NewHandler wraps an already-constructed decision.Engine — typically one
// loaded from internal/fixturepolicy.CedarSource.
func NewHandler(engine *decision.Engine) *Handler {
	return &Handler{engine: engine}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/evaluate" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"method not allowed, use POST"}`, http.StatusMethodNotAllowed)
		return
	}

	var wire evaluateRequestWire
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		http.Error(w,
			`{"error":"request body could not be decoded as the evaluate-request contract: `+jsonEscape(err.Error())+`"}`,
			http.StatusBadRequest)
		return
	}

	result := h.engine.Evaluate(wire.toDomain())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toWireResult(result))
}

// jsonEscape is a minimal helper so a decode error's message (which may
// contain quotes) doesn't break the hand-written JSON error body above.
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	if len(b) < 2 {
		return s
	}
	return string(b[1 : len(b)-1]) // strip the surrounding quotes json.Marshal adds
}
