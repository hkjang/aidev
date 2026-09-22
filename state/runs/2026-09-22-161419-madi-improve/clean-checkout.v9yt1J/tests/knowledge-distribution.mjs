import assert from "node:assert/strict";
import { mkdir, readFile } from "node:fs/promises";
import path from "node:path";
import { createRequire } from "node:module";
import { createPublicKey, verify } from "node:crypto";
import { chromium, expect } from "playwright/test";
const require = createRequire(new URL("../web/package.json", import.meta.url)),
  { unzipSync, zipSync } = require("fflate");
const sender = process.env.MADI_BASE_URL,
  receiver = process.env.MADI_RECEIVER_URL,
  output = path.resolve("test-results/knowledge-distribution");
await mkdir(output, { recursive: true });
const browser = await chromium.launch(),
  sourceContext = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  targetContext = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  });
const source = await sourceContext.newPage(),
  target = await targetContext.newPage(),
  errors = [],
  external = [];
for (const page of [source, target]) {
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", (route) => {
    const url = route.request().url();
    if (
      ![sender, receiver].some((base) => url.startsWith(base + "/")) &&
      /^https?:/.test(url)
    ) {
      external.push(url);
      return route.abort();
    }
    return route.continue();
  });
}
async function api(context, base, url, method = "GET", data, status = 200) {
  const res = await context.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(res.status(), status, `${method} ${url} ${await res.text()}`);
  return res.json();
}
const a = (...args) => api(sourceContext, sender, ...args),
  b = (...args) => api(targetContext, receiver, ...args);
