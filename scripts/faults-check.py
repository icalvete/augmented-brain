#!/usr/bin/env python3
"""faults-check.py: what augmented brain does when something it uses is missing.

    python3 -I scripts/faults-check.py

Builds the program from this checkout and, in a throwaway home with made-up spaces,
takes away one thing at a time (Claude Code, gh, git, the network, Tailscale, systemd,
a backup, a configuration that loads, write access to the notes) and checks two things
each time: the program says what is missing and what to do, and it leaves nothing
broken or half done. One line per check, "ok " or "BAD"; exit code 1 when anything is
BAD. Linux, macOS and Windows; CI runs it on all three (ci.yml).

The programs it may use are put on PATH one by one, through small wrappers in a folder
of their own, so "no gh" means no gh even where gh sits next to git. Nothing outside
the throwaway home is touched. Downloads that fail (504, 404, a cut file) are tested in
Go: cmd/augmented-brain/installer_test.go and update_test.go, internal/update.
"""
import json, os, shutil, stat, subprocess, sys, tempfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WIN = os.name == "nt"
EXE = ".exe" if WIN else ""
ENC = {"encoding": "utf-8", "errors": "replace"}
good = bad = 0


def report(name, ok, evidence=""):
    global good, bad
    good += bool(ok)
    bad += not ok
    print(("ok " if ok else "BAD"), f"{name:58} {str(evidence)[:160]}".replace("\n", " "), flush=True)


TMP = tempfile.mkdtemp(prefix="ab-faults-")
HOME = os.path.join(TMP, "home")
BIN = os.path.join(TMP, "bin")  # the program alone
os.makedirs(HOME)
os.makedirs(BIN)
subprocess.run(["go", "build", "-o", os.path.join(BIN, "augmented-brain" + EXE), "./cmd/augmented-brain"], cwd=ROOT, check=True)
CFG = os.path.join(HOME, "cfg")
REAL = {n: shutil.which(n) for n in ("git", "gh", "claude", "tailscale", "systemctl", "secret-tool", "security")}


def tools(*names):
    """A folder with wrappers for the named programs (those this machine has), so the
    program finds those and no others."""
    d = tempfile.mkdtemp(prefix="tools-", dir=TMP)
    for n in names:
        real = REAL.get(n) or shutil.which(n)
        if not real:
            continue
        if WIN:
            with open(os.path.join(d, n + ".cmd"), "w") as f:
                f.write(f'@"{real}" %*\r\n')
        else:
            p = os.path.join(d, n)
            with open(p, "w") as f:
                f.write(f'#!/bin/sh\nexec "{real}" "$@"\n')
            os.chmod(p, 0o755)
    return d


def env(*names, **extra):
    """The environment of a run: the throwaway home and PATH with only the program and
    the named tools (plus the system's own folder on Windows, which cmd needs)."""
    path = [BIN, tools(*names)]
    if WIN:
        path.append(os.path.join(os.environ.get("SystemRoot", r"C:\Windows"), "System32"))
    e = {k: v for k, v in os.environ.items() if not k.startswith(("AB_", "GIT_", "GH_", "CLAUDE"))}
    e.update(HOME=HOME, USERPROFILE=HOME, AB_CONFIG_DIR=CFG, AB_LANGUAGE="en", AB_NO_BROWSER="1",
             AB_NO_UPDATE_CHECK="1", PATH=os.pathsep.join(path), XDG_CONFIG_HOME=os.path.join(HOME, ".config"),
             APPDATA=os.path.join(HOME, "AppData", "Roaming"), LOCALAPPDATA=os.path.join(HOME, "AppData", "Local"),
             CLAUDE_CONFIG_DIR=os.path.join(HOME, ".claude"), GIT_CONFIG_NOSYSTEM="1",
             GIT_AUTHOR_NAME="Lucia", GIT_AUTHOR_EMAIL="lucia@ejemplo.com",
             GIT_COMMITTER_NAME="Lucia", GIT_COMMITTER_EMAIL="lucia@ejemplo.com")
    e.update(extra)
    return e


def ab(args, tools_=("git",), cwd=None, inp="", timeout=120, **extra):
    try:
        p = subprocess.run([os.path.join(BIN, "augmented-brain" + EXE), *args], input=inp, capture_output=True,
                           text=True, cwd=cwd or HOME, env=env(*tools_, **extra), timeout=timeout, **ENC)
        return p.returncode, p.stdout + p.stderr
    except subprocess.TimeoutExpired:
        return "timeout", ""


def mcp(cwd, calls, tools_=("git",)):
    """Runs MCP tool calls through `augmented-brain mcp`; returns [(is_error, text)]."""
    # One message at a time, reading each answer, as Claude Code does: the server keeps
    # its input open while it works.
    p = subprocess.Popen([os.path.join(BIN, "augmented-brain" + EXE), "mcp"], cwd=cwd, env=env(*tools_),
                         stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, **ENC)

    def send(msg, answer=True):
        p.stdin.write(json.dumps(msg) + "\n")
        p.stdin.flush()
        return json.loads(p.stdout.readline()) if answer else None

    out = []
    try:
        send({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "faults", "version": "1"}}})
        send({"jsonrpc": "2.0", "method": "notifications/initialized"}, answer=False)
        for i, (name, a) in enumerate(calls, 1):
            r = send({"jsonrpc": "2.0", "id": i, "method": "tools/call", "params": {"name": name, "arguments": a}})
            res = r.get("result") or {}
            out.append((not res or bool(res.get("isError")), (res.get("content") or [{}])[0].get("text", "") if res else json.dumps(r.get("error"))))
    finally:
        p.stdin.close()
        p.wait(timeout=30)
    return out


