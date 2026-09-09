# OSS Readiness Backlog

**Status:** Deferred — action before public launch, not before
**Date recorded:** 2026-09-09
**Owner:** Team Lead

This is a checklist of what a reviewer expects from a professional open-source repository, assessed
against the current state of this one. None of these block Phase 1 implementation. They are
recorded here so they are not lost between now and whenever public launch is actually scheduled —
per `CLAUDE.md`, durable project context belongs in the repository, not in chat history.

Do not action items here as part of an unrelated implementation task. Do not "fix" the known,
deliberate inconsistency in item 6 — it is intentional, not a bug.

## 1. License decision — the actual blocker

No `LICENSE` file exists yet. This is correctly left open: it is tracked as **O-4** in
`docs/TECH_STACK.md §4` (Apache-2.0 vs. AGPL-3.0/BUSL), an explicit management-level decision, not
one to guess. `README.md`'s License section already states this honestly ("Not yet decided. ...
No license is granted for use of this code until that decision is made").

Until this is decided:
- the repo cannot legally accept external contributions;
- most companies' legal/compliance functions will not evaluate an unlicensed repo at all;
- every item below that assumes "open for contributions" is downstream of this one.

**Action:** escalate this decision before public launch — it is the highest-leverage single item
on this list.

## 2. Missing GitHub community-health files

Currently folded into `README.md` sections only, which is fine pre-launch but does not populate
GitHub's own community-standards checklist or Security tab.

- [ ] `SECURITY.md` — GitHub recognizes this specially (populates the repo's Security tab, enables
  private vulnerability reporting). Notable specifically because AgentGate *is* a security
  product — shipping without a formal disclosure policy undercuts the pitch.
- [ ] `CODE_OF_CONDUCT.md` — cheap (e.g. adopt Contributor Covenant), standard, checked by GitHub's
  community profile.
- [ ] `CONTRIBUTING.md` as a real file, once the project is actually opened to outside
  contributions. `README.md` currently and correctly says it is not open yet — leave the inline
  section until that changes.

## 3. `.github/` automation that doesn't exist yet

- [ ] `.github/dependabot.yml` for the Go module + GitHub Actions — cheap, high-signal for a
  security-conscious project. Most useful once `go.mod` has real dependencies (Cedar, pgx, etc.
  land starting TASK-01-02) — low value while the module has zero dependencies.
- [ ] `.github/ISSUE_TEMPLATE/` + `.github/PULL_REQUEST_TEMPLATE.md` — not urgent pre-launch,
  standard polish.
- [ ] `CODEOWNERS` — worth adding once there is more than one active contributor.

## 4. `.golangci.yml`

`docs/DEVELOPMENT/CI_BASELINE.md` deliberately deferred a project-specific lint config until real
Go code existed ("so the ruleset can be chosen against real code rather than being guessed in
advance"). Real code now exists (`internal/config`, `internal/logging`, `internal/httpserver`).
Worth revisiting so lint rules are chosen deliberately rather than left at golangci-lint's bare
defaults indefinitely as more packages land.

## 5. Repository metadata (not file changes)

GitHub UI only, no commit needed: repo description, topics/tags, social preview image, "About"
section link. Part of "looks professional" in a directory listing / search result, separate from
anything version-controlled.

## 6. Repository location / naming — deliberately deferred, do not touch

**Current state (known, intentional):** `agentgate/go.mod`'s module path is
`github.com/Dynamisch-LLC/agentgate`, but the actual `git remote origin` is
`git@github.com:rushi-dynmsh/agentgate-dev.git`, and `README.md`/`docs/DEVELOPMENT/SETUP.md`'s
clone instructions and CI badge point at `rushi-dynmsh/agentgate-dev` to match the real remote.
These two paths disagree with each other.

This was raised explicitly (2026-09-09) and the Team Lead's decision was: **leave it as-is, move
the repository to the `Dynamisch-LLC` org after development is done, and do not let this block
anything in the meantime.** A future agent should not "fix" this mismatch on its own — it is not a
bug, it is a known pending decision.

When the move actually happens, it needs:
- [ ] transfer/recreate the repo under `github.com/Dynamisch-LLC/agentgate` (or whatever name is
  chosen — see naming note below);
- [ ] update `git remote origin`;
- [ ] update `README.md`'s badges + clone command, and `docs/DEVELOPMENT/SETUP.md`'s clone
  command/URL, to match;
- [ ] confirm `agentgate/go.mod`'s module path still matches the final location (it already
  anticipates `Dynamisch-LLC/agentgate` — only revisit if the final org/repo name differs).

**Separate naming question, also deferred:** the current repo is named `agentgate-dev`, which
reads like an internal/staging name. Worth deciding the permanent public name (`agentgate`?)
before it accumulates external clones/stars/forks under the current name — GitHub redirects old
names after a rename, but it is still a visible, one-time flip best done deliberately rather than
late.

## Decision protocol for this file

Same spirit as `docs/DECISIONS/OPEN_DECISIONS.md`: an item here is not an invitation to guess. When
an item is actioned, update this file (check it off or remove it) in the same change, so it stays
an accurate picture of what remains.
