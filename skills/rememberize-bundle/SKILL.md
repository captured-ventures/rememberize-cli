---
name: rememberize-bundle
description: Use when preparing a memory bundle for import into rememberize (a portable memory system for AI). Applies when the user wants to concat Claude auto-memory files into a single upload-ready markdown file, or merge memory-style markdown from multiple locations (Claude projects, directories, files, globs). Also triggers on "bundle my memory", "prepare for rememberize import", "rememberize onboarding upload".
user_invocable: true
---

# rememberize-bundle

Concatenate Claude auto-memory markdown (and/or arbitrary memory-style markdown) into a single file that rememberize's onboarding importer can parse — one memory per source file, full frontmatter preserved.

Claude stores per-project auto-memory at `~/.claude/projects/<slug>/memory/*.md` plus a `MEMORY.md` index. The index only yields shallow memories on upload; the individual files yield rich ones. Users reach for the index by default — this skill does the right thing for them.

## When to invoke

- "Bundle my Claude memory for rememberize"
- "Prepare an upload file for rememberize onboarding"
- "Concat my memory files into one markdown"
- User is on rememberize onboarding step 3 and asks what to upload

## How it works

This skill drives a portable `bundle.sh` script (POSIX sh; runs on macOS, Linux, Git Bash, WSL). The conversation is elicitation-first, script-second.

### Step 1 — elicit the branch

Ask the user exactly:

```
What do you want to bundle?
  1) A single Claude project's auto-memory (default)
  2) Multiple locations (Claude projects, directories, files, globs)
```

### Branch A — single project

1. List `~/.claude/projects/` entries with a `memory/` dir, sorted by most-recently-modified `memory/`
2. If the user's cwd maps to an entry (encoding: absolute path with separators replaced by `-`, e.g. `C:\path\to\x` → `C--path-to-x` or `/home/u/x` → `-home-u-x`), offer it as the default pick
3. Confirm or let the user pick a different project
4. Invoke:
   ```
   sh "$(dirname "$0")/scripts/bundle.sh" --claude-project "$HOME/.claude/projects/<slug>"
   ```
5. The script reads all `*.md` files in the project's `memory/` except `MEMORY.md`, keeps files that start with `---` (frontmatter), and reports any skipped.

### Branch B — multi-source

1. Loop: ask for source type + location, show running count after each add, then "add another or done?"
2. Source types the script accepts (pass via repeated flags):
   - Claude auto-memory project → `--claude-project <path>`
   - Directory (ask: recursive y/n) → `--source <dir>` (recursive) or `--source-flat <dir>` (top-level only)
   - Single file → `--source <file>`
   - Glob pattern → `--source '<glob>'` (quote to defer shell expansion)
3. On done, invoke `bundle.sh` with all collected flags. The script dedupes by `name:` frontmatter (first wins), warns on conflicts, preserves source order.

### Shared tail

The script writes to `~/.claude/bundles/<slug>-<YYYY-MM-DD>.md` (slug = single-project name or `multi`). Print:
- Output path
- Per-source file counts (branch B) or total count (branch A)
- Byte count
- Dedupe summary (branch B only)
- Hint: "Upload this on rememberize onboarding step 3, or via the CLI: `rememberize import <path>`"

## Script location

`scripts/bundle.sh` lives alongside this SKILL.md. When invoking, resolve the script path relative to the skill's own directory. If the skill was installed to `~/.claude/skills/rememberize-bundle/`, the script is at `~/.claude/skills/rememberize-bundle/scripts/bundle.sh`.

## Examples

**Default invocation (bundle current project's memory):**

```
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh
```

**Explicit single project:**

```
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh \
  --claude-project "$HOME/.claude/projects/D--new-projects-rememberize-app"
```

**Multi-source:**

```
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh \
  --claude-project "$HOME/.claude/projects/D--new-projects-rememberize-app" \
  --source "$HOME/notes/memory" \
  --source "/d/code/**/CLAUDE.md" \
  --out "$HOME/.claude/bundles/my-bundle.md"
```

## Edge cases

- No project matches cwd → skip default, show picker
- Memory dir missing or empty → script exits non-zero with a clear error; surface it
- Non-UTF-8 file → script skips with warning, continues
- Glob matches zero files → script warns, continues
- Duplicate `name:` in branch A → include all verbatim (single source, preserve)
- Duplicate `name:` in branch B → dedupe, first wins, warn

## Portability

This skill is rememberize-branded but generic underneath. The only rememberize-specific line is the final hint. A reader cloning this skill for a different memory system should be able to adapt it by editing the hint line alone.
