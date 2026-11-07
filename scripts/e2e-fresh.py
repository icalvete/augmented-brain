#!/usr/bin/env python3
"""e2e-fresh.py: several real Claude Code sessions on one space, end to end.

    scripts/e2e-fresh.py

What it proves, with a real, isolated Claude Code and the plugin of this checkout:

  1. A session that read the index at its start hears, before its next answer, that
     another session closed a task (the case of 7-nov), and answers from the news.
  2. Its own writes are never reported back to it.
  3. A journal entry of another session in a task this session worked on is news; one
     in a task it never touched is not.
  4. After /clear (the same Claude process, a new session id) the news still arrive.
  5. Closing a task with an open watch date says it will keep coming up.
  6. A capture sent from another space comes up at the start of a session there, and
     capture_done takes it out.

"Another session" is the program's own MCP server run directly (as compat-check does):
it writes through the same tools, outside the Claude process under test.

Needs the Claude Code configuration of scripts/e2e-claude.sh, logged in once by hand:
  CLAUDE_CONFIG_DIR=~/.config/augmented-brain-dev/claude claude    (log in, then /exit)
Override with AB_E2E_CLAUDE_CONFIG; pick a model with AB_E2E_MODEL (default: sonnet).
AB_E2E_KEEP=1 keeps the temporary folder. It spends usage of that account: run it
before a release or after touching what it covers, not on every change. The real home,
configuration and Claude Code are never touched.
"""

import json
import os
import shutil
import subprocess
import sys
import tempfile
import threading
import time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CLAUDE_DIR = os.environ.get("AB_E2E_CLAUDE_CONFIG", os.path.expanduser("~/.config/augmented-brain-dev/claude"))
MODEL = os.environ.get("AB_E2E_MODEL", "sonnet")
NOTICE = "changed outside it"

passed, failed = 0, 0


def check(name, ok, detail=""):
    global passed, failed
    if ok:
        passed += 1
        print(f"  OK    {name}")
    else:
        failed += 1
        print(f"  FAIL  {name}" + (f"\n        {detail[:600]}" if detail else ""))


def main():
    if not shutil.which("claude"):
        sys.exit("claude is not on the PATH")
    if not os.path.isdir(CLAUDE_DIR):
        sys.exit(f"no Claude configuration at {CLAUDE_DIR}: log in there first (see the top of this script)")
    claude = shutil.which("claude")
    tmp = tempfile.mkdtemp(prefix="ab-e2e-fresh-")
    try:
        run(tmp, claude)
    finally:
        if os.environ.get("AB_E2E_KEEP") == "1":
            print("kept:", tmp)
        else:
            shutil.rmtree(tmp, ignore_errors=True)
    print(f"\n{passed} passed, {failed} failed")
    sys.exit(1 if failed else 0)


