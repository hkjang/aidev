import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";

const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const screenshots = new URL("../docs/screenshots/", import.meta.url);
const requests = [],
  errors = [],
  external = [];
const provider = createServer(async (request, response) => {
  let raw = "";
  for await (const chunk of request) raw += chunk;
  const input = JSON.parse(raw);
  requests.push({ path: request.url, input });
  response.writeHead(200, { "Content-Type": "text/event-stream" });
  response.write(
    `data: ${JSON.stringify({ choices: [{ delta: { content: "GPU 메모리 부족 시 실행 중인 작업을 먼저 확인합니다. " } }] })}\n\n`,
  );
  setTimeout(() => {
    response.write(
      `data: ${JSON.stringify({ choices: [{ delta: { content: "최근 변경을 대조하고 담당자에게 점검을 요청하세요. [1]" } }] })}\n\n`,
    );
    response.end("data: [DONE]\n\n");
  }, 1200);
});
await new Promise((resolve) => provider.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch();
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  reducedMotion: "reduce",
});
const page = await context.newPage();
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`);
});
await context.route("**/*", (route) => {
  if (new URL(route.request().url()).origin === new URL(base).origin)
    return route.continue();
  external.push(route.request().url());
  return route.abort();
});
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${path}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  await mkdir(screenshots, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: new URL(name + ".png", screenshots).pathname,
    fullPage: !name.includes("source-verification"),
    animations: "disabled",
  });
}
let workspace;
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  workspace = await api("/workspaces", "POST", {
    name: "출처를 확인하는 AI 지식",
  });
  const saved = await api(`/workspaces/${workspace.id}/settings`);
  await api(`/workspaces/${workspace.id}/settings`, "PUT", {
    version: saved.version,
    data: {
      ai_enabled: true,
      ai_base_url: `http://127.0.0.1:${provider.address().port}/v1`,
      ai_model: "local-citation-contract",
      ai_max_tokens: 262144,
    },
  });
  const markdown =
    "# GPU 운영 기준\n\n이 문서는 장애 대응 원문 확인을 위한 검증용 운영 가이드입니다.\n\n## 메모리 점검\n\nGPU 메모리 부족 시 실행 중인 작업을 먼저 확인합니다.\n최근 변경을 대조하고 담당자에게 점검을 요청하세요.\n";
  const doc = await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "GPU 메모리 운영 가이드",
    markdown,
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspace.id,
  );
  await page.goto(`${base}/app`);
  await page
    .locator(".topbar")
    .getByRole("button", { name: "AI 도우미", exact: true })
    .click();
  await page
    .getByLabel("AI 질문", { exact: true })
    .fill("GPU 메모리 부족 시 점검할 절차를 요약해 줘");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page
    .locator(".ai-answer")
    .getByText("GPU 메모리 부족 시 실행 중인 작업을 먼저 확인합니다.", {
      exact: false,
    })
    .waitFor();
  assert.equal(
    await page.getByRole("button", { name: "생성 중지" }).count(),
    1,
    "first delta was not rendered while streaming",
  );
  await page.getByRole("button", { name: "답변 복사" }).waitFor();
  assert.equal(requests.length, 1);
  assert.equal(requests[0].input.stream, true);
  assert.equal(requests[0].input.max_tokens, 262144);
  assert.ok(
    requests[0].input.messages.some((m) => m.content.includes("원문 1~")),
  );
  await shot("ai-citations");
  const source = page.locator(".ai-sources button").first();
  const verification = page.waitForResponse((r) =>
    r.url().includes("/citation?"),
  );
  await source.click();
  const verified = await (await verification).json();
  const dialog = page.getByRole("dialog", { name: "인용 원문 확인" });
  await dialog.locator("pre").waitFor();
  assert.equal(
    await dialog.locator("pre").textContent(),
    Buffer.from(markdown)
      .subarray(verified.source.start_byte, verified.source.end_byte)
      .toString("utf8"),
  );
  await shot("ai-source-verification");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1, null, {timeout:3000});
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot("mobile-ai-source-verification");
  await page.keyboard.press("Escape");
  await page.setViewportSize({ width: 1512, height: 1080 });
  await api(`/documents/${doc.id}`, "PUT", {
    version: doc.version,
    markdown: markdown + "\n원문이 변경되었습니다.\n",
  });
  await page.locator(".ai-sources button").first().click();
  await dialog.getByText(/문서가 변경되/).waitFor();
  assert.equal(
    await dialog.locator("pre").count(),
    0,
    "stale source body was retained",
  );
  await page.keyboard.press("Escape");
  await page
    .getByLabel("AI 질문", { exact: true })
    .fill("현재 GPU 운영 절차를 다시 요약해 줘");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page.getByRole("button", { name: "답변 복사" }).waitFor();
  await page.locator(".ai-sources button").first().click();
  await dialog.getByRole("link", { name: "문서의 해당 줄로 이동" }).click();
  await page.waitForURL(/line=1/);
  await page.getByLabel("Markdown 원문 편집", { exact: true }).waitFor();
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    "AI citations PASS: real SSE deltas, 256k max_tokens, exact source/hash UI, stale version refusal, source-line jump, mobile and offline assets",
  );
} catch (error) {
  console.log(
    await page.evaluate(() => ({
      width: innerWidth,
      scroll: document.documentElement.scrollWidth,
      elements: [...document.querySelectorAll("body *")]
        .filter((e) => e.getBoundingClientRect().right > innerWidth + 1)
        .slice(0, 20)
        .map((e) => ({
          tag: e.tagName,
          cls: e.className,
          width: e.getBoundingClientRect().width,
          right: e.getBoundingClientRect().right,
        })),
    })),
  );
  await page.screenshot({
    path: "test-results/ai-citations-failure.png",
    fullPage: true,
  });
  throw error;
} finally {
  if (workspace) {
    const saved = await api(`/workspaces/${workspace.id}/settings`).catch(
      () => null,
    );
    if (saved)
      await api(`/workspaces/${workspace.id}/settings`, "PUT", {
        version: saved.version,
        data: { ai_enabled: false },
      }).catch(() => {});
  }
  await browser.close();
  await new Promise((resolve) => provider.close(resolve));
}
