# CLAUDE.md — rememberize-cli

Project context for any Claude Code session that opens this repo. Keep concise.

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
