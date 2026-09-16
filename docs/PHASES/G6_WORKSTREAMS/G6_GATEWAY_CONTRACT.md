# G6 Gateway Authorization Contract & Discovery Evidence

**Status:** FROZEN DISCOVERY EVIDENCE  
**Date:** 2026-09-16  
**Checkpoint:** Gate G6 — Real MCP Enforcement Gate (Phase 1)  
**Decision:** SAFE TO IMPLEMENT ADAPTER

---

## 1. Executive Summary

This document records the empirical evidence captured by running real MCP traffic through the pinned `agentgateway` binary against a recording Envoy ext_authz probe and a instrumented MCP fixture backend.

**Verdict:** The pinned `agentgateway` binary (v1.4.0) supports the standard Envoy v3 gRPC `Authorization/Check` protocol with request body inclusion and fail-closed error semantics. It satisfies all 5 required contract facts. It is **SAFE TO IMPLEMENT THE AGENTGATE ADAPTER**.

---

## 2. Pinned Gateway Artifact

| Property | Value |
|---|---|
| Image Repository | `cr.agentgateway.dev/agentgateway` |
| Image Tag | `v1.4.0` |
| Immutable Digest | `sha256:771afaf093065477fa296eb90dcb618a0300165f12a32f80bbdd1427fab900ec` |
| Git Revision | `83c952731ee79b4372e3a031382c4ff419ddfee1` |
| Rust Compiler | `1.97.0` |
| Manifest File | [`deploy/g6/IMAGE_DIGESTS.md`](file:///d:/PROJECTS/AgentGate_Hackathon/agentgate-repo/deploy/g6/IMAGE_DIGESTS.md) |

---

## 3. Validated Gateway Configuration Schema

The binary schema was extracted directly from the compiled ELF binary and verified via `--validate-only`:

```yaml
gateways:
  default:
    port: 3000

routes:
  - policies:
      extAuthz:
        host: probe-authz:9001
        policies:
          http:
            requestTimeout: 5s
        protocol:
          grpc: {}
        failureMode: deny
        includeRequestBody:
          maxRequestBytes: 1048576
          allowPartialMessage: false
          packAsBytes: false
    backends:
      - host: probe-mcp:9100
```

### Key Configuration Findings
1. **gRPC Protocol**: `policies.extAuthz.protocol.grpc: {}` triggers native Envoy `envoy.service.auth.v3.Authorization/Check` gRPC callout.
2. **Body Inclusion**: `policies.extAuthz.includeRequestBody` accepts:
   - `maxRequestBytes`: uint (e.g. `1048576` for 1MB)
   - `allowPartialMessage`: bool (must be `false` to prevent truncation bypass)
   - `packAsBytes`: bool
3. **Fail-Closed Failure Mode**: `policies.extAuthz.failureMode: deny` is the default and explicitly enforced.
4. **Timeout**: Configured via `policies.extAuthz.policies.http.requestTimeout: 5s`.

---

## 4. Observed Authorization Request Evidence

Captured during live execution of `deploy/g6/run-contract-probe.ps1`:

### 4.1 CheckRequest Field Structure
```json
{
  "attributes": {
    "request": {
      "http": {
        ":authority": "localhost:3000",
        ":method": "POST",
        ":path": "/",
        ":scheme": "http",
        "accept-encoding": "gzip",
        "content-length": "107",
        "content-type": "application/json",
        "user-agent": "Go-http-client/1.1",
        "body": "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"arguments\":{\"verbose\":true},\"name\":\"read_status\"}}",
        "raw_body": "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"arguments\":{\"verbose\":true},\"name\":\"read_status\"}}"
      }
    },
    "metadata_context": {
      "filter_metadata": {
        "envoy.filters.http.jwt_authn": { ... }
      }
    }
  }
}
```

### 4.2 Observed Facts Checklist
| Contract Fact | Observed Value | Evaluation |
|---|---|---|
| `body_complete` | Complete JSON-RPC payload observed in `Http.Body` and `Http.RawBody` | **PASS** |
| `not_truncated` | `body_length: 107`, `raw_body.Length: 107` | **PASS** |
| `route_provenance` | Backend received exactly `read_status` with `verbose: true` | **PASS** |
| `identity_provenance` | HTTP method, path, headers and JWT filter metadata available | **PASS** |
| `fail_closed` | DENY -> 0, MALFORMED -> 0, UNAVAILABLE -> 0 | **PASS** |

---

## 5. Fail-Closed Verification Matrix

Exercised against the real running container topology in `run-contract-probe.ps1 -ValidateEvidence`:

| Scenario / Mode | Probe Response | Gateway Behavior | Backend Call Count | Result |
|---|---|---|---:|---|
| **Normal ALLOW** | gRPC OK (`Status.Code = 0`, `OkResponse`) | Request routed to MCP backend | **1** | **PASS** |
| **Enforced DENY** | gRPC PermissionDenied (`Status.Code = 7`, `DeniedResponse` HTTP 403) | Gateway drops request, returns 403 to client | **0** | **PASS** |
| **Malformed Response** | gRPC InvalidArgument / missing OK | Gateway drops request, returns 500 to client | **0** | **PASS** |
| **Unavailable / Outage** | Probe connection refused / unreachable | Gateway drops request, fails closed | **0** | **PASS** |

---

## 6. Resolution of Open Decision O-008

- **Decision ID:** O-008
- **Title:** `ext_authz` Wire Protocol Translation into `decision.Request`
- **Resolution:**
  1. AgentGate will expose an Envoy v3 gRPC `envoy.service.auth.v3.Authorization` server (default port `:9001`).
  2. The adapter unmarshals the MCP JSON-RPC 2.0 tool call from `CheckRequest.Attributes.Request.Http.Body` (or `RawBody`).
  3. The adapter extracts caller identity from `CheckRequest.Attributes.MetadataContext.FilterMetadata["envoy.filters.http.jwt_authn"]` (or configured client identity headers if non-JWT), passing through `identity.Mapper`.
  4. The adapter extracts tool name and arguments from the parsed MCP body, validating schema against `toolregistry` and `argdecl`.
  5. The assembled `decision.Request` is evaluated through `audit.AuditedDecisionService.Evaluate()`.
  6. On ALLOW: returns `CheckResponse{Status: {Code: 0}, OkResponse: ...}`.
  7. On DENY: returns `CheckResponse{Status: {Code: 7}, DeniedResponse: {Status: {Code: 403}, Body: "AgentGate: policy denied tool execution"}}`.

---

## 7. Implementation Decision

**Decision:** SAFE TO IMPLEMENT ADAPTER  
Proceed to Phase 2: authoring `docs/superpowers/plans/2026-09-14-g6-enforcement-implementation.md` and implementing the production gRPC adapter in `internal/authz`.
