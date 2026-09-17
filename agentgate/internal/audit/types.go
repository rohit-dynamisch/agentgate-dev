package audit

import (
	"errors"
	"time"
)

// GenesisHash is the canonical predecessor hash for sequence number 1 of any workspace chain.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

const (
	// EventTypeDecision represents an authorization evaluation audit record.
	EventTypeDecision = "decision"

	// EventTypeMutation represents a policy lifecycle change audit record.
	EventTypeMutation = "mutation"
)

var (
	// ErrNotFound is returned when an audit record does not exist.
	ErrNotFound = errors.New("audit: record not found")

	// ErrInvalidChain is returned when hash chain validation fails.
	ErrInvalidChain = errors.New("audit: broken cryptographic hash chain")

	// ErrImmutableModification is returned when an illegal modification to historical audit records is attempted.
	ErrImmutableModification = errors.New("audit: records are append-only and cannot be modified")
)

// DecisionRecord contains all audit context captured during an authorization decision or mutation.
type DecisionRecord struct {
	WorkspaceID          string            `json:"workspace_id"`
	ExecutionID          string            `json:"execution_id"`
	Timestamp            time.Time         `json:"timestamp"`
	EventType            string            `json:"event_type"`
	Decision             string            `json:"decision"`
	Reason               string            `json:"reason"`
	PrincipalAgentID     string            `json:"principal_agent_id"`
	PrincipalRoles       []string          `json:"principal_roles"`
	PrincipalOnBehalfOf  string            `json:"principal_on_behalf_of"`
	ToolBackendID        string            `json:"tool_backend_id"`
	ToolName             string            `json:"tool_name"`
	ToolRisk             string            `json:"tool_risk"`
	PolicyVersion        string            `json:"policy_version"`
	PolicyHash           string            `json:"policy_hash"`
	RedactedArguments    map[string]string `json:"redacted_arguments"`
	CanonicalPayload     string            `json:"canonical_payload"`
	PrevHash             string            `json:"prev_hash"`
	RowHash              string            `json:"row_hash"`
}

// StoredRecord represents a fully persisted audit record in the database.
type StoredRecord struct {
	ID                   int64             `json:"id"`
	WorkspaceID          string            `json:"workspace_id"`
	SequenceNumber       int64             `json:"sequence_number"`
	ExecutionID          string            `json:"execution_id"`
	Timestamp            time.Time         `json:"timestamp"`
	EventType            string            `json:"event_type"`
	Decision             string            `json:"decision"`
	Reason               string            `json:"reason"`
	PrincipalAgentID     string            `json:"principal_agent_id"`
	PrincipalRoles       []string          `json:"principal_roles"`
	PrincipalOnBehalfOf  string            `json:"principal_on_behalf_of"`
	ToolBackendID        string            `json:"tool_backend_id"`
	ToolName             string            `json:"tool_name"`
	ToolRisk             string            `json:"tool_risk"`
	PolicyVersion        string            `json:"policy_version"`
	PolicyHash           string            `json:"policy_hash"`
	RedactedArguments    map[string]string `json:"redacted_arguments"`
	CanonicalPayload     string            `json:"canonical_payload"`
	PrevHash             string            `json:"prev_hash"`
	RowHash              string            `json:"row_hash"`
}
