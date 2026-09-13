# G2 Workstream 04 — QA / Security

## Mission
Independently verify that identity mapping, tool governance, schema fingerprinting, and argument declarations cannot be bypassed or ambiguously interpreted.

## Tickets

### G2-QA-01 — Authoritative test matrix
Cover identity present/missing/malformed/ambiguous; mapping changes; unknown tool; fingerprint match/mismatch; invalid schema; risk missing/invalid; declared/undeclared args; wrong type; null; missing optional; duplicate declarations; stale records; arbitrary extra metadata. Record expected result and trust boundary.

### G2-QA-02 — Identity abuse
Verify untrusted claims cannot override trusted mapped identity; conflicting identity fails closed; missing identity cannot become a default identity; no hardcoded claim assumption leaks into behavior.

### G2-QA-03 — Tool-governance abuse
Verify unknown tools cannot inherit classification; schema drift denies; same name on another backend is distinct; malformed schema cannot receive a valid governance fingerprint.

### G2-QA-04 — Argument authorization
Verify undeclared args cannot influence policy; wrong types/null fail; omitted optional differs from null; declaration/schema mismatch fails; arbitrary client policy attributes cannot override declared inputs.

### G2-QA-05 — Regression/determinism
Run complete G1 black-box suite. Add deterministic repeated canonicalization/fingerprint checks.

### G2-QA-06 — Trust-boundary review
Look for caller-controlled self-assertion of identity, risk, known status, fingerprint, or policy-input declarations. Test-only paths must be explicit.

### G2-QA-07 — Evidence
Produce security evidence with tests, failures/findings, severity, fixed/carried-forward status, open questions, and exact DoD evidence. Do not silently fix unrelated findings.

## DoD
- No unresolved critical/high security bypass introduced by G2.
- Identity/tool boundaries independently tested.
- G1 regression green.
- O-005/O-006 are implemented only according to approved semantics or explicitly carried forward.
