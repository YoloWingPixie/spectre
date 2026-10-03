// Reader feedback. Each finding autosaves to the local viewer's audit folder, db feedback/<id>, or this browser when db is absent.
// Claude is notified from the same change/blur handler that saves the field.
var fbCopyBtn = document.getElementById("fb-copy");
var fbHint = document.getElementById("fb-hint");
var fbRetry = document.getElementById("fb-retry");
var diskUrl = tools.dataset.feedbackUrl;
var diskRevisions = {};
var diskReady = false;
var FB_REPORT = { title: tools.dataset.title, commit: tools.dataset.commit };
var FB_LS_KEY = "audit-report:feedback:" + FB_REPORT.title + "@" + FB_REPORT.commit;
var FB = { mode: "pending", db: null, comments: null, userId: null, canNotify: null, notifyOff: false };
var fbStates = Object.create(null);

Array.prototype.slice.call(document.querySelectorAll("section.fb")).forEach(function (sec) {
  var row = sec.closest(".row");
  var st = {
    id: sec.dataset.fid,
    title: sec.dataset.title,
    sec: sec,
    row: row,
    body: row.querySelector(".body"),
    coas: Array.prototype.slice.call(sec.querySelectorAll('input[data-coa]')).map(function (r) {
      return { id: r.dataset.coa, name: r.dataset.name };
    }),
    saved: null, savedKey: null, notify: {},
    saving: false, want: null, wantKey: null,
    error: null, invalid: false, notifying: false, blockedHash: null,
    pendingEdits: [],
  };
  fbStates[st.id] = st;
  wireFinding(st);
});

function fbEls(st) {
  return {
    radios: Array.prototype.slice.call(st.sec.querySelectorAll('input[type="radio"]')),
    reason: st.sec.querySelector('[data-fb="reason"]'),
    whyWrap: st.sec.querySelector(".why-wrap"),
    err: st.sec.querySelector(".err"),
    note: st.sec.querySelector('[data-fb="note"]'),
    coaNotes: Array.prototype.slice.call(st.body.querySelectorAll('[data-fb="coaNote"]')),
  };
}

function keyOf(doc) {
  var n = doc.notify || {};
  return JSON.stringify([contentOf(doc), n.threadId || "", n.hash || ""]);
}

function hasOption(st, id) {
  return st.coas.some(function (coa) { return coa.id === id; });
}

function readUi(st) {
  var e = fbEls(st);
  var checked = e.radios.filter(function (r) { return r.checked; })[0];
  var decision = null;
  if (checked && checked.value === "coa") decision = { type: "coa", coaId: checked.dataset.coa };
  else if (checked && checked.value === "none") decision = { type: "none", reason: e.reason.value };
  var coaNotes = Object.create(null);
  e.coaNotes.forEach(function (t) { coaNotes[t.dataset.coa] = t.value; });
  var c = contentOf({ decision: decision, coaNotes: coaNotes, note: e.note.value });
  var saved = contentOf(st.saved);
  st.invalid = !!(c.decision && c.decision.type === "none" && !c.decision.reason);
  // "None of these" needs a reason; until then keep the last saved decision.
  if (st.invalid) c.decision = saved.decision;
  if (!c.decision && saved.decision && saved.decision.type === "coa" && !hasOption(st, saved.decision.coaId)) c.decision = saved.decision;
  Object.keys(saved.coaNotes).forEach(function (id) {
    if (!hasOption(st, id)) c.coaNotes[id] = saved.coaNotes[id];
  });
  return contentOf(c);
}

function syncVisibility(st) {
  var e = fbEls(st);
  var none = e.radios.filter(function (r) { return r.value === "none"; })[0];
  e.whyWrap.hidden = !none.checked;
  e.err.hidden = !(none.checked && !cleanText(e.reason.value));
  e.coaNotes.forEach(function (t) {
    var wrap = t.closest(".note-wrap");
    var btn = wrap.previousElementSibling;
    if (cleanText(t.value) && wrap.hidden) {
      wrap.hidden = false;
      btn.hidden = true;
    }
  });
}

