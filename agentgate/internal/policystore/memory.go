package policystore

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryStore is an in-memory, concurrency-safe implementation of Store.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]map[string]PolicyRecord // workspaceID -> version -> record
}

// NewMemoryStore creates a new initialized MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]map[string]PolicyRecord),
	}
}

func (m *MemoryStore) CreatePolicy(_ context.Context, p PolicyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p.WorkspaceID == "" {
		return ErrInvalidStateTransition
	}
	if p.Version == "" {
		p.Version = ComputeVersion(p.Content)
	}
	if p.State == "" {
		p.State = StateCandidate
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	ws, exists := m.data[p.WorkspaceID]
	if !exists {
		ws = make(map[string]PolicyRecord)
		m.data[p.WorkspaceID] = ws
	}

	if _, found := ws[p.Version]; found {
		return ErrAlreadyExists
	}

	ws[p.Version] = p
	return nil
}

func (m *MemoryStore) GetPolicy(_ context.Context, workspaceID, version string) (PolicyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ws, exists := m.data[workspaceID]
	if !exists {
		return PolicyRecord{}, ErrNotFound
	}
	rec, found := ws[version]
	if !found {
		return PolicyRecord{}, ErrNotFound
	}
	return rec, nil
}

func (m *MemoryStore) GetActivePolicy(_ context.Context, workspaceID string) (PolicyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ws, exists := m.data[workspaceID]
	if !exists {
		return PolicyRecord{}, ErrNoActivePolicy
	}
	for _, rec := range ws {
		if rec.State == StateActive {
			return rec, nil
		}
	}
	return PolicyRecord{}, ErrNoActivePolicy
}

func (m *MemoryStore) ListPolicies(_ context.Context, workspaceID string) ([]PolicyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ws, exists := m.data[workspaceID]
	if !exists {
		return []PolicyRecord{}, nil
	}

	list := make([]PolicyRecord, 0, len(ws))
	for _, rec := range ws {
		list = append(list, rec)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	return list, nil
}

func (m *MemoryStore) ActivatePolicy(_ context.Context, workspaceID, version string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ws, exists := m.data[workspaceID]
	if !exists {
		return "", ErrNotFound
	}

	target, found := ws[version]
	if !found {
		return "", ErrNotFound
	}

	now := time.Now().UTC()
	var previousVersion string

	for v, rec := range ws {
		if rec.State == StateActive {
			previousVersion = v
			rec.State = StateHistorical
			ws[v] = rec
		}
	}

	target.State = StateActive
	target.ActivatedAt = &now
	ws[version] = target

	return previousVersion, nil
}

func (m *MemoryStore) RollbackPolicy(_ context.Context, workspaceID, targetVersion string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ws, exists := m.data[workspaceID]
	if !exists {
		return "", ErrNotFound
	}

	target, found := ws[targetVersion]
	if !found {
		return "", ErrNotFound
	}

	now := time.Now().UTC()
	var currentActive string

	for v, rec := range ws {
		if rec.State == StateActive {
			currentActive = v
			rec.State = StateHistorical
			ws[v] = rec
		}
	}

	target.State = StateActive
	target.ActivatedAt = &now
	ws[targetVersion] = target

	return currentActive, nil
}

func (m *MemoryStore) Close() error {
	return nil
}
