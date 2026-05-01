# ccstats — guidance for Claude Code

A Go statusline + detail TUI for Claude Code. Reads the JSON payload Claude
Code pipes to the statusline command on stdin, emits a 4–5 line summary on
stdout. Binary lives at `~/.local/bin/ccstats` after `./install.sh`.

## Layout

```
cmd/ccstats/main.go      entry; dispatches statusline / record / detail / version / help
internal/transcript/     JSONL transcript parser
internal/stats/          per-session aggregator + 5h window aggregator
internal/pricing/        hardcoded price table + cost compute
internal/render/         dim 4-line statusline + plan auto-detection
internal/tips/           markdown loader + expression evaluator + sticky picker
internal/csvlog/         atomic upsert into ~/.claude/ccstats/sessions.csv
internal/detail/         Bubble Tea full-screen TUI (`/usage`)
internal/account/        reads ~/.claude.json for plan tier
commands/usage.md        /usage slash command
tips/                    default tip knowledge base (one .md per tip)
settings.example.json    snippet to merge into ~/.claude/settings.json
install.sh               build + copy tips + copy slash command
```

## Invariants

- **Payload is authoritative.** `rate_limits.five_hour.used_percentage`,
  `rate_limits.five_hour.resets_at`, and `context_window.used_percentage`
  come straight from Claude Code's stdin payload — never recompute these
  from the transcript. They must match what `/usage` shows.
- **Transcript is the fallback** for things the payload doesn't carry
  (per-model breakdown, burn rate, turn count).
- **No stdin = exit 0.** The statusline must never block or error when
  invoked outside Claude Code; tests / probes feed empty stdin.
- **`cp -n` in install.sh is intentional** — re-running the installer
  must not overwrite user-edited tips or commands.

## Common tasks

- Build: `go build -o ccstats ./cmd/ccstats` (or `./install.sh` for full install).
- Smoke test the statusline: `echo '{}' | ./ccstats` — should print nothing
  catastrophic and exit 0.
- After rebuilding, the user's running Claude Code session picks up the new
  binary on the next statusline tick — no Claude Code restart needed.
  ("Restart the statusline" in user-speak means *confirm the new binary is
  in place*, not a literal process restart.)

## Tips system

Each tip is a markdown file with frontmatter (`id`, `trigger`, `priority`,
`cooldown_turns`). The trigger is a small expression DSL (see README for the
variable list and operators). The highest-priority matching tip whose
cooldown has expired wins; state is per-session in
`~/.claude/ccstats/state/<session-id>.json`.

When adding new trigger variables, update both `internal/tips/tips.go`
(evaluator) and `cmd/ccstats/main.go` `pickTip` (where `Vars` is populated).

## Plan limits

Defaults match the Claude-Code-Usage-Monitor project (see README). Plan
auto-bumps based on observed usage — don't hard-code a plan. Pricing lives
in `internal/pricing/pricing.go` and approximates Anthropic's public table.
