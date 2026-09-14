package audit

import (
	"context"
	"fmt"
)

// VerificationResult summarizes the cryptographic integrity check of a workspace's audit log.
type VerificationResult struct {
	Valid          bool   `json:"valid"`
	TotalRecords   int64  `json:"total_records"`
	BrokenSequence int64  `json:"broken_sequence,omitempty"`
	Details        string `json:"details,omitempty"`
}

// ChainVerifier independently inspects audit trails for tampering, deletion, or linkage corruption.
type ChainVerifier struct{}

// NewChainVerifier constructs a new audit chain verifier.
func NewChainVerifier() *ChainVerifier {
	return &ChainVerifier{}
}

// VerifyWorkspace validates all audit records for a workspace sequentially from Genesis to the latest record.
func (v *ChainVerifier) VerifyWorkspace(ctx context.Context, store Store, workspaceID string) (*VerificationResult, error) {
	// ListRecords returns in descending sequence order (newest first)
	descRecords, err := store.ListRecords(ctx, workspaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("verifier: retrieve records for workspace %q: %w", workspaceID, err)
	}

	total := int64(len(descRecords))
	if total == 0 {
		return &VerificationResult{
			Valid:        true,
			TotalRecords: 0,
		}, nil
	}

	// Reverse to process chronologically (seq 1 to total)
	records := make([]StoredRecord, total)
	for i := range descRecords {
		records[int64(i)] = descRecords[total-1-int64(i)]
	}

	expectedPrev := GenesisHash

	for i, rec := range records {
		expectedSeq := int64(i + 1)

		// 1. Verify monotonic sequence without gaps or duplicates
		if rec.SequenceNumber != expectedSeq {
			return &VerificationResult{
				Valid:          false,
				TotalRecords:   total,
				BrokenSequence: rec.SequenceNumber,
				Details:        fmt.Sprintf("sequence gap or duplication: expected sequence %d, got %d", expectedSeq, rec.SequenceNumber),
			}, nil
		}

		// 2. Verify previous hash linkage
		if rec.PrevHash != expectedPrev {
			return &VerificationResult{
				Valid:          false,
				TotalRecords:   total,
				BrokenSequence: rec.SequenceNumber,
				Details:        fmt.Sprintf("broken hash chain linkage at seq %d: expected prev_hash %s, got %s", rec.SequenceNumber, expectedPrev, rec.PrevHash),
			}, nil
		}

		// 3. Recompute canonical payload and detect in-place column modification
		dRec := DecisionRecord{
			WorkspaceID:         rec.WorkspaceID,
			ExecutionID:         rec.ExecutionID,
			Timestamp:           rec.Timestamp,
			EventType:           rec.EventType,
			Decision:            rec.Decision,
			Reason:              rec.Reason,
			PrincipalAgentID:    rec.PrincipalAgentID,
			PrincipalRoles:      rec.PrincipalRoles,
			PrincipalOnBehalfOf: rec.PrincipalOnBehalfOf,
			ToolBackendID:       rec.ToolBackendID,
			ToolName:            rec.ToolName,
			ToolRisk:            rec.ToolRisk,
			PolicyVersion:       rec.PolicyVersion,
			PolicyHash:          rec.PolicyHash,
			RedactedArguments:   rec.RedactedArguments,
		}

		recomputedCanonical, err := ComputeCanonicalPayload(dRec)
		if err != nil {
			return nil, fmt.Errorf("verifier: canonicalize record seq %d: %w", rec.SequenceNumber, err)
		}

		if string(recomputedCanonical) != rec.CanonicalPayload {
			return &VerificationResult{
				Valid:          false,
				TotalRecords:   total,
				BrokenSequence: rec.SequenceNumber,
				Details:        fmt.Sprintf("payload tampering detected at seq %d: stored canonical payload does not match row contents", rec.SequenceNumber),
			}, nil
		}

		// 4. Verify cryptographic row hash
		expectedRowHash := ComputeRowHash(rec.PrevHash, recomputedCanonical)
		if rec.RowHash != expectedRowHash {
			return &VerificationResult{
				Valid:          false,
				TotalRecords:   total,
				BrokenSequence: rec.SequenceNumber,
				Details:        fmt.Sprintf("row hash corruption detected at seq %d: expected %s, got %s", rec.SequenceNumber, expectedRowHash, rec.RowHash),
			}, nil
		}

		expectedPrev = rec.RowHash
	}

	return &VerificationResult{
		Valid:        true,
		TotalRecords: total,
	}, nil
}
