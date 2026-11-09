#!/usr/bin/env python3
"""mac-check.py: what only a real Mac can tell, without a person at the screen.

    python3 -I scripts/mac-check.py

macOS only. Builds the program from this checkout and, in a throwaway home with made-up
spaces, checks three things a newcomer on a Mac would meet:

- Gatekeeper: the program as the installer leaves it (fetched with curl, so with no
  quarantine mark) runs; and what macOS does with the same program marked as downloaded
  with a browser, which is what happens to someone who gets it from the Releases page.
  That second part is reported, not judged: the program is not signed by Apple (a known
  limitation), and the guides say how to open it anyway.
- The keychain, locked: the panel's Secrets page asks the keychain about each secret's
  name; with the keychain locked it must still answer in a few seconds, and never call a
  secret "not on this computer" because it could not look. Only with
  AB_MAC_CHECK_KEYCHAIN=1 (CI sets it): it adds a throwaway keychain to the user's
  keychain list for the run, which you may not want on your own Mac.
- Safari itself, through safaridriver (CI enables it first; on your Mac, Safari's
  Develop menu → Allow Remote Automation): every page of the panel loads, with its
  stylesheet and its title, and the inbox's capture works.

The network missing is already in faults-check.py, which CI runs on macOS too. One line per
check, "ok " or "BAD"; lines that start with "--" only report. Exit code 1 when anything is
BAD.
"""
import json, os, re, shutil, socket, subprocess, sys, tempfile, time, urllib.error, urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if sys.platform != "darwin":
    print("mac-check.py runs on macOS only")
    raise SystemExit(2)
good = bad = 0
PAGES = ["today", "tasks", "reminders", "inbox", "areas", "knowledge", "rules", "people", "search?q=bread",
         "review", "secrets", "profile", "space", "sessions", "repos", "guard", "versions", "help"]


def report(name, ok, evidence=""):
    global good, bad
    good += bool(ok)
    bad += not ok
    print(("ok " if ok else "BAD"), f"{name:60} {str(evidence)[:150]}".replace("\n", " "), flush=True)


def info(name, evidence):
    print("-- ", f"{name:60} {str(evidence)[:200]}".replace("\n", " "), flush=True)


TMP = tempfile.mkdtemp(prefix="ab-mac-")
HOME = os.path.join(TMP, "home")
os.makedirs(HOME)
AB = os.path.join(TMP, "augmented-brain")
subprocess.run(["go", "build", "-o", AB, "./cmd/augmented-brain"], cwd=ROOT, check=True)
CFG = os.path.join(HOME, "cfg")
ENV = dict(os.environ, HOME=HOME, AB_CONFIG_DIR=CFG, AB_LANGUAGE="en", AB_NO_BROWSER="1",
           AB_NO_UPDATE_CHECK="1", CLAUDE_CONFIG_DIR=os.path.join(HOME, ".claude"))
# The keychain is the user's, not the throwaway home's: security reads the real one.
KEYCHAIN_ENV = dict(ENV, HOME=os.path.expanduser("~"))


def run(cmd, env=None, timeout=30, **kw):
    t = time.monotonic()
    try:
        r = subprocess.run(cmd, env=env, capture_output=True, text=True, timeout=timeout, **kw)
        return r.returncode, (r.stdout + r.stderr).strip(), time.monotonic() - t
    except subprocess.TimeoutExpired:
        return None, f"no answer in {timeout} s", time.monotonic() - t


def tools(cwd, calls):
    """MCP tool calls, one message at a time, as Claude Code makes them."""
    p = subprocess.Popen([AB, "mcp"], cwd=cwd, env=ENV, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)

    def send(msg, answer=True):
        p.stdin.write(json.dumps(msg) + "\n")
        p.stdin.flush()
        return json.loads(p.stdout.readline()) if answer else None

    send({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "mac", "version": "1"}}})
    send({"jsonrpc": "2.0", "method": "notifications/initialized"}, answer=False)
    for i, (name, a) in enumerate(calls, 1):
        send({"jsonrpc": "2.0", "id": i, "method": "tools/call", "params": {"name": name, "arguments": a}})
    p.stdin.close()
    p.wait(timeout=30)


