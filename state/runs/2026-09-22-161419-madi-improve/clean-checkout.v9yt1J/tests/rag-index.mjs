import assert from "node:assert/strict";
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import { documentPanel } from "./document-ui.mjs";
const base = process.env.MADI_BASE_URL,
  documentID = process.env.MADI_RAG_DOCUMENT,
  workspaceID = process.env.MADI_RAG_WORKSPACE;
assert.ok(
  base && documentID && workspaceID,
  "Run through TestBrowserRAGIndexConsent isolated PostgreSQL fixture",
);
const out = new URL("../docs/screenshots/", import.meta.url);
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    timezoneId: "Asia/Seoul",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${path}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
const statusPath = `/documents/${documentID}/rag-index`;
async function configure(patch) {
  const saved = await api(`/workspaces/${workspaceID}/search-ai`);
  await api(`/workspaces/${workspaceID}/settings`, "PUT", {
    version: saved.version,
    data: patch,
  });
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    fullPage: false,
    animations: "disabled",
  });
}
const modal = page.getByRole("dialog", { name: "문서 검색 색인", exact: true });
const consent = modal.getByRole("checkbox", {
  name: /위 공급자에게 이 문서의 저장된 내용을 전송/,
});
const start = modal.getByRole("button", {
  name: "동의하고 색인 시작",
  exact: true,
});
async function begin() {
  await consent.check();
  const pending = page.waitForResponse(
    (r) => r.url().endsWith(statusPath) && r.request().method() === "POST",
  );
  await start.click();
  const response = await pending;
  assert.equal(response.status(), 202, await response.text());
  return response.request().postDataJSON();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password:
      process.env.MADI_TEST_PASSWORD || "Integration-Test-Password-2026!",
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspaceID,
  );
  await page.goto(base + `/app/documents/${documentID}?mode=preview`);
  await documentPanel(page, "속성");
  await page.getByRole("button", { name: "AI 검색 색인", exact: true }).click();
  await consent.waitFor();
  assert.equal((await api(statusPath)).grant, null);
  assert.ok(await start.isDisabled());
  assert.equal(await consent.isChecked(), false);
  assert.equal(
    await modal
      .getByRole("checkbox", { name: /이 문서가 변경되면/ })
      .isChecked(),
    false,
  );
  assert.equal(
    await modal
      .getByRole("checkbox", { name: /위 재정렬 공급자에게도/ })
      .isChecked(),
    false,
  );
  assert.ok(
    (await modal.innerText()).includes(
      "‘나만 보기’ 문서도 선택한 공급자에게는 전송됩니다.",
    ),
  );
  assert.ok(!(await modal.innerText()).includes("secret=hidden-query"));
  await shot("rag-index-consent");
  const payload = await begin();
  assert.equal(payload.auto_reindex, false);
  assert.equal(payload.allow_rerank, false);
  assert.equal(payload.consent, true);
  await modal
    .getByText("작업 완료", { exact: true })
    .waitFor({ timeout: 60000 });
  const ready = await api(statusPath);
  assert.equal(ready.index.index_status, "ready");
  assert.ok(ready.index.indexed_chunks > 0);
  await shot("document-rag-index");
  console.log(
    "PASS private document explicit consent, independent default-off grants, safe provider URL and real queued index completion",
  );

  await consent.check();
  await configure({ rag_embedding_model: "changed-browser-model" });
  await modal.getByText(/공급자 설정이 변경되어 재동의가 필요합니다/).waitFor();
  assert.equal(await consent.isChecked(), false);
  assert.ok(await start.isDisabled());
  await begin();
  await modal
    .getByText("작업 완료", { exact: true })
    .waitFor({ timeout: 60000 });
  assert.equal((await api(statusPath)).grant.requires_reconsent, false);
  console.log(
    "PASS provider fingerprint change clears pending consent and requires new explicit authorization",
  );

  await configure({ rag_embedding_model: "slow-browser-model" });
  await modal.getByText("slow-browser-model", { exact: true }).waitFor();
  await begin();
  const stop = modal.getByRole("button", { name: "색인 중지", exact: true });
  await stop.waitFor();
  await stop.click();
  const stopDialog = page.getByRole("dialog", {
    name: "색인 작업을 중지할까요?",
    exact: true,
  });
  await stopDialog
    .getByRole("button", { name: "색인 중지", exact: true })
    .click();
  await stopDialog.waitFor({ state: "hidden" });
  await modal
    .getByText("작업 취소됨", { exact: true })
    .waitFor({ timeout: 30000 });
  let stopped = await api(statusPath);
  assert.equal(stopped.grant.auto_reindex, false);
  assert.equal(stopped.grant.active, true);
  await modal
    .getByRole("button", { name: "동의 철회 및 색인 삭제", exact: true })
    .click();
  const revokeDialog = page.getByRole("dialog", {
    name: "전송 동의를 철회하고 색인을 삭제할까요?",
    exact: true,
  });
  await revokeDialog
    .getByRole("button", { name: "돌아가기", exact: true })
    .click();
  assert.equal((await api(statusPath)).grant.active, true);
  await modal
    .getByRole("button", { name: "동의 철회 및 색인 삭제", exact: true })
    .click();
  await revokeDialog
    .getByRole("button", { name: "동의 철회 및 삭제", exact: true })
    .click();
  await revokeDialog.waitFor({ state: "hidden" });
  await modal
    .getByText("활성화된 전송 동의가 없습니다.", { exact: true })
    .waitFor();
  stopped = await api(statusPath);
  assert.equal(stopped.grant.active, false);
  assert.equal((await api(`/documents/${documentID}`)).id, documentID);
  console.log(
    "PASS running job cancellation, independent revoke confirmation, cleared grant and original document preservation",
  );

  await page.keyboard.press("Escape");
  await modal.waitFor({ state: "hidden" });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "AI 검색 색인", exact: true }).click();
  await modal.waitFor();
  await modal
    .getByText("활성화된 전송 동의가 없습니다.", { exact: true })
    .waitFor();
  await shot("mobile-rag-index");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  assert.deepEqual(errors, []);
  console.log("PASS mobile consent controls and zero page errors");
} catch (error) {
  await shot("rag-index-failure").catch(() => {});
  throw error;
} finally {
  await browser.close();
}
