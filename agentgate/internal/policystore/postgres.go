package policystore

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_create_policies.sql
var migration001SQL string

// PostgresStore implements Store using PostgreSQL via pgx/v5.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore establishes a connection pool to PostgreSQL.
func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	poolCfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("policystore: parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("policystore: connect postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("policystore: ping postgres: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Migrate executes deterministic schema migrations against PostgreSQL.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("policystore: begin migration tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if _, err := tx.Exec(ctx, migration001SQL); err != nil {
		return fmt.Errorf("policystore: apply 001_create_policies: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO schema_migrations (version)
		VALUES ('001_create_policies')
		ON CONFLICT (version) DO NOTHING;
	`)
	if err != nil {
		return fmt.Errorf("policystore: record migration: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("policystore: commit migration tx: %w", err)
	}

	return nil
}

func (s *PostgresStore) CreatePolicy(ctx context.Context, p PolicyRecord) error {
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

	query := `
		INSERT INTO policies (workspace_id, version, content, state, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.pool.Exec(ctx, query, p.WorkspaceID, p.Version, p.Content, p.State, p.Description, p.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrAlreadyExists
		}
		return fmt.Errorf("policystore: create policy: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetPolicy(ctx context.Context, workspaceID, version string) (PolicyRecord, error) {
	query := `
		SELECT id, workspace_id, version, content, state, description, created_at, activated_at
		FROM policies
		WHERE workspace_id = $1 AND version = $2
	`
	var p PolicyRecord
	err := s.pool.QueryRow(ctx, query, workspaceID, version).Scan(
		&p.ID,
		&p.WorkspaceID,
		&p.Version,
		&p.Content,
		&p.State,
		&p.Description,
		&p.CreatedAt,
		&p.ActivatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PolicyRecord{}, ErrNotFound
		}
		return PolicyRecord{}, fmt.Errorf("policystore: get policy: %w", err)
	}
	return p, nil
}

func (s *PostgresStore) GetActivePolicy(ctx context.Context, workspaceID string) (PolicyRecord, error) {
	query := `
		SELECT id, workspace_id, version, content, state, description, created_at, activated_at
		FROM policies
		WHERE workspace_id = $1 AND state = 'active'
	`
	var p PolicyRecord
	err := s.pool.QueryRow(ctx, query, workspaceID).Scan(
		&p.ID,
		&p.WorkspaceID,
		&p.Version,
		&p.Content,
		&p.State,
		&p.Description,
		&p.CreatedAt,
		&p.ActivatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PolicyRecord{}, ErrNoActivePolicy
		}
		return PolicyRecord{}, fmt.Errorf("policystore: get active policy: %w", err)
	}
	return p, nil
}

func (s *PostgresStore) ListPolicies(ctx context.Context, workspaceID string) ([]PolicyRecord, error) {
	query := `
		SELECT id, workspace_id, version, content, state, description, created_at, activated_at
		FROM policies
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("policystore: list policies: %w", err)
	}
	defer rows.Close()

	var list []PolicyRecord
	for rows.Next() {
		var p PolicyRecord
		if err := rows.Scan(
			&p.ID,
			&p.WorkspaceID,
			&p.Version,
			&p.Content,
			&p.State,
			&p.Description,
			&p.CreatedAt,
			&p.ActivatedAt,
		); err != nil {
			return nil, fmt.Errorf("policystore: scan policy: %w", err)
		}
		list = append(list, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("policystore: rows error: %w", err)
	}

	if list == nil {
		list = []PolicyRecord{}
	}
	return list, nil
}

func (s *PostgresStore) ActivatePolicy(ctx context.Context, workspaceID, version string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("policystore: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM policies WHERE workspace_id = $1 AND version = $2)", workspaceID, version).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("policystore: check version exists: %w", err)
	}
	if !exists {
		return "", ErrNotFound
	}

	var previousVersion string
	err = tx.QueryRow(ctx, "SELECT version FROM policies WHERE workspace_id = $1 AND state = 'active'", workspaceID).Scan(&previousVersion)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("policystore: get active version: %w", err)
	}

	if previousVersion != "" {
		_, err = tx.Exec(ctx, "UPDATE policies SET state = 'historical' WHERE workspace_id = $1 AND state = 'active'", workspaceID)
		if err != nil {
			return "", fmt.Errorf("policystore: transition previous active to historical: %w", err)
		}
	}

	_, err = tx.Exec(ctx, "UPDATE policies SET state = 'active', activated_at = NOW() WHERE workspace_id = $1 AND version = $2", workspaceID, version)
	if err != nil {
		return "", fmt.Errorf("policystore: activate target version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("policystore: commit activation: %w", err)
	}

	return previousVersion, nil
}

func (s *PostgresStore) RollbackPolicy(ctx context.Context, workspaceID, targetVersion string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("policystore: begin rollback tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM policies WHERE workspace_id = $1 AND version = $2)", workspaceID, targetVersion).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("policystore: check target version exists: %w", err)
	}
	if !exists {
		return "", ErrNotFound
	}

	var currentActive string
	err = tx.QueryRow(ctx, "SELECT version FROM policies WHERE workspace_id = $1 AND state = 'active'", workspaceID).Scan(&currentActive)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("policystore: get active version: %w", err)
	}

	if currentActive != "" {
		_, err = tx.Exec(ctx, "UPDATE policies SET state = 'historical' WHERE workspace_id = $1 AND state = 'active'", workspaceID)
		if err != nil {
			return "", fmt.Errorf("policystore: transition current active to historical: %w", err)
		}
	}

	_, err = tx.Exec(ctx, "UPDATE policies SET state = 'active', activated_at = NOW() WHERE workspace_id = $1 AND version = $2", workspaceID, targetVersion)
	if err != nil {
		return "", fmt.Errorf("policystore: set rollback target to active: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("policystore: commit rollback: %w", err)
	}

	return currentActive, nil
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}
