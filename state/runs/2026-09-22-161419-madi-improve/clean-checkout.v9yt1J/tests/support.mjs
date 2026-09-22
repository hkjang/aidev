import assert from "node:assert/strict";
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
const base = process.env.MADI_BASE_URL,
  wid = process.env.MADI_SUPPORT_WORKSPACE,
  did = process.env.MADI_SUPPORT_DOCUMENT,
  tid = process.env.MADI_SUPPORT_TARGET;
assert.ok(
  base && wid && did && tid,
  "Run isolated TestBrowserSupportDiagnostics fixture",
);
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    timezoneId: "Asia/Seoul",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const issues = [],
  external = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) issues.push(`${r.status()} ${r.url()}`);
});
page.on("request", (r) => {
  if (r.url().startsWith("https://example.invalid")) external.push(r.url());
});
async function api(path, method = "GET", data, status = 200) {
  const r = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(r.status(), status, `${method} ${path}: ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  const out = new URL("../docs/screenshots/", import.meta.url);
  await mkdir(out, { recursive: true });
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    animations: "disabled",
  });
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const original = (await api("/auth/me")).id;
  await context.addInitScript(
    (w) => localStorage.setItem("madi.workspace", w),
    wid,
  );
  await page.goto(base + "/admin/support");
  await page
    .getByRole("heading", { name: "읽기 전용 지원 진단", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "지원 정책", exact: true }).click();
  const modal = page.getByRole("dialog", { name: "지원 진단 정책" });
  await modal.waitFor();
  assert.equal(
    await modal
      .getByRole("checkbox", { name: "읽기 전용 지원 진단 허용 (기본 꺼짐)" })
      .isChecked(),
    true,
  );
  await shot("support-policy");
  await modal.getByRole("button", { name: "정책 저장", exact: true }).click();
  await modal.waitFor({ state: "hidden" });
  await page.getByLabel("지원 대상", { exact: true }).selectOption(tid);
  await page.getByLabel("지원 시간 (분)", { exact: true }).fill("10");
  await page
    .getByLabel("지원 사유", { exact: true })
    .fill("공통 문서 메뉴 접근 상태를 함께 확인합니다");
  assert.equal(
    await page
      .getByRole("button", { name: "진단 시작", exact: true })
      .isDisabled(),
    true,
  );
  await page.getByRole("checkbox", { name: /계정 대체가 아닌/ }).check();
  await shot("support-start");
  await page.getByRole("button", { name: "진단 시작", exact: true }).click();
  await page.waitForURL(/session=/);
  const sid = new URL(page.url()).searchParams.get("session");
  await page.getByText("읽기 전용 · 지원 대상", { exact: true }).waitFor();
  assert.equal((await api("/auth/me")).id, original, "principal changed");
  await page.getByRole("button", { name: "문서 확인", exact: true }).click();
  await page.getByRole("button", { name: /공통 열람 운영 문서/ }).waitFor();
  assert.equal(
    (await page.locator(".support-page").innerText()).includes(
      "TARGET_PRIVATE_BROWSER_SECRET",
    ),
    false,
  );
  await page.getByRole("button", { name: /공통 열람 운영 문서/ }).click();
  await page.getByLabel("진단 문서 원문", { exact: true }).waitFor();
  assert.match(
    await page.getByLabel("진단 문서 원문", { exact: true }).innerText(),
    /\*\*문서 원문 보존\*\*/,
  );
  await shot("support-document");
  await api(
    `/support/sessions/${sid}/documents/${did}`,
    "PUT",
    { markdown: "MUST_NOT_WRITE" },
    403,
  );
  await api(
    `/support/sessions/${sid}/ai/chat`,
    "POST",
    { message: "MUST_NOT_CALL" },
    403,
  );
  await api(
    `/support/sessions/${sid}/attachments/${did}`,
    "GET",
    undefined,
    403,
  );
  await page.reload();
  await page.getByText("읽기 전용 · 지원 대상", { exact: true }).waitFor();
  assert.equal(new URL(page.url()).searchParams.get("session"), sid);
  await page.getByRole("button", { name: "문서 확인", exact: true }).click();
  await page.getByRole("button", { name: /공통 열람 운영 문서/ }).click();
  await page.getByLabel("진단 문서 원문", { exact: true }).waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  await shot("mobile-support");
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    true,
    "mobile overflow",
  );
  await api(`/documents/${did}/shares`, "PUT", {
    user_id: tid,
    permission: "remove",
  });
  await page
    .getByRole("heading", {
      name: "안전을 위해 진단 내용을 지웠습니다",
      exact: true,
    })
    .waitFor({ timeout: 2500 });
  assert.equal(
    await page.getByLabel("진단 문서 원문", { exact: true }).count(),
    0,
  );
  assert.equal(
    (await page.locator(".support-page").innerText()).includes(
      "공통 열람 운영 문서",
    ),
    false,
  );
  await page
    .getByRole("button", { name: "진단 종료", exact: true })
    .first()
    .click();
  await page.waitForURL((u) => !u.searchParams.has("session"));
  assert.equal((await api("/auth/me")).id, original);
  assert.deepEqual(external, [], "source image made external request");
  assert.deepEqual(issues, []);
  console.log(
    "PASS support policy, explicit consent, ACL intersection, unchanged principal, no writes/AI/download/external rendering, refresh, mobile, live revocation",
  );
} catch (e) {
  await page.screenshot({
    path: "/tmp/madi-support-failure.png",
    fullPage: true,
  });
  throw e;
} finally {
  await browser.close();
}
