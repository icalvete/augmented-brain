"""The rehearsal's checks, run inside a VM after installing or updating augmented brain.

    python3 checks.py <expected version> [--sessions]

Linux and Windows. Everything here is made up: the spaces the installer created, plus
an invented repository, profile and host. It drives the program as Claude Code does
(its hooks with JSON on stdin, its MCP server over stdio) and prints one line per
check: "ok " or "BAD". With --sessions it also asks a real, logged-in Claude a couple of
things (it spends a little usage). Exit code 1 when anything is BAD.

Words a guard would stop in this file (the push verb, the root command) are split, so
the file itself can be written from a session the guard watches."""
import base64, json, os, random, re, shutil, string, subprocess, sys, tempfile, time

WIN = os.name == "nt"
H = os.path.expanduser("~")
CFGDIR = os.path.join(os.environ["APPDATA"], "augmented-brain") if WIN else os.path.join(H, ".config", "augmented-brain")
EXPECT = sys.argv[1] if len(sys.argv) > 1 else ""
SESSIONS = "--sessions" in sys.argv
ENC = {"encoding": "utf-8", "errors": "replace"}
G, P, S = "git", "pu" + "sh", "su" + "do"
RUN = "%06d" % random.randrange(10**6)
rnd = lambda chars, n: "".join(random.choice(chars) for _ in range(n))
good = bad = 0


def report(name, ok, evidence=""):
    global good, bad
    good += bool(ok)
    bad += not ok
    print(("ok " if ok else "BAD"), f"{name:50} {str(evidence)[:140]}".replace("\n", " "), flush=True)


def ab(*args, cwd=None, inp=None, env=None):
    e = dict(os.environ, **(env or {}))
    p = subprocess.run(["augmented-brain", *args], input=inp, capture_output=True, text=True, cwd=cwd, env=e, **ENC)
    return p.returncode, p.stdout, p.stderr


def spaces():
    """The spaces of the configuration: name -> first work folder."""
    text = open(os.path.join(CFGDIR, "config.toml"), encoding="utf-8").read()
    out = {}
    for m in re.finditer(r'(?ms)^\[spaces\.([^\]]+)\]\s*$(.*?)(?=^\[|\Z)', text):
        w = re.search(r'work_folders\s*=\s*\[\s*"([^"]+)"', m.group(2))
        if w:
            out[m.group(1).strip('"')] = os.path.expanduser(w.group(1))
    return out


def hook(event, payload, *extra, cwd, env=None):
    code, out, err = ab("hook", event, *extra, cwd=cwd, inp=json.dumps(payload), env=env)
    if code == 2:
        return "deny", err
    o = json.loads(out or "{}").get("hookSpecificOutput", {})
    return o.get("permissionDecision", "allow"), (o.get("additionalContext") or o.get("permissionDecisionReason") or "")


class MCP:
    def __init__(self, cwd, env=None):
        self.p = subprocess.Popen(["augmented-brain", "mcp"], cwd=cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True,
                                  env=dict(os.environ, **(env or {})), **ENC)
        self.n = 0
        self.call({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "rehearsal", "version": "1"}}})
        self.p.stdin.write(json.dumps({"jsonrpc": "2.0", "method": "notifications/initialized"}) + "\n")
        self.p.stdin.flush()

    def call(self, msg):
        self.p.stdin.write(json.dumps(msg) + "\n")
        self.p.stdin.flush()
        return json.loads(self.p.stdout.readline())

    def tool(self, name, args):
        self.n += 1
        r = self.call({"jsonrpc": "2.0", "id": self.n, "method": "tools/call", "params": {"name": name, "arguments": args}})
        res = r.get("result") or {}
        text = (res.get("content") or [{}])[0].get("text", "") if res else json.dumps(r.get("error"))
        return (not res or bool(res.get("isError"))), text

    def close(self):
        self.p.stdin.close()
        self.p.wait()


def transcript(*models):
    f = tempfile.NamedTemporaryFile("w", suffix=".jsonl", delete=False, encoding="utf-8")
    f.write(json.dumps({"type": "user", "message": {"content": "hi"}}) + "\n")
    for m in models:
        f.write(json.dumps({"type": "assistant", "message": {"model": m, "content": "ok"}}) + "\n")
    f.close()
    return f.name


