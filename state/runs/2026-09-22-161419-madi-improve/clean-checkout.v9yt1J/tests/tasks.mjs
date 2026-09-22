import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  out = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true }),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    timezoneId: "Asia/Seoul",
    reducedMotion: "reduce",
  }),
  page = await context.newPage(),
  issues = [],
  external = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) issues.push(`${r.status()} ${r.url()}`);
});
page.on("console", (m) => {
  if (m.type() === "error" && !m.text().includes("401 (Unauthorized)"))
    issues.push(m.text());
});
await context.route("**/*", (r) => {
  if (new URL(r.request().url()).origin === new URL(base).origin)
    return r.continue();
  external.push(r.request().url());
  return r.abort();
});
async function api(endpoint, method = "GET", data) {
  const r = await context.request.fetch(`${base}/api/v1${endpoint}`, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${method} ${endpoint}: ${r.status()} ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  if (await page.locator(".toast .icon-button").count())
    await page.locator(".toast .icon-button").click();
  await page.screenshot({
    path: path.join(out, `${name}.png`),
    fullPage: true,
    animations: "disabled",
  });
}
try {
  const me = await api("/auth/login", "POST", {
      email: process.env.MADI_TEST_EMAIL || "admin@example.test",
      password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
    }),
    ws = await api("/workspaces", "POST", { name: "팀 업무와 일정 검증" });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  const doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "운영 점검 계획",
    markdown: "- [ ] 운영 점검\n- [ ] 결과 공유 2026-09-16\n",
  });
  const db = await api("/databases", "POST", {
    workspace_id: ws.id,
    name: "배포 일정",
    properties: [
      { id: "title", name: "제목", type: "text" },
      { id: "date", name: "예정일", type: "date" },
    ],
  });
  await api(`/databases/${db.id}/rows`, "POST", {
    values: { title: "지식 플랫폼 배포", date: "2026-09-12" },
  });
  await page.goto(`${base}/app/tasks`);
  await page
    .getByRole("heading", { name: "할 일과 일정", exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "운영 점검 속성 수정", exact: true })
    .click();
  let modal = page.getByRole("dialog");
  await modal.getByLabel("할 일 담당자", { exact: true }).selectOption(me.id);
  await modal.getByLabel("할 일 마감일", { exact: true }).fill("2026-09-10");
  await modal
    .getByLabel("할 일 우선순위", { exact: true })
    .selectOption("high");
  await modal.getByLabel("할 일 상태", { exact: true }).selectOption("doing");
  await shot("task-properties");
  await modal.getByRole("button", { name: "할 일 저장", exact: true }).click();
  await modal.waitFor({ state: "hidden" });
  await page
    .locator(".managed-task")
    .filter({ has: page.getByText("운영 점검", { exact: true }) })
    .getByText("높음", { exact: true })
    .waitFor();
  await shot("tasks-list");
  await page.getByRole("button", { name: "칸반", exact: true }).click();
  const card = page
    .locator(".status-doing .managed-task")
    .filter({ hasText: "운영 점검" });
  await card.waitFor();
  await shot("tasks-kanban");
  await card.locator(".task-grip").dragTo(page.locator(".status-done"));
  await page
    .locator(".status-done .managed-task")
    .filter({ hasText: "운영 점검" })
    .waitFor();
  await page.reload();
  await page
    .locator(".status-done .managed-task")
    .filter({ hasText: "운영 점검" })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("view"), "board");
  let board = await api(`/tasks/board?workspace_id=${ws.id}`);
  const task = board.items.find((t) => t.text === "운영 점검");
  assert.equal(task.status, "done");
  assert.equal(task.assignee_id, me.id);
  assert.ok(task.task_id);
  console.log(
    "PASS task assignment/due/priority, native selectors, Markdown identity, Kanban drag-drop and reload",
  );
  await page.goto(`${base}/app/tasks?view=calendar&month=2026-09`);
  await page
    .getByRole("heading", { name: "2026년 9월", exact: true })
    .waitFor();
  await page
    .locator(".task-calendar-list")
    .getByText("배포 일정 · 지식 플랫폼 배포", { exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "회의·마일스톤 추가", exact: true })
    .click();
  modal = page.getByRole("dialog");
  await modal
    .getByLabel("일정 제목", { exact: true })
    .fill("플랫폼 운영 주간 회의");
  await modal.getByLabel("일정 종류", { exact: true }).selectOption("meeting");
  await modal.getByLabel("시작일", { exact: true }).fill("2026-09-14");
  await modal.getByLabel("종료일", { exact: true }).fill("2026-09-15");
  await modal
    .getByLabel("일정 공유 범위", { exact: true })
    .selectOption("workspace");
  await modal.getByLabel("연결할 문서", { exact: true }).selectOption(doc.id);
  await modal.getByRole("button", { name: "일정 저장", exact: true }).click();
  await modal.waitFor({ state: "hidden" });
  await page
    .locator(".task-calendar-list")
    .getByText("플랫폼 운영 주간 회의", { exact: true })
    .waitFor();
  await shot("tasks-calendar");
  await page.reload();
  await page
    .locator(".task-calendar-list")
    .getByText("플랫폼 운영 주간 회의", { exact: true })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("month"), "2026-09");
  await api("/tasks/calendar/events", "POST", {
    workspace_id: ws.id,
    title: "연결하지 않은 별도 일정",
    kind: "meeting",
    start_date: "2026-09-10",
    end_date: "2026-09-10",
    visibility: "private",
  });
  const documentBoardResponse = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      url.pathname === "/api/v1/tasks/board" &&
      url.searchParams.get("document_id") === doc.id
    );
  });
  const documentCalendarResponse = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      url.pathname === "/api/v1/tasks/calendar" &&
      url.searchParams.get("document_id") === doc.id
    );
  });
  await page.goto(
    `${base}/app/tasks?view=calendar&month=2026-09&document_id=${doc.id}&task=${task.task_id}`,
  );
  const scopedBoard = await (await documentBoardResponse).json(),
    scopedCalendar = await (await documentCalendarResponse).json();
  assert.equal(
    scopedBoard.items.length,
    1,
    "task deep link remains a task selection",
  );
  assert.equal(scopedBoard.items[0].task_id, task.task_id);
  assert.equal(scopedBoard.total_documents_exact, true);
  assert.equal(
    scopedCalendar.events.length,
    3,
    "calendar contains two source tasks and one linked meeting, not a global task filter",
  );
  assert.ok(
    scopedCalendar.events.every(
      (event) => event.document_id === doc.id && event.kind !== "database",
    ),
  );
  await page
    .locator(".task-calendar-list")
    .getByText("플랫폼 운영 주간 회의", { exact: true })
    .waitFor();
  assert.equal(
    await page
      .locator(".task-calendar-list")
      .getByText("배포 일정 · 지식 플랫폼 배포", { exact: true })
      .count(),
    0,
  );
  assert.equal(
    await page
      .locator(".task-calendar-list")
      .getByText("연결하지 않은 별도 일정", { exact: true })
      .count(),
    0,
  );
  await page.reload();
  await page
    .locator(".task-calendar-list")
    .getByText("플랫폼 운영 주간 회의", { exact: true })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("document_id"), doc.id);
  await page.goto(`${base}/app/tasks?document_id=invalid`);
  await page
    .getByText(
      "문서 ID를 확인하세요. 잘못된 문서 조건으로 전체 할 일을 조회하지 않습니다.",
      { exact: true },
    )
    .waitFor();
  assert.equal(await page.locator(".managed-task").count(), 0);
  await page.goto(`${base}/app/tasks?view=list`);
  await page.getByLabel("할 일 범위", { exact: true }).waitFor();
  console.log(
    "PASS server document scope, independent calendar exclusion, task selection, reload and invalid-scope recovery",
  );
  await page.getByRole("button", { name: "목록", exact: true }).click();
  await page.getByLabel("할 일 범위", { exact: true }).selectOption("mine");
  await page.getByLabel("할 일 검색", { exact: true }).fill("운영 점검");
  await page.reload();
  await page.getByLabel("할 일 범위", { exact: true }).waitFor();
  assert.equal(
    await page.getByLabel("할 일 범위", { exact: true }).inputValue(),
    "mine",
  );
  assert.equal(
    await page.getByLabel("할 일 검색", { exact: true }).inputValue(),
    "운영 점검",
  );
  console.log(
    "PASS unified calendar task/database/meeting, source linking, month and task-filter URL persistence",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${base}/app/tasks?view=list`);
  await page
    .getByRole("button", { name: "운영 점검 속성 수정", exact: true })
    .waitFor();
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot("mobile-tasks-list");
  await page.getByRole("button", { name: "칸반", exact: true }).click();
  await page.locator(".task-board").waitFor();
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot("mobile-tasks-kanban");
  await page.goto(`${base}/app/tasks?view=calendar&month=2026-09`);
  await page
    .locator(".task-calendar-list")
    .getByText("플랫폼 운영 주간 회의", { exact: true })
    .waitFor();
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot("mobile-tasks-calendar");
  assert.deepEqual(issues, []);
  assert.deepEqual(external, []);
  console.log(
    "PASS desktop/mobile tasks, board and calendar with zero overflow, external requests and browser errors",
  );
} catch (error) {
  await shot("tasks-failure");
  console.error("BROWSER ISSUES", issues);
  console.error(await page.locator("body").innerText());
  throw error;
} finally {
  await browser.close();
}
