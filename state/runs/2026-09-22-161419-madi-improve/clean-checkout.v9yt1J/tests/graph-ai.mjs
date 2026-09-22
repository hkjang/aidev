import assert from "node:assert/strict";
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
const base = process.env.MADI_BASE_URL,
  wid = process.env.MADI_GRAPH_AI_WORKSPACE,
  aid = process.env.MADI_GRAPH_AI_SOURCE_A,
  bid = process.env.MADI_GRAPH_AI_SOURCE_B;
assert.ok(base && wid && aid && bid, "Use isolated TestBrowserGraphAI fixture");
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    timezoneId: "Asia/Seoul",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const issues = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) issues.push(`${r.status()} ${r.url()}`);
});
async function api(path, method = "GET", data, status = 200) {
  const r = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(r.status(), status, `${method} ${path}: ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  const out = new URL("../docs/screenshots/", import.meta.url);
  await mkdir(out, { recursive: true });
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    animations: "disabled",
  });
}
const start = () =>
  page.getByRole("button", { name: "동의하고 분석", exact: true });
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  await context.addInitScript(
    (w) => localStorage.setItem("madi.workspace", w),
    wid,
  );
  await page.goto(base + "/app/graph-ai");
  await page
    .getByRole("heading", { name: "AI 지식 제안", exact: true })
    .waitFor();
  await page.getByRole("checkbox", { name: /PostgreSQL 운영 기준/ }).check();
  await page.getByRole("checkbox", { name: /PostgreSQL 장애 복구/ }).check();
  assert.equal(
    await page.getByRole("checkbox", { name: /문서 주제/ }).isDisabled(),
    true,
  );
  await page
    .getByRole("button", { name: "전송 범위 확인", exact: true })
    .click();
  await start().waitFor();
  assert.equal(await start().isDisabled(), true);
  await page.getByRole("checkbox", { name: /개인·기밀 문서를 포함한/ }).check();
  await api("/admin/settings", "PUT", { ai_model: "현재 공급자 변경 검증" });
  await page
    .getByText(
      "원문·공급자·기능 설정이 변경되었습니다. 전송 범위를 다시 확인하고 동의하세요.",
      { exact: true },
    )
    .waitFor({ timeout: 3500 });
  assert.equal(await start().count(), 0);
  await api("/admin/settings", "PUT", { ai_model: "지식 제안 검증 모델" });
  await page
    .getByRole("button", { name: "전송 범위 확인", exact: true })
    .click();
  await start().waitFor();
  assert.equal(
    await page
      .getByRole("checkbox", { name: /개인·기밀 문서를 포함한/ })
      .isChecked(),
    false,
  );
  await page
    .getByRole("button", { name: "전송 원문 확인", exact: true })
    .first()
    .click();
  const citation = page.getByRole("dialog", {
    name: "인용 원문 확인",
    exact: true,
  });
  await citation.locator("pre").waitFor();
  assert.match(await citation.locator("pre").innerText(), /PostgreSQL/);
  await citation.getByRole("button", { name: "닫기", exact: true }).click();
  await page.getByRole("checkbox", { name: /개인·기밀 문서를 포함한/ }).check();
  await page.locator(".graph-ai-consent").scrollIntoViewIfNeeded();
  await shot("graph-ai-consent");
  await start().click();
  await page.waitForURL(/run=/);
  const runID = new URL(page.url()).searchParams.get("run");
  await page.locator('[data-action-kind="relation"]').waitFor();
  assert.equal(await page.locator(".graph-ai-actions article").count(), 4);
  let view = await api(`/graph-ai/runs/${runID}`);
  assert.ok(
    view.actions.every((a) => a.status === "proposed"),
    "automatic action",
  );
  assert.equal(
    (await api(`/documents?workspace_id=${wid}`)).length,
    2,
    "analysis wrote documents before confirmation",
  );
  await page.locator(".graph-ai-results-heading").scrollIntoViewIfNeeded();
  await shot("graph-ai-proposals");
  const relation = page.locator('[data-action-kind="relation"]');
  await relation
    .getByRole("button", { name: "내용 확인 후 적용", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "이 후보 한 건 적용",
    exact: true,
  });
  await dialog.waitFor();
  assert.equal(
    await dialog
      .getByRole("button", { name: "확인한 한 건 적용", exact: true })
      .isDisabled(),
    true,
  );
  await dialog
    .getByRole("checkbox", { name: /출처와 변경 내용을 확인/ })
    .check();
  await shot("graph-ai-confirm");
  await dialog
    .getByRole("button", { name: "확인한 한 건 적용", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await relation.getByText("적용됨", { exact: true }).waitFor();
  const entity = page.locator('[data-action-kind="entity"]');
  await entity
    .getByRole("button", { name: "내용 확인 후 적용", exact: true })
    .click();
  await dialog
    .getByRole("checkbox", { name: /출처와 변경 내용을 확인/ })
    .check();
  await dialog
    .getByRole("button", { name: "확인한 한 건 적용", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await entity.getByText("적용됨", { exact: true }).waitFor();
  view = await api(`/graph-ai/runs/${runID}`);
  const result = view.actions.find((a) => a.kind === "entity").result
    .document_id;
  const entityDoc = await api(`/documents/${result}`);
  assert.equal(entityDoc.visibility, "private");
  assert.equal(
    (await api(`/documents/${result}/knowledge`)).classification,
    "confidential",
  );
  await page.reload();
  await page
    .locator('[data-action-kind="entity"]')
    .getByText("적용됨", { exact: true })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("run"), runID);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.locator(".graph-ai-results-heading").scrollIntoViewIfNeeded();
  await shot("mobile-graph-ai");
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    true,
    "mobile overflow",
  );
  await api("/admin/settings", "PUT", { feature_flags: { "ai-graph": false } });
  await page
    .locator(".graph-ai-actions article")
    .first()
    .waitFor({ state: "hidden", timeout: 3000 });
  assert.equal(
    await page
      .getByRole("dialog", { name: "이 후보 한 건 적용", exact: true })
      .count(),
    0,
  );
  await api("/admin/settings", "PUT", { feature_flags: { "ai-graph": true } });
  await page.locator('[data-action-kind="relation"]').waitFor();
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.getByRole("button", { name: "새 분석", exact: true }).click();
  // Reload never restores transmission consent or implicitly selects sources.
  await page.getByRole("checkbox", { name: /PostgreSQL 운영 기준/ }).check();
  await page.getByRole("checkbox", { name: /PostgreSQL 장애 복구/ }).uncheck();
  for (const label of [
    "문서 관계",
    "유사·중복 후보",
    "엔터티 초안",
    "지식 공백 후보",
  ])
    await page.getByRole("checkbox", { name: new RegExp(label) }).uncheck();
  await page.getByRole("checkbox", { name: /문서 주제/ }).check();
  await page
    .getByRole("button", { name: "전송 범위 확인", exact: true })
    .click();
  await start().waitFor();
  await page.getByRole("checkbox", { name: /개인·기밀 문서를 포함한/ }).check();
  await start().click();
  await page.locator('[data-action-kind="topic"]').waitFor();
  await page
    .locator('[data-action-kind="topic"]')
    .getByRole("button", { name: "내용 확인 후 적용", exact: true })
    .click();
  await dialog
    .getByRole("checkbox", { name: /출처와 변경 내용을 확인/ })
    .check();
  await dialog
    .getByRole("button", { name: "확인한 한 건 적용", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await page
    .locator('[data-action-kind="topic"]')
    .getByText("적용됨", { exact: true })
    .waitFor();
  await page.locator(".graph-ai-results-heading").scrollIntoViewIfNeeded();
  await shot("graph-ai-topic");
  const doc = await api(`/documents/${aid}`);
  assert.equal(doc.version, 1);
  assert.match(doc.markdown, /\n$/, "canonical source changed");
  await api("/admin/settings", "PUT", { ai_model: "취소 검증 모델" });
  await page.getByRole("button", { name: "새 분석", exact: true }).click();
  await page.getByRole("checkbox", { name: /PostgreSQL 운영 기준/ }).check();
  await page
    .getByRole("button", { name: "전송 범위 확인", exact: true })
    .click();
  await start().waitFor();
  await page.getByRole("checkbox", { name: /개인·기밀 문서를 포함한/ }).check();
  await start().click();
  await page.getByText(/응답 [1-9][0-9,]*자 수신/).waitFor();
  const cancelledID = new URL(page.url()).searchParams.get("run");
  await page.getByRole("button", { name: "분석 취소", exact: true }).click();
  await page
    .locator(".graph-ai-results-heading")
    .getByText("취소됨", { exact: true })
    .waitFor();
  assert.equal(
    (await api(`/graph-ai/runs/${cancelledID}`)).status,
    "cancelled",
  );
  await page.getByRole("button", { name: "기록 삭제", exact: true }).click();
  await page
    .getByRole("dialog", { name: "분석 기록 삭제", exact: true })
    .getByRole("button", { name: "기록만 삭제", exact: true })
    .click();
  await page.waitForURL((u) => !u.searchParams.has("run"));
  await api(`/graph-ai/runs/${cancelledID}`, "GET", undefined, 404);
  assert.equal(
    (await api(`/documents?workspace_id=${wid}`)).length,
    3,
    "history deletion removed canonical data",
  );
  const summaries = await api(`/documents?workspace_id=${wid}`);
  const summary = summaries.find((d) => d.id === aid);
  assert.equal(Object.hasOwn(summary, "markdown"), false);
  assert.equal(Object.hasOwn(summary, "block_metadata"), false);
  assert.ok(summary.excerpt.includes("서비스의 데이터 관리"));
  const canonical = await api(`/documents/${aid}`);
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.goto(base + "/app/documents");
  const card = page.locator(".document-table-row").filter({ hasText: "PostgreSQL 운영 기준" });
  await card.waitFor();
  assert.match(await card.locator("small").innerText(), /서비스의 데이터 관리/);
  await shot("document-summary-list");
  await page.goto(base + `/app/documents/${aid}?mode=read`);
  await page.locator(".markdown-content").getByText("서비스의 데이터 관리 원칙과 검증 기준을 기록합니다.", { exact: true }).waitFor();
  await page.reload();
  await page.locator(".markdown-content").getByText("서비스의 데이터 관리 원칙과 검증 기준을 기록합니다.", { exact: true }).waitFor();
  const afterRead = await api(`/documents/${aid}`);
  assert.equal(afterRead.markdown, canonical.markdown, "summary/read navigation mutated canonical Markdown");
  assert.equal(afterRead.version, canonical.version, "read refresh changed the document version");
  assert.deepEqual(issues, []);
  console.log(
    "PASS Graph AI explicit preview/provider reconsent, exact citation, real SSE, no auto changes, per-action consent, private/classification inheritance, refresh, live feature revocation, single-source topic, mobile",
  );
} catch (e) {
  await page.screenshot({
    path: "/tmp/madi-graph-ai-failure.png",
    fullPage: true,
  });
  throw e;
} finally {
  await browser.close();
}
