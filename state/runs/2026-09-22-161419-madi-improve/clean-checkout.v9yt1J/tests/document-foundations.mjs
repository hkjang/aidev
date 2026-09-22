import {documentTool,documentPanel} from "./document-ui.mjs";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium } from "playwright";

const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const screenshotDirectory = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
const browser = await chromium.launch();
const admin = await browser.newContext();
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  reducedMotion: "reduce",
});
const page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(path, method = "GET", data, want = 200, client = context) {
  const response = await client.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(
    response.status(),
    want,
    `${method} ${path}: ${await response.text()}`,
  );
  return response.json();
}
const field = (name) => page.getByLabel(name, { exact: true });
async function noOverflow() {
  await page.waitForTimeout(250);
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "page has horizontal overflow",
  );
}
async function shot(name) {
  await mkdir(screenshotDirectory, {
    recursive: true,
  });
  await page.evaluate(() => document.fonts.ready);
  await page
    .locator(".toast .icon-button")
    .click({ timeout: 400 })
    .catch(() => {});
  await page.screenshot({
    path: path.join(screenshotDirectory, name + ".png"),
    fullPage: (await page.getByRole("dialog").count()) === 0,
    animations: "disabled",
  });
}
async function openHistory() {
  await page.getByRole('tab',{name:'이력',exact:true}).click();
  await page.getByRole("button", { name: "변경 비교·복원 열기", exact: true }).click();
  await page
    .getByRole("dialog", { name: "문서 변경 이력", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "버전 1 원문 보기", exact: true })
    .waitFor();
}
async function restore(expectStatus) {
  await page
    .getByRole("button", { name: "이 버전으로 복원", exact: true })
    .click();
  const response = page.waitForResponse(
    (r) =>
      r.request().method() === "POST" &&
      r.url().includes("/versions/1/restore"),
  );
  await page.getByRole("button", { name: "복원 확인", exact: true }).click();
  assert.equal((await response).status(), expectStatus);
}
try {
  await page.goto(base + "/login");
  await page.locator(".login-version").waitFor();
  assert.match(
    await page.locator(".login-version").innerText(),
    /v\d+\.\d+\.\d+/,
  );
  await api(
    "/auth/login",
    "POST",
    {
      email: "admin@example.test",
      password:
        process.env.MADI_ADMIN_PASSWORD || "Browser-Test-Password-2026!",
    },
    200,
    admin,
  );
  const suffix = Date.now(),
    email = `foundation-${suffix}@example.test`,
    password = "Foundation-browser-password-2026!";
  await api(
    "/admin/users",
    "POST",
    { email, password, name: "기본문서 검증자", role: "editor" },
    200,
    admin,
  );
  await api("/auth/login", "POST", { email, password });
  const ws = await api("/workspaces", "POST", {
    name: "문서 기본 기능 · 검증 " + suffix,
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  await page.goto(base + "/app/profile");
  await field("글꼴").waitFor();
  await field("글꼴").selectOption("serif");
  await field("글자 크기").selectOption("18");
  await field("문서 본문 너비").selectOption("full");
  await field("맞춤법 검사").selectOption("off");
  await field("코드 색상").selectOption("dark");
  await field("날짜 표시").selectOption("iso");
  await field("기본 편집 모드").selectOption("source");
  await field("시간대").selectOption("UTC");
  let saved = page.waitForResponse(
    (r) => r.request().method() === "PUT" && r.url().endsWith("/profile"),
  );
  await page
    .getByRole("button", { name: "변경사항 저장", exact: true })
    .click();
  assert.equal((await saved).status(), 200);
  await page.reload();
  await field("글꼴").waitFor();
  for (const [name, value] of [
    ["글꼴", "serif"],
    ["글자 크기", "18"],
    ["문서 본문 너비", "full"],
    ["맞춤법 검사", "off"],
    ["코드 색상", "dark"],
    ["날짜 표시", "iso"],
    ["시간대", "UTC"],
  ])
    assert.equal(await field(name).inputValue(), value);
  assert.equal(
    await page.evaluate(
      () => getComputedStyle(document.documentElement).fontSize,
    ),
    "18px",
  );
  assert.equal(
    await page.evaluate(() => document.documentElement.dataset.codeTheme),
    "dark",
  );
  assert.match(
    await page.evaluate(
      () => getComputedStyle(document.documentElement).fontFamily,
    ),
    /Georgia/,
  );
  await shot("profile-expanded");
  await page.locator(".profile-trigger").click();
  await page.locator(".menu-version").waitFor();
  assert.match(
    await page.locator(".menu-version").innerText(),
    /v\d+\.\d+\.\d+/,
  );
  await page.keyboard.press("Escape");
  // Return only this disposable account to a readable default for the documentation captures.
  await field("글꼴").selectOption("sans");
  await field("글자 크기").selectOption("16");
  saved = page.waitForResponse(
    (r) => r.request().method() === "PUT" && r.url().endsWith("/profile"),
  );
  await page
    .getByRole("button", { name: "변경사항 저장", exact: true })
    .click();
  assert.equal((await saved).status(), 200);
  const original =
    '---\ntags: [운영, 문서]\n---\n\n# 운영 표준\n\n## 확인 절차\n\n첫 번째 원문입니다.\n\n- [ ] 점검 항목\n- [x] 완료 항목\n\n| 항목 | 값 |\n| --- | --- |\n| 서비스 | madi |\n\n```go\nfmt.Println("[[코드 안 위키 예시]]")\n```\n\n';
  let doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "운영 표준",
    markdown: original,
    visibility: "workspace",
    aliases: ["운영 별칭"],
    block_metadata: {
      blocks: [
        { id: "foundation-h1", type: "heading", text: "운영 표준" },
        { id: "foundation-h2", type: "heading", text: "확인 절차" },
      ],
    },
  });
  const source = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "관련 문서",
    markdown:
      "[[운영 별칭#확인 절차|확인 절차로 이동]]\n\n[[중복 문서]]\n\n`[[인라인 코드]]`\n",
    visibility: "workspace",
  });
  await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "중복 문서",
    markdown: "첫 문서",
    visibility: "workspace",
  });
  await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "중복 문서",
    markdown: "두 번째 문서",
    visibility: "workspace",
  });
  await page.goto(base + `/app/documents/${doc.id}`);
  await field("Markdown 원문 편집").waitFor();
  assert.equal(await field("Markdown 원문 편집").inputValue(), original);
  assert.equal(
    await field("Markdown 원문 편집").getAttribute("spellcheck"),
    "false",
  );
  assert.equal(
    await page
      .locator(".document-body")
      .evaluate((e) => getComputedStyle(e).maxWidth),
    "none",
  );
  let modeReloads=0;
  const modeRequest=async route=>{
    if(route.request().method()==='GET'){
      modeReloads++;
      await new Promise(resolve=>setTimeout(resolve,500));
    }
    await route.continue();
  };
  await page.route(base+'/api/v1/documents/'+doc.id,modeRequest);
  await documentTool(page,'Markdown 원문');
  await field("Markdown 원문 편집").fill(
    original.replace("첫 번째 원문", "두 번째 원문"),
  );
  await page.waitForTimeout(650);
  assert.equal(modeReloads,0,'clicking active editor mode must not reload the document');
  assert.match(await field('Markdown 원문 편집').inputValue(),/두 번째 원문/);
  await page.unroute(base+'/api/v1/documents/'+doc.id,modeRequest);
  let put = page.waitForResponse(
    (r) =>
      r.request().method() === "PUT" &&
      new URL(r.url()).pathname === "/api/v1/documents/" + doc.id,
  );
  await page.keyboard.press("Control+Enter");
  assert.equal((await put).status(), 200);
  doc = await api("/documents/" + doc.id);
  assert.equal(doc.version, 2);
  await page.goto(base + `/app/documents/${doc.id}?mode=read`);
  await page.locator(".editor-area .markdown-content table").waitFor();
  assert.equal(
    await page
      .locator(".editor-area .markdown-content .task-list-item input")
      .count(),
    2,
  );
  assert.match(
    await page.locator(".editor-area .markdown-content pre").innerText(),
    /\[\[코드 안 위키 예시\]\]/,
  );
  assert.equal(
    await page
      .locator(".editor-area .markdown-content pre")
      .evaluate((e) => getComputedStyle(e).backgroundColor),
    "rgb(24, 35, 48)",
  );
  assert.equal(
    await page.locator(".editor-area .markdown-content pre a").count(),
    0,
  );
  assert.equal((await api("/documents/" + doc.id)).version, 2);
  const before = await api("/documents/" + doc.id + "/versions");
  assert.ok(
    before.every((v) => !Object.hasOwn(v, "markdown")),
    "metadata list contains source",
  );
  await openHistory();
  await shot("document-history");
  await field("비교 이전 버전").selectOption("1");
  await field("비교 다음 버전").selectOption("2");
  await page.getByRole("button", { name: "변경 비교", exact: true }).click();
  await page.getByRole("table", { name: "문서 원문 줄 변경 비교" }).waitFor();
  assert.match(
    await page.locator(".history-diff-row.add").innerText(),
    /두 번째 원문/,
  );
  assert.match(
    await page.locator(".history-diff-row.remove").innerText(),
    /첫 번째 원문/,
  );
  await shot("document-diff");
  await field("비교 이전 버전").selectOption("2");
  await page
    .getByRole("button", { name: "버전 1 원문 보기", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "버전 1 원문 보기", exact: true })
    .click();
  await page
    .getByRole("button", { name: "이 버전으로 복원", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "Markdown 원문", exact: true })
    .click();
  assert.equal(await page.locator(".history-source").textContent(), original);
  // Updating after history opens must reject restore rather than silently overwriting.
  await api("/documents/" + doc.id, "PUT", {
    version: 2,
    markdown: "최신 작성자의 별도 내용\n",
  });
  await restore(409);
  await page
    .getByRole("dialog", { name: "문서 변경 이력", exact: true })
    .getByRole("alert")
    .waitFor();
  assert.equal(
    (await api("/documents/" + doc.id)).markdown,
    "최신 작성자의 별도 내용\n",
  );
  await page
    .getByRole("button", { name: "최신 이력 다시 불러오기", exact: true })
    .click();
  await page
    .getByRole("button", { name: "버전 3 원문 보기", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "버전 1 원문 보기", exact: true })
    .click();
  await page
    .getByRole("button", { name: "이 버전으로 복원", exact: true })
    .waitFor();
  await restore(200);
  await page
    .getByRole("dialog", { name: "문서 변경 이력", exact: true })
    .waitFor({ state: "hidden" });
  doc = await api("/documents/" + doc.id);
  assert.equal(doc.markdown, original);
  assert.equal(doc.version, 4);
  await page.goto(base + `/app/documents/${source.id}?mode=read`);
  await page
    .getByRole("link", { name: "확인 절차로 이동", exact: true })
    .waitFor();
  assert.match(
    await page
      .locator(".editor-area")
      .getByRole("link", { name: "중복 문서", exact: true })
      .getAttribute("href"),
    /^\/app\/search\?q=/,
  );
  assert.equal(
    await page.locator(".editor-area code").innerText(),
    "[[인라인 코드]]",
  );
  await page
    .getByRole("link", { name: "확인 절차로 이동", exact: true })
    .click();
  await page.waitForURL("**?mode=read#*");
  await page.locator('[id="확인 절차"]').waitFor({state:"attached"});
  const heading=page.locator('.editor-area').getByRole('heading',{name:'확인 절차',exact:true});
  await heading.waitFor();
  assert.equal(await heading.getAttribute('id'),'^foundation-h2');
  assert.equal((await api("/documents/" + doc.id)).version, 4);
  const attachmentDoc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "파일 원본 검증",
    markdown: "첨부파일을 보관합니다.\n",
    visibility: "private",
  });
  await page.goto(base + `/app/documents/${attachmentDoc.id}?mode=source`);
  await field("Markdown 원문 편집").waitFor();
  const uploaded = page.waitForResponse(
    (r) =>
      r.request().method() === "POST" &&
      r.url().includes("/attachments?document_id="),
  );
  await page
    .locator(".document-body input[type=file]")
    .setInputFiles({
      name: "한국어 첨부.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("첨부파일 원본\n"),
    });
  assert.equal((await uploaded).status(), 200);
  await page
    .locator(".attachment-index a")
    .filter({ hasText: "한국어 첨부.txt" })
    .waitFor();
  const attachments = await api(`/documents/${attachmentDoc.id}/attachments`);
  const download = await context.request.get(base + attachments[0].url);
  assert.equal(download.status(), 200);
  assert.equal(await download.text(), "첨부파일 원본\n");
  await page.waitForFunction(() =>
    document
      .querySelector("textarea.markdown-source")
      ?.value.includes("한국어 첨부.txt"),
  );
  const attachmentSaved = page.waitForResponse(
    (r) =>
      r.request().method() === "PUT" &&
      new URL(r.url()).pathname === "/api/v1/documents/" + attachmentDoc.id,
  );
  await page.keyboard.press("Control+Enter");
  assert.equal((await attachmentSaved).status(), 200);
  await page.goto(base + "/app/workspace-audit?days=7&action=DOCUMENT_UPDATE");
  await page
    .getByRole("heading", { name: "팀 감사로그", exact: true })
    .waitFor();
  await page.locator(".audit-table tbody tr").first().waitFor();
  assert.equal(await field("감사 동작").inputValue(), "DOCUMENT_UPDATE");
  await page.reload();
  await page.locator(".audit-table tbody tr").first().waitFor();
  assert.equal(await field("조회 기간").inputValue(), "7");
  await shot("workspace-audit");
  assert.match(
    await page.locator(".audit-table small").first().innerText(),
    /^\d{4}-\d{2}-\d{2}/,
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await noOverflow();
  await shot("mobile-workspace-audit");
  await page.goto(base + `/app/documents/${doc.id}?mode=read`);
  await field("문서 제목").waitFor();
  await openHistory();
  await field("비교 이전 버전").selectOption("1");
  await field("비교 다음 버전").selectOption("4");
  await page.getByRole("button", { name: "변경 비교", exact: true }).click();
  await page.locator(".history-diff").waitFor();
  await noOverflow();
  await shot("mobile-document-history");
  await page.keyboard.press("Escape");
  await page.goto(base + "/app/profile");
  await field("글꼴").waitFor();
  await noOverflow();
  await shot("mobile-profile-expanded");
  assert.deepEqual(errors, []);
  console.log(
    "PASS foundations: local font/width/code/date/spellcheck persisted, versions exact metadata/detail/diff/restore CAS, GFM/FM/read-only, wiki aliases+ambiguous links+anchors, workspace audit+mobile, JS errors 0",
  );
} catch (error) {
  await page
    .screenshot({
      path: path.join(screenshotDirectory, "document-foundations-failure.png"),
      fullPage: true,
    })
    .catch(() => {});
  console.error(error);
  process.exitCode = 1;
} finally {
  await browser.close();
}
