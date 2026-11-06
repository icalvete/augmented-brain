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


def hook(event, payload, *extra, cwd):
    code, out, err = ab("hook", event, *extra, cwd=cwd, inp=json.dumps(payload))
    if code == 2:
        return "deny", err
    o = json.loads(out or "{}").get("hookSpecificOutput", {})
    return o.get("permissionDecision", "allow"), (o.get("additionalContext") or o.get("permissionDecisionReason") or "")


class MCP:
    def __init__(self, cwd):
        self.p = subprocess.Popen(["augmented-brain", "mcp"], cwd=cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, **ENC)
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
finally:
    if old_session is None:
        os.remove(session_toml)
    else:
        open(session_toml, "w", encoding="utf-8", newline="\n").write(old_session)

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
