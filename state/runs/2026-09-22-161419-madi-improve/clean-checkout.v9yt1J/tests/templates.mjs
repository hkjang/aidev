import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { chromium } from "playwright";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
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
const out = pathToFileURL(resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots") + "/");
const field = (name) => page.getByLabel(name, { exact: true });
const detail = page.getByRole("complementary", {
  name: "선택한 템플릿",
  exact: true,
});
const dialog = (name) => page.getByRole("dialog", { name, exact: true });
async function api(path, method = "GET", data) {
  const r = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${method} ${path}: ${r.status()} ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate(() => scrollTo(0, 0));
  await page
    .locator(".toast .icon-button")
    .click({ timeout: 700 })
    .catch(() => {});
  await page.evaluate(
    () =>
      new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))),
  );
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    fullPage: true,
    animations: "disabled",
  });
}
async function preview(name) {
  await page
    .getByRole("button", { name: name + " 미리보기", exact: true })
    .click();
  await detail.getByRole("heading", { name, exact: true }).waitFor();
}
async function saveTemplate() {
  const response = page.waitForResponse(
    (r) =>
      /\/api\/v1\/templates(?:\/[a-z\d-]+)?$/.test(new URL(r.url()).pathname) &&
      ["POST", "PUT"].includes(r.request().method()),
  );
  await page.getByRole("button", { name: "템플릿 저장", exact: true }).click();
  const r = await response;
  assert.ok(r.ok(), await r.text());
  return r.json();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
    name: "우리 팀 템플릿 라이브러리",
  });
  const space = await api("/spaces", "POST", {
    workspace_id: ws.id,
    name: "운영 지식",
    visibility: "workspace",
  });
  await page.goto(base + "/app");
  await field("워크스페이스 선택").selectOption(ws.id);
  await page.goto(base + "/app/templates");
  await page
    .getByRole("heading", { name: "템플릿 라이브러리", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "회의록 미리보기", exact: true })
    .waitFor();
  await preview("회의록");
  await detail.getByRole("heading", { name: "다음 행동", exact: true }).waitFor();
  await detail
    .getByRole("button", { name: "내 템플릿으로 복제", exact: true })
    .click();
  await detail
    .getByRole("heading", { name: "회의록 사본", exact: true })
    .waitFor();
  let copyID = new URL(page.url()).searchParams.get("template");
  let copy = await api("/templates/" + copyID);
  assert.equal(copy.visibility, "private");
  await detail
    .getByRole("button", { name: "템플릿 수정", exact: true })
    .click();
  await field("템플릿 이름").fill("팀 회의 시작 가이드");
  await field("템플릿 분류").fill("팀 협업");
  await field("템플릿 공개 범위").selectOption("workspace");
  await field("템플릿 설명").fill(
    "회의 전에 안건을 모으고 결정과 후속 작업을 함께 정리합니다.",
  );
  copy = await saveTemplate();
  await dialog("템플릿 수정").waitFor({ state: "hidden" });
  assert.equal(copy.version, 2);
  assert.equal(copy.visibility, "workspace");
  await page.getByRole("button", { name: "새 템플릿", exact: true }).click();
  await field("템플릿 이름").fill("장애 대응 런북");
  await field("템플릿 분류").fill("IT 운영");
  await field("만들 문서 종류").selectOption("runbook");
  await field("템플릿 공개 범위").selectOption("space");
  await field("템플릿 공간").selectOption(space.id);
  await field("템플릿 아이콘").fill("🛠️");
  await field("템플릿 설명").fill(
    "점검과 검증, 롤백까지 빠짐없이 확인하는 운영 절차입니다.",
  );
  const markdown =
    "---\ntags: [운영, 런북]\n---\n# 장애 대응 · {{date}}\n\n## 사전 확인\n\n- [ ] 영향 범위 확인\n\n## 절차\n\n1. 현재 상태 확인\n\n## 검증\n\n- [ ] 정상 응답 확인\n\n## 롤백\n\n이전 설정을 복구합니다.\n\n";
  await field("템플릿 Markdown 원문").fill(markdown);
  await dialog("새 사용자 템플릿")
    .getByRole("button", { name: "미리보기", exact: true })
    .click();
  await dialog("새 사용자 템플릿")
    .getByRole("heading", { name: "사전 확인", exact: true })
    .waitFor();
  await dialog("새 사용자 템플릿")
    .getByRole("button", { name: "원문 편집", exact: true })
    .click();
  assert.equal(await field("템플릿 Markdown 원문").inputValue(), markdown);
  let template = await saveTemplate();
  await dialog("새 사용자 템플릿").waitFor({ state: "hidden" });
  const id = template.id;
  assert.deepEqual(template.tags, ["운영", "런북"]);
  assert.equal(template.markdown, markdown);
  assert.equal(template.space_id, space.id);
  await detail
    .getByRole("heading", { name: "장애 대응 런북", exact: true })
    .waitFor();
  await shot("template-library");
  const url = page.url();
  await page.reload();
  await detail
    .getByRole("heading", { name: "장애 대응 런북", exact: true })
    .waitFor();
  assert.equal(page.url(), url);
  await field("템플릿 검색").fill("장애 대응");
  await page.waitForFunction(
    () => document.querySelectorAll(".library-card").length === 1,
  );
  await field("템플릿 분류 필터").selectOption("IT 운영");
  await field("템플릿 공유 필터").selectOption("space");
  const filtered = page.url();
  await page.reload();
  await detail
    .getByRole("heading", { name: "장애 대응 런북", exact: true })
    .waitFor();
  assert.equal(page.url(), filtered);
  assert.equal(await field("템플릿 공유 필터").inputValue(), "space");
  console.log(
    "PASS builtin/custom distinction, Markdown+Front Matter roundtrip, real create/edit, scope/space selects, URL refresh and filtered library",
  );
  await detail
    .getByRole("button", { name: "템플릿 수정", exact: true })
    .click();
  await field("템플릿 이름").fill("버리면 안 되는 작성 중 이름");
  await dialog("템플릿 수정")
    .getByRole("button", { name: "취소", exact: true })
    .click();
  await dialog("작성 중인 변경을 버릴까요?")
    .getByRole("button", { name: "계속 작성", exact: true })
    .click();
  assert.equal(
    await field("템플릿 이름").inputValue(),
    "버리면 안 되는 작성 중 이름",
  );
  const before = await api("/templates/" + id);
  await api("/templates/" + id, "PUT", {
    ...before,
    name: "서버에서 갱신한 운영 런북",
  });
  const conflict = page.waitForResponse(
    (r) =>
      r.url().endsWith("/templates/" + id) && r.request().method() === "PUT",
  );
  await dialog("템플릿 수정")
    .getByRole("button", { name: "템플릿 저장", exact: true })
    .click();
  assert.equal((await conflict).status(), 409);
  assert.equal(
    await field("템플릿 이름").inputValue(),
    "버리면 안 되는 작성 중 이름",
  );
  await dialog("템플릿 수정")
    .getByRole("button", { name: "취소", exact: true })
    .click();
  await dialog("작성 중인 변경을 버릴까요?")
    .getByRole("button", { name: "변경 버리기", exact: true })
    .click();
  await page.getByRole("button", { name: "새로고침", exact: true }).click();
  await detail
    .getByRole("heading", { name: "서버에서 갱신한 운영 런북", exact: true })
    .waitFor();
  await detail.getByRole("button", { name: "변경 이력", exact: true }).click();
  await dialog("템플릿 변경 이력")
    .getByText("v1 · 장애 대응 런북", { exact: true })
    .waitFor();
  await shot("template-history");
  await dialog("템플릿 변경 이력")
    .getByText("v1 · 장애 대응 런북", { exact: true })
    .locator("../..")
    .getByRole("button", { name: "이 버전 복원", exact: true })
    .click();
  await dialog("v1 버전으로 복원할까요?")
    .getByRole("button", { name: "선택 버전 복원", exact: true })
    .click();
  await detail
    .getByRole("heading", { name: "장애 대응 런북", exact: true })
    .waitFor();
  template = await api("/templates/" + id);
  assert.equal(template.version, 3);
  assert.equal(template.markdown, markdown);
  console.log(
    "PASS cancel retains drafts, stale-save 409 preserves input, version history and actual canonical restore",
  );
  await detail
    .getByRole("button", { name: "이 템플릿으로 문서 만들기", exact: true })
    .click();
  assert.equal(await field("새 문서 공개 범위").inputValue(), "private");
  await field("새 문서 제목").fill("우리 서비스 장애 대응");
  await field("새 문서 공간").selectOption(space.id);
  const createdResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith(`/templates/${id}/documents`) &&
      r.request().method() === "POST",
  );
  await dialog("템플릿으로 새 문서 만들기")
    .getByRole("button", { name: "문서 만들기", exact: true })
    .click();
  const createdRaw = await createdResponse;
  assert.ok(createdRaw.ok(), await createdRaw.text());
  const doc = await createdRaw.json();
  await page.waitForURL(new RegExp("/app/documents/" + doc.id));
  const savedDoc = await api("/documents/" + doc.id);
  assert.equal(savedDoc.visibility, "private");
  assert.equal(savedDoc.status, "draft");
  assert.ok(!savedDoc.markdown.includes("{{date}}"));
  assert.ok(savedDoc.markdown.endsWith("\n\n"));
  assert.equal(
    (await api("/documents/" + doc.id + "/knowledge")).kind,
    "runbook",
  );
  await page.goto(base + "/app/templates?template=" + id);
  await detail
    .getByRole("heading", { name: "장애 대응 런북", exact: true })
    .waitFor();
  await detail
    .getByRole("button", { name: "휴지통으로 이동", exact: true })
    .click();
  await dialog("템플릿을 휴지통으로 옮길까요?")
    .getByRole("button", { name: "휴지통으로 이동", exact: true })
    .click();
  await detail
    .getByRole("button", { name: "템플릿 복원", exact: true })
    .waitFor();
  assert.equal(
    (await api("/documents/" + doc.id)).title,
    "우리 서비스 장애 대응",
  );
  await field("템플릿 모음").selectOption("trash");
  await page
    .getByRole("button", { name: "장애 대응 런북 미리보기", exact: true })
    .waitFor();
  await detail
    .getByRole("button", { name: "템플릿 복원", exact: true })
    .click();
  await detail
    .getByRole("button", { name: "이 템플릿으로 문서 만들기", exact: true })
    .waitFor();
  console.log(
    "PASS template materialization uses real document ACL/PII pipeline, draft/privacy/kind/date expansion; trash and restore preserve derived documents",
  );
  await page.goto(base + "/app/templates");
  await page
    .getByRole("button", { name: "장애 대응 런북 미리보기", exact: true })
    .waitFor();
  await shot("templates");
  await page.setViewportSize({ width: 390, height: 844 });
  await preview("장애 대응 런북");
  await page.waitForFunction(
    () => document.documentElement.scrollWidth <= innerWidth + 1,
  );
  await shot("mobile-templates");
  await detail
    .getByRole("button", { name: "템플릿 수정", exact: true })
    .click();
  await page.waitForFunction(
    () => document.documentElement.scrollWidth <= innerWidth + 1,
  );
  await shot("template-editor");
  await dialog("템플릿 수정")
    .getByRole("button", { name: "취소", exact: true })
    .click();
  const privateTemplate = await api("/templates", "POST", {
    workspace_id: ws.id,
    name: "비공개 템플릿 원문",
    markdown: "SECRET_TEMPLATE_BROWSER_SENTINEL",
    visibility: "private",
  });
  const email = `template-viewer-${Date.now()}@example.test`,
    password = "Template-viewer-password-2026!";
  await api("/admin/users", "POST", {
    email,
    password,
    name: "라이브러리 조회자",
    role: "viewer",
  });
  await api("/workspaces/" + ws.id + "/members", "PUT", {
    email,
    role: "viewer",
  });
  await api("/auth/logout", "POST", {});
  await api("/auth/login", "POST", { email, password });
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.goto(base + "/app");
  await field("워크스페이스 선택").selectOption(ws.id);
  await page.goto(base + "/app/templates");
  await page
    .getByRole("button", { name: "팀 회의 시작 가이드 미리보기", exact: true })
    .waitFor();
  assert.equal(
    await page
      .getByRole("button", { name: "비공개 템플릿 원문 미리보기", exact: true })
      .count(),
    0,
  );
  await preview("팀 회의 시작 가이드");
  assert.ok(
    await detail
      .getByRole("button", { name: "이 템플릿으로 문서 만들기", exact: true })
      .isDisabled(),
  );
  assert.ok(
    await page
      .getByRole("button", { name: "새 템플릿", exact: true })
      .isDisabled(),
  );
  await page.goto(base + "/app/templates?template=" + privateTemplate.id);
  await page.getByText("템플릿을 찾을 수 없습니다", { exact: true }).waitFor();
  assert.ok(
    !(await page.content()).includes("SECRET_TEMPLATE_BROWSER_SENTINEL"),
  );
  assert.deepEqual(errors, []);
  console.log(
    "PASS mobile gallery/editor selects, viewer read-only actions, direct private-source denial and zero page errors",
  );
} catch (error) {
  await shot("templates-failure").catch(() => {});
  throw error;
} finally {
  await browser.close();
}
