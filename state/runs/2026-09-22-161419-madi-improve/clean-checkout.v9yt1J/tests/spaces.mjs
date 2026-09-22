import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";

// Dedicated QA deployment only: creates a workspace/organization and UI fixtures.
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
page.on("pageerror", (e) => issues.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) issues.push(`${r.status()} ${r.url()}`);
});
page.on("console", (m) => {
  if (m.type() === "error" && !m.text().includes("401 (Unauthorized)"))
    issues.push(m.text());
});
await context.route("**/*", (r) =>
  new URL(r.request().url()).origin === new URL(base).origin
    ? r.continue()
    : r.abort(),
);
async function api(endpoint, method = "GET", data) {
  const r = await context.request.fetch(`${base}/api/v1${endpoint}`, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${method} ${endpoint}: ${r.status()} ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  if (await page.locator(".toast .icon-button").count())
    await page.locator(".toast .icon-button").click();
  await page.screenshot({
    path: path.join(out, `${name}.png`),
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
    name: "공간·조직 검증 워크스페이스",
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspace.id,
  );
  await page.goto(`${base}/app/spaces`);
  await page.getByRole("heading", { name: "공간", exact: true }).waitFor();
  await page.getByRole("button", { name: "새 공간", exact: true }).click();
  let dialog = page.getByRole("dialog");
  await dialog.getByLabel("공간 이름", { exact: true }).fill("플랫폼 운영");
  await dialog
    .getByLabel("주소 이름", { exact: true })
    .fill("platform-operations");
  await dialog
    .getByLabel("공개 범위", { exact: true })
    .selectOption("restricted");
  await dialog
    .getByLabel("문서 기본 등급", { exact: true })
    .selectOption("confidential");
  await dialog.getByRole("button", { name: "공간 저장", exact: true }).click();
  await dialog.waitFor({ state: "hidden" });
  const sid = new URL(page.url()).searchParams.get("space");
  assert.ok(sid);
  await api("/documents", "POST", {
    workspace_id: workspace.id,
    space_id: sid,
    title: "플랫폼 운영 표준",
    markdown:
      "# 플랫폼 운영 표준\n\n서비스 운영 원칙과 팀의 지식을 이 공간에서 관리합니다.",
    tags: ["운영", "플랫폼"],
  });
  await page.reload();
  await page
    .getByRole("heading", { name: "플랫폼 운영", exact: true })
    .waitFor();
  await page
    .locator(".space-content")
    .getByRole("link", { name: /플랫폼 운영 표준/ })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("space"), sid);
  await shot("spaces");
  console.log(
    "PASS space create, native visibility/classification and route refresh",
  );
  await page.getByRole("button", { name: "멤버 권한", exact: true }).click();
  dialog = page.getByRole("dialog");
  await dialog
    .getByRole("combobox", { name: "워크스페이스 멤버", exact: true })
    .selectOption({ label: "관리자 (admin@example.test)" });
  await dialog
    .getByRole("combobox", { name: "공간 권한", exact: true })
    .selectOption("admin");
  await shot("space-members");
  await dialog.getByRole("button", { name: "닫기", exact: true }).click();
  await page.getByRole("button", { name: "새 공간", exact: true }).click();
  dialog = page.getByRole("dialog");
  await dialog.getByLabel("공간 이름", { exact: true }).fill("서비스 지식");
  await dialog.getByLabel("상위 공간", { exact: true }).selectOption(sid);
  await dialog.getByRole("button", { name: "공간 저장", exact: true }).click();
  await dialog.waitFor({ state: "hidden" });
  await page
    .getByRole("heading", { name: "서비스 지식", exact: true })
    .waitFor();
  console.log("PASS nested space parent selection");
  await page.goto(`${base}/app/workspace-settings`);
  await page
    .getByRole("heading", { name: "워크스페이스 설정", exact: true })
    .waitFor();
  await page.getByLabel("문서 검토 주기 (일)", { exact: true }).fill("90");
  await page
    .getByLabel("문서 생명주기 자동화", { exact: true })
    .selectOption("off");
  await page.getByRole("button", { name: "설정 저장", exact: true }).click();
  await page.getByText("설정 버전 2", { exact: true }).waitFor();
  await page.reload();
  await page.getByLabel("문서 검토 주기 (일)", { exact: true }).waitFor();
  assert.equal(
    await page.getByLabel("문서 검토 주기 (일)", { exact: true }).inputValue(),
    "90",
  );
  assert.equal(
    await page.getByLabel("문서 생명주기 자동화", { exact: true }).inputValue(),
    "off",
  );
  await shot("workspace-settings");
  assert.equal(
    await page
      .getByRole("link", { name: "로고·파비콘·팀 브랜딩", exact: true })
      .getAttribute("href"),
    "/app/workspace-operations?tab=branding",
  );
  assert.equal(
    await page
      .getByRole("link", { name: "기능별 사용 정책", exact: true })
      .getAttribute("href"),
    "/app/workspace-operations?tab=features",
  );
  console.log(
    "PASS workspace policy persistence and dedicated branding/feature settings links",
  );
  await page.goto(`${base}/app/organizations`);
  await page
    .getByLabel("새 조직 이름", { exact: true })
    .fill("지식 플랫폼 조직");
  await page.getByRole("button", { name: "조직 만들기", exact: true }).click();
  await page
    .getByRole("heading", { name: "지식 플랫폼 조직", exact: true })
    .waitFor();
  const oid = new URL(page.url()).searchParams.get("organization");
  assert.ok(oid);
  await page.reload();
  await page
    .getByRole("heading", { name: "지식 플랫폼 조직", exact: true })
    .waitFor();
  await shot("organizations");
  console.log("PASS organization creation and refresh");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${base}/app/spaces?space=${sid}`);
  await page
    .getByRole("heading", { name: "플랫폼 운영", exact: true })
    .waitFor();
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot("mobile-spaces");
  await page.goto(`${base}/app/workspace-settings`);
  await page.getByLabel("문서 검토 주기 (일)", { exact: true }).waitFor();
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot("mobile-workspace-settings");
  console.log("PASS mobile spaces/settings zero overflow");
  assert.deepEqual(issues, []);
  console.log("PASS spaces UI zero console/runtime errors");
} finally {
  await browser.close();
}