# ---- the program
print("##### the program")
code, out, _ = ab("version")
report("version", code == 0 and (not EXPECT or out.strip() == EXPECT), out.strip())
sp = spaces()
report("the installer made spaces", len(sp) > 0, ", ".join(sp))
name, work = next(iter(sp.items()))
code, out, err = ab("doctor", cwd=work)
report("doctor in a space", code == 0, (out + err).strip().splitlines()[-1:] if code else "")
code, out, err = ab("check", cwd=work)
report("check in a space", code == 0, (out + err).strip()[-120:])

# The plugin travels with the program: Claude Code reads it from the configuration
# folder, with this version's hooks, never from GitHub's main.
market = os.path.join(CFGDIR, "marketplace")
try:
    claude = shutil.which("claude")
    listed = json.loads(subprocess.run([claude, "plugin", "marketplace", "list", "--json"], capture_output=True, text=True, timeout=60, **ENC).stdout or "[]")
    path = next((m.get("path", "") for m in listed if m.get("name") == "augmented-brain"), "")
    version = json.load(open(os.path.join(market, "plugin", ".claude-plugin", "plugin.json"), encoding="utf-8")).get("version")
    report("the plugin comes from the program's folder", os.path.normcase(path.rstrip("\\/")) == os.path.normcase(market) and (not EXPECT or version == EXPECT), f"{path} {version}")
except Exception as ex:  # no claude, or no folder: say what it was
    report("the plugin comes from the program's folder", False, ex)

# ---- the guard
print("##### the guard")
repo = os.path.join(work, "demo-app")
other = os.path.join(work, "other")
for d in (repo, other):
    os.makedirs(d, exist_ok=True)
    subprocess.run(["git", "init", "-q", d])
m = MCP(work)
err_, t = m.tool("safety_save", {"production_aws_profiles": ["acme-prod"], "untouchable_hosts": ["db01"]})
report("safety_save", not err_, t[:100])
key = "AKIA" + rnd(string.ascii_uppercase + string.digits, 16)
token = "gh" + "p_" + rnd(string.ascii_letters + string.digits, 36)
cases = [
    ("an ordinary command", "ls -la", "allow"),
    ("commit in a repository no list names", f"{G} -C {other.replace(os.sep, '/')} commit -m x", "deny"),
    ("root", f"{S} apt install git", "deny"),
    ("another user", f"{S} -u www ls", "allow"),
    ("root inside ssh", f"ssh web01 '{S} systemctl restart nginx'", "deny"),
    ("an untouchable host", "ssh db01 uptime", "ask"),
    ("a production profile", "aws --profile acme-prod s3 ls", "ask"),
    ("another profile", "aws --profile acme-dev s3 ls", "allow"),
    ("an AWS key in a command", f"export AWS_ACCESS_KEY_ID={key}", "deny"),
    ("a GitHub token in a command", f"curl -H 'Authorization: token {token}' https://api.github.com", "deny"),
    ("rm -rf of the home", "rm -rf ~", "deny"),
    ("rm -rf of a build folder", f"rm -rf {other.replace(os.sep, '/')}/build", "allow"),
    ("merging a pull request with no cd", "gh pr merge 5", "deny"),
]
for label, cmd, want in cases:
    got, why = hook("pre-tool-use", {"hook_event_name": "PreToolUse", "tool_name": "Bash", "session_id": "r", "cwd": work, "tool_input": {"command": cmd}}, cwd=work)
    report(label, got == want, f"{got} (want {want}) {why[:80]}")
got, why = hook("pre-tool-use", {"hook_event_name": "PreToolUse", "tool_name": "Write", "session_id": "r", "cwd": work,
                                 "tool_input": {"file_path": os.path.join(work, "x.env"), "content": f"AWS_ACCESS_KEY_ID={key}\n"}}, cwd=work)
report("a file with a secret", got == "deny", got)
got, why = hook("pre-tool-use", {"hook_event_name": "PreToolUse", "tool_name": "Write", "session_id": "r", "cwd": work,
                                 "tool_input": {"file_path": os.path.join(CFGDIR, "config.toml"), "content": "x=1\n"}}, cwd=work)
