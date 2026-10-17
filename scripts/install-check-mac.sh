#!/bin/sh
# install-check-mac.sh — what only a Mac can get wrong in install.sh, hands-off.
#
#   scripts/install-check-mac.sh
#
# Runs install.sh, built from this checkout, in throwaway homes, and checks:
#   - the PATH line goes where the Mac's Terminal reads it: ~/.zshrc for zsh, and for
#     bash the first of ~/.bash_profile, ~/.bash_login, ~/.profile (never ~/.bashrc);
#   - a macOS older than 13 is refused before anything is installed;
#   - without Apple's developer tools, the stub /usr/bin/git is not taken for git: the
#     installer asks for the tools; with a terminal it waits for them and carries on.
# sw_vers and xcode-select are replaced by small fakes, so nothing on the Mac changes.
# CI runs it on every Mac row of install.yml; scripts/install-check.sh does the rest.
set -u

[ "$(uname -s)" = Darwin ] || { echo "install-check-mac.sh runs on a Mac" >&2; exit 2; }
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
pass=0
fail=0
ok() { pass=$((pass + 1)); echo "  OK    $1"; }
bad() { fail=$((fail + 1)); echo "  FAIL  $1"; }
SYS="/usr/bin:/bin:/usr/sbin:/sbin"

# AB_CHECK_RELEASE: a folder with a release already built (a Mac without Go).
REL="${AB_CHECK_RELEASE:-$T/release}"
if [ -z "${AB_CHECK_RELEASE:-}" ]; then
  echo "building a release from this checkout"
  "$ROOT/scripts/release-build.sh" 0.0.0-check "$REL" >/dev/null || exit 2
fi
export AB_RELEASE_URL="file://$REL" AB_MARKETPLACE="$ROOT"

# run <home> <path> [VAR=value…]: install.sh in that home, hands-off, output in <home>.out
run() {
  h="$1"; p="$2"; shift 2
  mkdir -p "$h"
  env HOME="$h" PATH="$p" AB_YES=1 "$@" sh "$ROOT/install.sh" </dev/null >"$h.out" 2>&1
}

echo "1. the PATH line, where the Mac's Terminal reads it"
# path_case <name> <shell> <files that exist first> <file expected>
path_case() {
  h="$T/home-$1"
  mkdir -p "$h"
  for f in $3; do echo "# the user's own" >"$h/$f"; done
  run "$h" "$SYS" SHELL="$2" AB_NO_INIT=1 AB_LANGUAGE=en
  if grep -q "added by augmented brain" "$h/$4" 2>/dev/null; then ok "$1: in $4"; else bad "$1: not in $4"; cat "$h.out"; fi
  for f in .zshrc .bashrc .bash_profile .bash_login .profile; do
    [ "$f" = "$4" ] && continue
    if grep -q "added by augmented brain" "$h/$f" 2>/dev/null; then bad "$1: also in $f"; fi
  done
}
path_case zsh /bin/zsh "" .zshrc
path_case bash /bin/bash "" .bash_profile
path_case bash-with-profile /bin/bash ".profile" .profile
path_case bash-with-both /bin/bash ".bash_profile .profile" .bash_profile

echo "2. the macOS version"
mkdir -p "$T/old"
printf '#!/bin/sh\necho 12.7.4\n' >"$T/old/sw_vers"
chmod +x "$T/old/sw_vers"
if run "$T/home-old" "$T/old:$SYS" AB_LANGUAGE=en; then
  bad "macOS 12 was not refused"; cat "$T/home-old.out"
elif grep -q "macOS 13 (Ventura)" "$T/home-old.out" && [ ! -e "$T/home-old/.local/bin/augmented-brain" ]; then
  ok "macOS 12 is refused, before installing anything"
else
  bad "macOS 12: refused, but not as expected"; cat "$T/home-old.out"
fi
if run "$T/home-now" "$SYS" AB_NO_INIT=1 AB_LANGUAGE=en; then
  ok "macOS $(sw_vers -productVersion) goes ahead"
else
  bad "macOS $(sw_vers -productVersion) was refused"; cat "$T/home-now.out"
fi

echo "3. git without Apple's developer tools"
# A fake xcode-select: no tools until --install has run for a few seconds. And a fake
# claude, so the run goes on past git without downloading Claude Code.
mkdir -p "$T/notools"
printf '#!/bin/sh\nexit 0\n' >"$T/notools/claude"
chmod +x "$T/notools/claude"
cat >"$T/notools/xcode-select" <<EOF
#!/bin/sh
case "\$1" in
  -p) [ -f "$T/tools-done" ] && echo /Library/Developer/CommandLineTools && exit 0; exit 2 ;;
  --install) touch "$T/tools-asked"; (sleep 8; touch "$T/tools-done") >/dev/null 2>&1 & exit 0 ;;
esac
EOF
chmod +x "$T/notools/xcode-select"
run "$T/home-git" "$T/notools:$SYS" AB_LANGUAGE=en AB_NO_DESKTOP=1
if [ -f "$T/tools-asked" ] && grep -q "Apple's developer tools" "$T/home-git.out" && [ ! -e "$T/home-git/.local/bin/augmented-brain" ]; then
  ok "no terminal: the tools are asked for, and nothing else is done"
else
  bad "no terminal: the stub /usr/bin/git was taken for git"; cat "$T/home-git.out"
fi
# With a terminal (script(1) gives it one), it waits for the tools and carries on.
rm -f "$T/tools-asked" "$T/tools-done"
mkdir -p "$T/home-wait"
printf '#!/bin/sh\nexec env HOME="%s" PATH="%s" AB_LANGUAGE=en AB_NO_DESKTOP=1 sh "%s"\n' \
  "$T/home-wait" "$T/notools:$SYS" "$ROOT/install.sh" >"$T/wait.sh"
script -q "$T/home-wait.out" sh "$T/wait.sh" </dev/null >/dev/null 2>&1
if grep -q "I wait here" "$T/home-wait.out" && grep -q "git installed" "$T/home-wait.out" && [ -x "$T/home-wait/.local/bin/augmented-brain" ]; then
  ok "with a terminal: it waits for the tools, then installs"
else
  bad "with a terminal: it did not wait and carry on"; cat "$T/home-wait.out"
fi

echo
echo "$pass OK, $fail FAIL"
[ "$fail" -eq 0 ]
