---
name: developer-docs-architect
description: Build and maintain a developer-facing Docusaurus documentation website for a repository, acting as the authoritative onboarding and developer knowledge base so a new developer can understand, run, navigate, modify, test, and debug the project without a KT session. Use this skill whenever the user asks to create, generate, update, refresh, or maintain project documentation, a docs site, a Docusaurus site, a developer wiki, an onboarding guide, an architecture overview, or a commit/history index for a codebase — even if they just say "document this repo" or "set up docs" without naming Docusaurus explicitly. Also use when asked to add diagrams (Mermaid) explaining architecture or data flow, to link documentation to GitHub source/commits/PRs, or to audit existing docs against the current source code for staleness.
---

# Docs Architect

## Purpose

Build and maintain a developer-facing Docusaurus documentation website for the repository. The documentation is the authoritative onboarding and developer knowledge base — a new developer should be able to understand, run, navigate, modify, test, and debug the project without requiring a KT session or tribal knowledge.

## Scope and safety (read first, non-negotiable)

- Read the entire repository and its Git history as needed.
- You may modify **only** files inside the documentation project directory.
- **Never** modify source code, tests, dependencies, lockfiles, CI configuration, or any file outside the documentation project.
- Work independently from other coding agents. Prefer a dedicated Git worktree when running alongside active development, so uncommitted source changes elsewhere don't interfere.
- Do not invent behavior, architecture, commands, or rationale. If something is uncertain, mark it explicitly (e.g. `> ⚠️ Unverified: ...`) and investigate further before finalizing — never silently guess.
- Never fabricate GitHub URLs, commit SHAs, PR numbers, or file paths. If the GitHub remote or a specific reference can't be verified, omit the link rather than invent it.

## Core principle

- **Current source code is the truth** for how the system works *now*.
- **Git history is evidence** for how and why the system evolved.

Inspect historical commits chronologically, but do not create one documentation page per commit. Synthesize history into current-state documentation and preserve important historical context through commit references and a dedicated commit index.

## Workflow

Work through these steps in order. Don't skip ahead to writing large volumes of content before the information architecture is designed.

1. **Survey the repo.** Inspect repository structure, package/workspace configuration, entry points, major components, configuration, tests, scripts, and any existing documentation.
2. **Mine Git history.** Inspect relevant commit messages, changed files, and diffs. Identify major architectural and functional milestones — don't just skim `git log --oneline`; look at diffs for turning points.
3. **Build and verify a mental model.** Form a picture of the current system, then check it against the actual source — don't document architecture that no longer matches the code.
4. **Design the information architecture** before generating large amounts of content (see "Documentation structure" below).
5. **Scaffold or update the Docusaurus site** and its navigation (`sidebars.js`/`sidebars.ts`, `docusaurus.config.js`).
6. **Write concise documentation** aimed at a developer unfamiliar with the project. Prefer concrete examples over prose walls.
7. **Add Mermaid diagrams** where they materially improve understanding (architecture, request/data flow, component relationships, state transitions) — see "Diagrams" below.
8. **Add direct GitHub links** to relevant source files, directories, commits, issues, or PRs — see "GitHub traceability" below.
9. **Generate a browsable commit index** (paginated if large) with SHA, message, date, and GitHub link — see "Commit index" below.
10. **Cross-link history to claims.** Where a design decision or architectural statement rests on specific history, link it to the supporting commit(s).
11. **Build the site and fix errors** — broken links, invalid Markdown/MDX, navigation problems, broken Mermaid blocks.
12. **Final consistency review** against current source — see "Final validation" checklist below before declaring done.

## Documentation structure

Adapt to the project, but default to this shape. Organize navigation around developer needs, not the physical directory tree.

- **Getting Started** — Project Overview, Prerequisites, Local Setup, First Development Task
- **Architecture** — System Overview, Components, Data/Request Flows, Important Design Decisions
- **Repository Guide** — Repository Structure, Where Things Live
- **Development** — Development Workflow, Testing, Common Tasks, Debugging/Troubleshooting
- **Reference** — Configuration, Environment Variables, APIs/Interfaces
- **History** — Architectural Evolution, Commits (the commit index)

## Documentation quality bar

Every significant component's page should answer:

1. What is it?
2. Why does it exist?
3. How does it work?
4. Where is it implemented? (link to source)
5. How does it interact with other components?
6. How do I change or test it?

Rules of thumb:
- Use simple language; skip obvious implementation detail unless it matters for modifying, operating, or debugging the system.
- Prefer short explanations plus a concrete example over long narrative prose.
- **Never duplicate large chunks of source code** — link to it instead. A short illustrative snippet (a few lines) is fine when it clarifies an interface or usage pattern.

## GitHub traceability

Derive links from the repository's actual GitHub remote (`git remote -v` / `origin` URL) — never guess an org/repo name.

| Thing | Link type |
|---|---|
| Source file | `blob` (pin to commit SHA for historical claims, default branch for "current" references) |
| Directory | `tree` |
| Commit | `commit` |
| Pull request | `pull` |

Example patterns (fill in with verified values only):
`https://github.com/{org}/{repo}/blob/{sha-or-branch}/{path}`
`https://github.com/{org}/{repo}/tree/{sha-or-branch}/{path}`
`https://github.com/{org}/{repo}/commit/{sha}`
`https://github.com/{org}/{repo}/pull/{number}`

## Diagrams

Use Mermaid only when it materially improves comprehension — not for decoration. Good candidates: system architecture, request/response flow, data flow, event/message flow, auth flow, package/component relationships, state machines. Keep diagrams small enough to stay readable and maintainable as the system evolves.

## Commit index

The commit index is a reference system, not the primary documentation — keep it separate from narrative docs and paginate/split if the history is large.

For each commit, capture at minimum:
- SHA (short, with full available via the link)
- Commit message
- Date
- Author (when useful)
- Direct GitHub commit link

For **significant** commits only, add a concise description of what changed and why, grounded in the actual diff — never reproduce full diffs, and never editorialize beyond what the diff and message support.

## Final validation checklist

Before declaring the work done, confirm:

- [ ] The Docusaurus build succeeds (`npm run build` or equivalent) with no errors.
- [ ] Navigation and internal links all resolve.
- [ ] Mermaid/MDX blocks render without errors.
- [ ] Source and commit links point to real, verified locations (spot-check a sample).
- [ ] Documentation describes the **current** implementation, not an obsolete historical state.
- [ ] No files outside the documentation project directory were modified.
- [ ] Nothing was left in as an unverified assumption — uncertain claims are either resolved or explicitly flagged.

## Notes for the agent running this skill

- If a Docusaurus site doesn't exist yet in the repo, scaffold one (`npx create-docusaurus@latest`) inside a clearly named docs directory (e.g. `developer-docs/` or `website/`) rather than at repo root, unless the user specifies otherwise.
- If a Docusaurus site already exists, read its current `sidebars` and `docusaurus.config` before restructuring — extend rather than clobber unless asked to redo it.
- Prefer incremental commits/checkpoints within the docs directory so the user can review documentation growth, especially on large repos.
- When repo or history is very large, prioritize: entry points → most-changed files/directories (by commit frequency) → core business logic → peripheral utilities. Don't try to exhaustively document everything before validating the structure with the user.