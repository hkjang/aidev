import { documentTool, documentPanel } from "./document-ui.mjs";
import assert from "node:assert/strict";
import { chromium } from "playwright";
import { expect } from "playwright/test";
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const screenshots = pathToFileURL(
  resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots") + "/",
);
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
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${path}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function wait(fn) {
  const start = Date.now();
  while (Date.now() - start < 20000) {
    const result = await fn();
    if (result) return result;
    await page.waitForTimeout(150);
  }
  throw Error("Timed out waiting for persisted UI state");
}
async function shot(name) {
  await mkdir(screenshots, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  if (await page.locator(".toast .icon-button").count())
    await page
      .locator(".toast .icon-button")
      .click({ timeout: 1000 })
      .catch(() => {});
  const fixedOverlay =
    (await page.getByRole("dialog").count()) > 0 ||
    (await page.locator(".app-layout.mobile-open").count()) > 0;
  await page.screenshot({
    path: new URL(`${name}.png`, screenshots).pathname,
    fullPage: !fixedOverlay,
    animations: "disabled",
  });
}
async function command(name) {
  await page.keyboard.press("Control+k");
  const input = page.getByRole("combobox", { name: "문서 또는 명령 검색" });
  await input.fill(name);
  await expect(page.getByRole("option", { selected: true })).toContainText(
    name,
  );
  await input.press("Enter");
}
async function saved(id) {
  await page
    .locator('.collaboration-bar [data-save-state="confirmed"]')
    .waitFor();
  return api("/documents/" + id);
}
let user;
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const email = `navigation-${Date.now()}@example.test`,
    password = "Navigation-Test-Password-2026!";
  user = await api("/admin/users", "POST", {
    email,
    password,
    name: "지식 탐색 사용자",
    role: "editor",
    kind: "user",
  });
  await api("/auth/logout", "POST", {});
  await api("/auth/login", "POST", { email, password });
  const ws = await api("/workspaces", "POST", {
    name: "탐색과 편집 워크스페이스",
  });
  const make = (title, markdown, extra = {}) =>
    api("/documents", "POST", {
      workspace_id: ws.id,
      title,
      markdown,
      visibility: "workspace",
      ...extra,
    });
  const parent = await make(
    "프로젝트 지식",
    "첫 번째 생각\n\n두 번째 생각\n\n세 번째 생각\n\n## 다음 장\n\n발표 마무리",
  );
  const child = await make("검토 체크리스트", "- [ ] 동작 확인", {
    parent_id: parent.id,
  });
  const privateDoc = await make("개인 아이디어", "비공개 메모", {
    visibility: "private",
  });
  await page.goto(base + "/app");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(base + "/app/documents/" + parent.id + "?mode=preview");
  await page.getByLabel("문서 제목", { exact: true }).waitFor();
  const tree = page.getByRole("tree", { name: "페이지 계층" });
  if (process.env.MADI_NAV_MENU_BUTTON)
    await tree
      .getByRole("button", { name: "프로젝트 지식 문서 메뉴", exact: true })
      .click();
  else
    await tree
      .locator(`a[data-tree-link="${parent.id}"]`)
      .click({ button: "right" });
  await page
    .getByRole("menuitem", { name: "사이드바에 고정", exact: true })
    .click();
  await wait(async () =>
    ((await api("/auth/me")).preferences.pinned_documents || []).includes(
      parent.id,
    ),
  );
  await tree
    .getByRole("button", { name: "프로젝트 지식 접기", exact: true })
    .click();
  await wait(
    async () =>
      (await api("/auth/me")).preferences.collapsed_documents?.[parent.id],
  );
  await page.reload();
  await page.getByLabel("문서 제목", { exact: true }).waitFor();
  assert.equal(
    await tree.locator(`a[data-tree-link="${child.id}"]`).count(),
    0,
  );
  assert.ok(await page.getByText("고정 문서", { exact: true }).isVisible());
  await tree
    .getByRole("button", { name: "프로젝트 지식 펼치기", exact: true })
    .click();
  await tree.locator(`a[data-tree-link="${child.id}"]`).click();
  await wait(
    async () =>
      (await api("/auth/me")).preferences.recent_documents?.[0] === child.id,
  );
  await page
    .getByLabel("문서 탐색 필터", { exact: true })
    .selectOption("private");
  assert.equal(
    await tree.locator(`a[data-tree-link="${parent.id}"]`).count(),
    0,
  );
  assert.equal(
    await tree.locator(`a[data-tree-link="${privateDoc.id}"]`).count(),
    1,
  );
  await page.reload();
  await page.getByLabel("문서 탐색 필터", { exact: true }).waitFor();
  assert.equal(
    await page.getByLabel("문서 탐색 필터", { exact: true }).inputValue(),
    "private",
  );
  await page.getByLabel("문서 탐색 필터", { exact: true }).selectOption("all");
  const separator = page.getByRole("separator", { name: "사이드바 너비 조정" }),
    rect = await separator.boundingBox();
  await page.mouse.move(rect.x + 4, 200);
  await page.mouse.down();
  await page.mouse.move(320, 200, { steps: 6 });
  await page.mouse.up();
  await wait(
    async () =>
      Number((await api("/auth/me")).preferences.sidebar_width) === 320,
  );
  await page.getByRole("button", { name: "메뉴 접기", exact: true }).click();
  await wait(
    async () => (await api("/auth/me")).preferences.sidebar_collapsed === true,
  );
  await page.reload();
  await page
    .getByRole("button", { name: "메뉴 펼치기", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "메뉴 펼치기", exact: true }).click();
  console.log(
    "PASS document context menu, pin/recent/private filter, nested tree state and resized/collapsed sidebar persist in server preferences",
  );
  await page.keyboard.press("Control+p");
  const combo = page.getByRole("combobox", { name: "문서 또는 명령 검색" });
  await combo.fill("프로젝트 지식");
  await expect(page.getByRole("option", { selected: true })).toContainText(
    "프로젝트 지식",
  );
  await combo.press("Enter");
  await page.waitForURL("**/documents/" + parent.id);
  await expect(page.locator(".document-main")).toHaveAttribute(
    "data-document-id",
    parent.id,
  );
  await expect(page.getByLabel("문서 제목", { exact: true })).toHaveValue(
    "프로젝트 지식",
  );
  await page.keyboard.press("Control+/");
  await page.getByRole("dialog", { name: "키보드로 더 빠르게" }).waitFor();
  await page.keyboard.press("Escape");
  await command("집중 모드 전환");
  assert.ok(page.url().includes("view=focus"));
  await page.reload();
  await page.locator(".focus-mode").waitFor();
  await documentTool(page, "집중 모드");
  await expect(page.locator(".document-page")).not.toHaveClass(/focus-mode/);
  await command("프레젠테이션 보기");
  await page
    .getByRole("dialog", { name: "프로젝트 지식", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "다음", exact: true }).click();
  assert.ok(page.url().includes("slide=1"));
  await page.reload();
  await page.getByRole("article", { name: "슬라이드 2" }).waitFor();
  await shot("presentation");
  await page.getByRole("button", { name: "발표 종료" }).click();
  await page
    .getByRole("dialog", { name: "프로젝트 지식", exact: true })
    .waitFor({ state: "hidden" });
  await page.keyboard.press("Control+k");
  await page
    .getByRole("combobox", { name: "문서 또는 명령 검색" })
    .fill("문서");
  await shot("command-palette");
  await page.keyboard.press("Escape");
  await page.goto(base + "/app/profile");
  const shortcut = page.getByLabel("문서 바로 열기", { exact: true });
  await shortcut.focus();
  await shortcut.press("Control+j");
  await page
    .getByRole("button", { name: "변경사항 저장", exact: true })
    .click();
  await wait(
    async () =>
      (await api("/auth/me")).preferences.keyboard_shortcuts?.quickOpen ===
      "Mod+J",
  );
  await page.getByRole("heading", { name: "개인 설정", exact: true }).click();
  await page.keyboard.press("Control+j");
  await page.getByRole("dialog", { name: "어디로 연결할까요?" }).waitFor();
  await page.keyboard.press("Escape");
  await shot("keyboard-settings");
  console.log(
    "PASS keyboard palette/quick open/help/custom binding and refresh-stable focus/presentation slides",
  );
  await page.goto(base + "/app/documents/" + parent.id + "?line=3");
  await page.getByLabel("Markdown 원문 편집", { exact: true }).waitFor();
  await wait(
    async () =>
      (await page
        .getByLabel("Markdown 원문 편집", { exact: true })
        .evaluate((input) =>
          input.value.slice(input.selectionStart, input.selectionEnd),
        )) === "두 번째 생각",
  );
  await shot("source-location");
  console.log(
    "PASS search deep link opens Markdown source and selects the requested line",
  );
  await page.goto(base + "/app/documents/" + parent.id + "?mode=edit");
  await page.locator(".collaboration-bar[data-state=connected]").waitFor();
  await page.getByRole("button", { name: "블록 정리", exact: true }).click();
  const manager = page.getByRole("dialog", { name: "블록 정리", exact: true }),
    rows = manager.locator(".organizer-row");
  await wait(async () => (await rows.count()) === 5);
  const originalIDs = await rows.evaluateAll((nodes) =>
    nodes.map((n) => n.dataset.blockId),
  );
  await manager.getByLabel("블록 1 선택", { exact: true }).check();
  await manager.getByLabel("블록 2 선택", { exact: true }).check();
  await manager.getByRole("button", { name: "인용 안에 중첩" }).click();
  await wait(async () => (await rows.count()) === 6);
  assert.equal(await rows.nth(1).getAttribute("aria-level"), "2");
  await manager.getByRole("button", { name: "블록 실행 취소" }).click();
  await wait(async () => (await rows.count()) === 5);
  await manager.getByRole("button", { name: "블록 다시 실행" }).click();
  await wait(async () => (await rows.count()) === 6);
  await manager.getByLabel("전체 선택", { exact: true }).check();
  await manager.getByLabel("전체 선택", { exact: true }).uncheck();
  await manager.getByLabel("블록 1 선택", { exact: true }).check();
  await manager.getByLabel("블록 2 선택", { exact: true }).check();
  await manager.getByRole("button", { name: "복제", exact: true }).click();
  await wait(async () => (await rows.count()) === 9);
  const clonedIDs = await rows.evaluateAll((nodes) =>
    nodes.map((n) => n.dataset.blockId),
  );
  assert.equal(new Set(clonedIDs).size, clonedIDs.length);
  await manager.getByRole("button", { name: "블록 실행 취소" }).click();
  await wait(async () => (await rows.count()) === 6);
  await manager.getByLabel("전체 선택", { exact: true }).check();
  await manager.getByLabel("전체 선택", { exact: true }).uncheck();
  await manager.getByLabel("블록 2 선택", { exact: true }).check();
  await manager.getByLabel("블록 3 선택", { exact: true }).check();
  await manager.getByRole("button", { name: "한 단계 밖으로" }).click();
  await wait(async () => (await rows.count()) === 7);
  await manager.getByRole("button", { name: "블록 실행 취소" }).click();
  await wait(async () => (await rows.count()) === 6);
  await manager.getByLabel("전체 선택", { exact: true }).check();
  await manager.getByLabel("전체 선택", { exact: true }).uncheck();
  await manager
    .getByLabel("블록 놓는 위치", { exact: true })
    .selectOption("inside");
  await manager
    .getByRole("button", { name: "블록 4 드래그 손잡이", exact: true })
    .dragTo(rows.first());
  await wait(
    async () => (await rows.nth(3).getAttribute("aria-level")) === "2",
  );
  await manager.getByLabel("블록 2 선택", { exact: true }).check();
  await manager.getByLabel("블록 3 선택", { exact: true }).check();
  await shot("block-organizer");
  await page.keyboard.press("Escape");
  const stored = await saved(parent.id);
  for (const id of originalIDs)
    assert.ok(stored.block_metadata.blocks.some((block) => block.id === id));
  await page.reload();
  await page.locator(".collaboration-bar[data-state=connected]").waitFor();
  await page.getByRole("button", { name: "블록 정리", exact: true }).click();
  await wait(
    async () =>
      (await manager
        .locator(".organizer-row")
        .nth(3)
        .getAttribute("aria-level")) === "2",
  );
  await page.keyboard.press("Escape");
  console.log(
    "PASS real CRDT nested multi-block drag/outdent/duplicate with fresh IDs, undo/redo, persisted Markdown and block IDs after refresh",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(base + "/app/documents/" + parent.id + "?mode=preview");
  await page.getByLabel("문서 제목", { exact: true }).waitFor();
  await page.getByRole("button", { name: "메뉴 열기", exact: true }).click();
  await shot("mobile-navigation");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  assert.deepEqual(errors, []);
  console.log("All navigation and block UX checks passed.");
} catch (error) {
  await page
    .screenshot({ path: ".local/navigation-failure.png", fullPage: true })
    .catch(() => {});
  throw error;
} finally {
  await browser.close();
}
