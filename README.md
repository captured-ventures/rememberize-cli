# rememberize-cli

Command-line client and Claude Code skill for [rememberize](https://rememberize.app).

## What is rememberize?

Portable, multi-directional memory for AI. Your decisions, notes, and working context live in one place; any AI agent you give a connection to can recall them, and (with the right permissions) write new ones back. Built for engineers and builders who want their AI tools to stop forgetting.

This repo ships the CLI half: a single-binary command-line client, plus a Claude Code skill that bundles your existing Claude auto-memory into an importable file.

## Install

### One-liner (macOS, Linux, Windows via Git Bash)

```sh
curl -sSL https://rememberize.app/install.sh | sh
```

The installer detects your OS and architecture, downloads the latest release, and places `rememberize` on your `PATH` (defaults to `~/.local/bin` — make sure that's on `PATH`).

If `rememberize.app/install.sh` is temporarily unreachable, use the raw fallback:

```sh
curl -sSL https://raw.githubusercontent.com/captured-ventures/rememberize-cli/main/install.sh | sh
```

### Homebrew (macOS, Linux)

```sh
brew install captured-ventures/rememberize/rememberize
```

### Scoop (Windows)

```powershell
scoop bucket add rememberize https://github.com/captured-ventures/scoop-rememberize
scoop install rememberize
```

### Manual

Grab a binary for your platform from [Releases](https://github.com/captured-ventures/rememberize-cli/releases/latest), unpack, and place on your `PATH`.

## Quick start

```sh
rememberize pair <code-from-dashboard>    # exchange one-time code for an API key
rememberize recall "test query"           # query your memories
rememberize --help                        # full command reference
```

The `<code-from-dashboard>` is generated on [rememberize.app](https://rememberize.app) under **Connections → New connection**. After pairing, the CLI writes its config to `~/.rememberize/config.toml` — no further setup needed.

## Claude Code skill

The `rememberize-bundle` skill helps you prepare an initial memory import from your Claude Code auto-memory files. See [`skills/rememberize-bundle/README.md`](skills/rememberize-bundle/README.md) for install and usage.

## Platform

- Home: [rememberize.app](https://rememberize.app)
- Docs: [docs.rememberize.app](https://rememberize.app/docs) *(coming)*
- Dashboard: [platform.rememberize.app](https://platform.rememberize.app)

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). In short: small PRs, matching commit style (`feat(cli): ...`, `fix(pair): ...`), tests green before review. This repo is CLI + skill + install-pipeline only; server-side changes go to the main (private) repo — see `CONTRIBUTING.md` for how to route those reports.

## License

MIT. See [`LICENSE`](LICENSE).
