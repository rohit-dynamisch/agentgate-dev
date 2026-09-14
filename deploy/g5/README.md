# AgentGate — Gate G5 Deployment Topology & Operational Runbook

This directory contains the reproducible Docker Compose environment and database configuration for **Gate G5: Durable Audit Boundary**.

---

## 1. Architecture & Service Topology

```text
               +----------------------------------------------------+
               |                   g5net (bridge)                   |
               |                                                    |
               |   +-------------------+    +--------------------+  |
               |   |   g5-agentgate    |    |    g5-postgres     |  |
HTTP :8090 --->+-->|  (:8090 / core)   |--->|      (:5432)       |  |
               |   +-------------------+    +--------------------+  |
               |                                ^                   |
               |   +-------------------+        |                   |
               |   |    g5-qa-probe    |--------+                   |
               |   |  (audit verifier) |                            |
               |   +-------------------+                            |
               +----------------------------------------------------+
```

- **`g5-postgres`**: PostgreSQL 16 Alpine container with persistent volume storage (`g5_pgdata`). Initialized with separate migration and runtime user roles via `init-roles.sql`.
- **`g5-agentgate`**: AgentGate core application running with Go 1.24, configured with PostgreSQL audit persistence, pre-persistence argument redaction, and SHA-256 hash chaining.
- **`g5-qa-probe`**: Integration probe profile for independent acceptance and security verification.

---

## 2. Database Privilege Separation

To uphold the G5 immutability invariant, database access is divided into two distinct identities:

| Role Name | Purpose | Privileges on `policies` | Privileges on `audit_events` |
|:---|:---|:---|:---|
| `agentgate` | Migration / Schema Owner | `ALL PRIVILEGES` | `ALL PRIVILEGES` (subject to trigger) |
| `agentgate_app` | Runtime Daemon | `SELECT, INSERT, UPDATE` | `SELECT, INSERT` only (**NO UPDATE**, **NO DELETE**) |

### Immutability Enforcement Layers
1. **Role-Based Access Control (RBAC)**: `agentgate_app` has no `UPDATE` or `DELETE` grants on `audit_events`.
2. **Database Engine Trigger (`prevent_audit_modification`)**: Even if executed by a privileged role, the PostgreSQL trigger rejects any `UPDATE` or `DELETE` attempt on `audit_events` rows with:
   `RAISE EXCEPTION 'audit_events is immutable and append-only: UPDATE and DELETE are prohibited'`

---

## 3. Configuration & Environment Variables

| Variable | Description | Test Fixture Default | Production Requirement |
|:---|:---|:---|:---|
| `AGENTGATE_ADMIN_TOKEN` | Bearer token for admin governance API | `agentgate-admin-secret-dev` | Secret manager injection |
| `AGENTGATE_AUDIT_SALT` | Deployment salt for argument hashing | `audit-salt-g5-dev-fixture` | High-entropy secret key |
| `AGENTGATE_DATABASE_URL` | Migration connection string (owner) | `postgres://agentgate:...` | Managed DB connection |
| `AGENTGATE_RUNTIME_DATABASE_URL` | Runtime app connection string | `postgres://agentgate_app:...` | Restricted app credentials |

---

## 4. Operational Runbook & Verification Steps

### Step 1: Start the Environment
```bash
docker compose -f deploy/g5/docker-compose.yml up -d
```

### Step 2: Verify Service Health
```bash
docker compose -f deploy/g5/docker-compose.yml ps
curl -i http://localhost:8090/healthz
curl -i http://localhost:8090/readyz
```

### Step 3: Verify Database Privilege Denial (Immutability Proof)
Connect using the runtime credentials and attempt to modify history:
```bash
docker compose -f deploy/g5/docker-compose.yml exec postgres psql -U agentgate_app -d agentgate_db -c \
  "DELETE FROM audit_events WHERE id = 1;"
```
**Expected Outcome**: Permission denied (`ERROR: permission denied for table audit_events`).

Attempt modification using the migration owner credentials:
```bash
docker compose -f deploy/g5/docker-compose.yml exec postgres psql -U agentgate -d agentgate_db -c \
  "UPDATE audit_events SET decision = 'ALLOW' WHERE id = 1;"
```
**Expected Outcome**: Trigger exception (`ERROR: audit_events is immutable and append-only: UPDATE and DELETE are prohibited`).

### Step 4: Restart Recovery Test
Prove that committed audit events survive container restarts:
```bash
# 1. Check existing record count
docker compose -f deploy/g5/docker-compose.yml exec postgres psql -U agentgate -d agentgate_db -c \
  "SELECT count(*), max(sequence_number) FROM audit_events;"

# 2. Restart core service
docker compose -f deploy/g5/docker-compose.yml restart agentgate

# 3. Confirm all records and cryptographic hashes remain intact
curl -i http://localhost:8090/readyz
```

### Step 5: Database Outage / Fail-Closed Verification (O-002)
Inject a simulated database outage:
```bash
docker compose -f deploy/g5/docker-compose.yml pause postgres
```
Attempt an authorization request that would normally evaluate to `ALLOW`.  
**Expected Outcome**: The request **must fail closed**, returning `DENY` with reason code `evaluation_error`. No `ALLOW` decision is ever returned without durable audit confirmation.

Resume database service:
```bash
docker compose -f deploy/g5/docker-compose.yml unpause postgres
```

---

## 5. Retention, Partitioning & Backup Policy

- **Partitioning**: In high-throughput production environments, `audit_events` is partitioned by `timestamp` (monthly intervals: `audit_events_y2026m09`).
- **Chain Continuity across Retention Boundaries**: Each partition preserves `prev_hash` linking back to the final row of the preceding partition. When cold partitions are archived to immutable WORM object storage (e.g. S3 Object Lock), the cryptographic checksum of the partition is stored in the audit manifest.
- **Backup & Recovery**: Standard WAL archiving (Point-in-Time Recovery). Restoring from backup preserves sequence numbers and row hashes; any row modification or truncation will be flagged immediately by `ChainVerifier`.
