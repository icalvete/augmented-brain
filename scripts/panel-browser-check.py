#!/usr/bin/env python3
"""panel-browser-check.py: the panel in a real browser engine, page by page.

    python3 -I scripts/panel-browser-check.py [chromium|webkit|firefox …]

Builds the program from this checkout, makes a throwaway home with made-up spaces and a
few notes, starts the panel and opens every page in each browser engine named (Chromium
by default; WebKit is Safari's, the iPhone's), on a desktop window and on a phone-sized
one. For each page: it loads, the browser's console has no error, nothing on the page
fails to load, and on the phone nothing is wider than the screen. Then a few things a
person does: send a capture from the inbox, open a task, and on the Guard page see that
adding a repository where the assistant commits and pushes without asking needs its box
ticked, and that a change made outside the page is shown. And the menu: on a 1366×768
window, at the foot of a long page, its last entry and "Stop the panel" can be reached,
and so can the last entry of the phone's open menu. One line per check, "ok " or
"BAD"; exit code 1 when anything is BAD.

Needs Playwright for Python and its browsers (pip install playwright; python -m
playwright install chromium webkit). CI runs it on Linux with Chromium and WebKit
(`panel` in ci.yml); Firefox, once before a release.
"""
import json, os, re, shutil, subprocess, sys, tempfile, urllib.error, urllib.request

from playwright.sync_api import sync_playwright

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WIN = os.name == "nt"
EXE = ".exe" if WIN else ""
ENGINES = [a for a in sys.argv[1:] if not a.startswith("-")] or ["chromium"]
PAGES = ["today", "tasks", "reminders", "inbox", "areas", "knowledge", "rules", "people", "search?q=bread",
         "review", "secrets", "backup", "profile", "space", "sessions", "repos", "guard", "versions", "help"]
# The lowest contrast of any text on the page (WCAG's ratio): for each element with a
# text of its own, its color against the first background behind it, with opacity.
# Answers [ratio, "what"] for the worst one.
CONTRAST = """() => {
  const lum = c => { const v = c.match(/[\\d.]+/g).map(Number); return v.slice(0, 3).map(x => x / 255)
    .map(x => x <= .03928 ? x / 12.92 : ((x + .055) / 1.055) ** 2.4).reduce((s, x, i) => s + x * [.2126, .7152, .0722][i], 0) };
  const alpha = c => { const v = c.match(/[\\d.]+/g).map(Number); return v.length > 3 ? v[3] : 1 };
  const behind = el => { for (let e = el; e; e = e.parentElement) { const b = getComputedStyle(e).backgroundColor; if (alpha(b) > 0) return b } return 'rgb(255,255,255)' };
  let worst = [99, ''];
  for (const el of document.querySelectorAll('body *')) {
    if (![...el.childNodes].some(n => n.nodeType === 3 && n.textContent.trim())) continue;
    const st = getComputedStyle(el), r = el.getBoundingClientRect();
    if (st.visibility === 'hidden' || st.display === 'none' || !r.width || !r.height || el.closest('[hidden], option')) continue;
    let fg = lum(st.color), bg = lum(behind(el)), op = 1;
    for (let e = el; e; e = e.parentElement) op *= Number(getComputedStyle(e).opacity);
    fg = fg * op + bg * (1 - op);
    const ratio = (Math.max(fg, bg) + .05) / (Math.min(fg, bg) + .05);
    if (ratio < worst[0]) worst = [Math.round(ratio * 100) / 100, el.tagName.toLowerCase() + (el.className ? '.' + el.className : '') + ' "' + el.textContent.trim().slice(0, 30) + '"'];
  }
  return worst }"""
# Each blocked resource says what and where, so a report names the culprit.
WATCH = """document.addEventListener('securitypolicyviolation', e =>
    console.error('CSP ' + e.violatedDirective + ' blocked ' + (e.blockedURI || 'inline') +
      ' at ' + (e.sourceFile || '?') + ':' + e.lineNumber + ' sample=' + (e.sample || '')))"""