def gatekeeper():
    print("##### Gatekeeper")
    code, out, _ = run([AB, "version"], env=ENV)
    report("the program runs as the installer leaves it", code == 0, out)
    _, attrs, _ = run(["xattr", AB])
    report("a program built or fetched with curl has no quarantine mark", "com.apple.quarantine" not in attrs, attrs or "no attributes")
    _, sig, _ = run(["codesign", "-dv", AB])
    info("its signature", re.sub(r"\s+", " ", sig))
    # The mark a browser leaves on a download: flags;time;agent;id.
    q = os.path.join(TMP, "downloaded", "augmented-brain")
    os.makedirs(os.path.dirname(q))
    shutil.copy2(AB, q)
    mark = "0083;%08x;Safari;" % int(time.time())
    run(["xattr", "-w", "com.apple.quarantine", mark, q])
    _, verdict, _ = run(["spctl", "--assess", "--type", "execute", "-vv", q])
    info("what Gatekeeper says of it, downloaded with a browser", verdict)
    code, out, took = run([q, "version"], env=ENV, timeout=20)
    if code == 0:
        info("run from Terminal after a browser download", f"it runs ({out})")
    else:
        info("run from Terminal after a browser download", f"stopped: exit {code}, {took:.1f} s, {out}")
    report("a browser download is told apart from the installer's",
           True, "see the two lines above; the guides say how to open it anyway")


def keychain(base):
    print("##### the keychain, locked")
    if os.environ.get("AB_MAC_CHECK_KEYCHAIN") != "1":
        info("skipped", "set AB_MAC_CHECK_KEYCHAIN=1 to add a throwaway keychain for the run (CI does)")
        return
    kc = os.path.join(TMP, "ab-check.keychain-db")
    pw = "ab-check-%d" % os.getpid()
    _, before, _ = run(["security", "list-keychains", "-d", "user"], env=KEYCHAIN_ENV)
    old = [k.strip().strip('"') for k in before.splitlines() if k.strip()]
    try:
        run(["security", "create-keychain", "-p", pw, kc], env=KEYCHAIN_ENV)
        run(["security", "list-keychains", "-d", "user", "-s", kc, *old], env=KEYCHAIN_ENV)
        # A made-up secret, readable by every program without asking (-A): only its
        # presence is checked, never its value.
        code, out, _ = run(["security", "add-generic-password", "-s", "ab-check", "-a", "probe", "-w", "made-up", "-A", kc], env=KEYCHAIN_ENV)
        report("a made-up secret goes into a throwaway keychain", code == 0, out)
        page = f"{base}/secrets?s=work&check=1"

        def secrets_page():
            t = time.monotonic()
            try:
                with urllib.request.urlopen(page, timeout=30) as r:
                    return r.read().decode(), time.monotonic() - t
            except (urllib.error.URLError, TimeoutError) as e:
                return str(e), time.monotonic() - t

        html, took = secrets_page()
        report("unlocked: the Secrets page finds it", "in the keychain" in html, f"{took:.1f} s")
        run(["security", "lock-keychain", kc], env=KEYCHAIN_ENV)
        code, out, took = run(["security", "find-generic-password", "-s", "ab-check", "-a", "probe", kc], env=KEYCHAIN_ENV, timeout=15)
        info("locked: what security says, asked as the panel asks", f"exit {code}, {took:.1f} s: {out[:120]}")
        html, took = secrets_page()
        state = ("in the keychain" if "in the keychain" in html else "not on this computer" if "not on this computer" in html
                 else "cannot tell here" if "cannot tell here" in html else "no answer")
        report("locked: the Secrets page answers in a few seconds", took < 10, f"{took:.1f} s, says {state!r}")
        report("locked: a secret is never called missing because it could not look", state != "not on this computer", state)
    finally:
        run(["security", "list-keychains", "-d", "user", "-s", *old], env=KEYCHAIN_ENV)
        run(["security", "delete-keychain", kc], env=KEYCHAIN_ENV)


