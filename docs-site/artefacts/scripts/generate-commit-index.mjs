#!/usr/bin/env node
// Generate docs/history/commit-index.md from the monorepo git log.
//
// Run from the REPO ROOT (docs-site/../), not from docs-site/:
//
//   node docs-site/artefacts/scripts/generate-commit-index.mjs
//
// The index lists the FULL product history from HEAD (non-merge commits,
// newest first) and links every hash to its GitHub permalink:
//
//   https://github.com/rushi-dynmsh/agentgate-dev/commit/<hash>
//
// SELF-REFERENCE LOOP — docs-site commits are EXCLUDED from the index. If the
// index included them, regenerating it would itself create a docs-site commit,
// which would then appear in the index, forcing another regeneration, forever.
// Excluding the docs-site path keeps product commits flowing into the index
// while making the regeneration commit itself invisible.
//
// Links only resolve once the branch is pushed, so the script checks the live
// remote tip and WARNs if HEAD is ahead of it (unpublished commits would 404).
//
// Output is written to docs-site/docs/history/commit-index.md. The generated
// file is committed so the static site reflects the git history at generation
// time; regenerate whenever meaningful commits land. Safe to re-run — it
// overwrites only its own target file.

import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, join, relative } from "node:path";
import { writeFileSync } from "node:fs";

const scriptDir = dirname(fileURLToPath(import.meta.url));

// The repo root = three levels above this script (docs-site/artefacts/scripts/).
const repoRoot = join(scriptDir, "..", "..", "..");
const outFile = join(repoRoot, "docs-site", "docs", "history", "commit-index.md");

const REPO_BASE = "https://github.com/rushi-dynmsh/agentgate-dev";
const REMOTE = "origin";
const PUBLISH_BRANCH = "development";

function gitTry(args) {
  try {
    return execFileSync("git", args, {
      cwd: repoRoot,
      encoding: "utf8",
      stdio: "pipe",
    }).trim();
  } catch {
    return null;
  }
}

function run() {
  const warnings = [];

  const head = gitTry(["rev-parse", "--verify", "HEAD^{commit}"]);
  if (!head) throw new Error("cannot resolve HEAD — not a git repo?");

  // Sanity check that published history covers HEAD, else links would 404.
  const remoteLine = gitTry(["ls-remote", REMOTE, `refs/heads/${PUBLISH_BRANCH}`]);
  if (remoteLine) {
    const remoteSha = remoteLine.split(/\s+/)[0];
    if (remoteSha !== head) {
      const ahead = gitTry(["rev-list", "--count", `${remoteSha}..HEAD`]);
      warnings.push(
        `HEAD (${head.slice(0, 7)}) is ahead of the remote ${REMOTE}/${PUBLISH_BRANCH} ` +
          `(${remoteSha.slice(0, 7)}) by ${ahead ?? "?"} commit(s) — those hashes will 404 ` +
          `on GitHub until pushed. Push, then regenerate.`,
      );
    }
  } else {
    warnings.push(
      `Could not query ${REMOTE}/${PUBLISH_BRANCH} (offline?) — links not verified.`,
    );
  }

  // NOTE: --pretty=%h%x09%cs%x09%an%x09%s keeps subjects free of emoticons/
  // control chars that would break the markdown table. git log is newest-first
  // by default; --no-merges keeps only real commits. Pathspec EXCLUDES docs-site
  // so maintenance commits don't feed the self-reference loop described above.
  const log = gitTry([
    "log",
    "--pretty=format:%h%x09%cs%x09%an%x09%s",
    "--date-order",
    "--no-merges",
    head,
    "--",
    ".",
    ":(exclude)docs-site",
  ]);
  if (log === null) throw new Error("git log failed — cannot generate commit index.");

  const rows = log
    .split("\n")
    .filter(Boolean)
    .map((line) => {
      const [hash, date, author, ...rest] = line.split("\t");
      const subject = rest.join("\t").replace(/\|/g, "\\|");
      const short = hash.toUpperCase();
      const link = `${REPO_BASE}/commit/${hash}`;
      return `| [\`${short}\`](${link}) | ${date} | ${author.replace(/\|/g, "\\|")} | ${subject} |`;
    });

  const count = rows.length;
  const generatedAt = new Date().toISOString().slice(0, 10);
  const headShort = head.slice(0, 7);

  const md = `# Commit Index

<!-- GENERATED FILE — do not edit by hand. -->
<!-- Source: node docs-site/artefacts/scripts/generate-commit-index.mjs (run from repo root). -->

Automatically transcribed from the monorepo \`git log\` (non-merge **product**
commits — docs-site maintenance commits are excluded so the index does not feed
its own regeneration) at generation time, **newest first**, for all product commits
reachable from \`HEAD\` (\`${headShort}\`). ${count} commits listed; each hash links
to its GitHub permalink. Regenerate with:

\`\`\`bash
node docs-site/artefacts/scripts/generate-commit-index.mjs
\`\`\`

| Commit | Date | Author | Subject |
|---|---|---|---|
${rows.join("\n")}

---
*Generated ${generatedAt} — ${count} commits, tip \`${headShort}\`.*
`;

  writeFileSync(outFile, md, "utf8");
  console.log(
    `wrote ${relative(process.cwd(), outFile)} (${count} commits; tip ${headShort})`,
  );
  for (const w of warnings) {
    console.warn(`WARN: ${w}`);
  }
}

run();