# Changelog

All notable changes to the `rememberize` CLI are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] — 2026-04-29

First public release. The CLI was previously a private subcommand of the `rememberize` server monorepo; this release publishes it as a standalone, distributable Go binary with installable taps.

### Added

#### Command surface (13 commands)

- `pair <code>` — exchange a one-time code from the dashboard for a permanent API key. Auto-detects integration target (Claude Code, Cursor, CLI-only, generic paste) from the server response and writes the appropriate config file.
- `add <content>` — create a new memory. Reads from `-` (stdin) or argv. `--ns`, `--type`, `--meta`, `--expires` flags.
- `recall <query>` — semantic / vector search.
- `search <query>` — full-text search.
- `list` / `get` / `rm` — memory CRUD.
- `namespaces` — list namespaces.
- `connections` — list paired clients.
- `audit` — view the audit log with `--action` / `--connection-id` / `--limit` filters.
- `keys create|list|revoke` — manage API keys for external clients.
- `import <file>` — bulk import from Claude `MEMORY.md`, ChatGPT JSON export, or CSV. `--dry-run` for preview.
- `export` — bulk export in `memory-md`, `json`, or `csv`. Stream to stdout or `--output` file.
- `config` / `config set <key> <value>` — show or update CLI config.
- `completion bash|zsh|fish|powershell` — shell completion scripts (provided via fang).

#### Output presentation

- Styled `--help` and `--version` rendering via [`charmbracelet/fang`](https://github.com/charmbracelet/fang) — structured USAGE / COMMANDS / FLAGS sections in real terminals.
- Lipgloss-rendered tables for `list`, `namespaces`, `connections`, `audit`, `keys list`, and `recall`/`search` results when stdout is a terminal. Rounded borders, subdued color-8 styling matching the rememberize.app dashboard tone.
- Plain `text/tabwriter` fallback when stdout is piped, redirected, or otherwise non-TTY — output stays parseable by `cut`/`awk`/`jq`.
- `--json` flag on every command for machine-readable output (stdout pristine; logs suppressed unless `-v`).

#### Logging

- [`charmbracelet/log`](https://github.com/charmbracelet/log) leveled logger.
- `--verbose` / `-v` (count): default `WARN`; `-v` → `INFO`; `-vv` → `DEBUG`.
- `--quiet`: clamps to `ERROR` only; overrides any `-v`.
- Logs always go to stderr; stdout is reserved for command data.
- Network HTTP calls instrumented with `DEBUG`-level entry/exit/error logs for diagnosing pair/recall/list behavior.

#### Interactive prompts (pair flow)

- `pair` server selection now uses [`charmbracelet/huh`](https://github.com/charmbracelet/huh) `Select` + `Input` in TTY mode — arrow-key picker for production / localhost / custom URL.
- `pair` post-pair namespace selection uses `huh.Select` with explicit "skip" option.
- F24 preflight (`Found existing .mcp.json — add rememberize MCP entry here?`) uses `huh.Confirm` with explicit Yes/No buttons.
- All three prompts fall back to legacy bufio implementations in non-TTY contexts (CI, scripted use, piped input) so existing test contracts and automation paths are preserved.

#### Pre-flight safety

- `NewClient()` now refuses to construct a client without both `auth.api_key` and `auth.api_url` configured, surfacing an actionable error before any network call:
  > No API key configured. Run `rememberize pair <code>` to authenticate — get a code from https://rememberize.app/app/connections/new — or set `REMEMBERIZE_API_KEY=<key>` for scripted use.
- F24 (PR #6 in pre-release): `pair` prompts before merging into an existing `.mcp.json` / Cursor MCP config; on decline, falls through to a printable paste block.

#### Distribution

- Cross-platform release binaries via [goreleaser](https://goreleaser.com/) on tag push: `darwin/{amd64,arm64}`, `linux/{amd64,arm64}`, `windows/amd64`. All builds are `CGO_ENABLED=0`, no C deps, no platform-specific build tags.
- Homebrew formula auto-published to [`captured-ventures/homebrew-rememberize`](https://github.com/captured-ventures/homebrew-rememberize) on each release.
- Scoop manifest auto-published to [`captured-ventures/scoop-rememberize`](https://github.com/captured-ventures/scoop-rememberize) on each release.
- Styled POSIX-sh `install.sh` for one-line install via `curl -sSL https://rememberize.app/install.sh | sh`. Pure-ANSI styling matches the lipgloss aesthetic; respects `NO_COLOR=1` and non-TTY stderr for opt-out.

#### Skill bundle

- `rememberize-bundle` Claude Code skill at `skills/rememberize-bundle/` for concatenating Claude `MEMORY.md` files from multiple project directories into a single upload-ready bundle.

### Configuration

- Config file at `~/.rememberize/config.toml` (TOML format, BurntSushi/toml).
- Env overrides: `REMEMBERIZE_API_URL`, `REMEMBERIZE_API_KEY`.
- Keys: `auth.api_url`, `auth.api_key`, `defaults.namespace`, `defaults.type`, `defaults.format`.

### Notes for users

- **`-v` short flag** is bound to `--verbose`. `--version` is still available in long form. (Cobra's default `-v`-for-`--version` binding was overridden — universal CLI convention favors `-v` for verbose.)
- API keys are masked on display by `rememberize config` (only the last 4 characters are shown).

### Known limitations

- Windows ARM64 builds are not currently published (the `install.sh` errors cleanly on that combination).
- macOS builds are not signed/notarized; expect Gatekeeper prompts on first launch. Workaround: `xattr -d com.apple.quarantine /usr/local/bin/rememberize`.
- Per-invocation latency on Windows is higher than other platforms (Defender on-access scanning + lack of HTTP keepalive across CLI invocations). Daemon mode is on the roadmap.

[Unreleased]: https://github.com/captured-ventures/rememberize-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/captured-ventures/rememberize-cli/releases/tag/v0.1.0
