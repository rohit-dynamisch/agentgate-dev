package governanceintegration

import (
	"context"

	"github.com/Dynamisch-LLC/agentgate/internal/decision"
	"github.com/Dynamisch-LLC/agentgate/internal/policy"
	"github.com/Dynamisch-LLC/agentgate/internal/policymanager"
)

// DryRunComparison pairs the evaluation result of a request against
// the currently active policy and a candidate policy.
type DryRunComparison struct {
	Request         decision.Request
	ActiveResult    decision.Result
	CandidateResult decision.Result
}

// GovernanceDecisionService connects the G3 policy lifecycle to the G1
// decision engine, proving that persisted policy changes actually affect
// live authorization decisions.
type GovernanceDecisionService struct {
	manager *policymanager.Manager
}

// NewGovernanceDecisionService constructs the integration service.
func NewGovernanceDecisionService(mgr *policymanager.Manager) *GovernanceDecisionService {
	return &GovernanceDecisionService{manager: mgr}
}

// EvaluateWithActivePolicy evaluates a decision request against the currently
// active policy engine for the workspace. If no active policy exists, the
// request is denied with ReasonNoPolicyLoaded.
func (s *GovernanceDecisionService) EvaluateWithActivePolicy(_ context.Context, workspaceID string, req decision.Request) (decision.Result, error) {
	eng, err := s.manager.GetActiveEngine(workspaceID)
	if err != nil {
		// No active engine → deny with no-policy-loaded
		noPolicy := decision.NewEngineWithoutPolicy()
		return noPolicy.Evaluate(req), nil
	}
	de := decision.NewEngineWithPolicy(eng)
	return de.Evaluate(req), nil
}

// DryRunCompare evaluates each request against both the active and candidate
// policies, returning paired comparison results. The active policy is NOT
// mutated by this operation — dry-run isolation is guaranteed.
func (s *GovernanceDecisionService) DryRunCompare(ctx context.Context, workspaceID, candidateVersion string, requests []decision.Request) ([]DryRunComparison, error) {
	// Load candidate engine from store
	candidateRec, err := s.manager.Store().GetPolicy(ctx, workspaceID, candidateVersion)
	if err != nil {
		return nil, err
	}
	candidateEng, err := policy.LoadFromBytes([]byte(candidateRec.Content))
	if err != nil {
		return nil, err
	}

	// Get active engine (may not exist)
	activeEng, activeErr := s.manager.GetActiveEngine(workspaceID)

	comparisons := make([]DryRunComparison, 0, len(requests))
	for _, req := range requests {
		var activeResult decision.Result
		if activeErr != nil {
			noPolicy := decision.NewEngineWithoutPolicy()
			activeResult = noPolicy.Evaluate(req)
		} else {
			de := decision.NewEngineWithPolicy(activeEng)
			activeResult = de.Evaluate(req)
		}

		candidateDe := decision.NewEngineWithPolicy(candidateEng)
		candidateResult := candidateDe.Evaluate(req)

		comparisons = append(comparisons, DryRunComparison{
			Request:         req,
			ActiveResult:    activeResult,
			CandidateResult: candidateResult,
		})
	}
	return comparisons, nil
}
