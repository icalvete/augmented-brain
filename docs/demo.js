// The landing page's moving parts: the demo and the install tabs.
//
// The demo is three days with augmented brain in an app window, one scene per session.
// Each [data-step] shows in turn: what the user says is typed, the assistant's answer
// comes after a moment of "typing", a tool line lights up the lines of the task's file
// it writes (data-key on the step, data-on and data-off on the file's lines), and the
// guard's question gets its Allow pressed (data-press). It plays once, when it comes into
// view, and then offers to play again; the day buttons jump to a scene.
//
// Without JavaScript every scene shows, one after the other, and the file its last state.
// With reduced motion nothing moves: the day buttons show each scene whole, still.
(function () {
  var demo = document.querySelector("[data-demo]");
  if (demo) play(demo);
  tabs();

  function play(demo) {
    var chat = demo.querySelector(".app-chat");
    var scenes = Array.prototype.slice.call(demo.querySelectorAll(".scene"));
    var steps = Array.prototype.slice.call(demo.querySelectorAll("[data-step]"));
    var lines = Array.prototype.slice.call(demo.querySelectorAll(".note-file .fl"));
    var sessions = Array.prototype.slice.call(demo.querySelectorAll(".app-side li[data-for]"));
    var title = demo.querySelector("[data-title]");
    var ctl = demo.querySelector(".demo-ctl");
    var days = Array.prototype.slice.call(ctl.querySelectorAll("[data-goto]"));
    var button = ctl.querySelector(".play");
    var still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    var says = steps.map(function (s) { return s.classList.contains("say") ? s.textContent : null; });
    var sceneOf = steps.map(function (s) { return scenes.indexOf(s.closest(".scene")); });
    var typing = document.createElement("div");
    typing.className = "typing"; typing.setAttribute("aria-hidden", "true");
    typing.innerHTML = "<i></i><i></i><i></i>";

    // i: the next step to show. scene: the scene on screen. running: a chain of timers is
    // going. gen: bumped on a jump, so the timers of the chain before it do nothing.
    var i = 0, scene = 0, paused = false, visible = false, running = false, done = false, gen = 0;

    function later(f, ms) { var g = gen; setTimeout(function () { if (g === gen) f(); }, ms); }
    function bottom() { chat.scrollTop = chat.scrollHeight; }
    function first(n) { return sceneOf.indexOf(n); }
    function last(n) { return sceneOf.lastIndexOf(n) + 1; }

    // Put scene n on screen: its session in the sidebar, its title, its day button.
    function show(n) {
      scene = n;
      scenes.forEach(function (s, k) { s.classList.toggle("now", k === n); });
      sessions.forEach(function (li) {
        var k = scenes.findIndex(function (s) { return s.dataset.scene === li.dataset.for; });
        li.classList.toggle("later", k > n);
        li.classList.toggle("now", k === n);
      });
      title.textContent = scenes[n].dataset.title;
      days.forEach(function (b, k) {
        if (k === n) b.setAttribute("aria-current", "true"); else b.removeAttribute("aria-current");
      });
    }
    // The file as it is once the steps before k have happened.
    function file(k) {
      var keys = {};
      steps.slice(0, k).forEach(function (s) { if (s.dataset.key) keys[s.dataset.key] = true; });
      lines.forEach(function (l) {
        var on = !l.dataset.on || keys[l.dataset.on], off = l.dataset.off && keys[l.dataset.off];
        l.classList.toggle("gone", !on || !!off);
        l.classList.remove("fresh");
      });
    }
    // What a step does to the file when it happens: new lines light up for a moment.
    function touch(key) {
      lines.forEach(function (l) {
        if (l.dataset.off === key) l.classList.add("gone");
        if (l.dataset.on === key) {
          l.classList.remove("gone"); l.classList.add("fresh");
          setTimeout(function () { l.classList.remove("fresh"); }, 1400);
        }
      });
    }
    // Show every step before k at once, hide the rest, and carry on from k.
    function jump(k) {
      gen++; running = false; done = false; i = k;
      if (typing.parentNode) typing.parentNode.removeChild(typing);
      steps.forEach(function (s, n) {
        if (says[n] !== null) s.textContent = says[n];
        s.classList.toggle("shown", n < k);
        if (s.dataset.press !== undefined) s.classList.toggle("pressed", n < k);
      });
      show(k > 0 ? sceneOf[k - 1] : 0);
      if (k < steps.length && sceneOf[k] !== scene && k > 0) show(sceneOf[k]);
      file(k);
      chat.scrollTop = 0;
      label();
      go();
    }
    function label() {
      button.textContent = done ? button.dataset.again : paused ? button.dataset.play : button.dataset.pause;
      button.setAttribute("aria-pressed", String(paused && !done));
    }
    function go() { if (!running && !paused && visible && !done) { running = true; next(); } }
    function next() {
      // Paused or off screen: stop before the next step, and carry on from it later.
      if (paused || !visible) { running = false; return; }
      if (i >= steps.length) { running = false; done = true; label(); return; }
      var s = steps[i], k = i;
      if (sceneOf[k] !== scene) {
        // A new session: a moment on the empty chat, then its first step.
        show(sceneOf[k]); chat.scrollTop = 0;
        later(next, 900);
        return;
      }
      i++;
      if (s.classList.contains("say")) {
        var full = says[k], n = 0;
        s.textContent = ""; s.classList.add("shown"); bottom();
        (function type() {
          if (paused) { s.textContent = full; running = false; return; }
          n += 2; s.textContent = full.slice(0, n); bottom();
          if (n < full.length) later(type, 30); else later(next, 450);
        })();
      } else if (s.classList.contains("reply")) {
        s.parentNode.insertBefore(typing, s); bottom();
        later(function () {
          if (typing.parentNode) typing.parentNode.removeChild(typing);
          s.classList.add("shown"); bottom();
          later(next, 900 + s.textContent.length * 12);
        }, 700);
      } else if (s.dataset.press !== undefined) {
        s.classList.add("shown"); bottom();
        later(function () { s.classList.add("pressed"); later(next, 700); }, 1700);
      } else {
        s.classList.add("shown"); bottom();
        if (s.dataset.key) touch(s.dataset.key);
        later(next, 800);
      }
    }

    demo.classList.add("on");
    ctl.hidden = false;
    // Scroll again once a step has settled: fonts and wrapping can change its height.
    chat.addEventListener("animationend", bottom);

    if (still) {
      // Nothing moves: each day button shows its scene whole, and the file after it.
      button.hidden = true;
      steps.forEach(function (s) { s.classList.add("shown"); if (s.dataset.press !== undefined) s.classList.add("pressed"); });
      var still_show = function (n) { show(n); file(last(n)); chat.scrollTop = 0; };
      days.forEach(function (b, n) { b.addEventListener("click", function () { still_show(n); }); });
      still_show(0);
      return;
    }

    button.addEventListener("click", function () {
      if (done) { paused = false; jump(0); return; }
      paused = !paused;
      label();
      go();
    });
    days.forEach(function (b, n) {
      b.addEventListener("click", function () { paused = false; jump(first(n)); });
    });
    jump(0);
    // Start when the demo comes into view, and wait while it is off screen.
    new IntersectionObserver(function (es) {
      visible = es[0].isIntersecting;
      go();
    }, { threshold: 0.3 }).observe(demo);
  }

  // The install command: a tab per system, the visitor's own open first. Without
  // JavaScript both commands show, each with its label.
  function tabs() {
    var box = document.querySelector(".install");
    if (!box) return;
    var list = box.querySelector(".tabs");
    var buttons = Array.prototype.slice.call(list.querySelectorAll("[data-os]"));
    var panels = Array.prototype.slice.call(box.querySelectorAll(".os"));
    function pick(os) {
      buttons.forEach(function (b) { b.setAttribute("aria-selected", String(b.dataset.os === os)); b.tabIndex = b.dataset.os === os ? 0 : -1; });
      panels.forEach(function (p) { p.hidden = p.dataset.os !== os; });
    }
    buttons.forEach(function (b) { b.addEventListener("click", function () { pick(b.dataset.os); }); });
    list.addEventListener("keydown", function (e) {
      if (e.key !== "ArrowLeft" && e.key !== "ArrowRight") return;
      var k = buttons.findIndex(function (b) { return b.getAttribute("aria-selected") === "true"; });
      var b = buttons[(k + (e.key === "ArrowRight" ? 1 : buttons.length - 1)) % buttons.length];
      pick(b.dataset.os); b.focus();
    });
    box.classList.add("tabbed");
    list.hidden = false;
    pick(/Win/i.test(navigator.platform || navigator.userAgent) ? "win" : "unix");
  }
})();