def run(tmp, claude_bin):
    bin_dir = os.path.join(tmp, "bin")
    home = os.path.join(tmp, "home")
    lab, other = os.path.join(home, "lab"), os.path.join(home, "other")
    for d in (bin_dir, lab, other):
        os.makedirs(d)
    print("building augmented-brain")
    subprocess.run(["go", "build", "-o", os.path.join(bin_dir, "augmented-brain"), "./cmd/augmented-brain"], cwd=ROOT, check=True)
    os.symlink(claude_bin, os.path.join(bin_dir, "claude"))
    env = dict(os.environ)
    env.pop("XDG_CONFIG_HOME", None)
    env.pop("CLAUDE_PID", None)
    env.pop("CLAUDECODE", None)
    env.update(HOME=home, AB_CONFIG_DIR=os.path.join(home, ".config", "augmented-brain"),
               CLAUDE_CONFIG_DIR=CLAUDE_DIR, PATH=bin_dir + ":/usr/bin:/bin", AB_NO_UPDATE_CHECK="1",
               AB_NO_AUTO_BACKUP="1", AB_LANGUAGE="en")
    ab = os.path.join(bin_dir, "augmented-brain")

    def sh(*args, cwd=None):
        return subprocess.run(list(args), cwd=cwd or home, env=env, capture_output=True, text=True, check=True).stdout

    print("setting up two spaces")
    sh(ab, "init", "--yes", "--no-plugin", "--language", "en", "--space", "lab", "--work", lab, "--memory", os.path.join(home, "brain-lab"))
    sh(ab, "space", "add", "--yes", "--space", "other", "--work", other, "--memory", os.path.join(home, "brain-other"))
    memory = os.path.join(home, "brain-lab")

    other_session = Outside(ab, lab, env)
    other_session.call("profile_save", {"name": "Lucia", "about": "- Tests augmented brain.", "work_style": "- Short answers."})
    other_session.call("task_new", {"title": "Logo"})
    other_session.call("task_new", {"title": "Report"})

    trust(os.path.join(CLAUDE_DIR, ".claude.json"), [lab, other])

    print("1. another session closes a task this one read at its start")
    a = Session(env, lab, os.path.join(ROOT, "plugin"), MODEL)
    try:
        out = a.ask("Without using any tool: what is the status of task task-logo, from what you were given when "
                    "this session started? Reply with the status word alone.")
        check("starts knowing the task as open", "definition" in out.lower(), out)
        other_session.call("task_close", {"id": "task-logo", "result": "done by the other session"})
        out = a.ask("Is task-logo still open, or is it closed now? Reply with OPEN or CLOSED alone.")
        check("hears the close before answering", "CLOSED" in out.upper() and "OPEN" not in out.upper(), out)
        check("the news came through augmented brain", a.transcript_count(NOTICE) >= 1)

        print("2. its own writes are not news to it")
        before = a.transcript_count(NOTICE)
        out = a.ask("Add the journal entry 'checked the figures' to task task-report, then reply DONE.")
        check("wrote in the journal", "checked the figures" in read(memory, "tasks/task-report.md"), out)
        a.ask("Reply OK.")
        check("no news of its own write", a.transcript_count(NOTICE) == before)

        print("3. another session writes in the task this one worked on")
        other_session.call("task_journal", {"id": "task-report", "entry": "numbers changed by the other session"})
        out = a.ask("Did anything change in task-report since you last looked? Reply YES or NO, then one line.")
        check("hears it", out.upper().startswith("YES") or "YES" in out.upper().split("\n")[0], out)
        before = a.transcript_count(NOTICE)
        other_session.call("task_new", {"title": "Side"})
        a.ask("Reply OK.")
        after_new = a.transcript_count(NOTICE)
        check("a new task is news", after_new > before)
        other_session.call("task_journal", {"id": "task-side", "entry": "quiet entry"})
        a.ask("Reply OK.")
        check("a journal entry in a task it never touched is not", a.transcript_count(NOTICE) == after_new)

        print("4. after /clear the news still arrive")
        a.ask("/clear", expect_result=False)
        other_session.call("task_update", {"id": "task-report", "status": "parked", "next": "wait for the auditors"})
        out = a.ask("What is the next step of task task-report? Reply with it alone.")
        check("hears the change after /clear", "auditor" in out.lower(), out)

        print("5. closing a task that keeps an open watch date")
        other_session.call("task_update", {"id": "task-report", "watch_date": "2030-01-08", "watch_what": "auditors replied"})
        a.ask("Reply OK.")  # takes the news of the watch date
        out = a.ask("Close task task-report with result 'handed over'. Then reply with what the tool said about "
                    "watch dates, in one line, or NONE.")
        check("closed", "status: closed" in read(memory, "tasks/task-report.md"))
        check("it says the watch date keeps coming up", "2030-01-08" in out or "watch" in out.lower(), out)
    finally:
        a.close()

    print("6. a capture sent from another space")
    sender = Outside(ab, other, env)
    sender.call("capture", {"text": "Purge php8.0 on the old server", "space": "lab"})
    b = Session(env, lab, os.path.join(ROOT, "plugin"), MODEL)
    try:
        out = b.ask("Without using any tool: was anything sent to this space from another space? "
                    "Reply with its text alone, or NONE.")
        check("comes up at the start", "php8.0" in out, out)
        b.ask("Mark that capture done with capture_done, then reply DONE.")
        check("capture_done took it out", "Purge php8.0" not in read(memory, "inbox.md"))
    finally:
        b.close()
    c = Session(env, lab, os.path.join(ROOT, "plugin"), MODEL)
    try:
        out = c.ask("Without using any tool: was anything sent to this space from another space? "
                    "Reply with its text alone, or NONE.")
        check("and no longer comes up", "php8.0" not in out, out)
    finally:
        c.close()