function setDecided(st, c) {
  var chip = st.row.querySelector(".dec-chip");
  var d = c.decision;
  if (d && d.type === "coa") {
    var coa = st.coas.filter(function (x) { return x.id === d.coaId; })[0];
    chip.textContent = "Decided: " + (coa ? coa.name : d.coaId);
  } else if (d) chip.textContent = "Rejected all";
  chip.hidden = !d;
  st.row.dataset.decided = d ? "1" : "0";
  var n = rows.filter(function (r) { return r.dataset.decided === "1"; }).length;
  document.getElementById("decided-n").textContent = n;
  document.querySelector('[data-count="decided"]').textContent = n;
  document.querySelector('[data-count="undecided"]').textContent = rows.length - n;
  apply();
}

function renderState(st) {
  var el = st.sec.querySelector(".fb-state");
  var t = "";
  var bad = false;
  if (FB.mode === "readonly") t = "Read only: you can see this report but not save feedback here.";
  else if (st.error) { t = "Not saved — " + st.error; bad = true; }
  else if (st.saving || (FB.mode === "pending" && st.want)) t = "Saving…";
  else if (st.invalid) { t = "Not saved — say why none of these fit."; bad = true; }
  else if (st.saved) {
    if (FB.mode === "disk") t = "Saved to the audit folder";
    else if (FB.mode === "local") t = "Saved on this device only";
    else if (st.notify.hash && st.notify.hash === contentHash(st.saved)) t = "Saved · Claude notified";
    else if (st.notifying) t = "Saved · sending to Claude…";
    else if (st.notifyNote) t = "Saved · not sent to Claude: " + st.notifyNote;
    else t = "Saved";
  }
  el.textContent = t;
  el.classList.toggle("bad", bad);
  if (fbRetry) fbRetry.hidden = !Object.keys(fbStates).some(function (id) { return !!fbStates[id].error; });
}

function applyDoc(st, doc) {
  st.saved = doc;
  st.savedKey = keyOf(doc);
  st.notify = { threadId: (doc.notify || {}).threadId, hash: (doc.notify || {}).hash };
  var c = contentOf(doc);
  var e = fbEls(st);
  var active = document.activeElement;
  var preserve = function (control) { return active === control || st.pendingEdits.indexOf(control) !== -1; };
  var radioActive = e.radios.some(preserve);
  if (!radioActive) {
    e.radios.forEach(function (r) {
      r.checked = !!c.decision && (c.decision.type === "none" ? r.value === "none" : r.dataset.coa === c.decision.coaId);
    });
  }
  if (!preserve(e.reason) && c.decision && c.decision.type === "none") e.reason.value = c.decision.reason;
  if (!preserve(e.note)) e.note.value = c.note;
  e.coaNotes.forEach(function (t) { if (!preserve(t)) t.value = c.coaNotes[t.dataset.coa] || ""; });
  var current = readUi(st);
  syncVisibility(st);
  setDecided(st, current);
  renderState(st);
}

// Saving

function lsRead() {
  try { return JSON.parse(localStorage.getItem(FB_LS_KEY) || "{}") || {}; } catch (err) { return {}; }
}

