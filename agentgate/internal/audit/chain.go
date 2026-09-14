package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

type canonicalRecord struct {
	WorkspaceID         string            `json:"workspace_id"`
	ExecutionID         string            `json:"execution_id"`
	Timestamp           string            `json:"timestamp"`
	EventType           string            `json:"event_type"`
	Decision            string            `json:"decision"`
	Reason              string            `json:"reason"`
	PrincipalAgentID    string            `json:"principal_agent_id"`
	PrincipalRoles      []string          `json:"principal_roles"`
	PrincipalOnBehalfOf string            `json:"principal_on_behalf_of"`
	ToolBackendID       string            `json:"tool_backend_id"`
	ToolName            string            `json:"tool_name"`
	ToolRisk            string            `json:"tool_risk"`
	PolicyVersion       string            `json:"policy_version"`
	PolicyHash          string            `json:"policy_hash"`
	RedactedArguments   map[string]string `json:"redacted_arguments"`
}

// ComputeCanonicalPayload serializes a decision or mutation record into a deterministic JSON byte slice.
func ComputeCanonicalPayload(record DecisionRecord) ([]byte, error) {
	roles := make([]string, len(record.PrincipalRoles))
	copy(roles, record.PrincipalRoles)
	sort.Strings(roles)

	args := make(map[string]string, len(record.RedactedArguments))
	for k, v := range record.RedactedArguments {
		args[k] = v
	}

	tsStr := ""
	if !record.Timestamp.IsZero() {
		tsStr = record.Timestamp.UTC().Format(time.RFC3339Nano)
	}

	eventType := record.EventType
	if eventType == "" {
		eventType = EventTypeDecision
	}

	cr := canonicalRecord{
		WorkspaceID:         record.WorkspaceID,
		ExecutionID:         record.ExecutionID,
		Timestamp:           tsStr,
		EventType:           eventType,
		Decision:            record.Decision,
		Reason:              record.Reason,
		PrincipalAgentID:    record.PrincipalAgentID,
		PrincipalRoles:      roles,
		PrincipalOnBehalfOf: record.PrincipalOnBehalfOf,
		ToolBackendID:       record.ToolBackendID,
		ToolName:            record.ToolName,
		ToolRisk:            record.ToolRisk,
		PolicyVersion:       record.PolicyVersion,
		PolicyHash:          record.PolicyHash,
		RedactedArguments:   args,
	}

	return json.Marshal(cr)
}

// ComputeRowHash calculates the SHA-256 hash linking the previous row hash with the canonical payload:
// row_hash = SHA256(prev_hash || canonical_payload)
func ComputeRowHash(prevHash string, canonicalPayload []byte) string {
	combined := append([]byte(prevHash), canonicalPayload...)
	sum := sha256.Sum256(combined)
	return hex.EncodeToString(sum[:])
}
