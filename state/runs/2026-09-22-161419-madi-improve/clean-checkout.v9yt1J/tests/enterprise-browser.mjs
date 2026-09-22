import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const output = path.resolve(
  process.env.MADI_SCREENSHOT_DIR || "test-results/enterprise",
);
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
  locale: "ko-KR",
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
async function shot(name) {
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
    name: "엔터티 UI 검증 " + Date.now(),
  });
  await page.evaluate(
    (id) => localStorage.setItem("madi.workspace", id),
    workspace.id,
  );
  await page.goto(base + "/app/entities");
  await page
    .getByRole("button", { name: "엔터티 만들기", exact: true })
    .click();
  await page
    .getByLabel("엔터티 이름", { exact: true })
    .fill("PostgreSQL 지식 중심");
  await page
    .getByLabel("엔터티 종류", { exact: true })
    .selectOption("database");
  await page
    .getByLabel("설명", { exact: true })
    .fill(
      "운영 데이터베이스의 정책, 테이블 정의와 변경 이력을 함께 연결합니다.",
    );
  await shot("entity-create");
  await page.getByRole("button", { name: "엔터티 생성", exact: true }).click();
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "운영 데이터베이스 정책",
    markdown:
      "# 운영 정책\n\n[[PostgreSQL 지식 중심]]을 운영하는 담당자를 확인합니다.\n\n## 백업\n매일 별도 보관소에 백업합니다.",
  });
  await api("/enterprise/entities", "POST", {
    workspace_id: workspace.id,
    title: "AI 플랫폼팀",
    entity_type: "team",
    description: "서비스 운영과 지식 품질을 담당하는 팀입니다.",
  });
  await page.reload();
  await page
    .getByRole("heading", { name: "PostgreSQL 지식 중심", exact: true })
    .waitFor();
  await shot("entities");
  await page
    .getByRole("link")
    .filter({
      has: page.getByRole("heading", {
        name: "PostgreSQL 지식 중심",
        exact: true,
      }),
    })
    .click();
  await page
    .getByRole("heading", { name: "PostgreSQL 지식 중심", exact: true })
    .waitFor();
  await page
    .getByRole("link", { name: "운영 데이터베이스 정책", exact: true })
    .last()
    .waitFor();
  await shot("entity-profile");
  const entityURL = page.url();
  await page.reload();
  await page
    .getByRole("heading", { name: "PostgreSQL 지식 중심", exact: true })
    .waitFor();
  assert.equal(page.url(), entityURL);
  await page
    .getByLabel("엔터티 종류 변경", { exact: true })
    .selectOption("technology");
  await page.getByText("엔터티 종류를 변경했습니다", { exact: true }).waitFor();
  await page.reload();
  await page.getByLabel("엔터티 종류 변경", { exact: true }).waitFor();
  assert.equal(
    await page.getByLabel("엔터티 종류 변경", { exact: true }).inputValue(),
    "technology",
  );
  await page.goto(base + "/app/enterprise");
  await page
    .getByRole("heading", { name: "지식 소스 탐색", exact: true })
    .waitFor();
  await page.getByLabel("지식 소스 검색", { exact: true }).fill("운영");
  await page.getByRole("button", { name: "검색", exact: true }).click();
  await page
    .getByRole("link", { name: "운영 데이터베이스 정책", exact: true })
    .last()
    .waitFor();
  await shot("enterprise");
  const searchURL = page.url();
  await page.reload();
  await page.getByLabel("지식 소스 검색", { exact: true }).waitFor();
  assert.equal(
    await page.getByLabel("지식 소스 검색", { exact: true }).inputValue(),
    "운영",
  );
  assert.equal(page.url(), searchURL);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(entityURL);
  await page
    .getByRole("heading", { name: "PostgreSQL 지식 중심", exact: true })
    .waitFor();
  await shot("entity-profile-mobile");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "entity mobile overflow",
  );
  await page.goto(base + "/app/enterprise");
  await page
    .getByRole("heading", { name: "지식 소스 탐색", exact: true })
    .waitFor();
  await shot("enterprise-mobile");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "catalog mobile overflow",
  );
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
