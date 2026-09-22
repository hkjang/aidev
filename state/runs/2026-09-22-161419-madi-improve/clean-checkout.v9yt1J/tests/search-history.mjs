import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  requests = [],
  errors = [],
  external = [];
const provider = createServer(async (req, res) => {
  let raw = "";
  for await (const v of req) raw += v;
  requests.push(JSON.parse(raw));
  const result = {
    notice:
      "선택한 개인 검색 실패만 분석한 제안입니다. 실제 문서가 없는지 확인이 필요합니다.",
    suggestions: [
      {
        title: "GPU 장애 조치 기준 정리",
        reason:
          "같은 질문의 검색 실패가 반복되어 용어와 조치 기준을 정리할 후보입니다.",
        outline:
          "# 목적\n\n조치 기준과 담당자를 확인합니다.\n\n## 확인할 질문\n\n- 어떤 경고부터 조치하나요?\n- 관련 문서가 다른 용어로 존재하나요?",
        references: [1],
      },
    ],
  };
  res.writeHead(200, { "Content-Type": "text/event-stream" });
  res.write(
    `data: ${JSON.stringify({ choices: [{ delta: { content: JSON.stringify(result).slice(0, 25) } }] })}\n\n`,
  );
  setTimeout(
    () =>
      res.end(
        `data: ${JSON.stringify({ choices: [{ delta: { content: JSON.stringify(result).slice(25) } }] })}\n\ndata: [DONE]\n\n`,
      ),
    200,
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
async function api(context, path, method = "GET", data) {
  const r = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${path}: ${r.status()} ${await r.text()}`);
  return r.json();
}
let workspace;
try {
  await api(admin, "/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  workspace = await api(admin, "/workspaces", "POST", {
    name: "검색에서 발견하는 지식의 빈자리",
  });
  const user = await api(admin, "/admin/users", "POST", {
    email: `search-history-${Date.now()}@example.test`,
    name: "지식 탐색 사용자",
    password: "Personal-Search-Password-2026!",
    role: "editor",
  });
  await api(admin, `/workspaces/${workspace.id}/members`, "PUT", {
    email: user.email,
    role: "editor",
  });
  const cfg = await api(admin, `/workspaces/${workspace.id}/settings`);
  await api(admin, `/workspaces/${workspace.id}/settings`, "PUT", {
    version: cfg.version,
    data: {
      ai_enabled: true,
      ai_base_url: `http://127.0.0.1:${provider.address().port}/v1`,
      ai_model: "personal-search-gap",
      ai_max_tokens: 262144,
    },
  });
  await api(member, "/auth/login", "POST", {
    email: user.email,
    password: "Personal-Search-Password-2026!",
  });
  await member.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspace.id,
  );
  await page.goto(base + "/app/search-history");
  await page.getByText("검색어 저장 꺼짐", { exact: false }).waitFor();
  await api(
    member,
    `/search?workspace_id=${workspace.id}&q=${encodeURIComponent("GPU 장애 조치 기준")}&type=document`,
  );
  assert.equal(
    (await api(member, `/search/history?workspace_id=${workspace.id}`)).items
      .length,
    0,
  );
  await page.getByRole("button", { name: "기록 설정", exact: true }).click();
  const settings = page.getByRole("dialog", { name: "개인 검색 기록 설정" });
  await settings.getByRole("checkbox").check();
  await settings.getByLabel("보존 기간 (일)").fill("14");
  await settings
    .getByRole("button", { name: "설정 저장", exact: true })
    .click();
  await page.getByText("검색어 저장 중", { exact: false }).waitFor();
  for (let i = 0; i < 2; i++)
    await api(
      member,
      `/search?workspace_id=${workspace.id}&q=${encodeURIComponent("GPU 장애 조치 기준")}&type=document`,
    );
  await page.getByRole("button", { name: "새로고침", exact: true }).click();
  await page.getByLabel("GPU 장애 조치 기준 분석에 선택").check();
  assert.equal(
    (await api(admin, `/search/history?workspace_id=${workspace.id}`)).items
      .length,
    0,
    "admin bypassed personal search history",
  );
  await mkdir("docs/screenshots", { recursive: true });
  await page.screenshot({
    path: "docs/screenshots/personal-search-history.png",
    fullPage: true,
  });
  await page
    .getByRole("button", { name: /선택 1\/30건 지식 보완 제안/ })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "검색 실패에서 지식 보완하기",
  });
  assert.ok(
    await dialog
      .getByRole("button", { name: "보완할 지식 제안", exact: true })
      .isDisabled(),
  );
  assert.equal(requests.length, 0);
  await dialog.getByRole("checkbox").check();
  await dialog
    .getByRole("button", { name: "보완할 지식 제안", exact: true })
    .click();
  await dialog
    .getByRole("heading", { name: "GPU 장애 조치 기준 정리", exact: true })
    .waitFor();
  assert.equal(requests[0].stream, true);
  assert.equal(requests[0].max_tokens, 262144);
  assert.ok(JSON.stringify(requests[0]).includes("GPU 장애 조치 기준"));
  await page.screenshot({ path: "docs/screenshots/search-knowledge-gap.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: "docs/screenshots/mobile-search-gap.png" });
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  );
  await dialog
    .getByRole("button", { name: "새 개인 문서로 저장", exact: true })
    .click();
  const save = page.getByRole("dialog", { name: "AI 초안 검토" });
  await save
    .getByRole("button", {
      name: "검토한 내용을 개인 초안으로 저장",
      exact: true,
    })
    .click();
  await page.waitForURL(/\/app\/documents\//);
  const id = new URL(page.url()).pathname.split("/").pop(),
    doc = await api(member, "/documents/" + id);
  assert.equal(doc.visibility, "private");
  assert.equal(doc.status, "draft");
  await page.goto(base + "/app/search-history");
  await page.getByRole("button", { name: "기록 설정", exact: true }).click();
  await settings.getByRole("checkbox").uncheck();
  await settings
    .getByRole("button", { name: "설정 저장", exact: true })
    .click();
  await page.getByText("검색어 저장 꺼짐", { exact: false }).waitFor();
  await page
    .getByRole("button", { name: "이 워크스페이스 기록 삭제", exact: true })
    .click();
  await page
    .getByRole("dialog", { name: "개인 검색 기록 삭제" })
    .getByRole("button", { name: "확인하고 모두 삭제", exact: true })
    .click();
  await page
    .getByText("저장된 검색 기록이 없습니다", { exact: true })
    .waitFor();
  await page.reload();
  await page
    .getByText("저장된 검색 기록이 없습니다", { exact: true })
    .waitFor();
  assert.equal(
    (await api(member, `/search/history?workspace_id=${workspace.id}`)).items
      .length,
    0,
  );
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    JSON.stringify({
      ok: true,
      defaultOff: true,
      personalOnly: true,
      explicitAIConsent: true,
      privateDraft: true,
      groupedCounts: true,
      erasure: true,
      mobile: 390,
      externalAssets: 0,
    }),
  );
} finally {
  if (workspace) {
    const cfg = await api(admin, `/workspaces/${workspace.id}/settings`).catch(
      () => null,
    );
    if (cfg)
      await api(admin, `/workspaces/${workspace.id}/settings`, "PUT", {
        version: cfg.version,
        data: { ai_enabled: false },
      }).catch(() => {});
  }
  await browser.close();
  await new Promise((resolve) => provider.close(resolve));
}
