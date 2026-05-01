#!/usr/bin/env bash
# ccstats installer — builds the binary, copies tips + slash command into place,
# and prints the settings.json snippet to merge.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
CLAUDE_DIR="${CLAUDE_DIR:-$HOME/.claude}"
CCSTATS_DIR="$CLAUDE_DIR/ccstats"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

echo "==> Building ccstats…"
cd "$HERE"
go build -o ccstats ./cmd/ccstats

mkdir -p "$BIN_DIR"
mv ccstats "$BIN_DIR/ccstats"
echo "    installed to $BIN_DIR/ccstats"

echo "==> Installing default tips into $CCSTATS_DIR/tips/"
mkdir -p "$CCSTATS_DIR/tips"
cp -n "$HERE"/tips/*.md "$CCSTATS_DIR/tips/" || true

echo
echo "==> Merge this into $CLAUDE_DIR/settings.json (or settings.local.json):"
cat "$HERE/settings.example.json"
echo
echo "==> Make sure $BIN_DIR is on your PATH."
echo "    Then start a new Claude Code session — the statusline should appear."
