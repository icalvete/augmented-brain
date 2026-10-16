// Plays the demo on the home page: each [data-step] shows in turn, the user's lines are
// typed, the assistant's come after a short "typing" pause, and the chat scrolls down.
// It loops, has a pause button, and does nothing with reduced motion or without JavaScript.
(function () {
  var demo = document.querySelector("[data-demo]");
  if (!demo || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  var body = demo.querySelector(".demo-body");
  var steps = Array.prototype.slice.call(body.querySelectorAll("[data-step]"));
  var button = demo.querySelector(".demo-play");
  var typing = document.createElement("div");
  typing.className = "demo-typing"; typing.setAttribute("aria-hidden", "true");
  typing.innerHTML = "<i></i><i></i><i></i>";
  var texts = steps.map(function (s) { return s.classList.contains("you") ? s.textContent : null; });
  var i = 0, paused = false, visible = false, stalled = true;

  function later(f, ms) { setTimeout(f, ms); }
  function scroll() { body.scrollTop = body.scrollHeight; }
  function reset() {
    steps.forEach(function (s, k) { s.classList.remove("shown"); if (texts[k] !== null) s.textContent = texts[k]; });
    if (typing.parentNode) typing.parentNode.removeChild(typing);
    body.scrollTop = 0; i = 0;
  }
  function next() {
    // Paused or off screen: stop before the next step, and carry on from it later.
    if (paused || !visible) { stalled = true; return; }
    if (i >= steps.length) { later(function () { reset(); next(); }, 5000); return; }
    var s = steps[i], k = i; i++;
    if (s.classList.contains("you")) {
      var full = texts[k], n = 0;
      s.textContent = ""; s.classList.add("shown"); scroll();
      (function type() {
        if (paused) { s.textContent = full; stalled = true; return; }
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
    if (!paused && stalled) { stalled = false; next(); }
  });
  // Start when the demo comes into view, and wait while it is off screen.
  new IntersectionObserver(function (es) {
    visible = es[0].isIntersecting;
    if (visible && stalled && !paused) { stalled = false; next(); }
  }, { threshold: 0.35 }).observe(demo);
  demo.classList.add("on");
  button.hidden = false;
})();
