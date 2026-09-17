package audit

import (
	"context"

	"github.com/Dynamisch-LLC/agentgate/internal/auditevents"
)

// Store defines the append-only storage interface for audit records.
type Store interface {
	// AppendDecision durably records an authorization decision event.
	AppendDecision(ctx context.Context, record DecisionRecord) (*StoredRecord, error)

	// AppendMutation durably records a policy lifecycle mutation event.
	AppendMutation(ctx context.Context, event auditevents.MutationEvent, prevHash string) (*StoredRecord, error)

	// GetLatestRecord retrieves the highest-sequenced audit record for a workspace.
	// Returns ErrNotFound if no records exist for the workspace.
	GetLatestRecord(ctx context.Context, workspaceID string) (*StoredRecord, error)

	// ListRecords retrieves audit records for a workspace in descending sequence order, up to limit.
	ListRecords(ctx context.Context, workspaceID string, limit int) ([]StoredRecord, error)

	// GetRecordBySequence retrieves a specific record by its workspace and sequence number.
	// Returns ErrNotFound if the record does not exist.
	GetRecordBySequence(ctx context.Context, workspaceID string, seq int64) (*StoredRecord, error)

	// Migrate applies required database schema migrations.
	Migrate(ctx context.Context) error

	// Close releases any allocated resources.
	Close() error
}
