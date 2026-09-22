import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import http from "node:http";
import { createHmac } from "node:crypto";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  output = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/notifications",
  );
await mkdir(output, { recursive: true });
const secret = "browser-notification-HMAC-test-secret-2026";
const received = [];
let originalPolicy;
const receiver = http.createServer(async (req, res) => {
  let body = "";
  for await (const chunk of req) body += chunk;
  const wanted =
    "sha256=" +
    createHmac("sha256", secret)
      .update(
        req.headers["x-madi-timestamp"] +
          "." +
          req.headers["x-madi-id"] +
          "." +
          body,
      )
      .digest("hex");
  assert.equal(req.headers["x-madi-signature"], wanted);
  assert.ok(!body.includes("PRIVATE_SENTINEL"));
  received.push(JSON.parse(body));
  res.end("ok");
});
await new Promise((resolve) => receiver.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch({ headless: true }),
  context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
    locale: "ko-KR",
  }),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`);
});
async function api(endpoint, method = "GET", data, client = context) {
  const response = await client.request.fetch(base + "/api/v1" + endpoint, {
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
  assert.ok(
    !(await page.locator("body").innerText()).includes(secret),
    "plain notification secret in screenshot",
  );
  assert.equal(
    await page
      .locator('input:not([type="password"])')
      .evaluateAll(
        (inputs, value) => inputs.some((input) => input.value === value),
        secret,
      ),
    false,
  );
  await page.screenshot({
    path: path.join(output, name + ".png"),
    fullPage: true,
    animations: "disabled",
  });
}
try {
  await page.goto(base + "/login");
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const stamp = Date.now(),
    workspace = await api("/workspaces", "POST", {
      name: "외부 알림 UI 검증 " + stamp,
    });
  await page.evaluate(
    (id) => localStorage.setItem("madi.workspace", id),
    workspace.id,
  );
  const policy = await api("/admin/notification-settings");
  originalPolicy = policy;
  await api("/admin/notification-settings", "PUT", {
    enabled: true,
    allowed_hosts: [...new Set([...(policy.allowed_hosts || []), "127.0.0.1"])],
  });
  await page.goto(base + "/admin/notification-channels");
  await page.getByRole("button", { name: "채널 추가", exact: true }).click();
  await page
    .getByLabel("채널 이름", { exact: true })
    .fill("검증용 보안 알림 " + stamp);
  await page
    .getByLabel("알림 채널 종류", { exact: true })
    .selectOption("webhook");
  await page
    .getByLabel("채널 대상 범위", { exact: true })
    .selectOption(workspace.id);
  await page
    .getByLabel("알림 Webhook URL", { exact: true })
    .fill("http://127.0.0.1:" + receiver.address().port + "/notify");
  await page.getByLabel("Webhook 서명 비밀", { exact: true }).fill(secret);
  await page.getByLabel("사내망 알림 HTTP 허용", { exact: true }).check();
  await page.getByLabel("알림 채널 활성화", { exact: true }).check();
  await shot("notification-channel-editor");
  await page
    .getByRole("button", { name: "알림 채널 저장", exact: true })
    .click();
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  await shot("admin-notification-channels");
  await page.goto(base + "/app/notification-settings");
  const selection = page.getByRole("checkbox", {
    name: new RegExp("검증용 보안 알림 " + stamp),
  });
  await selection.check();
  await page.getByText("알림 채널을 선택했습니다", { exact: true }).waitFor();
  await page.reload();
  await selection.waitFor();
  assert.equal(await selection.isChecked(), true);
  await shot("notification-preferences");
  const email = `notification-reviewer-${stamp}@example.test`;
  await api("/admin/users", "POST", {
    email,
    name: "알림 검증 동료",
    role: "editor",
    password: "Notification-browser-password-2026!",
  });
  await api(`/workspaces/${workspace.id}/members`, "PUT", {
    email,
    role: "editor",
  });
  const doc = await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "PRIVATE_SENTINEL",
    markdown: "# PRIVATE_SENTINEL_BODY",
  });
  const colleague = await browser.newContext();
  await api(
    "/auth/login",
    "POST",
    { email, password: "Notification-browser-password-2026!" },
    colleague,
  );
  await api(
    "/documents/" + doc.id + "/comments",
    "POST",
    { body: "PRIVATE_SENTINEL_COMMENT" },
    colleague,
  );
  await colleague.close();
  for (let i = 0; i < 30 && received.length === 0; i++) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    await page
      .getByRole("button", { name: "이력 새로고침", exact: true })
      .click();
  }
  assert.equal(received.length, 1);
  await page
    .getByRole("button", { name: "이력 새로고침", exact: true })
    .click();
  await page.getByText("전송됨", { exact: true }).last().waitFor();
  await shot("notification-delivery-history");
  await page.setViewportSize({ width: 390, height: 844 });
  await shot("notification-preferences-mobile");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "mobile overflow",
  );
  await selection.uncheck();
  await page
    .getByText("외부 알림 선택을 해제했습니다", { exact: true })
    .waitFor();
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      workspace_id: workspace.id,
      deliveries: received.length,
      screenshots: output,
    }),
  );
} finally {
  try {
    if (originalPolicy)
      await api("/admin/notification-settings", "PUT", {
        enabled: originalPolicy.enabled,
        allowed_hosts: originalPolicy.allowed_hosts,
      });
  } finally {
    await browser.close();
    await new Promise((resolve) => receiver.close(resolve));
  }
}
