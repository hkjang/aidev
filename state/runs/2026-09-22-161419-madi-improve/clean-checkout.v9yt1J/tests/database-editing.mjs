import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  out = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/database-editing",
  );
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1040 },
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
async function paste(cell, text) {
  await cell.focus();
  await cell.evaluate((node, text) => {
    const clipboardData = new DataTransfer();
    clipboardData.setData("text/plain", text);
    node.dispatchEvent(
      new ClipboardEvent("paste", {
        clipboardData,
        bubbles: true,
        cancelable: true,
      }),
    );
  }, text);
  return page.getByRole("dialog", { name: "범위 붙여넣기 검토", exact: true });
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
      name: "빠르고 안전한 데이터베이스 편집",
    }),
    db = await api("/databases", "POST", {
      workspace_id: ws.id,
      name: "팀 작업 현황",
      properties: [
        { id: "title", name: "이름", type: "text" },
        { id: "amount", name: "수량", type: "number" },
        {
          id: "state",
          name: "상태",
          type: "select",
          options: ["대기", "완료"],
        },
        { id: "due", name: "예정일", type: "date" },
      ],
    });
  const first = await api(`/databases/${db.id}/rows`, "POST", {
      values: { title: "첫 항목", amount: 1, state: "대기", due: "2026-09-10" },
    }),
    second = await api(`/databases/${db.id}/rows`, "POST", {
      values: {
        title: "둘째 항목",
        amount: 2,
        state: "대기",
        due: "2026-09-11",
      },
    });
  await page.goto(base + "/app");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(`${base}/app/databases/${db.id}`);
  const cell = (r, name) =>
    page.getByRole("gridcell", { name: `${r}행 ${name}`, exact: true });
  await cell(1, "수량").focus();
  await cell(1, "수량").press("Enter");
  let input = page.getByLabel("수량 셀 입력", { exact: true });
  await input.fill("10");
  await input.press("Tab");
  await expect(cell(1, "상태")).toBeFocused();
  await expect
    .poll(
      async () =>
        (await api(`/databases/${db.id}/rows`)).find((r) => r.id === first.id)
          .values.amount,
    )
    .toBe(10);
  await cell(1, "상태").press("Enter");
  const stateInput = page.getByLabel("상태 셀 입력", { exact: true });
  await stateInput.selectOption("완료");
  const stateSaved = page.waitForResponse(
    (response) =>
      response.url() === `${base}/api/v1/databases/${db.id}/rows/${first.id}` &&
      response.request().method() === "PUT",
  );
  await page.getByRole("button", { name: "셀 저장", exact: true }).click();
  if (process.env.MADI_DATABASE_EDITING_ACK_GATE === "1") {
    try {
      await expect
        .poll(async () => {
          const response = await context.request.get(
            base + "/__test/database-editing/ack",
          );
          assert.equal(response.status(), 200);
          return (await response.json()).committed;
        })
        .toBe(true);
      // The real CAS write has committed, but no ACK has reached the UI.
      // Native option text already includes "완료": cell text alone is NOT
      // a save-completion condition. Editing another cell is still rejected.
      await expect(stateInput).toBeDisabled();
      await expect(stateInput).toHaveValue("완료");
      await expect(cell(1, "상태")).toContainText("완료");
      await expect(
        cell(1, "상태").getByRole("button", { name: "완료", exact: true }),
      ).toHaveCount(0);
      await cell(1, "수량").press("Enter");
      await expect(page.getByLabel("수량 셀 입력", { exact: true })).toHaveCount(0);
      await expect(page.getByRole("alert")).toContainText(
        "편집 중인 셀을 저장하거나 취소한 뒤",
      );
    } finally {
      const released = await context.request.post(
        base + "/__test/database-editing/ack",
      );
      assert.equal(released.status(), 200);
    }
  }
  const stateResponse = await stateSaved;
  assert.equal(stateResponse.status(), 200, await stateResponse.text());
  const savedState = await stateResponse.json();
  assert.equal(savedState.version, first.version + 2);
  assert.equal(savedState.values.state, "완료");
  // Wait for both the old editor to close and the refreshed read-only cell
  // and focus to appear. save() restores focus only after onChanged resolves.
  await expect(stateInput).toHaveCount(0);
  await expect(
    cell(1, "상태").getByRole("button", { name: "완료", exact: true }),
  ).toBeVisible();
  await expect(cell(1, "상태")).toBeFocused();
  const confirmed = (await api(`/databases/${db.id}/rows`)).find(
    (row) => row.id === first.id,
  );
  assert.equal(confirmed.version, savedState.version);
  assert.equal(confirmed.values.state, "완료");
  await cell(1, "수량").press("Enter");
  input = page.getByLabel("수량 셀 입력", { exact: true });
  await expect(input).toBeEnabled();
  await input.fill("20");
  let latest = (await api(`/databases/${db.id}/rows`)).find(
    (r) => r.id === first.id,
  );
  await api(`/databases/${db.id}/rows/${first.id}`, "PUT", {
    expected_version: latest.version,
    values: { amount: 99 },
  });
  await input.press("Enter");
  await expect(page.getByRole("alert")).toContainText("입력은 보관");
  await expect(input).toHaveValue("20");
  await shot("database-inline-conflict");
  await page.getByRole("button", { name: "셀 취소", exact: true }).click();
  await page.reload();
  await expect(cell(1, "수량")).toContainText("99");
  let modal = await paste(cell(1, "수량"), "12\t알수없음\n13\t완료");
  await expect(modal).toBeVisible();
  await expect(modal).toContainText("등록된 선택 옵션이 아닙니다");
  await expect(
    modal.getByRole("button", { name: "현재 행과 타입 검사", exact: true }),
  ).toBeDisabled();
  await shot("database-range-errors");
  await modal.getByLabel("2번 셀 상태", { exact: true }).fill("대기");
  await modal
    .getByRole("button", { name: "현재 행과 타입 검사", exact: true })
    .click();
  await expect(
    modal.getByRole("heading", { name: "모든 셀 검사 완료", exact: true }),
  ).toBeVisible();
  await expect(
    modal.getByRole("button", { name: "확인한 셀 모두 저장", exact: true }),
  ).toBeDisabled();
  await modal.getByRole("checkbox").check();
  await shot("database-range-review");
  await modal
    .getByRole("button", { name: "확인한 셀 모두 저장", exact: true })
    .click();
  await expect(modal).not.toBeVisible();
  let rows = await api(`/databases/${db.id}/rows`);
  assert.equal(rows.find((r) => r.id === first.id).values.amount, 12);
  assert.equal(rows.find((r) => r.id === second.id).values.amount, 13);
  await page
    .getByRole("button", { name: "1행 항목 상세", exact: true })
    .click();
  modal = page.getByRole("dialog", { name: "항목 편집", exact: true });
  await expect(modal).toBeVisible();
  await modal.getByLabel("이름", { exact: true }).fill("패널에서 편집");
  await modal.getByRole("button", { name: "저장", exact: true }).click();
  await expect(modal).not.toBeVisible();
  await expect(cell(1, "이름")).toContainText("패널에서 편집");
  await expect(
    page.getByRole("button", { name: "1행 항목 상세", exact: true }),
  ).toBeFocused();
  await page
    .getByRole("button", { name: "현재 보기 저장", exact: true })
    .click();
  modal = page.getByRole("dialog", { name: "보기 구성 저장", exact: true });
  await modal.getByLabel("보기 이름", { exact: true }).fill("나의 업무 표");
  await modal
    .getByRole("button", { name: "확인한 보기 저장", exact: true })
    .click();
  await expect(modal).not.toBeVisible();
  await page
    .getByRole("button", { name: "내 기본 보기로 설정", exact: true })
    .click();
  const own = await api(`/databases/${db.id}/views`);
  assert.ok(own.default_view_id);
  assert.equal(own.views[0].visibility, "private");
  await page.getByRole("button", { name: "보드", exact: true }).click();
  await expect(page).toHaveURL(/view=board/);
  await page
    .getByRole("button", { name: "현재 보기 저장", exact: true })
    .click();
  modal = page.getByRole("dialog", { name: "보기 구성 저장", exact: true });
  await modal.getByLabel("보기 이름", { exact: true }).fill("팀 보드");
  await modal
    .getByLabel("보기 공개 범위", { exact: true })
    .selectOption("workspace");
  await expect(
    modal.getByRole("button", { name: "확인한 보기 저장", exact: true }),
  ).toBeDisabled();
  await modal.getByRole("checkbox").check();
  await shot("database-team-view-review");
  await modal
    .getByRole("button", { name: "확인한 보기 저장", exact: true })
    .click();
  await expect(modal).not.toBeVisible();
  let views = await api(`/databases/${db.id}/views`);
  const shared = views.views.find((v) => v.name === "팀 보드");
  assert.equal(shared.visibility, "workspace");
  assert.equal(shared.data.view, "board");
  // Loading the bare route applies only this user's server-side default.
  await page.goto(`${base}/app/databases/${db.id}`);
  await expect(page).toHaveURL(/view=table/);
  await expect(page.getByLabel("저장된 보기", { exact: true })).toHaveValue(
    own.default_view_id,
  );
  await page.reload();
  await expect(page.getByLabel("저장된 보기", { exact: true })).toHaveValue(
    own.default_view_id,
  );
  // A settings dialog retains its opening version even if polling sees a newer shared view.
  await page.getByLabel("저장된 보기", { exact: true }).selectOption(shared.id);
  await page
    .getByRole("button", { name: "현재 보기 저장", exact: true })
    .click();
  modal = page.getByRole("dialog", { name: "보기 구성 저장", exact: true });
  await modal
    .getByLabel("보기 저장 방식", { exact: true })
    .selectOption("update");
  await modal
    .getByLabel("보기 이름", { exact: true })
    .fill("내가 작성 중인 제목");
  await modal.getByRole("checkbox").check();
  await api(`/databases/${db.id}/views/${shared.id}`, "PUT", {
    name: "외부에서 갱신한 팀 보기",
    visibility: "workspace",
    data: shared.data,
    expected_version: shared.version,
    share_consent: true,
  });
  await modal
    .getByRole("button", { name: "확인한 보기 저장", exact: true })
    .click();
  await expect(modal.getByRole("alert")).toContainText(/변경|다시/);
  await expect(modal.getByLabel("보기 이름", { exact: true })).toHaveValue(
    "내가 작성 중인 제목",
  );
  await modal.getByRole("button", { name: "취소", exact: true }).click();
  await page.getByRole("button", { name: "테이블", exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(
    () =>
      getComputedStyle(document.querySelector(".main")).marginLeft === "0px",
  );
  await page
    .getByRole("button", { name: "1행 항목 상세", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    )
    .toBe(true);
  await shot("database-row-panel-mobile");
  await page
    .getByRole("button", { name: "항목 상세 닫기", exact: true })
    .click();
  await page.goto(`${base}/app/databases/${db.id}`);
  await expect(page.getByLabel("저장된 보기", { exact: true })).toHaveValue(
    own.default_view_id,
  );
  await page.evaluate(() => {
    window.scrollTo(0, 0);
    document.querySelectorAll(".table-scroll").forEach((el) => {
      el.scrollLeft = 0;
    });
  });
  await shot("database-private-views-mobile");
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      inlineKeyboard: true,
      heldCommitAck: process.env.MADI_DATABASE_EDITING_ACK_GATE === "1",
      cellCAS: true,
      typedAtomicPaste: true,
      rowPanel: true,
      privateDefaults: true,
      explicitSharedSave: true,
      viewCAS: true,
      mobile: true,
      jsErrors: errors.length,
    }),
  );
} catch (e) {
  await shot("database-editing-failure");
  console.error(e);
  process.exitCode = 1;
} finally {
  await browser.close();
}
