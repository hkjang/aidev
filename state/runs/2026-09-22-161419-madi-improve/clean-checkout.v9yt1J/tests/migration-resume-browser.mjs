import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  output = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/migration-resume",
  );
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
    viewport: { width: 1440, height: 1024 },
    locale: "ko-KR",
  }),
  page = await context.newPage();
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
    fullPage: !(await page.getByRole("dialog").isVisible().catch(() => false)),
  });
}
try {
  await page.goto(base + "/login");
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  const workspace = await api("/workspaces", "POST", {
    name: "재개 이관 실제 검증 " + Date.now(),
  });
  await page.evaluate(
    (id) => localStorage.setItem("madi.workspace", id),
    workspace.id,
  );
  await page.goto(base + "/app/import");
  await page.getByRole("heading", { name: "가져오기", exact: true }).waitFor();
  await page
    .getByLabel("원본 식별자", { exact: true })
    .fill("resume-browser-source");
  await page
    .getByLabel("이관 이름", { exact: true })
    .fill("원문 보존과 자료형 검토");
  const original =
    "---\ntitle: 재개 이관 문서\ntags: [운영]\n---\n\n# 확인할 원문\n\n공백 두 개를 보존합니다.  \n[[둘째]]\n";
  const fileData = [
    {
      name: "첫째.md",
      mimeType: "text/markdown",
      buffer: Buffer.from(original),
    },
    {
      name: "둘째.md",
      mimeType: "text/markdown",
      buffer: Buffer.from("# 둘째 문서\n[[첫째]]\n"),
    },
    {
      name: "업무.csv",
      mimeType: "text/csv",
      buffer: Buffer.from(
        "이름,수량,전화,완료\n서버,12,01012345678,true\n노드,8,01011112222,false\n",
      ),
    },
  ];
  await page
    .getByLabel("재개 이관 파일", { exact: true })
    .setInputFiles(fileData);
  await shot("migration-resume-source");
  await page
    .getByRole("button", { name: "암호화 원본 업로드", exact: true })
    .click();
  await page
    .getByText("업로드가 완료되었습니다. 준비·검증을 실행하세요.", {
      exact: true,
    })
    .waitFor({ timeout: 45000 });
  const session = new URL(page.url()).searchParams.get("session");
  assert.ok(session);
  let v = await api("/migrations/sessions/" + session);
  assert.equal(v.status, "uploading");
  let docs = await api("/documents?workspace_id=" + workspace.id);
  assert.equal(docs.length, 0, "staging must not publish documents");
  await page.reload();
  await page
    .getByRole("button", { name: "준비·검증 실행", exact: true })
    .waitFor();
  assert.equal(new URL(page.url()).searchParams.get("session"), session);
  await page
    .getByRole("button", { name: "준비·검증 실행", exact: true })
    .click();
  await page
    .getByText("검토 대기", { exact: true })
    .last()
    .waitFor({ timeout: 45000 });
  await shot("migration-resume-checkpoints");
  const csvRow = page
    .locator(".resume-items tr")
    .filter({ has: page.getByText("업무.csv", { exact: true }) });
  await csvRow
    .getByRole("button", { name: "원문 · 변환 검토", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("수량 자료형", { exact: true }).waitFor();
  assert.equal(
    await dialog.getByLabel("수량 자료형", { exact: true }).inputValue(),
    "number",
  );
  assert.equal(
    await dialog.getByLabel("전화 자료형", { exact: true }).inputValue(),
    "text",
  );
  assert.equal(
    await dialog.getByLabel("완료 자료형", { exact: true }).inputValue(),
    "checkbox",
  );
  await shot("migration-resume-csv");
  await dialog.getByLabel("CSV 자료형 명시 확인", { exact: true }).check();
  await dialog
    .getByRole("button", { name: "자료형 확인 저장", exact: true })
    .click();
  await dialog.waitFor({ state: "hidden" });
  const docRow = page
    .locator(".resume-items tr")
    .filter({ has: page.getByText("첫째.md", { exact: true }) });
  await docRow
    .getByRole("button", { name: "원문 · 변환 검토", exact: true })
    .click();
  await dialog.getByRole("heading", { name: "원본", exact: true }).waitFor();
  assert.equal(
    await dialog
      .locator(".resume-source-comparison section")
      .first()
      .locator("pre")
      .textContent(),
    original,
  );
  assert.ok(
    (
      await dialog
        .locator(".resume-source-comparison section")
        .last()
        .locator("pre")
        .textContent()
    ).includes("/app/documents/"),
  );
  await shot("migration-resume-diff");
  await dialog.getByRole("button", { name: "닫기", exact: true }).click();
  await page
    .getByRole("button", { name: "변경 목록 확인 후 확정", exact: true })
    .click();
  await page.getByLabel("확정 문구 IMPORT", { exact: true }).fill("IMPORT");
  await shot("migration-resume-confirm");
  await page.getByRole("button", { name: "이관 확정", exact: true }).click();
  await page.getByText(/한 번에 반영했습니다/).waitFor({ timeout: 45000 });
  await shot("migration-resume-completed");
  v = await api("/migrations/sessions/" + session);
  assert.equal(v.status, "completed");
  docs = await api("/documents?workspace_id=" + workspace.id);
  assert.equal(docs.length, 2);
  assert.ok(docs.every((d) => d.visibility === "private"));
  assert.equal(
    (await api("/databases?workspace_id=" + workspace.id)).length,
    1,
  );
  await page.setViewportSize({ width: 390, height: 844 });
  assert.equal(
    await page
      .getByRole("link", { name: "내보내기", exact: true })
      .evaluate((element) => getComputedStyle(element).whiteSpace),
    "nowrap",
    "mobile export action remains on one line",
  );
  await page.reload();
  await page.getByText(/한 번에 반영했습니다/).waitFor();
  await shot("migration-resume-mobile");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "mobile page overflow " +
      JSON.stringify(
        await page.evaluate(() =>
          [...document.querySelectorAll("body *")]
            .filter((e) => e.getBoundingClientRect().width > innerWidth)
            .slice(0, 12)
            .map((e) => ({
              tag: e.tagName,
              cls: e.className,
              width: e.getBoundingClientRect().width,
            })),
        ),
      ),
  );
  const edited = await api("/documents/" + docs[0].id);
  const editedMarkdown = edited.markdown + "\n\n수정 후 재이관  \n";
  await api("/documents/" + edited.id, "PUT", {
    ...edited,
    markdown: editedMarkdown,
    expected_version: edited.version,
  });
  const exported = await api("/exports", "POST", {
    workspace_id: workspace.id,
    format: "json",
    document_ids: docs.map((d) => d.id),
  });
  let artifact;
  for (let attempt = 0; attempt < 60; attempt++) {
    artifact = await api("/exports/" + exported.id);
    if (artifact.status === "ready") break;
    assert.notEqual(artifact.status, "failed");
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  assert.equal(artifact.status, "ready");
  const download = await context.request.get(
    base + "/api/v1/exports/" + exported.id + "/download",
  );
  assert.ok(download.ok());
  const body = await download.body();
  const vault = JSON.parse(body);
  assert.ok(
    vault.files.some(
      (f) => Buffer.from(f.data_base64, "base64").toString() === editedMarkdown,
    ),
    "export must preserve edited canonical bytes",
  );
  await page.setViewportSize({ width: 1440, height: 1024 });
  await page.getByRole("button", { name: "새 이관", exact: true }).click();
  await page
    .getByLabel("원본 식별자", { exact: true })
    .fill("json-roundtrip-source");
  await page
    .getByLabel("이관 이름", { exact: true })
    .fill("수정 · 내보내기 · 재가져오기");
  await page.getByLabel("재개 이관 파일", { exact: true }).setInputFiles({
    name: "roundtrip.json",
    mimeType: "application/json",
    buffer: body,
  });
  await page
    .getByRole("button", { name: "암호화 원본 업로드", exact: true })
    .click();
  await page
    .getByText("업로드가 완료되었습니다. 준비·검증을 실행하세요.", {
      exact: true,
    })
    .waitFor({ timeout: 45000 });
  await page
    .getByRole("button", { name: "준비·검증 실행", exact: true })
    .click();
  await page
    .getByText("검토 대기", { exact: true })
    .last()
    .waitFor({ timeout: 45000 });
  await page
    .getByRole("button", { name: "변경 목록 확인 후 확정", exact: true })
    .click();
  await page.getByLabel("확정 문구 IMPORT", { exact: true }).fill("IMPORT");
  await page.getByRole("button", { name: "이관 확정", exact: true }).click();
  await page.getByText(/한 번에 반영했습니다/).waitFor({ timeout: 45000 });
  await shot("migration-resume-roundtrip");
  const after = await api("/documents?workspace_id=" + workspace.id);
  assert.equal(after.length, 4);
  const copies = await Promise.all(
    after
      .filter((d) => !docs.some((old) => old.id === d.id))
      .map((d) => api("/documents/" + d.id)),
  );
  assert.ok(copies.some((d) => d.markdown.includes("수정 후 재이관  \n")));
  for (const copy of copies) {
    assert.equal(copy.visibility, "private");
    for (const old of docs)
      assert.ok(
        !copy.markdown.includes("/app/documents/" + old.id),
        "reimport links must use new target IDs",
      );
  }
  await page.getByRole("button", { name: "새 이관", exact: true }).click();
  await page
    .getByLabel("원본 식별자", { exact: true })
    .fill("paused-upload-source");
  await page
    .getByLabel("이관 이름", { exact: true })
    .fill("전송 중단과 청크 재개");
  const large = {
    name: "체크포인트.txt",
    mimeType: "text/plain",
    buffer: Buffer.alloc(2 * 1048576 + 32, 97),
  };
  let held, arrived;
  const hold = new Promise((resolve) => (held = resolve)),
    arrival = new Promise((resolve) => (arrived = resolve));
  let initialChunks = 0;
  const countInitial = (req) => {
    if (req.method() === "PUT" && req.url().endsWith("/chunks/0"))
      initialChunks++;
  };
  page.on("request", countInitial);
  await page.route(
    "**/migrations/sessions/*/items/*/chunks/1",
    async (route) => {
      arrived();
      await hold;
      await route.continue().catch(() => {});
    },
  );
  await page.getByLabel("재개 이관 파일", { exact: true }).setInputFiles(large);
  await page
    .getByRole("button", { name: "암호화 원본 업로드", exact: true })
    .click();
  await arrival;
  await page
    .getByRole("button", { name: "업로드 중단 · 체크포인트 유지", exact: true })
    .click();
  held();
  await page.unroute("**/migrations/sessions/*/items/*/chunks/1");
  const pausedID = new URL(page.url()).searchParams.get("session");
  const paused = await api("/migrations/sessions/" + pausedID);
  assert.equal(paused.uploaded_bytes, 1048576);
  await shot("migration-resume-paused");
  await page.getByLabel("재개 이관 파일", { exact: true }).setInputFiles(large);
  await page
    .getByRole("button", { name: "체크포인트에서 재개", exact: true })
    .click();
  await page
    .getByText("업로드가 완료되었습니다. 준비·검증을 실행하세요.", {
      exact: true,
    })
    .waitFor({ timeout: 45000 });
  assert.equal(
    initialChunks,
    1,
    "already accepted first chunk must not be resent",
  );
  const resumed = await api("/migrations/sessions/" + pausedID);
  assert.equal(resumed.uploaded_bytes, large.buffer.length);
  await api("/migrations/sessions/" + pausedID, "DELETE");
  page.off("request", countInitial);
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      workspace_id: workspace.id,
      session,
      screenshots: output,
    }),
  );
} finally {
  await browser.close();
}
