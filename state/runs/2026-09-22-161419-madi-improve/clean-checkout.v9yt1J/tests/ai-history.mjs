import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const answer =
  "점검 담당자를 확인하고 운영 절차를 검토합니다. 저장한 답변의 출처를 다시 확인하세요. [1]";
const providerRequests = [];
const provider = createServer(async (req, res) => {
  let raw = "";
  for await (const chunk of req) {
    raw += chunk;
  }
  providerRequests.push(JSON.parse(raw));
  res.writeHead(200, { "Content-Type": "text/event-stream" });
  res.end(
    `data: ${JSON.stringify({ choices: [{ delta: { content: answer } }] })}\n\ndata: [DONE]\n\n`,
  );
});
await new Promise((resolve) => provider.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch(),
  admin = await browser.newContext(),
  member = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await member.newPage();
const errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`);
});
await member.route("**/*", (route) => {
  if (new URL(route.request().url()).origin === new URL(base).origin)
    return route.continue();
  external.push(route.request().url());
  return route.abort();
});
async function api(context, path, method = "GET", data, status = 200) {
  const r = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(r.status(), status, `${path}: ${r.status()} ${await r.text()}`);
  return r.json();
}
let workspace;
try {
  await api(admin, "/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  workspace = await api(admin, "/workspaces", "POST", {
    name: "나만의 AI 지식 기록",
  });
  const user = await api(admin, "/admin/users", "POST", {
    email: `ai-history-${Date.now()}@example.test`,
    name: "지식 기록 사용자",
    password: "Browser-History-Password-2026!",
    role: "editor",
  });
  await api(admin, `/workspaces/${workspace.id}/members`, "PUT", {
    email: user.email,
    role: "editor",
  });
  const settings = await api(admin, `/workspaces/${workspace.id}/settings`);
  await api(admin, `/workspaces/${workspace.id}/settings`, "PUT", {
    version: settings.version,
    data: {
      ai_enabled: true,
      ai_base_url: `http://127.0.0.1:${provider.address().port}/v1`,
      ai_model: "local-history-contract",
    },
  });
  const doc = await api(admin, "/documents", "POST", {
    workspace_id: workspace.id,
    title: "운영 점검 가이드",
    markdown: "# 운영 점검\n\n담당자를 확인하고 운영 절차를 검토합니다.\n",
  });
  await api(member, "/auth/login", "POST", {
    email: user.email,
    password: "Browser-History-Password-2026!",
  });
  await member.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspace.id,
  );
  await page.goto(`${base}/app/documents/${doc.id}?mode=preview`);
  await page
    .locator(".topbar")
    .getByRole("button", { name: "AI 도우미", exact: true })
    .click();
  await page
    .getByLabel("AI 질문", { exact: true })
    .fill("운영 점검은 어떤 순서로 진행할까요?");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page
    .getByRole("button", { name: "개인 대화 기록에 저장", exact: true })
    .waitFor();
  assert.equal(
    (await api(member, `/ai/conversations?workspace_id=${workspace.id}`)).items
      .length,
    0,
    "history saved without consent",
  );
  await page
    .getByRole("button", { name: "개인 대화 기록에 저장", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "개인 대화 기록 저장" });
  await dialog
    .getByRole("button", { name: "동의하고 개인 기록에 저장" })
    .click();
  const link = page.getByRole("link", { name: "저장한 대화 보기" });
  await link.waitFor();
  const id = new URL(await link.getAttribute("href"), base).searchParams.get(
    "id",
  );
  await link.click();
  await page
    .getByRole("heading", { name: "저장한 답변", exact: true })
    .waitFor();
  await mkdir(new URL("../docs/screenshots/", import.meta.url), {
    recursive: true,
  });
  await page.screenshot({
    path: new URL(
      "../docs/screenshots/personal-ai-history.png",
      import.meta.url,
    ).pathname,
    fullPage: true,
  });
  await page.reload();
  await page.getByText(answer, { exact: true }).waitFor();
  await page
    .getByRole("button", { name: "기록을 AI에 보내고 이어서 질문" })
    .click();
  await page
    .getByLabel("AI 질문", { exact: true })
    .fill("운영 점검의 구체적인 다음 단계를 정리해 주세요");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page
    .getByRole("button", { name: "개인 대화 기록에 저장", exact: true })
    .waitFor();
  assert.ok(
    providerRequests[1].messages.some(
      (message) => message.role === "assistant" && message.content === answer,
    ),
    "selected prior conversation was not sent",
  );
  await page
    .getByRole("button", { name: "개인 대화 기록에 저장", exact: true })
    .click();
  await page
    .getByRole("dialog", { name: "개인 대화 기록 저장" })
    .getByRole("button", { name: "동의하고 개인 기록에 저장" })
    .click();
  await page.getByRole("link", { name: "저장한 대화 보기" }).click();
  const continued = await api(member, `/ai/conversations/${id}`);
  assert.equal(continued.version, 2);
  assert.equal(continued.messages.length, 2);
  await page.reload();
  await page.getByText(answer, { exact: true }).first().waitFor();
  await api(admin, `/ai/conversations/${id}`, "GET", undefined, 404);
  await page
    .getByRole("button", { name: /인용 원문/ })
    .first()
    .click();
  await page
    .getByRole("dialog", { name: "인용 원문 확인" })
    .locator("pre")
    .waitFor();
  await page.keyboard.press("Escape");
  await page.goto(
    `${base}/app/search?type=ai_conversation&q=${encodeURIComponent("운영 점검")}`,
  );
  await page.locator('a[href*="/app/ai-history?id="]').first().waitFor();
  await page.screenshot({
    path: new URL(
      "../docs/screenshots/search-personal-ai-history.png",
      import.meta.url,
    ).pathname,
    fullPage: true,
  });
  await page.locator('a[href*="/app/ai-history?id="]').first().click();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByText(answer, { exact: true }).first().waitFor();
  await page.waitForFunction(
    () => document.documentElement.scrollWidth <= innerWidth + 1,
  );
  await page.screenshot({
    path: new URL(
      "../docs/screenshots/mobile-personal-ai-history.png",
      import.meta.url,
    ).pathname,
    fullPage: true,
  });
  await api(admin, `/documents/${doc.id}`, "PUT", {
    version: doc.version,
    visibility: "private",
  });
  await page
    .getByText(/참조 문서 권한이 변경되었습니다/)
    .waitFor({ timeout: 10000 });
  assert.equal(
    await page.getByText(answer, { exact: true }).count(),
    0,
    "cached answer remained after source revocation",
  );
  assert.equal(
    (await api(member, `/ai/conversations?workspace_id=${workspace.id}`)).items
      .length,
    0,
  );
  const revokedSearch = await api(
    member,
    `/search?workspace_id=${workspace.id}&type=ai_conversation&q=${encodeURIComponent("운영 점검")}`,
  );
  assert.equal(revokedSearch.results.length, 0);
  await api(member, `/ai/conversations/${id}`, "DELETE", {
    version: 2,
    confirmation: "DELETE",
  });
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    "AI history PASS: explicit consent, durable owner-only multi-turn context, same provider and versioned append, exact citation, universal search, reload/mobile, ACL revocation clears cached body, owner cleanup",
  );
} finally {
  if (workspace) {
    try {
      const settings = await api(admin, `/workspaces/${workspace.id}/settings`);
      await api(admin, `/workspaces/${workspace.id}/settings`, "PUT", {
        version: settings.version,
        data: { ...settings.data, ai_enabled: false },
      });
    } catch {}
  }
  await browser.close();
  await new Promise((resolve) => provider.close(resolve));
}
