# Deploy Topology: Gate G6 Real MCP Enforcement Gate

This directory defines the reproducible integration environment for Gate G6:
`Real MCP Client -> agentgateway -> AgentGate -> Real MCP Backend`.

## Pinned Artifacts

- **agentgateway**: `cr.agentgateway.dev/agentgateway:v1.4.0@sha256:771afaf093065477fa296eb90dcb618a0300165f12a32f80bbdd1427fab900ec`
- **MCP Protocol**: Streamable HTTP JSON-RPC 2.0 (2026-07-28 revision)
- **External Authz**: Envoy `envoy.service.auth.v3.Authorization/Check` gRPC callout with `failureMode: deny` and `includeRequestBody: { maxRequestBytes: 1048576, allowPartialMessage: false, packAsBytes: false }`.

## Architecture Topology (`agentgate-g6`)

```text
[Real MCP Client / QA Probe] (MCP 2026-07-28 client)
       │
       ▼ (HTTP POST :3000)
[g6-agentgateway] (v1.4.0, fail-closed)
       │
       ├─► [g6-agentgate] (Envoy v3 gRPC ext_authz on :9001, Gov HTTP on :8090)
       │   ├── Identity Mapping & Tool Governance
       │   ├── Cedar Policy Engine Evaluation
       │   └── Durable Tamper-Evident PostgreSQL Audit Chain
       │
       ▼ (Routed only if AgentGate authorization returns OK)
[g6-probe-mcp] (Governed MCP Backend on :9100)
       └── Atomic invocation counter on :9101 (/_g6/count)
```

## Reproducible Commands

### 1. Verify Pinned Image Digest
```powershell
powershell -ExecutionPolicy Bypass -File deploy/g6/verify-image.ps1
```

### 2. Validate Gateway Configuration
```powershell
docker run --rm -v "${PWD}/deploy/g6/agentgateway.yaml:/config.yaml:ro" cr.agentgateway.dev/agentgateway:v1.4.0 --validate-only -f /config.yaml
```

### 3. Run Phase 1 Contract Probe & Evidence Validation
```powershell
powershell -ExecutionPolicy Bypass -File deploy/g6/run-contract-probe.ps1 -ValidateEvidence
```

### 4. Run Phase 2 Integrated E2E Enforcement Matrix
```powershell
powershell -ExecutionPolicy Bypass -File deploy/g6/run-e2e-matrix.ps1
```

## Credential Hygiene & Production Boundary

> [!IMPORTANT]
> **G6 Local Integration Fixture != Production Secret Configuration**  
> The `deploy/g6/docker-compose.yml` topology is designed for deterministic local and CI verification. Sensitive values (`AGENTGATE_ADMIN_TOKEN`, `AGENTGATE_DATABASE_URL`, `POSTGRES_PASSWORD`) use environment interpolation (`${VAR:-default}`) with non-secret development fixtures. Production deployments must **never** use default fallback credentials and must supply external secrets via dedicated orchestrator secret stores (e.g. Kubernetes Secrets, Vault, or AWS Secrets Manager).

