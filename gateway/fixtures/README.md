# G1 gateway-side fixtures (AG-GW-G1-05)

Deterministic, sanitized JSON fixtures for the frozen G1 wire contract
(`docs/PHASES/G1_WORKSTREAMS/GO_BACKEND_G1_CONTRACT.md`, AG-GO-G1-04's `POST /evaluate` shape).
No real tokens, secrets or production identities appear anywhere in this directory.

## Format

Each fixture is a JSON object:

```jsonc
{
  "name": "...",
  "category": "ALLOW | DENY | missing_identity | unknown_tool | malformed_input | evaluation_error | malformed_input_transport",
  "description": "...",
  "request": { /* exact mock wire request shape — sent verbatim as the POST body */ },
  // OR, for a transport-level malformed case instead of "request":
  "raw_body": "<literal string sent verbatim as the POST body, deliberately invalid/undocumented>",
  "expect": {
    "http_status": 200,
    "decision": "ALLOW | DENY | \"\"",
    "reason": "<decision.ReasonCode string, or \"\" when http_status != 200>",
    "policy_version_nonempty": true,
    "backend_reached": true
  }
}
```

`expect.backend_reached` encodes the enforcement semantics this fixture set exists to prove: it
is `true` only for the ALLOW fixture. Every DENY/error/malformed fixture asserts `false` — i.e.
a governed continuation must not happen. There is no real backend behind these fixtures (see
`../README.md`); `backend_reached` is evaluated structurally by
`gateway/harness/g1_scenarios_test.go`'s `wouldForwardToBackend` gate, not by observing an actual
call.

## Required G1 categories and which file covers them

| G1 checkpoint scenario (`00_G1_CHECKPOINT_REFERENCE.md`) | Fixture(s) |
|---|---|
| valid identity + allow | `01_allow_reader_read.json` |
| valid identity + deny | `02_deny_explicit_destructive.json`, `03_deny_no_matching_policy.json` |
| missing identity | `04_missing_identity.json` |
| unknown/unclassified tool | `05_unknown_tool.json` |
| malformed request | `06_malformed_argument.json` (decision-core layer), `08_transport_unknown_field.json` (transport layer, bonus) |
| policy/evaluation error | `07_evaluation_error_broken_role.json` |

Fixtures 02/03 both fall in the DENY category but exercise the two distinct DENY reasons the
frozen contract documents (`policy_deny` vs `no_matching_policy`) — included for stronger
coverage, not because the checkpoint requires two.

## Reuse

These files are plain JSON with no dependency on the harness's Go types — QA/Security or any
other stream can load and POST them with `curl`, another language, or their own harness. See
`../README.md` for the exact reproduction commands and the Go harness that consumes them
automatically.
