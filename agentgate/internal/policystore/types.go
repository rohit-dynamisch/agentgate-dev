package policystore

import (
	"errors"
	"time"
)

// PolicyState represents the lifecycle status of a persisted policy.
type PolicyState string

const (
	StateCandidate  PolicyState = "candidate"
	StateActive     PolicyState = "active"
	StateHistorical PolicyState = "historical"
)

// PolicyRecord is the canonical persisted record for a versioned policy.
type PolicyRecord struct {
	ID          int64       `json:"id,omitempty"`
	WorkspaceID string      `json:"workspace_id"`
	Version     string      `json:"version"`
	Content     string      `json:"content"`
	State       PolicyState `json:"state"`
	Description string      `json:"description"`
	CreatedAt   time.Time   `json:"created_at"`
	ActivatedAt *time.Time  `json:"activated_at,omitempty"`
}

var (
	// ErrNotFound is returned when a requested policy version is not present.
	ErrNotFound = errors.New("policystore: policy not found")

	// ErrNoActivePolicy is returned when a workspace has no active policy configured.
	ErrNoActivePolicy = errors.New("policystore: no active policy for workspace")

	// ErrAlreadyExists is returned when attempting to insert an identical candidate.
	ErrAlreadyExists = errors.New("policystore: policy version already exists")

	// ErrInvalidStateTransition is returned when a lifecycle transition is forbidden.
	ErrInvalidStateTransition = errors.New("policystore: invalid policy state transition")
)
