# ccstats

A Claude Code statusline + detail viewer that surfaces real-time usage at the
bottom of your CLI:

```
Opus 4.7 · 14m · 23 turns · ctx 47% (94k/1M)
Usage 18% (Max5) · Resets 3:20pm
opus 65% · sonnet 30% · haiku 5%
221 words · +40/-12 LOC · 5M tok · 2k tok/min
tip: context filling up — consider /compact before next big task
```

The 5-hour usage % and reset time on line 2 come from Claude Code's
statusline payload (`rate_limits.five_hour.used_percentage` and
`resets_at`) — they match what `/usage` shows because they share the
same data source.

When you're past plan limit (overage active), the line just shows the
percentage above 100%:

```
Usage 130% (Max5) · Resets 3:20pm
```

## What it shows

**Statusline (always visible, 4–5 lines, dim styling that works in dark and
light terminals):**

- Line 1 — model · session duration · turn count · context window % (from
  payload `context_window.used_percentage` and `context_window_size`)
- Line 2 — 5h `used_percentage` and `resets_at` from payload, plan name
  auto-detected from `~/.claude.json` `oauthAccount.organizationRateLimitTier`
- Line 3 — per-model usage breakdown (computed from transcript)
- Line 4 — words sent · `+added/-removed` LOC from payload · cumulative
  tokens · burn rate
- Line 5 — at most one contextual tip from the knowledge base

On a fresh session before the first API response, line 2 shows
`Usage — (Max5) · Resets —` because `rate_limits` is absent until the model
has responded once.

**Detail TUI** (`/usage` or `ccstats detail`): full-screen Bubble Tea view
with token breakdown, per-model cost, 5-hour-window progress bars, and a
projected end-of-window estimate based on burn rate.

**CSV log** (`~/.claude/ccstats/sessions.csv`): one row per session, upserted
after every turn (via the `Stop` hook). Schema:

```
session_id,start,end,duration_min,model_primary,model_breakdown,
turns,words,loc_delta,total_tokens,input_tokens,output_tokens,
cache_read,cache_create,cost_usd,burn_rate_tpm
```

## Install

```bash
./install.sh
```

This builds `ccstats`, copies the default tips into `~/.claude/ccstats/tips/`,
installs the `/usage` slash command into `~/.claude/commands/`, and prints the
settings snippet to merge.

Manual install:

1. `go build -o ~/.local/bin/ccstats ./cmd/ccstats`
2. Copy `tips/*.md` to `~/.claude/ccstats/tips/`
3. Copy `commands/usage.md` to `~/.claude/commands/usage.md`
4. Merge `settings.example.json` into `~/.claude/settings.json`
5. Restart Claude Code

## How tips work

Tips live as one markdown file per tip, with YAML frontmatter:

```markdown
---
id: my-tip
trigger: ctx_pct > 75 && turns > 10
priority: 80
cooldown_turns: 5
---

Body of the tip — shown verbatim on the statusline tip line.
```

Available trigger variables:

| Variable               | Meaning                              |
| ---------------------- | ------------------------------------ |
| `ctx_pct`              | current-turn context window %        |
| `session_minutes`      | wall-clock session duration          |
| `turns`                | assistant turns this session         |
| `total_tokens`         | sum of all tokens this session       |
| `total_cost`           | $ cost this session                  |
| `overage_cost`         | $ over the 5h plan limit             |
| `burn_rate`            | tokens/min over the last ~10 minutes |
| `tool_failures_recent` | count of `is_error: true` results    |
| `model`                | substring match (e.g. `model == "opus"`) |

Operators: `>` `>=` `<` `<=` `==` `!=`, joined with `&&`.

The highest-priority tip whose trigger matches and whose cooldown has expired
wins. Cooldown is measured in turns. State is per-session.

## Plan limits

Defaults are taken from the
[Claude-Code-Usage-Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor)
project:

| Plan  | Tokens / 5h | Cost limit |
| ----- | ----------- | ---------- |
| Pro   | 19,000      | $18        |
| Max5  | 88,000      | $35        |
| Max20 | 220,000     | $140       |

The plan auto-bumps based on observed usage in the current window. Pricing
for cost computation is from a hardcoded table that approximates Anthropic's
public pricing — adjust `internal/pricing/pricing.go` if it drifts.

## Architecture

```
cmd/ccstats/main.go         entry; subcommand dispatch
internal/transcript/         JSONL parser
internal/pricing/            price table + cost compute
internal/stats/              session aggregator + 5h window aggregator
internal/render/             dim 4-line statusline + plan picker
internal/tips/               markdown loader + expression evaluator
internal/csvlog/             atomic upsert into sessions.csv
internal/detail/             Bubble Tea TUI
tips/                        default tip knowledge base
commands/usage.md            /usage slash command
settings.example.json        snippet to merge into ~/.claude/settings.json
```
