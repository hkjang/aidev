import { chromium, request } from "playwright";
import assert from "node:assert/strict";
import path from "node:path";
import { mkdtemp, mkdir } from "node:fs/promises";
import os from "node:os";
import {
  serverOrigin,
  sourceURL,
  captureMarkdown,
} from "../sdk/clipper/common.js";
import { execFileSync } from "node:child_process";
assert.equal(serverOrigin("https://madi.local"), "https://madi.local");
assert.throws(() => serverOrigin("http://madi.local"));
assert.throws(() => serverOrigin("https://user:key@madi.local"));
assert.throws(() => serverOrigin("https://madi.local/path"));
assert.equal(sourceURL("javascript:alert(1)"), "");
assert.equal(
  captureMarkdown({ text: "plain", author: "author\nname" }),
  "작성자: author name\n\nplain",
);
console.log("PASS clipper URL and text validation");
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const admin = await request.newContext({
  baseURL: base,
  extraHTTPHeaders: { "X-Madi-Request": "1" },
});
async function api(url, method = "GET", data) {
  const res = await admin.fetch("/api/v1" + url, { method, data });
  assert.ok(res.ok(), `${url}: ${res.status()} ${await res.text()}`);
  return res.json();
}
await api("/auth/login", "POST", {
  email: "admin@example.test",
  password: "Browser-Test-Password-2026!",
});
const [ws] = await api("/workspaces");
const key = await api("/keys", "POST", {
  name: "웹 클리퍼 브라우저 검증",
  workspace_id: ws.id,
  scopes: ["document:read", "document:write"],
  expires_in_days: 1,
  rate_limit: 1000,
});
const extension = path.resolve("sdk/clipper/dist/chrome");
const profile = await mkdtemp(path.join(os.tmpdir(), "madi-clipper-test-"));
const headed = !!process.env.MADI_CLIPPER_HEADED;
const context = await chromium.launchPersistentContext(profile, {
  channel: "chromium",
  headless: !headed,
  args: [
    `--disable-extensions-except=${extension}`,
    `--load-extension=${extension}`,
  ],
  viewport: { width: 1000, height: 850 },
});
let worker =
  context.serviceWorkers()[0] || (await context.waitForEvent("serviceworker"));
