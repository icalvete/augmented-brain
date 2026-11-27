// The panel works without this file; it only saves a click (a new space in the
// selector opens at once), says when the note on screen changed elsewhere, and keeps
// the panel running while someone types.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll("select.autosubmit").forEach(function (s) {
    s.addEventListener("change", function () { s.form.submit(); });
  });
  document.querySelectorAll(".nojs").forEach(function (b) { b.hidden = true; });

  // A page waiting on something outside (Telegram's QR code, scanned on the phone)
  // looks again every few seconds, unless someone is typing in it.
  var reload = document.querySelector("[data-reload]");
  if (reload) {
    setInterval(function () {
      var a = document.activeElement;
      if (a && (a.tagName === "INPUT" || a.tagName === "TEXTAREA") && a.value) { return; }
      location.replace(location.pathname + location.search);
    }, (parseInt(reload.dataset.reload, 10) || 5) * 1000);
  }

  // "Saved" is said once: the address loses its ok= at once, so reloading the page or
  // coming back to it later does not say it again.
  if (window.history && history.replaceState && window.URLSearchParams) {
    var q = new URLSearchParams(location.search);
    if (q.has("ok")) {
      q.delete("ok");
      var rest = q.toString();
      history.replaceState(history.state, "", location.pathname + (rest ? "?" + rest : "") + location.hash);
    }
  }

  // Typing is using the panel, even without changing page: while someone types, the
  // page says so at most once a minute, so the panel does not stop (after 30 minutes
  // without use) under a long text not yet saved. A tab left open without typing
  // still does not count.
  var aliveURL = document.body.dataset.alive;
  if (aliveURL && window.fetch) {
    var lastAlive = 0;
    document.addEventListener("input", function (e) {
      var el = e.target;
      if (!el || !(el.tagName === "TEXTAREA" || el.tagName === "INPUT")) { return; }
      var now = Date.now();
      if (now - lastAlive < 60000) { return; }
      lastAlive = now;
      fetch(aliveURL, { credentials: "same-origin" }).catch(function () {});
    });
  }

  // A note's page asks now and then whether the note changed on disk (Claude, another
  // tab): if it did, it says so, so what you read and save is not behind. Only while
  // the tab is in view; the asking does not keep the panel running.
  var changed = document.getElementById("changed");
  if (changed && window.fetch) {
    var check = function () {
      if (document.hidden || !changed.hidden) { return; }
      fetch(changed.dataset.url, { credentials: "same-origin" }).then(function (r) {
        return r.ok ? r.text() : "";
      }).then(function (t) {
        if (t && t !== changed.dataset.stamp) { changed.hidden = false; }
      }).catch(function () {});
    };
    setInterval(check, 15000);
    document.addEventListener("visibilitychange", check);
  }
});