function writeDoc(st, doc) {
  if (diskUrl) {
    if (!diskReady) return Promise.reject({ message: "Feedback has not loaded. Reload the page before editing." });
    var content = contentOf(doc);
    if (content.decision && content.decision.type === "coa" && !hasOption(st, content.decision.coaId)) content.decision = null;
    Object.keys(content.coaNotes).forEach(function (id) { if (!hasOption(st, id)) delete content.coaNotes[id]; });
    return diskRequest("PUT", { findingId: st.id, content: content, expectedRevision: diskRevisions[st.id], reportRevision: tools.dataset.reportRevision })
      .then(function (result) { diskRevisions[st.id] = result.revision; return result.record; });
  }
  if (FB.mode === "local") {
    try {
      var all = lsRead();
      all[st.id] = doc;
      localStorage.setItem(FB_LS_KEY, JSON.stringify(all));
      return Promise.resolve();
    } catch (err) {
      return Promise.reject({ code: "local_blocked" });
    }
  }
  if (FB.mode === "readonly") return Promise.reject({ code: "readonly" });
  var ref = FB.db.collection(FEEDBACK_COLLECTION).doc(st.id);
  return ref.set(doc).catch(function (e) {
    if (!e || e.code !== "unavailable") throw e;
    return new Promise(function (res) { setTimeout(res, 300 + Math.random() * 700); }).then(function () { return ref.set(doc); });
  });
}

function saveError(st, e) {
  if (diskUrl) return (e && e.message) || "The local viewer is unavailable. Your edits are still here; retry when it is running.";
  var code = (e && e.code) || "unavailable";
  if (code === "invalid_argument" && FB.canWrite !== true) { setReadOnly(); return null; }
  if (code === "revoked" || code === "not_granted" || code === "capability_disabled" || code === "capability_removed") {
    FB.mode = "local";
    showHint("Saving to this page stopped working, so feedback is now kept in this browser only. Use Copy feedback JSON to hand it to Claude.");
    return "retry-local";
  }
  var msg = {
    local_blocked: "this browser blocks storage. Use Copy feedback JSON instead.",
    resource_exhausted: "too many saves at once. Change the field again in a moment.",
    quota_exceeded: "this page's storage is full.",
    unavailable: "the service is busy. Change the field again to retry.",
    invalid_argument: "the feedback could not be stored.",
  }[code];
  return msg || "something went wrong (" + code + ").";
}

function requestSave(st) {
  var c = readUi(st);
  syncVisibility(st);
  setDecided(st, c);
  var doc = buildDoc(st.id, c, { updatedBy: FB.userId, notify: st.notify });
  var k = keyOf(doc);
  if (k === st.savedKey && !st.saving) { st.error = null; st.want = null; renderState(st); return; }
  st.want = doc;
  st.wantKey = k;
  if (!st.saving) runSave(st);
  else renderState(st);
}

function runSave(st) {
  var doc = st.want;
  var key = st.wantKey;
  var startedPending = FB.mode === "pending";
  st.want = null;
  st.saving = true;
  st.error = null;
  renderState(st);
  FB.ready
    .then(function () {
      if (startedPending) {
        doc = buildDoc(st.id, readUi(st), { updatedBy: FB.userId, notify: st.notify });
        key = keyOf(doc);
        st.want = null;
      }
      return writeDoc(st, doc);
    })
    .then(function (stored) {
      st.saved = diskUrl && stored ? stored : doc;
      st.savedKey = diskUrl && stored ? keyOf(stored) : key;
    }, function (e) {
      var r = saveError(st, e);
      if (r === "retry-local") return writeDoc(st, doc).then(function () { st.saved = doc; st.savedKey = key; }, function () { st.error = saveError(st, { code: "local_blocked" }); });
      st.error = r;
    })
    .then(function () {
      st.saving = false;
      if (st.want && st.wantKey !== st.savedKey && !st.error) runSave(st);
      else { st.want = null; renderState(st); }
    });
}

// Notifying Claude. Called from the viewer's own change/blur handler, in the same tick as the save:
// sendToClaude must be the first async step, so nothing is awaited before it
// (the anchor is built on focusin and the send permission is cached).

var SKIP_REASONS = {
  no_session: "no Claude session watching",
  writers_only: "only editors of this page can notify Claude",
  off: "sending to Claude is off here",
};
var ERROR_REASONS = {
  consent_required: "consent declined",
  forbidden: "commenting from this page is off for you",
  rate_limited: "rate_limited, will send with your next change",
};

