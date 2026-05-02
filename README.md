# ccmate

A Claude Code statusline that surfaces real-time usage at the bottom of your
CLI:

![ccmate statusline](Group%2011.jpg)

The 5-hour and 7-day usage bars come from Claude Code's statusline payload
(`rate_limits.five_hour` and `rate_limits.seven_day`) — same data source as
Claude Code's built-in usage view. Context % comes from
`context_window.used_percentage`.

When you're past plan limit (overage active), the percentage just keeps
counting past 100%:

```
... Usage 130% ████████ (Max5) · Resets 3:20pm
```

## What it shows

**Statusline (always visible, dim styling that works in dark and light
terminals; lines are skipped when the underlying data isn't available):**

- Line 1 — model · context bar (`used/size`) · 5-hour usage bar (plan name) ·
  reset time. Plan name is auto-detected from `~/.claude.json`
  (`oauthAccount.organizationRateLimitTier`).
- Line 2 — 7-day usage bar · weekly reset · per-model token share over the
  last 7 days (computed from `sessions.csv` plus the live session).
- Line 3 — cache hit % · cache read tokens · cache create tokens.
- Line 4 — wall-clock duration · turns · words sent · `+added/-removed` LOC
  from the payload · cumulative tokens · burn rate (tokens/min).
- Line 5 — per-model breakdown for the current session (only shown when more
  than one model has been used).
- Tip line — at most one contextual tip from the knowledge base.

On a fresh session before the first API response, the usage segments show
`—` because `rate_limits` is absent until the model has responded once.

**CSV log** (`~/.claude/ccmate/sessions.csv`): one row per session, upserted
after every turn (via the `Stop` hook). Schema:

```
session_id,start,end,duration_min,model_primary,model_breakdown,
turns,words,loc_delta,total_tokens,input_tokens,output_tokens,
cache_read,cache_create,cost_usd,burn_rate_tpm
```

`model_breakdown` is a JSON object keyed by full model id
(e.g. `{"claude-opus-4-7": 4823100}`); the weekly row aggregates from this.

## Install

Requires the Go toolchain (`go` on PATH). Then:

```bash
# macOS / Linux
./install.sh
```

```powershell
# Windows (PowerShell 7+)
pwsh -File .\install.ps1
```

The installer:

1. Builds `ccmate` into `~/.local/bin/` (or `%USERPROFILE%\.local\bin\` on
   Windows).
2. Copies the default tips into `~/.claude/ccmate/tips/` without
   overwriting any tips you've already edited.
3. Merges the `statusLine` and `Stop` hook entries into
   `~/.claude/settings.json` in place. A backup is written to
   `settings.json.ccmate.bak`. Re-running the installer is safe; it won't
   duplicate hook entries.

The bash installer needs `jq` for the merge step; if `jq` isn't installed it
prints the snippet for you to paste manually. The PowerShell installer
requires PowerShell 7+ (`winget install Microsoft.PowerShell`) for the merge
step, and falls back to printing the snippet on Windows PowerShell 5.

Make sure `~/.local/bin` is on `PATH`, then start a new Claude Code session.

Manual install:

1. Build the binary into a directory on PATH:
   - macOS / Linux: `mkdir -p ~/.local/bin && go build -o ~/.local/bin/ccmate ./cmd/ccmate`
   - Windows:       `go build -o "$env:USERPROFILE\.local\bin\ccmate.exe" ./cmd/ccmate`
2. Copy `tips/*.md` to `~/.claude/ccmate/tips/`.
3. Merge `settings.example.json` into `~/.claude/settings.json`.
4. Restart Claude Code.

## Update

To pick up a new version, pull the repo and re-run the installer:

```bash
# macOS / Linux
git pull && ./install.sh
```

```powershell
# Windows
git pull; pwsh -File .\install.ps1
```

Re-running the installer rebuilds the binary, refreshes the default tips
in `~/.claude/ccmate/tips/`, and re-merges `~/.claude/settings.json`
idempotently (existing entries aren't duplicated). Your `sessions.csv`
history is untouched.

## Uninstall

```bash
# macOS / Linux
./uninstall.sh           # remove the binary + strip ccmate from settings.json
./uninstall.sh --purge   # also delete ~/.claude/ccmate/ (history + state)
./uninstall.sh --purge --yes   # skip the confirmation prompt
```

```powershell
# Windows
pwsh -File .\uninstall.ps1
pwsh -File .\uninstall.ps1 -Purge
pwsh -File .\uninstall.ps1 -Purge -Yes
```

The uninstaller strips the `ccmate` `statusLine` and the `ccmate record`
`Stop` hook from `~/.claude/settings.json` (with a backup at
`settings.json.ccmate.bak`). Other entries you've added are left alone.

`--purge` deletes `~/.claude/ccmate/` — including `sessions.csv` (your full
session history), any tips you've edited, and per-session state. When stdin
isn't a terminal (e.g. piping from another tool), pass `--yes` / `-Yes` to
skip the confirmation prompt.

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
| `overage_cost`         | percentage points over the 5h plan limit |
| `burn_rate`            | tokens/min over the session          |
| `tool_failures_recent` | count of `is_error: true` results    |
| `model`                | substring match (e.g. `model == "opus"`) |

Operators: `>` `>=` `<` `<=` `==` `!=`, joined with `&&`.

The highest-priority tip whose trigger matches and whose cooldown has expired
wins. Cooldown is measured in turns. State is per-session under
`~/.claude/ccmate/state/<session-id>.json`.

## Plan limits

Defaults are taken from the
[Claude-Code-Usage-Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor)
project:

| Plan  | Tokens / 5h | Cost limit |
| ----- | ----------- | ---------- |
| Pro   | 19,000      | $18        |
| Max5  | 88,000      | $35        |
| Max20 | 220,000     | $140       |

The statusline itself doesn't consult these — the 5-hour and 7-day bars come
straight from the payload `rate_limits.*.used_percentage`. The table is used
for cost-estimation context only. Pricing for cost computation is from a
hardcoded table that approximates Anthropic's public pricing — adjust
`internal/pricing/pricing.go` if it drifts.

## Architecture

```
cmd/ccmate/main.go         entry; dispatches statusline / record / version / help
internal/transcript/         JSONL parser
internal/pricing/            price table + cost compute
internal/stats/              per-session aggregator (duration, burn rate, model breakdown)
internal/render/             dim multi-line statusline with colored bars
internal/tips/               markdown loader + expression evaluator + sticky picker
internal/csvlog/             atomic upsert into sessions.csv + 7-day weekly aggregator
internal/account/            reads ~/.claude.json for plan tier
tips/                        default tip knowledge base
settings.example.json        snippet merged by install scripts
install.sh / install.ps1     build + copy tips + merge settings.json
uninstall.sh / uninstall.ps1 remove binary + strip settings.json (--purge wipes data dir)
```
