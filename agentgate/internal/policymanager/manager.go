package policymanager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Dynamisch-LLC/agentgate/internal/policy"
	"github.com/Dynamisch-LLC/agentgate/internal/policystore"
)

var (
	// ErrNoActiveEngine indicates that the workspace does not currently have an active policy engine.
	ErrNoActiveEngine = errors.New("policymanager: no active policy engine for workspace")

	// ErrInvalidPolicy indicates that the provided Cedar syntax is invalid.
	ErrInvalidPolicy = errors.New("policymanager: invalid Cedar policy syntax")
)

// Manager coordinates the lifecycle, validation, atomic activation, rollback,
// and concurrency-safe in-memory caching of active policy engines.
type Manager struct {
	store policystore.Store

	mu      sync.RWMutex
	engines map[string]*policy.Engine // workspaceID -> loaded active Cedar engine
}

// New constructs a new Manager backed by store.
func New(store policystore.Store) *Manager {
	return &Manager{
		store:   store,
		engines: make(map[string]*policy.Engine),
	}
}

// Store returns the underlying policystore.Store.
func (m *Manager) Store() policystore.Store {
	return m.store
}

// Validate checks the Cedar syntax of the candidate policy source bytes.
// If valid, returns its content-hash version without changing active policy or persistence.
func (m *Manager) Validate(content string) (string, error) {
	eng, err := policy.LoadFromBytes([]byte(content))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}
	return eng.Version(), nil
}

// CreateCandidate validates and persists a new policy candidate.
// Invalid Cedar syntax is rejected and never persisted.
func (m *Manager) CreateCandidate(ctx context.Context, workspaceID, content, description string) (policystore.PolicyRecord, error) {
	version, err := m.Validate(content)
	if err != nil {
		return policystore.PolicyRecord{}, err
	}

	rec := policystore.PolicyRecord{
		WorkspaceID: workspaceID,
		Version:     version,
		Content:     content,
		State:       policystore.StateCandidate,
		Description: description,
	}

	if err := m.store.CreatePolicy(ctx, rec); err != nil {
		if errors.Is(err, policystore.ErrAlreadyExists) {
			return m.store.GetPolicy(ctx, workspaceID, version)
		}
		return policystore.PolicyRecord{}, fmt.Errorf("policymanager: persist candidate: %w", err)
	}

	return rec, nil
}

// Activate validates and atomically activates the specified policy version for a workspace.
// The in-memory cache is updated immediately on successful activation.
func (m *Manager) Activate(ctx context.Context, workspaceID, version string) (policystore.PolicyRecord, error) {
	rec, err := m.store.GetPolicy(ctx, workspaceID, version)
	if err != nil {
		return policystore.PolicyRecord{}, fmt.Errorf("policymanager: get policy for activation: %w", err)
	}

	// Verify syntax before attempting activation in DB
	eng, err := policy.LoadFromBytes([]byte(rec.Content))
	if err != nil {
		return policystore.PolicyRecord{}, fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}

	if _, err := m.store.ActivatePolicy(ctx, workspaceID, version); err != nil {
		return policystore.PolicyRecord{}, fmt.Errorf("policymanager: activate in store: %w", err)
	}

	// Atomically swap the in-memory engine pointer
	m.mu.Lock()
	m.engines[workspaceID] = eng
	m.mu.Unlock()

	return m.store.GetActivePolicy(ctx, workspaceID)
}

// Rollback restores a previously stored policy version to active state.
func (m *Manager) Rollback(ctx context.Context, workspaceID, targetVersion string) (policystore.PolicyRecord, error) {
	rec, err := m.store.GetPolicy(ctx, workspaceID, targetVersion)
	if err != nil {
		return policystore.PolicyRecord{}, fmt.Errorf("policymanager: get rollback target: %w", err)
	}

	eng, err := policy.LoadFromBytes([]byte(rec.Content))
	if err != nil {
		return policystore.PolicyRecord{}, fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}

	if _, err := m.store.RollbackPolicy(ctx, workspaceID, targetVersion); err != nil {
		return policystore.PolicyRecord{}, fmt.Errorf("policymanager: rollback in store: %w", err)
	}

	// Atomically swap the in-memory engine pointer
	m.mu.Lock()
	m.engines[workspaceID] = eng
	m.mu.Unlock()

	return m.store.GetActivePolicy(ctx, workspaceID)
}

// GetActiveEngine returns the currently active policy.Engine for workspaceID.
// Concurrency safe: readers observe either the previous or newly activated valid engine,
// never an invalid or intermediate state.
func (m *Manager) GetActiveEngine(workspaceID string) (*policy.Engine, error) {
	m.mu.RLock()
	eng, ok := m.engines[workspaceID]
	m.mu.RUnlock()

	if !ok || eng == nil {
		return nil, ErrNoActiveEngine
	}
	return eng, nil
}

// Preview evaluates a batch of sample evaluation inputs against a candidate or historical policy
// without changing the active policy or persisting any changes.
func (m *Manager) Preview(ctx context.Context, workspaceID, version string, inputs []policy.EvalInput) ([]policy.EvalOutput, error) {
	rec, err := m.store.GetPolicy(ctx, workspaceID, version)
	if err != nil {
		return nil, fmt.Errorf("policymanager: get preview policy: %w", err)
	}

	eng, err := policy.LoadFromBytes([]byte(rec.Content))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}

	results := make([]policy.EvalOutput, 0, len(inputs))
	for _, in := range inputs {
		results = append(results, eng.Evaluate(in))
	}

	return results, nil
}

// LoadActivePolicies hydrates the in-memory engine cache from persistent storage.
func (m *Manager) LoadActivePolicies(ctx context.Context, workspaces []string) error {
	for _, ws := range workspaces {
		rec, err := m.store.GetActivePolicy(ctx, ws)
		if err != nil {
			if errors.Is(err, policystore.ErrNoActivePolicy) {
				continue
			}
			return fmt.Errorf("policymanager: load active for %s: %w", ws, err)
		}
		eng, err := policy.LoadFromBytes([]byte(rec.Content))
		if err != nil {
			return fmt.Errorf("policymanager: parse active policy for %s: %w", ws, err)
		}
		m.mu.Lock()
		m.engines[ws] = eng
		m.mu.Unlock()
	}
	return nil
}
