#!/usr/bin/env python3
"""remote-check.py: `augmented-brain remote` for real, with a claude that only pretends.

    python3 -I scripts/remote-check.py

Builds the program from this checkout and, in a throwaway home with made-up spaces,
runs `remote install`, `status`, `doctor` and `uninstall` against the system's real
service manager: launchd on macOS (a LaunchAgent in gui/<uid>), the Task Scheduler on
Windows (a task that runs only while the user is logged in). Linux is left out: its
systemd user manager reads units only from the real home (internal/remote tests it with
fakes). The `claude` it starts is a small program built here that writes
what it was given (arguments, folder, PATH) on stderr and waits: the check reads the
service's log for it, and looks that the process is gone after uninstall. One line per check, "ok " or "BAD"; exit code 1 when anything is
BAD. CI runs it on macOS and Windows (ci.yml, windows.yml).

Without Go (the Windows VM), AB_REMOTE_CHECK_PREBUILT names a folder holding both
programs already built for it. On a computer of your own it refuses to start when a service of the space it uses
("remotecheck") is already there, and it takes away only what it made.
"""
import json, os, shutil, subprocess, sys, tempfile, time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WIN = os.name == "nt"
MAC = sys.platform == "darwin"
EXE = ".exe" if WIN else ""
ENC = {"encoding": "utf-8", "errors": "replace"}
SPACE = "remotecheck"
good = bad = 0
sys.stdout.reconfigure(encoding="utf-8", errors="replace")


def report(name, ok, evidence=""):
    global good, bad
    good += bool(ok)
    bad += not ok
    print(("ok " if ok else "BAD"), f"{name:58} {str(evidence)[:300]}".replace("\n", " | "), flush=True)


FAKE = r'''package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "remote-control" {
		return // what else the program asks of claude (its version, the plugin)
	}
	wd, _ := os.Getwd()
	fmt.Fprintf(os.Stderr, "fake claude: args=%s\nfake claude: folder=%s\nfake claude: path=%s\n",
		strings.Join(os.Args[1:], " "), wd, os.Getenv("PATH"))
	time.Sleep(time.Hour)
}
'''


def already_there():
    if MAC:
        return subprocess.run(["launchctl", "print", f"gui/{os.getuid()}/com.augmented-brain.remote.{SPACE}"], capture_output=True).returncode == 0
    if WIN:
        return subprocess.run(["schtasks", "/Query", "/TN", rf"\augmented-brain\remote-{SPACE}"], capture_output=True).returncode == 0
    return False


def alive(pid):
    if WIN:
        out = subprocess.run(["tasklist", "/FI", f"PID eq {pid}", "/NH"], capture_output=True, **ENC).stdout
        return str(pid) in out
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


if not MAC and not WIN:
    print("remote-check: macOS and Windows only; nothing to try here")
    sys.exit(0)
if already_there():
    print(f"remote-check: a service of the space {SPACE} is already here; not touching it")
    sys.exit(2)

TMP = tempfile.mkdtemp(prefix="ab-remote-")
HOME = os.path.join(TMP, "home")
BIN = os.path.join(TMP, "bin")
FAKEBIN = os.path.join(TMP, "fake")
for d in (HOME, BIN, FAKEBIN, os.path.join(TMP, "src")):
    os.makedirs(d)
PREBUILT = os.environ.get("AB_REMOTE_CHECK_PREBUILT")  # a folder with both, for a machine without Go
if PREBUILT:
    shutil.copy(os.path.join(PREBUILT, "augmented-brain" + EXE), BIN)
    shutil.copy(os.path.join(PREBUILT, "claude" + EXE), FAKEBIN)
else:
    subprocess.run(["go", "build", "-o", os.path.join(BIN, "augmented-brain" + EXE), "./cmd/augmented-brain"], cwd=ROOT, check=True)
    with open(os.path.join(TMP, "src", "main.go"), "w") as f:
        f.write(FAKE)
    with open(os.path.join(TMP, "src", "go.mod"), "w") as f:
        f.write("module fakeclaude\n\ngo 1.21\n")
    subprocess.run(["go", "build", "-o", os.path.join(FAKEBIN, "claude" + EXE), "."], cwd=os.path.join(TMP, "src"), check=True)

CFG = os.path.join(HOME, "cfg")
WORK = os.path.join(HOME, "projects")
os.makedirs(WORK)
ENV = {k: v for k, v in os.environ.items() if not k.startswith(("AB_", "CLAUDE"))}
ENV.update(HOME=HOME, USERPROFILE=HOME, AB_CONFIG_DIR=CFG, AB_LANGUAGE="en", AB_NO_BROWSER="1", AB_NO_UPDATE_CHECK="1",
           XDG_CONFIG_HOME=os.path.join(HOME, ".config"), LOCALAPPDATA=os.path.join(HOME, "AppData", "Local"),
           CLAUDE_CONFIG_DIR=os.path.join(HOME, ".claude"), PATH=os.pathsep.join([FAKEBIN, BIN, os.environ["PATH"]]))


