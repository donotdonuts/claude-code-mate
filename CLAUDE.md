# ccmate — guidance for Claude Code

A Go statusline for Claude Code. Reads the JSON payload Claude Code pipes to
the statusline command on stdin, emits a 4–5 line summary on stdout. Binary
lives at `~/.local/bin/ccmate` after `./install.sh`.

## Layout

```
cmd/ccmate/main.go      entry; dispatches statusline / record / version / help
internal/transcript/     JSONL transcript parser
internal/stats/          per-session aggregator (wall-clock duration + avg burn rate)
internal/pricing/        hardcoded price table + cost compute
internal/render/         dim 4-line statusline + plan auto-detection
internal/tips/           markdown loader + expression evaluator + sticky picker
internal/csvlog/         atomic upsert into ~/.claude/ccmate/sessions.csv
internal/account/        reads ~/.claude.json for plan tier
tips/                    default tip knowledge base (one .md per tip)
settings.example.json    snippet to merge into ~/.claude/settings.json
install.sh               build + copy tips
uninstall.sh             remove binary; --purge also removes ~/.claude/ccmate/
```

## Invariants

- **Payload is authoritative.** `rate_limits.five_hour.used_percentage`,
  `rate_limits.five_hour.resets_at`, and `context_window.used_percentage`
  come straight from Claude Code's stdin payload — never recompute these
  from the transcript.
- **Transcript is the fallback** for things the payload doesn't carry
  (per-model breakdown, burn rate, turn count).
- **No stdin = exit 0.** The statusline must never block or error when
  invoked outside Claude Code; tests / probes feed empty stdin.
- **`cp -n` in install.sh is intentional** — re-running the installer
  must not overwrite user-edited tips or commands.

## Common tasks

- Build: `go build -o ccmate ./cmd/ccmate` (or `./install.sh` for full install).
- Smoke test the statusline: `echo '{}' | ./ccmate` — should print nothing
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
`~/.claude/ccmate/state/<session-id>.json`.

When adding new trigger variables, update both `internal/tips/tips.go`
(evaluator) and `cmd/ccmate/main.go` `pickTip` (where `Vars` is populated).

## Plan limits

Defaults match the Claude-Code-Usage-Monitor project (see README). Plan
auto-bumps based on observed usage — don't hard-code a plan. Pricing lives
in `internal/pricing/pricing.go` and approximates Anthropic's public table.