function cacheAnchor(st) {
  if (!FB.comments || st.anchor || st.anchorPending) return;
  st.anchorPending = true;
  FB.comments.anchorFor(st.row.querySelector("summary")).then(function (a) { st.anchor = a; }, function (e) {
    console.info("[audit-report] anchorFor failed:", e && e.code, e && e.message);
  }).then(function () { st.anchorPending = false; });
}

function maybeNotify(st) {
  if (diskUrl) return;
  var c = readUi(st);
  if (st.invalid || (isEmptyContent(c) && !st.notify.hash)) return;
  var h = contentHash(c);
  if (h === st.notify.hash) { st.notifyNote = null; return; }
  if (FB.mode === "local" || FB.mode === "readonly") return;
  var skip = null;
  if (FB.mode === "pending") skip = "still connecting to Claude";
  else if (!FB.comments) skip = "comments are not available on this page";
  else if (FB.notifyOff) skip = FB.notifyOff;
  else if (FB.canNotify === null) skip = "still checking whether Claude is watching";
  else if (FB.canNotify !== "available") skip = SKIP_REASONS[FB.canNotify] || String(FB.canNotify);
  else if (st.notifying) skip = "still sending your previous change; this one goes with your next change";
  else if (h === st.blockedHash) skip = st.notifyNote ? null : "not sent after an earlier failure; change a field to retry";
  if (skip) {
    st.notifyNote = skip;
    console.info("[audit-report] not sending " + st.id + " to Claude:", skip);
    return;
  }
  if (h === st.blockedHash) return;
  var text = formatMessage({ id: st.id, title: st.title, coas: st.coas }, c);
  var threadId = st.notify.threadId;
  var sent;
  st.notifying = true;
  st.notifyNote = null;
  if (threadId) sent = FB.comments.sendToClaude({ threadId: threadId, text: text });
  else if (st.anchor) sent = FB.comments.sendToClaude({ anchor: st.anchor, text: text });
  else {
    // No cached anchor (focus never entered this finding): build it now, then send.
    sent = FB.comments.anchorFor(st.row.querySelector("summary")).then(function (a) {
      st.anchor = a;
      return FB.comments.sendToClaude({ anchor: a, text: text });
    });
  }
  sent
    .then(function (res) {
      st.notify = { threadId: res.threadId, hash: h };
      st.blockedHash = null;
      st.notifyNote = null;
      requestSave(st);
    }, function (e) { notifyError(st, h, e); })
    .then(function () { st.notifying = false; renderState(st); });
  renderState(st);
}

function notifyError(st, h, e) {
  var code = (e && e.code) || "upstream_error";
  console.info("[audit-report] sendToClaude failed for " + st.id + ":", code, e && e.message);
  st.notifyNote = ERROR_REASONS[code] || code;
  if (code === "consent_required") {
    FB.notifyOff = "consent declined";
    showHint("You did not allow this page to comment for you, so Claude was not notified. Your feedback is still saved.");
  } else if (code === "forbidden" || code === "not_granted" || code === "capability_disabled" || code === "capability_removed") {
    FB.notifyOff = st.notifyNote;
    showHint("Notifying Claude from this page is off for you here. Your feedback is still saved, and Claude can read it later.");
  } else if (code === "not_found") {
    // The thread is gone; the next change starts a new one.
    st.notify = { hash: st.notify.hash };
    st.notifyNote = "thread not found, will start a new one with your next change";
  } else {
    // claude_unavailable, rate_limited, unavailable, upstream_error, invalid: wait for the next change.
    st.blockedHash = h;
    if (code === "claude_unavailable") checkNotify();
  }
}

var NOTIFY_HINTS = {
  no_session: "Claude isn't watching this page right now. Your feedback is saved and Claude can read it later.",
  writers_only: "Only editors of this page can notify Claude. Your feedback is saved and Claude can read it later.",
  off: "Notifying Claude is off on this page. Your feedback is saved and Claude can read it later.",
};