def ab(*args, timeout=120):
    p = subprocess.run([os.path.join(BIN, "augmented-brain" + EXE), *args], capture_output=True, text=True, cwd=WORK,
                       env=ENV, timeout=timeout, **ENC)
    return p.returncode, p.stdout + p.stderr


def log_file():
    if MAC:
        return os.path.join(HOME, "Library", "Logs", "augmented-brain", f"augmented-brain-remote-{SPACE}.log")
    return os.path.join(HOME, "AppData", "Local", "augmented-brain", "remote", f"augmented-brain-remote-{SPACE}.log")


def unit_file():
    if MAC:
        return os.path.join(HOME, "Library", "LaunchAgents", f"com.augmented-brain.remote.{SPACE}.plist")
    return os.path.join(HOME, "AppData", "Local", "augmented-brain", "remote", f"augmented-brain-remote-{SPACE}.xml")


def read(path):
    """A file's text: UTF-16 when it starts with its mark (the task of Windows)."""
    try:
        with open(path, "rb") as f:
            data = f.read()
    except OSError:
        return ""
    return data.decode("utf-16" if data[:2] == b"\xff\xfe" else "utf-8", errors="replace")


def fake_pids():
    """The fake claude processes started from this check's folder."""
    if WIN:
        out = subprocess.run(["powershell", "-NoProfile", "-NonInteractive", "-Command",
                              "Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -like '" + FAKEBIN.replace("'", "''") + "*' } | ForEach-Object { $_.ProcessId }"],
                             capture_output=True, **ENC).stdout
    else:
        out = subprocess.run(["pgrep", "-f", os.path.join(FAKEBIN, "claude")], capture_output=True, **ENC).stdout
    return [int(x) for x in out.split() if x.strip().isdigit()]


installed = False
try:
    code, out = ab("init", "--no-plugin", "--language", "en")
    if code == 0:
        code, out = ab("space", "add", SPACE, "--work", WORK, "--yes")
    report("a throwaway configuration", code == 0, out.strip()[-200:])
    with open(os.path.join(CFG, "remote.toml"), "w", encoding="utf-8") as f:
        f.write(f'spaces = ["{SPACE}"]\nname_prefix = "remote-check"\n')
    os.makedirs(os.path.join(HOME, ".claude"), exist_ok=True)
    with open(os.path.join(HOME, ".claude", ".claude.json"), "w", encoding="utf-8") as f:
        json.dump({"projects": {WORK: {"hasTrustDialogAccepted": True}, WORK.replace("\\", "/"): {"hasTrustDialogAccepted": True}}}, f)

    code, out = ab("remote", "install")
    installed = True
    report("remote install", code == 0, out.strip()[-300:])
    report("…names the session", "remote-check-" + SPACE in out, "")
    report("…wrote the service with its marker", "Written by augmented-brain remote install" in read(unit_file()), unit_file())

    deadline, status = time.time() + 120, ""
    while time.time() < deadline:
        code, status = ab("remote", "status")
        if "running" in status and "fake claude: args=" in read(log_file()):
            break
        time.sleep(3)
    report("remote status: running", code == 0 and "running" in status, status.strip())
    log = read(log_file())
    report("claude got its arguments", f"args=remote-control --name remote-check-{SPACE} --permission-mode default" in log, log)
    folder = [l for l in log.splitlines() if l.startswith("fake claude: folder=")]
    report("…in the space's folder", bool(folder) and os.path.realpath(folder[-1].split("=", 1)[1]).lower() == os.path.realpath(WORK).lower(), folder)
    path = [l for l in log.splitlines() if l.startswith("fake claude: path=")]
    first = path[-1].split("=", 1)[1].split(os.pathsep)[:2] if path else []
    report("…with claude's and the program's folders first in PATH",
           [os.path.normcase(p) for p in first] == [os.path.normcase(FAKEBIN), os.path.normcase(BIN)], first)
    code, out = ab("doctor")
    line = [l for l in out.splitlines() if "remote access" in l]
    report("doctor has a line for it", bool(line) and "running" in line[0], line)

    before = fake_pids()
    report("a claude is running", bool(before), before)
    code, out = ab("remote", "uninstall")
    installed = code != 0
    report("remote uninstall", code == 0 and "Stopped and removed" in out, out.strip()[-300:])
    report("…removed the service", not os.path.exists(unit_file()), unit_file())
    time.sleep(3)
    left = [p for p in before if alive(p)]
    report("…and the claude it ran is gone", not left, left)
    report("…and the service manager forgot it", not already_there(), "")
    code, out = ab("remote", "status")
    report("remote status after uninstall: not set up", code == 0 and "not set up" in out, out.strip())
finally:
    if installed:
        ab("remote", "uninstall")
    for p in fake_pids():
        try:
            if WIN:
                subprocess.run(["taskkill", "/F", "/T", "/PID", str(p)], capture_output=True)
            else:
                os.kill(p, 9)
        except OSError:
            pass
    shutil.rmtree(TMP, ignore_errors=True)

print(f"\nremote-check: {good} ok, {bad} BAD")
sys.exit(1 if bad else 0)