good = bad = 0
sys.stdout.reconfigure(encoding="utf-8", errors="replace")


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def spend_ticket(text):
    """The panel's pages, from what `panel` printed: it prints a ticket of one use
    (/t/<ticket>), whose visit answers 303 to the pages under /s/<session>. None when
    there is no ticket or it does not lead there."""
    m = re.search(r"(https?://127\.0\.0\.1:\d+)/t/[0-9a-f]{32}", text)
    if not m:
        return None
    try:
        urllib.request.build_opener(_NoRedirect).open(m.group(0), timeout=10)
        return None
    except urllib.error.HTTPError as e:
        s = re.match(r"/s/[0-9a-f]{32}", e.headers.get("Location", "")) if e.code == 303 else None
        return m.group(1) + s.group(0) if s else None


def report(name, ok, evidence=""):
    global good, bad
    good += bool(ok)
    bad += not ok
    print(("ok " if ok else "BAD"), f"{name:62} {str(evidence)[:150]}".replace("\n", " "), flush=True)


TMP = tempfile.mkdtemp(prefix="ab-browser-")
HOME = os.path.join(TMP, "home")
os.makedirs(HOME)
AB = os.path.join(TMP, "augmented-brain" + EXE)
subprocess.run(["go", "build", "-o", AB, "./cmd/augmented-brain"], cwd=ROOT, check=True)
# A configuration folder with a long path: the Profile and Health pages show it, and on a
# phone it must break instead of making the page wider than the screen (no hyphens, which
# would give the browser a place to break on its own).
CFG = os.path.join(HOME, "SettingsOfAPersonWhoNamesFoldersInFullAndNeverUsesSpaces", "augmented-brain")
ENV = dict(os.environ, HOME=HOME, USERPROFILE=HOME, AB_CONFIG_DIR=CFG, AB_LANGUAGE="en", AB_NO_BROWSER="1",
           AB_NO_UPDATE_CHECK="1", CLAUDE_CONFIG_DIR=os.path.join(HOME, ".claude"))


def ab(*args, cwd=None, inp=None):
    return subprocess.run([AB, *args], cwd=cwd or HOME, env=ENV, input=inp, capture_output=True, text=True, timeout=60)


