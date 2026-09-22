import assert from "node:assert/strict";
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
const base = process.env.MADI_BASE_URL,
  wid = process.env.MADI_AGENT_WORKSPACE,
  did = process.env.MADI_AGENT_DOCUMENT,
  aid = process.env.MADI_AGENT_ID;
assert.ok(
  base && wid && did && aid,
  "Run TestBrowserWorkspaceAgent isolated fixture",
);
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
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${path} ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  const out = new URL("../docs/screenshots/", import.meta.url);
  await mkdir(out, { recursive: true });
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    fullPage: false,
    animations: "disabled",
  });
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  await context.addInitScript(
    (w) => localStorage.setItem("madi.workspace", w),
    wid,
  );
  await page.goto(base + "/app/agents");
  await page
    .getByRole("heading", { name: "워크스페이스 Agent", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "설정", exact: true }).click();
  const settings = page.getByRole("dialog", {
    name: "Agent 설정",
    exact: true,
  });
  await settings.waitFor();
  await settings.getByLabel("최대 실행 단계", { exact: true }).fill("6");
  await shot("agent-settings");
  await settings
    .getByRole("button", { name: "설정 저장", exact: true })
    .click();
  await settings.waitFor({ state: "hidden" });
  await page
    .getByLabel("Agent에게 요청하기", { exact: true })
    .fill("지식 운영 원칙을 읽고 분기별 최신성 점검 항목을 제안해 주세요.");
  await shot("agent-workspace");
  await page.getByRole("button", { name: "실행 시작", exact: true }).click();
  await page.waitForURL(/\/app\/agents\/runs\//);
  await page
    .getByRole("button", { name: "내용 확인", exact: true })
    .waitFor({ timeout: 60000 });
  assert.equal((await api(`/documents/${did}`)).version, 1);
  await page.getByRole("button", { name: "내용 확인", exact: true }).click();
  const confirmation = page.getByRole("dialog", {
    name: "이 문서 변경을 확인하시겠습니까?",
    exact: true,
  });
  await confirmation.waitFor();
  assert.equal(
    await confirmation
      .getByRole("button", { name: "확인한 변경 저장", exact: true })
      .isDisabled(),
    true,
  );
  await shot("agent-action-confirm");
  await confirmation
    .getByRole("checkbox", { name: /대상과 본문을 읽었으며/ })
    .check();
  await confirmation
    .getByRole("button", { name: "확인한 변경 저장", exact: true })
    .click();
  await confirmation.waitFor({ state: "hidden" });
  await page
    .getByText("저장 완료", { exact: false })
    .waitFor({ timeout: 60000 });
  await page
    .getByText("확인한 지식 운영 원칙을 저장했습니다.", { exact: false })
    .waitFor({ timeout: 60000 });
  const document = await api(`/documents/${did}`);
  assert.equal(document.version, 2);
  assert.ok(document.markdown.includes("분기별로 최신성을 점검"));
  await shot("agent-run");
  const runURL = page.url();
  await page.reload();
  await page
    .getByText("확인한 지식 운영 원칙을 저장했습니다.", { exact: false })
    .waitFor({ timeout: 30000 });
  assert.equal(page.url(), runURL);
  await page.setViewportSize({ width: 390, height: 844 });
  await shot("mobile-agent-run");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "mobile overflow",
  );
  assert.deepEqual(issues, []);
  console.log(
    "PASS Agent settings, real streamed tools, immutable human confirmation, REST save, durable refresh, mobile; JS/500 errors 0",
  );
} catch (e) {
  await page
    .screenshot({ path: "/tmp/madi-agent-browser-failure.png", fullPage: true })
    .catch(() => {});
  console.error(await page.locator("body").innerText());
  console.error(issues);
  throw e;
} finally {
  await browser.close();
}
