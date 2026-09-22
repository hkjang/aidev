import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

// Read-only supplementary photography of existing successful QA fixtures.
// No service configuration, document, grant or provider is created or changed.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const out = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 }, locale: "ko-KR",
  timezoneId: "Asia/Seoul", reducedMotion: "reduce",
});
const page = await context.newPage(), errors = [], captures = [];
page.on("pageerror", e => errors.push(e.message));
await context.route("**/*", route => {
  const u = new URL(route.request().url());
  if (u.origin !== new URL(base).origin && !["blob:", "data:"].includes(u.protocol)) {
    errors.push("Unexpected external asset");
    return route.abort();
  }
  if (route.request().method() !== "GET" && u.pathname.startsWith("/api/")) {
    errors.push(`Unexpected browser mutation: ${route.request().method()} ${u.pathname}`);
    return route.abort();
  }
  return route.continue();
});
async function api(endpoint, method = "GET", data) {
  const r = await context.request.fetch(base + "/api/v1" + endpoint, {
    method, data, headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${endpoint}: HTTP ${r.status()}`);
  return r.json();
}
async function open(workspace, route, title) {
  await page.goto(base + "/app");
  if (workspace) await page.evaluate(id => localStorage.setItem("madi.workspace", id), workspace.id);
  await page.goto(base + route);
  await page.getByRole("heading", { name: title, exact: true }).waitFor();
  await page.locator(".loading").waitFor({ state: "detached" });
  assert.equal(await page.locator(".notice.error").count(), 0);
}
async function shot(name, route) {
  await page.evaluate(async () => { await document.fonts.ready; window.scrollTo(0, 0); });
  await page.screenshot({ path: path.join(out, name + ".png"), fullPage: false,
    animations: "disabled", style: ".toast{visibility:hidden!important}" });
  captures.push({ file: name + ".png", route, viewport: "1512×1080" });
}
try {
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const workspaces = await api("/workspaces");
  const latest = text => workspaces.filter(w => w.name.includes(text)).sort((a, b) => b.name.localeCompare(a.name))[0];
  const canvas = latest("캔버스"), plugins = latest("플러그인 격리"), sources = latest("데이터 소스 UI 검증");
  assert.ok(canvas && plugins && sources, "Existing successful QA fixture workspaces are required");
  await open(canvas, "/app/canvases", "캔버스");
  await page.locator(".canvas-list-card").first().waitFor();
  await shot("canvas-list", "/app/canvases");
  await open(canvas, "/app/canvases?trash=true", "캔버스 휴지통");
  await shot("canvas-trash", "/app/canvases?trash=true");
  await open(plugins, "/app/plugins", "플러그인");
  await page.getByRole("link", { name: "플러그인 열기", exact: true }).first().waitFor();
  await shot("plugin-list", "/app/plugins");
  await open(sources, "/app/data-sources", "외부 데이터 소스");
  await page.getByRole("button", { name: "테이블 보기", exact: true }).first().waitFor();
  await shot("data-source-list", "/app/data-sources");
  await open(null, "/admin/audit", "감사 로그");
  await shot("admin-audit", "/admin/audit");
  assert.deepEqual(errors, []);
  await mkdir("test-results/regression-shared", { recursive: true });
  await writeFile("test-results/regression-shared/route-gaps.json", JSON.stringify({
    ok: true, checked_at: new Date().toISOString(), read_only: true, captures, errors,
  }, null, 2) + "\n");
  console.log(JSON.stringify({ ok: true, read_only: true, screenshots: captures, errors }));
} finally { await browser.close(); }
