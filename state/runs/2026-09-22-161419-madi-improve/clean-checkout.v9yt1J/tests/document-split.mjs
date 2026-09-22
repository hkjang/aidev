import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  out = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/document-split",
  );
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1050 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (/^https?:/.test(r.url()) && !r.url().startsWith(base))
    external.push(r.url());
});
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
async function split(text) {
  const source = page.getByLabel("Markdown 원문 편집", { exact: true });
  await source.focus();
  await source.evaluate((node, text) => {
    const start = node.value.indexOf(text);
    if (start < 0) throw Error("selection missing");
    node.setSelectionRange(start, start + text.length);
    node.dispatchEvent(new Event("select", { bubbles: true }));
  }, text);
  await page
    .getByRole("button", { name: "선택 블록을 새 문서로 분리", exact: true })
    .click();
  return page.getByRole("dialog", {
    name: "선택 블록을 새 문서로 분리",
    exact: true,
  });
}
async function paste(text, html = "") {
  const editor = page.getByLabel("블록 문서 편집기", { exact: true });
  await editor.click();
  await editor.press("Control+End");
  await editor.evaluate(
    (node, { text, html }) => {
      const data = new DataTransfer();
      data.setData("text/plain", text);
      if (html) data.setData("text/html", html);
      node.dispatchEvent(
        new ClipboardEvent("paste", {
          bubbles: true,
          cancelable: true,
          clipboardData: data,
        }),
      );
    },
    { text, html },
  );
  return page.getByRole("dialog", { name: "붙여넣기 방식 선택", exact: true });
}
async function saved() {
  await expect(page.locator('[data-save-state="confirmed"]')).toBeVisible();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
      name: "문맥 편집과 안전한 문서 분리",
    }),
    selected =
      "## 새 지침\n\n분리할 완전한 문단 😀\n\n| 항목 | 확인 |\n| --- | --- |\n| 서버 | 정상 |\n",
    md =
      "---\ntags: [원본태그]\naliases: [원본별칭]\n---\n\n# 원본\n\n" +
      selected +
      "\n마지막 문장은 유지합니다.\n";
  let doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "분리 전 원본",
    markdown: md,
    visibility: "private",
  });
  await page.goto(base + "/app");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(`${base}/app/documents/${doc.id}?mode=source`);
  let modal = await split("분리할 완전한");
  await modal
    .getByRole("button", { name: "분리 영향 확인", exact: true })
    .click();
  await expect(modal.getByRole("alert")).toContainText("완전한 Markdown");
  assert.equal((await api(`/documents/${doc.id}`)).version, 1);
  await modal.getByRole("button", { name: "닫기", exact: true }).click();
  modal = await split(selected);
  await modal
    .getByLabel("분리할 문서 제목", { exact: true })
    .fill("새 운영 지침");
  await modal
    .getByRole("button", { name: "분리 영향 확인", exact: true })
    .click();
  await expect(
    modal.getByRole("heading", { name: "두 문서의 변경 확인", exact: true }),
  ).toBeVisible();
  await expect(
    modal.getByRole("button", { name: "확인한 블록 분리", exact: true }),
  ).toBeDisabled();
  await expect(modal).toContainText("나만 보기 원본");
  await capture("document-split-review");
  await modal.getByRole("checkbox").check();
  await api(`/documents/${doc.id}`, "PUT", {
    version: 1,
    title: "동료가 바꾼 제목",
  });
  await modal
    .getByRole("button", { name: "확인한 블록 분리", exact: true })
    .click();
  await expect(modal.getByRole("alert")).toContainText(/변경|버전/);
  assert.equal(
    (await api(`/documents?workspace_id=${ws.id}`)).filter(
      (d) => d.parent_id === doc.id,
    ).length,
    0,
  );
  await capture("document-split-conflict");
  await modal.getByRole("button", { name: "닫기", exact: true }).click();
  await page.reload();
  modal = await split(selected);
  await modal
    .getByLabel("분리할 문서 제목", { exact: true })
    .fill("새 운영 지침");
  await modal
    .getByRole("button", { name: "분리 영향 확인", exact: true })
    .click();
  await modal.getByRole("checkbox").check();
  const commit = page.waitForResponse(
    (r) =>
      r.url().endsWith(`/documents/${doc.id}/split`) &&
      r.request().method() === "POST",
  );
  await modal
    .getByRole("button", { name: "확인한 블록 분리", exact: true })
    .click();
  const response = await commit;
  assert.equal(response.status(), 200);
  const result = await response.json();
  await expect(modal).not.toBeVisible();
  assert.equal(result.child.markdown, selected);
  assert.equal(result.child.parent_id, doc.id);
  assert.equal(result.child.status, "draft");
  assert.equal(result.source.version, 3);
  assert.ok(
    result.source.markdown.includes(`[[${result.child.id}|새 운영 지침]]`),
  );
  assert.ok(result.source.markdown.startsWith("---\ntags: [원본태그]"));
  assert.equal(result.source.markdown.includes(selected), false);
  await expect(
    page.getByLabel("Markdown 원문 편집", { exact: true }),
  ).toHaveValue(result.source.markdown);
  const rich = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "문맥 도구 실동작",
    visibility: "private",
    markdown:
      "# 편집 실습\n\n선택 서식 확인\n\n| 항목 | 값 |\n| --- | --- |\n| 기존 | 유지 |\n\n```js\nconst value = 42;\n```\n\n붙여넣기 위치\n",
  });
  await page.goto(`${base}/app/documents/${rich.id}?mode=edit`);
  await saved();
  const editor = page.getByLabel("블록 문서 편집기", { exact: true });
  await editor.locator("td").first().click();
  await expect(
    page.getByRole("region", { name: "선택한 내용 편집 도구" }),
  ).toBeVisible();
  const rows = await editor.locator("tr").count();
  await page
    .getByRole("button", { name: "아래에 행 추가", exact: true })
    .click();
  await expect(editor.locator("tr")).toHaveCount(rows + 1);
  await page.getByRole("button", { name: "실행 취소", exact: true }).click();
  await expect(editor.locator("tr")).toHaveCount(rows);
  await editor.locator("pre").click();
  await expect(
    page.getByRole("button", { name: "코드 내용 복사", exact: true }),
  ).toBeVisible();
  let pm = await paste(
    "안전한 외부 서식",
    '<p data-id="duplicate-id"><strong>안전한 외부 서식</strong><img src="https://paste-tracker.invalid/pixel"><script>window.PASTE_EXECUTED=true</script></p>',
  );
  await expect(pm).toBeVisible();
  await pm.getByRole("button", { name: "기본 서식 유지", exact: true }).click();
  await expect(pm).not.toBeVisible();
  await expect(editor).toContainText("안전한 외부 서식");
  await expect
    .poll(async () => (await api(`/documents/${rich.id}`)).markdown)
    .toContain("**안전한 외부 서식**");
  assert.equal(await page.evaluate(() => window.PASTE_EXECUTED), undefined);
  assert.equal(await editor.locator('img[src*="paste-tracker"]').count(), 0);
  await saved();
  pm = await paste("https://knowledge.example.test/reference");
  await expect(pm).toBeVisible();
  await pm
    .getByRole("button", { name: "북마크로 붙여넣기", exact: true })
    .click();
  await expect
    .poll(async () => (await api(`/documents/${rich.id}`)).markdown)
    .toContain(":::bookmark");
  await saved();
  await capture("document-context-tools");
  const current = await api(`/documents/${rich.id}`),
    ids = (current.block_metadata?.blocks || []).map((b) => b.id);
  assert.equal(new Set(ids).size, ids.length);
  assert.ok(!ids.includes("duplicate-id"));
  await page.setViewportSize({ width: 390, height: 844 });
  pm = await paste("모바일 일반 텍스트", "<p><b>모바일 일반 텍스트</b></p>");
  await expect(pm).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    )
    .toBe(true);
  await capture("document-paste-mobile");
  await pm
    .getByRole("button", { name: "텍스트만 붙여넣기", exact: true })
    .click();
  await expect
    .poll(async () => (await api(`/documents/${rich.id}`)).markdown)
    .toContain("모바일 일반 텍스트");
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    JSON.stringify({
      ok: true,
      sourceCAS: true,
      atomicSplit: true,
      privateInherited: true,
      pasteNoRemote: true,
      tableUndo: true,
      mobile: true,
      jsErrors: errors.length,
    }),
  );
} catch (e) {
  await capture("document-split-failure");
  console.error(e);
  process.exitCode = 1;
} finally {
  await browser.close();
}
