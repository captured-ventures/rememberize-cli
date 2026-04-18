# rememberize-bundle

A Claude Code skill that concatenates memory-style markdown (Claude auto-memory, arbitrary directories, globs, or single files) into a single upload-ready markdown file. Designed for [rememberize](https://rememberize.app)'s onboarding importer, but the underlying script is generic.

## Why

Claude's auto-memory stores per-project memories as individual frontmatter files under `~/.claude/projects/<slug>/memory/*.md`, plus a `MEMORY.md` index. If you upload the index to rememberize directly, you get shallow memories (one per bullet). Upload the bundle this skill produces and you get rich memories (one per file, full body preserved).

## Install

### One-liner

```sh
mkdir -p ~/.claude/skills/rememberize-bundle/scripts
curl -sSL https://raw.githubusercontent.com/captured-ventures/rememberize-cli/main/skills/rememberize-bundle/SKILL.md \
  -o ~/.claude/skills/rememberize-bundle/SKILL.md
curl -sSL https://raw.githubusercontent.com/captured-ventures/rememberize-cli/main/skills/rememberize-bundle/scripts/bundle.sh \
  -o ~/.claude/skills/rememberize-bundle/scripts/bundle.sh
chmod +x ~/.claude/skills/rememberize-bundle/scripts/bundle.sh
```

### Alternative: clone + symlink

```sh
git clone https://github.com/captured-ventures/rememberize-cli.git ~/src/rememberize-cli
ln -s ~/src/rememberize-cli/skills/rememberize-bundle ~/.claude/skills/rememberize-bundle
```

Cloning keeps the skill in sync with upstream (`git pull` to update).

## Bundle Skill

Once installed, Claude Code will pick up the skill automatically. Trigger it by asking Claude something like:

- "Bundle my Claude memory for rememberize"
- "Prepare an upload for rememberize onboarding"
- "Concat my memory files into one markdown"

Claude will elicit whether you want a single project or multiple sources, then invoke the underlying script.

## Direct script usage

You don't have to use it through the skill — `bundle.sh` stands alone.

### 1. Bundle the current Claude project's memory

```sh
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh
```

Detects the current working directory, maps it to the matching `~/.claude/projects/<encoded-cwd>/`, and bundles the files in its `memory/` dir.

### 2. Bundle a specific Claude project

```sh
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh \
  --claude-project "$HOME/.claude/projects/D--new-projects-rememberize-app"
```

### 3. Bundle multiple Claude projects (dedupes by `name:`)

```sh
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh \
  --claude-project "$HOME/.claude/projects/D--new-projects-rememberize-app" \
  --claude-project "$HOME/.claude/projects/D--new-projects-promptmark"
```

### 4. Mix sources — projects, directories, globs

```sh
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh \
  --claude-project "$HOME/.claude/projects/D--new-projects-rememberize-app" \
  --source "$HOME/notes/memory" \
  --source "/d/code/**/CLAUDE.md" \
  --out "$HOME/.claude/bundles/my-bundle.md"
```

### 5. Flat directory (no recursion)

```sh
sh ~/.claude/skills/rememberize-bundle/scripts/bundle.sh \
  --source-flat "$HOME/notes/top-level-only"
```

## Flags

| Flag | Description |
| --- | --- |
| `--source <path>` | File, directory (recursive), or glob. Repeatable. |
| `--source-flat <dir>` | Directory, top-level `*.md` only. Repeatable. |
| `--claude-project <path>` | Claude project dir (expects `<path>/memory/`). Repeatable. |
| `--out <path>` | Output file. Defaults to `~/.claude/bundles/<slug>-<YYYY-MM-DD>.md`. |
| `--slug <name>` | Override the slug in the default output filename. |
| `-h`, `--help` | Show help. |

## Output

A single markdown file with:

- A header comment identifying the bundle (date + list of sources)
- Each source file concatenated verbatim, full frontmatter preserved, one blank-line separator between entries
- In multi-source mode: dedupe by frontmatter `name:` (first occurrence wins; duplicates logged to stderr)

Then printed to stdout:

- Output path
- File count (per-source breakdown in multi-source mode)
- Byte count
- Dedupe summary (multi-source only)
- Upload hint:
  ```
  Upload this on rememberize onboarding step 3, or via the CLI:
      rememberize import "<path>"
  ```

## Platforms

POSIX `sh`. Tested on macOS, Linux, Git Bash for Windows, and WSL. No bash-isms, no GNU-only flags.

## Testing

A basic harness lives at `scripts/bundle_test.sh`:

```sh
sh ~/.claude/skills/rememberize-bundle/scripts/bundle_test.sh
```

It creates fake Claude-project layouts in a `mktemp -d` dir and asserts on the bundle output (frontmatter preservation, MEMORY.md exclusion, dedupe behaviour, zero-match glob warnings).

## Portability

The skill is rememberize-branded but the script is generic. The only rememberize-specific line is the final hint. If you're using a different memory system, fork the skill and edit that one line.

## License

MIT — same as the parent `rememberize-cli` repo.
