import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, readFileSync, writeFileSync, unlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { once } from "node:events";
import { setTimeout as delay } from "node:timers/promises";

const root = fileURLToPath(new URL("..", import.meta.url));
const directory = mkdtempSync(join(tmpdir(), "audit-browser-"));
const project = join(directory, "project");
mkdirSync(project);
const binary = process.argv[3];
const environment = { ...process.env, AUDIT_REPORT_HOME: join(directory, "index"), XDG_CACHE_HOME: join(directory, "cache") };
function run(...args) {
  const result = spawnSync(binary, args, { cwd: directory, env: environment, encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr || result.error?.message);
  return result.stdout;
}
function resume() { return JSON.parse(run("audit", "resume", "--project", project)); }
JSON.parse(run("audit", "init", "--project", project, "--source", join(root, "internal/audit/templates/report.starter.json")));
const examplePath = join(directory, "example.html");
run("audit", "build", join(root, "internal/audit/testdata/example/report.json"), "--strict", "-o", examplePath);
const prototypeReport = JSON.parse(readFileSync(join(root, "internal/audit/templates/report.starter.json"), "utf8"));
prototypeReport.title = "Prototype option identifiers";
prototypeReport.findings[0].coas.forEach((coa, i) => { if (i < 2) coa.id = ["constructor", "toString"][i]; });
const prototypeSource = join(directory, "prototype.json"), prototypePath = join(directory, "prototype.html");
writeFileSync(prototypeSource, JSON.stringify(prototypeReport));
run("audit", "build", prototypeSource, "--strict", "-o", prototypePath);
async function startViewer(mode = "audit") {
  const args = mode === "spec" ? ["spec", "-dir", join(root, "internal/spec/testdata/atc-spec"), "-port", "0"] : ["audit", "view", "--project", project];
  const server = spawn(binary, args, { cwd: directory, env: environment, stdio: ["ignore", "pipe", "pipe"] });
  let errors = "";
  server.stderr.on("data", (chunk) => { errors += chunk; });
  const url = await new Promise((resolve, reject) => {
    let output = "";
    const timer = setTimeout(() => { server.kill(); reject(new Error("Spectre did not print a viewer URL: " + errors)); }, 10000);
    server.once("error", (error) => { clearTimeout(timer); reject(error); });
    server.once("exit", (code) => { clearTimeout(timer); reject(new Error("Spectre exited " + code + ": " + errors)); });
    server.stdout.on("data", (chunk) => {
      output += chunk;
      try { const value = JSON.parse(output); clearTimeout(timer); resolve(value.url); } catch {}
    });
    if (mode === "spec") server.stderr.on("data", () => {
      const match = errors.match(/url=(http:\/\/\S+)/);
      if (match) { clearTimeout(timer); resolve(match[1]); }
    });
  });
  let closing;
  return { url, close() {
    if (!closing) closing = (async () => {
      if (server.exitCode !== null) return;
      const exited = once(server, "exit"); server.kill("SIGINT");
      const [code] = await exited; assert.equal(code, 0, errors);
    })();
    return closing;
  } };
}
let viewer = await startViewer();
let specViewer = await startViewer("spec");
const browser = spawn(process.argv[2], ["--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
  "--remote-debugging-port=0", `--user-data-dir=${join(directory, "browser-profile")}`, "about:blank"], { stdio: ["ignore", "ignore", "pipe"] });
let socket;
let sequence = 0;
const pending = new Map();
const exceptions = [];

function command(method, params = {}, sessionId) {
  const id = ++sequence;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`Timed out: ${method}`)); }, 15000);
    pending.set(id, { resolve, reject, timer });
    socket.send(JSON.stringify({ id, method, params, sessionId }));
  });
}

async function page(url, preload) {
  const { targetId } = await command("Target.createTarget", { url: "about:blank" });
  const { sessionId } = await command("Target.attachToTarget", { targetId, flatten: true });
  await command("Runtime.enable", {}, sessionId);
  await command("Page.enable", {}, sessionId);
  if (preload) await command("Page.addScriptToEvaluateOnNewDocument", { source: preload }, sessionId);
  const evaluate = async (expression) => {
    const result = await command("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true }, sessionId);
    if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails));
    return result.result.value;
  };
  const waitFor = async (expression) => {
    const deadline = Date.now() + 10000;
    while (Date.now() < deadline) {
      if (await evaluate(expression)) return;
      await delay(50);
    }
    throw new Error(`Browser condition failed: ${expression}`);
  };
  await command("Page.navigate", { url }, sessionId);
  await waitFor("document.readyState === 'complete' && !!document.querySelector('h1')");
  return { targetId, sessionId, evaluate, waitFor };
}

