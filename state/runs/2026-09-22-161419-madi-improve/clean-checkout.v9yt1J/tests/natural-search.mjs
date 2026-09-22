import assert from "node:assert/strict";
import { createServer } from "node:http";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";

const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  requests = [],
  errors = [],
  external = [];
const plan = {
  explanation:
    "GPU 관련 문서를 수정 날짜 기준으로 찾습니다. 공간과 소유자는 직접 선택한 조건을 유지합니다.",
  plan: {
    q: "GPU",
    type: "document",
    from: "2026-01-01",
    to: "",
    tag: "",
    status: "",
    has_attachment: false,
    sort: "newest",
  },
};
const provider = createServer(async (req, res) => {
  let raw = "";
  for await (const part of req) raw += part;
  const input = JSON.parse(raw);
  requests.push(input);
  const natural = input.messages[0].content.includes("검색 조건 번역기");
  const answer = natural
    ? JSON.stringify(plan)
    : "두 문서의 운영 절차를 비교했습니다. GPU 설치와 점검 범위를 구분하고 담당자와 확인 절차를 함께 검토하세요. [1] [2]";
  res.writeHead(200, { "Content-Type": "text/event-stream" });
  res.write(
    `data: ${JSON.stringify({ choices: [{ delta: { content: answer.slice(0, 20) } }] })}\n\n`,
  );
  setTimeout(
    () =>
      res.end(
        `data: ${JSON.stringify({ choices: [{ delta: { content: answer.slice(20) } }] })}\n\ndata: [DONE]\n\n`,
      ),
    200,
  );
});
await new Promise((resolve) => provider.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
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
  const me = await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  workspace = await api("/workspaces", "POST", {
    name: "질문에서 시작하는 지식 탐색",
  });
  const cfg = await api(`/workspaces/${workspace.id}/settings`);
  await api(`/workspaces/${workspace.id}/settings`, "PUT", {
    version: cfg.version,
    data: {
      ai_enabled: true,
      ai_base_url: `http://127.0.0.1:${provider.address().port}/v1`,
      ai_model: "natural-search-contract",
      ai_max_tokens: 262144,
    },
  });
  const docs = [];
  for (const title of ["GPU 설치 가이드", "GPU 점검 절차", "GPU 제외할 문서"])
    docs.push(
      await api("/documents", "POST", {
        workspace_id: workspace.id,
        title,
        markdown: `# ${title}\n\n${title}를 확인하는 담당자와 수행 순서를 기록합니다.\n`,
      }),
    );
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    workspace.id,
  );
  await page.goto(
    `${base}/app/search?q=initial&type=document&author_id=${me.id}`,
  );
  await page.getByRole("button", { name: "자연어 검색", exact: true }).click();
  const dialog = page.getByRole("dialog", {
    name: "자연어로 검색 조건 만들기",
  });
  await dialog
    .getByLabel("찾고 싶은 지식")
    .fill("올해 GPU 운영 문서를 최근 수정 순으로 찾아줘");
  assert.ok(
    await dialog
      .getByRole("button", { name: "검색 조건 제안", exact: true })
      .isDisabled(),
  );
  assert.equal(requests.length, 0);
  await dialog.getByRole("checkbox").check();
  await dialog
    .getByRole("button", { name: "검색 조건 제안", exact: true })
    .click();
  await dialog
    .getByRole("button", { name: "조건 확인 후 검색", exact: true })
    .waitFor();
  assert.equal(
    new URL(page.url()).searchParams.get("q"),
    "initial",
    "search applied without explicit confirmation",
  );
  assert.equal(requests[0].stream, true);
  assert.equal(requests[0].max_tokens, 262144);
  assert.ok(
    !JSON.stringify(requests[0]).includes("설치 가이드"),
    "document sent during condition planning",
  );
  await mkdir("docs/screenshots", { recursive: true });
  await page.screenshot({
    path: "docs/screenshots/natural-search-review.png",
    fullPage: false,
  });
  await dialog
    .getByRole("button", { name: "조건 확인 후 검색", exact: true })
    .click();
  await page.getByLabel("GPU 설치 가이드 요약에 선택").waitFor();
  assert.equal(new URL(page.url()).searchParams.get("author_id"), me.id);
  await page.reload();
  await page.getByLabel("GPU 설치 가이드 요약에 선택").check();
  await page.getByLabel("GPU 점검 절차 요약에 선택").check();
  await page
    .getByRole("button", { name: "선택 문서 AI 요약", exact: true })
    .click();
  assert.equal(requests.length, 1, "summary sent before explicit send");
  await page.getByRole("button", { name: "AI 질문 보내기" }).click();
  await page
    .getByRole("button", { name: "개인 대화 기록에 저장", exact: true })
    .waitFor();
  assert.equal(requests.length, 2);
  assert.ok(JSON.stringify(requests[1]).includes("GPU 설치 가이드"));
  assert.ok(JSON.stringify(requests[1]).includes("GPU 점검 절차"));
  assert.ok(!JSON.stringify(requests[1]).includes("GPU 제외할 문서"));
  await page.screenshot({
    path: "docs/screenshots/search-selected-summary.png",
    fullPage: false,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: "docs/screenshots/mobile-search-summary.png",
    fullPage: false,
  });
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  );
  await page.getByRole("button", { name: "AI 도우미 닫기" }).click();
  await page.getByRole("button", { name: "자연어 검색", exact: true }).click();
  await dialog.getByRole("checkbox").check();
  await dialog
    .getByRole("button", { name: "검색 조건 제안", exact: true })
    .click();
  await dialog
    .getByRole("button", { name: "조건 확인 후 검색", exact: true })
    .waitFor();
  await page.screenshot({
    path: "docs/screenshots/mobile-natural-search.png",
    fullPage: false,
  });
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  );
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    JSON.stringify({
      ok: true,
      consent: true,
      closedPlan: true,
      manualFiltersPreserved: true,
      selectedOnly: true,
      stream: true,
      maxTokens: 262144,
      mobile: 390,
      externalAssets: 0,
    }),
  );
} finally {
  if (workspace) {
    const cfg = await api(`/workspaces/${workspace.id}/settings`).catch(
      () => null,
    );
    if (cfg)
      await api(`/workspaces/${workspace.id}/settings`, "PUT", {
        version: cfg.version,
        data: { ai_enabled: false },
      }).catch(() => {});
  }
  await browser.close();
  await new Promise((resolve) => provider.close(resolve));
}
