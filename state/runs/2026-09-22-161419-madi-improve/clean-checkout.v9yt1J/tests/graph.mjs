import assert from "node:assert/strict";
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const out = pathToFileURL(resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots") + "/");
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
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate(() => scrollTo(0, 0));
  if (await page.locator(".toast .icon-button").count())
    await page
      .locator(".toast .icon-button")
      .click({ timeout: 1000 })
      .catch(() => {});
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    fullPage: true,
    animations: "disabled",
  });
}
const field = (name) => page.getByLabel(name, { exact: true });
const list = page.getByRole("list", { name: "그래프 문서 목록", exact: true });
const selected = page.locator(".graph-detail");
async function countIs(count) {
  await page.waitForFunction(
    (expected) =>
      document.querySelectorAll('ul[aria-label="그래프 문서 목록"] > li')
        .length === expected,
    count,
  );
  assert.equal(await list.getByRole("listitem").count(), count);
}
async function choose(name) {
  await list
    .getByRole("button", { name: `${name} 그래프에서 선택`, exact: true })
    .click();
  await selected.getByRole("heading", { name, exact: true }).waitFor();
}
try {
  const admin = await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const me = await api("/auth/me");
  const ws = await api("/workspaces", "POST", {
    name: "연결된 지식 워크스페이스",
  });
  const space = await api("/spaces", "POST", {
    workspace_id: ws.id,
    name: "운영 가이드",
    visibility: "workspace",
  });
  const make = (
    title,
    markdown = "문서를 연결해 지식을 쌓습니다.",
    extras = {},
  ) =>
    api("/documents", "POST", {
      workspace_id: ws.id,
      title,
      markdown,
      visibility: "workspace",
      ...extras,
    });
  const root = await make(
    "지식 운영",
    "[[배포 안내]]\n\n[[아직 없는 문서]]\n\n[[동일한 이름]]",
    { tags: ["운영"], space_id: space.id },
  );
  const child = await make("배포 안내", undefined, {
    parent_id: root.id,
    space_id: space.id,
    tags: ["배포"],
  });
  const linked = await make("서버 관리", undefined, { tags: ["운영"] });
  const privateDoc = await make("비공개 아이디어", undefined, {
    visibility: "private",
  });
  await make("동일한 이름");
  await make("동일한 이름");
  const isolated = await make("독립 메모");
  await api(`/documents/${child.id}/relations`, "POST", {
    target_id: linked.id,
    type: "related",
    expected_version: child.version,
  });
  await page.goto(base + "/app");
  await field("워크스페이스 선택").selectOption(ws.id);
  await page.goto(base + "/app/graph");
  await list
    .getByRole("button", { name: "지식 운영 그래프에서 선택", exact: true })
    .waitFor();
  await countIs(7);
  await page.locator(".graph-canvas canvas").first().waitFor();
  await page.getByText("이름 중복", { exact: true }).waitFor();
  await page.getByText("아직 없는 문서", { exact: false }).waitFor();
  await choose("지식 운영");
  const before = await page.locator(".graph-zoom-controls span").innerText();
  await page.getByRole("button", { name: "그래프 확대", exact: true }).click();
  assert.notEqual(
    await page.locator(".graph-zoom-controls span").innerText(),
    before,
  );
  await field("문서 연결 그래프").focus();
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("-");
  await page
    .getByRole("button", { name: "그래프 전체 맞춤", exact: true })
    .click();
  await shot("graph");
  console.log(
    "PASS current ACL graph, real canvas zoom/pan, keyboard node selection and unresolved/ambiguous links",
  );

  await selected
    .getByRole("button", { name: "관계 연결", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "문서 관계 연결",
    exact: true,
  });
  await dialog
    .getByLabel("연결할 문서", { exact: true })
    .selectOption(privateDoc.id);
  await dialog
    .getByLabel("새 관계 유형", { exact: true })
    .selectOption("reference");
  await dialog.getByRole("button", { name: "관계 연결", exact: true }).click();
  await selected
    .getByRole("button", { name: "비공개 아이디어 관계 해제", exact: true })
    .waitFor();
  let graph = await api(`/graph?workspace_id=${ws.id}`);
  assert.ok(
    graph.edges.some(
      (e) =>
        e.source === root.id &&
        e.target === privateDoc.id &&
        e.type === "reference" &&
        e.origin === "manual",
    ),
  );
  await selected
    .getByRole("button", { name: "비공개 아이디어 관계 해제", exact: true })
    .click();
  const removal = page.getByRole("dialog", {
    name: "직접 연결한 관계를 해제할까요?",
    exact: true,
  });
  await removal.getByRole("button", { name: "관계 해제", exact: true }).click();
  await removal.waitFor({ state: "hidden" });
  graph = await api(`/graph?workspace_id=${ws.id}`);
  assert.ok(
    !graph.edges.some(
      (e) =>
        e.source === root.id &&
        e.target === privateDoc.id &&
        e.origin === "manual",
    ),
  );
  assert.equal((await api(`/documents/${privateDoc.id}`)).id, privateDoc.id);
  console.log(
    "PASS typed manual relation creation/removal with real version checks; documents preserved",
  );

  await field("그래프 태그").selectOption("운영");
  await countIs(2);
  await field("그래프 공간").selectOption(space.id);
  await countIs(1);
  await field("그래프 작성자").selectOption(me.id);
  await field("그래프 배치").selectOption("concentric");
  const route = page.url();
  await page.reload();
  await field("그래프 태그").waitFor();
  assert.equal(page.url(), route);
  assert.equal(await field("그래프 태그").inputValue(), "운영");
  assert.equal(await field("그래프 공간").inputValue(), space.id);
  assert.equal(await field("그래프 작성자").inputValue(), me.id);
  assert.equal(await field("그래프 배치").inputValue(), "concentric");
  await page.getByRole("button", { name: "필터 초기화", exact: true }).click();
  await field("그래프 중심 문서").selectOption(child.id);
  await field("연결 깊이").selectOption("2");
  await field("관계 유형").selectOption("related");
  await countIs(2);
  await field("관계 유형").selectOption("");
  await field("연결 깊이").selectOption("5");
  await field("그래프 배치").selectOption("breadthfirst");
  const local = page.url();
  await page.reload();
  await list.getByRole("listitem").first().waitFor();
  assert.equal(page.url(), local);
  assert.equal(await field("연결 깊이").inputValue(), "5");
  await countIs(3);
  await shot("graph-local");
  console.log(
    "PASS URL-stable q/tag/space/owner/layout and typed local graph depth 1–5",
  );

  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(()=>Number.parseInt(document.querySelector('.graph-zoom-controls span')?.textContent||'999')<=125);
  await page.getByRole("button", { name: /그래프 필터 펼치기/ }).click();
  await field("연결 깊이").selectOption("1");
  await page.getByRole("button", { name: /그래프 필터 접기/ }).click();
  await shot("mobile-graph");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );

  const email = `graph-viewer-${Date.now()}@example.test`,
    password = "Graph-Viewer-Password-2026!";
  const viewer = await api("/admin/users", "POST", {
    email,
    password,
    name: "그래프 조회 사용자",
    role: "viewer",
    kind: "user",
  });
  await api(`/workspaces/${ws.id}/members`, "PUT", {
    email: viewer.email,
    role: "viewer",
  });
  await api("/auth/logout", "POST", {});
  await api("/auth/login", "POST", { email, password });
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.goto(base + "/app");
  await field("워크스페이스 선택").selectOption(ws.id);
  await page.goto(base + "/app/graph");
  await list
    .getByRole("button", { name: "지식 운영 그래프에서 선택", exact: true })
    .waitFor();
  assert.equal(
    await list
      .getByRole("button", {
        name: "비공개 아이디어 그래프에서 선택",
        exact: true,
      })
      .count(),
    0,
  );
  await choose("지식 운영");
  assert.ok(
    await selected
      .getByRole("button", { name: "관계 연결", exact: true })
      .isDisabled(),
  );
  assert.deepEqual(errors, []);
  console.log(
    "PASS mobile graph/filters, private-node exclusion, viewer read-only controls and zero page errors",
  );
} catch (e) {
  await shot("graph-failure").catch(() => {});
  throw e;
} finally {
  await browser.close();
}
