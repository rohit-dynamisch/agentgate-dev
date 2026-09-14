# WS-A — Go Backend / AgentGate — G1 Detailed Execution Plan

**Owner:** Go Backend  
**Collaborators:** Gateway/MCP, QA/Security, Frontend/UI, DevOps  
**Execution:** Sequential tickets. Do not start the next ticket until the current ticket's verification passes.  
**Baseline:** Accepted Day-2 Decision Core.

## Operating rules

Inspect the repository before editing. Reuse existing Day-2 abstractions. Keep application-level request/result types separate from Cedar-specific policy representation. Do not invent JWT claim names or downstream credential semantics. Do not implement G2+ functionality except minimal test fixtures required by G1.

## AG-GO-G1-01 — Inventory and freeze the existing Day-2 boundary

**Goal:** Establish exactly where the authorization request enters the decision core and where the result exits.

**Steps:**
1. Locate the Day-2 request/result types and evaluator entry point.
2. Locate identity/context structures.
3. Locate tool/classification inputs and argument attributes.
4. Locate policy version/hash propagation.
5. Locate validation and fail-closed error handling.
6. Read existing tests before changing code.
7. Record the existing boundary and only the gaps required for G1.

**DoD:** One application-level authorization boundary is identified; existing Day-2 tests pass; no duplicate request/result model is introduced.

**Verification:** `gofmt` on touched files; `go test ./...`; `go test -race ./...` where repository setup permits.

**Collaboration:** Send the actual existing shape to Gateway and QA. Escalate missing semantic decisions rather than guessing.

## AG-GO-G1-02 — Implement the canonical authorization request

**Goal:** Make the request contract explicit and typed.

**Required semantic fields:** authenticated identity; tool identifier; tool classification; arguments/context; workspace where applicable; correlation/execution ID; trust provenance.

**Steps:**
1. Reuse Day-2 fields where their meaning already matches.
2. Add only fields required by G1.
3. Define required versus optional semantics.
4. Ensure missing required identity/tool data cannot become an implicit allow.
5. Keep gateway transport types out of `internal/decision`.
6. Keep Cedar conversion out of the request type.
7. Add validation tests for missing/invalid required values.

**DoD:** The type and its semantics are documented; invalid required input is deterministically rejected/denied; no transport-specific structure leaks into the decision model.

**Verification cases:** complete request; missing identity; missing tool; unknown/unclassified tool; malformed context.

**Collaboration:** Provide exact field names/types and representative serialized examples to Gateway and QA.

## AG-GO-G1-03 — Implement the canonical authorization result

**Goal:** Give Gateway a stable decision result it can enforce without interpreting internal Cedar details.

**Required semantic fields:** decision; stable denial/error category; policy version; policy hash/provenance; correlation ID where applicable.

**Steps:**
1. Reuse the Day-2 result where possible.
2. Add only stable external fields.
3. Map validation/evaluation failure to DENY.
4. Do not expose Cedar diagnostic strings as authorization semantics.
5. Add ALLOW/DENY/error fixtures.

**DoD:** ALLOW and DENY are distinguishable; errors cannot be mistaken for ALLOW; provenance survives the result boundary.

**Verification:** allow policy; deny policy; policy error; validation error; missing identity; unknown tool.

**Collaboration:** Gateway confirms it can enforce the result using only the documented fields.

## AG-GO-G1-04 — Build the deterministic AgentGate mock

**Goal:** Let Gateway and QA integrate before the complete service is ready.

**Steps:**
1. Use only the frozen request/result contract.
2. Implement deterministic ALLOW, DENY, malformed-input and simulated-failure cases.
3. Do not duplicate Cedar evaluation in the mock.
4. Keep behavior fixture-driven and repeatable.
5. Keep the mock isolated from production startup paths.
6. Document how to start and exercise it.

**DoD:** Gateway can invoke all four categories and receive the exact contract shape; no production component depends on the mock.

**Collaboration:** Handoff endpoint/configuration and fixtures to Gateway, QA and DevOps.

## AG-GO-G1-05 — Publish executable contract tests

**Goal:** Turn the contract into an automated compatibility boundary.

**Steps:** Add tests for valid ALLOW, valid DENY, missing identity, unknown/unclassified tool, malformed input, policy/evaluation failure and policy provenance.

**DoD:** Tests fail if an unsafe condition becomes ALLOW; fixtures are reusable by dependent streams where practical.

**Verification:** `go test ./...`; `go test -race ./...`; inspect final diff.

## AG-GO-G1-06 — Execute the cross-stream freeze

**Goal:** Deliver a package that allows all dependent streams to proceed without guessing.

**Handoff contents:** canonical request/result; field semantics; examples; negative cases; mock instructions; contract-test command; known limitations.

**DoD:** Gateway and QA confirm interoperability; Frontend confirms its immediate model needs; DevOps confirms reproducibility; Architect accepts G1.

**Freeze rule:** Any post-G1 payload change requires explicit review and updated dependent tests.
