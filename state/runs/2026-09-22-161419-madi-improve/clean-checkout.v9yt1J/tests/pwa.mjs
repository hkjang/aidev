import { chromium } from "playwright";
import { expect } from "playwright/test";
import assert from "node:assert/strict";
import path from "node:path";
import { mkdir } from "node:fs/promises";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const output = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(output, { recursive: true });
const browser = await chromium.launch();
const context = await browser.newContext({
  viewport: { width: 1512, height: 1100 },
  locale: "ko-KR",
  reducedMotion: "reduce",
});
const page = await context.newPage();
const errors = [];
page.on("pageerror", (error) => errors.push(error.message));
async function api(route, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + route, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${route} ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
const password = "Device-vault-password-2026!";
async function unlock() {
  await page.getByLabel("기기 보관함 암호", { exact: true }).fill(password);
  await page
    .getByRole("button", { name: "보관함 잠금 해제", exact: true })
    .click();
  await page
    .getByRole("button", { name: "보관함 잠그기", exact: true })
    .waitFor();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const [workspace] = await api("/workspaces");
  const doc = await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "오프라인에서 이어 쓰는 지식",
    markdown: "# 오프라인 기록\n\n암호화 테스트 원문 SECRET_OFFLINE_2026",
  });
  await page.goto(base + "/app/devices");
  await page
    .getByRole("heading", { name: "기기와 오프라인", exact: true })
    .waitFor();
  await page.getByLabel("기기 보관함 암호", { exact: true }).fill(password);
  await page
    .getByRole("button", { name: "암호화 보관함 만들기", exact: true })
    .click();
  await page
    .getByRole("button", { name: "보관함 잠그기", exact: true })
    .waitFor();
  await page
    .getByLabel("오프라인에 보관할 문서", { exact: true })
    .selectOption(doc.id);
  await page.getByRole("button", { name: "문서 보관", exact: true }).click();
  await page
    .locator(".offline-record")
    .filter({ hasText: doc.title })
    .waitFor();
  const storage = await page.evaluate(async () => {
    const db = await new Promise((resolve) => {
      const req = indexedDB.open("madi-offline-v1");
      req.onsuccess = () => resolve(req.result);
    });
    return new Promise((resolve) => {
      const tx = db.transaction("records");
      const req = tx.objectStore("records").getAll();
      req.onsuccess = () =>
        resolve(
          req.result.map((record) => ({
            keys: Object.keys(record),
            data: new TextDecoder().decode(record.data),
          })),
        );
    });
  });
  assert.equal(storage.length, 1);
  assert.deepEqual(storage[0].keys.sort(), ["data", "id", "iv"]);
  assert.ok(!storage[0].data.includes("SECRET_OFFLINE_2026"));
  console.log(
    "PASS explicit offline selection encrypts title/body in IndexedDB",
  );
  // waitForFunction tests a predicate's truthiness; a Promise is already truthy.
  // Poll an awaited evaluation so activation and the cache really complete.
  await expect.poll(() => page.evaluate(async () => {
    const reg = await navigator.serviceWorker.getRegistration();
    return reg?.active?.state === "activated";
  }), { timeout: 60000 }).toBe(true);
  await page.screenshot({
    path: path.join(output, "devices.png"),
    fullPage: true,
    animations: "disabled",
  });
  const cachePaths = await page.evaluate(async () => {
    const list = [];
    for (const name of await caches.keys())
      for (const req of await (await caches.open(name)).keys())
        list.push(new URL(req.url).pathname);
    return list;
  });
  assert.ok(cachePaths.includes("/offline.html"), JSON.stringify({cachePaths, state: await page.evaluate(async () => ({url: location.href, registrations: (await navigator.serviceWorker.getRegistrations()).map(r => ({script:r.active?.scriptURL,state:r.active?.state})), caches: await caches.keys()}))}));
  assert.ok(
    !cachePaths.some((value) => /^\/(api|auth|attachments|plugin)/.test(value)),
  );
  console.log(
    "PASS service worker caches static shell only, never API or documents",
  );
  await context.setOffline(true);
  await page.goto(base + "/app/devices");
  await page
    .getByRole("heading", { name: "madi 기기 보관함", exact: true })
    .waitFor();
  await unlock();
  await page.getByRole("button", { name: "열기", exact: true }).click();
  await page
    .getByLabel("Markdown 기기 초안", { exact: true })
    .fill("# 오프라인 초안\n\n연결 없이 작성한 실제 변경 사항.");
  await page
    .getByRole("button", { name: "기기 초안 저장", exact: true })
    .click();
  await page
    .getByRole("status")
    .filter({ hasText: "암호화해 저장했습니다" })
    .waitFor();
  await page
    .getByRole("button", { name: "빠른 임시 기록", exact: true })
    .click();
  const modal = page.getByRole("dialog", {
    name: "빠른 임시 기록",
    exact: true,
  });
  await modal.getByLabel("워크스페이스 ID", { exact: true }).fill(workspace.id);
  await modal
    .getByLabel("기록 제목", { exact: true })
    .fill("연결 없이 떠오른 아이디어");
  await modal
    .getByLabel("기록 내용", { exact: true })
    .fill("이 기록은 온라인 복귀 후 개인 인박스로만 전송합니다.");
  await modal
    .getByRole("button", { name: "기기에 임시 기록 저장", exact: true })
    .click();
  await modal.waitFor({ state: "hidden" });
  await page.screenshot({
    path: path.join(output, "offline-vault.png"),
    fullPage: true,
    animations: "disabled",
  });
  await page.reload();
  await page
    .getByRole("heading", { name: "암호로 잠금 해제", exact: true })
    .waitFor();
  await page
    .getByLabel("기기 보관함 암호", { exact: true })
    .fill("wrong-password");
  await page
    .getByRole("button", { name: "보관함 잠금 해제", exact: true })
    .click();
  await page
    .getByRole("alert")
    .filter({ hasText: "암호가 올바르지" })
    .waitFor();
  await unlock();
  assert.equal(await page.locator(".offline-record").count(), 2);
  console.log(
    "PASS real disconnected navigation, locked reload, wrong password, encrypted draft/capture persistence",
  );
  await context.setOffline(false);
  await page.goto(base + "/app/devices");
  await unlock();
  const draft = page.locator(".offline-record").filter({ hasText: doc.title });
  await draft.getByRole("button", { name: "서버에 전송", exact: true }).click();
  await draft.getByText("읽기용 기기 사본", { exact: true }).waitFor();
  assert.match((await api("/documents/" + doc.id)).markdown, /실제 변경 사항/);
  const capture = page
    .locator(".offline-record")
    .filter({ hasText: "연결 없이 떠오른 아이디어" });
  await capture
    .getByRole("button", { name: "서버에 전송", exact: true })
    .click();
  await capture.waitFor({ state: "hidden" });
  assert.ok(
    (await api("/captures?workspace_id=" + workspace.id)).some(
      (value) => value.title === "연결 없이 떠오른 아이디어",
    ),
  );
  console.log(
    "PASS online explicit permission-checked document and idempotent inbox synchronization",
  );
  await page.goto(base + "/offline.html");
  await unlock();
  await page.getByRole("button", { name: "열기", exact: true }).click();
  await page
    .getByLabel("Markdown 기기 초안", { exact: true })
    .fill("오래된 버전 기준의 오프라인 초안");
  await page
    .getByRole("button", { name: "기기 초안 저장", exact: true })
    .click();
  await page
    .getByRole("status")
    .filter({ hasText: "암호화해 저장했습니다" })
    .waitFor();
  const latest = await api("/documents/" + doc.id);
  await api("/documents/" + doc.id, "PUT", {
    version: latest.version,
    markdown: "다른 사용자의 새로운 서버 내용",
  });
  await page.goto(base + "/app/devices");
  await unlock();
  await page
    .locator(".offline-record")
    .filter({ hasText: doc.title })
    .getByRole("button", { name: "서버에 전송", exact: true })
    .click();
  await page
    .getByRole("alert")
    .filter({ hasText: "자동으로 덮어쓰지 않습니다" })
    .waitFor();
  assert.equal(
    (await api("/documents/" + doc.id)).markdown,
    "다른 사용자의 새로운 서버 내용",
  );
  console.log(
    "PASS version conflict keeps encrypted local draft without overwriting server",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: path.join(output, "mobile-devices.png"),
    fullPage: true,
    animations: "disabled",
  });
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  assert.deepEqual(errors, []);
  console.log("All PWA browser checks passed.");
} finally {
  await browser.close();
}
