// The panel works without this file; it only saves a click: a new space in the
// selector opens at once.
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll("select.autosubmit").forEach(function (s) {
    s.addEventListener("change", function () { s.form.submit(); });
  });
  document.querySelectorAll(".nojs").forEach(function (b) { b.hidden = true; });
});
