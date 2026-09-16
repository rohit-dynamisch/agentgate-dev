---
name: submit-checkpoint-for-review
description: Use when all workstream tasks in a checkpoint (G1, G2, ...) are implemented, verified, and ready for Lead Architect review — generates the comprehensive copy-paste submission report and exact file attachment list for ChatGPT without preempting the verdict
---

# Submit Checkpoint for Review

## Overview

Per `/WORKFLOW.md`, the Lead Architect (ChatGPT) holds sole authority to evaluate implementations, verify invariants, and render formal checkpoint verdicts (`PASS`, `FAIL`, `CORRECTIONS NEEDED`, `CLOSED`, `FROZEN`). The Senior Software Engineer (Claude / coding agent) does **not** decide whether a checkpoint passes.

This skill governs the handoff moment when all checkpoint workstreams are implemented and verified. It produces:
1. An explicit list of exact files for the human coordinator to attach to the Lead Architect session.
2. A self-contained, copy-pasteable review prompt containing structured empirical evidence, contract verification tables, and open decision statuses.
3. Proper repository status updates marking the checkpoint as `SUBMITTED FOR REVIEW`, never prematurely `PASS / CLOSED`.

---

## The Non-Negotiable Rule

> [!CRITICAL]
> **NEVER declare a checkpoint as `PASS`, `CLOSED`, or `FROZEN` yourself.**  
> Only the Lead Architect can make that call. The coding agent's job is to present complete, verifiable, empirical proof—not to pronounce judgment on its own work.

---

## Steps

### 1. Set Checkpoint Status to Pending Review
Before preparing the submission, ensure the repository records reflect the true state:
- In `docs/DEVELOPMENT/CURRENT_STATUS.md`: mark the milestone as `SUBMITTED FOR REVIEW` (with Lead Architect sign-off pending).
- In `docs/PHASES/G{N}_WORKSTREAMS/CLOSURE_SUMMARY.md`: mark header status as `SUBMITTED FOR REVIEW (Lead Architect Verdict Pending)`.

### 2. Verify and Gather Empirical Evidence
Execute full verification across touched modules:
- Run Go test suites across all packages. On Windows hosts with Smart App Control or Device Guard enabled, use project-local temporary directories (`$env:GOTMPDIR = "$PWD\.tmp"`) or execute within a containerized runner (`golang:1.24-alpine` with `GOTOOLCHAIN=auto`).
- Run Frontend test suites (`npm test` / `npm run build`).
- Run black-box / E2E security matrices against the clean deployment topology.
- Gather exact figures: pass rates, test counts, HTTP status codes, backend invocation counts, and database row chain audit results.

### 3. Identify the Exact File Attachment List
Compile a categorized list of files that ChatGPT needs to review without repo access:
- **Durable Summaries & Contracts:**
  - `docs/PHASES/G{N}_WORKSTREAMS/CLOSURE_SUMMARY.md` (codebase walkthrough, Mermaid diagram, directory purpose table, design rationale, trade-offs).
  - Pinned contract documents (e.g. `G{N}_GATEWAY_CONTRACT.md`).
- **Workstream Handoff Packages (in `docs/PHASES/G{N}_WORKSTREAMS/results/`):**
  - `<WORKSTREAM>_REPORT.md`: Detailed report covering what was implemented, tests run, deviations, and edge cases.
  - `<WORKSTREAM>_DIGEST.md`: Code digest containing actual code excerpts of critical implementation files.
- **Formatting:** Always provide both relative repository paths and clickable absolute links (`file:///...`) so the human coordinator can locate and attach them immediately.

### 4. Generate the Copy-Paste Submission Prompt
Format the review message in a clean, copy-pasteable Markdown block containing the following standard sections:

1. **Header:**
   - Checkpoint identifier (e.g., `G6 — Real MCP End-to-End Enforcement`).
   - Implementer role (`Claude / Senior Software Engineer`).
   - Active branch and commit log / commit hashes.
   - Core target invariant under review.
2. **Scope Executed:**
   - Bulleted summary of changes organized by workstream (Backend, Gateway/MCP, DevOps, QA/Security, Frontend).
   - Key files introduced or modified.
3. **Empirical Verification Evidence:**
   - A structured Markdown table of mandatory Definition-of-Done (DoD) test scenarios detailing:
     - Scenario # and description
     - Expected gateway status vs actual gateway status
     - Expected backend calls vs actual backend calls
     - Pass/fail outcome
   - Live outage & recovery evidence (demonstrating fail-closed behavior when services are unavailable).
   - Audit integrity evidence (database persistence, SHA-256 row chaining, immutability triggers).
4. **Architecture & Open Decisions Status:**
   - Decisions proposed for resolution (cross-referencing `docs/DECISIONS/OPEN_DECISIONS.md` IDs, e.g. O-008, O-003) with concrete justification.
   - Items deliberately carried forward to future checkpoints (e.g. O-001 downstream tokens for G7).
5. **Attached Artifacts List:**
   - Clear enumeration matching the files attached in Step 3.
6. **Closing Call to Action:**
   - Explicitly request the Lead Architect to review the evidence and render the formal checkpoint verdict.

---

## Common Mistakes

- **Preempting the Verdict:** Writing "Verdict: PASS / CLOSED / FROZEN" in reports or `CURRENT_STATUS.md` before the Lead Architect has reviewed.
- **Missing File Excerpts in Digests:** Giving ChatGPT just a list of filenames. ChatGPT has no repo access and cannot inspect files unless they are attached or digested.
- **Omitting Negative / Failure Cases:** Reporting only the happy path. The Lead Architect specifically evaluates whether fail-closed invariants hold when things break.
- **Unclear File Locations:** Not telling the human coordinator exactly where to find the handoff files. Always provide exact filesystem paths.
- **Hiding Known Gaps or Deviations:** Omitting known rough edges or unresolved architectural questions. Disclose all trade-offs and open decisions upfront.
