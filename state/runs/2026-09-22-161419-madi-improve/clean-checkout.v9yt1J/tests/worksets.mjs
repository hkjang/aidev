import assert from "node:assert/strict";
import path from "node:path";
import { mkdir } from "node:fs/promises";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL,
  out = process.env.MADI_SCREENSHOT_DIR || "test-results/worksets";
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(url, method = "GET", data, status = 200, ctx = context) {
  const r = await ctx.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(r.status(), status, await r.text());
  return r.json();
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(out, name + ".png"),
    animations: "disabled",
  });
}
async function overflow() {
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    )
    .toBe(true);
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", { name: "개인 업무 맥락 확인" });
  const md =
    "---\n# 보존할 주석\naliases: [기존별칭]\ntags: [기존태그]\ncustom: keep-me\n---\n\n# GPU 서버 운영\n\n점검할 GPU 서버 운영 기록입니다. #운영\n\n```text\n원문 코드 보존\n```\n";
  const doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "제목 없는 문서",
    markdown: md,
    visibility: "private",
  });
  const other = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "장애 대응 참고",
    markdown: "# 장애 대응\n\n문제를 확인하고 안전하게 복구합니다.\n",
    visibility: "workspace",
  });
  const db = await api("/databases", "POST", {
    workspace_id: ws.id,
    name: "점검 작업 표",
  });
  await page.goto(base + "/app/worksets");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(base + "/app/worksets");
  await page.getByRole("button", { name: "새 개인 묶음", exact: true }).click();
  let dialog = page.getByRole("dialog", {
    name: "내 작업 묶음 편집",
    exact: true,
  });
  await dialog
    .getByLabel("묶음 이름", { exact: true })
    .fill("GPU 운영 참고 선반");
  await dialog
    .getByLabel("묶음 종류", { exact: true })
    .selectOption("reference");
  for (const item of [doc, other]) {
    await dialog
      .getByLabel("현재 접근 가능한 참조", { exact: true })
      .selectOption(item.id);
    await dialog
      .getByRole("button", { name: "참조 추가", exact: true })
      .click();
  }
  await dialog
    .getByRole("button", { name: "개인 묶음 저장", exact: true })
    .click();
  await expect(page).toHaveURL(/\/app\/worksets\/[a-f0-9-]+$/);
  const setID = new URL(page.url()).pathname.split("/").pop();
  await expect(
    page.getByRole("heading", { name: "GPU 운영 참고 선반", exact: true }),
  ).toBeVisible();
  await page.reload();
  await page
    .getByLabel("1번째 비교 문서", { exact: true })
    .selectOption(doc.id);
  await page
    .getByLabel("2번째 비교 문서", { exact: true })
    .selectOption(other.id);
  await expect(
    page.locator(".workset-comparison .document-preview-panel"),
  ).toHaveCount(2);
  await expect(
    page
      .locator(".workset-comparison")
      .getByText("점검할 GPU 서버 운영 기록입니다."),
  ).toBeVisible();
  await shot("worksets-overview");
  await page
    .locator(".workset-comparison")
    .evaluate((el) => el.scrollIntoView({ block: "start" }));
  await shot("workset-reference-comparison");
  const item = page.locator(".workset-item").filter({
    has: page.getByRole("heading", { name: "제목 없는 문서", exact: true }),
  });
  await item.getByRole("button", { name: "문서 여권", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "문서 여권", exact: true });
  await expect(dialog.getByText("나만 보기", { exact: true })).toBeVisible();
  await expect(
    dialog.getByText("등록된 기간이 없습니다.", { exact: false }),
  ).toBeVisible();
  await shot("document-passport");
  await dialog.getByRole("button", { name: "닫기", exact: true }).click();
  await expect(
    item.getByRole("button", { name: "문서 여권", exact: true }),
  ).toBeFocused();
  await item.getByRole("button", { name: "문서 여권", exact: true }).click();
  await dialog
    .getByRole("button", { name: "본문에서 정리 후보 확인", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "본문 기반 정리 제안",
    exact: true,
  });
  await dialog
    .getByLabel("본문 기반 제목 후보", { exact: true })
    .selectOption("GPU 서버 운영");
  await dialog.getByLabel("운영", { exact: true }).check();
  await dialog
    .getByRole("button", { name: "선택한 변경 비교", exact: true })
    .click();
  await expect(
    dialog.getByText("제목·태그와 원문 변경 확인", { exact: true }),
  ).toBeVisible();
  assert.equal(
    (await api("/documents/" + doc.id)).markdown,
    md,
    "preview cannot mutate",
  );
  await expect(
    dialog.getByRole("button", { name: "확인한 정리 적용", exact: true }),
  ).toBeDisabled();
  await shot("document-cleanup-preview");
  await dialog
    .getByLabel("변경 전후를 확인했고 이 제목과 태그를 저장합니다.", {
      exact: true,
    })
    .check();
  await dialog
    .getByRole("button", { name: "확인한 정리 적용", exact: true })
    .click();
  await expect(dialog).not.toBeVisible();
  const saved = await api("/documents/" + doc.id);
  assert.equal(saved.title, "GPU 서버 운영");
  assert.equal(saved.visibility, "private");
  assert.ok(saved.markdown.endsWith(md.slice(md.indexOf("\n\n# GPU"))));
  assert.ok(
    saved.markdown.includes("keep-me") &&
      saved.markdown.includes("보존할 주석") &&
      saved.markdown.includes("기존별칭"),
  );
  await page.goto(base + `/app/documents/${doc.id}?mode=source`);
  const source = page.locator(".markdown-source");
  await expect(source).toHaveValue(saved.markdown);
  await source.evaluate((el) => {
    el.focus();
    const start = el.value.indexOf("점검할");
    el.setSelectionRange(start, start + 3);
  });
  await page.getByRole("tab", { name: "속성", exact: true }).click();
  await page
    .getByRole("button", { name: "작업 묶음에 보관", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "개인 작업 묶음에 보관",
    exact: true,
  });
  await dialog
    .getByLabel("새 묶음 이름", { exact: true })
    .fill("문서 위치 확인");
  await dialog.getByRole("button", { name: "참조 보관", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  const positionSet = (await api("/worksets?workspace_id=" + ws.id)).items.find(
    (v) => v.name === "문서 위치 확인",
  );
  assert.ok(positionSet);
  const positionData = await api("/worksets/" + positionSet.id),
    positionItem = positionData.workset.items[0];
  assert.equal(positionItem.context.version, saved.version);
  assert.equal(
    positionItem.context.line,
    saved.markdown.slice(0, saved.markdown.indexOf("점검할")).split("\n")
      .length,
  );
  await page.goto(base + "/app/worksets/" + positionSet.id);
  await page
    .getByRole("link", { name: "저장한 맥락으로 열기", exact: true })
    .click();
  await expect(page).toHaveURL(new RegExp("line=" + positionItem.context.line));
  await page.goto(base + `/app/databases/${db.id}?view=board`);
  await page
    .getByRole("button", { name: "현재 보기를 작업 묶음에 보관", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "개인 작업 묶음에 보관",
    exact: true,
  });
  await dialog
    .getByLabel("새 묶음 이름", { exact: true })
    .fill("점검 보드 업무");
  await expect(dialog.getByLabel("묶음 종류", { exact: true })).toHaveValue(
    "workset",
  );
  await dialog.getByRole("button", { name: "참조 보관", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  const sets = (await api("/worksets?workspace_id=" + ws.id)).items;
  const dbSet = sets.find((s) => s.name === "점검 보드 업무");
  assert.ok(dbSet);
  const dbData = await api("/worksets/" + dbSet.id);
  assert.equal(dbData.workset.items[0].context.view.view, "board");
  assert.match(dbData.resolved[0].url, /view=board/);
  await page.goto(base + "/app/worksets/" + dbSet.id);
  await page
    .getByRole("link", { name: "저장한 맥락으로 열기", exact: true })
    .click();
  await expect(page).toHaveURL(/view=board/);
  // Deleted workset CAS uses the version shown when confirmation was opened, not a later polling response.
  await page.goto(base + "/app/worksets/" + setID);
  await page.getByRole("button", { name: "묶음 삭제", exact: true }).click();
  dialog = page.getByRole("dialog", {
    name: "개인 작업 묶음 삭제",
    exact: true,
  });
  let before = (await api("/worksets/" + setID)).workset;
  await api("/worksets/" + setID, "PUT", {
    ...before,
    expected_version: before.version,
    name: "동시에 바뀐 참고 선반",
  });
  await expect(page.locator(".worksets-layout h2")).toHaveText(
    "동시에 바뀐 참고 선반",
  );
  await dialog
    .getByRole("button", { name: "묶음만 삭제", exact: true })
    .click();
  await expect(dialog).toBeVisible();
  await expect(
    dialog.getByText("작업 묶음이 변경되었습니다", { exact: true }),
  ).toBeVisible();
  assert.equal(
    (await api("/worksets/" + setID)).workset.name,
    "동시에 바뀐 참고 선반",
  );
  await dialog.getByRole("button", { name: "취소", exact: true }).click();
  // Reader keeps only their previously saved inaccessible slot; hidden source content disappears from comparison.
  await api("/admin/users", "POST", {
    email: "workset-reader@example.test",
    name: "참고 사용자",
    role: "viewer",
    password: "Worksets-Browser-Password-2026!",
  });
  await api(`/workspaces/${ws.id}/members`, "PUT", {
    email: "workset-reader@example.test",
    role: "viewer",
  });
  const reader = await browser.newContext({
    viewport: { width: 390, height: 844 },
    locale: "ko-KR",
  });
  await api(
    "/auth/login",
    "POST",
    {
      email: "workset-reader@example.test",
      password: "Worksets-Browser-Password-2026!",
    },
    200,
    reader,
  );
  const rs = await api(
    "/worksets",
    "POST",
    {
      workspace_id: ws.id,
      name: "내 참고 슬롯",
      kind: "reference",
      items: [
        {
          kind: "document",
          resource_id: other.id,
          context: { version: 1, mode: "read" },
        },
      ],
    },
    200,
    reader,
  );
  await api("/worksets/" + rs.id, "GET", undefined, 404);
  const rp = await reader.newPage();
  rp.on("pageerror", (e) => errors.push(e.message));
  await rp.goto(base + "/app/worksets/" + rs.id);
  await rp.getByRole("button", { name: "메뉴 열기", exact: true }).click();
  await rp.getByLabel("워크스페이스 선택", { exact: true }).selectOption(ws.id);
  await rp.goto(base + "/app/worksets/" + rs.id);
  await expect(
    rp.getByRole("heading", { name: "장애 대응 참고", exact: true }),
  ).toBeVisible();
  await rp
    .getByLabel("1번째 비교 문서", { exact: true })
    .selectOption(other.id);
  await expect(
    rp
      .locator(".workset-comparison")
      .getByText("문제를 확인하고 안전하게 복구합니다.", { exact: true }),
  ).toBeVisible();
  await api("/documents/" + other.id, "PUT", {
    version: 1,
    visibility: "private",
  });
  await expect(
    rp.getByRole("heading", { name: "현재 접근 불가", exact: true }),
  ).toBeVisible();
  await expect(rp.locator(".workset-comparison")).toHaveCount(0);
  assert.ok(
    !(await rp.locator(".worksets-layout").innerText()).includes(
      "장애 대응 참고",
    ),
  );
  const hidden = await api("/worksets/" + rs.id, "GET", undefined, 200, reader);
  assert.deepEqual(Object.keys(hidden.resolved[0]).sort(), [
    "available",
    "kind",
    "resource_id",
  ]);
  await rp.screenshot({
    path: path.join(out, "reference-unavailable-mobile.png"),
    animations: "disabled",
  });
  await reader.close();
  await page.goto(base + "/app/worksets/" + setID);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() =>
      page.evaluate(() => {
        const main = document.querySelector(".main");
        return main ? getComputedStyle(main).marginLeft : "";
      }),
    )
    .toBe("0px");
  await overflow();
  await shot("worksets-mobile");
  await page.goto(base + "/app");
  await expect(
    page.getByRole("region", { name: "최근 작업 묶음" }),
  ).toBeVisible();
  await page
    .getByRole("region", { name: "최근 작업 묶음" })
    .scrollIntoViewIfNeeded();
  await shot("home-worksets-mobile");
  assert.deepEqual(errors, []);
  console.log(
    "PASS U14: reference CRUD, server reload, native comparison, passport, explicit canonical cleanup, preserved Front Matter/body, saved board context, stale-delete CAS, current ACL retraction, private owner scope, 390px/home, JavaScript errors0",
  );
} catch (e) {
  await shot("failure");
  throw e;
} finally {
  await browser.close();
}
