import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";

// Run only against the disposable development database. Service top_k is
// restored immediately; all provider/key mutations use a new workspace.
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const screenshots = new URL("../docs/screenshots/", import.meta.url);
const requests = [];
let failProvider = false;
const provider = createServer(async (request, response) => {
  let body = "";
  for await (const chunk of request) body += chunk;
  const payload = JSON.parse(body);
  requests.push({
    path: request.url,
    authorization: request.headers.authorization,
    payload,
  });
  response.setHeader("Content-Type", "application/json");
  if (failProvider) {
    response.statusCode = 503;
    response.end(
      JSON.stringify({ error: { message: "Controlled diagnostic outage" } }),
    );
    return;
  }
  response.end(
    JSON.stringify({
      data: payload.input.map((_, index) => ({
        index,
        embedding: [0.1, 0.3, 0.9],
      })),
    }),
  );
});
await new Promise((resolve) => provider.listen(0, "127.0.0.1", resolve));
const endpoint = `http://127.0.0.1:${provider.address().port}/v1`;
const browser = await chromium.launch();
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  timezoneId: "Asia/Seoul",
  reducedMotion: "reduce",
});
const page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
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
  await mkdir(screenshots, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate(() => scrollTo(0, 0));
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  if (await page.locator(".toast .icon-button").count())
    await page
      .locator(".toast .icon-button")
      .click({ timeout: 1000 })
      .catch(() => {});
  await page.screenshot({
    path: new URL(name + ".png", screenshots).pathname,
    fullPage: true,
    animations: "disabled",
  });
}
const input = (label) => page.getByLabel(label, { exact: true });
const source = (label) => input(`${label} 설정 출처`);
async function override(label) {
  await source(label).selectOption("workspace");
}
async function save(workspace = true) {
  const pending = page.waitForResponse(
    (r) =>
      r.request().method() === "PUT" &&
      r.url().endsWith(workspace ? "/settings" : "/admin/settings"),
  );
  await page
    .getByRole("button", { name: "검색 AI 설정 저장", exact: true })
    .click();
  const response = await pending;
  assert.ok(response.ok(), await response.text());
  await page
    .getByText("모든 설정이 저장되어 있습니다.", { exact: true })
    .waitFor();
  return response.request().postDataJSON();
}
let originalTopK,
  testWorkspace,
  globalChanged = false;
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  originalTopK = (await api("/admin/settings")).rag_top_k;
  const ws = await api("/workspaces", "POST", { name: "검색 AI 설정 검증" });
  testWorkspace = ws.id;
  await page.goto(base + "/admin/search-ai");
  await input("최종 출처 수").waitFor();
  const nextTopK = originalTopK === 8 ? 9 : 8;
  await input("최종 출처 수").fill(String(nextTopK));
  globalChanged = true;
  const sent = await save(false);
  assert.deepEqual(sent, { rag_top_k: nextTopK });
  await shot("admin-search-ai");
  await api("/admin/settings", "PUT", { rag_top_k: originalTopK });
  globalChanged = false;
  console.log(
    "PASS service settings write changed RAG keys only, preserve unrelated settings and secrets",
  );

  await page.goto(base + "/app/search-ai-settings");
  await input("워크스페이스 선택").selectOption(ws.id);
  await page.goto(base + "/app/search-ai-settings");
  await input("최종 출처 수").waitFor();
  assert.ok(await input("최종 출처 수").isDisabled());
  assert.equal(await source("최종 출처 수").inputValue(), "service");
  await override("최종 출처 수");
  await input("최종 출처 수").fill("4");
  await override("검색 후보 수");
  await input("검색 후보 수").fill("12");
  await override("임베딩 API 주소");
  await input("임베딩 API 주소").fill(endpoint);
  await override("임베딩 모델");
  await input("임베딩 모델").fill("madi-test-embedding");
  await override("임베딩 API 키");
  await input("임베딩 API 키").fill("workspace-diagnostic-secret");
  await override("내부 HTTP 연결 허용");
  await input("내부 HTTP 연결 허용").selectOption("true");
  await override("검색 AI 사용");
  await input("검색 AI 사용").selectOption("true");
  await page.getByText(/HTTP를 사용하면 문서 내용과 API 키가 평문/).waitFor();
  assert.ok(
    await page
      .getByRole("button", { name: "임베딩 연결 진단", exact: true })
      .isDisabled(),
  );
  const data = (await save()).data;
  assert.equal(data.rag_top_k, 4);
  assert.equal(data.rag_embedding_api_key, "workspace-diagnostic-secret");
  assert.equal(await input("임베딩 API 키").inputValue(), "");
  assert.ok(
    (await page.locator("body").innerText()).includes(
      "키 등록됨 · 내용은 표시하지 않음",
    ),
  );
  const cfg = await api(`/workspaces/${ws.id}/search-ai`);
  assert.equal(cfg.data.rag_embedding_api_key, "");
  assert.equal(cfg.data.rag_embedding_api_key_configured, true);
  await page
    .getByRole("button", { name: "임베딩 연결 진단", exact: true })
    .click();
  await page.getByText("임베딩 연결 확인", { exact: true }).waitFor();
  assert.equal(requests.length, 1);
  assert.deepEqual(requests[0].payload.input, ["madi 연결 진단: 지식 검색"]);
  assert.equal(requests[0].authorization, "Bearer workspace-diagnostic-secret");
  assert.equal(requests[0].path, "/v1/embeddings");
  await shot("workspace-search-ai");
  console.log(
    "PASS workspace overrides, write-only secret, saved-config-only real HTTP diagnostic and fixed nonprivate text",
  );

  await page.reload();
  await input("최종 출처 수").waitFor();
  assert.equal(await input("최종 출처 수").inputValue(), "4");
  assert.equal(await source("최종 출처 수").inputValue(), "workspace");
  await input("최종 출처 수").fill("5");
  const blankPreserves = (await save()).data;
  assert.deepEqual(blankPreserves, { rag_top_k: 5 });
  await page
    .getByRole("button", { name: "임베딩 연결 진단", exact: true })
    .click();
  await page.getByText("임베딩 연결 확인", { exact: true }).waitFor();
  assert.equal(
    requests.at(-1).authorization,
    "Bearer workspace-diagnostic-secret",
  );
  await source("최종 출처 수").selectOption("service");
  const inherit = (await save()).data;
  assert.deepEqual(inherit, { rag_top_k: null });
  assert.equal(await input("최종 출처 수").inputValue(), String(originalTopK));
  assert.ok(await input("최종 출처 수").isDisabled());
  console.log(
    "PASS refresh keeps overrides, blank key preservation and null returns to current service value",
  );

  // A concurrent workspace edit must not be silently overwritten.
  const beforeConflict = await api(`/workspaces/${ws.id}/search-ai`);
  await api(`/workspaces/${ws.id}/settings`, "PUT", {
    version: beforeConflict.version,
    data: { rag_scan_limit: 4500 },
  });
  await input("검색 후보 수").fill("14");
  const conflict = page.waitForResponse(
    (r) =>
      r.request().method() === "PUT" &&
      r.url().endsWith(`/workspaces/${ws.id}/settings`),
  );
  await page
    .getByRole("button", { name: "검색 AI 설정 저장", exact: true })
    .click();
  assert.equal((await conflict).status(), 409);
  await page
    .getByRole("alert")
    .filter({ hasText: "다른 관리자가 설정을 변경했습니다" })
    .waitFor();
  assert.equal(await input("검색 후보 수").inputValue(), "14");
  await page
    .getByRole("button", { name: "다시 불러오기", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "취소", exact: true })
    .click();
  assert.equal(await input("검색 후보 수").inputValue(), "14");
  await page
    .getByRole("button", { name: "다시 불러오기", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "변경사항 버리기", exact: true })
    .click();
  await page
    .getByText("모든 설정이 저장되어 있습니다.", { exact: true })
    .waitFor();
  assert.equal(await input("검색 검사 한도").inputValue(), "4500");
  assert.equal(await input("검색 후보 수").inputValue(), "12");
  failProvider = true;
  await page
    .getByRole("button", { name: "임베딩 연결 진단", exact: true })
    .click();
  await page.getByRole("alert").waitFor();
  assert.equal(
    await page.getByText("임베딩 연결 확인", { exact: true }).count(),
    0,
  );
  failProvider = false;
  await page
    .getByRole("button", { name: "임베딩 연결 진단", exact: true })
    .click();
  await page.getByText("임베딩 연결 확인", { exact: true }).waitFor();
  console.log(
    "PASS version conflict preserves draft, discard confirmation, provider failure never reports success",
  );

  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => scrollTo(0, 0));
  await shot("mobile-search-ai");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  assert.deepEqual(errors, []);
  // Leave a disabled, credential-free workspace after a transient test server.
  const final = await api(`/workspaces/${ws.id}/search-ai`);
  await api(`/workspaces/${ws.id}/settings`, "PUT", {
    version: final.version,
    data: {
      rag_enabled: false,
      rag_embedding_base_url: "",
      rag_embedding_api_key: null,
    },
  });
  console.log(
    "PASS mobile layout and zero page errors; all RAG settings browser checks passed",
  );
} catch (error) {
  await shot("rag-settings-failure").catch(() => {});
  throw error;
} finally {
  if (globalChanged)
    await api("/admin/settings", "PUT", { rag_top_k: originalTopK }).catch(
      () => {},
    );
  if (testWorkspace) {
    const current = await api(`/workspaces/${testWorkspace}/search-ai`).catch(
      () => null,
    );
    if (current)
      await api(`/workspaces/${testWorkspace}/settings`, "PUT", {
        version: current.version,
        data: {
          rag_enabled: false,
          rag_embedding_base_url: "",
          rag_embedding_api_key: null,
        },
      }).catch(() => {});
  }
  await browser.close();
  await new Promise((resolve) => provider.close(resolve));
}