class Safari:
    """The few WebDriver calls this needs, against safaridriver."""

    def __init__(self):
        s = socket.socket()
        s.bind(("127.0.0.1", 0))
        self.port = s.getsockname()[1]
        s.close()
        self.proc = subprocess.Popen(["safaridriver", "-p", str(self.port)], stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        for _ in range(50):
            try:
                self.call("GET", "/status")
                break
            except OSError:
                time.sleep(0.2)
        self.sid = self.call("POST", "/session", {"capabilities": {"alwaysMatch": {"browserName": "safari"}}})["value"]["sessionId"]

    def call(self, method, path, body=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(f"http://127.0.0.1:{self.port}{path}", data=data, method=method,
                                     headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=60) as r:
                return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            raise RuntimeError(e.read().decode()[:300])

    def go(self, url):
        self.call("POST", f"/session/{self.sid}/url", {"url": url})

    def js(self, script, *args):
        return self.call("POST", f"/session/{self.sid}/execute/sync", {"script": script, "args": list(args)})["value"]

    def close(self):
        try:
            self.call("DELETE", f"/session/{self.sid}")
        finally:
            self.proc.terminate()


def safari(base):
    print("##### Safari")
    try:
        sf = Safari()
    except Exception as e:  # not enabled, or no Safari: say how
        report("Safari answers through safaridriver", False,
               f"{e}; enable it with: safaridriver --enable (as an admin), or Develop → Allow Remote Automation")
        return
    report("Safari answers through safaridriver", True, sf.js("return navigator.userAgent")[:120])
    try:
        for page in PAGES:
            url = f"{base}/{page}{'&' if '?' in page else '?'}s=work"
            sf.go(url)
            state = sf.js("""return {title: document.title, ready: document.readyState,
                sheets: [...document.styleSheets].filter(s => { try { return s.cssRules.length > 0 } catch (e) { return false } }).length,
                font: getComputedStyle(document.body).fontFamily,
                broken: [...document.images].filter(i => i.complete && i.naturalWidth === 0).map(i => i.src),
                wide: document.documentElement.scrollWidth > window.innerWidth + 1}""")
            ok = (state["ready"] == "complete" and "augmented brain" in state["title"] and state["sheets"] > 0
                  and not state["broken"] and not state["wide"])
            report(f"Safari: {page.split('?')[0]} loads with its stylesheet", ok,
                   f"{state['title']!r}, {state['sheets']} stylesheet(s), broken {state['broken']}, too wide {state['wide']}")
        # A capture from the inbox, as a person sends it.
        sf.go(f"{base}/inbox?s=work")
        sent = sf.js("""const f = [...document.forms].find(f => f.querySelector('textarea, input[name=text]'));
            if (!f) return 'no capture form';
            const box = f.querySelector('textarea, input[name=text]'); box.value = 'water the plants';
            f.submit(); return 'sent'""")
        time.sleep(1.5)
        body = sf.js("return document.body.innerText")
        report("Safari: a capture from the inbox arrives", sent == "sent" and "water the plants" in body, sent)
    finally:
        sf.close()


try:
    run([AB, "init", "--no-plugin"], env=ENV, cwd=HOME)
    work = os.path.join(HOME, "augmented-brain", "work")
    tools(work, [("task_new", {"title": "Bake bread", "summary": "two loaves for Sunday"}),
                 ("capture", {"text": "buy flour"}),
                 ("secret_name_save", {"service": "ab-check", "name": "probe", "what": "a made-up secret for the check"})])
    gatekeeper()
    code, out, _ = run([AB, "panel", "--no-browser"], env=ENV, cwd=work)
    m = re.search(r"https?://127\.0\.0\.1:\d+/s/[0-9a-f]+", out)
    report("the panel starts", bool(m), out if not m else "")
    if m:
        base = m.group(0)
        keychain(base)
        safari(base)
finally:
    run([AB, "panel", "stop"], env=ENV)
    shutil.rmtree(TMP, ignore_errors=True)

print(f"\n{good} ok, {bad} BAD")
raise SystemExit(1 if bad else 0)
