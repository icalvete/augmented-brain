"""The smoke test with a real systemd, in the rehearsal's Ubuntu VM, around a reboot.

    python3 smoke.py before     # remote install, with a pretend claude
    (the VM reboots)
    python3 smoke.py after      # the unit came back, right; session start runs one

Linux only. On 9-nov the services of `remote install` came back after a reboot with no
PATH, so the session found neither claude nor augmented-brain: the Go tests used a fake
systemd and could not see it. Here systemd is the real one, and only claude pretends: a
shell script, in a folder of its own (not the program's), that writes to the journal
what it got (its arguments, folder, PATH, which augmented-brain it finds, the task) and
then waits. For a session launched for a task it also runs the SessionStart hook, as a
real claude would, to see the hook tell it its task.

Everything is made up. The state between the two halves is in ~/ab-release/smoke.json.
Prints one line per check, "ok " or "BAD", and exits 1 when anything is BAD."""
import json, os, random, re, subprocess, sys, time

H = os.path.expanduser("~")
CFGDIR = os.path.join(H, ".config", "augmented-brain")
STATE = os.path.join(H, "ab-release", "smoke.json")
FAKEBIN = os.path.join(H, "ab-smoke", "bin")
MODE = "auto"  # not the default, so a unit that lost it shows
ENC = {"encoding": "utf-8", "errors": "replace"}
good = bad = 0

FAKE = r'''#!/bin/sh
[ "$1" = remote-control ] || exit 0
{
  echo "fake claude: args=$*"
  echo "fake claude: folder=$(pwd)"
  echo "fake claude: path=$PATH"
  echo "fake claude: program=$(command -v augmented-brain || echo none)"
  echo "fake claude: task=${AUGMENTED_BRAIN_TASK:-}"
  if [ -n "${AUGMENTED_BRAIN_TASK:-}" ]; then
    printf '{"hook_event_name":"SessionStart","session_id":"smoke","cwd":"%s","source":"startup"}' "$(pwd)" |
      augmented-brain hook session-start 1 | grep -q "$AUGMENTED_BRAIN_TASK" && echo "fake claude: hook=task" || echo "fake claude: hook=none"
  fi
} >&2
exec sleep 3600
'''


def report(name, ok, evidence=""):
    global good, bad
    good += bool(ok)
    bad += not ok
    print(("ok " if ok else "BAD"), f"{name:50} {str(evidence)[:200]}".replace("\n", " | "), flush=True)


def run(*args, env=None, cwd=None):
    p = subprocess.run(list(args), capture_output=True, text=True, cwd=cwd, env=dict(os.environ, **(env or {})), **ENC)
    return p.returncode, (p.stdout + p.stderr).strip()


def first_space():
    """The default space and its first work folder."""
    text = open(os.path.join(CFGDIR, "config.toml"), encoding="utf-8").read()
    want = re.search(r'(?m)^default_space\s*=\s*"([^"]+)"', text)
    for m in re.finditer(r'(?ms)^\[spaces\.([^\]]+)\]\s*$(.*?)(?=^\[|\Z)', text):
        name = m.group(1).strip('"')
        w = re.search(r'work_folders\s*=\s*\[\s*"([^"]+)"', m.group(2))
        if w and (not want or want.group(1) == name):
            return name, os.path.expanduser(w.group(1))
    raise SystemExit("smoke: no space with a work folder")


def trust(*folders):
    """What `claude` once in each folder would leave in ~/.claude.json."""
    path = os.path.join(H, ".claude.json")
    data = json.load(open(path, encoding="utf-8")) if os.path.exists(path) else {}
    for f in folders:
        data.setdefault("projects", {}).setdefault(f, {})["hasTrustDialogAccepted"] = True
    json.dump(data, open(path, "w", encoding="utf-8"))