def has(text, *words):
    t = text.lower()
    return all(w.lower() in t for w in words)


try:
    work = os.path.join(HOME, "augmented-brain", "work")
    personal = os.path.join(HOME, "augmented-brain", "personal")

    print("##### no Claude Code")
    code, out = ab(["init"])
    report("init works without Claude Code", code == 0 and os.path.isdir(work), out.strip().splitlines()[-1:] if code else "")
    report("init says how to install the plugin later", has(out, "plugin"), out.strip()[-160:])
    code, out = ab(["doctor"], cwd=work)
    report("doctor says Claude Code is missing", has(out, "Claude Code"), "")

    print("##### no backup")
    report("doctor names each space without a backup", has(out, "space work has no backup", "space personal has no backup"), "")
    code, out = ab(["uninstall", "--all"], inp="n\n")
    report("uninstall --all warns the notes would be gone for good", code != 0 and has(out, "gone for good"), out.strip()[-160:])
    report("answering no keeps everything", os.path.isfile(os.path.join(CFG, "config.toml")) and os.path.isdir(work), "")
    code, out = ab(["backup"], cwd=work)
    report("backup without one says how to set it up", code != 0 and has(out, "backup setup"), out.strip()[-160:])

    print("##### no gh")
    r = mcp(work, [("backup_setup", {})], tools_=("git",))
    report("backup_setup without gh returns the steps", len(r) == 1 and has(r[0][1], "gh"), r[0][1][:160] if r else "no answer")

    print("##### no git")
    remote = os.path.join(TMP, "remote.git")
    code, out = ab(["backup", "setup", remote], cwd=work, tools_=())
    report("backup setup without git says git is missing", code != 0 and has(out, "git"), out.strip()[-160:])
    report("…and leaves no backup half set up", "backup" not in open(os.path.join(CFG, "config.toml"), encoding="utf-8").read().split("[spaces.work]")[1].split("[")[0], "")
    code, out = ab(["check"], cwd=work, tools_=())
    report("check works without git", code == 0, out.strip()[-120:])
    code, out = ab(["doctor"], cwd=work, tools_=())
    report("doctor says git is missing", has(out, "git"), "")

    print("##### no network (a remote that is not there)")
    subprocess.run(["git", "init", "-q", "--bare", "-b", "main", remote], check=True)
    code, out = ab(["backup", "setup", remote], cwd=work)
    report("backup setup to a reachable remote", code == 0, out.strip()[-160:])
    code, out = ab(["backup"], cwd=work)
    report("backup", code == 0, out.strip()[-160:])
    shutil.move(remote, remote + ".away")
    code, out = ab(["backup"], cwd=work, timeout=200)
    report("backup to a remote that is gone fails, without hanging", code not in (0, "timeout"), out.strip()[-160:])
    shutil.move(remote + ".away", remote)
    code, out = ab(["backup"], cwd=work)
    report("…and the next one works", code == 0, out.strip()[-160:])

    print("##### no Tailscale")
    code, out = ab(["panel", "remote", "--no-browser"], cwd=work, timeout=60)
    report("panel remote without Tailscale says what to install", code != 0 and has(out, "tailscale"), out.strip()[-160:])
    ab(["panel", "stop"])

    print("##### no systemd")
    code, out = ab(["remote", "install"], cwd=work, tools_=("git", "claude"))
    report("remote install without systemd says so", code != 0 and (has(out, "systemd") or has(out, "not available")), out.strip()[-160:])
    remote_toml = os.path.join(CFG, "remote.toml")
    report("…and leaves remote access off", not os.path.exists(remote_toml) or "enabled = true" not in open(remote_toml, encoding="utf-8").read(), "")

    print("##### a configuration that does not load")
    ab(["space"], cwd=work)  # a command that keeps a version of the configuration
    cfgfile = os.path.join(CFG, "config.toml")
    good_text = open(cfgfile, encoding="utf-8").read()
    with open(cfgfile, "w", encoding="utf-8") as f:
        f.write(good_text.replace("[spaces.", "[[spaces.", 1))
    code, out = ab(["doctor"], cwd=work)
    report("doctor points to config restore", code != 0 and has(out, "config restore"), "")
    code, out = ab(["config", "history"], cwd=work)
    report("config history works with it broken", code == 0 and " 1  " in out, out.strip()[-160:])
    code, out = ab(["config", "restore", "1"], cwd=work)
    code2, _ = ab(["check"], cwd=work)
    report("config restore brings back one that loads", code == 0 and code2 == 0, out.strip()[-160:])

    if not WIN and os.geteuid() != 0:
        print("##### notes that cannot be written")
        notes = os.path.join(work, "notes")
        tasks = os.path.join(notes, "tasks")
        before = sorted(os.listdir(tasks))
        os.chmod(tasks, stat.S_IRUSR | stat.S_IXUSR)
        try:
            r = mcp(work, [("task_new", {"title": "Bake bread", "summary": "a test"})])
            report("task_new says it could not write", len(r) == 1 and r[0][0], r[0][1][:160] if r else "no answer")
        finally:
            os.chmod(tasks, stat.S_IRWXU)
        report("…and leaves no half-written note", sorted(os.listdir(tasks)) == before, "")
        code, out = ab(["check"], cwd=work)
        report("the notes are still valid", code == 0, out.strip()[-120:])
finally:
    shutil.rmtree(TMP, ignore_errors=True)

print(f"\n{good} ok, {bad} BAD")
sys.exit(1 if bad else 0)
