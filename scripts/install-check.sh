#!/bin/sh
# install-check.sh — the installer from nothing, hands-off, and a check of what it left.
#
#   scripts/install-check.sh
#
# Builds a release from this checkout and runs install.sh against it, answering yes to
# everything (AB_YES=1), in Spanish, with the plugin from this checkout. Then it checks
# what a newcomer would get: the program, the PATH line in the shell's startup file, the
# two spaces, the default space, the plugin, the first-session interview and doctor.
# Then the rest of its life: uninstall and install again (notes and configuration
# stay), uninstall --config and install again (the notes come back), a home with spaces
# and accents in its path, uninstall --all (nothing left), and the update people make,
# from the release published last on GitHub to this checkout's (AB_CHECK_NO_NETWORK=1
# skips that one).
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

# A desktop download that stalls must give up with its message and let the install go
# on (on 11-nov-2026 Anthropic's apt repository stalled and the installer waited until CI
# stopped it). A local server gives the key at once and the repository a byte every two
# seconds. Linux with apt and sudo with no password only (CI), before the real app is
# installed; in a home of its own, with a pretend claude, so step 2 still starts from
# nothing.
if [ "${AB_CHECK_DESKTOP:-}" = 1 ] && [ "$(uname -s)" = Linux ] && command -v apt-get >/dev/null 2>&1 &&
  sudo -n true 2>/dev/null && command -v python3 >/dev/null 2>&1; then
  echo "1b. a desktop download that stalls gives up"
  cat >"$TMP/slow.py" <<'PY'
import http.server, sys, time
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Length", "100000")
        self.end_headers()
        if self.path.endswith("/key.asc"):
            self.wfile.write(b"k" * 100000)
            return
        try:
            for _ in range(100000):
                self.wfile.write(b"x"); self.wfile.flush(); time.sleep(2)
        except OSError:
            pass
    def log_message(self, *a):
        pass
s = http.server.ThreadingHTTPServer(("127.0.0.1", 0), H)
s.daemon_threads = True
print(s.server_address[1], flush=True)
s.serve_forever()
PY
  python3 -I "$TMP/slow.py" >"$TMP/slow.port" &
  slow=$!
  for _ in 1 2 3 4 5 6 7 8 9 10; do [ -s "$TMP/slow.port" ] && break; sleep 1; done
  mkdir -p "$TMP/slowhome" "$TMP/slowclaude"
  printf '#!/bin/sh\n[ "$1" = --version ] && echo "9.9.9 (Claude Code)"\nexit 0\n' >"$TMP/slowclaude/claude"
  chmod +x "$TMP/slowclaude/claude"
  echo 9.9.9 >"$TMP/slow.stable"
  start=$(date +%s)
  # XDG_CONFIG_HOME too: GitHub's runners set it, and the configuration would land in
  # the real home, so step 2 would find it "already set up"
  HOME="$TMP/slowhome" XDG_CONFIG_HOME="$TMP/slowhome/.config" PATH="$TMP/slowclaude:$PATH" AB_YES=1 AB_LANGUAGE=es \
    AB_DESKTOP_URL="http://127.0.0.1:$(cat "$TMP/slow.port")" AB_DESKTOP_TIMEOUT=30 \
    AB_CLAUDE_STABLE_URL="file://$TMP/slow.stable" AB_RELEASE_URL="file://$TMP/release" AB_MARKETPLACE="$ROOT" \
    sh "$ROOT/install.sh" </dev/null >"$TMP/slow.log" 2>&1 || true
  took=$(($(date +%s) - start))
  kill "$slow" 2>/dev/null || true
  sed 's/^/    | /' "$TMP/slow.log"
  check "it gives up in time ($took s)" test "$took" -lt 120
  check "and says so" grep -q 'la he dejado' "$TMP/slow.log"
  check "and the install goes on" test -x "$TMP/slowhome/.local/bin/augmented-brain"
  # what the real install below writes again: the repository's line points here now
  sudo rm -f /etc/apt/sources.list.d/claude-desktop.list
fi

echo "2. run the installer, hands-off"
[ "${AB_CHECK_DESKTOP:-}" = 1 ] || AB_NO_DESKTOP=1
AB_YES=1 AB_LANGUAGE=es AB_NO_DESKTOP="${AB_NO_DESKTOP:-}" \
  AB_RELEASE_URL="file://$TMP/release" AB_MARKETPLACE="$ROOT" \
  sh "$ROOT/install.sh" </dev/null >"$TMP/install.log" 2>&1 &
installer=$!
# The installer's output as it comes, not only at the end: when CI stops a run that
# hangs (timeout-minutes), its log must say on which step.
# GNU tail follows until the installer ends; macOS's tail has no --pid, so there the log
# is shown once it is done.
if tail --pid=$$ -n 0 /dev/null >/dev/null 2>&1; then
  tail -n +1 -f --pid="$installer" "$TMP/install.log" | sed -u 's/^/    | /'
else
  while kill -0 "$installer" 2>/dev/null; do sleep 2; done
  sed 's/^/    | /' "$TMP/install.log"
