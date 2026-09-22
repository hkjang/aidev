import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { createHmac, randomUUID } from "node:crypto";
import path from "node:path";

const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const output = path.resolve(
  process.env.MADI_SCREENSHOT_DIR || "test-results/inbound-capture",
);
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
  locale: "ko-KR",
});
const page = await context.newPage(),
  errors = [];
let originalPolicy;
const secrets = [];
page.on("pageerror", (error) => errors.push(error.message));
page.on("response", (response) => {
  if (response.status() >= 500)
    errors.push(`${response.status()} ${response.url()}`);
});
async function api(endpoint, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + endpoint, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${endpoint}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  const text = await page.locator("body").innerText();
  for (const secret of secrets) {
    assert.ok(!text.includes(secret), "plain capture secret in screenshot");
    assert.equal(
      await page
        .locator('input:not([type="password"])')
        .evaluateAll(
          (inputs, value) => inputs.some((input) => input.value === value),
          secret,
        ),
      false,
    );
  }
  await page.screenshot({
    path: path.join(output, name + ".png"),
    fullPage: true,
    animations: "disabled",
  });
}
async function hook(endpoint, secret, id, body, wanted = 202) {
  const timestamp = String(Math.floor(Date.now() / 1000));
  const signature =
    "sha256=" +
    createHmac("sha256", secret)
      .update(timestamp + "." + id + "." + body)
      .digest("hex");
  const response = await fetch(endpoint, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Madi-ID": id,
      "X-Madi-Timestamp": timestamp,
      "X-Madi-Signature": signature,
    },
    body,
  });
  assert.equal(response.status, wanted);
  return response.json();
}
try {
  await page.goto(base + "/login");
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  originalPolicy = await api("/admin/inbound-capture-settings");
  const stamp = Date.now(),
    name = "개인 운영 자료 수집 " + stamp;
  const workspace = await api("/workspaces", "POST", {
    name: "메일·웹훅 수집 검증 " + stamp,
  });
  await page.evaluate(
    (id) => localStorage.setItem("madi.workspace", id),
    workspace.id,
  );
  await page.goto(base + "/admin/inbound-capture");
  await page.getByLabel("개인 HMAC Webhook 수집 허용", { exact: true }).check();
  await page
    .getByRole("button", { name: "수집 정책 저장", exact: true })
    .click();
  await page
    .getByText("외부 수집 정책을 저장했습니다", { exact: true })
    .waitFor();
  await shot("admin-inbound-capture");
  await page.goto(base + "/app/capture-channels");
  await page
    .getByRole("button", { name: "수집 채널 추가", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("수집 방식", { exact: true }).selectOption("imap");
  assert.equal(
    await dialog
      .getByLabel("처음 연결한 이후에 도착한 메일부터 수집")
      .isChecked(),
    true,
  );
  await dialog
    .getByLabel("IMAP 호스트", { exact: true })
    .fill("imap.example.internal");
  await shot("imap-capture-editor");
  await dialog.getByLabel("수집 방식", { exact: true }).selectOption("hmac");
  await dialog.getByLabel("수집 채널 이름", { exact: true }).fill(name);
  await dialog
    .getByLabel("수집 대상 워크스페이스", { exact: true })
    .selectOption(workspace.id);
  await dialog.getByLabel("수집 채널 활성화", { exact: true }).check();
  await dialog
    .getByRole("button", { name: "수집 채널 저장", exact: true })
    .click();
  await page
    .getByRole("dialog", { name: "수집 서명 비밀 · 한 번만 표시" })
    .waitFor();
  const endpoint = await page
    .getByLabel("수집 Webhook 주소", { exact: true })
    .inputValue();
  const secret = await page
    .getByLabel("새 수집 서명 비밀", { exact: true })
    .inputValue();
  secrets.push(secret);
  assert.ok(secret.length >= 32);
  const channelID = endpoint.split("/").at(-1);
  await page.keyboard.press("Escape");
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  assert.ok(!JSON.stringify(await api("/capture-channels")).includes(secret));
  const messageID = randomUUID(),
    title = "운영 점검 자료 " + stamp;
  const body = JSON.stringify({
    title,
    text: "# 오늘의 운영 점검\n\n서명된 외부 자료를 개인 문서로 보관했습니다.\n\n- [ ] 담당자 검토 후 직접 공유",
    attachments: [
      {
        name: "점검.txt",
        type: "text/plain",
        data: Buffer.from("안전하게 수집한 실제 첨부입니다.\n").toString(
          "base64",
        ),
      },
    ],
  });
  const receipt = await hook(endpoint, secret, messageID, body);
  const duplicate = await hook(endpoint, secret, messageID, body);
  assert.equal(duplicate.id, receipt.id);
  assert.equal(duplicate.duplicate, true);
  await hook(
    endpoint,
    secret,
    messageID,
    JSON.stringify({ title: "변경된 본문" }),
    409,
  );
  for (let attempt = 0; attempt < 30; attempt++) {
    await page.getByRole("button", { name: "이력 갱신", exact: true }).click();
    if (
      await page
        .getByRole("main")
        .getByRole("link", { name: title, exact: true })
        .count()
    )
      break;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  await page
    .getByRole("main")
    .getByRole("link", { name: title, exact: true })
    .waitFor();
  await page.reload();
  await page
    .getByRole("main")
    .getByRole("link", { name: title, exact: true })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("channel"), channelID);
  await shot("inbound-capture-history");
  const history = await api(`/capture-channels/${channelID}/history`);
  const captured = history.find((row) => row.id === receipt.id);
  const document = await api(`/documents/${captured.document_id}`);
  assert.equal(document.visibility, "private");
  assert.equal(document.workspace_id, workspace.id);
  assert.ok(document.markdown.includes("서명된 외부 자료"));
  const files = await api(`/documents/${document.id}/attachments`);
  assert.equal(files.length, 1);
  await page
    .getByRole("main")
    .getByRole("link", { name: title, exact: true })
    .click();
  await page
    .getByRole("heading", { name: "오늘의 운영 점검", exact: true })
    .first()
    .waitFor();
  await shot("captured-private-document");
  await page.goto(base + "/app/capture-channels?channel=" + channelID);
  const row = page
    .getByRole("row")
    .filter({ has: page.getByText(name, { exact: true }) });
  await row.getByRole("button", { name: "비밀 회전", exact: true }).click();
  await page
    .getByRole("button", { name: "새 비밀로 회전", exact: true })
    .click();
  await page.getByLabel("새 수집 서명 비밀", { exact: true }).waitFor();
  const rotated = await page
    .getByLabel("새 수집 서명 비밀", { exact: true })
    .inputValue();
  secrets.push(rotated);
  assert.notEqual(rotated, secret);
  await page.keyboard.press("Escape");
  await hook(endpoint, secret, randomUUID(), body, 401);
  await hook(endpoint, rotated, messageID, body);
  await page.setViewportSize({ width: 390, height: 844 });
  await shot("inbound-capture-mobile");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "mobile overflow",
  );
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      workspace_id: workspace.id,
      channel_id: channelID,
      screenshots: output,
    }),
  );
} finally {
  try {
    if (originalPolicy)
      await api("/admin/inbound-capture-settings", "PUT", {
        hooks_enabled: originalPolicy.hooks_enabled,
        imap_enabled: originalPolicy.imap_enabled,
        allowed_hosts: originalPolicy.allowed_hosts,
      });
  } finally {
    await browser.close();
  }
}
