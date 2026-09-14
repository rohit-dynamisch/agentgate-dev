# AgentGate v1 — G1 Checkpoint Reference

## Authorization Contract Freeze

**Target:** Day 3  
**Streams:** Go Backend, Gateway/MCP, Frontend/UI, QA/Security, DevOps  
**Authority:** Lead Architect

G1 freezes the shared authorization contract so all streams can work independently without inventing incompatible interfaces. The accepted Day-2 Decision Core is the baseline. G1 is a contract-and-evidence gate, not a redesign of the authorization core.

### Contract that must be frozen

The streams must agree on the exact semantics of:

- authorization request;
- authenticated identity and trust provenance;
- tool identifier;
- tool classification;
- arguments/context supplied to authorization;
- workspace identifier where applicable;
- correlation/execution identifier;
- authorization result;
- stable denial/error categories;
- policy version/hash provenance;
- authentication assumptions at the gateway boundary.

The security invariant is fail-closed:

```text
missing identity        -> DENY
unknown/unclassified    -> DENY
malformed input         -> DENY
policy/evaluation error -> DENY
```

### Integration sequence

```text
Day-2 accepted contract
        |
        v
Go freezes canonical request/result
        |
   +----+---------+---------+
   |              |         |
   v              v         v
Gateway mock   QA tests   Frontend models
   |              |         |
   +--------------+---------+
                  |
                  v
          Cross-stream review
                  |
                  v
             G1 FREEZE
```

### G1 Definition of Done

1. One canonical request/result contract exists.
2. Required/optional fields and trust semantics are documented.
3. ALLOW/DENY and failure semantics are deterministic.
4. Gateway can call a mock AgentGate using the contract.
5. QA independently verifies the negative/fail-closed cases.
6. Frontend has a stable immediate external model boundary.
7. DevOps can reproduce the contract-test environment from a clean checkout.
8. Contract tests pass across participating streams.
9. No unresolved ambiguity forces a dependent stream to guess.
10. After freeze, shared contract changes require Architect review and dependent-test updates.

### Explicitly out of scope

Do not pull full policy persistence/lifecycle, dry-run/replay, durable audit, downstream credentials, real MCP E2E, or production release hardening into G1 except for minimal fixtures needed to prove the contract.
