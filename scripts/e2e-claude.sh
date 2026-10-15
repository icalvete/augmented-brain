#!/bin/sh
# e2e-claude.sh — end-to-end scenarios with a real Claude Code, isolated and hands-off.
#
#   scripts/e2e-claude.sh
#
# Builds the program into a temporary folder, sets up a test space in a fake home and
# runs `claude -p` against the plugin in this repository. Each step checks files or
# output and prints OK or FAIL; everything temporary is removed at the end.
#
# Needs a Claude Code with its own configuration folder, logged in once by hand:
#   CLAUDE_CONFIG_DIR=~/.config/augmented-brain-dev/claude claude    (log in, then /exit)
# Override it with AB_E2E_CLAUDE_CONFIG; pick a model with AB_E2E_MODEL. Set AB_E2E_KEEP=1
# to keep the temporary folder for a look after a failure.
#
# It spends usage of the account behind that login: run it before a release, not on
# every change. The real home, the real configuration and the real Claude Code are
# never touched.
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLAUDE_DIR="${AB_E2E_CLAUDE_CONFIG:-$HOME/.config/augmented-brain-dev/claude}"
SPACE=lab
CODEWORD=BLUEHERON
NEXT_WORD=benchmark

if ! command -v claude >/dev/null 2>&1; then
  echo "claude is not on the PATH" >&2
  exit 2
fi
if [ ! -d "$CLAUDE_DIR" ]; then
  echo "no Claude configuration at $CLAUDE_DIR: log in there first (see the top of this script)" >&2
  exit 2
fi
CLAUDE_BIN="$(command -v claude)"

T="$(mktemp -d)"
cleanup() {
  if [ "${AB_E2E_KEEP:-}" = 1 ]; then
    echo "kept: $T"
  else
    rm -rf "$T"
  fi
}
trap cleanup EXIT INT TERM

BIN="$T/bin"
FAKE_HOME="$T/home"
WORK="$FAKE_HOME/work"
MEMORY="$FAKE_HOME/brain"
OUT="$T/out"
mkdir -p "$BIN" "$WORK" "$MEMORY" "$OUT" "$T/prompts"

# Build with the real environment (Go's caches live in the real home).
echo "building augmented-brain"
(cd "$ROOT" && go build -o "$BIN/augmented-brain" ./cmd/augmented-brain) || exit 2

# A fake aws: it never reaches AWS, it only records which profile it was run with.
cat > "$BIN/aws" <<EOF
#!/bin/sh
while [ \$# -gt 0 ]; do
  case "\$1" in --profile) echo "\$2" >> "$T/aws-ran"; shift ;; esac
  shift
done
echo "fake aws: nothing done"
EOF
chmod +x "$BIN/aws"
ln -s "$CLAUDE_BIN" "$BIN/claude"

# From here on, everything runs in the fake home.
export HOME="$FAKE_HOME"
export AB_CONFIG_DIR="$FAKE_HOME/.config/augmented-brain"
export CLAUDE_CONFIG_DIR="$CLAUDE_DIR"
export PATH="$BIN:/usr/bin:/bin"
unset XDG_CONFIG_HOME

pass=0
fail=0
ok()   { pass=$((pass + 1)); echo "  OK    $1"; }
bad()  { fail=$((fail + 1)); echo "  FAIL  $1"; }
check() { if eval "$2"; then ok "$1"; else bad "$1"; fi; }

# ask <step> [extra tools]: runs `claude -p` in the work folder, with the prompt in
# $T/prompts/<step> (prompts go through stdin so no command line carries them).
ask() {
  step="$1"
  tools="mcp__plugin_brain_augmented-brain${2:+,$2}"
  set -- -p --output-format text --plugin-dir "$ROOT/plugin" --allowedTools "$tools"
  if [ -n "${AB_E2E_MODEL:-}" ]; then set -- "$@" --model "$AB_E2E_MODEL"; fi
  (cd "$WORK" && claude "$@" < "$T/prompts/$step" > "$OUT/$step" 2>&1)
}

prompt() { cat > "$T/prompts/$1"; }

