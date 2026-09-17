# Developer Documentation Status & Tracking

This file tracks the synchronization status between the **AgentGate codebase** and the **Docusaurus Developer Documentation Site** (`developer-docs/`). Use this as a reference point for future documentation updates.

---

## Current Documentation Baseline

- **Last Documented Checkpoint:** **G5 — Durable Audit Boundary** (PASS/CLOSED/FROZEN as of commit `25b5ea4` on 2026-09-14)
- **Documented System State:** Complete coverage up to G5, including Cedar decision core, Identity Claims Mapper, Tool Registry, Versioned Policy Store & Manager, Admin Governance REST API, Dry-Run Compare & Rollback, Mutation Audit Events, Append-Only Tamper-Evident Audit Persistence (`internal/audit`), SHA-256 Hash Chaining & `ChainVerifier`, Pre-Persistence Argument Redaction, Fail-Closed Audit Enforcement, DB Immutability & Privilege Separation, and QA Proof Suites (`g1blackbox`, `g2security`, `g3governance`, `g4integration`, `g5audit`).
- **Generated Commit Index:** Synchronized up to commit `25b5ea4` (61 product commits recorded).

---

## Where to Start for Next Iteration (G6+)

When the next development checkpoint (**G6 — Real MCP End-to-End Enforcement**) completes, update the developer documentation as follows:

1. **Architecture & Gateway Docs (`docs/architecture/`):**
   - Document the real MCP wire-level integration and gateway routing (`agentgateway` ext_authz connection).
   - Document how `internal/audit` durable persistence is triggered from live gateway proxy calls.
2. **API & Interface References (`docs/reference/`):**
   - Update wire contract specifications for live MCP tool call intercepts and responses.
3. **Development & Status Updates (`docs/development/`):**
   - Update `docs/development/current-status.md` with G6 PASS/CLOSED milestone details.
   - Update system capabilities list in `current-status.md` to reflect live MCP enforcement.
4. **History & Commit Index (`docs/history/`):**
   - Add G6 narrative section to `docs/history/evolution.md`.
   - Regenerate the commit index from repo root:
     ```bash
     node developer-docs/artefacts/scripts/generate-commit-index.mjs
     ```
5. **Docusaurus Build Verification:**
   - Run `npm run build` inside `developer-docs/` to ensure zero broken links or rendering errors.