fi
wait "$installer" || bad "install.sh exited with an error"

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

notes() { find "$HOME/augmented-brain" -name '*.md' 2>/dev/null | wc -l | tr -d ' '; }
reinstall() { # the installer again, from this checkout's release
  AB_YES=1 AB_LANGUAGE=es AB_NO_DESKTOP=1 AB_RELEASE_URL="file://$TMP/release" AB_MARKETPLACE="$ROOT" \
    sh "$ROOT/install.sh" </dev/null >"$TMP/reinstall.log" 2>&1
}

echo "4. uninstall, then install again: the notes and the configuration stay"
before="$(notes)"
PATH="$HOME/.local/bin:$PATH" "$AB" uninstall --yes >"$TMP/uninstall.log" 2>&1 || bad "uninstall exited with an error"
check "the program is gone" sh -c "! test -e '$AB'"
check "the plugin is gone" sh -c "! PATH=\"$HOME/.local/bin:\$PATH\" claude plugin list 2>/dev/null | grep -q 'brain@augmented-brain'"
check "the configuration stays" test -f "$cfg"
check "every note stays" test "$(notes)" = "$before"
reinstall || bad "install.sh exited with an error"
check "installing again finds them" grep -q 'Ya estaba configurado' "$TMP/reinstall.log"
check "the program is back" test -x "$AB"
check "the plugin is back" sh -c "PATH=\"$HOME/.local/bin:\$PATH\" claude plugin list | grep -q 'brain@augmented-brain'"
check "doctor is happy again" sh -c "cd '$HOME/augmented-brain/trabajo' && PATH=\"$HOME/.local/bin:\$PATH\" '$AB' doctor"

echo "5. uninstall --config, then install again: the notes stay and come back"
PATH="$HOME/.local/bin:$PATH" "$AB" uninstall --yes --config >"$TMP/uninstall.log" 2>&1 || bad "uninstall --config exited with an error"
check "the configuration is gone" sh -c "! test -e '$cfg'"
check "every note stays" test "$(notes)" = "$before"
reinstall || bad "install.sh exited with an error"
check "the spaces are made again, over the notes" test -f "$cfg"
check "no note lost or doubled" test "$(notes)" = "$before"
check "doctor is happy again" sh -c "cd '$HOME/augmented-brain/trabajo' && PATH=\"$HOME/.local/bin:\$PATH\" '$AB' doctor"

echo "6. a home with spaces and accents in its path"
odd="$TMP/casa de Lucía"
mkdir -p "$odd"
check "init there" sh -c "HOME='$odd' AB_CONFIG_DIR='$odd/cfg' AB_LANGUAGE=es '$AB' init --no-plugin"
check "check there" sh -c "cd '$odd/augmented-brain/trabajo' && HOME='$odd' AB_CONFIG_DIR='$odd/cfg' '$AB' check"
check "a backup there" sh -c "git init -q --bare -b main '$odd/copia.git' && cd '$odd/augmented-brain/trabajo' && HOME='$odd' AB_CONFIG_DIR='$odd/cfg' '$AB' backup setup '$odd/copia.git' && HOME='$odd' AB_CONFIG_DIR='$odd/cfg' '$AB' backup"

echo "7. uninstall --all: starting from scratch leaves nothing behind"
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
  -iname '*augmented*' 2>/dev/null >"$TMP/left.txt" || true  # some folders do not exist
check "and nothing else is left" sh -c "! test -s '$TMP/left.txt'"
sed 's/^/    left: /' "$TMP/left.txt"

# The update people make: from the release published last, downloaded from GitHub, to
# this checkout's. AB_CHECK_NO_NETWORK=1 skips it.
if [ "${AB_CHECK_NO_NETWORK:-}" != 1 ]; then
  echo "8. update from the release published last"
  last="$(curl -fsSIL -o /dev/null -w '%{url_effective}' "https://github.com/icalvete/augmented-brain/releases/latest" | sed 's#.*/tag/v##')"
  if [ -n "$last" ]; then
    AB_YES=1 AB_LANGUAGE=es AB_NO_DESKTOP=1 AB_VERSION="$last" AB_MARKETPLACE="$ROOT" \
      sh "$ROOT/install.sh" </dev/null >"$TMP/old.log" 2>&1 || { sed 's/^/    | /' "$TMP/old.log"; bad "installing $last exited with an error"; }
    check "release $last installed" sh -c "'$AB' version | grep -qx '$last'"
    before="$(notes)"
    reinstall || bad "the update exited with an error"
    check "updated to this checkout" sh -c "'$AB' version | grep -qx '0.0.0-check'"
    check "every note stays" test "$(notes)" = "$before"
    check "the notes still check" sh -c "cd '$HOME/augmented-brain/trabajo' && '$AB' check"
    check "doctor is happy after the update" sh -c "cd '$HOME/augmented-brain/trabajo' && PATH=\"$HOME/.local/bin:\$PATH\" '$AB' doctor"
  else
    bad "cannot tell the release published last"
  fi
fi

echo
if [ "$fails" -gt 0 ]; then
  echo "$fails FAIL"
  exit 1
fi
echo "all OK"
