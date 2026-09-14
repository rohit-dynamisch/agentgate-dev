# G5 WS2 — QA / Security

## What
Independently prove durable audit correctness and security at the real AgentGate/PostgreSQL boundary.

## Why
Audit is a security control. Claims such as durable, immutable, provenance-preserving and tamper-evident must be demonstrated, not inferred from code structure.

## How
Build black-box/security tests for:
- ALLOW durable record.
- DENY durable record.
- Exact policy version/hash provenance, including after later policy activation.
- `workspace_id` isolation.
- Correlation/execution ID.
- Redaction before persistence/logging using synthetic secrets.
- `full`/`hash`/`omit` only where explicitly permitted.
- Independent hash-chain verification and tamper detection.
- Runtime application role UPDATE/DELETE rejection.
- AgentGate restart preserving already-committed records.
- PostgreSQL outage, insert failure, timeout, permission failure and buffer saturation (if applicable).
- Concurrent decisions/audit writes.

Tampering may use a privileged test mechanism solely to simulate an attacker/admin mutation; then prove runtime credentials cannot do the same.

Acceptance matrix:
| Scenario | Expected |
|---|---|
| ALLOW | Durable audit |
| DENY | Durable audit |
| Policy changes later | Old provenance unchanged |
| Sensitive argument | Raw secret absent |
| Tampered row | Verification fails |
| Runtime UPDATE/DELETE | Rejected |
| Restart | Committed records remain |
| Audit failure | Defined fail-safe behavior |
| Workspace A/B | Isolated |

Treat raw credentials in audit, provenance mismatch, mutable history via runtime credentials, unverifiable chain, workspace crossover, silent committed-record loss, or unsafe audit-failure behavior as release-blocking.

Return DONE / CONTRACT / BLOCKED / RISK / NEXT with exact evidence.
