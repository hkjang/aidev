import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { createPluginArchive } from "../sdk/plugins/package.mjs";

// Only run on a disposable QA instance: installs/replaces the example plugin,
// grants workspace capabilities and creates test documents.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const out = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  timezoneId: "Asia/Seoul",
  reducedMotion: "reduce",
});
const page = await context.newPage(),
  issues = [];
page.on("pageerror", (error) => issues.push(error.message));
page.on("console", (message) => {
  if (
    message.type() === "error" &&
    !message.text().includes("401 (Unauthorized)")
  )
    issues.push(message.text());
});
await context.route("**/*", (route) => {
  const url = new URL(route.request().url());
  if (
    url.origin !== new URL(base).origin &&
    !["data:", "blob:"].includes(url.protocol)
  ) {
    issues.push(`External asset: ${url.href}`);
    return route.abort();
  }
  return route.continue();
});
async function api(endpoint, method = "GET", data) {
  const response = await context.request.fetch(`${base}/api/v1${endpoint}`, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${endpoint} ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  const close = page.locator(".toast .icon-button");
  if (await close.count()) await close.click().catch(() => {});
  await page.screenshot({
    path: path.join(out, name + ".png"),
    fullPage: true,
    animations: "disabled",
  });
}
try {
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const workspace = await api("/workspaces", "POST", {
      name: "플러그인 격리 검증 " + Date.now(),
    }),
    wid = workspace.id;
  await context.addInitScript(
    (id) => { if (window === window.top) localStorage.setItem("madi.workspace", id); },
    wid,
  );
  const installed = await api("/admin/plugins"),
    replacing = installed.plugins.some((p) => p.id === "knowledge-helper");
  await page.goto(base + "/admin/plugins");
  await page
    .getByRole("heading", { name: "플러그인 관리", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "ZIP 설치", exact: true }).click();
  let dialog = page.getByRole("dialog", {
    name: "오프라인 플러그인 설치",
    exact: true,
  });
  await dialog.getByLabel("플러그인 ZIP", { exact: true }).setInputFiles({
    name: "knowledge-helper-1.0.0.zip",
    mimeType: "application/zip",
    buffer: await createPluginArchive(path.resolve("sdk/plugins/example")),
  });
  if (replacing)
    await dialog
      .getByLabel("기존 플러그인 업데이트 확인", { exact: true })
      .fill("REPLACE");
  await dialog
    .getByRole("button", { name: "검토한 ZIP 설치", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await page
    .getByRole("heading", { name: "지식 도우미", exact: true })
    .waitFor();
  await shot("admin-plugins");
  await page.goto(base + "/app/plugins?manage=1");
  await page.getByRole("button", { name: "권한 검토", exact: true }).click();
  dialog = page.getByRole("dialog", {
    name: "워크스페이스 플러그인 권한",
    exact: true,
  });
  await dialog.getByLabel("사용 상태", { exact: true }).selectOption("enabled");
  for (const checkbox of await dialog.getByRole("checkbox").all())
    await checkbox.check();
  await shot("plugin-permissions");
  await dialog.getByRole("button", { name: "권한 저장", exact: true }).click();
  await dialog.waitFor({ state: "hidden" });
  await page.goto(base + "/app/plugins");
  await page.getByRole("link", { name: "플러그인 열기", exact: true }).click();
  await page.waitForURL("**/app/plugins/knowledge-helper");
  let frame = page.frameLocator('iframe[title="지식 도우미 플러그인"]');
  await frame
    .getByRole("heading", { name: "팀의 지식을 한곳에서", exact: true })
    .waitFor({ timeout: 15000 });
  assert.equal(
    await page.locator("iframe").getAttribute("sandbox"),
    "allow-scripts",
  );
  await frame
    .getByRole("textbox", { name: "문서 검색어", exact: true })
    .fill("madi");
  await frame.getByRole("button", { name: "문서 찾기", exact: true }).click();
  await frame.getByText(/접근 가능한 문서 \d+개를 찾았습니다/).waitFor();
  const before = await api(`/documents?workspace_id=${wid}&limit=2000`);
  await frame
    .getByRole("button", { name: "오늘의 기록 만들기", exact: true })
    .click();
  await frame.getByText("개인 문서에 오늘의 기록을 만들었습니다.").waitFor();
  const after = await api(`/documents?workspace_id=${wid}&limit=2000`);
  assert.equal(after.length, before.length + 1);
  assert.ok(
    after.some(
      (doc) =>
        doc.visibility === "private" && doc.title.includes("오늘의 기록"),
    ),
  );
  await page.reload();
  frame = page.frameLocator('iframe[title="지식 도우미 플러그인"]');
  await frame
    .getByRole("heading", { name: "팀의 지식을 한곳에서", exact: true })
    .waitFor();
  await frame.getByRole("button", { name: "문서 찾기", exact: true }).click();
  await frame.getByText(/접근 가능한 문서 \d+개를 찾았습니다/).waitFor();
  await shot("plugins");
  const chooserPromise = page.waitForEvent("filechooser");
  await page
    .getByRole("button", { name: "Markdown 가져오기", exact: true })
    .click();
  const chooser = await chooserPromise;
  await chooser.setFiles({
    name: "플러그인으로 가져온 문서.md",
    mimeType: "text/markdown",
    buffer: Buffer.from(
      "# 가져오기 검증\n\n원문과 링크를 보존합니다.\n\n[[madi 시작 가이드]]",
    ),
  });
  dialog = page.getByRole("dialog", {
    name: "플러그인으로 문서 가져오기",
    exact: true,
  });
  await dialog
    .getByText("플러그인으로 가져온 문서.md", { exact: true })
    .waitFor();
  await dialog
    .getByRole("button", { name: "가져오기 실행", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "확장 기능 실행 결과",
    exact: true,
  });
  await dialog.getByText(/created_document/).waitFor();
  await dialog
    .getByRole("button", { name: "닫기", exact: true })
    .last()
    .click();
  await page
    .getByRole("button", { name: "문서 목록 내보내기", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "내보내기 결과 확인",
    exact: true,
  });
  await dialog.waitFor();
  const downloadPromise = page.waitForEvent("download");
  await dialog.getByRole("button", { name: "다운로드", exact: true }).click();
  const download = await downloadPromise;
  assert.equal(download.suggestedFilename(), "madi-knowledge.md");
  const blockDoc = await api("/documents", "POST", {
    workspace_id: wid,
    title: "플러그인 블록 · 팀의 결정",
    markdown:
      "# 플러그인 블록\n\n```madi-plugin\n" +
      JSON.stringify({
        plugin_id: "knowledge-helper",
        block_id: "summary-card",
        data: {
          title: "팀의 결정",
          summary: "플러그인 데이터도 Markdown으로 보존합니다.",
        },
      }) +
      "\n```",
  });
  await page.goto(`${base}/app/documents/${blockDoc.id}?mode=preview`);
  await page
    .getByRole("button", { name: "플러그인 블록 실행", exact: true })
    .waitFor();
  assert.equal(await page.locator("iframe").count(), 0);
  await page
    .getByRole("button", { name: "플러그인 블록 실행", exact: true })
    .click();
  frame = page.frameLocator('iframe[title="지식 도우미 플러그인"]');
  await frame
    .getByRole("heading", { name: "팀의 결정", exact: true })
    .waitFor();
  await frame.getByText("플러그인 데이터도 Markdown으로 보존합니다.").waitFor();
  await shot("plugin-block");
  await page.goto(base + "/app/plugins/knowledge-helper");
  frame = page.frameLocator('iframe[title="지식 도우미 플러그인"]');
  await frame
    .getByRole("heading", { name: "팀의 지식을 한곳에서", exact: true })
    .waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  await shot("mobile-plugins");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  assert.deepEqual(issues, []);
  console.log(
    "PASS: offline ZIP install, granular native-select grant, isolated Worker UI, real document write/import/export, route refresh, explicit block execution, mobile and console checks",
  );
} finally {
  await browser.close();
}
