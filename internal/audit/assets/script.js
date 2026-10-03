var chips = Array.prototype.slice.call(document.querySelectorAll(".chip"));
var q = document.getElementById("q");
var rows = Array.prototype.slice.call(document.querySelectorAll(".row"));
var areas = Array.prototype.slice.call(document.querySelectorAll(".area"));
var statusEl = document.getElementById("status");
var expandBtn = document.getElementById("expand");
var resetBtn = document.getElementById("reset");
var toast = document.getElementById("toast");
var tools = document.querySelector(".mast-tools");
var editorSel = document.getElementById("editor");
var ROOT = tools.getAttribute("data-root");
var DISTRO = tools.getAttribute("data-distro");
var STORE_KEY = "audit-report:editor";

function apply() {
  var active = { sev: {}, status: {}, dec: {} };
  chips.forEach(function (c) {
    if (c.getAttribute("aria-pressed") === "true") active[c.dataset.kind][c.dataset.val] = true;
  });
  var terms = q.value.toLowerCase().trim().split(/\s+/).filter(Boolean);
  var shown = 0;
  rows.forEach(function (r) {
    var text = r.dataset.text;
    var ok = !!active.sev[r.dataset.sev] && !!active.status[r.dataset.status] &&
      !!active.dec[r.dataset.decided === "1" ? "decided" : "undecided"] &&
      terms.every(function (t) { return text.indexOf(t) !== -1; });
    r.hidden = !ok;
    if (ok) shown++;
  });
  areas.forEach(function (a) {
    var any = a.querySelector(".row:not([hidden])");
    a.querySelector(".none").hidden = !!any;
    a.querySelector(".rows").hidden = !any;
  });
  statusEl.textContent = shown === rows.length ? "Showing all " + rows.length + " findings" : "Showing " + shown + " of " + rows.length + " findings";
}

chips.forEach(function (c) {
  c.addEventListener("click", function () {
    c.setAttribute("aria-pressed", c.getAttribute("aria-pressed") === "true" ? "false" : "true");
    apply();
  });
});
q.addEventListener("input", apply);
q.addEventListener("keydown", function (e) { if (e.key === "Escape" && q.value) { q.value = ""; apply(); } });

expandBtn.addEventListener("click", function () {
  var open = expandBtn.textContent === "Expand all";
  rows.forEach(function (r) { if (!r.hidden) r.open = open; });
  expandBtn.textContent = open ? "Collapse all" : "Expand all";
});
resetBtn.addEventListener("click", function () {
  chips.forEach(function (c) { c.setAttribute("aria-pressed", "true"); });
  q.value = "";
  apply();
});

function reveal(id) {
  var el = id && document.getElementById(id);
  if (!el || !el.classList.contains("row")) return;
  if (el.hidden) resetBtn.click();
  el.open = true;
  el.scrollIntoView({ block: "start" });
  var s = el.querySelector("summary");
  if (s) s.focus({ preventScroll: true });
}
document.querySelectorAll('a.jump[href^="#"]').forEach(function (a) {
  a.addEventListener("click", function (e) {
    e.preventDefault();
    var id = a.getAttribute("href").slice(1);
    try { history.replaceState(null, "", "#" + id); } catch (err) {}
    reveal(id);
  });
});

// Editor links
var toastTimer;
function say(msg) {
  toast.textContent = msg;
  toast.classList.add("on");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(function () { toast.classList.remove("on"); }, 2600);
}

function selectText(el) {
  try {
    var range = document.createRange();
    range.selectNodeContents(el);
    var sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
  } catch (err) {}
}

function copy(text, fallbackEl) {
  var fail = function () {
    selectText(fallbackEl);
    say("Could not copy. The text is selected: press Ctrl+C.");
  };
  try {
    if (!navigator.clipboard || !navigator.clipboard.writeText) return fail();
    navigator.clipboard.writeText(text).then(function () { say("Copied " + text); }, fail);
  } catch (err) {
    fail();
  }
}

function currentEditor() { return editorSel.value; }

function updateLinks() {
  var ed = currentEditor();
  document.querySelectorAll("a.fref").forEach(function (a) {
    var href = editorHref(ed, {
      root: ROOT, distro: DISTRO, path: a.dataset.path,
      line: Number(a.dataset.line), col: Number(a.dataset.col),
    });
    if (href) a.setAttribute("href", href);
    else a.setAttribute("href", "#copy");
  });
  document.documentElement.setAttribute("data-editor", ed);
}

try {
  var saved = localStorage.getItem(STORE_KEY);
  if (saved && EDITORS.some(function (e) { return e.id === saved; })) editorSel.value = saved;
} catch (err) {}
editorSel.addEventListener("change", function () {
  try { localStorage.setItem(STORE_KEY, editorSel.value); } catch (err) {}
  updateLinks();
});

document.addEventListener("click", function (e) {
  var btn = e.target.closest && e.target.closest("button.copy");
  if (btn) {
    e.preventDefault();
    e.stopPropagation();
    copy(btn.dataset.copy, btn.previousElementSibling);
    return;
  }
  var a = e.target.closest && e.target.closest("a.fref");
  if (a && currentEditor() === "copy") {
    e.preventDefault();
    copy(copyText({ path: a.dataset.path, line: Number(a.dataset.line) }), a);
  }
});

updateLinks();
if (location.hash) reveal(location.hash.slice(1));
apply();
