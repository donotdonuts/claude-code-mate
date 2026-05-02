#!/usr/bin/env bash
# ccmate uninstaller — removes the binary, strips ccmate entries from
# ~/.claude/settings.json, and optionally purges the data dir.
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
            sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'
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
SETTINGS="$CLAUDE_DIR/settings.json"

ask_confirm() {
    local prompt="$1" reply=""
    if (: < /dev/tty) 2>/dev/null && (: > /dev/tty) 2>/dev/null; then
        printf "%s" "$prompt" > /dev/tty
        read -r reply < /dev/tty || reply=""
    elif [ -t 0 ]; then
        printf "%s" "$prompt"
        read -r reply || reply=""
    else
        return 2
    fi
    case "$reply" in y|Y|yes|YES) return 0 ;; *) return 1 ;; esac
}

echo "==> Removing binary"
if [ -e "$BIN_PATH" ]; then
    rm -f "$BIN_PATH"
    echo "    removed $BIN_PATH"
else
    echo "    not found at $BIN_PATH (skipping)"
fi

echo "==> Stripping ccmate entries from $SETTINGS"
if [ ! -f "$SETTINGS" ]; then
    echo "    $SETTINGS not found (skipping)"
elif command -v jq >/dev/null 2>&1; then
    cp "$SETTINGS" "$SETTINGS.ccmate.bak"
    tmp="$(mktemp)"
    jq '
      (if (.statusLine.command // "") == "ccmate" then del(.statusLine) else . end)
      | if .hooks.Stop then
          .hooks.Stop |= map(
            .hooks = ((.hooks // []) | map(select((.command // "") != "ccmate record")))
          )
          | .hooks.Stop |= map(select(((.hooks // []) | length) > 0))
        else . end
    ' "$SETTINGS" > "$tmp"
    mv "$tmp" "$SETTINGS"
    echo "    stripped (backup: $SETTINGS.ccmate.bak)"
else
    echo "    jq not found — please remove these entries manually:"
    echo "      - the entire \"statusLine\" block, if its \"command\" is \"ccmate\""
    echo "      - the Stop hook entry whose \"command\" is \"ccmate record\""
fi

if [ "$PURGE" -eq 1 ]; then
    if [ -d "$CCSTATS_DIR" ]; then
        if [ "$ASSUME_YES" -ne 1 ]; then
            echo
            echo "==> --purge will delete $CCSTATS_DIR"
            echo "    This includes sessions.csv (your full session history),"
            echo "    any tips you may have edited, and per-session state."
            if ask_confirm "    Continue? [y/N] "; then
                :
            else
                rc=$?
                if [ "$rc" -eq 2 ]; then
                    echo "    no terminal available — re-run with --yes to confirm"
                else
                    echo "    aborted — data dir kept"
                fi
                PURGE=0
            fi
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
echo "==> Done. Restart Claude Code to drop the statusline."
