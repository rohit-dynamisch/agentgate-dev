package govapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/governanceintegration"
	"github.com/Dynamisch-LLC/agentgate/internal/policy"
	"github.com/Dynamisch-LLC/agentgate/internal/policymanager"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
)

// Handler serves policy governance REST endpoints.
type Handler struct {
	manager        *policymanager.Manager
	govIntegration *governanceintegration.GovernanceDecisionService
	adminToken     string
}

// NewHandler constructs a new Handler.
// govIntegration may be nil if dry-run compare is not needed (backward compatible).
func NewHandler(manager *policymanager.Manager, adminToken string, govIntegration *governanceintegration.GovernanceDecisionService) *Handler {
	return &Handler{
		manager:        manager,
		govIntegration: govIntegration,
		adminToken:     adminToken,
	}
}

// RegisterRoutes registers the policy governance routes on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	protect := func(fn http.HandlerFunc) http.HandlerFunc {
		return AdminAuthMiddleware(h.adminToken, fn)
	}

	mux.HandleFunc("POST /api/v1/workspaces/{workspace_id}/policies/validate", protect(h.handleValidate))
	mux.HandleFunc("POST /api/v1/workspaces/{workspace_id}/policies/rollback", protect(h.handleRollback))
	mux.HandleFunc("POST /api/v1/workspaces/{workspace_id}/policies/{version}/activate", protect(h.handleActivate))
	mux.HandleFunc("POST /api/v1/workspaces/{workspace_id}/policies/{version}/preview", protect(h.handlePreview))
	mux.HandleFunc("POST /api/v1/workspaces/{workspace_id}/policies/{version}/dryrun", protect(h.handleDryRunCompare))
	mux.HandleFunc("GET /api/v1/workspaces/{workspace_id}/policies/{version}", protect(h.handleGetPolicy))
	mux.HandleFunc("GET /api/v1/workspaces/{workspace_id}/policies", protect(h.handleListPolicies))
	mux.HandleFunc("POST /api/v1/workspaces/{workspace_id}/policies", protect(h.handleCreateCandidate))
}

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "malformed request body")
		return
	}

	version, err := h.manager.Validate(req.Content)
	if err != nil {
		writeJSON(w, http.StatusOK, ValidateResponse{
			Valid:  false,
			Errors: []string{err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, ValidateResponse{
		Valid:   true,
		Version: version,
	})
}

func (h *Handler) handleCreateCandidate(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id required")
		return
	}

	var req CreateCandidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "malformed request body")
		return
	}

	rec, err := h.manager.CreateCandidate(r.Context(), workspaceID, req.Content, req.Description)
	if err != nil {
		if errors.Is(err, policymanager.ErrInvalidPolicy) {
			writeError(w, http.StatusBadRequest, "INVALID_POLICY", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create candidate")
		return
	}

	writeJSON(w, http.StatusCreated, rec)
}

func (h *Handler) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id required")
		return
	}

	policies, err := h.manager.Store().ListPolicies(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list policies")
		return
	}

	writeJSON(w, http.StatusOK, ListPoliciesResponse{
		WorkspaceID: workspaceID,
		Policies:    policies,
	})
}

func (h *Handler) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	version := r.PathValue("version")
	if workspaceID == "" || version == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id and version required")
		return
	}

	rec, err := h.manager.Store().GetPolicy(r.Context(), workspaceID, version)
	if err != nil {
		if errors.Is(err, policystore.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "policy version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get policy")
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func (h *Handler) handleActivate(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	version := r.PathValue("version")
	if workspaceID == "" || version == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id and version required")
		return
	}

	rec, err := h.manager.Activate(r.Context(), workspaceID, version)
	if err != nil {
		if errors.Is(err, policystore.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "policy version not found")
			return
		}
		if errors.Is(err, policymanager.ErrInvalidPolicy) {
			writeError(w, http.StatusBadRequest, "INVALID_POLICY", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to activate policy")
		return
	}

	activatedAt := ""
	if rec.ActivatedAt != nil {
		activatedAt = rec.ActivatedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, ActivateResponse{
		WorkspaceID:   workspaceID,
		ActiveVersion: rec.Version,
		ActivatedAt:   activatedAt,
	})
}

func (h *Handler) handleRollback(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id required")
		return
	}

	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "malformed request body")
		return
	}

	prevActive, _ := h.manager.Store().GetActivePolicy(r.Context(), workspaceID)

	rec, err := h.manager.Rollback(r.Context(), workspaceID, req.TargetVersion)
	if err != nil {
		if errors.Is(err, policystore.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "target policy version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to rollback policy")
		return
	}

	activatedAt := ""
	if rec.ActivatedAt != nil {
		activatedAt = rec.ActivatedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, RollbackResponse{
		WorkspaceID:    workspaceID,
		ActiveVersion:  rec.Version,
		RolledBackFrom: prevActive.Version,
		ActivatedAt:    activatedAt,
	})
}

