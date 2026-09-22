import assert from "node:assert/strict";
import path from "node:path";
import { mkdir } from "node:fs/promises";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL,
  out =
    process.env.MADI_SCREENSHOT_DIR || "test-results/mobile-personalization";
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(url, method = "GET", data, status = 200) {
  const r = await context.request.fetch(base + "/api/v1" + url, {
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
async function noOverflow() {
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
  const ws = await api("/workspaces", "POST", {
    name: "모바일 접근성과 개인화 확인",
  });
  await page.goto(base + "/app");
  await page.getByRole("button", { name: "메뉴 열기", exact: true }).click();
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.getByRole("button", { name: "메뉴 닫기", exact: true }).click();
  const nav = page.getByRole("navigation", { name: "모바일 핵심 동작" });
  await expect(nav).toBeVisible();
  const sizes = await nav.locator("a,button").evaluateAll((nodes) =>
    nodes.map((n) => {
      const r = n.getBoundingClientRect();
      return {
        w: r.width,
        h: r.height,
        font: parseFloat(getComputedStyle(n).fontSize),
      };
    }),
  );
  assert.equal(sizes.length, 3);
  assert.ok(sizes.every((x) => x.w >= 44 && x.h >= 44 && x.font >= 16));
  await noOverflow();
  await shot("mobile-home-actions");
  const input = page.getByLabel("제목, 키워드 또는 궁금한 내용", {
    exact: true,
  });
  await input.focus();
  await expect(nav).not.toBeVisible();
  await page.getByRole("heading", { name: /어디서 이어갈까요/ }).click();
  await expect(nav).toBeVisible();
  await nav.getByRole("button", { name: "개인 메모", exact: true }).click();
  await expect(page).toHaveURL(/\/app\/documents\/[a-f0-9-]+\?mode=edit/);
  const id = new URL(page.url()).pathname.split("/").pop();
  assert.equal((await api("/documents/" + id)).visibility, "private");
  await page.goto(base + "/app/profile");
  await page.getByLabel("화면 밀도", { exact: true }).selectOption("relaxed");
  await page
    .getByLabel("모바일 데이터베이스 기본 보기", { exact: true })
    .selectOption("cards");
  await page.getByLabel("글자 크기", { exact: true }).selectOption("24");
  await page
    .getByRole("button", { name: "변경사항 저장", exact: true })
    .click();
  await expect
    .poll(async () => (await api("/auth/me")).preferences.density)
    .toBe("relaxed");
  await page.reload();
  await expect(page.getByLabel("화면 밀도", { exact: true })).toHaveValue(
    "relaxed",
  );
  await expect(page.locator("html")).toHaveAttribute("data-density", "relaxed");
  await noOverflow();
  await shot("mobile-preferences-readable");
  await api("/profile", "PUT", { preferences: { font_size: 16 } });
  const db = await api("/databases", "POST", {
      workspace_id: ws.id,
      name: "모바일 일정",
      properties: [
        { id: "name", name: "이름", type: "text" },
        { id: "date", name: "검토일", type: "date" },
        {
          id: "state",
          name: "상태",
          type: "select",
          options: ["대기", "완료"],
        },
      ],
    }),
    row = await api(`/databases/${db.id}/rows`, "POST", {
      values: {
        name: "담당자가 이해하기 쉽게 읽고 확인하는 긴 한글 항목",
        date: "2026-09-01",
        state: "대기",
      },
    });
  await page.goto(`${base}/app/databases/${db.id}`);
  await expect(page.getByLabel("모바일 표 표시", { exact: true })).toHaveValue(
    "cards",
  );
  await expect(page.getByLabel("데이터베이스 행 카드")).toBeVisible();
  await noOverflow();
  await page.getByLabel("데이터베이스 행 카드").scrollIntoViewIfNeeded();
  await shot("mobile-database-cards");
  await page
    .getByRole("button", { name: "1행 항목 상세", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "항목 편집", exact: true });
  await expect(nav).not.toBeVisible();
  await dialog.getByText("말로 날짜 찾기", { exact: true }).click();
  await dialog
    .getByLabel("검토일 날짜 표현", { exact: true })
    .fill("다음주 월요일");
  await expect(
    dialog.getByText("기준 시간대: Asia/Seoul", { exact: false }),
  ).toBeVisible();
  assert.equal(
    (await api(`/databases/${db.id}/rows`))[0].values.date,
    "2026-09-01",
  );
  await shot("mobile-date-confirmation");
  await dialog
    .getByRole("button", { name: "확인한 날짜 입력", exact: true })
    .click();
  const proposed = await dialog
    .getByLabel("검토일", { exact: true })
    .inputValue();
  assert.match(proposed, /^\d{4}-\d{2}-\d{2}$/);
  assert.equal(
    (await api(`/databases/${db.id}/rows`))[0].values.date,
    "2026-09-01",
  );
  await dialog
    .getByLabel("검토일 날짜 표현", { exact: true })
    .fill("2026-02-29");
  await expect(
    dialog.getByText("날짜를 확정할 수 없습니다.", { exact: false }),
  ).toBeVisible();
  await shot("mobile-date-invalid");
  await dialog.getByRole("button", { name: "저장", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect
    .poll(
      async () =>
        (await api(`/databases/${db.id}/rows`)).find((r) => r.id === row.id)
          .values.date,
    )
    .toBe(proposed);
  await expect(
    page.getByRole("button", { name: "1행 항목 상세", exact: true }),
  ).toBeFocused();
  await page
    .getByLabel("모바일 표 표시", { exact: true })
    .selectOption("table");
  await expect(page.getByRole("grid")).toBeVisible();
  await noOverflow();
  await expect
    .poll(async () => (await api("/auth/me")).preferences.mobile_table_view)
    .toBe("table");
  await page.reload();
  await expect(page.getByLabel("모바일 표 표시", { exact: true })).toHaveValue(
    "table",
  );
  await page.goto(base + "/app");
  await page.setViewportSize({ width: 320, height: 720 });
  await noOverflow();
  await shot("mobile-home-320");
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto(base + "/app/profile");
  await page.evaluate(() => (document.documentElement.style.zoom = "2"));
  await expect(page.getByLabel("화면 밀도", { exact: true })).toBeVisible();
  // CSS 200% zoom is an automated enlargement probe, not a full browser/assistive-technology audit.
  await expect
    .poll(() =>
      page
        .getByRole("button", { name: "변경사항 저장", exact: true })
        .isEnabled(),
    )
    .toBe(true);
  await shot("profile-css-zoom-200");
  await page.evaluate(() => (document.documentElement.style.zoom = ""));
  await page.goto(base + "/admin");
  await expect(nav).not.toBeVisible();
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      mobileActions: true,
      inputAndDialogClearance: true,
      privateCapture: true,
      serverPreferences: true,
      cardsAndTable: true,
      dateExplicitSave: true,
      focusReturn: true,
      reflow320: true,
      cssZoomProbe: true,
      jsErrors: 0,
    }),
  );
} catch (e) {
  await shot("mobile-personalization-failure");
  console.error(e);
  process.exitCode = 1;
} finally {
  await browser.close();
}
