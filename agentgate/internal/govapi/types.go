package govapi

import "github.com/Dynamisch-LLC/agentgate/internal/policystore"

// CreateCandidateRequest represents the payload to submit a candidate policy.
type CreateCandidateRequest struct {
	Content     string `json:"content"`
	Description string `json:"description"`
}

// ValidateRequest represents the payload to validate raw Cedar syntax.
type ValidateRequest struct {
	Content string `json:"content"`
}

// ValidateResponse describes the validation outcome.
type ValidateResponse struct {
	Valid   bool     `json:"valid"`
	Version string   `json:"version,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

// ActivateResponse describes the outcome of an atomic activation.
type ActivateResponse struct {
	WorkspaceID     string `json:"workspace_id"`
	ActiveVersion   string `json:"active_version"`
	PreviousVersion string `json:"previous_version,omitempty"`
	ActivatedAt     string `json:"activated_at"`
}

// RollbackRequest targets a specific previous policy version.
type RollbackRequest struct {
	TargetVersion string `json:"target_version"`
}

// RollbackResponse describes the outcome of a rollback.
type RollbackResponse struct {
	WorkspaceID    string `json:"workspace_id"`
	ActiveVersion  string `json:"active_version"`
	RolledBackFrom string `json:"rolled_back_from"`
	ActivatedAt    string `json:"activated_at"`
}

// ListPoliciesResponse returns the list of all policy versions for a workspace.
type ListPoliciesResponse struct {
	WorkspaceID string                    `json:"workspace_id"`
	Policies    []policystore.PolicyRecord `json:"policies"`
}

// PreviewSample is one evaluation input for policy dry-run.
type PreviewSample struct {
	PrincipalID    string   `json:"principal_id"`
	PrincipalRoles []string `json:"principal_roles"`
	ResourceID     string   `json:"resource_id"`
	ResourceRisk   string   `json:"resource_risk"`
}

// PreviewRequest encapsulates sample inputs for dry-run evaluation.
type PreviewRequest struct {
	SampleRequests []PreviewSample `json:"sample_requests"`
}

// PreviewResult describes evaluation output for one sample request.
type PreviewResult struct {
	Allowed  bool `json:"allowed"`
	Matched  bool `json:"matched"`
	HadError bool `json:"had_error"`
}

// PreviewResponse aggregates preview evaluation results.
type PreviewResponse struct {
	Version string          `json:"version"`
	Results []PreviewResult `json:"results"`
}

// ErrorDetail contains structured error information.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// DryRunSampleRequest is one decision request input for dry-run comparison.
type DryRunSampleRequest struct {
	ExecutionID    string   `json:"execution_id"`
	PrincipalID    string   `json:"principal_id"`
	PrincipalRoles []string `json:"principal_roles"`
	OnBehalfOf     string   `json:"on_behalf_of,omitempty"`
	BackendID      string   `json:"backend_id"`
	ToolName       string   `json:"tool_name"`
	Risk           string   `json:"risk"`
}

// DryRunCompareRequest carries sample decision requests for comparison.
type DryRunCompareRequest struct {
	SampleRequests []DryRunSampleRequest `json:"sample_requests"`
}

// DryRunCompareResult pairs the active and candidate outcomes.
type DryRunCompareResult struct {
	ActiveDecision    string `json:"active_decision"`
	ActiveReason      string `json:"active_reason"`
	ActiveVersion     string `json:"active_policy_version"`
	CandidateDecision string `json:"candidate_decision"`
	CandidateReason   string `json:"candidate_reason"`
	CandidateVersion  string `json:"candidate_policy_version"`
	Changed           bool   `json:"changed"`
}

// DryRunCompareResponse aggregates comparison results.
type DryRunCompareResponse struct {
	CandidateVersion string                `json:"candidate_version"`
	Results          []DryRunCompareResult  `json:"results"`
}
