#!/bin/sh
# install-check.sh — the installer from nothing, hands-off, and a check of what it left.
#
#   scripts/install-check.sh
#
# Builds a release from this checkout and runs install.sh against it, answering yes to
# everything (AB_YES=1), in Spanish, with the plugin from this checkout. Then it checks
# what a newcomer would get: the program, the PATH line in the shell's startup file, the
# two spaces, the default space, the plugin, the first-session interview and doctor.
#
# It installs Claude Code for the user who runs it, so run it on a throwaway machine:
# GitHub Actions does (.github/workflows/install.yml). With AB_CHECK_HOME it uses that
# folder as the home instead, which keeps a developer's own setup untouched (Claude Code
# is still downloaded into it). The desktop app is skipped unless AB_CHECK_DESKTOP=1.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fails=0
ok() { echo "  OK    $1"; }
bad() { echo "  FAIL  $1"; fails=$((fails + 1)); }
check() { # check <what> <command…>
  what="$1"; shift
  if "$@" >/dev/null 2>&1; then ok "$what"; else bad "$what"; fi
}

echo "1. build a release from this checkout"
"$ROOT/scripts/release-build.sh" 0.0.0-check "$TMP/release" >/dev/null

# Only now switch to the throwaway home: the build above keeps using the real Go cache.
if [ -n "${AB_CHECK_HOME:-}" ]; then
  mkdir -p "$AB_CHECK_HOME"
  HOME="$(cd "$AB_CHECK_HOME" && pwd)"
  export HOME
  PATH="/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin"
  export PATH
fi

# A newcomer's shell does not have ~/.local/bin in its PATH yet (GitHub's runners do).
PATH="$(printf '%s' "$PATH" | awk -v RS=: -v ORS=: -v d="$HOME/.local/bin" '$0 != d && $0 != d "/"' | sed 's/:$//')"
export PATH

echo "2. run the installer, hands-off"
[ "${AB_CHECK_DESKTOP:-}" = 1 ] || AB_NO_DESKTOP=1
AB_YES=1 AB_LANGUAGE=es AB_NO_DESKTOP="${AB_NO_DESKTOP:-}" \
  AB_RELEASE_URL="file://$TMP/release" AB_MARKETPLACE="$ROOT" \
  sh "$ROOT/install.sh" </dev/null >"$TMP/install.log" 2>&1 || { cat "$TMP/install.log"; bad "install.sh exited with an error"; }
sed 's/^/    | /' "$TMP/install.log"

echo "3. what a newcomer gets"
AB="$HOME/.local/bin/augmented-brain"
check "program in ~/.local/bin" test -x "$AB"
case "$(basename "${SHELL:-sh}")" in zsh) rc="$HOME/.zshrc" ;; bash) rc="$HOME/.bashrc" ;; *) rc="$HOME/.profile" ;; esac
check "PATH line in $(basename "$rc")" grep -q '\.local/bin' "$rc"
check "Claude Code installed" test -x "$HOME/.local/bin/claude"
cfg="$HOME/.config/augmented-brain/config.toml"
check "configuration in Spanish" grep -q '^language = "es"' "$cfg"
check "default space is personal" grep -q '^default_space = "personal"' "$cfg"
check "space trabajo, notes inside" test -f "$HOME/augmented-brain/trabajo/notas/MEMORY.md"
check "space personal, notes inside" test -f "$HOME/augmented-brain/personal/notas/MEMORY.md"
check "plugin installed" sh -c "PATH=\"$HOME/.local/bin:\$PATH\" claude plugin list | grep -q 'brain@augmented-brain'"
mkdir -p "$TMP/scratch"
printf '{"cwd":"%s"}' "$TMP/scratch" | (cd "$TMP/scratch" && "$AB" hook session-start) >"$TMP/start.json"
check "outside the folders, the personal space loads" grep -q 'the default space, personal, is loaded' "$TMP/start.json"
check "and the first-session interview starts" grep -q 'First session: get to know the user' "$TMP/start.json"
check "doctor finds nothing missing" sh -c "cd '$HOME/augmented-brain/trabajo' && PATH=\"$HOME/.local/bin:\$PATH\" '$AB' doctor"
check "a second run only updates" sh -c "AB_YES=1 AB_LANGUAGE=es AB_NO_DESKTOP=1 AB_RELEASE_URL='file://$TMP/release' sh '$ROOT/install.sh' </dev/null 2>&1 | grep -q 'Ya estaba configurado'"
if [ "${AB_CHECK_DESKTOP:-}" = 1 ]; then
  check "desktop app installed" sh -c 'command -v claude-desktop || test -d /Applications/Claude.app'
fi

echo
if [ "$fails" -gt 0 ]; then
  echo "$fails FAIL"
  exit 1
fi
echo "all OK"