def read(memory, rel):
    try:
        with open(os.path.join(memory, rel)) as f:
            return f.read()
    except OSError:
        return ""


def trust(claude_json, folders):
    """Accepts Claude Code's trust dialog for the test folders, in the test configuration."""
    try:
        with open(claude_json) as f:
            data = json.load(f)
    except OSError:
        data = {}
    for folder in folders:
        data.setdefault("projects", {}).setdefault(folder, {})["hasTrustDialogAccepted"] = True
    with open(claude_json, "w") as f:
        json.dump(data, f, indent=2)


class Outside:
    """Another session: the program's MCP server, run directly in a folder of a space."""

    def __init__(self, ab, folder, env):
        e = {k: v for k, v in env.items() if k not in ("CLAUDECODE", "CLAUDE_PID")}
        self.p = subprocess.Popen([ab, "mcp"], cwd=folder, env=e, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        self.n = 0
        self._send({"jsonrpc": "2.0", "id": 0, "method": "initialize",
                    "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "e2e", "version": "1"}}})
        self._send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    def _send(self, msg):
        self.p.stdin.write(json.dumps(msg) + "\n")
        self.p.stdin.flush()
        return json.loads(self.p.stdout.readline()) if "id" in msg else None

    def call(self, tool, args):
        self.n += 1
        r = self._send({"jsonrpc": "2.0", "id": self.n, "method": "tools/call", "params": {"name": tool, "arguments": args}})
        res = r.get("result") or {}
        if not res or res.get("isError"):
            raise SystemExit(f"the other session's {tool} failed: {json.dumps(r)[:400]}")
        return res["content"][0]["text"]


class Session:
    """A real Claude Code session kept open, fed one message at a time (stream-json)."""

    def __init__(self, env, folder, plugin, model):
        args = ["claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
                "--plugin-dir", plugin, "--model", model,
                "--allowedTools", "mcp__plugin_brain_augmented-brain"]
        self.p = subprocess.Popen(args, cwd=folder, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=subprocess.DEVNULL, text=True)
        self.lines, self.results, self.transcripts = [], [], set()
        self.cv = threading.Condition()
        threading.Thread(target=self._read, daemon=True).start()

    def _read(self):
        for line in self.p.stdout:
            try:
                m = json.loads(line)
            except ValueError:
                continue
            with self.cv:
                self.lines.append(m)
                if m.get("type") == "system" and m.get("session_id"):
                    self.sid = m["session_id"]
                if m.get("type") == "result":
                    self.results.append(m.get("result") or "")
                self.cv.notify_all()

    def ask(self, text, expect_result=True, timeout=300):
        with self.cv:
            n = len(self.results)
        msg = {"type": "user", "message": {"role": "user", "content": [{"type": "text", "text": text}]}}
        self.p.stdin.write(json.dumps(msg) + "\n")
        self.p.stdin.flush()
        if not expect_result:
            time.sleep(8)
            return ""
        with self.cv:
            if not self.cv.wait_for(lambda: len(self.results) > n, timeout):
                return "(no answer within %ds)" % timeout
            return self.results[-1]

    def transcript_count(self, text):
        """How many times text reached the model: hook context and tool answers in the
        session's transcripts (one per session id; /clear starts another)."""
        count = 0
        for m in list(self.lines):
            if m.get("type") == "system" and m.get("session_id"):
                self.transcripts.add(m["session_id"])
        root = os.path.join(CLAUDE_DIR, "projects")
        for sid in self.transcripts:
            for dirpath, _, files in os.walk(root):
                if sid + ".jsonl" in files:
                    with open(os.path.join(dirpath, sid + ".jsonl")) as f:
                        for line in f:
                            count += line.count(text)
        return count

    def close(self):
        try:
            self.p.stdin.close()
            self.p.wait(timeout=30)
        except Exception:
            self.p.kill()


if __name__ == "__main__":
    main()
