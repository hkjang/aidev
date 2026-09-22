import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";

const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const requests = [],
  errors = [],
  external = [];
const provider = createServer(async (request, response) => {
  let raw = "";
  for await (const chunk of request) raw += chunk;
  const input = JSON.parse(raw);
  requests.push(input);
  response.writeHead(200, { "Content-Type": "text/event-stream" });
  response.write(
    `data: ${JSON.stringify({ choices: [{ delta: { content: "# 검토할 운영 초안\n\n" } }] })}\n\n`,
  );
  setTimeout(
    () =>
      response.end(
        `data: ${JSON.stringify({ choices: [{ delta: { content: "사용자가 내용을 확인한 뒤 개인 초안으로 저장합니다. [1]" } }] })}\n\ndata: [DONE]\n\n`,
      ),
    250,
  );
});
await new Promise((resolve) => provider.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch();
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  reducedMotion: "reduce",
});
const page = await context.newPage();
page.on("pageerror", (error) => errors.push(error.message));
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
let workspace;
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  workspace = await api("/workspaces", "POST", {
    name: "생각을 다듬는 AI 작업",
  });
  const settings = await api(`/workspaces/${workspace.id}/settings`);
  await api(`/workspaces/${workspace.id}/settings`, "PUT", {
    version: settings.version,
    data: {
      ai_enabled: true,
      ai_base_url: `http://127.0.0.1:${provider.address().port}/v1`,
      ai_model: "document-actions",
      ai_max_tokens: 262144,
    },
  });
  const doc = await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "운영 절차 원문",
    markdown: "# 운영 절차\n\n검토자와 다음 실행 항목을 확인합니다.\n",
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspace.id,
  );
  await page.goto(base + `/app/documents/${doc.id}?mode=preview`);
  await page
    .locator(".topbar")
    .getByRole("button", { name: "AI 도우미", exact: true })
    .click();
  const select = page.getByLabel("AI 작업", { exact: true });
  await page.waitForFunction(
    () => document.querySelector("#madi-ai-action")?.options.length === 11,
  );
  const catalogue = await api("/ai/actions");
  for (const action of catalogue.actions) {
    await select.selectOption(action.id);
    assert.equal(await select.inputValue(), action.id);
    assert.equal(
      await page.getByLabel("AI 질문", { exact: true }).inputValue(),
      action.prompt,
    );
  }
  await select.selectOption("meeting");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page
    .getByRole("button", { name: "새 개인 문서로 저장", exact: true })
    .waitFor();
  assert.equal(requests.length, 1);
  assert.equal(requests[0].stream, true);
  assert.ok(requests[0].messages[0].content.includes("현재 작업: 회의록 정리"));
  assert.equal(
    (await api(`/documents?workspace_id=${workspace.id}`)).length,
    1,
    "AI automatically created a document",
  );
  assert.equal((await api(`/documents/${doc.id}`)).markdown, doc.markdown);
  await page
    .getByRole("button", { name: "새 개인 문서로 저장", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "AI 초안 검토" });
  await dialog.waitFor();
  await dialog.getByLabel("새 문서 제목").fill("검토한 회의 실행 계획");
  await mkdir(new URL("../docs/screenshots/", import.meta.url), {
    recursive: true,
  });
  await page.screenshot({
    path: new URL("../docs/screenshots/ai-draft-review.png", import.meta.url)
      .pathname,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(
    () => document.documentElement.scrollWidth <= innerWidth + 1,
  );
  await page.screenshot({
    path: new URL(
      "../docs/screenshots/mobile-ai-draft-review.png",
      import.meta.url,
    ).pathname,
  });
  await dialog
    .getByRole("button", { name: "검토한 내용을 개인 초안으로 저장" })
    .click();
  await page.waitForURL(
    (url) =>
      /\/app\/documents\//.test(url.pathname) && !url.pathname.endsWith(doc.id),
  );
  const saved = await api(
    `/documents/${page.url().split("/documents/")[1].split("?")[0]}`,
  );
  assert.equal(saved.visibility, "private");
  assert.equal(saved.status, "draft");
  assert.equal(saved.title, "검토한 회의 실행 계획");
  assert.ok(saved.markdown.includes("사용자가 내용을 확인한 뒤"));
  assert.equal((await api(`/documents/${doc.id}`)).version, doc.version);
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.goto(base + `/app/documents/${doc.id}?mode=preview`);
  await page
    .locator(".topbar")
    .getByRole("button", { name: "AI 도우미", exact: true })
    .click();
  await page.getByLabel("AI 작업", { exact: true }).selectOption("summarize");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page
    .getByRole("button", { name: "새 개인 문서로 저장", exact: true })
    .waitFor();
  await api(`/documents/${doc.id}`, "PUT", {
    version: doc.version,
    markdown: doc.markdown + "\n새로운 결정입니다.",
  });
  await page
    .getByRole("button", { name: "새 개인 문서로 저장", exact: true })
    .click();
  await dialog
    .getByRole("button", { name: "검토한 내용을 개인 초안으로 저장" })
    .click();
  await dialog.getByText(/문서가 변경/).waitFor();
  assert.equal(
    (await api(`/documents?workspace_id=${workspace.id}`)).length,
    2,
    "stale AI draft was saved",
  );
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    "AI actions PASS: 11 native select options, real task-specific SSE, explicit private draft, source version guard, original unchanged, mobile and offline assets",
  );
} finally {
  if (workspace) {
    try {
      const settings = await api(`/workspaces/${workspace.id}/settings`);
      await api(`/workspaces/${workspace.id}/settings`, "PUT", {
        version: settings.version,
        data: { ...settings.data, ai_enabled: false },
      });
    } catch {}
  }
  await browser.close();
  await new Promise((resolve) => provider.close(resolve));
}
