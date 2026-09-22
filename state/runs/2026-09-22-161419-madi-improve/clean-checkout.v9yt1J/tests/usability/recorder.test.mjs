import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { chromium, expect } from "playwright/test";
const allowed = {
  "/": "recorder.html",
  "/recorder.mjs": "recorder.mjs",
  "/model.mjs": "model.mjs",
  "/recorder.css": "recorder.css",
};
const server = createServer(async (req, res) => {
  const file = allowed[req.url];
  if (!file) {
    res.writeHead(404);
    res.end();
    return;
  }
  res.setHeader(
    "Content-Type",
    file.endsWith(".html")
      ? "text/html"
      : file.endsWith(".css")
        ? "text/css"
        : "text/javascript",
  );
  res.end(await readFile(new URL(file, import.meta.url)));
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch(),
  context = await browser.newContext({ viewport: { width: 390, height: 844 } }),
  page = await context.newPage(),
  origin = `http://127.0.0.1:${server.address().port}`,
  errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (!r.url().startsWith(origin) && !r.url().startsWith("blob:"))
    external.push(r.url());
});
try {
  await page.goto(origin);
  await page.getByRole("button", { name: "과제 시작", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("익명 식별자");
  await page.getByLabel("세션 익명 ID", { exact: true }).fill("TEST-SYNTHETIC");
  await page
    .getByLabel("참가자 익명 ID", { exact: true })
    .fill("NOT-A-REAL-PARTICIPANT");
  await page.getByLabel("빌드 ID", { exact: true }).fill("recorder-fixture");
  await page.getByRole("button", { name: "과제 시작", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("동의");
  await page
    .getByLabel("참가자가 목적·익명 항목·파일 보관에 동의했습니다.", {
      exact: true,
    })
    .check();
  await page.getByRole("button", { name: "과제 시작", exact: true }).click();
  await expect(page.getByLabel("세션 익명 ID", { exact: true })).toBeDisabled();
  await page
    .getByRole("button", { name: "도움 1회 기록", exact: true })
    .click();
  await page
    .getByRole("button", { name: "되돌림 1회 기록", exact: true })
    .click();
  await page
    .getByRole("button", { name: "과제 시간 종료", exact: true })
    .click();
  await page
    .getByRole("button", { name: "확인한 관찰 추가", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText("관찰자 확인");
  await page
    .getByLabel(
      "자동 실행이 아닌 실제 사람의 수행을 관찰했고 결과를 확인했습니다.",
      { exact: true },
    )
    .check();
  await page
    .getByRole("button", { name: "확인한 관찰 추가", exact: true })
    .click();
  await expect(page.locator("#count")).toContainText("1개");
  const event = page.waitForEvent("download");
  await page
    .getByRole("button", { name: "익명 관찰 JSON 내려받기", exact: true })
    .click();
  const download = await event,
    stream = await download.createReadStream();
  let text = "";
  for await (const chunk of stream) text += chunk;
  const data = JSON.parse(text);
  assert.equal(data.length, 1);
  assert.equal(data[0].assistance_count, 1);
  assert.equal(data[0].undo_count, 1);
  assert.equal(data[0].outcome, "partial");
  assert.ok(data[0].elapsed_ms >= 0);
  assert.equal(data[0].sharing_misunderstanding, "not_assessed");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  );
  assert.equal(
    await page.evaluate(() => localStorage.length + sessionStorage.length),
    0,
  );
  assert.deepEqual(external, []);
  assert.deepEqual(errors, []);
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "현재 기록 지우기", exact: true })
    .click();
  await expect(page.locator("#count")).toContainText("관찰 미실시");
  console.log(
    "PASS recorder synthetic browser inputs only: explicit consent/observer confirmation, timing and counts, empty reset, explicit local download, storage0/external0/JavaScript0/390px; actual human observations0. Download kept only in temporary Playwright context and discarded.",
  );
} finally {
  await context.close();
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}
