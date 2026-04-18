# Agent first-run briefing

This document exists once: the first time the dedicated owner agent is spawned against this repo. Brad will hand it to you as your initial input. Walk through every step, in order, before starting any work.

The goal of this session is **orientation**, not shipping. Get your bearings, write your first memory note, file any starter issues, pick the smallest valuable piece of work for session two.

## Step 1 — Read your playbook

Open `AGENTS.md`. Read it end to end. It is your role description: scope, cadence, escalation, PR template, release workflow.

## Step 2 — Read the project conventions

Open `CLAUDE.md`. Read it end to end. Conventions (commit style, PR size, build/test targets), the two-repo architecture, and the MCP memory-namespace rules (`rememberize-dev` read-only, `rememberize-cli-agent` read/write).

## Step 3 — Recall the sprint that birthed this repo

Call:

```
mcp__rememberize__recall  namespace="rememberize-dev"  query="EoC-1 Part 2"
```

You're looking for the plan memory that describes why this public repo was spun out, the two-repo split, the five success criteria, the friction items (F2, F4, F6, F7, F9, F12, F18), and the agent-ownership wiring. Read whatever comes back.

If the direct query is thin, try:

- `recall query="rememberize-cli public repo" namespace="rememberize-dev"`
- `recall query="agent ownership rememberize CLI" namespace="rememberize-dev"`
- `search query="two-repo split" namespace="rememberize-dev"`

## Step 4 — Recall CLI friction context

Call:

```
mcp__rememberize__recall  namespace="rememberize-dev"  query="CLI friction"
```

You should surface F18 (Windows per-invocation latency) at minimum, plus whatever else is in the friction log. These are your starter-task candidates.

Cross-reference with the "Known backlog / starter tasks" section of `AGENTS.md` — they should line up.

## Step 5 — Check the issues inbox

```
gh issue list
gh pr list
```

Both may be empty on first boot. That's fine — it means Brad hasn't filed anything preemptively and the starter backlog from `AGENTS.md` is yours to file.

## Step 6 — Write your first memory note

Call:

```
mcp__rememberize__remember  namespace="rememberize-cli-agent"  content="<your note>"
```

The note should cover:

- What you found during orientation (key decisions from `rememberize-dev`, any surprises).
- What you'll do in session two (the smallest piece of value you'll tackle first).
- Any questions for Brad that aren't urgent enough to be issues.

Keep it short, specific, and future-you-addressed — "next session: verify install.sh against a clean Ubuntu box, file issue if it breaks" beats "continue work".

## Step 7 — Start with the smallest piece of value

Your first real task: **verify the CLI actually installs and works on at least one platform.**

Suggested sequence:

1. Pull the latest `main`, `git tag --list` to see what version is out (`v0.1.0` after Brad cuts it).
2. On your own machine or a fresh container, run the documented one-liner:
   ```
   curl -sSL https://rememberize.app/install.sh | sh
   ```
   (Or the raw-GitHub fallback if the rememberize.app hop isn't wired up yet.)
3. `rememberize --version` — does it return the expected tag?
4. `rememberize pair <code>` against `platform.rememberize.app` — does it complete zero-config? (You'll need Brad to generate a one-time code in the dashboard, or a dedicated test connection.)
5. `rememberize recall "test"` — does it return something without further config commands?

If any step breaks, open an issue in this repo with the failure mode, the platform, and any output. Title format: `install: <short summary>` or `pair: <short summary>`.

If everything works, file the starter-backlog issues from `AGENTS.md` (F18, binary signing, `--vector` flag) and pick the one you want to tackle first.

## Step 8 — End-of-session note

Before you stop, write a second memory note to `rememberize-cli-agent` wrapping up session one: what you verified, what you filed, what's queued for session two. Next-you reads this first next time.

## Asking Brad

Brad reviews your PRs and reads issues you @-mention him on. For anything urgent-but-not-PR, drop a note in the issue you're working on; for anything non-urgent, stash it in your `rememberize-cli-agent` memory and surface the batch next time he checks in.

You're running this repo. Act accordingly.
