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

# AB_CHECK_NO_BREW=1: a Mac without Homebrew, the usual one; the installer must fetch
# the desktop app by itself.
if [ "${AB_CHECK_NO_BREW:-}" = 1 ]; then
  PATH="$(printf '%s' "$PATH" | awk -v RS=: -v ORS=: '$0 !~ /^(\/opt\/homebrew|\/usr\/local)\/s?bin\/?$/' | sed 's/:$//')"
  export PATH
  if command -v brew >/dev/null 2>&1; then bad "Homebrew is still in the PATH"; fi
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
# An old Claude Code is offered an update; a current one is left alone. A fake claude,
# first in the PATH, says its version and records when it is asked to update.
mkdir -p "$TMP/fakeclaude"
cat >"$TMP/fakeclaude/claude" <<EOF
#!/bin/sh
case "\$1" in --version) echo "2.1.100 (Claude Code)" ;; update) touch "$TMP/claude-updated" ;; esac
exit 0
EOF
chmod +x "$TMP/fakeclaude/claude"
rerun() { # rerun <stable version>: the installer again, with the fake claude
  echo "$1" >"$TMP/stable"
  AB_YES=1 AB_LANGUAGE=es AB_NO_DESKTOP=1 AB_RELEASE_URL="file://$TMP/release" \
    AB_CLAUDE_STABLE_URL="file://$TMP/stable" PATH="$TMP/fakeclaude:$PATH" \
    sh "$ROOT/install.sh" </dev/null >"$TMP/rerun.log" 2>&1
}
rerun 2.1.292
check "an old Claude Code is offered an update" grep -q 'versión 2.1.100 y la actual es la 2.1.292' "$TMP/rerun.log"
check "and updated with its own updater" test -f "$TMP/claude-updated"
rm -f "$TMP/claude-updated"
rerun 2.1.50
check "a current Claude Code is left alone" sh -c "! test -f '$TMP/claude-updated' && ! grep -q 'la actual es' '$TMP/rerun.log'"
if [ "${AB_CHECK_DESKTOP:-}" = 1 ]; then
  check "desktop app installed" sh -c 'command -v claude-desktop || test -d /Applications/Claude.app || test -d "$HOME/Applications/Claude.app"'
  if [ "$(uname -s)" = Darwin ]; then
    check "and its signature is intact" sh -c 'codesign --verify --deep --strict /Applications/Claude.app 2>/dev/null || codesign --verify --deep --strict "$HOME/Applications/Claude.app"'
  fi
fi

echo "4. uninstall --all: starting from scratch leaves nothing behind"
PATH="$HOME/.local/bin:$PATH" "$AB" uninstall --yes --all >"$TMP/uninstall.log" 2>&1 || { cat "$TMP/uninstall.log"; bad "uninstall --all exited with an error"; }
check "the program is gone" sh -c "! test -e '$AB'"
check "the configuration is gone" sh -c "! test -e '$HOME/.config/augmented-brain'"
check "the notes are gone" sh -c "! test -e '$HOME/augmented-brain'"
check "the plugin is gone" sh -c "! PATH=\"$HOME/.local/bin:\$PATH\" claude plugin list 2>/dev/null | grep -q 'brain@augmented-brain'"
# Nothing named after it is left where augmented brain and Claude Code write: the
# program's folder, the configuration, the notes, Claude Code's folder and its cache
# (~/.cache on Linux, ~/Library/Caches on macOS). A CI runner's home has other things
# with the name, like the checkout and Go's module cache: those do not count.
find "$HOME/.local/bin" "$HOME/.config" "$HOME/augmented-brain" "$HOME/.claude" \
  "$HOME/.cache/claude-cli-nodejs" "$HOME/Library/Caches/claude-cli-nodejs" \
  -iname '*augmented*' 2>/dev/null >"$TMP/left.txt"
check "and nothing else is left" sh -c "! test -s '$TMP/left.txt'"
sed 's/^/    left: /' "$TMP/left.txt"

echo
if [ "$fails" -gt 0 ]; then
  echo "$fails FAIL"
  exit 1
fi
echo "all OK"
