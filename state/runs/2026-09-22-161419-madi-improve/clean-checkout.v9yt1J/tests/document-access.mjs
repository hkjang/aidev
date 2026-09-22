import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium, expect } from "playwright/test";
import { documentPanel } from "./document-ui.mjs";
const base = process.env.MADI_BASE_URL,
  out = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/document-access",
  );
if (!base) throw new Error("MADI_BASE_URL fixture required");
const browser = await chromium.launch();
const options = {
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  reducedMotion: "reduce",
};
const owner = await browser.newContext(options),
  requester = await browser.newContext(options),
  page = await owner.newPage(),
  other = await requester.newPage();
const errors = [];
for (const p of [page, other]) p.on("pageerror", (e) => errors.push(e.message));
async function api(ctx, url, method = "GET", data, status = 200) {
  const r = await ctx.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(r.status(), status, `${method} ${url}: ${await r.text()}`);
  return r.json();
}
async function shot(p, name) {
  await mkdir(out, { recursive: true });
  await p.evaluate(() => document.fonts.ready);
  await p.screenshot({
    path: path.join(out, name + ".png"),
    animations: "disabled",
  });
}
async function choose(p, wid) {
  await p.goto(base + "/app");
  await p.getByLabel("워크스페이스 선택", { exact: true }).selectOption(wid);
}
async function noOverflow(p) {
  await expect
    .poll(() =>
      p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1),
    )
    .toBe(true);
}
async function remove(id) {
  await page.goto(`${base}/app/documents/${id}?mode=source`);
  await page.getByLabel("Markdown 원문 편집", { exact: true }).waitFor();
  await documentPanel(page, "속성");
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "휴지통으로 이동", exact: true })
    .click();
  await page.waitForURL(/\/app\/documents$/);
  const banner = page.getByRole("region", {
    name: "휴지통 이동 실행 취소",
    exact: true,
  });
  await expect(banner).toBeVisible();
  return banner;
}
try {
  await api(owner, "/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const ws = await api(owner, "/workspaces", "POST", {
      name: "문서 접근과 안전한 되돌리기",
    }),
    password = "Access-Browser-Password-2026!",
    email = "access-browser@example.test";
  await api(owner, "/admin/users", "POST", {
    email,
    password,
    name: "열람 요청자",
    role: "editor",
  });
  await api(owner, `/workspaces/${ws.id}/members`, "PUT", {
    email,
    role: "editor",
  });
  await api(requester, "/auth/login", "POST", { email, password });
  const doc = await api(owner, "/documents", "POST", {
    workspace_id: ws.id,
    title: "접근 전에는 보이지 않는 운영 기록",
    markdown: "# 승인과 권한은 별개\n\n공유된 원문은 그대로 유지합니다.\n",
    visibility: "private",
  });
  await choose(other, ws.id);
  await other.goto(`${base}/app/documents/${doc.id}?mode=preview`);
  await expect(
    other.getByRole("button", { name: "접근 권한 요청", exact: true }),
  ).toBeVisible();
  await expect(other.getByText(doc.title, { exact: true })).toHaveCount(0);
  await other
    .getByRole("button", { name: "접근 권한 요청", exact: true })
    .click();
  let modal = other.getByRole("dialog", {
    name: "문서 접근 요청",
    exact: true,
  });
  await expect(modal.getByLabel("요청 권한", { exact: true })).toHaveValue(
    "read",
  );
  await modal.getByLabel("요청 권한", { exact: true }).selectOption("write");
  await modal.getByLabel("요청 권한", { exact: true }).selectOption("read");
  await modal
    .getByLabel("요청 사유", { exact: true })
    .fill("운영 문서를 참고하여 후속 작업을 준비합니다.");
  await modal
    .getByRole("button", { name: "접근 요청 접수", exact: true })
    .click();
  await expect(modal.getByRole("status")).toContainText(
    "문서의 존재나 제목은 공개하지 않습니다.",
  );
  await shot(other, "access-request-sent");
  await modal
    .getByRole("link", { name: "내 접근 요청 확인", exact: true })
    .click();
  await expect(
    other.getByRole("heading", { name: "문서 접근 요청", exact: true }).first(),
  ).toBeVisible();
  await expect(other.locator(".access-request-list")).toContainText(doc.id);
  await expect(other.locator(".access-request-list")).not.toContainText(
    doc.title,
  );
  const unknown = "00000000-0000-4000-8000-000000000099";
  await api(
    requester,
    `/documents/${unknown}/access-requests`,
    "POST",
    { workspace_id: ws.id, permission: "read" },
    202,
  );
  await other.getByRole("button", { name: "새로고침", exact: true }).click();
  await expect(other.locator(".access-request-list")).toContainText(unknown);
  await choose(page, ws.id);
  await page.goto(base + "/app/access-requests?view=received");
  let row = page
    .locator(".access-request-list article")
    .filter({ hasText: doc.title });
  await expect(row).toContainText("운영 문서를 참고");
  await expect(page.locator(".access-request-list article")).toHaveCount(1);
  await row.getByRole("button", { name: "접근 허용", exact: true }).click();
  modal = page.getByRole("dialog", {
    name: "접근 권한 변경 확인",
    exact: true,
  });
  await expect(
    modal.getByRole("button", { name: "확인한 접근 권한 허용", exact: true }),
  ).toBeDisabled();
  await expect(modal).toContainText("나만 보기");
  await expect(modal).toContainText("선택한 사용자");
  await modal.getByRole("checkbox").check();
  await shot(page, "access-grant-review");
  await modal
    .getByRole("button", { name: "확인한 접근 권한 허용", exact: true })
    .click();
  await expect(modal).toHaveCount(0);
  await expect(row).toContainText("접근 허용");
  const granted = await api(requester, `/documents/${doc.id}`);
  assert.equal(granted.version, 2);
  assert.equal(granted.visibility, "selected");
  assert.equal(granted.can_write, false);
  assert.equal(granted.markdown, doc.markdown);
  await other.goto(`${base}/app/documents/${doc.id}?mode=preview`);
  await expect(other.locator(".document-body")).toContainText(
    "공유된 원문은 그대로 유지합니다.",
  );
  await expect(
    other.getByRole("button", { name: "저장", exact: true }),
  ).toBeDisabled();
  // Sharing/moving is a separate CAS review; a destination changed after the
  // preview must reject the old ticket without touching the source document.
  const destination = await api(owner, "/documents", "POST", {
    workspace_id: ws.id,
    title: "이동 대상 공간의 상위 문서",
    markdown: "위치 기준",
    visibility: "workspace",
  });
  await page.goto(`${base}/app/documents/${doc.id}?mode=source`);
  await page.getByRole("button", { name: "공유", exact: true }).click();
  modal = page.getByRole("dialog", { name: "문서 공유 및 속성", exact: true });
  await modal
    .getByLabel("공개 범위", { exact: true })
    .selectOption("workspace");
  const impact = modal.getByRole("region", {
    name: "공유·이동 영향 미리보기",
    exact: true,
  });
  await expect(impact.getByRole("checkbox")).toBeVisible();
  await expect(
    modal.getByRole("button", { name: "설정 저장", exact: true }),
  ).toBeDisabled();
  await impact.getByRole("checkbox").check();
  await modal
    .getByLabel("상위 문서", { exact: true })
    .selectOption(destination.id);
  await expect(impact.getByRole("checkbox")).not.toBeChecked();
  await expect(impact).toContainText(destination.title);
  await api(owner, `/documents/${destination.id}`, "PUT", {
    version: 1,
    title: "동료가 바꾼 이동 목적지",
  });
  await impact.getByRole("checkbox").check();
  const rejected = page.waitForResponse(
    (r) =>
      new URL(r.url()).pathname === `/api/v1/documents/${doc.id}` &&
      r.request().method() === "PUT",
  );
  await modal.getByRole("button", { name: "설정 저장", exact: true }).click();
  assert.equal((await rejected).status(), 409);
  assert.equal((await api(owner, `/documents/${doc.id}`)).version, 2);
  await impact
    .getByRole("button", { name: "현재 범위 다시 확인", exact: true })
    .click();
  await expect(impact.getByRole("checkbox")).not.toBeChecked();
  await expect(impact).toContainText("동료가 바꾼 이동 목적지");
  await impact.getByRole("checkbox").check();
  await shot(page, "sharing-move-impact-review");
  await modal.getByRole("button", { name: "설정 저장", exact: true }).click();
  await expect(modal).toHaveCount(0);
  const moved = await api(owner, `/documents/${doc.id}`);
  assert.equal(moved.version, 3);
  assert.equal(moved.parent_id, destination.id);
  assert.equal(moved.markdown, doc.markdown);
  // Undo applies the deletion response version, not whichever version exists later.
  const trash = await api(owner, "/documents", "POST", {
    workspace_id: ws.id,
    title: "휴지통 버전 경합 검증",
    markdown: "보존해야 할 정확한 원문\n",
    visibility: "private",
  });
  let banner = await remove(trash.id);
  await shot(page, "trash-undo");
  await banner
    .getByRole("button", { name: "휴지통 이동 실행 취소", exact: true })
    .click();
  await expect(banner).toHaveCount(0);
  assert.equal((await api(owner, `/documents/${trash.id}`)).version, 3);
  banner = await remove(trash.id);
  await api(owner, `/documents/${trash.id}/restore`, "POST", {
    expected_version: 4,
  });
  const newer = await api(owner, `/documents/${trash.id}`, "DELETE", {
    expected_version: 5,
  });
  assert.equal(newer.version, 6);
  await banner
    .getByRole("button", { name: "휴지통 이동 실행 취소", exact: true })
    .click();
  await expect(
    banner.getByRole("heading", {
      name: "다른 변경과 충돌했습니다",
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    banner.getByRole("button", { name: "휴지통 이동 실행 취소", exact: true }),
  ).toBeDisabled();
  assert.equal((await api(owner, `/documents/${trash.id}`)).version, 6);
  await shot(page, "trash-undo-conflict");
  await banner.getByRole("link", { name: "휴지통 확인", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "휴지통", exact: true }),
  ).toBeVisible();
  const restore = page
    .locator(".document-table-row")
    .filter({ hasText: trash.title });
  await restore.getByRole("button", { name: "문서 복원", exact: true }).click();
  await expect
    .poll(async () => (await api(owner, `/documents/${trash.id}`)).deleted_at)
    .toBe(null);
  assert.equal((await api(owner, `/documents/${trash.id}`)).version, 7);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(base + "/app/access-requests?view=received");
  await expect(page.locator(".access-request-list")).toContainText(doc.title);
  await noOverflow(page);
  await shot(page, "access-requests-mobile");
  await other.setViewportSize({ width: 390, height: 844 });
  await other.goto(base + "/app/access-requests?view=sent");
  await noOverflow(other);
  await shot(other, "access-request-sent-mobile");
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      checks: [
        "unknown-target-indistinguishable",
        "permission-select",
        "owner-grant-preview",
        "private-to-selected",
        "read-only",
        "CAS-undo",
        "stale-undo-rejected",
        "trash-restoration",
        "mobile390",
        "console0",
      ],
      screenshots: out,
    }),
  );
} catch (e) {
  for (const [p, name] of [
    [page, "owner-failure"],
    [other, "requester-failure"],
  ]) {
    await shot(p, name).catch(() => {});
  }
  throw e;
} finally {
  await browser.close();
}
