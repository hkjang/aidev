import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";

// Creates fixtures only on the dedicated disposable local QA instance.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const out = path.resolve(process.env.MADI_SCREENSHOT_DIR || "docs/screenshots");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true }),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    timezoneId: "Asia/Seoul",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const issues = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("console", (m) => {
  if (m.type() === "error" && !m.text().includes("401 (Unauthorized)"))
    issues.push(m.text());
});
await context.route("**/*", (route) => {
  const url = new URL(route.request().url());
  if (
    url.origin !== new URL(base).origin &&
    !["data:", "blob:"].includes(url.protocol)
  ) {
    issues.push(`External asset: ${url.href}`);
    return route.abort();
  }
  return route.continue();
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
  await page.evaluate(() =>
    Promise.all(
      [...document.images].map((image) => image.decode().catch(() => {})),
    ),
  );
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  if (await page.locator(".toast .icon-button").count())
    await page
      .locator(".toast .icon-button")
      .click()
      .catch(() => {});
  await page.screenshot({
    path: path.join(out, `${name}.png`),
    // Full-page viewport emulation mispaints scaled SVG foreignObject images.
    // The actual canvas viewport renders correctly.
    fullPage: name !== "canvas",
    animations: "disabled",
  });
}
async function save() {
  await page.getByRole("button", { name: "저장", exact: true }).click();
  await page
    .locator(".canvas-save-state")
    .filter({ hasText: "저장됨" })
    .waitFor();
}
async function add(kind, title, text) {
  await page.getByRole("button", { name: `${kind} 추가`, exact: true }).click();
  const dialog = page.getByRole("dialog", {
    name: `${kind} 추가`,
    exact: true,
  });
  if (await dialog.getByLabel("노드 제목", { exact: true }).count())
    await dialog.getByLabel("노드 제목", { exact: true }).fill(title);
  if (text !== undefined)
    await dialog
      .getByLabel(kind === "Mermaid" ? "Mermaid 소스" : "내용", { exact: true })
      .fill(text);
  await dialog
    .getByRole("button", { name: "캔버스에 추가", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
}
try {
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
      name: "캔버스 격리 검증 " + Date.now(),
    }),
    wid = ws.id;
  await context.addInitScript(
    (id) => localStorage.setItem("madi.workspace", id),
    wid,
  );
  const doc = await api("/documents", "POST", {
    workspace_id: wid,
    title: "캔버스 원본 · 지식 운영 원칙",
    markdown:
      "# 지식 운영 원칙\n\n결론뿐 아니라 결정의 이유를 함께 남깁니다.\n\n- 문서 소유자 지정\n- 신뢰할 수 있는 원본 연결\n- 정기 검토",
  });
  await page.goto(`${base}/app/canvases`);
  await page.getByRole("heading", { name: "캔버스", exact: true }).waitFor();
  await page.getByRole("button", { name: "새 캔버스", exact: true }).click();
  let dialog = page.getByRole("dialog", { name: "새 캔버스", exact: true });
  await dialog
    .getByLabel("캔버스 이름", { exact: true })
    .fill("지식이 연결되는 캔버스");
  await dialog
    .getByLabel("공개 범위", { exact: true })
    .selectOption("workspace");
  await dialog.getByRole("button", { name: "만들기", exact: true }).click();
  await page
    .getByRole("textbox", { name: "캔버스 제목", exact: true })
    .waitFor();
  const canvasID = new URL(page.url()).pathname.split("/").at(-1);
  await add(
    "메모",
    "팀 지식의 출발점",
    "# 기록에서 연결로\n\n좋은 아이디어를 놓치지 않고 **팀의 지식**으로 연결합니다.\n\n- 질문과 맥락 기록\n- 관련 문서 연결\n- 다음 행동 정리",
  );
  let header = page
      .locator('.canvas-node[data-node-kind="note"] .canvas-node-label')
      .first(),
    box = await header.boundingBox();
  assert.ok(box);
  await page.mouse.move(box.x + 35, box.y + 12);
  await page.mouse.down();
  await page.mouse.move(box.x - 240, box.y - 120, { steps: 12 });
  await page.mouse.up();
  await save();
  let canvas = await api("/canvases/" + canvasID);
  assert.equal(canvas.data.nodes.length, 1);
  const savedPosition = {
    x: canvas.data.nodes[0].x,
    y: canvas.data.nodes[0].y,
  };
  await page.reload();
  await page.locator('.canvas-node[data-node-kind="note"]').waitFor();
  canvas = await api("/canvases/" + canvasID);
  assert.deepEqual(
    { x: canvas.data.nodes[0].x, y: canvas.data.nodes[0].y },
    savedPosition,
  );
  console.log("PASS create canvas, Markdown note, drag, save and refresh");
  await page.getByRole("button", { name: "문서 추가", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "문서 추가", exact: true });
  await dialog.getByLabel("연결할 문서", { exact: true }).selectOption(doc.id);
  await dialog
    .getByRole("button", { name: "캔버스에 추가", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await save();
  await page
    .locator('.canvas-node[data-node-kind="document"]')
    .getByText(doc.title, { exact: true })
    .waitFor();
  await page.getByRole("button", { name: "노드 연결", exact: true }).click();
  await header.click();
  await page
    .locator('.canvas-node[data-node-kind="document"] .canvas-node-label')
    .click();
  await save();
  canvas = await api("/canvases/" + canvasID);
  assert.equal(canvas.data.edges.length, 1);
  console.log("PASS ACL-resolved document reference and connection");
  await page.getByRole("button", { name: "이미지 추가", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "이미지 추가", exact: true });
  await dialog
    .getByLabel("이미지가 첨부된 문서", { exact: true })
    .selectOption(doc.id);
  await dialog
    .getByLabel("선택한 문서에 이미지 업로드", { exact: true })
    .setInputFiles(path.resolve("docs/screenshots/workspace.png"));
  await dialog
    .getByLabel("연결할 이미지", { exact: true })
    .locator("option")
    .filter({ hasText: "workspace.png" })
    .waitFor({ state: "attached" });
  await dialog
    .getByRole("button", { name: "캔버스에 추가", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await save();
  await page.locator('.canvas-node[data-node-kind="image"] img').waitFor();
  assert.equal(
    await page
      .locator('.canvas-node[data-node-kind="image"] img')
      .evaluate((img) => img.complete && img.naturalWidth > 0),
    true,
  );
  console.log("PASS attachment image upload, native select and display");
  const stage = await page.locator(".canvas-stage").boundingBox();
  assert.ok(stage);
  await page.getByRole("button", { name: "사각형", exact: true }).click();
  await page.mouse.move(stage.x + 80, stage.y + 380);
  await page.mouse.down();
  await page.mouse.move(stage.x + 280, stage.y + 465, { steps: 8 });
  await page.mouse.up();
  await page.locator('.canvas-node[data-node-kind="rectangle"]').waitFor();
  await page.getByRole("button", { name: "펜", exact: true }).click();
  await page.mouse.move(stage.x + 370, stage.y + 410);
  await page.mouse.down();
  for (const [x, y] of [
    [400, 400],
    [430, 430],
    [460, 410],
    [490, 445],
  ])
    await page.mouse.move(stage.x + x, stage.y + y, { steps: 5 });
  await page.mouse.up();
  await save();
  canvas = await api("/canvases/" + canvasID);
  assert.ok(
    canvas.data.nodes.some((n) => n.kind === "drawing" && n.points.length > 3),
  );
  assert.ok(canvas.data.nodes.some((n) => n.kind === "rectangle"));
  const count = canvas.data.nodes.length;
  await page.getByRole("button", { name: "실행 취소", exact: true }).click();
  assert.equal(await page.locator(".canvas-node").count(), count - 1);
  await page.getByRole("button", { name: "다시 실행", exact: true }).click();
  assert.equal(await page.locator(".canvas-node").count(), count);
  console.log("PASS shape, freehand and undo/redo");
  await page.getByRole("button", { name: "Mermaid 추가", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "Mermaid 추가", exact: true });
  await dialog.getByLabel("노드 제목", { exact: true }).fill("지식 운영 흐름");
  await dialog
    .getByLabel("Mermaid 소스", { exact: true })
    .fill("flowchart LR\n A[아이디어] --> B[문서]\n B --> C[팀의 지식]");
  await dialog.getByRole("img", { name: "Mermaid 다이어그램" }).waitFor();
  assert.equal(
    await dialog.locator('foreignObject,script,a[href^="http"]').count(),
    0,
  );
  await dialog
    .getByRole("button", { name: "캔버스에 추가", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  await save();
  canvas = await api("/canvases/" + canvasID);
  const arranged = structuredClone(canvas.data);
  for (const n of arranged.nodes) {
    if (n.kind === "note") {
      Object.assign(n, { x: 60, y: 60, width: 330, height: 280 });
    }
    if (n.kind === "document") {
      Object.assign(n, { x: 440, y: 60, width: 330, height: 240 });
    }
    if (n.kind === "image") {
      Object.assign(n, { x: 820, y: 60, width: 330, height: 240 });
    }
    if (n.kind === "diagram") {
      Object.assign(n, { x: 440, y: 390, width: 710, height: 220 });
    }
    if (n.kind === "rectangle") {
      Object.assign(n, {
        x: 60,
        y: 400,
        width: 330,
        height: 160,
        text: "검토한 지식을 연결하고 공유하기",
      });
    }
    if (n.kind === "drawing") {
      Object.assign(n, { x: 60, y: 640 });
    }
  }
  await api("/canvases/" + canvasID, "PUT", {
    version: canvas.version,
    title: canvas.title,
    data: arranged,
  });
  await page.reload();
  await page
    .locator('.canvas-node[data-node-kind="diagram"]')
    .getByRole("img", { name: "Mermaid 다이어그램" })
    .waitFor();
  await page.getByRole("button", { name: "전체 보기", exact: true }).click();
  await shot("canvas");
  await page.getByRole("button", { name: "Mermaid 추가", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "Mermaid 추가", exact: true });
  await dialog
    .getByLabel("Mermaid 소스", { exact: true })
    .fill('%%{init: {"securityLevel":"loose"}}%%\nflowchart LR\n A-->B');
  await dialog
    .getByText("다이어그램 안에서 실행 설정을 변경할 수 없습니다.", {
      exact: true,
    })
    .waitFor();
  assert.equal(
    await dialog.getByRole("img", { name: "Mermaid 다이어그램" }).count(),
    0,
  );
  await dialog
    .getByRole("button", { name: "닫기", exact: true })
    .last()
    .click();
  console.log("PASS Mermaid render, local assets and unsafe config rejection");
  await page.getByRole("button", { name: "공유", exact: true }).click();
  dialog = page.getByRole("dialog", { name: "캔버스 공유", exact: true });
  await dialog.getByLabel("공개 범위", { exact: true }).selectOption("private");
  await dialog
    .getByRole("button", { name: "닫기", exact: true })
    .last()
    .click();
  assert.equal((await api("/canvases/" + canvasID)).visibility, "workspace");
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "내보내기", exact: true }).click(),
  ]);
  assert.ok(download.suggestedFilename().endsWith(".canvas.json"));
  console.log("PASS sharing cancel preserves state and JSON export");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "전체 보기", exact: true }).click();
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 2,
    ),
    true,
    "mobile canvas overflow",
  );
  await shot("mobile-canvas");
  assert.deepEqual(issues, []);
  console.log("All canvas browser checks passed.");
} catch (e) {
  await shot("failure-canvas").catch(() => {});
  throw e;
} finally {
  await context.close();
  await browser.close();
}
