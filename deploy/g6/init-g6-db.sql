-- deploy/g6/init-g6-db.sql
-- PostgreSQL initialization script for G6 privilege separation.
--
-- Migration Role: agentgate (superuser/owner created by POSTGRES_USER)
-- Runtime Role:   agentgate_app (restricted application identity)

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'agentgate_app') THEN
        CREATE USER agentgate_app WITH PASSWORD 'agentgate-app-dev-password';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE agentgate_db TO agentgate_app;
GRANT USAGE ON SCHEMA public TO agentgate_app;

ALTER DEFAULT PRIVILEGES FOR ROLE agentgate IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE ON TABLES TO agentgate_app;

ALTER DEFAULT PRIVILEGES FOR ROLE agentgate IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO agentgate_app;

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'audit_events') THEN
        REVOKE ALL ON TABLE audit_events FROM agentgate_app;
        GRANT SELECT, INSERT ON TABLE audit_events TO agentgate_app;
    END IF;
END
$$;
