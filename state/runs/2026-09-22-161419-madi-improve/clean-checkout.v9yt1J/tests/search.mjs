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
  errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`);
});
page.on("console", (m) => {
  if (m.type() === "error" && !m.text().includes("401 (Unauthorized)"))
    errors.push(m.text());
});
await context.route("**/*", (route) => {
  if (new URL(route.request().url()).origin === new URL(base).origin)
    return route.continue();
  external.push(route.request().url());
  return route.abort();
});
async function api(endpoint, method = "GET", data) {
  const r = await context.request.fetch(`${base}/api/v1${endpoint}`, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${endpoint}: ${r.status()} ${await r.text()}`);
  return r.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
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
  const email = `search-ui-${Date.now()}@example.test`,
    password = "Search-Browser-Password-2026!";
  await api("/admin/users", "POST", {
    email,
    name: "지식 탐색 검증",
    role: "editor",
    password,
  });
  await api("/auth/logout", "POST", {});
  await api("/auth/login", "POST", { email, password });
  const ws = await api("/workspaces", "POST", { name: "연결된 운영 지식" });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  const md =
    "# Kubernetes 운영 지식\n\n운영팀이 함께 관리하는 배포와 장애 대응 지식입니다.\n\n## 배포 점검\n\n- [ ] Kubernetes 배포 후 상태 확인\n\n```shell\nkubectl get pods -n knowledge\n```\n";
  const doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "Kubernetes 운영 가이드",
    markdown: md,
    tags: ["kubernetes", "운영"],
  });
  await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "GPU 서버 운영 점검",
    markdown: "# 운영 점검\n\nGPU 온도와 사용량을 점검합니다.",
    tags: ["gpu", "운영"],
  });
  await api(`/documents/${doc.id}/comments`, "POST", {
    body: "Kubernetes 운영 담당자는 배포 점검 결과를 확인해 주세요.",
  });
  const db = await api("/databases", "POST", {
    workspace_id: ws.id,
    name: "Kubernetes 운영 변경 이력",
    properties: [{ id: "title", name: "변경 내용", type: "text" }],
  });
  await api(`/databases/${db.id}/rows`, "POST", {
    values: { title: "Kubernetes 운영 배포 승인 기록" },
  });
  for (let n = 0; n < 80; n++) {
    const v = await api(`/search/index-status?workspace_id=${ws.id}`);
    if (v.indexed === v.documents && v.documents >= 2) break;
    await new Promise((resolve) => setTimeout(resolve, 250));
    if (n === 79) throw Error("index queue did not finish");
  }
  await page.goto(`${base}/app/search`);
  await page.getByRole("heading", { name: "통합 검색", exact: true }).waitFor();
  await page.getByLabel("통합 검색어").fill("Kubernetes");
  await page.getByRole("button", { name: "검색", exact: true }).click();
  await page.locator(".search-result").first().waitFor();
  assert.equal(new URL(page.url()).searchParams.get("q"), "Kubernetes");
  await shot("universal-search");
  await page
    .getByRole("group", { name: "검색 종류" })
    .getByRole("button", { name: "댓글", exact: true })
    .click();
  await page.waitForFunction(() => !document.querySelector('.search-results-header')?.textContent?.includes('찾고 있습니다') && document.querySelectorAll('.search-result').length === 1);
  await page
    .locator(".search-result .badge")
    .filter({ hasText: "댓글" })
    .waitFor();
  assert.equal(await page.locator(".search-result").count(), 1);
  await page.reload();
  await page
    .locator(".search-result .badge")
    .filter({ hasText: "댓글" })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("type"), "comment");
  await page
    .getByRole("group", { name: "검색 종류" })
    .getByRole("button", { name: "전체", exact: true })
    .click();
  await page.getByRole("button", { name: "상세 필터", exact: true }).click();
  await page.getByLabel("모든 태그", { exact: true }).selectOption("운영");
  await page.getByLabel("검색 정렬").selectOption("newest");
  await page.locator(".search-result").first().waitFor();
  await shot("search-filters");
  await page.reload();
  await page.getByRole("button", { name: /상세 필터/ }).click();
  assert.equal(
    await page.getByLabel("모든 태그", { exact: true }).inputValue(),
    "운영",
  );
  assert.equal(await page.getByLabel("검색 정렬").inputValue(), "newest");
  await page.getByRole("button", { name: "필터 초기화", exact: true }).click();
  await page.getByLabel("통합 검색어").fill("kubectl");
  await page.getByRole("button", { name: "검색", exact: true }).click();
  await page
    .getByRole("group", { name: "검색 종류" })
    .getByRole("button", { name: "코드", exact: true })
    .click();
  await page.locator(".search-result").first().waitFor();
  await shot("search-code");
  await page.locator(".search-result-title").first().click();
  await page.getByLabel("Markdown 원문 편집", {exact:true}).waitFor();
  assert.ok(new URL(page.url()).searchParams.get("line"));
  await page.getByText(/검색 결과 원문.*번째 줄/).waitFor();
  await page.goto(`${base}/app/search?q=Kubernetes`);
  await page.locator(".search-result").first().waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: /상세 필터/ }).click();
  await page.locator("#search-filters").waitFor();
  await page.waitForTimeout(350);
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth,
    ),
    false,
  );
  await shot("mobile-search-filters");
  await page.getByRole("button", { name: /상세 필터/ }).click();
  await shot("mobile-universal-search");
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    "PASS universal search kind/filter/sort selectors, deep-link reload, code source line, responsive mobile, no console/500/external requests",
  );
} catch (e) {
  console.error("search issues", errors, external);
  await page
    .screenshot({
      path: path.resolve("test-results/search-failure.png"),
      fullPage: true,
      animations: "disabled",
    })
    .catch(() => {});
  throw e;
} finally {
  await browser.close();
}
