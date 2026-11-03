#!/bin/sh
# compat-check.sh — what this version writes, the releases before it must still read.
#
#   scripts/compat-check.sh <new binary> [old version ...]
#
# A machine runs the old program for a while after an update: the Claude sessions that
# were open keep it until restarted, and other machines may not be updated yet. So the
# new program writes everything it can (tasks, people, secrets' names, captures, rules,
# what a repository is, a backup), and each old release, downloaded from GitHub, must read it all: doctor,
# check, its MCP tools, the session start, and a restore of the backup. With no version
# given, the last two releases. Needs git, python3, curl and tar. Linux and macOS.
set -eu

NEW="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
shift
REPO=icalvete/augmented-brain
# Everything happens in a folder of its own, left in place (CI throws the machine away;
# by hand, say where with COMPAT_DIR and delete it after).
WORK="${COMPAT_DIR:-$(mktemp -d)}"
mkdir -p "$WORK"
echo "compat-check: working in $WORK"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) echo "unknown machine $(uname -m)"; exit 1 ;; esac

if [ $# -eq 0 ]; then
  # The last two published releases.
  # With a token (CI passes GH_TOKEN), GitHub's API does not hit the low limit of
  # anonymous calls, which runners share and which answered 403 on macOS.
  if [ -n "${GH_TOKEN:-}" ]; then auth="Authorization: Bearer $GH_TOKEN"; else auth="Accept: application/json"; fi
  set -- $(curl -fsSL -H "$auth" "https://api.github.com/repos/$REPO/releases?per_page=2" |
    python3 -c 'import json,sys; print(" ".join(r["tag_name"].lstrip("v") for r in json.load(sys.stdin)))')
fi
if [ $# -eq 0 ]; then
  # Checking against nothing would pass without checking anything.
  echo "compat-check: no earlier release to check against (GitHub did not answer?)"
  exit 1
fi

# mcp <binary> <folder> <calls.json>: runs MCP tools as Claude Code would; fails on a
# tool error.
mcp() {
  python3 - "$1" "$2" "$3" <<'EOF'
import json, subprocess, sys
binary, folder, calls = sys.argv[1], sys.argv[2], json.load(open(sys.argv[3]))
p = subprocess.Popen([binary, "mcp"], cwd=folder, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
def call(msg):
    p.stdin.write(json.dumps(msg) + "\n"); p.stdin.flush()
    return json.loads(p.stdout.readline()) if "id" in msg else None
call({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "compat-check", "version": "1"}}})
call({"jsonrpc": "2.0", "method": "notifications/initialized"})
bad = 0
for i, (tool, args) in enumerate(calls, start=1):
    r = call({"jsonrpc": "2.0", "id": i, "method": "tools/call", "params": {"name": tool, "arguments": args}})
    res = r.get("result", {})
    text = (res.get("content") or [{}])[0].get("text", "") if res else json.dumps(r.get("error"))
    if not res or res.get("isError"):
        print(f"  FAIL {tool}: {text[:300]}"); bad += 1
    else:
        print(f"  ok   {tool}")
p.stdin.close(); p.wait()
sys.exit(1 if bad else 0)
EOF
}

export AB_NO_UPDATE_CHECK=1
# Each part runs with HOME and XDG_CONFIG_HOME inside $WORK: nothing outside it is touched.
fail=0

# The new program writes, in a home of its own.
export HOME="$WORK/home" XDG_CONFIG_HOME="$WORK/home/.config"
mkdir -p "$HOME"
git config --global user.email compat@example.com
git config --global user.name compat
git config --global init.defaultBranch main
"$NEW" init --yes >/dev/null
WORKDIR="$HOME/augmented-brain/work"
# A repository of the user's, for repo_describe (its note is an ordinary reference).
mkdir -p "$WORKDIR/code/site"
cat > "$WORK/write.json" <<'EOF'
[
 ["task_new", {"title": "Bake bread", "summary": "for the shop"}],
 ["task_journal", {"id": "task-bake-bread", "entry": "Tried the oven"}],
 ["task_step", {"id": "task-bake-bread", "add": "Buy flour"}],
 ["task_update", {"id": "task-bake-bread", "waiting_on": "Ana", "next": "Ask Ana", "watch_date": "2030-01-01", "watch_what": "check the oven"}],
 ["person_save", {"name": "Ana Ruiz", "aliases": ["Ana"], "role": "baker", "notes": "Starts at six."}],
 ["capture", {"text": "idea for later", "remind_on": "2030-01-02"}],
 ["knowledge_new", {"kind": "lesson", "title": "Bread needs time", "body": "Let it rest."}],
 ["rule_new", {"title": "Short answers", "summary": "one idea each", "body": "Answer in short sentences.", "shared": true}],
 ["secret_name_save", {"service": "mail", "name": "smtp", "what": "sending mail"}],
 ["safety_save", {"untouchable_hosts": ["db01"]}],
 ["repo_describe", {"repo": "code/site", "text": "The shop's website"}]
]
EOF
echo "new program ($("$NEW" version)) writes:"
mcp "$NEW" "$WORKDIR" "$WORK/write.json" || fail=1
(cd "$WORKDIR" && echo '{"session_id":"s1","cwd":"'"$WORKDIR"'"}' | "$NEW" hook session-start 1 >/dev/null)
(cd "$WORKDIR" && "$NEW" check >/dev/null) || fail=1
git init -q --bare "$WORK/backup.git"
(cd "$WORKDIR" && "$NEW" backup setup "$WORK/backup.git" >/dev/null && "$NEW" backup >/dev/null) || { echo "  FAIL backup"; fail=1; }
cp -R "$HOME" "$WORK/written"

cat > "$WORK/read.json" <<'EOF'
[
 ["where_am_i", {}],
 ["pending", {}],
 ["task_get", {"id": "task-bake-bread"}],
 ["task_journal", {"id": "task-bake-bread", "entry": "Read by an older program"}],
 ["weekly_review", {}]
]
EOF

for v in "$@"; do
  echo "release $v reads what the new one wrote:"
  dir="$WORK/old-$v"
  mkdir -p "$dir"
  if ! curl -fsSL "https://github.com/$REPO/releases/download/v$v/augmented-brain_${v}_${os}_${arch}.tar.gz" | tar -xz -C "$dir"; then
    echo "  FAIL could not download $v"; fail=1; continue
  fi
  OLD=$(find "$dir" -type f -name augmented-brain | head -n 1)
  # Each release reads its own copy of what the new one wrote, at the same paths as
  # where it was written, so the configuration's folders hold.
  mv "$WORK/home" "$WORK/home-before-$v"
  cp -R "$WORK/written" "$WORK/home"
  out=$(cd "$WORKDIR" && "$OLD" doctor 2>&1) || true
  if printf '%s' "$out" | grep -qi "unknown settings\|cannot be read\|newer augmented brain"; then
    echo "  FAIL doctor:"; printf '%s\n' "$out" | sed 's/^/    /'; fail=1
  else
    echo "  ok   doctor"
  fi
  if (cd "$WORKDIR" && "$OLD" check >/dev/null 2>&1); then echo "  ok   check"; else echo "  FAIL check"; fail=1; fi
  start=$(cd "$WORKDIR" && echo '{"session_id":"s2","cwd":"'"$WORKDIR"'"}' | "$OLD" hook session-start 1 2>&1)
  if printf '%s' "$start" | grep -qi "unknown settings\|could not\|cannot be read"; then
    echo "  FAIL session start:"; printf '%s\n' "$start" | grep -i "unknown settings\|could not\|cannot be read" | sed 's/^/    /'; fail=1
  else
    echo "  ok   session start"
  fi
  mcp "$OLD" "$WORKDIR" "$WORK/read.json" || fail=1
  # A new machine restores the backup with the old program.
  export HOME="$WORK/fresh-$v" XDG_CONFIG_HOME="$WORK/fresh-$v/.config"
  mkdir -p "$HOME"
  if "$OLD" restore "$WORK/backup.git" >/dev/null 2>&1; then echo "  ok   restore"; else echo "  FAIL restore"; "$OLD" restore "$WORK/backup.git" 2>&1 | sed 's/^/    /' | tail -5; fail=1; fi
  export HOME="$WORK/home" XDG_CONFIG_HOME="$WORK/home/.config"
done

if [ "$fail" -ne 0 ]; then
  echo "compat-check: an earlier release cannot read what this one writes."
  exit 1
fi
echo "compat-check: the earlier releases read it all."
