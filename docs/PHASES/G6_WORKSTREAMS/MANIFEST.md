# G6 Workstream Package

| File | Purpose |
|---|---|
| `00_G6_CHECKPOINT_REFERENCE.md` | Shared G6 objective, contract, DoD, proof matrix |
| `01_GATEWAY_MCP_G6.md` | Gateway/MCP task specification |
| `02_GO_BACKEND_G6.md` | Go Backend task specification |
| `03_QA_SECURITY_G6.md` | QA/Security task specification |
| `04_DEVOPS_G6.md` | DevOps task specification |

## Execution order

Run the four workstreams in parallel.

The Gateway/MCP and Go Backend streams must first establish the actual ext-authz contract from the pinned agentgateway artifact. QA and DevOps can build against the frozen contract/mock while the real implementation is integrated.

No workstream may silently resolve an architectural/security ambiguity.

## Handoff

Each stream returns:
- DONE
- CONTRACT
- BLOCKED
- RISK
- NEXT

plus the required implementation report and digest.

After all streams are reviewed and approved, merge/corrective-closeout occurs, then write:

`docs/PHASES/G6_WORKSTREAMS/CLOSURE_SUMMARY.md`

The closure summary must show the real wired G1→G6 path and clearly distinguish any still-unwired G7 credential boundary and later production gates.
