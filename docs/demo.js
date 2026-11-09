// Plays the demo on the home page: a week with augmented brain. Each [data-step] shows in
// turn, the user's lines are typed, the assistant's come after a short "typing" pause, and
// the chat scrolls down. The day buttons jump to a day; the play button pauses. It loops,
// and does nothing with reduced motion or without JavaScript: then every step shows at once.
(function () {
  var demo = document.querySelector("[data-demo]");
  if (!demo || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  var body = demo.querySelector(".demo-body");
  var steps = Array.prototype.slice.call(body.querySelectorAll("[data-step]"));
  var button = demo.querySelector(".demo-play");
  var days = Array.prototype.slice.call(demo.querySelectorAll("[data-goto]"));
  var typing = document.createElement("div");
  typing.className = "demo-typing"; typing.setAttribute("aria-hidden", "true");
  typing.innerHTML = "<i></i><i></i><i></i>";
  var texts = steps.map(function (s) { return s.classList.contains("you") ? s.textContent : null; });
  // i: the next step to show. running: a chain of timers is going. gen: bumped on a jump,
  // so the timers of the chain before it do nothing.
  var i = 0, paused = false, visible = false, running = false, gen = 0;

  function later(f, ms) { var g = gen; setTimeout(function () { if (g === gen) f(); }, ms); }
  function scroll() { body.scrollTop = body.scrollHeight; }
  function mark(day) {
    days.forEach(function (b) {
      if (b.dataset.goto === day) b.setAttribute("aria-current", "true"); else b.removeAttribute("aria-current");
    });
  }
  // Show every step before k at once, hide the rest, and carry on from k. A day button
  // jumps just past its day's heading, so the heading shows even while paused.
  function jump(k) {
    gen++; running = false; i = k;
    if (typing.parentNode) typing.parentNode.removeChild(typing);
    var day = null;
    steps.forEach(function (s, n) {
      if (texts[n] !== null) s.textContent = texts[n];
      s.classList.toggle("shown", n < k);
      if (n < k && s.dataset.day) day = s.dataset.day;
    });
    mark(day);
    if (k === 0) body.scrollTop = 0; else scroll();
    go();
  }
  function go() { if (!running && !paused && visible) { running = true; next(); } }
  function next() {
    // Paused or off screen: stop before the next step, and carry on from it later.
    if (paused || !visible) { running = false; return; }
    if (i >= steps.length) { later(function () { jump(0); }, 5000); return; }
    var s = steps[i], k = i; i++;
    if (s.dataset.day) mark(s.dataset.day);
    if (s.classList.contains("you")) {
      var full = texts[k], n = 0;
      s.textContent = ""; s.classList.add("shown"); scroll();
      (function type() {
        if (paused) { s.textContent = full; running = false; return; }
        n++; s.textContent = full.slice(0, n);
        if (n < full.length) later(type, 38); else later(next, 700);
      })();
    } else if (s.classList.contains("ai")) {
      s.parentNode.insertBefore(typing, s); typing.classList.add("shown"); scroll();
      later(function () {
        typing.classList.remove("shown"); s.classList.add("shown"); scroll();
        later(next, 1400 + s.textContent.length * 22);
      }, 1100);
    } else {
      s.classList.add("shown"); scroll();
      later(next, s.classList.contains("demo-day") ? 900 : 1600);
    }
  }
  // Scroll again once a step has settled: fonts and wrapping can change its height.
  body.addEventListener("animationend", scroll);
  button.addEventListener("click", function () {
    paused = !paused;
    button.textContent = paused ? button.dataset.play : button.dataset.pause;
    button.setAttribute("aria-pressed", String(paused));
    go();
  });
  days.forEach(function (b) {
    b.addEventListener("click", function () {
      var k = steps.findIndex(function (s) { return s.dataset.day === b.dataset.goto; });
      if (k >= 0) jump(k + 1);
    });
  });
  // Start when the demo comes into view, and wait while it is off screen.
  new IntersectionObserver(function (es) {
    visible = es[0].isIntersecting;
    go();
  }, { threshold: 0.35 }).observe(demo);
  demo.classList.add("on");
  button.hidden = false;
  days.forEach(function (b) { b.hidden = false; });
  if (days.length) days[0].parentNode.hidden = false;
})();
