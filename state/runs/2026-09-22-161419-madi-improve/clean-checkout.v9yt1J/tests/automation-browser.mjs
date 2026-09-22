import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";

// Dedicated test workspace only. Does not change global job processing settings.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const output = path.resolve(
  process.env.MADI_SCREENSHOT_DIR || "test-results/automation",
);
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
  locale: "ko-KR",
  timezoneId: "Asia/Seoul",
});
const page = await context.newPage();
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`);
});
async function api(endpoint, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + endpoint, {
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
async function screenshot(name) {
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(output, name + ".png"),
    fullPage: true,
  });
}
try {
  await page.goto(base + "/login");
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const workspace = await api("/workspaces", "POST", {
    name: "자동화 UI 검증 " + Date.now(),
  });
  await page.evaluate(
    (id) => localStorage.setItem("madi.workspace", id),
    workspace.id,
  );
  await page.goto(base + "/app/automations");
  await page.getByRole("heading", { name: "자동화", exact: true }).waitFor();
  await page
    .getByRole("button", { name: "자동화 만들기", exact: true })
    .click();
  await page.getByLabel("자동화 이름", { exact: true }).fill("문서 생성 알림");
  await page
    .getByLabel("시작 이벤트", { exact: true })
    .selectOption("document.created");
  await page
    .getByLabel("작업 1 유형", { exact: true })
    .selectOption("notification");
  await page
    .getByLabel("알림 제목", { exact: true })
    .fill("{{title}} 새 문서 알림");
  await screenshot("automation-editor");
  await page.getByRole("button", { name: "자동화 저장", exact: true }).click();
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  await page
    .getByRole("heading", { name: "문서 생성 알림", exact: true })
    .waitFor();
  await page.reload();
  await page
    .getByRole("heading", { name: "문서 생성 알림", exact: true })
    .waitFor();
  await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "자동화 검증 원본",
    markdown: "자동화 브라우저 테스트",
  });
  await screenshot("automations");
  await page.goto(base + "/app/webhooks");
  await page.getByRole("heading", { name: "Webhook", exact: true }).waitFor();
  await page.getByRole("button", { name: "Webhook 추가", exact: true }).click();
  await page.getByLabel("이름", { exact: true }).fill("내부 연동 예시");
  await page
    .getByLabel("수신 URL", { exact: true })
    .fill("http://127.0.0.1:9/disabled-test");
  await page.getByLabel("Webhook 사용", { exact: true }).uncheck();
  await page.getByLabel("문서 수정", { exact: true }).check();
  await screenshot("webhook-editor");
  await page.getByRole("button", { name: "저장", exact: true }).click();
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  await page
    .getByRole("heading", { name: "내부 연동 예시", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "설정 변경", exact: true }).click();
  await page
    .getByLabel("서명 비밀 교체 (빈 값은 유지)", { exact: true })
    .fill("");
  await page.getByLabel("최대 시도 횟수", { exact: true }).fill("4");
  await page.getByRole("button", { name: "저장", exact: true }).click();
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  await screenshot("webhooks");
  await page.goto(base + "/app/jobs");
  await page.getByRole("heading", { name: "작업 이력", exact: true }).waitFor();
  await page.getByLabel("작업 상태", { exact: true }).selectOption("succeeded");
  await page.waitForTimeout(4000);
  await page.getByRole("button", { name: "새로고침", exact: true }).click();
  await screenshot("jobs");
  await page.getByLabel("작업 상태", { exact: true }).selectOption("");
  await page
    .getByRole("button", { name: "상세 보기", exact: true })
    .first()
    .click();
  await page.getByRole("dialog").waitFor();
  await screenshot("job-detail");
  await page.getByRole("button", { name: "닫기", exact: true }).click();
  await page.goto(base + "/admin/jobs");
  await page
    .getByRole("heading", { name: "작업 처리 설정", exact: true })
    .waitFor();
  await screenshot("admin-jobs");
  await page.reload();
  await page
    .getByRole("heading", { name: "작업 처리 설정", exact: true })
    .waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  for (const [route, heading, name] of [
    ["/app/automations", "자동화", "automations-mobile"],
    ["/app/webhooks", "Webhook", "webhooks-mobile"],
    ["/app/jobs", "작업 이력", "jobs-mobile"],
    ["/admin/jobs", "작업 처리 설정", "admin-jobs-mobile"],
  ]) {
    await page.goto(base + route);
    await page.getByRole("heading", { name: heading, exact: true }).waitFor();
    await screenshot(name);
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth + 1,
      ),
      `mobile overflow ${route}`,
    );
    if (route === "/app/jobs") {
      const region = page.getByRole("region", {
        name: "작업 이력 표",
        exact: true,
      });
      const bounds = await region.evaluate((el) => ({
        client: el.clientWidth,
        scroll: el.scrollWidth,
      }));
      assert.ok(
        bounds.scroll > bounds.client,
        "job table should scroll internally instead of squashing 16px text",
      );
      const button = region
        .getByRole("button", { name: "상세 보기", exact: true })
        .first();
      const size = await button.boundingBox();
      assert.ok(
        size.height >= 44 && size.width >= 100,
        "job actions must retain readable touch target",
      );
      await region.focus();
      await page.keyboard.press("ArrowRight");
      await page.waitForTimeout(250);
      assert.ok(
        await region.evaluate((el) => el.scrollLeft > 0),
        "native keyboard table scroll",
      );
      await region.evaluate((el) => {
        el.scrollLeft = el.scrollWidth;
      });
      await screenshot("jobs-mobile-actions");
    }
  }
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      workspace_id: workspace.id,
      screenshots: output,
    }),
  );
} finally {
  await browser.close();
}
