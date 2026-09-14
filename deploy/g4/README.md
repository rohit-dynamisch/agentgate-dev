# AgentGate G4 — DevOps & Integrated Governance-to-Decision Environment

This directory contains the reproducible runtime and testing topology for Gate G4 (Governance Workflow Integration).

## Architectural Scope & Boundaries

- **O-008: OPEN** — Real agentgateway / MCP E2E enforcement is deferred to Gate G6.
- **Gate G4 Purpose**: Proves that the G3 persisted policy lifecycle actually modifies live authorization decisions via the governance-decision integration service, with dry-run evaluation, atomic activation propagation, rollback restoration, and mutation audit hooks (for G5).

## Topology Services

- **postgres**: PostgreSQL 16 Alpine container with pg_isready healthcheck and persistent volume (`g4_pgdata`).
- **agentgate**: AgentGate core service running with PostgreSQL persistence, executing migrations on startup, exposing the governance REST API (`:8090`) with dry-run comparator and active decision engine.
- **qa-probe**: Optional test container profile (`qa-manual`) for running the G4 integration proof suite inside the container network.

## Quickstart

### 1. Launch Topology

```bash
docker compose -f deploy/g4/docker-compose.yml up -d
```

### 2. Verify Health and Readiness

```bash
# Liveness probe
curl -i http://localhost:8090/healthz

# Readiness probe (verifies database connectivity)
curl -i http://localhost:8090/readyz
```

### 3. Governance Lifecycle Verification Flow

Execute the full G4 lifecycle proof sequence against the running instance:

1. **Create Candidate A** (permit readers):
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies \
     -H "Authorization: Bearer agentgate-admin-secret-dev" \
     -H "Content-Type: application/json" \
     -d '{"content":"permit(principal, action, resource);","description":"Candidate A"}'
   ```

2. **Activate Policy A**:
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies/<VERSION_A>/activate \
     -H "Authorization: Bearer agentgate-admin-secret-dev"
   ```

3. **Create Candidate B** (forbid all):
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies \
     -H "Authorization: Bearer agentgate-admin-secret-dev" \
     -H "Content-Type: application/json" \
     -d '{"content":"forbid(principal, action, resource);","description":"Candidate B"}'
   ```

4. **Validate Policy B**:
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies/validate \
     -H "Authorization: Bearer agentgate-admin-secret-dev" \
     -H "Content-Type: application/json" \
     -d '{"content":"forbid(principal, action, resource);"}'
   ```

5. **Dry-Run Compare Candidate B** (observes before/after differences without modifying active state):
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies/<VERSION_B>/dryrun \
     -H "Authorization: Bearer agentgate-admin-secret-dev" \
     -H "Content-Type: application/json" \
     -d '{"sample_requests":[{"execution_id":"ex-1","principal_id":"user-1","principal_roles":["reader"],"backend_id":"backend-1","tool_name":"tool-1","risk":"read"}]}'
   ```

6. **Activate Policy B**:
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies/<VERSION_B>/activate \
     -H "Authorization: Bearer agentgate-admin-secret-dev"
   ```

7. **Rollback to Policy A**:
   ```bash
   curl -X POST http://localhost:8090/api/v1/workspaces/ws-demo/policies/rollback \
     -H "Authorization: Bearer agentgate-admin-secret-dev" \
     -H "Content-Type: application/json" \
     -d '{"target_version":"<VERSION_A>"}'
   ```

### 4. Shutdown

```bash
docker compose -f deploy/g4/docker-compose.yml down -v
```

## Configuration Variables

| Variable | Default | Description |
|---|---|---|
| `AGENTGATE_HTTP_ADDR` | `:8090` | HTTP listen address for API and health endpoints |
| `AGENTGATE_DATABASE_URL` | `""` | PostgreSQL connection string |
| `AGENTGATE_ADMIN_TOKEN` | `agentgate-admin-secret-dev` | Shared secret for administrator governance mutations |
| `AGENTGATE_LOG_LEVEL` | `debug` | Minimum log severity (`debug`, `info`, `warn`, `error`) |
