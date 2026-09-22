import assert from "node:assert/strict";
import { spawn, execFileSync } from "node:child_process";
import { writeFile, mkdir, readFile, mkdtemp } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { request } from "playwright";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const binary = process.env.MADI_DESKTOP_BINARY;
if (!binary)
  throw new Error(
    "Set MADI_DESKTOP_BINARY to the native custom-protocol build",
  );
const driver = process.env.MADI_TAURI_DRIVER || "tauri-driver";
const driverArgs = ["--port", "4444", "--native-port", "4445"];
if (process.env.MADI_WEBKIT_DRIVER)
  driverArgs.push("--native-driver", process.env.MADI_WEBKIT_DRIVER);
const child = process.env.MADI_DESKTOP_SYSROOT
  ? spawn(process.execPath, ["tests/native-run.mjs", driver, ...driverArgs], {
      stdio: "inherit",
      env: process.env,
    })
  : spawn(driver, driverArgs, {
      stdio: "inherit",
      env: { ...process.env, TAURI_WEBVIEW_AUTOMATION: "true" },
    });
for (const signal of ["SIGINT", "SIGTERM"])
  process.on(signal, () => {
    child.kill(signal);
    process.exit(130);
  });
let sid;
const webdriver = "http://127.0.0.1:4444";
async function command(route, method = "GET", data) {
  const response = await fetch(webdriver + route, {
    method,
    headers: { "Content-Type": "application/json" },
    body: data === undefined ? undefined : JSON.stringify(data),
    signal: AbortSignal.timeout(60000),
  });
  const result = await response.json();
  if (result.value?.error) throw new Error(JSON.stringify(result.value));
  return result.value;
}
async function wait(fn, ms = 30000) {
  const start = Date.now();
  while (Date.now() - start < ms) {
    try {
      const result = await fn();
      if (result) return result;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error("Timed out waiting for native UI");
}
const admin = await request.newContext({
  baseURL: base,
  extraHTTPHeaders: { "X-Madi-Request": "1" },
});
async function api(route, method = "GET", data) {
  const res = await admin.fetch("/api/v1" + route, { method, data });
  assert.ok(res.ok(), `${route}: ${res.status()} ${await res.text()}`);
  return res.json();
}
let key;
const execute = (script, args = []) =>
  command(`/session/${sid}/execute/sync`, "POST", { script, args });
const find = (selector) =>
  command(`/session/${sid}/element`, "POST", {
    using: "css selector",
    value: selector,
  });
const click = async (selector) => {
  const element = await find(selector);
  await command(
    `/session/${sid}/element/${Object.values(element)[0]}/click`,
    "POST",
    {},
  );
};
const fill = async (selector, text) => {
  const element = await find(selector),
    id = Object.values(element)[0];
  await command(`/session/${sid}/element/${id}/clear`, "POST", {});
  await command(`/session/${sid}/element/${id}/value`, "POST", { text });
};
const text = () => execute("return document.body.innerText");
const nativeKey = (...args) =>
  execFileSync(process.env.MADI_XDOTOOL || "xdotool", args, {
    env: {
      ...process.env,
      ...(process.env.MADI_DESKTOP_SYSROOT
        ? {
            LD_LIBRARY_PATH:
              process.env.MADI_DESKTOP_SYSROOT + "/usr/lib/x86_64-linux-gnu",
          }
        : {}),
    },
    stdio: ["ignore", "pipe", "pipe"],
  }).toString();
async function chooseFile(filename) {
  let picker;
  try {
    picker = await wait(
      () =>
        nativeKey(
          "search",
          "--onlyvisible",
          "--name",
          "Open|Save|열기|저장|Select|선택|가져오기",
        )
          .trim()
          .split("\n")
          .at(-1),
      10000,
    );
  } catch (error) {
    console.error(
      "Native windows:",
      nativeKey("search", "--name", ".*")
        .trim()
        .split("\n")
        .map((id) => [id, nativeKey("getwindowname", id)]),
    );
    throw error;
  }
  nativeKey("windowfocus", "--sync", picker);
  const dialogTitle = nativeKey("getwindowname", picker).trim();
  console.log("Native file dialog:", dialogTitle);
  nativeKey("key", "--clearmodifiers", "ctrl+l");
  await new Promise((resolve) => setTimeout(resolve, 150));
  nativeKey("key", "--clearmodifiers", "ctrl+a");
  nativeKey("type", "--clearmodifiers", "--delay", "2", filename);
  nativeKey("key", "--clearmodifiers", "Return");
  await new Promise((resolve) => setTimeout(resolve, 300));
  nativeKey(
    "key",
    "--clearmodifiers",
    /save|저장/i.test(dialogTitle) ? "alt+s" : "alt+o",
  );
}
async function shot(name) {
  await mkdir("docs/screenshots", { recursive: true });
  const png = await command(`/session/${sid}/screenshot`);
  await writeFile(
    "docs/screenshots/" + name + ".png",
    Buffer.from(png, "base64"),
  );
}
try {
  await wait(() =>
    fetch(webdriver + "/status").then((response) => response.ok),
  );
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const [ws] = await api("/workspaces");
  key = await api("/keys", "POST", {
    name: "데스크톱 실제 웹뷰 검증",
    workspace_id: ws.id,
    scopes: ["document:read", "document:write"],
    expires_in_days: 1,
    rate_limit: 1000,
  });
  const session = await command("/session", "POST", {
    capabilities: {
      alwaysMatch: {
        "tauri:options": { application: binary, args: ["madi://capture"] },
      },
    },
  });
  sid = session.sessionId;
  await wait(async () => (await text()).includes("서버 연결"));
  assert.match(await text(), /madi v0.1.0/);
  await wait(async () => (await text()).includes("문서 링크를 여시겠습니까?"));
  await execute(
    "[...document.querySelectorAll('button')].find(x=>x.textContent==='취소').click()",
  );
  assert.ok(!(await text()).includes("문서 링크를 여시겠습니까?"));
  console.log(
    "PASS native startup deep link requires explicit confirmation and can be cancelled",
  );
  await fill("#server", base);
  await fill("#token", key.token);
  await click('form button[type="submit"],form button:not([type])');
  await wait(async () => (await text()).includes("사내 서버에 연결했습니다"));
  console.log(
    "PASS native WebKit local shell + actual Rust TLS/API bridge login",
  );
  await shot("desktop");
  await fill("#title", "데스크톱 실제 빠른 기록");
  await fill(
    "#text",
    "# 네이티브 기록\n\n로컬 React와 Rust 권한 경계를 검증했습니다.",
  );
  await click("section.card .actions button");
  await wait(async () => (await text()).includes("개인 인박스에 저장했습니다"));
  assert.ok(
    (await api("/captures?workspace_id=" + ws.id)).some(
      (doc) => doc.title === "데스크톱 실제 빠른 기록",
    ),
  );
  console.log("PASS native quick capture reaches actual private inbox");
  const fixtures = await mkdtemp(path.join(os.tmpdir(), "madi-desktop-files-"));
  const source = path.join(fixtures, "native-import.md");
  const exported = path.join(fixtures, "native-export.md");
  const markdown = "# 네이티브 파일 테스트\n\nOS 파일 선택만 허용합니다.\n";
  await writeFile(source, markdown);
  const openingFile = execute(
    "[...document.querySelectorAll('button')].find(x=>x.textContent.includes('로컬 Markdown 파일 가져오기')).click()",
  );
  await chooseFile(source);
  await openingFile;
  await wait(
    async () =>
      (await execute("return document.querySelector('#text')?.value")) ===
      markdown,
  );
  const savingFile = execute(
    "[...document.querySelectorAll('button')].find(x=>x.textContent.includes('Markdown 파일로 내보내기')).click()",
  );
  await chooseFile(exported);
  await savingFile;
  await wait(
    async () => (await readFile(exported, "utf8").catch(() => "")) === markdown,
  );
  console.log(
    "PASS actual OS file chooser import/export preserves UTF-8 Markdown",
  );
  await execute(
    "[...document.querySelectorAll('nav button')].find(x=>x.textContent==='오프라인 보관함').click()",
  );
  await wait(async () => (await text()).includes("암호화 오프라인 보관함"));
  assert.ok(await execute("return !!crypto.subtle && isSecureContext"));
  await fill("#passphrase", "Native-vault-password-2026!");
  await click("section.card form button");
  await wait(async () => (await text()).includes("보관함 잠그기"));
  console.log("PASS native WebCrypto encrypted vault available and unlocks");
  await shot("desktop-offline");
  nativeKey("key", "--clearmodifiers", "ctrl+shift+space");
  await wait(
    async () => !!(await execute("return !!document.querySelector('#title')")),
  );
  console.log("PASS actual operating-system global quick-capture shortcut");
  await execute(
    "[...document.querySelectorAll('nav button')].find(x=>x.textContent==='연결과 기기 설정').click()",
  );
  await wait(async () => (await text()).includes("서버 연결"));
  await click("header button");
  const handles = await wait(async () => {
    const values = await command(`/session/${sid}/window/handles`);
    return values.length > 1 ? values : null;
  });
  const main = await command(`/session/${sid}/window`);
  const remote = handles.find((handle) => handle !== main);
  await command(`/session/${sid}/window`, "POST", { handle: remote });
  await wait(async () =>
    (await execute("return location.href")).startsWith(base),
  );
  const denied = await command(`/session/${sid}/execute/async`, "POST", {
    script:
      "const done=arguments[arguments.length-1];if(!window.__TAURI_INTERNALS__){done('isolated');return;}window.__TAURI_INTERNALS__.invoke('connection_status').then(()=>done('UNEXPECTED_ALLOWED'),e=>done(String(e)));",
    args: [],
  });
  assert.notEqual(denied, "UNEXPECTED_ALLOWED");
  console.log(
    "PASS remote service webview cannot invoke native credentials/files commands",
  );
  await command(`/session/${sid}/window`, "POST", { handle: main });
  await execute(
    "[...document.querySelectorAll('button')].find(x=>x.textContent==='연결 해제·키 삭제').click()",
  );
  await wait(async () =>
    (await text()).includes("세션 키와 키체인 항목을 삭제"),
  );
  assert.equal((await command(`/session/${sid}/window/handles`)).length, 1);
  console.log(
    "PASS native disconnect closes remote window and locks device vault",
  );
  console.log("All native desktop checks passed.");
} catch (error) {
  if (sid)
    console.error("NATIVE UI:", await text().catch(() => "(unavailable)"));
  throw error;
} finally {
  if (key) await api("/keys/" + key.key.id, "DELETE").catch(() => {});
  if (sid) await command(`/session/${sid}`, "DELETE").catch(() => {});
  child.kill("SIGTERM");
  await admin.dispose();
}
