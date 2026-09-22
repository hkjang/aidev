import {documentTool,documentPanel} from "./document-ui.mjs";
import assert from "node:assert/strict";
import { chromium, expect } from "playwright/test";

const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const browser = await chromium.launch(),
  admin = await browser.newContext(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
  }),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(path, method = "GET", data, client = context) {
  const r = await client.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(r.ok(), `${method} ${path}: ${await r.text()}`);
  return r.json();
}
const title = () => page.getByLabel("문서 제목", { exact: true });
const source = () => page.getByLabel("Markdown 원문 편집", { exact: true });
async function navigate(doc) {
  await page.evaluate((id) => {
    history.pushState({}, "", `/app/documents/${id}?mode=source`);
    dispatchEvent(new PopStateEvent("popstate"));
  }, doc.id);
  await title().waitFor();
  await page.waitForFunction(
    (value) =>
      document.querySelector('[aria-label="문서 제목"]')?.value === value,
    doc.title,
  );
  assert.equal(await source().inputValue(), doc.markdown);
}
async function holdResponse(path, method) {
  let release, signal, finished;
  const gate = new Promise((r) => {
      release = r;
    }),
    started = new Promise((r) => {
      signal = r;
    }),
    completed = new Promise((r) => {
      finished = r;
    });
  const url = base + "/api/v1" + path;
  const handler = async (route) => {
    if (route.request().method() !== method) return route.continue();
    const response = await route.fetch();
    assert.ok(response.ok(), await response.text());
    signal();
    await gate;
    await route.fulfill({ response });
    finished();
  };
  await page.route(url, handler);
  return {
    started,
    async release() {
      release();
      await completed;
      await page.unroute(url, handler);
      await page.waitForTimeout(300);
    },
  };
}
try {
  await api(
    "/auth/login",
    "POST",
    {
      email: "admin@example.test",
      password:
        process.env.MADI_ADMIN_PASSWORD || "Browser-Test-Password-2026!",
    },
    admin,
  );
  const email = `async-doc-${Date.now()}@example.test`,
    password = "Async-document-test-2026!";
  await api(
    "/admin/users",
    "POST",
    { email, password, name: "문서 지연 응답 검증", role: "editor" },
    admin,
  );
  await api("/auth/login", "POST", { email, password });
  const ws = await api("/workspaces", "POST", {
    name: "문서 비동기 격리 검증",
  });
  const a = await api("/documents", "POST", {
      workspace_id: ws.id,
      title: "첫 번째 원문",
      markdown: "# 첫 번째\n\nA의 저장된 원문입니다.\n",
      visibility: "selected",
    }),
    b = await api("/documents", "POST", {
      workspace_id: ws.id,
      title: "두 번째 원문",
      markdown: "# 두 번째\n\nB의 저장된 원문입니다.\n",
      visibility: "selected",
    });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  await page.goto(base + `/app/documents/${a.id}?mode=source`);
  await source().waitFor();

  const favorite = await holdResponse(`/documents/${a.id}/favorite`, "POST");
  await page.getByRole("button", { name: "즐겨찾기", exact: true }).click();
  await favorite.started;
  await navigate(b);
  await favorite.release();
  assert.equal(await title().inputValue(), b.title);
  assert.equal(await source().inputValue(), b.markdown);
  assert.equal((await api(`/documents/${a.id}`)).is_favorite, true);

  await navigate(a);
  const shares = await holdResponse(`/documents/${a.id}/shares`, "GET");
  await page.getByRole("button", { name: "공유", exact: true }).click();
  await shares.started;
  await navigate(b);
  assert.equal(await page.getByRole("dialog").count(), 0);
  await page.getByRole("button", { name: "공유", exact: true }).click();
  await page.getByLabel("문서 링크", { exact: true }).waitFor();
  await shares.release();
  assert.match(
    await page.getByLabel("문서 링크", { exact: true }).inputValue(),
    new RegExp(b.id),
  );
  await page.getByRole("button", { name: "취소", exact: true }).click();

  await navigate(a);
  await page.getByRole("button", { name: "공유", exact: true }).click();
  await page.getByLabel("문서 별칭", { exact: true }).fill("첫 문서 별칭");
  const properties = await holdResponse(`/documents/${a.id}`, "PUT");
  await page.getByRole("button", { name: "설정 저장", exact: true }).click();
  await properties.started;
  await navigate(b);
  await properties.release();
  assert.equal(await title().inputValue(), b.title);
  assert.equal(await source().inputValue(), b.markdown);
  assert.equal(await page.getByRole("dialog").count(), 0);
  assert.deepEqual((await api(`/documents/${a.id}`)).aliases, ["첫 문서 별칭"]);

  await navigate(a);
  const duplicate = await holdResponse("/documents", "POST");
  await documentPanel(page,"속성");
  await page.getByRole("button", { name: "문서 복제", exact: true }).click();
  await duplicate.started;
  await navigate(b);
  await duplicate.release();
  assert.ok(
    page.url().includes(b.id),
    "late duplicate must not navigate away from the active document",
  );
  assert.equal(await source().inputValue(), b.markdown);

  // Property-only writes cannot replace locally edited Markdown with an older
  // server body. Let the normal autosave persist the retained draft afterwards.
  await navigate(a);
  const changed = a.markdown + "\n아직 저장하지 않은 입력을 보존합니다.\n";
  await source().fill(changed);
  await page.getByRole("button", { name: "공유", exact: true }).click();
  await page.getByLabel("문서 별칭", { exact: true }).fill("수정한 별칭");
  await page.getByRole("button", { name: "설정 저장", exact: true }).click();
  await page.getByRole("dialog").waitFor({ state: "hidden" });
  assert.equal(await source().inputValue(), changed);
  await page.waitForTimeout(2300);
  assert.equal((await api(`/documents/${a.id}`)).markdown, changed);
  assert.equal((await api(`/documents/${b.id}`)).markdown, b.markdown);
  const other = await api("/workspaces", "POST", { name: "다른 팀 설정 격리" });
  await api(`/workspaces/${ws.id}/settings`, "PUT", {
    version: 1,
    data: { review_period_days: 91 },
  });
  await api(`/workspaces/${other.id}/settings`, "PUT", {
    version: 1,
    data: { review_period_days: 181 },
  });
  const settingField = () =>
    page.getByLabel("문서 검토 주기 (일)", { exact: true });
  async function selectWorkspace(id) {
    await page
      .getByLabel("워크스페이스 선택", { exact: true })
      .selectOption(id);
    await page.waitForURL(base + "/app");
    // Browser history changes before React commits the route. Reading the old
    // settings route's open <details> here can skip the required summary click,
    // just before the home commit closes it. Wait for the visible router state.
    await expect(
      page.getByLabel("워크스페이스 선택", { exact: true }),
    ).toHaveValue(id);
    await expect(
      page
        .getByRole("navigation", { name: "주요 메뉴", exact: true })
        .getByRole("link", { name: "홈", exact: true }),
    ).toHaveAttribute("aria-current", "page");
    const management = page.locator("details.workspace-management");
    await management.locator("summary").waitFor();
    if ((await management.getAttribute("open")) === null)
      await management.locator("summary").click();
    await management
      .getByRole("link", { name: "워크스페이스 설정", exact: true })
      .click();
    await settingField().waitFor();
  }
  const settingsRead = await holdResponse(
    `/workspaces/${ws.id}/settings`,
    "GET",
  );
  await page.goto(base + "/app/workspace-settings");
  await settingsRead.started;
  await selectWorkspace(other.id);
  assert.equal(await settingField().inputValue(), "181");
  await settingsRead.release();
  assert.equal(await settingField().inputValue(), "181");

  await selectWorkspace(ws.id);
  assert.equal(await settingField().inputValue(), "91");
  await settingField().fill("92");
  const settingsWrite = await holdResponse(
    `/workspaces/${ws.id}/settings`,
    "PUT",
  );
  await page.getByRole("button", { name: "설정 저장", exact: true }).click();
  await settingsWrite.started;
  await selectWorkspace(other.id);
  await settingsWrite.release();
  assert.equal(await settingField().inputValue(), "181");
  assert.equal(
    (await api(`/workspaces/${ws.id}/settings`)).data.review_period_days,
    92,
  );
  assert.equal(
    (await api(`/workspaces/${other.id}/settings`)).data.review_period_days,
    181,
  );
  assert.equal(
    await page
      .getByText("워크스페이스 설정을 저장했습니다", { exact: true })
      .count(),
    0,
  );
  assert.deepEqual(errors, []);
  console.log(
    "PASS delayed favorite/shares/properties/duplicate never cross document routes; property-only save preserves unsaved Markdown; workspace GET/PUT scope isolation; JS errors 0",
  );
} catch (error) {
  await page.screenshot({
    path: "test-results/document-async-failure.png",
    fullPage: true,
  });
  throw error;
} finally {
  await browser.close();
}
