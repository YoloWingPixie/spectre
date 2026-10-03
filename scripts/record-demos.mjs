import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { launchBrowser, launchViewer, validateBrowser } from "./browser.mjs";

validateBrowser(process.argv[2]);
const root = fileURLToPath(new URL("..", import.meta.url));
const directory = mkdtempSync(join(tmpdir(), "spectre-demos-"));
const output = resolve(process.argv[4]);
const binary = process.argv[3];
const spec = join(directory, "spec");
const project = join(directory, "project");
const environment = { ...process.env, AUDIT_REPORT_HOME: join(directory, "index"), XDG_CACHE_HOME: join(directory, "cache") };
const width = 1024, height = 768;
function run(executable, args) {
  const result = spawnSync(executable, args, { cwd: directory, env: environment, encoding: "utf8", timeout: 60000 });
  assert.equal(result.status, 0, result.stderr || result.error?.message);
  return result.stdout;
}

let browser, specViewer, auditViewer, completed = false;
try {
  browser = await launchBrowser(process.argv[2], directory);
  cpSync(join(root, "internal/spec/testdata/atc-spec"), spec, { recursive: true });
  mkdirSync(project);
  const report = JSON.parse(readFileSync(join(root, "docs/audit/templates/report.starter.json"), "utf8"));
  report.title = "Example code audit";
  report.project = "demo";
  report.repoRoot = "/workspace/demo";
  report.scope = "Example findings prepared by an agent. Review the evidence, choose a response, and leave a note.";
  report.notes = "";
  report.openQuestions = [];
  report.clean = [];
  delete report.commit;
  delete report.branch;
  const source = join(directory, "report.json");
  writeFileSync(source, JSON.stringify(report));
  run(binary, ["audit", "init", "--project", project, "--source", source]);
  specViewer = await launchViewer(binary, ["spec", "-dir", spec, "-port", "0"], { env: environment });
  auditViewer = await launchViewer(binary, ["audit", "view", "--project", project, "--port", "0"], { env: environment });
  const { command, page } = browser;

  async function recording(url, name) {
    const tab = await page(url, 'localStorage.setItem("spectre.theme", "light")');
    await command("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 1, mobile: false }, tab.sessionId);
    const frames = [];
    async function frame(seconds = 1.5) {
      await tab.evaluate("document.fonts.ready");
      assert.equal(await tab.evaluate("document.documentElement.scrollWidth > innerWidth"), false, `${name} overflows`);
      const { data } = await command("Page.captureScreenshot", { format: "png", captureBeyondViewport: false }, tab.sessionId);
      const path = join(directory, `${name}-${frames.length}.png`);
      writeFileSync(path, Buffer.from(data, "base64"));
      frames.push({ path, seconds });
    }
    async function focus(selector) {
      await tab.evaluate(`document.querySelector(${JSON.stringify(selector)}).scrollIntoView({block:"center",behavior:"instant"})`);
      const point = await tab.evaluate(`(function(){var r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2}})()`);
      await command("Input.dispatchMouseEvent", { type: "mouseMoved", ...point }, tab.sessionId);
      return point;
    }
    async function click(selector) {
      const point = await focus(selector);
      await command("Input.dispatchMouseEvent", { type: "mousePressed", button: "left", clickCount: 1, ...point }, tab.sessionId);
      await command("Input.dispatchMouseEvent", { type: "mouseReleased", button: "left", clickCount: 1, ...point }, tab.sessionId);
    }
    async function type(selector, text) {
      await click(selector);
      for (let i = 0; i < text.length; i += 3) {
        await command("Input.insertText", { text: text.slice(i, i + 3) }, tab.sessionId);
        await frame(0.16);
      }
      await frame(1);
    }
    function encode() {
      const list = join(directory, `${name}.txt`);
      writeFileSync(list, frames.map(({ path, seconds }) => `file '${path}'\nduration ${seconds}\n`).join("") + `file '${frames.at(-1).path}'\n`);
      const gif = join(directory, `${name}.gif`);
      run("ffmpeg", ["-v", "error", "-y", "-f", "concat", "-safe", "0", "-i", list, "-filter_complex", "split[a][b];[a]palettegen=max_colors=128[p];[b][p]paletteuse=dither=bayer:bayer_scale=3", "-fps_mode", "vfr", "-loop", "0", gif]);
      const metadata = JSON.parse(run("ffprobe", ["-v", "error", "-show_streams", "-show_format", "-of", "json", gif]));
      assert.equal(metadata.streams[0].width, width);
      assert.equal(metadata.streams[0].height, height);
      assert.ok(Number(metadata.streams[0].nb_frames) > 5, "Recording must contain multiple workflow frames");
      assert.ok(Number(metadata.format.duration) >= 8, "Recording must show the workflow, not a single frame");
      assert.ok(statSync(gif).size < 4 * 1024 * 1024, "README GIF exceeds 4 MiB");
      return { name, gif, poster: frames.at(-1).path, seconds: Number(metadata.format.duration), bytes: statSync(gif).size };
    }
    return { ...tab, frame, focus, click, type, encode };
  }

  const specification = await recording(specViewer.url, "spec-review");
  await specification.frame(2);
  await specification.type('input[name="q"]', "callsign");
  await command("Input.dispatchKeyEvent", { type: "keyDown", key: "Enter", code: "Enter", windowsVirtualKeyCode: 13, text: "\r" }, specification.sessionId);
  await command("Input.dispatchKeyEvent", { type: "keyUp", key: "Enter", code: "Enter", windowsVirtualKeyCode: 13 }, specification.sessionId);
  await specification.waitFor("location.pathname === '/search' && !!document.querySelector('.res')");
  await specification.frame(2);
  await specification.click('.res a[href="/id/ATC-FR-IDN-001"]');
  await specification.waitFor("location.pathname === '/id/ATC-FR-IDN-001' && !!document.querySelector('form.fb')");
  await specification.frame(2);
  const specNote = "Please add an example for a repeated unidentified call.";
  await specification.type('form.fb input[name="text"]', specNote);
  await specification.click('form.fb button[value="Change"]');
  await specification.waitFor(`!!document.querySelector('.hist') && document.querySelector('.hist').textContent.includes(${JSON.stringify(specNote)})`);
  await specification.focus('.hist');
  await specification.frame(3);
  const savedSpec = readFileSync(join(spec, "review/feedback.md"), "utf8");
  assert.ok(savedSpec.includes(specNote) && savedSpec.includes("Change"));
  const specRecording = specification.encode();

  const audit = await recording(auditViewer.url, "audit-review");
  await audit.waitFor("document.querySelector('#fb-hint').textContent.includes('The agent reads them when it resumes.')");
  await audit.frame(2);
  await audit.focus('details.row');
  await audit.frame(1);
  await audit.click('details.row summary');
  await audit.frame(2);
  await audit.focus('.coas');
  await audit.frame(2.5);
  await audit.click('input[data-coa="expire-one-hour"]');
  await audit.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  await audit.frame(1.5);
  const auditNote = "Use the one-hour expiry and cover expired links in tests.";
  await audit.type('[data-fb="note"]', auditNote);
  await audit.click('.fb-dec legend');
  await audit.waitFor("document.querySelector('.fb-state').textContent === 'Saved to the audit folder'");
  await audit.frame(3);
  const resumed = JSON.parse(run(binary, ["audit", "resume", "--project", project]));
  assert.equal(resumed.feedback["SEC-01"].decision.coaId, "expire-one-hour");
  assert.equal(resumed.feedback["SEC-01"].note, auditNote);
  const auditRecording = audit.encode();
  assert.deepEqual(browser.exceptions, []);
  mkdirSync(output, { recursive: true });
  for (const recording of [specRecording, auditRecording]) {
    copyFileSync(recording.gif, join(output, `${recording.name}.gif`));
    copyFileSync(recording.poster, join(output, `${recording.name}.png`));
    console.log(`${recording.name}: ${recording.seconds}s, ${recording.bytes} bytes; saved feedback verified`);
  }
  completed = true;
} finally {
  const cleanup = await Promise.allSettled([specViewer?.close(), auditViewer?.close(), browser?.close()]);
  const errors = cleanup.filter(result => result.status === "rejected").map(result => result.reason);
  if (completed) rmSync(directory, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 });
  else console.error(`Recording failed; diagnostic frames remain in ${directory}`);
  if (errors.length && completed) throw new AggregateError(errors, "Recording cleanup failed");
  for (const error of errors) console.error(error);
}
