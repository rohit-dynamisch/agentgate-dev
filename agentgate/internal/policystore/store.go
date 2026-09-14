package policystore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// Store defines the persistence contract for versioned policies.
type Store interface {
	// CreatePolicy persists a new policy record. If the version already exists for the workspace,
	// it returns ErrAlreadyExists.
	CreatePolicy(ctx context.Context, p PolicyRecord) error

	// GetPolicy retrieves a specific policy by workspace and version hash.
	GetPolicy(ctx context.Context, workspaceID, version string) (PolicyRecord, error)

	// GetActivePolicy retrieves the single currently active policy for a workspace.
	// Returns ErrNoActivePolicy if none is active.
	GetActivePolicy(ctx context.Context, workspaceID string) (PolicyRecord, error)

	// ListPolicies returns all policy versions for a given workspace ordered by creation date descending.
	ListPolicies(ctx context.Context, workspaceID string) ([]PolicyRecord, error)

	// ActivatePolicy atomically transitions the specified version to active, marking
	// any previously active policy as historical. Returns the previous active version (or "" if none).
	ActivatePolicy(ctx context.Context, workspaceID, version string) (previousVersion string, err error)

	// RollbackPolicy atomically transitions the active policy back to a known historical or candidate version.
	// Returns the version that was active before rollback.
	RollbackPolicy(ctx context.Context, workspaceID, targetVersion string) (rolledBackFrom string, err error)

	// Close releases any database resources.
	Close() error
}

// ComputeVersion calculates the deterministic content-hash version (hex-encoded SHA-256)
// for Cedar policy source code, matching internal/policy.LoadFromBytes.
func ComputeVersion(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
