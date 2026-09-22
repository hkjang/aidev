import assert from "node:assert/strict";
import path from "node:path";
import { mkdir } from "node:fs/promises";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL,
  out = process.env.MADI_SCREENSHOT_DIR || "test-results/knowledge-conflicts";
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1440, height: 1050 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(url, method = "GET", data, status = 200, ctx = context) {
  const response = await ctx.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(response.status(), status, await response.text());
  return response.json();
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(out, name + ".png"),
    animations: "disabled",
  });
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
      name: "문서 차이와 개인 판단 확인",
    }),
    docs = [];
  for (const [name, n] of [
    ["기존 보관 정책", 30],
    ["운영 보관 정책", 90],
  ])
    docs.push(
      await api("/documents", "POST", {
        workspace_id: ws.id,
        title: name,
        markdown:
          "# 기록 보관 기준\n\n서비스 기록의 기본 보관 기간은 " +
          n +
          "일입니다.\n\n```text\n비교에서 제외할 예제는 999일입니다.\n```\n",
        visibility: "workspace",
      }),
    );
  await page.goto(base + "/app/knowledge-conflicts");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(base + "/app/knowledge-conflicts");
  await page.getByRole("button", { name: "새 비교", exact: true }).click();
  let dialog = page.getByRole("dialog", { name: "새 문서 비교", exact: true });
  for (const doc of docs)
    await dialog.getByRole("checkbox", { name: new RegExp(doc.title) }).check();
  await expect(
    dialog.getByRole("button", { name: "비교 보고서 만들기", exact: true }),
  ).toBeDisabled();
  await dialog.getByRole("checkbox", { name: /선택 원문의 구간/ }).check();
  await shot("conflicts-selection");
  await dialog
    .getByRole("button", { name: "비교 보고서 만들기", exact: true })
    .click();
  await expect(page.locator(".conflict-candidate")).toHaveCount(1);
  const reportID = new URL(page.url()).searchParams.get("id");
  assert.ok(reportID);
  await expect(page.getByText("수치·단위 차이", { exact: true })).toBeVisible();
  await shot("conflicts-comparison");
  await page.getByRole("button", { name: "판단 기록", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "개인 판단 기록", exact: true });
  await dialog
    .getByLabel("차이 판단", { exact: true })
    .selectOption("compatible");
  await dialog
    .getByLabel("차이 판단", { exact: true })
    .selectOption("conflict");
  await dialog
    .getByLabel("판단 근거", { exact: true })
    .fill(
      "같은 서비스의 동일한 기록 범위인지 담당자에게 확인한 뒤 정책을 정리해야 합니다.",
    );
  await dialog.getByRole("checkbox", { name: /두 구간을 확인/ }).check();
  await shot("conflicts-review");
  await dialog.getByRole("button", { name: "판단 저장", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await page
    .getByRole("button", { name: "판단 이력 보기", exact: true })
    .click();
  await expect(page.locator(".conflict-history li")).toHaveCount(1);
  await shot("conflicts-history");
  for (const doc of docs) {
    const current = await api(`/documents/${doc.id}`);
    assert.equal(current.version, 1);
    assert.equal(current.markdown, doc.markdown);
  }
  const candidateID = new URL(page.url()).searchParams.get("candidate");
  assert.ok(candidateID);
  await page
    .getByRole("link", { name: "첫째 원문 변경안", exact: true })
    .click();
  await expect(
    page.getByText("비교 후보를 변경안의 출처로 연결", { exact: true }),
  ).toBeVisible();
  await page
    .getByLabel("변경 이유", { exact: true })
    .fill("기록 분류 담당자가 범위 차이를 확인할 변경 초안입니다.");
  await page
    .getByLabel("제안 Markdown 원문", { exact: true })
    .fill("# 기록 보관 기준\n\n서비스 기록의 기본 보관 기간은 60일입니다.\n");
  await page.getByRole("checkbox", { name: /현재 문서 열람자가/ }).check();
  await expect(
    page.getByRole("button", { name: "공유 전 변경 비교", exact: true }),
  ).toBeDisabled();
  await page.getByRole("checkbox", { name: /비교 출처를 연결/ }).check();
  await shot("conflicts-proposal-consent");
  await page
    .getByRole("button", { name: "공유 전 변경 비교", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "원문을 유지하고 변경안 공유",
    exact: true,
  });
  await dialog
    .getByRole("button", { name: "동의한 범위로 변경안 공유", exact: true })
    .click();
  await expect(
    page.getByText("문서 차이 검토에서 연결한 변경안", { exact: true }),
  ).toBeVisible();
  const proposalID = new URL(page.url()).searchParams.get("id");
  assert.ok(proposalID);
  await shot("conflicts-proposal-origin");
  await api(`/documents/${docs[1].id}`, "PUT", {
    version: 1,
    markdown:
      "# 기록 보관 기준\n\n서비스 기록의 기본 보관 기간은 120일입니다.\n",
  });
  await page.reload();
  await expect(
    page.getByRole("button", { name: "비교 후 변경안 반영", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByText("비교 출처 또는 판단이 바뀌어 원문 반영이 중단되었습니다.", {
      exact: true,
    }),
  ).toBeVisible();
  await shot("conflicts-stale-origin");
  await page.goto(
    base + `/app/knowledge-conflicts?id=${reportID}&candidate=${candidateID}`,
  );
  await expect(
    page.getByRole("button", { name: "판단 기록", exact: true }),
  ).toBeDisabled();
  await page.reload();
  await expect(page.locator(".conflict-history li")).toHaveCount(1);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() =>
      page.evaluate(() => {
        const main = document.querySelector(".main");
        return main ? getComputedStyle(main).marginLeft : "";
      }),
    )
    .toBe("0px");
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    )
    .toBe(true);
  await shot("conflicts-mobile");
  await page.locator(".conflict-candidate").scrollIntoViewIfNeeded();
  await shot("conflicts-mobile-comparison");
  await api("/admin/users", "POST", {
    email: "revocation@example.test",
    name: "새 문서 소유자",
    role: "editor",
    password: "Conflict-Revocation-Password!",
  });
  await api(`/workspaces/${ws.id}/members`, "PUT", {
    email: "revocation@example.test",
    role: "editor",
  });
  const users = await api("/admin/users"),
    otherUser = users.find((u) => u.email === "revocation@example.test");
  assert.ok(otherUser);
  // Real current ACL removal from the report owner; admin role is not a bypass.
  const beforeOwnership = await api(`/documents/${docs[1].id}`);
  await api(
    `/documents/${docs[1].id}/knowledge`,
    "PUT",
    { version: beforeOwnership.version, owner_id: otherUser.id },
    200,
  );
  const current = await api(`/documents/${docs[1].id}`);
  assert.equal(current.owner_id, otherUser.id);
  const otherContext = await browser.newContext();
  await api(
    "/auth/login",
    "POST",
    {
      email: "revocation@example.test",
      password: "Conflict-Revocation-Password!",
    },
    200,
    otherContext,
  );
  await api(
    `/documents/${docs[1].id}`,
    "PUT",
    { version: current.version, visibility: "private" },
    200,
    otherContext,
  );
  await expect(page.locator(".conflict-candidate")).toHaveCount(0);
  await expect(page.getByText(/서비스 기록의 기본 보관 기간은/)).toHaveCount(0);
  await shot("conflicts-access-removed");
  await otherContext.close();
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      passed: true,
      report_id: reportID,
      source_changes: "only explicit test PUT",
      js_errors: errors.length,
    }),
  );
} catch (error) {
  await shot("failure").catch(() => {});
  throw error;
} finally {
  await context.close();
  await browser.close();
}