def tools(cwd, calls):
    """MCP tool calls, one message at a time, as Claude Code makes them."""
    p = subprocess.Popen([AB, "mcp"], cwd=cwd, env=ENV, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)

    def send(msg, answer=True):
        p.stdin.write(json.dumps(msg) + "\n")
        p.stdin.flush()
        return json.loads(p.stdout.readline()) if answer else None

    send({"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "browser", "version": "1"}}})
    send({"jsonrpc": "2.0", "method": "notifications/initialized"}, answer=False)
    for i, (name, a) in enumerate(calls, 1):
        send({"jsonrpc": "2.0", "id": i, "method": "tools/call", "params": {"name": name, "arguments": a}})
    p.stdin.close()
    p.wait(timeout=30)


try:
    ab("init", "--no-plugin")
    work = os.path.join(HOME, "augmented-brain", "work")
    tools(work, [("task_new", {"title": "Bake bread", "summary": "two loaves for Sunday"}),
                 ("capture", {"text": "buy flour"}),
                 ("knowledge_new", {"type": "lesson", "title": "Let the dough rest", "summary": "an hour at least", "body": "It rises better."}),
                 ("person_save", {"name": "Lucía", "summary": "runs the bakery"}),
                 # A profile with an assistant: Profile then shows how it signs.
                 ("profile_save", {"name": "María José Fernández-Villaverde", "about": "- bakes on Sundays",
                                   "assistant_name": "Bartolomé", "assistant_personality": "calm and to the point"})]
          # A long page: the task's journal, for the menu checks below.
          + [("task_journal", {"id": "task-bake-bread", "entry": f"Day {i}: fed the starter, weighed the flour, wrote down how it rose."}) for i in range(60)])
    r = ab("panel", "--no-browser", cwd=work)
    base = spend_ticket(r.stdout + r.stderr)
    if not base:
        report("the panel starts", False, (r.stdout + r.stderr).strip())
        raise SystemExit(1)
    report("the panel starts", True, "")
    # The panel's own security policy, as it sends it, for the calibration below.
    with urllib.request.urlopen(f"{base}/today?s=work", timeout=10) as r:
        CSP = r.headers.get("Content-Security-Policy", "")
    report("the panel sends a security policy", "style-src 'self'" in CSP, CSP[:80])

    with sync_playwright() as pw:
        for engine in ENGINES:
            browser = getattr(pw, engine).launch()
            # Calibration: a bare <select> under the panel's own policy. WebKit on Linux
            # reports a style-src-attr violation for it, from its own controls, with no
            # source; when this engine does, that one report (and only it) is not the
            # panel's, and does not count against a page.
            calctx = browser.new_context()
            calctx.add_init_script(WATCH)
            cal = calctx.new_page()
            noise = []
            cal.on("console", lambda msg: noise.append(msg.text) if msg.type == "error" else None)
            cal.route("http://calibration.invalid/", lambda route: route.fulfill(status=200, headers={"Content-Type": "text/html", "Content-Security-Policy": CSP},
                      body="<!doctype html><select><option>a</option></select>"))
            cal.goto("http://calibration.invalid/")
            cal.wait_for_timeout(200)
            calctx.close()
            same = lambda e: re.sub(r":\d+", ":N", e)  # the same report on another line
            known = {same(e) for e in noise if "style-src" in e}
            if known:
                print("    known:", sorted(known))
            if known:
                print(f"    ({engine} reports its own <select> under the panel's policy: {len(known)} known line(s), not counted)")
            for size, viewport in (("desktop", {"width": 1280, "height": 800}), ("phone", {"width": 375, "height": 740})):
                ctx = browser.new_context(viewport=viewport)
                ctx.add_init_script(WATCH)
                page = ctx.new_page()
                errors, failed = [], []
                page.on("console", lambda msg: errors.append(msg.text) if msg.type == "error" else None)
                page.on("pageerror", lambda exc: errors.append(str(exc)))
                page.on("requestfailed", lambda req: failed.append(req.url))
                for path in PAGES:
                    del errors[:], failed[:]
                    resp = page.goto(f"{base}/{path}{'&' if '?' in path else '?'}s=work")
                    page.wait_for_load_state("networkidle")
                    errors[:] = [e for e in errors if same(e) not in known]
                    ok = resp is not None and resp.status == 200 and not errors and not failed
                    report(f"{engine} {size}: {path.split('?')[0]} loads clean", ok, [e for e in errors if e.startswith('CSP')][:1] + (errors + failed)[:1] or (resp.status if resp else "no answer"))
                    if size == "desktop":
                        low, what = page.evaluate(CONTRAST)
                        report(f"{engine}: {path.split('?')[0]} text has 4.5:1 contrast or more", low >= 4.5, f"{low}:1 {what}")
                    if size == "phone":
                        wide = page.evaluate("document.documentElement.scrollWidth - document.documentElement.clientWidth")
                        report(f"{engine} phone: {path.split('?')[0]} fits the screen", wide <= 1, f"{wide}px wider")
                ctx.close()

            # The menu can be reached on a low window, at the foot of a long page: the
            # sidebar scrolls on its own. Each element is scrolled into view the way a
            # person would get there, then must be on screen and the one under its middle.
            reach = """el => { el.scrollIntoView({block: 'nearest'}); const r = el.getBoundingClientRect();
                const top = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
                const ok = r.bottom <= innerHeight + 1 && r.top >= -1 && !!top && (top === el || el.contains(top));
                return [ok, `bottom ${Math.round(r.bottom)} of ${innerHeight}, under it: ${top ? top.tagName + '.' + top.className : 'nothing'}`] }"""
            ctx = browser.new_context(viewport={"width": 1366, "height": 768})
            page = ctx.new_page()
            page.goto(f"{base}/task?id=task-bake-bread&s=work")
            page.evaluate("window.scrollTo(0, document.documentElement.scrollHeight)")
            long = page.evaluate("document.documentElement.scrollHeight > 2 * innerHeight")
            report(f"{engine} 1366x768: the task's page is long", long, "")
            for what, sel in (("the menu's last entry", ".sections a >> nth=-1"), ("Stop the panel", ".side footer button")):
                report(f"{engine} 1366x768: {what} can be reached", *page.locator(sel).evaluate(reach))
            ctx.close()
            ctx = browser.new_context(viewport={"width": 375, "height": 600})
            page = ctx.new_page()
            page.goto(f"{base}/profile?s=work")
            label = page.locator(".menu summary").inner_text().strip()
            report(f"{engine} phone: the menu button names Profile there", label == "Profile", label)
            page.click(".menu summary")
            report(f"{engine} phone: the open menu's last entry can be reached",
                   *page.locator(".menu-list nav a >> nth=-1").evaluate(reach))
            ctx.close()

            # What a person does.
            ctx = browser.new_context()
            page = ctx.new_page()
            page.goto(f"{base}/inbox?s=work")
            page.fill("textarea[name=text], input[name=text]", "call the miller")
            page.click("form[action$='/inbox/capture'] button[type=submit]")
            page.wait_for_load_state("networkidle")
            report(f"{engine}: a capture from the inbox page", "call the miller" in page.content(), "")
            said = page.locator(".flash.ok").count() == 1 and "ok=" not in page.url
            page.reload()
            page.wait_for_load_state("networkidle")
            report(f"{engine}: saved is said once, not again on reload", said and page.locator(".flash.ok").count() == 0, page.url)
            # A form that cannot be saved comes back with what was typed and the error next to it.
            page.goto(f"{base}/guard?s=work")
            root = page.locator("form[action$='/guard/root']:has(input[name=host])")
            root.locator("input[name=host]").fill("lab-pc")
            root.locator("input[name=until]").fill("next friday")
            root.locator("input[name=confirm]").fill("lab-pc")
            root.locator("button[type=submit]").click()
            page.wait_for_load_state("networkidle")
            root = page.locator("form[action$='/guard/root']:has(input[name=host])")
            err = root.locator(".ferr")
            report(f"{engine}: a form that fails keeps what was typed, with the error next to it",
                   err.count() == 1 and "next friday" in err.inner_text() and root.locator("input[name=until]").input_value() == "next friday", page.url)
            page.goto(f"{base}/tasks?s=work")
            page.click("text=Bake bread")
            page.wait_for_load_state("networkidle")
            report(f"{engine}: a task opens from the list", "two loaves for Sunday" in page.content(), page.url)

            page.goto(f"{base}/guard?s=work")
            form = page.locator("form:has(input[name=list][value=allowed]):has(input[name=op][value=add])").first
            repo = os.path.join(work, "app-" + engine)  # one per engine: the same home serves them all
            form.locator("input[name=value]").fill(repo)
            form.locator("button[type=submit]").click()
            page.wait_for_timeout(300)
            cfgtext = open(os.path.join(CFG, "config.toml"), encoding="utf-8").read()
            report(f"{engine}: adding to the allowed list needs the box ticked", "app-" + engine not in cfgtext, "")
            form.locator("input[name=confirm]").check()
            form.locator("button[type=submit]").click()
            page.wait_for_load_state("networkidle")
            cfgtext = open(os.path.join(CFG, "config.toml"), encoding="utf-8").read()
            report(f"{engine}: with the box ticked it is added", "app-" + engine in cfgtext, "")
            # What the assistant saves on its own: choose "ask first", then back.
            for level in ("ask", "tell"):
                page.goto(f"{base}/guard?s=work")
                trust = page.locator("form[action$='/guard/trust']")
                trust.locator(f"input[name=value][value={level}]").check()
                trust.locator("button[type=submit]").click()
                page.wait_for_load_state("networkidle")
                sestext = open(os.path.join(CFG, "session.toml"), encoding="utf-8").read()
                shown = page.locator(f"form[action$='/guard/trust'] input[name=value][value={level}]").is_checked()
                report(f"{engine}: the Guard page saves memory_trust = {level}", f'memory_trust = "{level}"' in sestext and shown, sestext)
            ctx.close()
            browser.close()

    # A change made outside the page shows on it (with the history the panel keeps).
    with open(os.path.join(CFG, "config.toml"), "a", encoding="utf-8") as f:
        f.write('\nuntouchable_hosts = ["sneaky-host"]\n')
    with sync_playwright() as pw:
        browser = getattr(pw, ENGINES[0]).launch()
        page = browser.new_page()
        page.goto(f"{base}/guard?s=work")
        text = page.content()
        report(f"{ENGINES[0]}: the Guard page shows a change made outside it", "sneaky-host" in text and ("d-add" in text or "not among the kept versions" in text), "")
        browser.close()
finally:
    ab("panel", "stop")
    shutil.rmtree(TMP, ignore_errors=True)

print(f"\n{good} ok, {bad} BAD")
sys.exit(1 if bad else 0)
