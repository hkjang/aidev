import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

// Disposable development service only; global telemetry edits are restored in
// finally. All branding/feature document fixtures use a newly created workspace.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const receiverCalls = [];
const receiver = createServer(async (req, res) => {
  const chunks = [];
  for await (const chunk of req) chunks.push(chunk);
  receiverCalls.push({
    type: req.headers["content-type"],
    body: Buffer.concat(chunks),
  });
  res.writeHead(200, { "Content-Type": "application/x-protobuf" });
  res.end();
});
await new Promise((resolve) => receiver.listen(0, "127.0.0.1", resolve));
const endpoint = `http://127.0.0.1:${receiver.address().port}/v1/traces`;
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
const out = pathToFileURL(resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots") + "/");
const field = (name) => page.getByLabel(name, { exact: true });
async function api(path, method = "GET", data, want = 200) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(
    response.status(),
    want,
    `${method} ${path}: ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page
    .locator(".toast .icon-button")
    .click({ timeout: 700 })
    .catch(() => {});
  await page.evaluate(() => scrollTo(0, 0));
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    fullPage: true,
    animations: "disabled",
  });
}
async function save(name, pattern, want = 200) {
  const request = page.waitForResponse(
    (r) =>
      r.request().method() === "PUT" &&
      new URL(r.url()).pathname.endsWith(pattern),
  );
  await page.getByRole("button", { name, exact: true }).click();
  const response = await request;
  assert.equal(response.status(), want, await response.text());
  return response.json();
}
async function noOverflow() {
  const dimensions = await page.evaluate(() => ({
    viewport: innerWidth,
    width: document.documentElement.scrollWidth,
  }));
  assert.ok(
    dimensions.width <= dimensions.viewport,
    JSON.stringify(dimensions),
  );
}
let restore = null;
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const original = await api("/admin/settings");
  assert.equal(
    original.otel_enabled,
    false,
    "Use disposable service with telemetry off",
  );
  assert.equal(
    original.otel_auth_token_configured,
    false,
    "Do not forward an existing administrator token to this fixture",
  );
  restore = Object.fromEntries(
    [
      "otel_enabled",
      "otel_endpoint",
      "otel_allow_http",
      "otel_sample_rate",
      "otel_timeout_seconds",
      "operations_errors_enabled",
      "operations_retention_days",
    ].map((k) => [k, original[k]]),
  );
  const ws = await api("/workspaces", "POST", {
    name: "지식 운영 · 브랜드 스튜디오",
  });
  const member = await api("/admin/users", "POST", {
    email: `operations-${Date.now()}@example.test`,
    name: "운영 정책 확인 사용자",
    role: "editor",
    password: "Operations-user-password-2026!",
  });
  await api("/workspaces/" + ws.id + "/members", "PUT", {
    email: member.email,
    role: "editor",
  });
  const canvas = await api("/canvases", "POST", {
    workspace_id: ws.id,
    title: "정책으로 안전하게 보존되는 캔버스",
    data: { nodes: [], edges: [] },
  });
  await page.goto(base + "/app");
  await field("워크스페이스 선택").selectOption(ws.id);
  await page.goto(base + "/app/workspace-operations?tab=branding");
  await field("팀 서비스 이름").waitFor();
  const png = await page.evaluate(() => {
    const canvas = document.createElement("canvas");
    canvas.width = 96;
    canvas.height = 96;
    const c = canvas.getContext("2d");
    c.fillStyle = "#e9f4e9";
    c.fillRect(0, 0, 96, 96);
    c.strokeStyle = "#176d60";
    c.lineWidth = 9;
    c.lineCap = "round";
    c.beginPath();
    c.moveTo(22, 69);
    c.lineTo(22, 29);
    c.lineTo(48, 52);
    c.lineTo(74, 29);
    c.lineTo(74, 69);
    c.stroke();
    return canvas.toDataURL("image/png").split(",")[1];
  });
  await field("새 브랜딩 이미지").setInputFiles({
    name: "team-logo.png",
    mimeType: "image/png",
    buffer: Buffer.from(png, "base64"),
  });
  const uploaded = page.waitForResponse(
    (r) =>
      r.request().method() === "POST" && r.url().endsWith("/branding/assets"),
  );
  await page.getByRole("button", { name: "이미지 등록", exact: true }).click();
  assert.equal((await uploaded).status(), 200);
  await field("팀 서비스 이름").fill("우리 팀 지식 연구소");
  await field("대표 색상").fill("#ffffff");
  await save("브랜딩 저장", "/settings", 400);
  await page.locator('.operations-page').getByText(/최소 4\.5:1/).first().waitFor();
  await field("대표 색상").fill("#176d60");
  const selected = await field("로고 이미지").inputValue();
  await field("파비콘 이미지").selectOption(selected);
  await save("브랜딩 저장", "/settings");
  await page.reload();
  assert.equal(
    await field("팀 서비스 이름").inputValue(),
    "우리 팀 지식 연구소",
  );
  assert.equal(await field("파비콘 이미지").inputValue(), selected);
  await shot("workspace-branding");
  await page.getByRole("button", { name: "기능 정책", exact: true }).click();
  await field("캔버스 정책").selectOption("off");
  await save("기능 정책 저장", "/settings");
  await api("/canvases/" + canvas.id, "GET", undefined, 404);
  await page.reload();
  assert.equal(new URL(page.url()).searchParams.get("tab"), "features");
  assert.equal(await field("캔버스 정책").inputValue(), "off");
  await page.locator('a[href="/app/canvases"]').waitFor({ state: "hidden" });
  await shot("workspace-features");
  await field("캔버스 정책").selectOption("inherit");
  await save("기능 정책 저장", "/settings");
  assert.equal((await api("/canvases/" + canvas.id)).title, canvas.title);
  await page.goto(base + "/admin/operations?tab=features");
  await field("정책 적용 대상").selectOption(member.id);
  await field("플러그인 실행 정책").selectOption("off");
  await save("기능 정책 저장", "/admin/features/users/" + member.id);
  let uf = await api("/admin/features/users/" + member.id);
  assert.equal(uf.data.plugins, false);
  await shot("admin-user-features");
  await field("플러그인 실행 정책").selectOption("inherit");
  await save("기능 정책 저장", "/admin/features/users/" + member.id);
  await page.getByRole("button", { name: "설정 이력", exact: true }).click();
  const history = page.getByRole("dialog", {
    name: "사용자 기능 정책 이력",
    exact: true,
  });
  await history.waitFor();
  page.once("dialog", (d) => d.accept());
  await history
    .getByRole("button", { name: "복원", exact: true })
    .filter({ hasNot: page.locator("[disabled]") })
    .last()
    .click();
  await history.waitFor({ state: "hidden" });
  uf = await api("/admin/features/users/" + member.id);
  assert.equal(uf.data.plugins, false);
  await page
    .getByRole("button", { name: "OpenTelemetry", exact: true })
    .click();
  await field("OTLP 수신 주소").fill(endpoint);
  await field("HTTP 평문 전송 허용").selectOption("on");
  await field("Trace 수집 비율").selectOption("1");
  await field("OpenTelemetry 전송").selectOption("on");
  // External admin edit causes 409, preserving the unsaved endpoint and choices.
  await api("/admin/settings", "PUT", {
    operations_retention_days: original.operations_retention_days === 7 ? 8 : 7,
  });
  await save("운영 진단 설정 저장", "/admin/settings", 409);
  assert.equal(await field("OTLP 수신 주소").inputValue(), endpoint);
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "다시 불러오기", exact: true })
    .click();
  await page.waitForFunction(
    (expected) => document.querySelector("input[type=url]")?.value === expected,
    original.otel_endpoint,
  );
  await field("OTLP 수신 주소").fill(endpoint);
  await field("HTTP 평문 전송 허용").selectOption("on");
  await field("Trace 수집 비율").selectOption("1");
  await field("OpenTelemetry 전송").selectOption("on");
  await save("운영 진단 설정 저장", "/admin/settings");
  for(let attempt=0;attempt<30;attempt++){
    if((await api('/admin/operations/telemetry')).enabled)break;
    await new Promise(resolve=>setTimeout(resolve,100));
  }
  assert.equal((await api('/admin/operations/telemetry')).enabled,true);
  await page
    .getByRole("button", { name: "저장된 연결 테스트", exact: true })
    .click();
  await page
    .getByText(
      "저장된 수신 주소로 고정 진단 Trace 1개를 실제 전송했습니다. 문서·질문·개인정보는 보내지 않았습니다.",
      { exact: true },
    )
    .waitFor();
  assert.ok(receiverCalls.length > 0);
  assert.ok(
    receiverCalls.every(
      (c) => c.type === "application/x-protobuf" && c.body.length > 0,
    ),
  );
  await shot("admin-telemetry");
  await page.getByRole("button", { name: "상태와 오류", exact: true }).click();
  await page.getByText("정상 연결", { exact: true }).waitFor();
  await shot("admin-operations");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.reload();
  await page.getByText("정상 연결", { exact: true }).waitFor();
  await noOverflow();
  await shot("mobile-operations");
  await page
    .getByRole("button", { name: "OpenTelemetry", exact: true })
    .click();
  await field("OTLP 수신 주소").waitFor();
  await noOverflow();
  await shot("mobile-telemetry");
  await page.goto(base + "/app/workspace-operations?tab=branding");
  await field("팀 서비스 이름").waitFor();
  await noOverflow();
  await shot("mobile-branding");
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      workspace: ws.id,
      checks: [
        "safe PNG branding",
        "color contrast validation",
        "URL tab refresh",
        "workspace feature API gate and sidebar",
        "original canvas preserved",
        "administrator user flags and history",
        "service settings 409 keeps draft",
        "actual saved OTLP protobuf receiver",
        "actual health/errors",
        "390px no overflow",
      ],
      receiver_requests: receiverCalls.length,
      errors,
    }),
  );
} catch (error) {
  await mkdir(out, { recursive: true });
  await page
    .screenshot({
      path: new URL("operations-failure.png", out).pathname,
      fullPage: true,
    })
    .catch(() => {});
  throw error;
} finally {
  if (restore)
    await api("/admin/settings", "PUT", restore).catch((e) =>
      console.error("Failed to restore telemetry fixture settings:", e.message),
    );
  await browser.close();
  await new Promise((resolve) => receiver.close(resolve));
}
