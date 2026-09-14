# WS-D — QA / Security — G1 Detailed Execution Plan

**Owner:** QA/Security  
**Collaborators:** Go Backend, Gateway/MCP, DevOps  
**Acceptance authority:** Lead Architect  
**Execution:** Sequential tickets; tests must be independent enough to detect implementation drift.

## AG-QA-G1-01 — Create the authoritative G1 test matrix

Create concrete inputs and expected externally observable outcomes for:

| Scenario | Expected |
|---|---|
| Valid identity + allow | ALLOW |
| Valid identity + deny | DENY |
| Missing identity | DENY |
| Unknown/unclassified tool | DENY |
| Malformed authorization input | DENY |
| Policy/evaluation failure | DENY |

Also verify policy version/hash provenance on decisions where defined by the contract.

**DoD:** Every row has an executable input and assertion.

## AG-QA-G1-02 — Implement independent negative tests

**Steps:**
1. Remove identity.
2. Malform identity structure.
3. Use an unknown/unclassified tool.
4. Malform arguments/context.
5. Simulate policy/evaluation failure.
6. Where possible, return malformed/unsupported authorization responses.
7. Assert blocking/DENY, not merely an error string.

**DoD:** The suite fails if any mandatory unsafe case becomes ALLOW.

## AG-QA-G1-03 — Validate Go contract and fixtures

Check request/result fields, types, required/optional semantics, deterministic denial categories and provenance.

**DoD:** No undocumented required field or semantic is discovered.

## AG-QA-G1-04 — Verify Gateway interoperability

Run the six mandatory cases through the gateway/mock environment. Record request, result and enforcement behavior.

**DoD:** ALLOW continues; DENY blocks; failure blocks; no direct governed bypass is observable.

## AG-QA-G1-05 — Review trust boundaries

Explicitly answer:

- Which identity fields are trusted?
- Which values originate from authenticated gateway state?
- Which tool fields are client-controlled?
- Can the client assert its own classification?
- Are arguments authorization context rather than trusted authorization state?
- Are sensitive values unnecessarily exposed?
- Can an error be mistaken for ALLOW?

**DoD:** Trust assumptions are documented; unresolved security ambiguity is escalated before freeze.

## AG-QA-G1-06 — Produce G1 evidence report

Record test commands, results, fixtures, findings, limitations and PASS/FAIL/CONDITIONAL recommendation.

**Exit:** G1 cannot PASS from Go unit tests alone; Gateway interoperability and independent negative tests must pass.
