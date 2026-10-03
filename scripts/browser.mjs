import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { openSync, readSync, closeSync } from "node:fs";
import { join } from "node:path";
import { once } from "node:events";
import { setTimeout as delay } from "node:timers/promises";

export async function launchViewer(binary, args, options) {
  const server = spawn(binary, args, { ...options, stdio: ["ignore", "pipe", "pipe"] });
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
    server.stderr.on("data", () => {
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

export function validateBrowser(executable) {
  assert.equal(process.platform, "linux", "Browser checks require Linux and Linux Chromium.");
  const browserFile = openSync(executable, "r");
  try {
    const header = Buffer.alloc(4);
    readSync(browserFile, header, 0, header.length, 0);
    assert.notEqual(header.subarray(0, 2).toString(), "MZ", "BROWSER must point to Linux Chromium, not Windows Chrome.");
  } finally { closeSync(browserFile); }
}

export async function launchBrowser(executable, directory) {
  validateBrowser(executable);
  const browser = spawn(executable, ["--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
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
    const navigation = await command("Page.navigate", { url }, sessionId);
    assert.equal(navigation.errorText, undefined, `Navigation failed for ${url}: ${navigation.errorText}`);
    await waitFor("document.readyState === 'complete' && !!document.querySelector('h1')");
    return { targetId, sessionId, evaluate, waitFor };
  }

  async function close() {
    const exited = browser.exitCode === null ? once(browser, "close") : Promise.resolve();
    if (browser.exitCode === null) {
      if (socket && socket.readyState === WebSocket.OPEN) {
        try { await command("Browser.close"); } catch { browser.kill(); }
      } else browser.kill();
    }
    await exited;
    if (socket) socket.close();
    for (const request of pending.values()) { clearTimeout(request.timer); request.reject(new Error("Browser closed")); }
    pending.clear();
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
    return { command, page, exceptions, close };
  } catch (error) {
    await close();
    throw error;
  }
}