def new_task(cwd, title):
    """A task made through the MCP server, as a session would."""
    p = subprocess.Popen(["augmented-brain", "mcp"], cwd=cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, **ENC)

    def call(msg):
        p.stdin.write(json.dumps(msg) + "\n")
        p.stdin.flush()
        return json.loads(p.stdout.readline()) if "id" in msg else None

    call({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "smoke", "version": "1"}}})
    call({"jsonrpc": "2.0", "method": "notifications/initialized"})
    r = call({"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "task_new", "arguments": {"title": title, "goal": "a smoke test"}}})
    p.stdin.close()
    p.wait()
    text = ((r.get("result") or {}).get("content") or [{}])[0].get("text", "")
    m = re.search(r"\b(task-[a-z0-9-]+)", text)
    return m.group(1) if m else ""


def journal(unit, wait_for, seconds=90):
    """The fake claude's lines in the unit's journal for this boot, once wait_for is there."""
    lines = ""
    for _ in range(seconds):
        _, lines = run("journalctl", "--user", "-b", "-u", unit, "-o", "cat", "--no-pager", "-q")
        if wait_for in lines:
            break
        time.sleep(1)
    return {m.group(1): m.group(2) for m in re.finditer(r"(?m)^fake claude: (\w+)=(.*)$", lines)}, lines


def boot_id():
    return open("/proc/sys/kernel/random/boot_id").read().strip()


def check_path(what, got, program_dir):
    dirs = got.get("path", "").split(":")
    report(f"{what}: claude's folder first in PATH", dirs[:1] == [FAKEBIN], got.get("path"))
    report(f"{what}: the program's folder in PATH", program_dir in dirs, got.get("path"))
    report(f"{what}: augmented-brain found from it", got.get("program") == os.path.join(program_dir, "augmented-brain"), got.get("program"))


def before():
    space, work = first_space()
    os.makedirs(FAKEBIN, exist_ok=True)
    claude = os.path.join(FAKEBIN, "claude")
    with open(claude, "w", encoding="utf-8") as f:
        f.write(FAKE)
    os.chmod(claude, 0o755)
    with open(os.path.join(CFGDIR, "remote.toml"), "w", encoding="utf-8") as f:
        f.write(f'permission_mode = "{MODE}"\n')
    trust(work)
    # The shell's PATH finds the pretend claude first; systemd will not have this PATH.
    code, out = run("augmented-brain", "remote", "install", cwd=work, env={"PATH": FAKEBIN + ":" + os.environ["PATH"]})
    report("remote install", code == 0, out[-200:])
    code, units = run("systemctl", "--user", "list-unit-files", "augmented-brain-remote-*", "--no-legend")
    unit = units.split()[0] if units.split() else ""
    report("a unit was written and enabled", unit and "enabled" in units, units)
    _, program = run("sh", "-c", "command -v augmented-brain")
    json.dump({"space": space, "work": work, "unit": unit, "program_dir": os.path.dirname(program), "boot": boot_id()}, open(STATE, "w"))
    code, state = run("systemctl", "--user", "is-active", unit)
    report("it runs before the reboot", state == "active", state)


def after():
    s = json.load(open(STATE))
    unit, work, program_dir = s["unit"], s["work"], s["program_dir"]
    report("the VM rebooted", boot_id() != s["boot"], boot_id())
    got, raw = journal(unit, "fake claude: program=")
    code, state = run("systemctl", "--user", "is-active", unit)
    report("the unit came back after the reboot", state == "active" and got, f"{state} {raw[-160:]}")
    report("…with its permission mode", f"--permission-mode {MODE}" in got.get("args", ""), got.get("args"))
    report("…in the space's folder", got.get("folder") == work, got.get("folder"))
    check_path("the unit", got, program_dir)
    code, out = run("augmented-brain", "remote", "status", cwd=work)
    report("remote status says it runs", code == 0, out[-160:])

    # session start: a transient unit for one task, in a folder of its own
    folder = os.path.join(work, "smoke-%06d" % random.randrange(10**6))
    os.makedirs(folder)
    trust(folder)
    task = new_task(work, "Smoke " + os.path.basename(folder))
    report("a task to launch", task, task)
    # The same pretend claude, found first in the shell's PATH as at install: the real
    # one the installer put in ~/.local/bin is not signed in.
    code, out = run("augmented-brain", "session", "start", task, "--folder", folder, cwd=work,
                    env={"PATH": FAKEBIN + ":" + os.environ["PATH"]})
    report("session start", code == 0, out[-200:])
    sunit = "augmented-brain-session-" + re.sub(r"[^A-Za-z0-9_.-]+", "-", task.removeprefix("task-")) + ".service"
    got, raw = journal(sunit, "fake claude: hook=", 30)
    _, state = run("systemctl", "--user", "is-active", sunit)
    report("its unit runs", state == "active", state)
    report("…as a session spawner, with the mode", "--spawn session" in got.get("args", "") and f"--permission-mode {MODE}" in got.get("args", ""), got.get("args"))
    report("…in its folder, with its task", got.get("folder") == folder and got.get("task") == task, f"{got.get('folder')} {got.get('task')}")
    check_path("the session", got, program_dir)
    report("…and the session start tells it its task", got.get("hook") == "task", got.get("hook") or raw[-160:])
    code, out = run("augmented-brain", "session", "stop", task, cwd=work)
    report("session stop", code == 0, out[-120:])
    code, out = run("augmented-brain", "remote", "uninstall", cwd=work)
    report("remote uninstall", code == 0, out[-120:])


if __name__ == "__main__":
    phase = sys.argv[1] if len(sys.argv) > 1 else ""
    print(f"##### smoke with a real systemd: {phase}")
    {"before": before, "after": after}.get(phase, lambda: sys.exit("usage: smoke.py before|after"))()
    print(f"\n{good} ok, {bad} BAD")
    sys.exit(1 if bad else 0)
