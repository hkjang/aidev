import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  out = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/ai-selection",
  );
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1040 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const errors = [];
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
async function capture(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(out, name + ".png"),
    animations: "disabled",
  });
}
const phrase = "선택해서 다듬을 원문 😀",
  answer = "명료해진 제안 문장 😀";
async function select(text) {
  const source = page.getByLabel("Markdown 원문 편집", { exact: true });
  await source.focus();
  await source.evaluate((node, text) => {
    const start = node.value.indexOf(text);
    if (start < 0) throw new Error("source missing");
    node.setSelectionRange(start, start + text.length);
    node.dispatchEvent(new Event("select", { bubbles: true }));
  }, text);
  await page.getByRole("button", { name: "선택 AI", exact: true }).click();
  const modal = page.getByRole("dialog", { name: "선택 영역 AI", exact: true });
  await expect(
    modal.getByText("전송 대상: selection-browser", { exact: true }),
  ).toBeVisible();
  return modal;
}
async function generate(modal) {
  const checkbox = modal.getByRole("checkbox");
  await expect(checkbox).not.toBeChecked();
  await expect(
    modal.getByRole("button", { name: "선택 원문 전송", exact: true }),
  ).toBeDisabled();
  await checkbox.check();
  await modal
    .getByRole("button", { name: "선택 원문 전송", exact: true })
    .click();
  await expect(
    modal.getByRole("heading", { name: "선택 원문과 제안 비교" }),
  ).toBeVisible();
  await expect(modal.locator(".review-comparison")).toContainText(answer);
}
async function selectRendered(text) {
  await page.locator(".editor-area").evaluate((root, text) => {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let node;
    while ((node = walker.nextNode())) {
      const start = node.textContent.indexOf(text);
      if (start < 0) continue;
      const range = document.createRange();
      range.setStart(node, start);
      range.setEnd(node, start + text.length);
      const selection = window.getSelection();
      selection.removeAllRanges();
      selection.addRange(range);
      return;
    }
    throw new Error("rendered source selection not found");
  }, text);
  await page.getByRole("button", { name: "선택 AI", exact: true }).click();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password:
      process.env.MADI_ADMIN_PASSWORD || "Integration-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", { name: "선택 AI 실제 검증" }),
    md = `---\ntags: [원본태그]\naliases: [원본별칭]\n---\n\n# 원문\n\nNEVER_SEND_OUTSIDE_SELECTION\n\n${phrase}\n\n끝 문장은 유지합니다.\n`;
  const d = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "제목과 태그 보존 검증",
    markdown: md,
    visibility: "private",
  });
  await page.goto(base + "/app");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(`${base}/app/documents/${d.id}?mode=source`);
  let modal = await select(phrase);
  await modal
    .getByLabel("선택 영역 작업", { exact: true })
    .selectOption("translate");
  await modal
    .getByLabel("추가 지시", { exact: true })
    .fill("한국어로 명료하게");
  await generate(modal);
  await capture("selection-ai-review");
  assert.equal(
    (await api("/documents/" + d.id)).version,
    1,
    "proposal must not write",
  );
  await modal
    .getByRole("button", { name: "비교한 변경 적용", exact: true })
    .click();
  await expect(modal).not.toBeVisible();
  await expect(
    page.getByLabel("Markdown 원문 편집", { exact: true }),
  ).toHaveValue(md.replace(phrase, answer));
  const changed = await api("/documents/" + d.id);
  assert.equal(changed.title, d.title);
  assert.deepEqual(changed.tags, d.tags);
  assert.deepEqual(changed.aliases, d.aliases);
  assert.equal(changed.visibility, "private");
  assert.equal(changed.version, 2);
  modal = await select(answer);
  await generate(modal);
  await modal
    .getByLabel("제안 적용 위치", { exact: true })
    .selectOption("append");
  await modal
    .getByRole("button", { name: "비교한 변경 적용", exact: true })
    .click();
  await expect(modal).not.toBeVisible();
  const appended = await api("/documents/" + d.id);
  assert.equal(
    appended.markdown,
    changed.markdown.replace(answer, answer + "\n\n" + answer),
  );
  assert.equal(appended.version, 3);
  modal = await select(answer);
  await generate(modal);
  await modal
    .getByLabel("제안 적용 위치", { exact: true })
    .selectOption("private");
  await modal
    .getByRole("button", { name: "새 개인 문서로 저장", exact: true })
    .click();
  await page.waitForURL(
    (url) =>
      url.pathname.includes("/documents/") && !url.pathname.endsWith(d.id),
  );
  const copy = await api(
    "/documents/" + new URL(page.url()).pathname.split("/").at(-1),
  );
  assert.equal(copy.visibility, "private");
  assert.equal(copy.markdown, answer);
  assert.equal((await api("/documents/" + d.id)).version, 3);
  await page.goto(`${base}/app/documents/${d.id}?mode=source`);
  modal = await select(answer);
  await generate(modal);
  await api("/documents/" + d.id, "PUT", {
    version: 3,
    markdown: appended.markdown + "\n외부 변경",
  });
  await modal
    .getByRole("button", { name: "비교한 변경 적용", exact: true })
    .click();
  await expect(modal.getByRole("alert")).toContainText(
    "다른 변경과 충돌했습니다",
  );
  assert.equal((await api("/documents/" + d.id)).version, 4);
  await capture("selection-ai-conflict");
  await modal.getByRole("button", { name: "닫기", exact: true }).click();
  // Actual read and rich-editor DOM selections resolve only unique raw spans.
  const rendered = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "렌더링 선택 검증",
    markdown: "# 선택 검증\n\n유일한 원문 문장 😀\n",
    visibility: "private",
  });
  for (const mode of ["preview", "edit"]) {
    await page.goto(`${base}/app/documents/${rendered.id}?mode=${mode}`);
    const content =
      mode === "edit"
        ? page.locator(".tiptap")
        : page.locator(".editor-area .markdown-content");
    await expect(content).toContainText("유일한 원문 문장 😀");
    if (mode === "edit")
      await expect(page.locator("[data-save-state=confirmed]")).toBeVisible();
    await selectRendered("유일한 원문 문장 😀");
    const renderedModal = page.getByRole("dialog", {
      name: "선택 영역 AI",
      exact: true,
    });
    await expect(renderedModal).toContainText("selection-browser");
    await generate(renderedModal);
    await renderedModal
      .getByRole("button", { name: "닫기", exact: true })
      .click();
    assert.equal(
      (await api("/documents/" + rendered.id)).version,
      1,
      "read/seed/proposal alone must not modify source",
    );
  }
  const ambiguous = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "중복 문장 선택 검증",
    markdown: "반복 원문\n\n반복 원문\n",
    visibility: "private",
  });
  await page.goto(`${base}/app/documents/${ambiguous.id}?mode=preview`);
  await expect(page.locator(".editor-area")).toContainText("반복 원문");
  await selectRendered("반복 원문");
  await expect(
    page.getByRole("dialog", { name: "선택 영역 AI", exact: true }),
  ).toHaveCount(0);
  await expect(page.getByText(/서식 때문에 위치가 모호하면/)).toBeVisible();
  assert.equal((await api("/documents/" + ambiguous.id)).version, 1);
  await page.goto(`${base}/app/documents/${d.id}?mode=source`);
  await page.setViewportSize({ width: 390, height: 844 });
  modal = await select("끝 문장은 유지합니다.");
  await generate(modal);
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth + 1,
      ),
    )
    .toBe(true);
  await capture("selection-ai-mobile");
  await modal
    .getByRole("heading", { name: "선택 원문과 제안 비교", exact: true })
    .scrollIntoViewIfNeeded();
  await capture("selection-ai-mobile-comparison");
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      checks: [
        "selection-only",
        "explicit-consent",
        "stream-complete",
        "partial-replace",
        "append",
        "private-copy-ticket",
        "metadata-preserved",
        "CAS409",
        "read-rich-selection",
        "ambiguous-selection-no-guess",
        "mobile390",
        "no-console-errors",
      ],
      screenshots: out,
    }),
  );
} catch (e) {
  await mkdir(out, { recursive: true });
  await page
    .screenshot({ path: path.join(out, "failure.png") })
    .catch(() => {});
  console.error(await page.locator("body").ariaSnapshot());
  throw e;
} finally {
  await browser.close();
}