function checkNotify() {
  if (!FB.comments) return;
  FB.comments.canSendToClaude().then(function (v) {
    FB.canNotify = v;
    if (FB.mode === "db") showHint(v === "available" ? "" : NOTIFY_HINTS[v] || NOTIFY_HINTS.off);
  }, function () {
    FB.canNotify = "off";
    if (FB.mode === "db") showHint(NOTIFY_HINTS.off);
  });
}

function showHint(text) {
  fbHint.textContent = text;
  fbHint.hidden = !text;
}

function setReadOnly() {
  FB.mode = "readonly";
  document.querySelectorAll("section.fb input, section.fb textarea, .coa-note textarea, .coa-note button").forEach(function (el) { el.disabled = true; });
  showHint("You can read this report but not save feedback here. Ask the owner for Contributor access.");
  Object.keys(fbStates).forEach(function (id) { renderState(fbStates[id]); });
}

function wireFinding(st) {
  function trackPendingEdit(ev) {
    if (FB.mode !== "pending" || !ev.target.matches('input[type="radio"], textarea[data-fb]')) return;
    if (st.pendingEdits.indexOf(ev.target) === -1) st.pendingEdits.push(ev.target);
  }
  st.body.addEventListener("input", trackPendingEdit, true);
  st.body.addEventListener("change", trackPendingEdit, true);
  // Save and notify start in the same tick, inside the viewer's gesture.
  function changed() {
    requestSave(st);
    maybeNotify(st);
    renderState(st);
  }
  st.sec.addEventListener("change", function (ev) {
    if (ev.target.type === "radio" && ev.target.value === "none") {
      syncVisibility(st);
      var reason = fbEls(st).reason;
      if (!cleanText(reason.value)) reason.focus();
    }
    changed();
  });
  st.body.addEventListener("change", function (ev) {
    if (ev.target.dataset && ev.target.dataset.fb === "coaNote") changed();
  });
  fbEls(st).reason.addEventListener("input", function () { syncVisibility(st); });
  st.body.addEventListener("click", function (ev) {
    var btn = ev.target.closest && ev.target.closest(".add-note");
    if (!btn) return;
    var wrap = document.getElementById(btn.getAttribute("aria-controls"));
    wrap.hidden = false;
    btn.hidden = true;
    btn.setAttribute("aria-expanded", "true");
    wrap.querySelector("textarea").focus();
  });
  st.body.addEventListener("focusin", function () {
    if (!FB.comments) return;
    cacheAnchor(st);
    if (!FB.notifyOff) checkNotify();
  });
}

// Copy feedback JSON: the hand-off for local files.
fbCopyBtn.addEventListener("click", function () {
  var docs = {};
  Object.keys(fbStates).forEach(function (id) {
    var st = fbStates[id];
    var content = readUi(st);
    if (st.saved && JSON.stringify(content) === JSON.stringify(contentOf(st.saved))) docs[id] = st.saved;
    else docs[id] = buildDoc(id, content, { updatedBy: FB.userId, notify: st.notify });
  });
  var exported = buildExport(FB_REPORT, docs);
  if (diskUrl && FB.projectId) { exported.projectId = FB.projectId; exported.auditId = FB.auditId; }
  var text = JSON.stringify(exported, null, 2);
  var n = Object.keys(JSON.parse(text).findings).length;
  var fail = function () {
    var ta = document.getElementById("fb-json") || document.createElement("textarea");
    ta.id = "fb-json";
    ta.className = "fb-json mono";
    ta.readOnly = true;
    ta.setAttribute("aria-label", "Feedback JSON");
    ta.value = text;
    if (!ta.parentNode) fbHint.parentNode.insertBefore(ta, fbHint.nextSibling);
    ta.focus();
    ta.select();
    say("Could not copy. The JSON is selected below: press Ctrl+C.");
  };
  try {
    if (!navigator.clipboard || !navigator.clipboard.writeText) return fail();
    navigator.clipboard.writeText(text).then(function () {
      say("Copied feedback for " + n + (n === 1 ? " finding" : " findings"));
    }, fail);
  } catch (err) {
    fail();
  }
});

