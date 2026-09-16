-- deploy/g5/init-roles.sql
-- PostgreSQL initialization script for G5 privilege separation.
--
-- Migration Role: agentgate (superuser/owner created by POSTGRES_USER)
-- Runtime Role:   agentgate_app (restricted application identity)

-- 1. Create runtime application user if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'agentgate_app') THEN
        CREATE USER agentgate_app WITH PASSWORD 'agentgate-app-dev-password';
    END IF;
END
$$;

-- 2. Grant connection and schema usage
GRANT CONNECT ON DATABASE agentgate_db TO agentgate_app;
GRANT USAGE ON SCHEMA public TO agentgate_app;

-- 3. Set default privileges for tables created by migration role (agentgate)
ALTER DEFAULT PRIVILEGES FOR ROLE agentgate IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE ON TABLES TO agentgate_app;

ALTER DEFAULT PRIVILEGES FOR ROLE agentgate IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO agentgate_app;

-- 4. Explicit G5 Immutability Privilege Boundary:
-- Revoke UPDATE and DELETE on audit_events from agentgate_app.
-- (Executed once migrations have created audit_events, or via trigger if table exists)
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'audit_events') THEN
        REVOKE ALL ON TABLE audit_events FROM agentgate_app;
        GRANT SELECT, INSERT ON TABLE audit_events TO agentgate_app;
    END IF;
END
$$;