async function screenshots(tab, name) {
  await command("Page.bringToFront", {}, tab.sessionId);
  for (const width of [1280, 400]) {
    await command("Emulation.setDeviceMetricsOverride", { width, height: 1000, deviceScaleFactor: 1, mobile: false }, tab.sessionId);
    await tab.evaluate("var finding = document.querySelector('details.row'); if (finding) finding.open = true; document.fonts.ready.then(() => true)");
    const overflow = await tab.evaluate("document.documentElement.scrollWidth > window.innerWidth");
    assert.equal(overflow, false, `${name} overflows at ${width}px`);
    const { data } = await command("Page.captureScreenshot", { format: "png", captureBeyondViewport: true }, tab.sessionId);
    const path = join(directory, `${name}-${width}.png`);
    writeFileSync(path, Buffer.from(data, "base64"));
    console.log(path);
  }
}

try {
  const endpoint = await new Promise((resolve, reject) => {
    let output = "";
    const timer = setTimeout(() => reject(new Error(`Chromium did not start: ${output}`)), 15000);
    browser.once("error", (error) => { clearTimeout(timer); reject(error); });
    browser.once("exit", (code) => { clearTimeout(timer); reject(new Error(`Chromium exited ${code}: ${output}`)); });
    browser.stderr.on("data", (chunk) => {
      output += chunk;
      const match = output.match(/DevTools listening on (ws:\/\/\S+)/);
      if (match) { clearTimeout(timer); resolve(match[1]); }
    });
  });
  socket = new WebSocket(endpoint);
  await once(socket, "open");
  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    if (message.method === "Runtime.exceptionThrown") exceptions.push(message.params.exceptionDetails);
    const request = pending.get(message.id);
    if (!request) return;
    pending.delete(message.id);
    clearTimeout(request.timer);
    if (message.error) request.reject(new Error(JSON.stringify(message.error)));
    else request.resolve(message.result);
  });
  const report = await page(viewer.url);
  const specification = await page(specViewer.url);
  const visualStyles = `JSON.stringify((function () {
    var body = getComputedStyle(document.body), heading = getComputedStyle(document.querySelector('h1'));
    var control = getComputedStyle(document.querySelector('#theme')), header = getComputedStyle(document.querySelector('.app-bar'));
    return { font: body.fontFamily, size: body.fontSize, lineHeight: body.lineHeight,
      background: body.backgroundColor, foreground: body.color,
      headingFont: heading.fontFamily, headingSize: heading.fontSize, headingCase: heading.textTransform,
      controlFont: control.fontFamily, controlRadius: control.borderRadius, controlPadding: control.padding,
      controlBackground: control.backgroundColor, headerBackground: header.backgroundColor };
  })())`;
  assert.equal(await report.evaluate(visualStyles), await specification.evaluate(visualStyles), "Audit and spec visual foundations differ");
  for (const tab of [report, specification]) assert.equal(await tab.evaluate("!!document.querySelector('#theme')"), true, "Both modes need the shared theme control");
  const initialThemeStyles = await specification.evaluate(visualStyles);
  await specification.evaluate("document.querySelector('#theme').click()");
  assert.notEqual(await specification.evaluate(visualStyles), initialThemeStyles, "Theme control must change the displayed palette");
  await command("Page.reload", {}, report.sessionId);
  await report.waitFor("document.readyState === 'complete' && !!document.querySelector('.row')");
  assert.equal(await report.evaluate(visualStyles), await specification.evaluate(visualStyles), "Theme preference must carry across modes");
  await specification.evaluate("document.querySelector('#theme').click()");
  await command("Page.reload", {}, report.sessionId);
  await report.waitFor("document.readyState === 'complete' && !!document.querySelector('.row')");
  await report.waitFor("!document.querySelector('input[data-coa]').disabled");
  for (const text of ["Review note 512789", "Review note 749192"]) {
    await report.evaluate(`var note = document.querySelector('[data-fb="note"]'); note.value = ${JSON.stringify(text)}; note.dispatchEvent(new Event('change', { bubbles: true }));`);
    await report.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
    assert.equal(resume().feedback["SEC-01"].note, text, "Distinct notes must both save even when their hashes collide");
  }
  await report.evaluate(`document.querySelector('.row').open = true;
    document.querySelector('input[data-coa="expire-one-hour"]').click();
    var note = document.querySelector('[data-fb="note"]');
    note.value = 'Resume this decision tomorrow'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await report.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  assert.equal(resume().feedback["SEC-01"].note, "Resume this decision tomorrow");
  await screenshots(report, "viewer");
  await screenshots(specification, "spec");

  const lock = resume().paths.feedback + ".lock";
  writeFileSync(lock, String(process.pid), { flag: "wx" });
  const release = () => unlinkSync(lock);
  try {
    await report.evaluate(`var note = document.querySelector('[data-fb="note"]'); note.value = 'Retry this note'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
    await report.waitFor("document.querySelector('.fb-state').textContent.includes('Locked:')");
    assert.equal(resume().feedback["SEC-01"].note, "Resume this decision tomorrow");
  } finally { release(); }
  await report.evaluate("document.querySelector('#fb-retry').click()");
  await report.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  assert.equal(resume().feedback["SEC-01"].note, "Retry this note");

  const stale = await page(viewer.url);
  await stale.waitFor("!document.querySelector('input[data-coa]').disabled");
  await report.evaluate(`var note = document.querySelector('[data-fb="note"]'); note.value = 'Newer saved note'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await report.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  await stale.evaluate(`var note = document.querySelector('[data-fb="note"]'); note.value = 'My stale unsaved edit'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await stale.waitFor("document.querySelector('.fb-state').textContent.includes('conflict')");
  assert.equal(await stale.evaluate("document.querySelector('[data-fb=note]').value"), "My stale unsaved edit");
  assert.equal(resume().feedback["SEC-01"].note, "Newer saved note");

  await viewer.close();
  await specViewer.close();
  await report.evaluate(`var note = document.querySelector('[data-fb="note"]'); note.value = 'Keep my offline edit'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await report.waitFor("document.querySelector('.fb-state').textContent.startsWith('Not saved')");
  assert.equal(await report.evaluate("document.querySelector('[data-fb=note]').value"), "Keep my offline edit");
  assert.equal(await report.evaluate("document.querySelector('#fb-retry').hidden"), false);

  viewer = await startViewer();
  const restored = await page(viewer.url);
  await restored.waitFor("!document.querySelector('input[data-coa]').disabled");
  assert.equal(await restored.evaluate("document.querySelector('[data-fb=note]').value"), "Newer saved note");
  assert.equal(await restored.evaluate("document.querySelector('input[data-coa=expire-one-hour]').checked"), true);

  await restored.evaluate(`var note = document.querySelector('[data-coa="expire-one-hour"][data-fb="coaNote"]'); note.value = 'Retain the retired option note'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await restored.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  const reportPath = resume().paths.report;
  const withoutOptions = JSON.parse(readFileSync(reportPath, "utf8"));
  withoutOptions.findings[0].coas = [];
  writeFileSync(reportPath, JSON.stringify(withoutOptions));
  const retired = await page(viewer.url);
  assert.equal(await retired.evaluate("!!document.querySelector('section.fb')"), true, "Findings without options still need feedback");
  await retired.waitFor("!document.querySelector('[data-fb=note]').disabled");
  assert.equal(await retired.evaluate("document.querySelector('[data-fb=note]').value"), "Newer saved note");
  const captureClipboard = `Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: function (text) { window.copiedFeedback = JSON.parse(text); return Promise.resolve(); } } });`;
  await retired.evaluate(captureClipboard);
  await retired.evaluate("document.querySelector('#fb-copy').click()");
  assert.equal(await retired.evaluate("copiedFeedback.findings['SEC-01'].decision.coaId"), "expire-one-hour");
  await retired.evaluate(`var note = document.querySelector('[data-fb=note]'); note.value = 'General note after options removed'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await retired.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  assert.equal(resume().feedback["SEC-01"].decision.coaId, "expire-one-hour");
  assert.equal(resume().feedback["SEC-01"].coaNotes["expire-one-hour"], "Retain the retired option note");
  assert.equal(await retired.evaluate("document.querySelector('.dec-chip').hidden"), false);
  await retired.evaluate("document.querySelector('#fb-copy').click()");
  assert.equal(await retired.evaluate("copiedFeedback.findings['SEC-01'].note"), "General note after options removed");
  assert.equal(await retired.evaluate("copiedFeedback.findings['SEC-01'].coaNotes['expire-one-hour']"), "Retain the retired option note");

  const example = await page(pathToFileURL(examplePath).href);
  await screenshots(example, "example");
  await example.waitFor("document.querySelector('#fb-hint').textContent.includes('saved in this browser only')");
  await example.evaluate(captureClipboard);
  await example.evaluate(`Storage.prototype.setItem = function () { throw new Error('Storage blocked for regression check'); };
    var note = document.querySelector('[data-fb=note]'); note.value = 'Export this unsaved note'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await example.waitFor("document.querySelector('.fb-state').textContent.startsWith('Not saved')");
  await example.evaluate("document.querySelector('#fb-copy').click()");
  assert.equal(await example.evaluate("Object.values(copiedFeedback.findings)[0]?.note"), "Export this unsaved note", "Copy must recover unsaved standalone edits");

  const prototype = await page(pathToFileURL(prototypePath).href);
  await prototype.waitFor("document.querySelector('#fb-hint').textContent.includes('saved in this browser only')");
  await prototype.evaluate(`var note = document.querySelector('[data-fb=note]'); note.value = 'Create saved feedback'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await prototype.waitFor("document.querySelector('.fb-state').textContent === 'Saved on this device only'");
  await command("Page.reload", {}, prototype.sessionId);
  await prototype.waitFor("document.readyState === 'complete' && document.querySelector('.fb-state')?.textContent === 'Saved on this device only'");
  for (const id of ["constructor", "toString"]) {
    assert.equal(await prototype.evaluate(`document.querySelector('[data-fb=coaNote][data-coa="${id}"]').value`), "", "Missing notes must not resolve to inherited properties");
    await prototype.evaluate(`var note = document.querySelector('[data-fb=coaNote][data-coa="${id}"]'); note.value = 'Real ${id} note'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
    await prototype.waitFor("document.querySelector('.fb-state').textContent === 'Saved on this device only'");
  }
  await prototype.evaluate(captureClipboard);
  await prototype.evaluate("document.querySelector('#fb-copy').click()");
  assert.equal(await prototype.evaluate("copiedFeedback.findings['SEC-01'].coaNotes.constructor"), "Real constructor note");
  assert.equal(await prototype.evaluate("copiedFeedback.findings['SEC-01'].coaNotes.toString"), "Real toString note");

  const localKey = "audit-report:feedback:" + prototypeReport.title + "@" + (prototypeReport.commit || "");
  const savedBeforeLoad = { "SEC-01": { findingId: "SEC-01", note: "Old saved note", decision: { type: "coa", coaId: "retired" }, coaNotes: { constructor: "Keep untouched option note", retired: "Keep retired note" }, notify: {} } };
  const delayed = await page(pathToFileURL(prototypePath).href, `localStorage.setItem(${JSON.stringify(localKey)}, ${JSON.stringify(JSON.stringify(savedBeforeLoad))});
    window.capabilityWaiters = [];
    window.claude = { use: function () { return new Promise(function (resolve) { capabilityWaiters.push(resolve); }); } };
    window.finishCapabilities = function () { capabilityWaiters.forEach(function (resolve) { resolve(null); }); };`);
  await delayed.waitFor("capabilityWaiters.length === 3");
  await delayed.evaluate(`var note = document.querySelector('[data-fb=note]'); note.focus(); note.value = 'New pending draft'; note.dispatchEvent(new Event('input', { bubbles: true })); note.blur(); note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await delayed.waitFor("document.querySelector('.fb-state').textContent === 'Saving…'");
  await delayed.evaluate("finishCapabilities()");
  await delayed.waitFor("document.querySelector('.fb-state').textContent === 'Saved on this device only'");
  assert.equal(await delayed.evaluate("document.querySelector('[data-fb=note]').value"), "New pending draft", "Hydration must preserve pending edits after blur");
  assert.equal(await delayed.evaluate("document.querySelector('[data-fb=coaNote][data-coa=constructor]').value"), "Keep untouched option note");
  await delayed.evaluate(`var note = document.querySelector('[data-fb=coaNote][data-coa=toString]'); note.value = 'A later edit'; note.dispatchEvent(new Event('change', { bubbles: true }));`);
  await delayed.waitFor("document.querySelector('.fb-state').textContent === 'Saved on this device only'");
  const pendingSaved = await delayed.evaluate(`JSON.parse(localStorage.getItem(${JSON.stringify(localKey)}))['SEC-01']`);
  assert.equal(pendingSaved.note, "New pending draft");
  assert.equal(pendingSaved.decision.coaId, "retired");
  assert.equal(pendingSaved.coaNotes.constructor, "Keep untouched option note");
  assert.equal(pendingSaved.coaNotes.retired, "Keep retired note");
  await delayed.evaluate(captureClipboard);
  await delayed.evaluate("document.querySelector('#fb-copy').click()");
  assert.equal(await delayed.evaluate("copiedFeedback.findings['SEC-01'].note"), "New pending draft");
  assert.deepEqual(exceptions, []);
  console.log("Browser checks passed: shared styles and theme, disk saves and recovery, hash-collision saves, retained feedback without options, standalone export after storage failure, inherited option identifiers, pending local edits, desktop and phone layouts.");
} finally {
  await viewer.close();
  await specViewer.close();
  const exited = browser.exitCode === null ? once(browser, "close") : Promise.resolve();
  if (browser.exitCode === null) {
    if (socket && socket.readyState === WebSocket.OPEN) {
      try { await command("Browser.close"); } catch { browser.kill(); }
    } else browser.kill();
  }
  await exited;
  if (socket) socket.close();
  rmSync(join(directory, "browser-profile"), { recursive: true, force: true, maxRetries: 10, retryDelay: 100 });
}
