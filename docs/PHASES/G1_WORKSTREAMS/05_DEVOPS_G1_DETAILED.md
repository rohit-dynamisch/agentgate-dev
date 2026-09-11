# WS-E — DevOps / Release Engineering — G1 Detailed Execution Plan

**Owner:** DevOps  
**Collaborators:** Go Backend, Gateway/MCP, QA/Security  
**Execution:** Sequential tickets. G1 requires reproducible test infrastructure, not the complete production release stack.

## AG-OPS-G1-01 — Inspect existing build/runtime conventions

Inspect Go build scripts, CI, Docker/compose files, gateway configuration and test services. Identify what can be extended instead of creating a parallel environment framework.

**DoD:** Exact G1 environment topology and files to change are known.

## AG-OPS-G1-02 — Define the reproducible G1 topology

Minimum path:

```text
MCP Client
   ↓
agentgateway
   ↓
mock AgentGate
```

Add only dependencies necessary for contract testing.

**DoD:** Every service, port, dependency and environment variable is explicit; no hidden developer state exists.

## AG-OPS-G1-03 — Package the mock AgentGate dependency

**Steps:**
1. Build/package the Go mock using repository conventions.
2. Configure service addressing.
3. Add readiness behavior sufficient for deterministic startup.
4. Keep mock-only configuration isolated from production.
5. Document test-only assumptions.

**DoD:** Gateway and QA can reach the mock from the reproducible environment.

## AG-OPS-G1-04 — Freeze G1 configuration

Document mock endpoint, gateway ext_authz endpoint, test authentication assumptions, MCP test endpoint and required environment variables. Never commit real secrets.

**DoD:** Another developer/agent can configure the environment without inspecting a developer machine.

## AG-OPS-G1-05 — Run the clean-environment smoke test

**Steps:**
1. Start from clean checkout/fresh environment.
2. Build/start services.
3. Verify readiness.
4. Execute ALLOW.
5. Execute DENY.
6. Execute at least one negative case.
7. Collect diagnostic logs.
8. Tear down and repeat to detect hidden state.

**DoD:** G1 scenarios execute without undocumented manual intervention.

## AG-OPS-G1-06 — Handoff environment evidence

Deliver startup/shutdown commands, configuration reference, test command, known limitations and clean-run evidence.

**DoD:** Go Backend, Gateway and QA can independently reproduce G1. Production image/Helm/SBOM/signing work remains outside this checkpoint unless already required by the repository baseline.
