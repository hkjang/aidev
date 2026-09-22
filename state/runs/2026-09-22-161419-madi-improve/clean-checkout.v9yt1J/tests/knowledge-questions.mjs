import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL,
  wid = process.env.MADI_QUESTION_WORKSPACE,
  sid = process.env.MADI_QUESTION_SOURCE,
  mid = process.env.MADI_QUESTION_MESSAGE,
  cid = process.env.MADI_QUESTION_CONVERSATION;
const out = path.resolve("test-results/knowledge-questions"),
  browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (/^https?:/.test(r.url()) && !r.url().startsWith(base + "/"))
    external.push(r.url());
});
async function api(url, method = "GET", data, status = 200) {
  const r = await context.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(r.status(), status, `${method} ${url}: ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(out, name + ".png"),
    fullPage: page.viewportSize().width > 500,
    animations: "disabled",
  });
}
async function overflow() {
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth + 1,
      ),
    )
    .toBe(true);
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: process.env.MADI_ADMIN_PASSWORD,
  });
  await page.goto(base + "/app");
  await page.getByLabel("워크스페이스 선택", { exact: true }).selectOption(wid);
  await page.goto(base + `/app/ai-history?id=${cid}`);
  await page
    .getByRole("link", {
      name: "이 질문을 공식 답변 초안으로 정리",
      exact: true,
    })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "개인 질문을 답변 초안으로 정리",
      exact: true,
    }),
  ).toBeVisible();
  await page
    .getByLabel("정리 이유", { exact: true })
    .fill("개인 질문을 팀에서 유지할 답변 초안으로 명시적으로 정리합니다.");
  await page.getByLabel("다음 검토일", { exact: true }).fill("2099-01-01");
  await page
    .getByRole("button", { name: "질문 등록 내용 비교", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.getByRole("button", {
      name: "동의하고 관리 질문 저장",
      exact: true,
    }),
  ).toBeDisabled();
  await shot("question-personal-consent");
  await dialog
    .getByRole("checkbox", {
      name: "선택한 질문·답변·근거와 현재 공유 범위를 확인했습니다.",
      exact: true,
    })
    .check();
  await dialog
    .getByRole("button", { name: "동의하고 관리 질문 저장", exact: true })
    .click();
  await expect(page).toHaveURL(/knowledge-questions\?id=/);
  const id = new URL(page.url()).searchParams.get("id"),
    p = "/knowledge/questions/" + id;
  let record = await api(p),
    doc = await api("/documents/" + record.document_id);
  assert.equal(doc.visibility, "private");
  assert.equal(record.origin, "personal_ai_question");
  assert.equal(record.official, false);
  await expect(
    page.getByRole("button", { name: "담당자로 공식 답변 확인", exact: true }),
  ).toBeDisabled();
  await shot("question-private-draft");
  // Publish through the ordinary document API. The question must not assume
  // that publication also refreshes its registered version or confirms facts.
  doc = await api("/documents/" + doc.id, "PUT", {
    version: doc.version,
    status: "published",
  });
  await expect(
    page.getByText("변경됨 · 재확인 필요", { exact: false }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "근거·담당자·검토일 정리", exact: true })
    .click();
  await page
    .getByLabel("정리 이유", { exact: true })
    .fill("게시 버전의 답변과 근거를 새 기준으로 확인했습니다.");
  await page
    .getByRole("button", { name: "질문 등록 내용 비교", exact: true })
    .click();
  await dialog
    .getByRole("checkbox", {
      name: "선택한 질문·답변·근거와 현재 공유 범위를 확인했습니다.",
      exact: true,
    })
    .check();
  await dialog
    .getByRole("button", { name: "동의하고 관리 질문 저장", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "담당자로 공식 답변 확인", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "담당자로 공식 답변 확인", exact: true })
    .click();
  await dialog
    .getByLabel("확인 이유", { exact: true })
    .fill("작업 순서와 현재 근거를 담당자로 확인했습니다.");
  await dialog
    .getByRole("checkbox", {
      name: "현재 조건과 처리 영향을 확인했습니다.",
      exact: true,
    })
    .check();
  await dialog
    .getByRole("button", { name: "현재 답변 확인 기록", exact: true })
    .click();
  await expect(
    page.getByText("공식 답변 · 현재 조건 충족", { exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByText("공식 답변 · 현재 조건 충족", { exact: true }),
  ).toBeVisible();
  await shot("question-official-conditions");
  // Check API-backed native select and source consent for manual registration.
  await page.getByRole("button", { name: "새 관리 질문", exact: true }).click();
  const fresh = await api("/documents", "POST", {
    workspace_id: wid,
    title: "다른 검증 답변",
    markdown: "검증 초안",
    visibility: "private",
  });
  await page.reload();
  await page
    .getByLabel("정본 답변 문서", { exact: true })
    .selectOption(fresh.id);
  await page
    .getByLabel("관리할 질문", { exact: true })
    .fill("장애 발생 시 기록은 어디에 남기나요?");
  await page
    .getByLabel("정리 이유", { exact: true })
    .fill("반복 질문의 관리 기준");
  await page
    .getByRole("checkbox", { name: "GPU 운영 확인 기준", exact: true })
    .check();
  await page
    .getByRole("button", { name: "질문 등록 내용 비교", exact: true })
    .click();
  await expect(
    dialog.getByText(/다른 검증 답변 · v1 · 나만 보기/),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", { name: "질문 등록 내용 비교", exact: true }),
  ).toBeFocused();
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "질문 목록", exact: true }).click();
  await page.locator(`.evidence-list a[href$="?id=${id}"]`).click();
  await expect(
    page.getByRole("heading", { name: "현재 정본 답변", exact: true }),
  ).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() => page.evaluate(() => matchMedia("(max-width:700px)").matches))
    .toBe(true);
  await overflow();
  await shot("question-mobile");
  await api("/documents/" + sid, "PUT", {
    version: 1,
    markdown: "현재 기준을 변경했습니다.",
  });
  await expect(
    page.getByText("현재 공식 답변으로 표시하지 않음", { exact: true }),
  ).toBeVisible();
  await shot("question-stale-evidence");
  const source = await api("/documents/" + sid);
  await api("/documents/" + sid, "DELETE");
  await expect(
    page.getByRole("heading", { name: "현재 정본 답변", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByText("GPU 사용량을 확인하고 변경 기록을 남깁니다.", {
      exact: true,
    }),
  ).toHaveCount(0);
  assert.equal(errors.length, 0, errors.join("\n"));
  assert.equal(external.length, 0, external.join("\n"));
  console.log(
    JSON.stringify({
      ok: true,
      id,
      source_version: source.version,
      screenshots: 5,
      js_errors: 0,
      external_requests: 0,
      human_observations: 0,
    }),
  );
} catch (e) {
  await shot("failure");
  throw e;
} finally {
  await context.close();
  await browser.close();
}
