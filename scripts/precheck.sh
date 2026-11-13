#!/bin/sh
# precheck.sh — what CI would say, before you push: so nothing reaches CI untried.
#
#   scripts/precheck.sh          # the quick part (about a minute): run by the pre-push hook
#   scripts/precheck.sh --full   # also compat-check against the last two releases
#   scripts/precheck.sh --windows  # also the tests run on Windows, in the VM (3 minutes)
#
# The quick part: format; vet compiled for Linux, Windows and macOS; the tests compiled
# for Windows and macOS too (a test that does not build there fails CI); the tests with
# -race; the guard's test coverage, not below its floor (Linux); the privacy check of
# what is committed; the generated guides up to date; and the branch's changelog entry
# in changelog.d/ (scripts/changelog.py).
# Windows and macOS are only compiled here, not run: CI runs them. --windows runs the
# tests on Windows too (scripts/test-windows.sh, in the VM of test/vm/windows-11): for
# a change to paths, files, permissions, hooks or commands, the kind that passed on
# Linux and failed on Windows in CI. The flags can go together.
set -eu
cd "$(git rev-parse --show-toplevel)"
# Run from a git hook, git hands down GIT_DIR, GIT_INDEX_FILE and the like: the tests
# that make repositories of their own would then work on this one (they committed to
# the branch the first time). Without them, every git below finds its repository by
# its folder, as from a terminal.
for v in $(git rev-parse --local-env-vars); do unset "$v"; done
full=0
windows=0
for a in "$@"; do
  case "$a" in
    --full) full=1 ;;
    --windows) windows=1 ;;
    *) echo "precheck: unknown option $a (--full, --windows)"; exit 2 ;;
  esac
done

fail=0
step() { printf '%s… ' "$1"; }
ok() { echo "ok"; }
bad() { echo "MAL"; [ -n "${1:-}" ] && printf '%s\n' "$1" | sed 's/^/  /'; fail=1; }

step "format"
out=$(gofmt -l .); [ -z "$out" ] && ok || bad "$out"

for os in linux windows darwin; do
  step "vet ($os)"
  if out=$(GOOS=$os go vet ./... 2>&1); then ok; else bad "$out"; fi
  if [ "$os" != linux ]; then
    step "tests build ($os)"
    if out=$(GOOS=$os go test -count=1 -run '^$' -exec /bin/true ./... 2>&1); then ok; else bad "$out"; fi
  fi
done

step "tests (-race)"
if out=$(go test -race -count=1 ./... 2>&1); then ok; else bad "$(printf '%s\n' "$out" | grep -v '^ok\|no test files' | tail -40)"; fi

if [ "$(uname)" = Linux ]; then
  step "guard coverage"
  if out=$(scripts/guard-coverage.sh 2>&1); then ok; else bad "$out"; fi
fi

step "privacy"
if out=$(scripts/privacy-check.sh --all 2>&1); then ok; else bad "$out"; fi

# What regenerating changes, not what you changed by hand and have not committed yet.
step "guides up to date"
docs_sum() { find docs -type f | LC_ALL=C sort | xargs cat | cksum; }
before=$(docs_sum)
python3 -I scripts/build-guides.py >/dev/null
if [ "$before" = "$(docs_sum)" ]; then ok; else bad "scripts/build-guides.py changed the generated pages: commit them"; fi

# Each pull request leaves its changelog entry in changelog.d/ (scripts/changelog.py).
step "changelog entries"
base=origin/main
git rev-parse -q --verify "$base" >/dev/null || base=
if out=$(python3 -I scripts/changelog.py check ${base:+--base "$base"} 2>&1); then ok; else bad "$out"; fi

step "third-party notices"
if out=$(python3 -I scripts/third-party-notices.py --check 2>&1); then ok; else bad "$out"; fi

if [ "$full" = 1 ]; then
  step "compat-check (last two releases)"
  dir=$(mktemp -d)
  go build -o "$dir/ab" ./cmd/augmented-brain
  if out=$(COMPAT_DIR="$dir/run" scripts/compat-check.sh "$dir/ab" 2>&1); then ok; else bad "$(printf '%s\n' "$out" | grep -E 'FAIL|compat-check:')"; fi
  echo "  (compat-check left its files in $dir)"
fi

if [ "$windows" = 1 ]; then
  step "tests on Windows (VM)"
  if out=$(scripts/test-windows.sh 2>&1); then ok; else bad "$(printf '%s\n' "$out" | grep -v '^ok ')"; fi
fi

if [ "$fail" = 1 ]; then
  echo "precheck: something to fix before pushing."
  exit 1
fi
echo "precheck: all good."
