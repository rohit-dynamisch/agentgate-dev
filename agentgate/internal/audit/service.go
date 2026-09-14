package audit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Dynamisch-LLC/agentgate/internal/auditevents"
	"github.com/Dynamisch-LLC/agentgate/internal/decision"
)

// DecisionEvaluator evaluates a decision request against the active policy of a workspace.
type DecisionEvaluator interface {
	EvaluateWithActivePolicy(ctx context.Context, workspaceID string, req decision.Request) (decision.Result, error)
}

// ProvenanceProvider retrieves the authoritative policy version and content hash for a workspace.
type ProvenanceProvider interface {
	GetActiveProvenance(workspaceID string) (version string, hash string, err error)
}

// AuditedDecisionService intercepts every authorization decision and policy mutation,
// ensuring strict redaction, deterministic canonicalization, cryptographic hash chaining,
// and durable audit logging before an ALLOW decision can ever be returned to the client (O-002).
type AuditedDecisionService struct {
	evaluator DecisionEvaluator
	store     Store
	redactor  *Redactor
	mu        sync.Mutex
}

// NewAuditedDecisionService constructs an AuditedDecisionService wrapping an evaluator and audit store.
func NewAuditedDecisionService(evaluator DecisionEvaluator, store Store, redactor *Redactor) *AuditedDecisionService {
	if redactor == nil {
		redactor = NewRedactor("", nil)
	}
	return &AuditedDecisionService{
		evaluator: evaluator,
		store:     store,
		redactor:  redactor,
	}
}

// Store returns the underlying audit Store.
func (s *AuditedDecisionService) Store() Store {
	return s.store
}

// SetEvaluator sets or updates the decision evaluator.
func (s *AuditedDecisionService) SetEvaluator(evaluator DecisionEvaluator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evaluator = evaluator
}

// Evaluate evaluates the request and durably persists the audit record.
// Crucially, if evaluating produces ALLOW but the audit write fails, it FAILS CLOSED (returns DENY with ReasonEvaluationError).
func (s *AuditedDecisionService) Evaluate(ctx context.Context, req decision.Request) (decision.Result, error) {
	// 1. Evaluate request against active policy
	var res decision.Result
	if s.evaluator != nil {
		var err error
		res, err = s.evaluator.EvaluateWithActivePolicy(ctx, req.WorkspaceID, req)
		if err != nil {
			res = decision.Result{
				Decision:    decision.Deny,
				Reason:      decision.ReasonEvaluationError,
				Message:     fmt.Sprintf("policy evaluation error: %v", err),
				ExecutionID: req.ExecutionID,
			}
		}
	} else {
		noPolicy := decision.NewEngineWithoutPolicy()
		res = noPolicy.Evaluate(req)
	}

	// 2. Resolve authoritative policy version and policy hash
	var policyVersion string
	var policyHash string
	if res.PolicyVersion != "" {
		// Cedar was reached. In G1 decision.Engine, res.PolicyVersion is the SHA-256
		// hash of the evaluated policy source bytes (e.policy.Version()).
		policyHash = res.PolicyVersion
		policyVersion = res.PolicyVersion

		// If the evaluator implements ProvenanceProvider, retrieve the active policy version identifier
		if prov, ok := s.evaluator.(ProvenanceProvider); ok {
			if v, h, err := prov.GetActiveProvenance(req.WorkspaceID); err == nil && v != "" {
				policyVersion = v
				if h != "" {
					policyHash = h
				}
			}
		}
		// Reflect the active policy version identifier in the returned result
		res.PolicyVersion = policyVersion
	}

	// 3. Redact arguments prior to canonicalization or persistence
	redactedArgs := s.redactor.RedactArguments(req.Tool.Name, req.Arguments)

	// 4. Serialize append operation to maintain sequential chain
	s.mu.Lock()
	defer s.mu.Unlock()

	latest, err := s.store.GetLatestRecord(ctx, req.WorkspaceID)
	prevHash := GenesisHash
	if err == nil && latest != nil {
		prevHash = latest.RowHash
	}

	record := DecisionRecord{
		WorkspaceID:         req.WorkspaceID,
		ExecutionID:         req.ExecutionID,
		Timestamp:           time.Now().UTC(),
		EventType:           EventTypeDecision,
		Decision:            string(res.Decision),
		Reason:              string(res.Reason),
		PrincipalAgentID:    req.Identity.AgentID,
		PrincipalRoles:      req.Identity.Roles,
		PrincipalOnBehalfOf: req.Identity.OnBehalfOf,
		ToolBackendID:       req.Tool.BackendID,
		ToolName:            req.Tool.Name,
		ToolRisk:            req.Classification.Risk,
		PolicyVersion:       policyVersion,
		PolicyHash:          policyHash,
		RedactedArguments:   redactedArgs,
		PrevHash:            prevHash,
	}

	canonical, err := ComputeCanonicalPayload(record)
	if err != nil {
		// Canonicalization error: fail closed on ALLOW
		if res.Decision == decision.Allow {
			return decision.Result{
				Decision:      decision.Deny,
				Reason:        decision.ReasonEvaluationError,
				Message:       fmt.Sprintf("audit canonicalization error: %v", err),
				ExecutionID:   req.ExecutionID,
				PolicyVersion: res.PolicyVersion,
			}, nil
		}
		return res, nil
	}

	record.CanonicalPayload = string(canonical)
	record.RowHash = ComputeRowHash(prevHash, canonical)

	// 4. Durably persist
	_, appendErr := s.store.AppendDecision(ctx, record)
	if appendErr != nil {
		// CRITICAL INVARIANT (O-002): An ALLOW decision must NEVER be returned
		// without confirmed durable audit write. If the audit write fails, fail closed!
		if res.Decision == decision.Allow {
			return decision.Result{
				Decision:      decision.Deny,
				Reason:        decision.ReasonEvaluationError,
				Message:       fmt.Sprintf("audit failure: unable to durably record ALLOW decision: %v", appendErr),
				ExecutionID:   req.ExecutionID,
				PolicyVersion: res.PolicyVersion,
			}, nil
		}
		// If DENY: safe to return DENY since security is preserved
		return res, nil
	}

	return res, nil
}

// OnMutation records a policy mutation into the audit trail, implementing auditevents.MutationListener.
func (s *AuditedDecisionService) OnMutation(ctx context.Context, event auditevents.MutationEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	latest, err := s.store.GetLatestRecord(ctx, event.WorkspaceID)
	prevHash := GenesisHash
	if err == nil && latest != nil {
		prevHash = latest.RowHash
	}

	record := DecisionRecord{
		WorkspaceID:       event.WorkspaceID,
		ExecutionID:       event.CorrelationID,
		Timestamp:         event.Timestamp,
		EventType:         EventTypeMutation,
		Decision:          "MUTATION",
		Reason:            string(event.Action),
		PrincipalAgentID:  event.OperatorID,
		PolicyVersion:     event.NewVersion,
		PolicyHash:        event.PreviousVersion,
		RedactedArguments: make(map[string]string),
		PrevHash:          prevHash,
	}

	canonical, err := ComputeCanonicalPayload(record)
	if err == nil {
		record.CanonicalPayload = string(canonical)
		record.RowHash = ComputeRowHash(prevHash, canonical)
	}

	_, _ = s.store.AppendDecision(ctx, record)
}
