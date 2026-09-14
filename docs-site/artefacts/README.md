# Docs Site — Working Artefacts

This folder holds the working artifacts of the docs site. Project convention:
everything produced *in between* while building/updating this documentation
lives here, inside the docs-site project — not scattered across the repo.

It is committed to the monorepo, because these files are the durable evidence
and tooling behind the published docs (research snapshot, generator scripts,
generated commit-index data).

| Path | What it is |
|---|---|
| `research-notes/` | Survey notes produced while researching the current repo state — the verified facts and source references the docs are built on. |
| `scripts/` | Content-generation scripts (only one today: the commit-index generator). |

The generator emits `docs/history/commit-index.md` (it does not write into
`artefacts/`), so the committed `docs/` tree is always consistent with `git log`: