// Pure helpers for reader feedback. Shared by the tests and the page:
// the build inlines this file with `export` removed, so keep it import-free.

var FEEDBACK_COLLECTION = "feedback";
// Comment text limit is 4 KiB as UTF-8; stay a little under it.
var MESSAGE_MAX_BYTES = 4000;
var PUBLISH_CAPABILITIES = { db: {}, comments: {}, user: {} };

// db path segments allow letters, digits and _ - . ~ : @ + only.
var DOC_ID_RE = /^[A-Za-z0-9_.~:@+-]+$/;

function utf8Length(s) {
  var n = 0;
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    if (c < 0x80) n += 1;
    else if (c < 0x800) n += 2;
    else if (c >= 0xd800 && c <= 0xdbff) { n += 4; i++; }
    else n += 3;
  }
  return n;
}

// Cut to at most maxBytes of UTF-8 without splitting a character; adds "…" when cut.
function truncateUtf8(s, maxBytes) {
  if (utf8Length(s) <= maxBytes) return s;
  var budget = maxBytes - 3; // "…" is 3 bytes
  var out = "";
  var used = 0;
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    var ch = c >= 0xd800 && c <= 0xdbff ? s.slice(i, i + 2) : s[i];
    var len = utf8Length(ch);
    if (used + len > budget) break;
    out += ch;
    used += len;
    i += ch.length - 1;
  }
  return out + "…";
}

// Comments reject control characters other than newline and tab.
function cleanText(s) {
  return String(s == null ? "" : s)
    .replace(/\r\n?/g, "\n")
    .replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/g, "")
    .trim();
}

// The part of a feedback doc that counts as "what the reader said".
function contentOf(doc) {
  doc = doc || {};
  var d = doc.decision || null;
  var decision = null;
  if (d && d.type === "coa" && d.coaId) decision = { type: "coa", coaId: String(d.coaId) };
  else if (d && d.type === "none") decision = { type: "none", reason: cleanText(d.reason) };
  var notes = Object.create(null);
  var src = doc.coaNotes || {};
  Object.keys(src).sort().forEach(function (k) {
    var t = cleanText(src[k]);
    if (t) notes[k] = t;
  });
  return { decision: decision, coaNotes: notes, note: cleanText(doc.note) };
}

function isEmptyContent(c) {
  return !c.decision && !c.note && Object.keys(c.coaNotes).length === 0;
}

// FNV-1a over the canonical JSON of the content. Used for notification metadata;
// save deduplication compares the complete canonical content.
function contentHash(doc) {
  var s = JSON.stringify(contentOf(doc));
  var h = 0x811c9dc5;
  for (var i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193) >>> 0;
  }
  return ("0000000" + h.toString(16)).slice(-8);
}

function coaName(finding, id) {
  for (var i = 0; i < finding.coas.length; i++) if (finding.coas[i].id === id) return finding.coas[i].name;
  return id;
}

function decisionText(finding, doc) {
  var d = contentOf(doc).decision;
  if (!d) return "not chosen";
  if (d.type === "none") return "none — " + (d.reason || "(no reason given)");
  return coaName(finding, d.coaId);
}

// finding: {id, title, coas: [{id, name}]}. Plain text for Claude to read.
function formatMessage(finding, doc) {
  var c = contentOf(doc);
  var head = "Audit feedback · " + finding.id + " · " + cleanText(finding.title);
  var tail = "Full feedback: db " + FEEDBACK_COLLECTION + "/" + finding.id;
  var body = ["Decision: " + decisionText(finding, doc)];
  finding.coas.forEach(function (coa) {
    if (c.coaNotes[coa.id]) body.push("Note on " + coa.name + ": " + c.coaNotes[coa.id]);
  });
  Object.keys(c.coaNotes).forEach(function (k) {
    if (!finding.coas.some(function (coa) { return coa.id === k; })) body.push("Note on " + k + ": " + c.coaNotes[k]);
  });
  if (c.note) body.push("General note: " + c.note);
  head = truncateUtf8(head, 300);
  var budget = MESSAGE_MAX_BYTES - utf8Length(head) - utf8Length(tail) - 2;
  return head + "\n" + truncateUtf8(body.join("\n"), budget) + "\n" + tail;
}

// The whole doc as stored at feedback/<findingId>. Never includes names.
function buildDoc(findingId, content, meta) {
  meta = meta || {};
  var doc = {
    findingId: findingId,
    decision: content.decision,
    coaNotes: content.coaNotes,
    note: content.note,
    updatedAt: meta.updatedAt || new Date().toISOString(),
    notify: {},
  };
  if (meta.updatedBy) doc.updatedBy = meta.updatedBy;
  if (meta.notify && meta.notify.threadId) doc.notify.threadId = meta.notify.threadId;
  if (meta.notify && meta.notify.hash) doc.notify.hash = meta.notify.hash;
  return doc;
}

// What "Copy feedback JSON" puts on the clipboard.
function buildExport(report, docs) {
  var findings = {};
  Object.keys(docs).sort().forEach(function (id) {
    var d = docs[id];
    if (d && !isEmptyContent(contentOf(d))) findings[id] = d;
  });
  var out = { report: report.title };
  if (report.commit) out.commit = report.commit;
  out.findings = findings;
  return out;
}
