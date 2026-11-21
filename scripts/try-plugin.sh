#!/bin/sh
# try-plugin.sh — a real Claude Code session with this checkout's plugin and program.
#
#   scripts/try-plugin.sh [-- <claude arguments>]
#   scripts/try-plugin.sh --prepare
#
# `claude --plugin-dir plugin` alone is not enough: every hook in plugin/hooks/hooks.json
# and the MCP server in plugin/.mcp.json put ~/.local/bin first in PATH, so the installed
# program runs, not the one you just built. This script builds the program into a
# temporary folder, copies the plugin there with those paths pointing at it, sets up a
# made-up test space with a configuration of its own (AB_CONFIG_DIR) and starts Claude
# in that space's work folder. The installed plugin is not loaded alongside: Claude Code
# takes the --plugin-dir copy in its place. The real notes and configuration are never
# touched.
#
# --prepare does everything but start Claude, and prints the variables to use it by hand.
# The temporary folder is removed when Claude exits, unless AB_TRY_KEEP=1 (--prepare
# always keeps it). Linux and macOS.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPACE=try

prepare=
case "${1:-}" in
  --prepare) prepare=1; shift ;;
  --) shift ;;
  "") ;;
  *) echo "usage: scripts/try-plugin.sh [--prepare | -- <claude arguments>]" >&2; exit 2 ;;
esac

if [ -z "$prepare" ] && ! command -v claude >/dev/null 2>&1; then
  echo "claude is not on the PATH" >&2
  exit 2
fi

T="$(mktemp -d)"
cleanup() {
  if [ -n "$prepare" ] || [ "${AB_TRY_KEEP:-}" = 1 ]; then
    echo "kept: $T"
  else
    rm -rf "$T"
  fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM

BIN="$T/bin"
PLUGIN="$T/plugin"
WORK="$T/work"
MEMORY="$T/notes"
mkdir -p "$BIN" "$WORK" "$MEMORY"

echo "building augmented-brain"
(cd "$ROOT" && go build -o "$BIN/augmented-brain" ./cmd/augmented-brain)

# The copy of the plugin, with ~/.local/bin replaced by the folder of the new program.
cp -R "$ROOT/plugin" "$PLUGIN"
point() { # point <file> <text to replace>
  sed "s|$2|$BIN|g" "$1" > "$1.new"
  mv "$1.new" "$1"
  if grep -q '\.local/bin' "$1" || ! grep -q "$BIN" "$1"; then
    echo "$1: its PATH no longer looks like \"$2\"; update this script" >&2
    exit 1
  fi
}
point "$PLUGIN/hooks/hooks.json" '\$HOME/\.local/bin'
point "$PLUGIN/.mcp.json" '\${HOME:-}/\.local/bin'

# A configuration and a space of their own: hooks and the MCP server inherit them.
export AB_CONFIG_DIR="$T/config"
export AB_NO_UPDATE_CHECK=1
"$BIN/augmented-brain" init --yes --no-plugin --language en --space "$SPACE" \
  --work "$WORK" --memory "$MEMORY" > "$T/init.log" 2>&1 || { cat "$T/init.log" >&2; exit 1; }

echo "program: $BIN/augmented-brain ($("$BIN/augmented-brain" version))"
echo "plugin:  $PLUGIN"
echo "space:   $SPACE (work $WORK, notes $MEMORY)"

if [ -n "$prepare" ]; then
  echo
  echo "to start Claude with them:"
  echo "  cd $WORK && AB_CONFIG_DIR=$AB_CONFIG_DIR AB_NO_UPDATE_CHECK=1 claude --plugin-dir $PLUGIN"
  exit 0
fi

cd "$WORK"
status=0
claude --plugin-dir "$PLUGIN" "$@" || status=$?
exit "$status"