// Capabilities arrive late (up to 10 s) or never; the page works without them.
function useCap(name) {
  try {
    if (window.claude && typeof window.claude.use === "function") {
      return Promise.resolve(window.claude.use(name)).catch(function () { return null; });
    }
  } catch (err) {}
  return Promise.resolve(null);
}

function diskRequest(method, body) {
  return fetch(diskUrl, {
    method: method,
    headers: { "Content-Type": "application/json", "X-Audit-Token": tools.dataset.feedbackToken },
    body: body ? JSON.stringify(body) : undefined,
  }).then(function (response) {
    return response.json().then(function (data) {
      if (!response.ok) throw new Error(data.error || "Could not save feedback");
      return data;
    });
  });
}

function startDiskFeedback() {
  FB.mode = "disk";
  var controls = document.querySelectorAll("section.fb input, section.fb textarea, .coa-note textarea, .coa-note button");
  controls.forEach(function (el) { el.disabled = true; });
  return diskRequest("GET").then(function (data) {
    if (data.reportRevision !== tools.dataset.reportRevision) throw new Error("The report changed. Reload to see the current options.");
    FB.projectId = data.projectId;
    FB.auditId = data.auditId;
    diskRevisions = data.revisions;
    Object.keys(data.findings).forEach(function (id) {
      if (fbStates[id]) applyDoc(fbStates[id], data.findings[id]);
    });
    diskReady = true;
    controls.forEach(function (el) { el.disabled = false; });
    showHint(data.review.length ? "Saved feedback needs review: " + data.review.join("; ") : "Decisions and notes save to this audit's folder. The agent reads them when it resumes.");
  }).catch(function (error) {
    showHint("Feedback did not load: " + error.message + " Reload the page to try again.");
  });
}

if (fbRetry) fbRetry.addEventListener("click", function () {
  Object.keys(fbStates).forEach(function (id) { if (fbStates[id].error) requestSave(fbStates[id]); });
});

FB.ready = diskUrl ? startDiskFeedback() : Promise.all([useCap("db"), useCap("comments"), useCap("user")]).then(function (caps) {
  FB.db = caps[0];
  FB.comments = caps[1];
  var user = caps[2];
  var who = user
    ? Promise.all([
        user.id().catch(function () { return null; }),
        user.can("data.write").catch(function () { return null; }),
      ])
    : Promise.resolve([null, null]);
  return who.then(function (w) {
    FB.userId = w[0];
    FB.canWrite = w[1];
    if (!FB.db) {
      FB.mode = "local";
      var all = lsRead();
      Object.keys(all).forEach(function (id) { if (fbStates[id] && all[id]) applyDoc(fbStates[id], all[id]); });
      showHint("Feedback is saved in this browser only. Use Copy feedback JSON to hand it to Claude.");
      return;
    }
    if (FB.canWrite === false) setReadOnly();
    else FB.mode = "db";
    FB.db.collection(FEEDBACK_COLLECTION).onSnapshot(function (snap) {
      snap.docs.forEach(function (d) {
        var st = fbStates[d.id];
        if (!st || !d.exists || st.saving || st.want) return;
        var data = JSON.parse(JSON.stringify(d.data() || {}));
        if (keyOf(data) !== st.savedKey) applyDoc(st, data);
      });
    }, function () {});
    checkNotify();
  });
}).catch(function () {
  FB.mode = "local";
});
FB.ready.then(function () {
  Object.keys(fbStates).forEach(function (id) { fbStates[id].pendingEdits = []; renderState(fbStates[id]); });
});
