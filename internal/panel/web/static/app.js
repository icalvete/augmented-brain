// The panel works without this file; it only saves a click (a new space in the
// selector opens at once) and says when the note on screen changed elsewhere.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll("select.autosubmit").forEach(function (s) {
    s.addEventListener("change", function () { s.form.submit(); });
  });
  document.querySelectorAll(".nojs").forEach(function (b) { b.hidden = true; });

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
