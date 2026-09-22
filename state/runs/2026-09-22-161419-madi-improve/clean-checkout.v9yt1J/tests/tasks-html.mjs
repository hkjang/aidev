import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  out = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true }),
  ctx = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await ctx.newPage(),
  issues = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) issues.push(`${r.status()} ${r.url()}`);
});
async function api(endpoint, method = "GET", data) {
  const r = await ctx.request.fetch(`${base}/api/v1${endpoint}`, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${method} ${endpoint}: ${r.status()} ${await r.text()}`);
  return r.json();
}
try {
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
    name: "표 속 체크리스트 검증",
  });
  await ctx.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  const markdown =
    '<table><tr><td colspan="2"><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>첫 점검</p></li><li data-type="taskItem" data-checked="false"><p>두 번째 점검</p></li></ul></td></tr></table>';
  const doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "표 안 운영 점검",
    markdown,
  });
  await page.goto(`${base}/app/tasks`);
  await page
    .getByRole("button", { name: "두 번째 점검 속성 수정", exact: true })
    .click();
  const modal = page.getByRole("dialog");
  await modal.getByLabel("할 일 상태", { exact: true }).selectOption("done");
  await modal.getByLabel("할 일 마감일", { exact: true }).fill("2026-09-20");
  await modal.getByRole("button", { name: "할 일 저장", exact: true }).click();
  await modal.waitFor({ state: "hidden" });
  await page.locator('input[aria-label="두 번째 점검 완료 여부"]:checked').waitFor();
  assert.ok(
    await page
      .getByRole("checkbox", { name: "두 번째 점검 완료 여부", exact: true })
      .isChecked(),
  );
  assert.equal(
    await page
      .getByRole("checkbox", { name: "첫 점검 완료 여부", exact: true })
      .isChecked(),
    false,
  );
  const saved = await api(`/documents/${doc.id}`);
  assert.ok(saved.markdown.includes('colspan="2"'));
  assert.equal((saved.markdown.match(/data-checked="true"/g) || []).length, 1);
  assert.ok(saved.markdown.includes("/app/tasks?task="));
  await page.reload();
  await page
    .getByRole("checkbox", { name: "두 번째 점검 완료 여부", exact: true })
    .waitFor();
  assert.ok(
    await page
      .getByRole("checkbox", { name: "두 번째 점검 완료 여부", exact: true })
      .isChecked(),
  );
  await page.goto(`${base}/app/documents/${doc.id}`);
  await page.getByRole('button',{name:'읽기',exact:true}).click();
  await page.locator(".markdown-content table").waitFor();
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(out, "tasks-html-source.png"),
    fullPage: true,
    animations: "disabled",
  });
  assert.ok(
    await page
      .locator(".markdown-content table input[type=checkbox]")
      .nth(1)
      .isChecked(),
  );
  assert.deepEqual(issues, []);
  console.log(
    "PASS same-line HTML table tasks: exact target, native selectors, preserved table markup, stable task link and document read view",
  );
} catch(error) {
  await page.screenshot({path:path.join(out,'tasks-html-failure.png'),fullPage:true});
  console.error(await page.locator('body').innerText());
  throw error;
} finally {
  await browser.close();
}