const id = worker.url().split("/")[2];
const page = await context.newPage();
const errors = [];
page.on("pageerror", (error) => errors.push(error.message));
try {
  await page.goto(`chrome-extension://${id}/options.html`);
  await page.getByLabel("madi 서버 주소", { exact: true }).fill(base);
  await page.getByLabel("개인 API 키", { exact: true }).fill(key.token);
  await page
    .getByRole("button", { name: "서버 연결 권한 허용·확인", exact: true })
    .click();
  if (headed) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    execFileSync(process.env.MADI_XDOTOOL || "xdotool", [
      "key",
      "Tab",
      "Return",
    ]);
  }
  await page
    .getByLabel("기본 워크스페이스", { exact: true })
    .waitFor({ timeout: 15000 });
  await page
    .getByLabel("기본 워크스페이스", { exact: true })
    .selectOption(ws.id);
  await page
    .getByRole("button", { name: "워크스페이스 저장", exact: true })
    .click();
  await page
    .getByRole("status")
    .filter({ hasText: "워크스페이스를 저장했습니다" })
    .waitFor();
  const stored = await worker.evaluate(async () => ({
    local: await chrome.storage.local.get(null),
    session: await chrome.storage.session.get(null),
    permissions: await chrome.permissions.getAll(),
  }));
  assert.ok(!JSON.stringify(stored.local).includes(key.token));
  assert.equal(stored.session.token, key.token);
  assert.deepEqual(stored.permissions.origins, [base + "/*"]);
  console.log(
    "PASS actual extension permissions limited to configured server; key session-only",
  );
  await mkdir("docs/screenshots", { recursive: true });
  await page.screenshot({
    path: "docs/screenshots/clipper-settings.png",
    fullPage: true,
  });
  const source = await context.newPage();
  await context.route(base + "/clipper-test-fixture", (route) =>
    route.fulfill({
      contentType: "text/html; charset=utf-8",
      body: '<!doctype html><html lang="ko"><head><meta charset="UTF-8"><title>팀의 지식을 연결하는 원칙</title><meta name="author" content="지식 운영팀"></head><body><header>웹사이트 메뉴</header><article><h1>팀의 지식을 연결하는 원칙</h1><p>좋은 기록은 다음 사람의 질문에 답합니다.</p><p>결정의 이유와 운영 맥락을 함께 남깁니다.</p></article><footer>하단 메뉴</footer></body></html>',
    }),
  );
  await source.goto(base + "/clipper-test-fixture");
  if (headed) {
    await source.bringToFront();
    execFileSync(process.env.MADI_XDOTOOL || "xdotool", [
      "key",
      "ctrl+shift+y",
    ]);
    await new Promise((resolve) => setTimeout(resolve, 500));
    execFileSync(process.env.MADI_XDOTOOL || "xdotool", ["key", "Escape"]);
  }
  const popup = await context.newPage();
  await popup.goto(`chrome-extension://${id}/popup.html`);
  await source.bringToFront();
  await popup
    .getByRole("button", { name: "내용 가져오기", exact: true })
    .evaluate((button) => button.click());
  await popup.getByLabel("기록 제목", { exact: true }).waitFor();
  assert.equal(
    await popup.getByLabel("기록 제목", { exact: true }).inputValue(),
    "팀의 지식을 연결하는 원칙",
  );
  assert.ok(
    !(
      await popup.getByLabel("보관할 내용", { exact: true }).inputValue()
    ).includes("하단 메뉴"),
  );
  await popup.screenshot({
    path: "docs/screenshots/web-clipper.png",
    fullPage: true,
  });
  await popup
    .getByRole("button", { name: "개인 인박스에 저장", exact: true })
    .click();
  await popup
    .getByRole("status")
    .filter({ hasText: "개인 인박스에 저장했습니다" })
    .waitFor();
  const docs = await api("/captures?workspace_id=" + ws.id);
  const saved = docs.find((doc) => doc.title === "팀의 지식을 연결하는 원칙");
  assert.ok(saved);
  assert.equal(saved.visibility, "private");
  assert.match((await api('/documents/'+saved.id)).markdown, /좋은 기록/);
  console.log(
    "PASS real extension article extraction → preview → authenticated private inbox",
  );
  await popup
    .getByLabel("저장 방식", { exact: true })
    .selectOption("screenshot");
  await source.bringToFront();
  await popup
    .getByRole("button", { name: "내용 가져오기", exact: true })
    .evaluate((button) => button.click());
  await popup
    .getByAltText("저장할 현재 화면 스크린샷", { exact: true })
    .waitFor();
  await popup
    .getByLabel("기록 제목", { exact: true })
    .fill("클리퍼 실제 화면 캡처");
  await popup
    .getByRole("button", { name: "개인 인박스에 저장", exact: true })
    .click();
  await popup
    .getByRole("status")
    .filter({ hasText: "개인 인박스에 저장했습니다" })
    .waitFor();
  console.log(
    "PASS native visible-tab screenshot + actual idempotent attachment upload",
  );
  await page
    .getByRole("button", { name: "연결 해제·키 삭제", exact: true })
    .click();
  await page.getByRole("status").filter({ hasText: "회수했습니다" }).waitFor();
  const cleared = await worker.evaluate(async () => ({
    session: await chrome.storage.session.get(null),
    permissions: await chrome.permissions.getAll(),
  }));
  assert.deepEqual(cleared.session, {});
  assert.equal(cleared.permissions.origins?.length || 0, 0);
  assert.deepEqual(errors, []);
  console.log(
    "PASS disconnect removes token, draft and granted origin; all clipper browser checks passed.",
  );
} catch (error) {
  console.error("EXTENSION PAGE:", await page.locator("body").innerText());
  throw error;
} finally {
  await api("/keys/" + key.key.id, "DELETE");
  await context.close();
  await admin.dispose();
}