async function shot(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(output, name + ".png"),
    fullPage: page.viewportSize().width > 500,
    animations: "disabled",
  });
}
async function confirmation(page, button) {
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("checkbox").check();
  await dialog.getByRole("button", { name: button, exact: true }).click();
  await expect(dialog).toBeHidden();
}
async function policy(page, base) {
  await page.goto(base + "/admin/knowledge-distribution");
  await page.getByLabel("서명 반출·반입 사용", { exact: true }).check();
  await page
    .getByRole("button", { name: "정책 변경 확인", exact: true })
    .click();
  await confirmation(page, "확인한 설정 저장");
  await expect(
    page.getByLabel("현재 망 식별자", { exact: true }),
  ).toBeVisible();
}
try {
  for (const [context, base] of [
    [sourceContext, sender],
    [targetContext, receiver],
  ])
    await api(context, base, "/auth/login", "POST", {
      email: "admin@example.test",
      password: process.env.MADI_ADMIN_PASSWORD,
    });
  const ws = await a("/workspaces", "POST", { name: "운영 지식 반출망" }),
    targetWS = await b("/workspaces", "POST", { name: "운영 지식 수신망" });
  const doc = await a("/documents", "POST", {
    workspace_id: ws.id,
    title: "폐쇄망 운영 기준",
    markdown: "# 폐쇄망 운영 기준\n\n검토한 원문과 첨부를 함께 전달합니다.\n",
    visibility: "workspace",
  });
  const attached = await sourceContext.request.post(
    sender + `/api/v1/attachments?document_id=${doc.id}`,
    {
      headers: { "X-Madi-Request": "1" },
      multipart: {
        file: {
          name: "운영-참고.txt",
          mimeType: "text/plain",
          buffer: Buffer.from("오프라인 첨부의 근거와 정확한 파일 해시\n"),
        },
      },
    },
  );
  assert.equal(attached.status(), 200);
  const attachment = await attached.json();
  await a(`/documents/${doc.id}`, "PUT", {
    version: doc.version,
    markdown:
      doc.markdown + `\n[참고 자료](/api/v1/attachments/${attachment.id})\n`,
    status: "published",
  });
  await policy(source, sender);
  await source
    .getByLabel("키 이름", { exact: true })
    .fill("운영망 반출 서명 키");
  await source
    .getByRole("button", { name: "생성 내용 확인", exact: true })
    .click();
  await confirmation(source, "확인한 설정 저장");
  let sourceConfig;
  await expect
    .poll(async () => {
      sourceConfig = await a("/admin/knowledge-distribution");
      return sourceConfig.keys.length;
    })
    .toBe(1);
  const key = sourceConfig.keys[0];
  await shot(source, "distribution-policy");
  await policy(target, receiver);
  const targetConfig = await b("/admin/knowledge-distribution");
  assert.notEqual(
    sourceConfig.policy.instance_id,
    targetConfig.policy.instance_id,
  );
  await target.getByLabel("키 용도", { exact: true }).selectOption("trusted");
  await target
    .getByLabel("키 이름", { exact: true })
    .fill("별도 확인한 운영망 공개키");
  for (const [label, value] of [
    ["반출망 식별자", key.source_instance],
    ["반출망 키 ID", key.source_key_id],
    ["공개키(base64)", key.public_key],
    ["별도 확인한 공개키 SHA256 지문", key.fingerprint],
  ])
    await target.getByLabel(label, { exact: true }).fill(value);
  await target
    .getByRole("button", { name: "신뢰 등록 내용 확인", exact: true })
    .click();
  await confirmation(target, "확인한 설정 저장");
  await shot(target, "distribution-trust");
  await source.goto(sender + "/app");
  await source
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await source.goto(sender + "/app/knowledge-distribution");
  await source.getByRole("checkbox", { name: /폐쇄망 운영 기준/ }).check();
  await source.getByLabel("서명 키", { exact: true }).selectOption(key.id);
  await source
    .getByLabel("수신망 식별자", { exact: true })
    .fill(targetConfig.policy.instance_id);
  await source
    .getByRole("button", { name: "반출 범위 확인", exact: true })
    .click();
  await expect(
    source.getByRole("button", {
      name: "확인한 버전으로 서명 준비",
      exact: true,
    }),
  ).toBeDisabled();
  await shot(source, "distribution-export-confirm");
  await confirmation(source, "확인한 버전으로 서명 준비");
  const downloadLink = source.getByRole("link", {
    name: "서명 ZIP 내려받기",
    exact: true,
  });
  await expect(downloadLink).toBeVisible({ timeout: 45000 });
  await shot(source, "distribution-export-ready");
  const downloadPromise = source.waitForEvent("download");
  await downloadLink.click();
  const download = await downloadPromise;
  const filename = path.join(output, download.suggestedFilename());
  await download.saveAs(filename);
  const archive = await readFile(filename),
    files = unzipSync(archive),
    manifestBytes = Buffer.from(files["madi-distribution.json"]),
    manifest = JSON.parse(manifestBytes.toString());
  assert.equal(manifest.files.length, 2);
  assert.equal(manifest.receiver_instance, targetConfig.policy.instance_id);
  const publicKey = createPublicKey({
    key: Buffer.concat([
      Buffer.from("302a300506032b6570032100", "hex"),
      Buffer.from(key.public_key, "base64"),
    ]),
    format: "der",
    type: "spki",
  });
  assert.equal(
    verify(
      null,
      manifestBytes,
      publicKey,
      Buffer.from(
        Buffer.from(files["madi-distribution.sig"]).toString(),
        "base64",
      ),
    ),
    true,
  );
  await target.goto(receiver + "/app");
  await target
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(targetWS.id);
  await target.goto(receiver + "/app/knowledge-distribution?mode=import");
  // Corrupted actual file is rejected locally before any signed receipt exists.
  const documentFile = manifest.files.find((f) => f.kind === "document");
  const corrupted = {
    ...files,
    [documentFile.path]: Buffer.from("변조된 원문"),
  };
  await target.getByLabel("서명 ZIP 파일", { exact: true }).setInputFiles({
    name: "tampered.zip",
    mimeType: "application/zip",
    buffer: Buffer.from(zipSync(corrupted)),
  });
  await expect(
    target.getByText("실제 파일과 서명 매니페스트의 SHA256이 다릅니다.", {
      exact: true,
    }),
  ).toBeVisible();
  assert.equal(
    (await b(`/migrations/sessions?workspace_id=${targetWS.id}`)).length,
    0,
  );
  await target
    .getByLabel("서명 ZIP 파일", { exact: true })
    .setInputFiles(filename);
  await expect(
    target.getByText("파일 해시 비교 완료 · 서버 서명 미확인", { exact: true }),
  ).toBeVisible();
  await shot(target, "distribution-import-preview");
  await target
    .getByRole("button", { name: "신뢰 확인·비공개 업로드", exact: true })
    .click();
  await confirmation(target, "서명 확인·업로드 시작");
  await expect(
    target.getByRole("status").filter({ hasText: "서명 원본 업로드 완료" }),
  ).toBeVisible({ timeout: 45000 });
  const session = new URL(target.url()).searchParams.get("session");
  assert.ok(session);
  let receipt = await b(`/migrations/sessions/${session}/signed-source`);
  assert.equal(receipt.imported_at, null);
  assert.equal((await b(`/documents?workspace_id=${targetWS.id}`)).length, 0);
  await target
    .getByRole("link", {
      name: "이관 센터에서 준비·원문 비교·반영",
      exact: true,
    })
    .click();
  await target
    .getByRole("button", { name: "준비·검증 실행", exact: true })
    .click();
  await target
    .getByRole("button", { name: "변경 목록 확인 후 확정", exact: true })
    .click({ timeout: 45000 });
  await target.getByLabel("확정 문구 IMPORT", { exact: true }).fill("IMPORT");
  await target.getByRole("button", { name: "이관 확정", exact: true }).click();
  await target.getByText(/한 번에 반영했습니다/).waitFor({ timeout: 45000 });
  receipt = await b(`/migrations/sessions/${session}/signed-source`);
  assert.ok(receipt.imported_at);
  const mapping = receipt.mapping.find((m) => m.kind === "document");
  const imported = await b(`/documents/${mapping.target_id}`);
  assert.equal(imported.visibility, "private");
  assert.equal(imported.status, "draft");
  assert.notEqual(imported.id, doc.id);
  assert.ok(imported.markdown.includes("검토한 원문과 첨부"));
  await target.goto(
    receiver + `/app/knowledge-distribution?mode=import&session=${session}`,
  );
  await target.getByText("서명 반입 기록", { exact: true }).click();
  await expect(target.getByText(/반영 완료/)).toBeVisible();
  await shot(target, "distribution-import-receipt");
  await target.setViewportSize({ width: 390, height: 844 });
  await target.reload();
  await expect(
    target.getByRole("heading", { name: "망별 지식 배포", exact: true }),
  ).toBeVisible();
  await target.evaluate(() => document.fonts.ready);
  assert.ok(
    await target.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await shot(target, "distribution-mobile");

  // A publication approval does not approve later attached bytes, another
  // destination or the whole export. Exercise the separate human bundle gate.
  const reviewerContext = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  });
  const reviewer = await reviewerContext.newPage();
  reviewer.on("pageerror", (e) => errors.push(e.message));
  await reviewer.route("**/*", (route) => {
    const url = route.request().url();
    if (/^https?:/.test(url) && !url.startsWith(sender + "/")) {
      external.push(url);
      return route.abort();
    }
    return route.continue();
  });
  const reviewerUser = await a("/admin/users", "POST", {
    email: "distribution-reviewer@example.test",
    name: "독립 배포 검토자",
    role: "editor",
    password: "Distribution-Reviewer-2026!",
  });
  await a(`/workspaces/${ws.id}/members`, "PUT", {
    email: reviewerUser.email,
    role: "viewer",
  });
  const r = (...args) => api(reviewerContext, sender, ...args);
  await r("/auth/login", "POST", {
    email: reviewerUser.email,
    password: "Distribution-Reviewer-2026!",
  });
  await a("/admin/settings", "PUT", { approval_enabled: true });
  for (const kind of ["document", "knowledge_distribution"])
    await a("/admin/approval/policies", "POST", {
      name: `명시 ${kind} 승인`,
      workspace_id: ws.id,
      resource_kind: kind,
      enabled: true,
      stages: [
        {
          name: "독립 검토",
          mode: "all",
          gates: [{ name: "검토 담당자", kind: "user", id: reviewerUser.id }],
        },
      ],
    });
  await a(`/documents/${doc.id}/approval`, "POST", { action: "submit" });
  const documentApproval = (await a(`/documents/${doc.id}/approval`)).request;
  await r(`/approvals/requests/${documentApproval.id}/decisions`, "POST", {
    action: "approve",
    request_version: documentApproval.version,
  });
  await source.goto(sender + "/app/knowledge-distribution");
  await source.getByRole("checkbox", { name: /폐쇄망 운영 기준/ }).check();
  await source.getByLabel("서명 키", { exact: true }).selectOption(key.id);
  await source
    .getByLabel("수신망 식별자", { exact: true })
    .fill(targetConfig.policy.instance_id);
  await source
    .getByRole("button", { name: "반출 범위 확인", exact: true })
    .click();
  await confirmation(source, "확인한 버전으로 서명 준비");
  await source
    .getByRole("link", { name: "배포 전체 검토 열기", exact: true })
    .click({ timeout: 45000 });
  const bundleID = new URL(source.url()).searchParams.get("review");
  assert.ok(bundleID);
  await expect(
    source.getByRole("heading", { name: "배포 전체 검토", exact: true }),
  ).toBeVisible();
  await shot(source, "distribution-bundle-review");
  await source
    .getByRole("button", { name: "문서·첨부 전체 검토 요청", exact: true })
    .click();
  await confirmation(source, "확인하고 진행");
  await expect(
    source.getByRole("button", { name: "승인 정책·검토·결정", exact: true }),
  ).toBeVisible();
  await reviewer.goto(
    sender + `/app/knowledge-distribution?review=${bundleID}`,
  );
  await reviewer
    .getByRole("button", { name: "승인 정책·검토·결정", exact: true })
    .click();
  const approvalDialog = reviewer.getByRole("dialog");
  await expect(
    approvalDialog.getByRole("button", { name: "검토 승인", exact: true }),
  ).toBeDisabled();
  await approvalDialog.getByRole("checkbox").check();
  await shot(reviewer, "distribution-bundle-approval");
  await approvalDialog
    .getByRole("button", { name: "검토 승인", exact: true })
    .click();
  await expect(approvalDialog).toBeHidden();
  await source
    .getByRole("button", { name: "승인 확인 후 서명 준비", exact: true })
    .click();
  await confirmation(source, "확인하고 진행");
  const reviewedDownload = source.getByRole("link", {
    name: "서명 ZIP 내려받기",
    exact: true,
  });
  await expect(reviewedDownload).toBeVisible({ timeout: 45000 });
  const signedResponse = await sourceContext.request.get(
    sender + `/api/v1/knowledge/distribution/exports/${bundleID}/download`,
  );
  assert.equal(signedResponse.status(), 200);
  const reviewedFiles = unzipSync(Buffer.from(await signedResponse.body()));
  const reviewedBytes = Buffer.from(reviewedFiles["madi-distribution.json"]);
  const reviewedManifest = JSON.parse(reviewedBytes.toString());
  assert.equal(reviewedManifest.bundle_approval.required, true);
  assert.equal(
    verify(
      null,
      reviewedBytes,
      publicKey,
      Buffer.from(
        Buffer.from(reviewedFiles["madi-distribution.sig"]).toString(),
        "base64",
      ),
    ),
    true,
  );
  await shot(source, "distribution-bundle-approved");
  await reviewerContext.close();
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    "PASS two isolated installations plus independent reviewer: admin policy/signing/trust, current-version export consent, actual Ed25519 verification, tampered ZIP rejected before upload, trusted private staging→review→atomic document+attachment mapping, separate full-bundle approval→original requester signing, URL reload and390px, zero JS/external requests.",
  );
} catch (error) {
  await shot(source, "failure-source");
  await shot(target, "failure-target");
  throw error;
} finally {
  await browser.close();
}
