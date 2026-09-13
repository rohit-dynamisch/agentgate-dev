# AgentGate G3 — DevOps & Reproducible Persistence Environment

This directory contains the reproducible runtime and testing environment for Gate G3 (Policy Persistence & Governance Lifecycle).

## Services

- **postgres**: PostgreSQL 16 Alpine container with healthcheck.
- **agentgate**: AgentGate core service connected to PostgreSQL, running automatic migrations on startup and exposing the policy governance HTTP API (:8090).
- **qa-probe**: Optional probe container for running the test suite hermetically in docker networks.

## Quickstart

### 1. Launch Environment

```bash
docker compose -f deploy/g3/docker-compose.yml up -d
```

### 2. Verify Readiness

AgentGate's `/readyz` probe tests database health:
```bash
curl http://localhost:8090/readyz
# Returns 200 OK: "ready"
```

If PostgreSQL is down or unreachable, `/readyz` returns 503:
```bash
# Returns 503 Service Unavailable: "dependency not ready: ..."
```

### 3. Run Migrations & Verification

```powershell
./deploy/g3/test_migrations.ps1
```

## Configuration Variables

| Variable | Default | Description |
|---|---|---|
| `AGENTGATE_HTTP_ADDR` | `:8090` | HTTP listen address for API and health endpoints |
| `AGENTGATE_DATABASE_URL` | `""` | PostgreSQL connection string. If omitted, falls back to in-memory store |
| `AGENTGATE_ADMIN_TOKEN` | `agentgate-admin-secret-dev` | Shared secret for administrator governance mutations |
| `AGENTGATE_LOG_LEVEL` | `info` | Minimum log severity (`debug`, `info`, `warn`, `error`) |
