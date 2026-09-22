import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";

// Dedicated disposable QA instance only: this creates database fixtures.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const out = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  timezoneId: "Asia/Seoul",
  reducedMotion: "reduce",
});
const page = await context.newPage(),
  issues = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("console", (message) => {
  if (
    message.type() === "error" &&
    !message.text().includes("401 (Unauthorized)")
  )
    issues.push(message.text());
});
async function api(endpoint, method = "GET", data) {
  const response = await context.request.fetch(`${base}/api/v1${endpoint}`, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${endpoint}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  if (await page.locator(".toast .icon-button").count())
    await page
      .locator(".toast .icon-button")
      .click()
      .catch(() => {});
  await page.screenshot({
    path: path.join(out, `${name}.png`),
    fullPage: true,
    animations: "disabled",
  });
}
try {
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const workspaces = await api("/workspaces"),
    wid = workspaces[0].id;
  const target = await api("/databases", "POST", {
    workspace_id: wid,
    name: "고급 DB 검증 · 원본 자산",
    properties: [
      { id: "name", name: "자산 이름", type: "text" },
      { id: "price", name: "가치", type: "number" },
    ],
  });
  const asset1 = await api(`/databases/${target.id}/rows`, "POST", {
      values: { name: "운영 문서", price: 40 },
    }),
    asset2 = await api(`/databases/${target.id}/rows`, "POST", {
      values: { name: "AI 가이드", price: 70 },
    });
  const properties = [
    { id: "name", name: "프로젝트", type: "text" },
    { id: "budget", name: "예산", type: "number" },
    {
      id: "assets",
      name: "연결 자산",
      type: "relation",
      target_database_id: target.id,
    },
    {
      id: "total",
      name: "합계",
      type: "rollup",
      relation_property_id: "assets",
      target_property_id: "price",
      aggregation: "sum",
    },
    {
      id: "tax",
      name: "세금 포함",
      type: "formula",
      expression: 'round(prop("합계") * 1.1, 2)',
    },
    {
      id: "status",
      name: "상태",
      type: "select",
      options: ["할 일", "진행 중", "완료"],
    },
    { id: "start", name: "시작 날짜", type: "date" },
    { id: "end", name: "종료 날짜", type: "date" },
    { id: "progress", name: "진행률", type: "progress" },
    {
      id: "summary",
      name: "AI 요약",
      type: "ai",
      prompt: "프로젝트 이름을 한 문장으로 설명하세요.",
      source_property_ids: ["name"],
    },
    {
      id: "document",
      name: "문서 만들기",
      type: "button",
      action: "create_document",
      template: "# {{프로젝트}}\n\n총계: {{합계}}",
    },
  ];
  const db = await api("/databases", "POST", {
    workspace_id: wid,
    name: "지식 운영 포트폴리오",
    properties,
  });
  for (const [name, budget, assets, status, start, end, progress] of [
    [
      "지식 플랫폼 구축",
      150,
      [asset1.id, asset2.id],
      "할 일",
      "2026-09-08",
      "2026-09-22",
      65,
    ],
    [
      "운영 체계 정비",
      80,
      [asset1.id],
      "진행 중",
      "2026-09-12",
      "2026-09-25",
      40,
    ],
    [
      "AI 가이드 발행",
      100,
      [asset2.id],
      "완료",
      "2026-09-10",
      "2026-09-18",
      100,
    ],
  ])
    await api(`/databases/${db.id}/rows`, "POST", {
      values: { name, budget, assets, status, start, end, progress },
    });
  await page.goto(`${base}/app/databases/${db.id}`);
  await page.getByRole("heading", { name: db.name, exact: true }).waitFor();
  await page.getByRole("gridcell").filter({ hasText: /^121$/ }).waitFor();
  assert.equal(await page.locator(".editable-table tbody tr").count(), 3);
  assert.equal(await page.locator(".computed-error").count(), 0);
  console.log("PASS relation labels, rollup and formula rendering");

  await page.getByRole("button", { name: "속성", exact: true }).click();
  const propertiesDialog = page.getByRole("dialog", {
    name: "데이터베이스 속성",
  });
  await propertiesDialog
    .getByRole("button", { name: "속성 추가", exact: true })
    .click();
  const newProperty = propertiesDialog.locator(".property-editor").last();
  await newProperty
    .getByRole("textbox", { name: "속성 12 이름" })
    .fill("연간 예상");
  await newProperty
    .getByRole("combobox", { name: "속성 12 유형" })
    .selectOption("formula");
  await newProperty
    .getByLabel("수식", { exact: true })
    .fill('prop("합계") * 12');
  await propertiesDialog
    .getByRole("button", { name: "속성 저장", exact: true })
    .click();
  await propertiesDialog.waitFor({ state: "hidden" });
  await page.getByRole("columnheader", { name: "연간 예상" }).waitFor();
  const annualProperty = (await api(`/databases/${db.id}`)).properties.find(
    (p) => p.name === "연간 예상",
  );
  assert.equal(
    (await api(`/databases/${db.id}/query`, "POST", {})).rows[0]
      .computed_values[annualProperty.id],
    1320,
  );
  await page
    .getByRole("gridcell", { name: /연간 예상/ })
    .filter({ hasText: "1320" })
    .waitFor();
  console.log("PASS advanced property native select and formula save");

  await page
    .getByRole("button", { name: "1행 항목 상세", exact: true })
    .click();
  const rowDialog = page.getByRole("dialog", { name: "항목 편집" });
  await rowDialog.getByLabel("상태", { exact: true }).selectOption("진행 중");
  await rowDialog.getByLabel("진행률", { exact: true }).fill("75");
  await rowDialog
    .getByRole("checkbox", { name: "AI 가이드", exact: true })
    .uncheck();
  await rowDialog
    .getByRole("checkbox", { name: "AI 가이드", exact: true })
    .check();
  await rowDialog.getByRole("button", { name: "저장", exact: true }).click();
  await rowDialog.waitFor({ state: "hidden" });
  await page.getByText("75%", { exact: true }).waitFor();
  console.log("PASS row select, progress and relation checkbox editing");

  await page.getByRole("button", { name: "필터 / 정렬", exact: true }).click();
  let queryDialog = page.getByRole("dialog", { name: "열별 필터와 정렬" });
  await queryDialog
    .getByRole("button", { name: "필터 추가", exact: true })
    .click();
  await queryDialog
    .getByLabel("필터 1 속성", { exact: true })
    .selectOption("tax");
  await queryDialog
    .getByLabel("필터 1 조건", { exact: true })
    .selectOption("gte");
  await queryDialog
    .getByLabel("필터 1 값 유형", { exact: true })
    .selectOption("number");
  await queryDialog.getByLabel("필터 1 값", { exact: true }).fill("100");
  assert.equal(
    new URL(page.url()).searchParams.has("filters"),
    false,
    "draft filter changed route before Apply",
  );
  await queryDialog
    .getByRole("button", { name: "정렬 추가", exact: true })
    .click();
  await queryDialog
    .getByLabel("정렬 1 속성", { exact: true })
    .selectOption("tax");
  await queryDialog
    .getByLabel("정렬 1 방향", { exact: true })
    .selectOption("desc");
  await queryDialog.getByRole("button", { name: "적용", exact: true }).click();
  await queryDialog.waitFor({ state: "hidden" });
  await page.waitForFunction(
    () => document.querySelectorAll(".editable-table tbody tr").length === 1,
  );
  await page.reload();
  await page.getByRole("gridcell").filter({ hasText: /^121$/ }).waitFor();
  assert.equal(await page.locator(".editable-table tbody tr").count(), 1);
  console.log("PASS typed formula filter, descending sort and refresh state");
  await page.getByRole("button", { name: /필터 \/ 정렬/ }).click();
  queryDialog = page.getByRole("dialog", { name: "열별 필터와 정렬" });
  await queryDialog
    .getByRole("button", { name: "모두 초기화", exact: true })
    .click();
  await queryDialog.getByRole("button", { name: "적용", exact: true }).click();
  await queryDialog.waitFor({ state: "hidden" });
  await page.waitForFunction(
    () => document.querySelectorAll(".editable-table tbody tr").length === 3,
  );
  await shot("database-advanced");
  for (const [view, label, selector] of [
    ["gallery", "갤러리", ".advanced-gallery"],
    ["list", "목록", ".advanced-list"],
    ["timeline", "타임라인", ".advanced-timeline"],
    ["board", "보드", ".kanban-board"],
    ["calendar", "캘린더", ".calendar-grid"],
  ]) {
    await page.getByRole("button", { name: label, exact: true }).click();
    await page.locator(selector).waitFor();
    assert.equal(new URL(page.url()).searchParams.get("view"), view);
    await page.reload();
    await page.locator(selector).waitFor();
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth + 2,
      ),
      true,
      `${view} page horizontal overflow`,
    );
    await shot(`database-${view}`);
  }
  console.log("PASS all six views and view refresh retention");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${base}/app/databases/${db.id}?view=gallery`);
  await page.locator(".advanced-gallery").waitFor();
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 2,
    ),
    true,
    "mobile gallery overflow",
  );
  await shot("mobile-database-gallery");
  console.log("PASS mobile gallery layout");
  assert.deepEqual(issues, []);
  console.log("All advanced database browser checks passed.");
} catch (e) {
  await shot("failure-database-advanced").catch(() => {});
  throw e;
} finally {
  await context.close();
  await browser.close();
}