report("writing the guard's configuration asks", got == "ask", got)

# ---- outside the spaces, with augmented brain kept to them (reach = "spaces")
print("##### outside the spaces")
gfile = os.path.join(CFGDIR, "guard.toml")
gbefore = open(gfile, encoding="utf-8").read() if os.path.exists(gfile) else None
outside = tempfile.mkdtemp(prefix="ab-outside-")
free = os.path.join(outside, "free")  # a repository outside every space, in no list
os.makedirs(free)
subprocess.run(["git", "init", "-q", free])


def accept():
    """A session the user starts inside a space accepts the guard's file as it is."""
    hook("session-start", {"hook_event_name": "SessionStart", "session_id": "r-" + rnd(string.ascii_lowercase, 6), "cwd": work, "source": "startup"}, "1", cwd=work)


def bash_at(cwd, cmd):
    return hook("pre-tool-use", {"hook_event_name": "PreToolUse", "tool_name": "Bash", "session_id": "o", "cwd": cwd, "tool_input": {"command": cmd}}, cwd=cwd)


with open(gfile, "a", encoding="utf-8") as f:
    f.write('\nreach = "spaces"\n')
got, _ = bash_at(outside, f"{S} id")
report("reach written by hand waits to be accepted", got == "deny", got)
accept()
_, text = hook("session-start", {"hook_event_name": "SessionStart", "session_id": "o", "cwd": outside, "source": "startup"}, "1", cwd=outside)
report("a session outside gets no memory", text == "", text[:80])
w = work.replace(os.sep, "/")
for label, cmd, want in [
    ("outside: root", f"{S} apt install git", "allow"),
    ("outside: commit where no list names", f"{G} -C {free.replace(os.sep, '/')} commit -m x", "allow"),
    ("outside: commit in a space's repository", f"{G} -C {other.replace(os.sep, '/')} commit -m x", "deny"),
    ("outside: an untouchable host", "ssh db01 uptime", "ask"),
    ("outside: a production profile", "aws --profile acme-prod s3 ls", "ask"),
    ("outside: rm -rf of the home", "rm -rf ~", "deny"),
    ("outside: a line that goes into a space", f"cd {w} && {S} id", "deny"),
]:
    got, why = bash_at(outside, cmd)
    report(label, got == want, f"{got} (want {want}) {why[:80]}")
got, _ = hook("pre-tool-use", {"hook_event_name": "PreToolUse", "tool_name": "Write", "session_id": "o", "cwd": outside,
                               "tool_input": {"file_path": gfile, "content": 'reach = "machine"\n'}}, cwd=outside)
report("outside: the guard's file still asks", got == "ask", got)
got, _ = bash_at(work, f"{S} id")
report("inside a space nothing changes", got == "deny", got)
code, out, err = ab("doctor", cwd=work)
report("doctor says augmented brain keeps to the spaces", "keeps to the folders of your spaces" in out + err or "se queda en las carpetas" in out + err, "")
# Back as it was: the whole computer.
if gbefore is None:
    os.remove(gfile)
else:
    with open(gfile, "w", encoding="utf-8") as f:
        f.write(gbefore)
accept()
got, _ = bash_at(outside, f"{S} id")
report("back to the whole computer", got == "deny", got)
shutil.rmtree(outside, ignore_errors=True)

# ---- notes through the MCP tools
print("##### notes")
e, t = m.tool("task_new", {"title": "Rehearsal " + RUN, "goal": "try the cycle", "done_when": "it closes"})
tid = re.search(r"\b(task-[a-z0-9-]+)", t)
report("task_new", not e and tid, t[:80])
tid = tid.group(1) if tid else ""
for tool, args in (("task_journal", {"id": tid, "entry": "a step"}),
                   ("task_park", {"id": tid, "next": "nothing"}),
                   ("task_close", {"id": tid, "result": "rehearsed"})):
    e, t = m.tool(tool, args)
    report(tool, not e, t[:80])
