# Contributing

Thanks for your interest in `rememberize-cli`. This is a small, focused repo; most PRs are welcome if they match the scope below and arrive in a reviewable size.

## Scope

This repo owns:

- The `rememberize` CLI binary (`cmd/rememberize/`)
- Import/export format parsers (`internal/transfer/`)
- The `rememberize-bundle` Claude Code skill (`skills/rememberize-bundle/`)
- The install script (`install.sh`), release pipeline (`.goreleaser.yml`, GitHub Actions), and docs

It does **not** own:

- Server-side behavior of the rememberize platform
- The pair-exchange wire protocol (coordinated with the main repo)
- Anything on `platform.rememberize.app` or `rememberize.app` beyond the CLI's view

If your bug report is a server-side issue, or your feature request requires a protocol change, flag it in an issue titled `[MAIN-REPO] ...` so the maintainer can route it to the right repo.

## Filing a good issue

A useful bug report has:

- **What happened** (exact command, exact error, any relevant output).
- **What you expected.**
- **Your platform**: OS + arch (`uname -a` on macOS/Linux; `ver` or `systeminfo` on Windows).
- **CLI version**: `rememberize --version`.
- **Reproduction steps** if you can get them.

For feature requests: describe the problem first, the proposed solution second. Small scoped asks are easier to land than big rewrites.

## Submitting a PR

1. **Open an issue first** for anything non-trivial so we can agree on approach before you spend time.
2. **Branch from `main`**. Name: `feat/...`, `fix/...`, `docs/...`, `chore/...`.
3. **Small, focused changes.** If a PR grows past ~300 lines, split it.
4. **Conventional Commits** for commit titles:
   - `feat(cli): add --json flag to recall`
   - `fix(pair): handle empty namespaces array`
   - `docs(install): clarify PATH setup on Windows`
   - `chore(release): bump goreleaser to v2.3`
   - `test(skill): cover empty-directory case`
   - `ci: cache Go modules across jobs`
5. **Tests required.** Add or update tests to cover the change. CI runs `go test -race`, `go vet`, `golangci-lint`, and a cross-platform build — all must be green.
6. **One topic per PR.** Mixed "drive-by fix plus feature" PRs take longer to review.

## Building and testing locally

You need Go 1.25+. No CGO, no build tags, no platform-specific deps.

```sh
make build            # build ./bin/rememberize
make test             # go test -race ./...
make lint             # golangci-lint run (install: https://golangci-lint.run/)
make install-dev      # build + copy into ~/.local/bin
```

Tests run directly on the host on macOS, Linux, and Windows.

## What kinds of PRs get accepted

Likely to land:

- Bug fixes with a clear reproduction, a test covering the bug, and a scoped fix.
- Docs improvements (install troubleshooting, clearer error messages, typo fixes).
- Small user-facing CLI ergonomics (flag additions, better error text, config-file improvements).
- Cross-platform compatibility fixes.
- Skill improvements in `skills/rememberize-bundle/`.

Unlikely to land without prior discussion:

- Large refactors.
- New top-level commands or subsystems.
- Changes to the wire protocol with the server (we can't land those unilaterally here).
- Dependencies on CGO or platform-specific build tags.
- New runtime dependencies.

If you're unsure, open an issue and ask before writing code.

## Code of conduct

Be kind. Assume good faith. We're here to ship useful software, not win arguments.

## License

By contributing, you agree that your contributions are licensed under the MIT License (see [`LICENSE`](LICENSE)).
