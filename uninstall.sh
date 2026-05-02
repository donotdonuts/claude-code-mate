#!/usr/bin/env bash
# ccmate uninstaller — removes the binary, optionally purges the data dir,
# and prints the settings.json entries to remove by hand.
#
# Flags:
#   --purge   also delete ~/.claude/ccmate/ (tips, sessions.csv, state)
#   --yes     skip the confirmation prompt for --purge
#   --help    print this help

set -euo pipefail

PURGE=0
ASSUME_YES=0
for arg in "$@"; do
    case "$arg" in
        --purge) PURGE=1 ;;
        --yes|-y) ASSUME_YES=1 ;;
        --help|-h)
            sed -n '2,8p' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *)
            echo "unknown flag: $arg" >&2
            echo "try: $0 --help" >&2
            exit 2
            ;;
    esac
done

CLAUDE_DIR="${CLAUDE_DIR:-$HOME/.claude}"
CCSTATS_DIR="$CLAUDE_DIR/ccmate"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
BIN_PATH="$BIN_DIR/ccmate"

echo "==> Removing binary"
if [ -e "$BIN_PATH" ]; then
    rm -f "$BIN_PATH"
    echo "    removed $BIN_PATH"
else
    echo "    not found at $BIN_PATH (skipping)"
fi

if [ "$PURGE" -eq 1 ]; then
    if [ -d "$CCSTATS_DIR" ]; then
        if [ "$ASSUME_YES" -ne 1 ]; then
            echo
            echo "==> --purge will delete $CCSTATS_DIR"
            echo "    This includes sessions.csv (your full session history),"
            echo "    any tips you may have edited, and per-session state."
            printf "    Continue? [y/N] "
            read -r reply </dev/tty || reply=""
            case "$reply" in
                y|Y|yes|YES) ;;
                *) echo "    aborted — data dir kept"; PURGE=0 ;;
            esac
        fi
        if [ "$PURGE" -eq 1 ]; then
            rm -rf "$CCSTATS_DIR"
            echo "    removed $CCSTATS_DIR"
        fi
    else
        echo "==> --purge: $CCSTATS_DIR not found (skipping)"
    fi
else
    if [ -d "$CCSTATS_DIR" ]; then
        echo
        echo "==> Keeping $CCSTATS_DIR (tips, sessions.csv, state)"
        echo "    Pass --purge to delete it."
    fi
fi

echo
echo "==> Remove these entries from $CLAUDE_DIR/settings.json (or settings.local.json):"
cat <<'EOF'
    - the entire "statusLine" block, if its "command" is "ccmate"
    - the Stop hook entry whose "command" is "ccmate record"
      (leave the rest of "hooks.Stop" intact)
EOF
echo
echo "==> Done. Restart Claude Code to drop the statusline."
