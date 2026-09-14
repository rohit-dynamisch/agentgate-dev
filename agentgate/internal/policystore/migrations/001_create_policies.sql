-- AgentGate G3 Policy Persistence Schema
-- Provides versioned, workspace-isolated policy storage with atomic activation.

CREATE TABLE IF NOT EXISTS policies (
    id SERIAL PRIMARY KEY,
    workspace_id VARCHAR(128) NOT NULL,
    version VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    state VARCHAR(32) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    UNIQUE (workspace_id, version)
);

CREATE INDEX IF NOT EXISTS idx_policies_workspace_state ON policies(workspace_id, state);

-- Enforce at the DB constraint level that there is at most ONE active policy per workspace.
CREATE UNIQUE INDEX IF NOT EXISTS idx_policies_unique_active 
ON policies(workspace_id) 
WHERE state = 'active';

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(64) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
