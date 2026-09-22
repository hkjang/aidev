import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL,
  output = path.resolve("test-results/knowledge-operations");
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1512, height: 1080 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage();
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
async function api(url, method = "GET", data, status = 200) {
  const res = await context.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(res.status(), status, `${method} ${url}: ${await res.text()}`);
  return res.json();
}
async function shot(name) {
  await mkdir(output, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(output, name + ".png"),
    fullPage: page.viewportSize().width > 500,
    animations: "disabled",
  });
}
async function overflow() {
  // CDP viewport acknowledgement can precede the CSS media-query layout update.
  // Wait for the actual mobile shell, then keep the original overflow assertion.
  if (page.viewportSize().width <= 800) {
    await expect.poll(() => page.locator(".main").evaluate((node) =>
      matchMedia("(max-width: 800px)").matches && getComputedStyle(node).marginLeft === "0px",
    )).toBe(true);
  }
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  const bad = await page.evaluate(() =>
    [...document.querySelectorAll("body *")]
      .map((e) => ({
        tag: e.tagName,
        cls: e.className,
        right: e.getBoundingClientRect().right,
        width: e.getBoundingClientRect().width,
        text: e.textContent?.slice(0, 80),
      }))
      .filter((e) => e.right > innerWidth + 1 && e.width > 0)
      .slice(-15),
  );
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "horizontal overflow " + JSON.stringify(bad),
  );
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: process.env.MADI_ADMIN_PASSWORD,
  });
  await api("/admin/settings", "PUT", { approval_enabled: true });
  const ws = await api("/workspaces", "POST", {
    name: "근거와 변경 영향 검증",
  });
  const original =
    "# GPU 운영 정책\n\n동시 작업은 2개입니다. 변경 전에 담당 검토가 필요합니다.\n";
  const source = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "GPU 운영 정책",
    markdown: original,
  });
  const target = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "GPU 운영 절차",
    markdown: "# GPU 운영 절차\n\n정책 기준으로 동시 작업을 설정합니다.",
  });
  await api(`/documents/${target.id}/relations`, "POST", {
    target_id: source.id,
    type: "policy",
    expected_version: 1,
  });
  await page.goto(base + "/admin/evidence");
  await page
    .getByLabel("근거 보관 활성화", { exact: true })
    .selectOption("true");
  await page.getByLabel("최대 보존기간 (일)", { exact: true }).fill("30");
  await page.getByRole("button", { name: "정책 저장", exact: true }).click();
  await expect
    .poll(async () => (await api("/admin/evidence-policy")).retention_days)
    .toBe(30);
  await shot("evidence-policy");
  await page.goto(base + "/admin/knowledge-packages");
  await page
    .getByLabel("허용할 토큰 계산", { exact: true })
    .selectOption("estimate");
  await page
    .getByRole("button", { name: "패키지 정책 저장", exact: true })
    .click();
  await shot("knowledge-package-policy");
  await page.goto(base + "/app");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(ws.id);
  await page.goto(base + `/app/documents/${source.id}?mode=read`);
  await page.getByRole("tab", { name: "AI", exact: true }).click();
  await page
    .getByLabel("AI 질문", { exact: true })
    .fill("이 정책의 동시 작업 기준을 알려 주세요");
  await page.getByLabel("AI 질문 보내기", { exact: true }).click();
  await page
    .getByRole("button", { name: "당시 근거 보관", exact: true })
    .click();
  await page
    .getByRole("button", { name: "동의하고 근거 보관", exact: true })
    .click();
  await page
    .getByRole("link", { name: "보관한 근거 보기", exact: true })
    .click();
  await page
    .getByRole("heading", { name: "당시 질문과 답변", exact: true })
    .waitFor();
  const evidenceID = new URL(page.url()).searchParams.get("id");
  assert.ok(evidenceID);
  await expect(page.getByText("최신 버전", { exact: true })).toBeVisible();
  await page
    .getByLabel("검토 결과", { exact: true })
    .selectOption("insufficient");
  await page
    .getByLabel("검토 의견", { exact: true })
    .fill("운영 적용 전 예외 조건을 추가 확인합니다.");
  await page
    .getByRole("button", { name: "검토 기록 저장", exact: true })
    .click();
  await expect
    .poll(async () => (await api("/ai/evidence/" + evidenceID)).review_status)
    .toBe("insufficient");
  await shot("evidence-record");
  await page.goto(base + "/app/knowledge-packages");
  await page
    .getByLabel("업무 목적", { exact: true })
    .fill("GPU 운영 변경안 검토");
  await page.getByLabel("사용 모델", { exact: true }).fill("사내 검증 모델");
  await page.getByLabel("입력 예산 (512~262144)", { exact: true }).fill("8192");
  await page
    .getByRole("checkbox", { name: "GPU 운영 정책", exact: true })
    .check();
  await page
    .getByRole("checkbox", { name: "필수 정책 (전체 원문 보존)", exact: true })
    .check();
  await page
    .getByLabel("GPU 운영 정책 포함 이유", { exact: true })
    .fill("변경 전에 반드시 확인할 필수 정책");
  await page
    .getByRole("checkbox", { name: /현재 권한이 적용되는 암호화 사본/ })
    .check();
  await shot("knowledge-package-compose");
  await page
    .getByRole("button", { name: "지식 패키지 구성", exact: true })
    .click();
  await page
    .getByRole("button", { name: "전달 조건 확인·내보내기", exact: true })
    .waitFor();
  const packageID = new URL(page.url()).searchParams.get("id");
  assert.ok(packageID);
  await expect(
    page.getByText("현재 원문 버전과 일치", { exact: false }),
  ).toBeVisible();
  await shot("knowledge-package-record");
  await api("/documents/" + source.id, "PUT", {
    version: 1,
    markdown: "# GPU 운영 정책\n\n승인 후 동시 작업을 4개로 변경합니다.\n",
  });
  await expect(
    page.getByRole("button", { name: "전달 조건 확인·내보내기", exact: true }),
  ).toBeDisabled();
  await page.goto(base + "/app/evidence?id=" + evidenceID);
  await expect(
    page.getByText("변경됨 · 현재 v2", { exact: true }),
  ).toBeVisible();
  await expect(
    page.locator("pre").filter({ hasText: "동시 작업은 2개입니다." }).first(),
  ).toBeVisible();
  await shot("evidence-historical");
  await page.goto(base + `/app/knowledge-impact?document_id=${source.id}`);
  await page.getByLabel("이전 원문 버전", { exact: true }).fill("1");
  await page
    .getByRole("button", { name: "현재 버전의 영향 분석", exact: true })
    .click();
  await page
    .getByRole("heading", { name: "GPU 운영 절차", exact: true })
    .waitFor();
  await expect(
    page.getByText("수치 변경 후보", { exact: false }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "문서 미리보기", exact: true })
    .click();
  await page.getByRole("dialog").waitFor();
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", { name: "문서 미리보기", exact: true }),
  ).toBeFocused();
  await page
    .getByRole("button", { name: "소유자에게 영향 검토 요청", exact: true })
    .click();
  await page
    .getByRole("button", { name: "검토 내용·이력", exact: true })
    .waitFor();
  await shot("knowledge-impact");
  await page
    .getByRole("button", { name: "검토 내용·이력", exact: true })
    .click();
  await page
    .getByLabel("검토 처리 결과", { exact: true })
    .selectOption("needs_change");
  await page
    .getByLabel("검토 근거 의견", { exact: true })
    .fill("정책의 새 동시 작업 한도에 맞춰 운영 절차를 검토합니다.");
  await page
    .getByRole("button", { name: "현재 버전 검토 결과 저장", exact: true })
    .click();
  await expect(page.getByText("수정 필요", { exact: false })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(
      () =>
        page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      { timeout: 5000 },
    )
    .toBe(true);
  await overflow();
  await page.evaluate(() => window.scrollTo(0, 0));
  await shot("knowledge-impact-mobile");
  for (const [route, name] of [
    [`/app/evidence?id=${evidenceID}`, "evidence-mobile"],
    [`/app/knowledge-packages?id=${packageID}`, "knowledge-package-mobile"],
  ]) {
    await page.goto(base + route);
    await page.getByRole("heading", { level: 1 }).waitFor();
    await overflow();
    await shot(name);
  }
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.goto(base + `/app/knowledge-proposals?document_id=${target.id}`);
  await page
    .getByLabel("제안 Markdown 원문", { exact: true })
    .fill(
      "# GPU 운영 절차\n\n정책 변경에 따라 승인 후 동시 작업 4개를 적용합니다.",
    );
  await page
    .getByLabel("변경 이유", { exact: true })
    .fill("정책 변경의 영향 검토에 따른 운영 절차 수정");
  await page
    .getByRole("checkbox", { name: /현재 문서 열람자가 이 제안/ })
    .check();
  await page
    .getByRole("button", { name: "공유 전 변경 비교", exact: true })
    .click();
  await page
    .getByRole("button", { name: "동의한 범위로 변경안 공유", exact: true })
    .click();
  await page
    .getByRole("heading", { name: "변경 내용 비교", exact: true })
    .waitFor();
  const proposalID = new URL(page.url()).searchParams.get("id");
  assert.ok(proposalID);
  await shot("knowledge-proposal");
  assert.equal((await api("/documents/" + target.id)).version, 1);
  await page
    .getByRole("button", { name: "비교 후 변경안 반영", exact: true })
    .click();
  await page
    .getByRole("button", { name: "현재 버전에 변경안 반영", exact: true })
    .click();
  await expect
    .poll(async () => (await api("/documents/" + target.id)).version)
    .toBe(2);
  await expect(
    page.getByText("반영 완료", { exact: false }).first(),
  ).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(
      () =>
        page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      { timeout: 5000 },
    )
    .toBe(true);
  await shot("knowledge-proposal-mobile");
  await page.setViewportSize({ width: 1512, height: 1080 });
  await page.goto(
    base + `/app/knowledge-time?date=2026-06-30&document_id=${source.id}`,
  );
  await page
    .getByRole("button", { name: "유효기간 추가", exact: true })
    .click();
  await page.getByLabel("기간 1 원문 버전", { exact: true }).selectOption("1");
  await page
    .getByLabel("기간 1 시작일 (포함)", { exact: true })
    .fill("2026-01-01");
  await page
    .getByLabel("기간 1 종료일 (제외)", { exact: true })
    .fill("2026-07-01");
  await page
    .getByRole("button", { name: "유효기간 추가", exact: true })
    .click();
  await page.getByLabel("기간 2 원문 버전", { exact: true }).selectOption("2");
  await page
    .getByLabel("기간 2 시작일 (포함)", { exact: true })
    .fill("2026-07-01");
  await page
    .getByLabel("유효기간 변경 이유", { exact: true })
    .fill("7월 운영 기준 변경의 업무 유효일 등록");
  await page
    .getByRole("button", { name: "등록 전 변경 비교", exact: true })
    .click();
  await page
    .getByRole("button", { name: "동의하고 유효기간 등록", exact: true })
    .click();
  await expect
    .poll(async () => (await api(`/documents/${source.id}/validity`)).revision)
    .toBe(1);
  await page
    .getByRole("button", { name: "당시 원문 미리보기", exact: true })
    .click();
  await expect(
    page
      .getByRole("dialog")
      .locator("pre")
      .filter({ hasText: "동시 작업은 2개입니다." }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", { name: "당시 원문 미리보기", exact: true }),
  ).toBeFocused();
  await page.evaluate(() => window.scrollTo(0, 0));
  await shot("knowledge-time");
  await page.getByLabel("업무 기준일", { exact: true }).fill("2026-07-01");
  await page.getByRole("button", { name: "기준일 검색", exact: true }).click();
  await expect(
    page.getByText("기준일 버전 v2 · 현재 v2", { exact: true }),
  ).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await overflow();
  await page.evaluate(() => window.scrollTo(0, 0));
  await shot("knowledge-time-mobile");
  // A failed current-access check clears already shown historical snippets.
  await api(`/documents/${source.id}/validity`, "PUT", {
    revision: 1,
    document_version: 2,
    periods: [],
    reason: "업무 유효일 등록 철회",
    consent: true,
  });
  await expect(
    page.getByRole("button", { name: "당시 원문 미리보기", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByText(
      "현재 권한·문서·유효기간 또는 보호 정책이 변경되었습니다. 다시 검색하세요.",
      { exact: true },
    ),
  ).toBeVisible();
  // Current policy disabling removes previously displayed historical content.
  await api("/admin/evidence-policy", "PUT", {
    enabled: false,
    retention_days: 30,
    version: (await api("/admin/evidence-policy")).version,
  });
  await page.goto(base + "/app/evidence?id=" + evidenceID);
  await expect(
    page.getByRole("heading", { name: "당시 질문과 답변", exact: true }),
  ).toHaveCount(0);
  assert.deepEqual(errors, []);
  console.log(
    "PASS actual AI stream → evidence consent/review/history; package UI/mandatory/estimate/stale export; typed impact/review/preview focus; proposal shared compare→atomic merge; 390px layouts.",
  );
} catch (error) {
  await shot("failure-private");
  throw error;
} finally {
  await context.close();
  await browser.close();
}
