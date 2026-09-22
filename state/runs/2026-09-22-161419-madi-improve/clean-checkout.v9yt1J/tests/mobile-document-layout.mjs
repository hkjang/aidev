import assert from "node:assert/strict";
import { chromium } from "playwright";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
  }),
  page = await context.newPage();
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(response.ok(), await response.text());
  return response.json();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", {
    name: "아주 긴 워크스페이스 이름을 가진 AI 문서 연구 공간",
  });
  const doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "모바일 화면 가로 넘침 검증 문서",
    markdown: "# 화면 검증\n\n문서 원문과 긴 제목을 확인합니다.",
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  await page.goto(base + `/app/documents/${doc.id}?mode=preview`);
  await page.getByLabel("문서 제목", { exact: true }).waitFor();
  await page.getByRole("button", { name: "공유", exact: true }).click();
  await page
    .getByRole("dialog", { name: "문서 공유 및 속성", exact: true })
    .waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForTimeout(250);
  const result = await page.evaluate(() => ({
    width: innerWidth,
    scroll: document.documentElement.scrollWidth,
    body: {
      style: document.body.getAttribute("style"),
      width: getComputedStyle(document.body).width,
      marginRight: getComputedStyle(document.body).marginRight,
    },
    elements: [...document.querySelectorAll("body *")]
      .filter((el) => el.getBoundingClientRect().right > innerWidth + 1)
      .slice(0, 40)
      .map((el) => ({
        tag: el.tagName,
        cls: String(el.className),
        width: el.getBoundingClientRect().width,
        left: el.getBoundingClientRect().left,
        right: el.getBoundingClientRect().right,
        min: getComputedStyle(el).minWidth,
        flex: getComputedStyle(el).flex,
      })),
  }));
  console.log(JSON.stringify(result, null, 2));
  assert.ok(result.scroll <= result.width + 1);
  console.log(
    "PASS document desktop-to-mobile resize while modal remains open",
  );
} finally {
  await browser.close();
}
