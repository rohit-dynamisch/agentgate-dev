# Deploy Topology: Gate G6 Real MCP Enforcement Gate

This directory defines the reproducible integration environment for Gate G6:
`Real MCP Client -> agentgateway -> AgentGate / Probe -> Real MCP Backend`.

## Pinned Artifacts

- **agentgateway**: `cr.agentgateway.dev/agentgateway:v1.4.0@sha256:771afaf093065477fa296eb90dcb618a0300165f12a32f80bbdd1427fab900ec`
- **MCP Protocol**: Streamable HTTP JSON-RPC 2.0 (2026-07-28 revision)
- **External Authz**: Envoy `envoy.service.auth.v3.Authorization/Check` gRPC callout with `failureMode: deny` and `includeRequestBody: { maxRequestBytes: 1048576, allowPartialMessage: false, packAsBytes: false }`.

## Architecture Topology (`agentgate-g6`)

```text
[probe-client] (MCP 2026-07-28 client)
       │
       ▼ (HTTP POST :3000)
[g6-agentgateway] (v1.4.0, fail-closed)
       │
       ├─► [g6-probe-authz] (Envoy v3 gRPC Check on :9001)
       │   └── Mode: allow / deny / malformed / unavailable
       │
       ▼ (Routed only if ext_authz returns OK)
[g6-probe-mcp] (MCP Backend on :9100)
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

### 3. Run Contract Probe & Evidence Validation
```powershell
powershell -ExecutionPolicy Bypass -File deploy/g6/run-contract-probe.ps1
```
