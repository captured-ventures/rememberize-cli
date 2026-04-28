# CLAUDE.md — rememberize-cli

Project context for any Claude Code session that opens this repo. Keep concise.

> **First time in this repo?** Start with [`docs/agent-briefing.md`](docs/agent-briefing.md) — it walks through orientation (read playbook → recall rememberize-dev context → file first memory note → verify install end-to-end). After the first session, this file is your day-to-day reference.

## What this repo is

Public-facing CLI and Claude Code skill for [rememberize](https://rememberize.app), a portable multi-directional memory system for AI.

Two shipped artifacts:

- `rememberize` — a pure-Go binary. Pairs a machine against the rememberize platform, reads and writes memories, imports/exports bundles.
- `skills/rememberize-bundle/` — a Claude Code skill that concatenates Claude auto-memory files into an upload-ready bundle.

## Two-repo architecture

This is the **public CLI half**. The server lives in a separate private repo: `ironystock/rememberize`.

- The CLI imports nothing from the server repo.
- It talks to `https://platform.rememberize.app` (and its MCP surface) via HTTPS only.
- DTOs the CLI needs (pair exchange payload, memory list responses) are redefined locally under `cmd/rememberize/` — intentional duplication, keeps the public repo free of private-source bleed.
- Any protocol change between CLI and server is coordinated with Brad and the main repo, not landed unilaterally here.

## Tech stack

- Go 1.25+ (see `go.mod`)
- [cobra](https://github.com/spf13/cobra) for CLI structure
- [BurntSushi/toml](https://github.com/BurntSushi/toml) for config
- Pure Go: `CGO_ENABLED=0` everywhere. No C deps, no platform-specific build tags.

## Build / test / lint

Use the Makefile targets:

```
make build            # build ./bin/rememberize
make test             # go test -race ./...
make lint             # golangci-lint run
make install-dev      # build + copy into ~/.local/bin
make release-snapshot # local goreleaser dry run (needs goreleaser installed)
```

Tests run directly on the host — no dev container required (pure Go, no CGO). CI runs the same three targets on Linux, macOS, Windows.

## Conventions

- **Commit style**: Conventional Commits. Scope matches area:
  - `feat(cli): ...`, `fix(pair): ...`, `docs(install): ...`, `chore(release): ...`, `test(skill): ...`, `ci: ...`
- **PR size**: small. Prefer three reviewable PRs over one monster.
- **Tests required** before merge. CI blocks on `go test`, `go vet`, `golangci-lint`, and cross-platform build.
- **Branch naming**: `feat/...`, `fix/...`, `docs/...`, `chore/...`. Target `main`.
- **Releases**: tag `vX.Y.Z` after a meaningful PR lands; goreleaser cuts binaries + updates Homebrew/Scoop taps automatically.

## Pointers

- `AGENTS.md` — playbook for the dedicated owner agent (scope, cadence, escalation, PR template).
- `docs/agent-briefing.md` — one-time first-run orientation for a freshly spawned owner agent.
- `README.md` — human-facing install + quick start.
- `skills/rememberize-bundle/README.md` — skill install + usage.
- `CONTRIBUTING.md` — for outside contributors.

## Memory awareness (rememberize MCP)

This repo ships a `.mcp.json` that wires the rememberize MCP server into any Claude Code session opened here. Two namespaces are relevant:

- **`rememberize-dev`** — Brad's project memory. The owner agent has **read-only** access via scoped connection grants (`recall`, `search`, `list` only; no `remember`, no `update`, no `forget`). Use it to look up prior decisions: architecture, friction log, sprint history, "why is X the way it is".
- **`rememberize-cli-agent`** — the owner agent's own working memory. Full read/write. Use it to log what you tried, what worked, what broke, what's queued — so future sessions don't repeat yourself.

Usage sketch:

```
mcp__rememberize__recall    namespace="rememberize-dev"        query="CLI friction F18"
mcp__rememberize__search    namespace="rememberize-dev"        query="pair exchange protocol"
mcp__rememberize__remember  namespace="rememberize-cli-agent"  content="..."
mcp__rememberize__recall    namespace="rememberize-cli-agent"  query="last release notes draft"
```

**Always write working memory to `rememberize-cli-agent`.** Never try to `remember` / `update` / `forget` on `rememberize-dev` — the key is scoped to reject those calls, and it would be the wrong namespace anyway.

### API key supply

`.mcp.json` uses the literal placeholder `${REMEMBERIZE_API_KEY}`. Claude Code's MCP harness does not reliably interpolate env vars inline; the agent runtime must supply the real key. Options:

1. Export `REMEMBERIZE_API_KEY=...` in the shell before launching Claude Code — some MCP wrappers pass it through to the HTTP transport.
2. Brad pastes the real key into a local-only `.mcp.json` override (untracked) on the agent's host.
3. The agent runtime injects it via Claude Code settings.

Never commit a real key. `.gitignore` already covers common env files.

## Lessons & gotchas (hard-won from the bootstrap sprint)

These are things future-you wants to know before they bite. Add to this list when you learn something non-obvious.

### Go toolchain alignment

- **`golangci-lint` version is load-bearing when Go minor versions shift.** The bootstrap failed CI three times before we landed on the right combination: `golangci/golangci-lint-action@v8` + pinned `version: v2.11.4`. Earlier versions (v1.64.8, v2.1.6) were built with Go 1.24 and refused configs targeting Go 1.25+. When Go bumps a minor version, check that the pinned lint version has a release built against the new toolchain before merging.
- **Pin, don't `latest`.** Auto-latest picks up whatever GitHub flagged as `Latest` at run time — which may not match your Go target. Explicit version pin in CI is worth the occasional manual bump.

### `.golangci.yml` v2 vs v1

This repo is on the v2 config format (`version: "2"` at the top). Legacy v1 stanzas like `linters.disable-all: true` and enabling `gofmt` as a linter will silently lint nothing. v2 uses `linters.default: none` and moves `gofmt` into a separate `formatters:` section.

### errcheck in CLI code — exclude the usual noise

`.golangci.yml` already excludes `fmt.Fprint`, `fmt.Fprintf`, `fmt.Fprintln`, `(io.Closer).Close`, `(*os.File).Close`, `(*text/tabwriter.Writer).Flush`. Checking these in CLI code produces 20+ false positives and zero insight. If you add a new area that writes to stdout/stderr, you do NOT need to check those returns.

What you DO still need to check (the bootstrap bit us): **`multipart.Writer.Write`, `WriteField`, `Close`**. A silently-dropped error here produces malformed upload bodies. errcheck catches it; don't exclude it.

### Pair protocol — backwards-compat sentinel is load-bearing

`cmd/rememberize/pair.go` sends `client_name: "@auto"` + `hostname: <os.Hostname()>`. Server composes the full display name from its stored `client_hint` + the received hostname. **Older CLI versions still in production send a pre-composed `client_name`** (not `@auto`). The server path keeps both branches alive — if you ever see code that drops the `@auto` check, you've broken every currently-installed binary. Don't. If you genuinely need to change the sentinel, coordinate with the server side first (main repo).

The OTK exchange response includes `connection.name` and `connection.config_target` — **trust those fields.** Use `connection.name` for display, use `connection.config_target` to decide which config file the client writes (claude-code → `.mcp.json`, cursor → `~/.cursor/mcp.json`, cli → no MCP config, generic → print to stdout). Don't resurrect the cwd-sniffing `detectClient()` that used to live here; it was the whole point of F7+F9.

**A user-visible signal that the sentinel path didn't fire:** if a freshly-paired connection appears in the dashboard literally named `@auto` AND the CLI prints "No known integration — paste this into your MCP client", the server you hit is on a pre-`server-composed-name` build (i.e., older than `ironystock/rememberize` PR #45). The CLI is doing the right thing; the server didn't translate the sentinel. Surface it to Brad as a deploy issue, not a CLI bug.

### F24 — preflight before clobbering an existing MCP config

`runPair` now prompts before merging into an existing `.mcp.json` / Cursor config: "Found existing X — add rememberize MCP entry here? [Y/n]". On `n`, it falls through to the `generic` branch and prints a paste-able config block instead of writing the file.

**Why this exists:** the dogfooding scenario was Brad pairing from inside a project repo that had a tracked `.mcp.json`, getting the new entry merged into it silently, and only seeing "Config written: .mcp.json" *after* the fact. The preflight is the user's chance to back out cleanly — say, by Ctrl-C-ing and `cd`-ing somewhere harmless first.

**Don't strip this prompt.** It's interactive in the same way the namespace-default prompt is, and tests in `pair_flow_test.go` (`TestPair_F24_*`) pin its behavior. If you need a non-interactive path for CI/scripts, add a `--yes` flag — don't remove the prompt.

**Skip rule:** prompt only fires when the target file *exists with non-zero size*. A missing or empty file is the green-field case and proceeds without prompting.

### Two-repo discipline

You cannot import `github.com/ironystock/rememberize/...` — that's the private server repo. If you find yourself reaching for it, you're about to duplicate logic (acceptable, local copies are fine) or design a protocol change (not acceptable here — file a `[MAIN-REPO]` issue and @-mention Brad).

The `internal/transfer/` package here is a minimal local copy, not a sync target. It has its own `Memory` struct (field set reflects only wire formats the CLI parses/emits). If the main repo evolves its `internal/memory.Memory`, this copy does NOT need to follow unless import/export wire formats themselves change.

### IDE diagnostics on a Windows dev host

If you open this repo alongside the private rememberize main repo (`D:\new-projects\rememberize.app\`), gopls may flag spurious "undefined: X" errors on files in `rememberize-cli` — the main repo's workspace doesn't see the sibling module. **These are not real.** Source of truth is `go build ./...` + `go test ./...` + CI, not the IDE. Add a `go.work` at the parent directory if you want cross-repo IDE coherence; don't change source to silence the warnings.

### Release mechanics

- goreleaser cuts on tag push (`v*`). Tags are append-only — don't retag an existing version; cut `v0.x.y+1`.
- `RELEASE_PAT` secret is what lets the release workflow push formula/manifest updates to the sibling `homebrew-rememberize` + `scoop-rememberize` repos. `GITHUB_TOKEN` alone can't do cross-repo pushes.
- Homebrew tap uses the `brews:` goreleaser block (deprecation notice on `brews:` → `homebrew_casks:` exists; we stayed on `brews:` because it still works and `casks` is for GUI apps).
- Windows builds skip `arm64` (goreleaser `ignore: windows/arm64` + `install.sh` errors cleanly on that combo). Add it if demand arrives.

### Don't trust runbooks blindly — re-check against current dashboard

The cross-repo provisioning runbooks (e.g. `~/.claude/plans/rememberize-cli-agent-bootstrap.md`) drift fast because the dashboard ships frequently. Examples seen during dogfooding: the runbook's "Step 2 — name the connection" referred to a wizard step that was deleted when the server-composed-name path landed; "click + New namespace" referred to a button that was hardcoded `disabled` for several weeks.

**Rule:** before walking a user through a multi-step provisioning plan, spot-check the steps against the current dashboard templates in `cmd/web/*.templ` (or live UI). If they disagree, the dashboard wins — file a friction note, then either update the runbook or pause until the gap is fixed. Don't assume the runbook is current just because it was right last time.

### Misc

- **Binary name is `rememberize`, not `rememberize-cli`.** The repo is `rememberize-cli` for namespace clarity, but the built binary on disk is `rememberize`. Homebrew formula, Scoop manifest, and install.sh all reflect this.
- **Pure Go, `CGO_ENABLED=0` everywhere.** Don't add a CGO dep without a very good reason — cross-compile-for-5-platforms-in-CI becomes a mess.
- **README's canonical install URL is `https://rememberize.app/install.sh`** (served via CF Pages redirect). Raw-GitHub fallback works today but is ugly in docs. If the redirect breaks, fix the redirect, don't edit the README back to raw URLs.
