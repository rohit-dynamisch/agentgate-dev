# G4-W4 — DevOps/Integration Harness

## What
Provide a reproducible integrated PostgreSQL + AgentGate environment for the full governance-to-decision workflow.

## Why
G4 claims require executable integrated evidence, not unit-only tests.

## How
Reuse G3 topology. Provide deterministic startup, migration, readiness, policy lifecycle calls, decision requests, provenance capture, and A→B→rollback-A verification.

Clearly label the external agentgateway/MCP path as untested; O-008 remains open.

## Tests
- compose/config validation
- clean DB migration
- readiness before tests
- deterministic lifecycle E2E run
- restart/persistence
- safe dependency failure where practical

## Out
No real agentgateway E2E, downstream credentials, durable audit engine, or HA.