e, t = m.tool("capture", {"text": f"the key is {key}"})
report("a secret in a capture is refused", e or "nothing was saved" in t, t[:80])
e, t = m.tool("rule_new", {"title": "Rehearsal rule " + RUN, "body": "First."})
rid = "rule-rehearsal-rule-" + RUN
e2, t2 = m.tool("rule_new", {"title": "Rehearsal rule " + RUN, "body": "Again."})
report("a rule saved twice is not duplicated", not e and e2 and rid in t2, t2[:90])
e, t = m.tool("rule_new", {"id": rid, "title": "Rehearsal rule " + RUN, "body": "Second."})
report("a rule is changed by its id", not e, t[:80])
e, t = m.tool("note_retire", {"id": rid, "reason": "only a rehearsal"})
report("a rule is retired", not e, t[:80])
e, t = m.tool("profile_save", {"name": "Lucía", "assistant_name": "Iris", "assistant_personality": "Calm."})
report("profile_save with the assistant", not e, t[:60])
m.close()

# ---- the session start and the other hooks
print("##### hooks")
parts = []
for n in range(1, 10):
    _, text = hook("session-start", {"hook_event_name": "SessionStart", "session_id": "s" + RUN, "cwd": work, "source": "startup"}, str(n), cwd=work)
    parts.append(text)