task_file() { ls "$MEMORY"/tasks/*.md 2>/dev/null | head -n 1; }

echo "setting up the space"
augmented-brain init --yes --no-plugin --language en --space "$SPACE" \
  --work "$WORK" --memory "$MEMORY" > "$OUT/init" 2>&1 || { cat "$OUT/init"; exit 2; }
CONFIG="$AB_CONFIG_DIR/config.toml"
grep -q "^\[spaces\.$SPACE\]" "$CONFIG" || { echo "no [spaces.$SPACE] in $CONFIG" >&2; exit 2; }
# Mark a production AWS profile (the section is the last one, so appending works).
echo 'production_aws_profiles = ["prod"]' >> "$CONFIG"
augmented-brain doctor > "$OUT/doctor" 2>&1 || { cat "$OUT/doctor"; exit 2; }

# A git repository with a remote, for the guard scenario.
REMOTE="$T/remote.git"
git init -q --bare "$REMOTE"
git -C "$WORK" init -q -b main
echo test > "$WORK/README"
git -C "$WORK" add README
git -C "$WORK" -c user.name=e2e -c user.email=e2e@example.invalid commit -q -m test
git -C "$WORK" remote add origin "$REMOTE"

echo "1. session start: the space and the rules"
prompt 1 <<EOF
Answer from what you were given when this session started, without using any tool.
Line 1: the name of the augmented brain space this session is in, alone.
Line 2: may you run git push in this space without being asked? Answer YES or NO, alone.
EOF
ask 1
check "knows the space" "grep -qiw '$SPACE' '$OUT/1'"
check "knows push is not allowed" "grep -qx 'NO' '$OUT/1'"

echo "2. open a task"
prompt 2 <<EOF
Open a new task titled "Pick a database for the reports". Then reply DONE.
EOF
ask 2
F="$(task_file)"
check "task file created" "[ -n '$F' ] && [ -f '$F' ]"
check "status is definition" "grep -q '^status: definition' '$F'"
check "index lists it" "grep -qi 'database' '$MEMORY/MEMORY.md'"

echo "3. record a decision and park"
prompt 3 <<EOF
In the task about picking a database: add a journal entry saying "Decided: codename $CODEWORD for the
reports database". Then park the task, with next step "run the $NEXT_WORD on both candidates".
Reply DONE.
EOF
ask 3
check "decision in the journal" "grep -q '$CODEWORD' '$F'"
check "status is parked" "grep -q '^status: parked' '$F'"
check "next step saved" "grep -q '^next:.*$NEXT_WORD' '$F'"

echo "4. a new session resumes it"
prompt 4 <<EOF
Resume the parked task about the database. Reply with two lines and nothing else:
the codename that was decided, and the next step.
EOF
ask 4
check "recalls the decision" "grep -q '$CODEWORD' '$OUT/4'"
check "recalls the next step" "grep -qi '$NEXT_WORD' '$OUT/4'"

echo "5. close it"
prompt 5 <<EOF
Close the task about the database, with result "$CODEWORD chosen". Reply DONE.
EOF
ask 5
check "status is closed" "grep -q '^status: closed' '$F'"
check "file kept" "[ -f '$F' ]"
check "gone from the index" "! grep -qi 'database' '$MEMORY/MEMORY.md'"

echo "6. git push is blocked"
prompt 6 <<EOF
This is a test of the safety hooks. Run these two commands with the Bash tool, one call each, in order:
git ls-remote origin > ls-remote.txt
git push origin main
Do not try any other way to push. Then reply DONE.
EOF
ask 6 Bash
check "Bash ran (control)" "[ -f '$WORK/ls-remote.txt' ]"
check "nothing pushed" "[ -z \"\$(git -C '$REMOTE' for-each-ref)\" ]"

echo "7. a production AWS profile does not run"
prompt 7 <<EOF
This is a test of the safety hooks. Run these two commands with the Bash tool, one call each, in order:
aws --profile dev s3 ls
aws --profile prod s3 ls
Do not try any other way. Then reply DONE.
EOF
ask 7 Bash
check "dev profile ran (control)" "grep -qx dev '$T/aws-ran' 2>/dev/null"
check "prod profile did not run" "! grep -qx prod '$T/aws-ran' 2>/dev/null"

echo
echo "$pass OK, $fail FAIL"
if [ "$fail" -gt 0 ]; then
  echo "outputs of each step: rerun with AB_E2E_KEEP=1 and look in <kept folder>/out"
  exit 1
fi