func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	version := r.PathValue("version")
	if workspaceID == "" || version == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id and version required")
		return
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "malformed request body")
		return
	}

	inputs := make([]policy.EvalInput, 0, len(req.SampleRequests))
	for _, sample := range req.SampleRequests {
		inputs = append(inputs, policy.EvalInput{
			PrincipalID:    sample.PrincipalID,
			PrincipalRoles: sample.PrincipalRoles,
			ResourceID:     sample.ResourceID,
			ResourceRisk:   sample.ResourceRisk,
		})
	}

	outputs, err := h.manager.Preview(r.Context(), workspaceID, version, inputs)
	if err != nil {
		if errors.Is(err, policystore.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "policy version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to preview policy")
		return
	}

	results := make([]PreviewResult, 0, len(outputs))
	for _, out := range outputs {
		results = append(results, PreviewResult{
			Allowed:  out.Allowed,
			Matched:  out.Matched,
			HadError: out.HadError,
		})
	}

	writeJSON(w, http.StatusOK, PreviewResponse{
		Version: version,
		Results: results,
	})
}

func (h *Handler) handleDryRunCompare(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspace_id")
	version := r.PathValue("version")
	if workspaceID == "" || version == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "workspace_id and version required")
		return
	}

	if h.govIntegration == nil {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "governance integration not configured")
		return
	}

	var req DryRunCompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "malformed request body")
		return
	}

	decRequests := make([]decision.Request, 0, len(req.SampleRequests))
	for _, sample := range req.SampleRequests {
		decRequests = append(decRequests, decision.Request{
			ExecutionID: sample.ExecutionID,
			WorkspaceID: workspaceID,
			Identity: decision.Identity{
				AgentID:    sample.PrincipalID,
				Roles:      sample.PrincipalRoles,
				OnBehalfOf: sample.OnBehalfOf,
			},
			Tool: decision.ToolRef{
				BackendID: sample.BackendID,
				Name:      sample.ToolName,
			},
			Classification: decision.ToolClassification{
				Known: true,
				Risk:  sample.Risk,
			},
		})
	}

	comparisons, err := h.govIntegration.DryRunCompare(r.Context(), workspaceID, version, decRequests)
	if err != nil {
		if errors.Is(err, policystore.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "policy version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to perform dry-run comparison")
		return
	}

	results := make([]DryRunCompareResult, 0, len(comparisons))
	for _, comp := range comparisons {
		changed := comp.ActiveResult.Decision != comp.CandidateResult.Decision ||
			comp.ActiveResult.Reason != comp.CandidateResult.Reason

		results = append(results, DryRunCompareResult{
			ActiveDecision:    string(comp.ActiveResult.Decision),
			ActiveReason:      string(comp.ActiveResult.Reason),
			ActiveVersion:     comp.ActiveResult.PolicyVersion,
			CandidateDecision: string(comp.CandidateResult.Decision),
			CandidateReason:   string(comp.CandidateResult.Reason),
			CandidateVersion:  comp.CandidateResult.PolicyVersion,
			Changed:           changed,
		})
	}

	writeJSON(w, http.StatusOK, DryRunCompareResponse{
		CandidateVersion: version,
		Results:          results,
	})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: msg,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
