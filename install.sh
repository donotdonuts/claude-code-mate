#!/usr/bin/env bash
# ccmate installer — builds the binary, copies tips into place,
# and merges the statusLine + Stop hook into ~/.claude/settings.json.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
CLAUDE_DIR="${CLAUDE_DIR:-$HOME/.claude}"
CCSTATS_DIR="$CLAUDE_DIR/ccmate"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
SETTINGS="$CLAUDE_DIR/settings.json"

echo "==> Building ccmate…"
cd "$HERE"
go build -o ccmate ./cmd/ccmate

mkdir -p "$BIN_DIR"
mv ccmate "$BIN_DIR/ccmate"
echo "    installed to $BIN_DIR/ccmate"

echo "==> Installing default tips into $CCSTATS_DIR/tips/"
mkdir -p "$CCSTATS_DIR/tips"
cp "$HERE"/tips/*.md "$CCSTATS_DIR/tips/"

echo "==> Merging statusLine + Stop hook into $SETTINGS"
mkdir -p "$CLAUDE_DIR"
if [ ! -f "$SETTINGS" ]; then
    echo "{}" > "$SETTINGS"
fi

if command -v jq >/dev/null 2>&1; then
    cp "$SETTINGS" "$SETTINGS.ccmate.bak"
    tmp="$(mktemp)"
    jq '
      .statusLine = {type: "command", command: "ccmate", padding: 0}
      | .hooks = (.hooks // {})
      | .hooks.Stop = (.hooks.Stop // [])
      | (.hooks.Stop | map((.matcher // "") == "") | index(true)) as $idx
      | if $idx == null then
          .hooks.Stop += [{matcher: "", hooks: [{type: "command", command: "ccmate record"}]}]
        else
          .hooks.Stop[$idx].hooks = (.hooks.Stop[$idx].hooks // [])
          | if (.hooks.Stop[$idx].hooks | map((.command // "") == "ccmate record") | any)
            then .
            else .hooks.Stop[$idx].hooks += [{type: "command", command: "ccmate record"}]
            end
        end
    ' "$SETTINGS" > "$tmp"
    mv "$tmp" "$SETTINGS"
    echo "    merged (backup: $SETTINGS.ccmate.bak)"
else
    echo "    jq not found — please merge this snippet manually:"
    sed 's/^/    /' "$HERE/settings.example.json"
fi

echo "==> Make sure $BIN_DIR is on your PATH."
echo "    Then start a new Claude Code session — the statusline should appear."