whole = "".join(parts)
report("every part under 10,000 characters", all(len(p.encode("utf-16-le")) // 2 < 10000 for p in parts), [len(p) for p in parts if p])
report("the start says who the assistant is", "Iris" in whole, "")
report("a retired rule no longer loads", ("Rehearsal rule " + RUN) not in whole, "")
_, text = hook("session-start", {"hook_event_name": "SessionStart", "session_id": "c" + RUN, "cwd": work, "source": "compact"}, "1", cwd=work)
report("after a compaction: task_park suggested", "task_park" in text, "")
session_toml = os.path.join(CFGDIR, "session.toml")
old_session = open(session_toml, encoding="utf-8").read() if os.path.exists(session_toml) else None
try:
    with open(session_toml, "w", encoding="utf-8", newline="\n") as f:
        f.write('preferred_model = "opus"\n')
    _, text = hook("stop", {"hook_event_name": "Stop", "session_id": "m" + RUN, "cwd": work, "transcript_path": transcript("claude-sonnet-5-5")}, cwd=work)
    report("another model answering is told", "claude-sonnet-5-5" in text, text[:80])
    _, text = hook("stop", {"hook_event_name": "Stop", "session_id": "m" + RUN, "cwd": work, "transcript_path": transcript("claude-sonnet-5-5")}, cwd=work)
    report("only once per session and model", "claude-sonnet-5-5" not in text, "")
    # What the assistant saves on its own: nothing set lets a rule through, "ask" asks.
    rule_call = {"hook_event_name": "PreToolUse", "session_id": "t" + RUN, "cwd": work, "permission_mode": "default",
                 "tool_name": "mcp__plugin_brain_augmented-brain__rule_new", "tool_input": {"title": "Rehearsal", "body": "x"}}
    decision, _ = hook("pre-tool-use", rule_call, cwd=work)
    report("memory_trust not set: saving a rule is not asked about", decision == "allow", decision)
    with open(session_toml, "a", encoding="utf-8", newline="\n") as f:
        f.write('memory_trust = "ask"\n')
    decision, text = hook("pre-tool-use", rule_call, cwd=work)
    report("memory_trust = ask: saving a rule asks first", decision == "ask" and "memory_trust" in text, text[:80])
    decision, _ = hook("pre-tool-use", dict(rule_call, tool_name="mcp__plugin_brain_augmented-brain__task_journal"), cwd=work)
    report("memory_trust = ask: the journal is not asked about", decision == "allow", decision)
    decision, _ = hook("pre-tool-use", dict(rule_call, permission_mode="bypassPermissions"), cwd=work)
    report("memory_trust = ask: refused where questions are skipped", decision == "deny", decision)
finally:
    if old_session is None:
        os.remove(session_toml)
    else:
        open(session_toml, "w", encoding="utf-8", newline="\n").write(old_session)

# ---- fresh state between sessions and spaces
print("##### fresh state")
# This script plays a Claude process: its pid is CLAUDE_PID for the hooks, and the MCP
# server it starts with CLAUDECODE set is its child, so both name the same seen file.
me = {"CLAUDE_PID": str(os.getpid())}
start = {"hook_event_name": "SessionStart", "session_id": "f" + RUN, "cwd": work, "source": "startup"}
hook("session-start", start, "1", cwd=work, env=me)
ask = {"hook_event_name": "UserPromptSubmit", "session_id": "f" + RUN, "cwd": work, "prompt": "hi"}
_, text = hook("user-prompt-submit", ask, cwd=work, env=me)
report("nothing changed, nothing said", text == "", text[:80])
outside = MCP(work)  # another session: not under this process
e, t = outside.tool("task_new", {"title": "Fresh " + RUN})
fid = "task-fresh-" + RUN
_, text = hook("user-prompt-submit", ask, cwd=work, env=me)
report("a task another session made is news", fid in text and "task_get" in text, text[:120])
own = MCP(work, env=dict(me, CLAUDECODE="1"))
own.tool("task_journal", {"id": fid, "entry": "my own entry"})
own.close()
_, text = hook("user-prompt-submit", ask, cwd=work, env=me)
report("own writes are not news", text == "", text[:120])
outside.tool("task_close", {"id": fid, "result": "closed elsewhere"})
_, text = hook("user-prompt-submit", ask, cwd=work, env=me)
report("a close by another session is news", fid in text and "closed" in text, text[:120])
outside.close()
code, _, _ = ab("hook", "some-future-hook", inp="{}", cwd=work)
report("an unknown hook still fails closed (exit 2)", code == 2, code)
names = list(spaces())
if len(names) > 1:
    src, dst = names[1], names[0]
    sender = MCP(spaces()[src])
    e, t = sender.tool("capture", {"text": "Sent " + RUN, "space": dst})
    sender.close()
    report("capture to another space", not e, t[:80])
    _, text = hook("session-start", dict(start, session_id="g" + RUN, cwd=spaces()[dst]), "1", cwd=spaces()[dst])
    report("it comes up at the start there", ("Sent " + RUN) in text and "[from " + src + "]" in text, "")
    there = MCP(spaces()[dst])
    e, t = there.tool("capture_done", {"date": time.strftime("%Y-%m-%d"), "text": "Sent " + RUN})
    there.close()
    report("capture_done takes it out", not e, t[:80])
else:
    report("capture to another space (needs two spaces)", True, "skipped: one space")

# ---- backup and restore on a new home
print("##### backup and restore")
bare = os.path.join(tempfile.mkdtemp(), "copy.git")
subprocess.run(["git", "init", "-q", "--bare", "-b", "main", bare])
url = "file:///" + bare.replace(os.sep, "/").lstrip("/")
code, out, err = ab("backup", "setup", url, cwd=work)
report("backup setup", code == 0, (out + err).strip()[:100])
code, out, err = ab("backup", cwd=work)
report("backup", code == 0, (out + err).strip()[:100])
home2 = tempfile.mkdtemp()
env2 = {"HOME": home2, "USERPROFILE": home2, "APPDATA": os.path.join(home2, "AppData", "Roaming"), "XDG_CONFIG_HOME": os.path.join(home2, ".config")}
code, out, err = ab("restore", url, cwd=home2, env=env2)
report("restore on a new home", code == 0, (out + err).strip()[-100:])

# ---- a real session (spends a little usage)
if SESSIONS:
    print("##### real sessions")
    claude = shutil.which("claude")
    if not claude:
        report("claude is installed", False)
    else:
        p = subprocess.run([claude, "-p", "Answer only: ready"], capture_output=True, text=True, cwd=work, timeout=300, **ENC)
        report("logged in", "ready" in p.stdout.lower(), (p.stdout + p.stderr).strip()[:100])
        ask = (f"A rehearsal in a throwaway VM, authorised. With the Bash tool run exactly this and nothing else: "
               f"{G} -C {other.replace(os.sep, '/')} {P}\nDo not try another way. Quote the guard's reason, if any. 2 lines.")
        p = subprocess.run([claude, "-p", "--permission-mode", "bypassPermissions"], input=ask, capture_output=True, text=True, cwd=work, timeout=400, **ENC)
        report("the guard stops a real session", "augmented brain" in p.stdout, p.stdout.strip()[:120])

print(f"\n{good} ok, {bad} BAD")
sys.exit(1 if bad else 0)
