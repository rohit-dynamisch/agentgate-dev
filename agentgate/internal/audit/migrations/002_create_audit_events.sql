CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    workspace_id VARCHAR(128) NOT NULL,
    sequence_number BIGINT NOT NULL,
    execution_id VARCHAR(128) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type VARCHAR(32) NOT NULL DEFAULT 'decision',
    decision VARCHAR(16) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    principal_agent_id VARCHAR(128) NOT NULL,
    principal_roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    principal_on_behalf_of VARCHAR(128) NOT NULL DEFAULT '',
    tool_backend_id VARCHAR(128) NOT NULL,
    tool_name VARCHAR(128) NOT NULL,
    tool_risk VARCHAR(32) NOT NULL,
    policy_version VARCHAR(64) NOT NULL DEFAULT '',
    policy_hash VARCHAR(64) NOT NULL DEFAULT '',
    redacted_arguments JSONB NOT NULL DEFAULT '{}'::jsonb,
    canonical_payload TEXT NOT NULL DEFAULT '',
    prev_hash VARCHAR(64) NOT NULL,
    row_hash VARCHAR(64) NOT NULL,
    UNIQUE (workspace_id, sequence_number)
);

CREATE INDEX IF NOT EXISTS idx_audit_events_workspace_seq ON audit_events(workspace_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_audit_events_execution_id ON audit_events(execution_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);

CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is immutable and append-only: UPDATE and DELETE are prohibited';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_audit_modification ON audit_events;
CREATE TRIGGER trg_prevent_audit_modification
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();
