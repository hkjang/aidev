import assert from "node:assert/strict";
import path from "node:path";
import { mkdir } from "node:fs/promises";
import { chromium, expect } from "playwright/test";
const base = process.env.MADI_BASE_URL,
  out = process.env.MADI_SCREENSHOT_DIR || "test-results/structured-drafts";
const browser = await chromium.launch(),
  context = await browser.newContext({
    viewport: { width: 1440, height: 1050 },
    locale: "ko-KR",
    reducedMotion: "reduce",
  }),
  page = await context.newPage(),
  errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (request) => {
  const url = request.url();
  if (/^https?:/.test(url) && new URL(url).origin !== new URL(base).origin)
    external.push(url);
});
async function api(url, method = "GET", data, status = 200, ctx = context) {
  const response = await ctx.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.equal(response.status(), status, await response.text());
  return response.json();
}
async function shot(name) {
  await mkdir(out, { recursive: true });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(out, name + ".png"),
    animations: "disabled",
  });
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const workspace = await api("/workspaces", "POST", {
    name: "문서 근거로 데이터 정리",
  });
  const props = [
    { id: "amount", name: "수량", type: "number" },
    { id: "state", name: "상태", type: "select", options: ["운영", "검토"] },
    {
      id: "tags",
      name: "분류",
      type: "multi_select",
      options: ["장비", "운영"],
    },
    { id: "hidden", name: "내부 식별자", type: "text" },
  ];
  const db = await api("/databases", "POST", {
    workspace_id: workspace.id,
    name: "인프라 점검 현황",
    properties: props,
  });
  const selected =
      "GPU 수량은 2개입니다. 상태는 운영입니다. 분류는 장비입니다.",
    markdown =
      "# GPU 점검\r\n\r\n선택하지 않은 앞쪽 설명\r\n" +
      selected +
      "\r\n선택하지 않은 뒤쪽 설명\r\n";
  const doc = await api("/documents", "POST", {
    workspace_id: workspace.id,
    title: "GPU 용량 현황",
    markdown,
    visibility: "private",
  });
  await page.goto(base + "/app/structured-drafts");
  await page
    .getByLabel("워크스페이스 선택", { exact: true })
    .selectOption(workspace.id);
  await page.goto(base + "/app/structured-drafts");
  await page
    .getByLabel("구조화 원문 문서", { exact: true })
    .selectOption(doc.id);
  await page
    .getByLabel("값을 담을 데이터베이스", { exact: true })
    .selectOption(db.id);
  const source = page.getByLabel("구조화할 저장 원문", { exact: true });
  await expect(source).toBeVisible();
  await source.evaluate((node, selection) => {
    node.focus();
    const start = node.value.indexOf(selection);
    if (start < 0) throw Error("fixture selection absent");
    node.setSelectionRange(start, start + selection.length);
  }, selected);
  await page
    .getByRole("button", { name: "선택 구간 사용", exact: true })
    .click();
  await expect(page.locator(".structured-selection pre")).toHaveText(selected);
  await page.getByText("키보드로 줄 범위 지정", { exact: true }).click();
  await page.getByLabel("구조화 시작 줄", { exact: true }).fill("4");
  await page.getByLabel("구조화 끝 줄", { exact: true }).fill("999999");
  const nextContext = page.waitForResponse(
    (response) =>
      response.url().includes(`/documents/${doc.id}/structured-context`) &&
      response.ok(),
  );
  await page
    .getByRole("button", { name: "줄 범위를 선택 구간으로 사용", exact: true })
    .click();
  await expect(page.getByText(/시작 줄과 끝 줄을 1~/)).toBeVisible();
  await nextContext;
  await expect(page.getByText(/시작 줄과 끝 줄을 1~/)).toBeVisible();
  await expect(page.locator(".structured-selection")).toHaveCount(0);
  await page.getByLabel("구조화 끝 줄", { exact: true }).fill("4");
  await page
    .getByRole("button", { name: "줄 범위를 선택 구간으로 사용", exact: true })
    .click();
  await expect(page.locator(".structured-selection pre")).toHaveText(selected);
  await page.getByText("키보드로 줄 범위 지정", { exact: true }).click();
  for (const name of ["수량", "상태", "분류"])
    await page
      .locator(".structured-properties")
      .getByRole("checkbox", { name: new RegExp(name) })
      .check();
  await expect(
    page.getByRole("button", { name: "구조화 제안 생성", exact: true }),
  ).toBeDisabled();
  await page.getByRole("checkbox", { name: /위 구간과 선택 속성/ }).check();
  await page.evaluate(() => window.scrollTo(0, 0));
  await shot("structured-workspace");
  await page.locator(".structured-selection").scrollIntoViewIfNeeded();
  await shot("structured-selection");
  await page
    .getByRole("button", { name: "구조화 제안 생성", exact: true })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "3. 완료된 제안을 개인 검토함에 보관",
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.locator(".structured-proposal-summary article"),
  ).toHaveCount(3);
  await page
    .getByRole("heading", {
      name: "3. 완료된 제안을 개인 검토함에 보관",
      exact: true,
    })
    .scrollIntoViewIfNeeded();
  await shot("structured-ai-proposal");
  let rows = await api(`/databases/${db.id}/rows`);
  assert.equal(rows.length, 0);
  await page
    .getByRole("checkbox", { name: /값과 근거 구간을 내 검토함/ })
    .check();
  await page
    .getByRole("button", { name: "개인 검토함에 보관", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "개인 구조화 초안 검토", exact: true }),
  ).toBeVisible();
  const draftID = new URL(page.url()).searchParams.get("id");
  assert.ok(draftID);
  await shot("structured-private-draft");
  await page
    .getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    })
    .click();
  await expect(
    page.getByRole("dialog", {
      name: "대상 데이터베이스에 새 행 공유",
      exact: true,
    }),
  ).toHaveCount(0);
  await expect(page.getByLabel("상태 값", { exact: true })).toHaveValue(
    "미등록",
  );
  await expect(page.getByText(/다중 선택 값을 확인/)).toBeVisible();
  await page.getByLabel("상태 값", { exact: true }).selectOption("검토");
  await page.getByLabel("상태 값", { exact: true }).selectOption("운영");
  await page
    .getByRole("group", { name: "분류 값", exact: true })
    .getByRole("checkbox", { name: "장비", exact: true })
    .check();
  await page.getByLabel("수량 값", { exact: true }).fill("3");
  await page.getByLabel("수량 근거 인용", { exact: true }).fill("없는 근거");
  await page.getByLabel("수량 인용 시작 바이트", { exact: true }).fill("");
  await page
    .getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    })
    .click();
  await expect(page.getByLabel("수량 근거 인용", { exact: true })).toHaveValue(
    "없는 근거",
  );
  await expect(
    page.getByRole("dialog", {
      name: "대상 데이터베이스에 새 행 공유",
      exact: true,
    }),
  ).toHaveCount(0);
  await page.getByLabel("수량 값", { exact: true }).fill("2");
  await page
    .getByLabel("수량 근거 인용", { exact: true })
    .fill("GPU 수량은 2개입니다.");
  await page
    .getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    })
    .click();
  let dialog = page.getByRole("dialog", {
    name: "대상 데이터베이스에 새 행 공유",
    exact: true,
  });
  await expect(dialog).toBeVisible();
  await expect(
    dialog.getByRole("button", {
      name: "확인한 값으로 새 행 생성",
      exact: true,
    }),
  ).toBeDisabled();
  await expect(dialog.getByText(/사람이 수정·확인한 값/).first()).toBeVisible();
  await shot("structured-sharing-preview");
  await dialog
    .getByRole("checkbox", { name: /근거와 값을 확인했으며/ })
    .check();
  await dialog
    .getByRole("button", { name: "확인한 값으로 새 행 생성", exact: true })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "행 생성 완료 · 보관한 근거",
      exact: true,
    }),
  ).toBeVisible();
  await shot("structured-committed");
  rows = await api(`/databases/${db.id}/rows`);
  assert.equal(rows.length, 1);
  assert.equal(rows[0].values.amount, 2);
  assert.equal(rows[0].values.state, "운영");
  assert.deepEqual(rows[0].values.tags, ["장비"]);
  const unchanged = await api(`/documents/${doc.id}`);
  assert.equal(unchanged.markdown, markdown);
  assert.equal(unchanged.version, 1);
  await page.reload();
  await expect(
    page.getByRole("heading", {
      name: "행 생성 완료 · 보관한 근거",
      exact: true,
    }),
  ).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() =>
      page.evaluate(() => {
        const main = document.querySelector(".main");
        return main ? getComputedStyle(main).marginLeft : "";
      }),
    )
    .toBe("0px");
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    )
    .toBe(true);
  assert.ok(
    await page
      .locator(".structured-page .page-heading p")
      .evaluate((node) => parseFloat(getComputedStyle(node).fontSize) >= 16),
  );
  await shot("structured-mobile");
  await api(`/documents/${doc.id}`, "PUT", {
    version: 1,
    markdown: markdown + "\r\n다음 확인 기록\r\n",
  });
  await expect(page.getByText(/아래 구간은 과거 제안의 근거/)).toBeVisible();
  await shot("structured-source-changed");
  await page
    .getByRole("button", { name: "근거 사본 삭제", exact: true })
    .click();
  dialog = page.getByRole("dialog", {
    name: "개인 근거 사본 삭제",
    exact: true,
  });
  await dialog
    .getByRole("button", { name: "근거 사본 삭제 확인", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "내 검토함", exact: true }),
  ).toBeVisible();
  assert.equal((await api(`/databases/${db.id}/rows`)).length, 1);
  await api(`/knowledge/structured-drafts/${draftID}`, "GET", undefined, 404);
  // A second actual provider proposal isolates live review invalidation checks.
  const meta = await api(
      `/documents/${doc.id}/structured-context?database_id=${db.id}`,
    ),
    offset = Buffer.byteLength(
      meta.markdown.slice(0, meta.markdown.indexOf(selected)),
    ),
    streamed = await context.request.post(
      `${base}/api/v1/documents/${doc.id}/structured-draft`,
      {
        headers: { "X-Madi-Request": "1" },
        data: {
          database_id: db.id,
          expected_version: meta.version,
          start_byte: offset,
          end_byte: offset + Buffer.byteLength(selected),
          selected_text: selected,
          property_ids: ["amount", "state", "tags"],
          provider_fingerprint: meta.provider.fingerprint,
          schema_hash: meta.schema_hash,
          destination_hash: meta.destination_hash,
          consent: true,
        },
      },
    );
  assert.equal(streamed.status(), 200);
  const events = (await streamed.text())
      .split("\n")
      .filter((line) => line.startsWith("data: {"))
      .map((line) => JSON.parse(line.slice(6))),
    proposal = events.find((event) => event.proposal)?.proposal;
  assert.ok(proposal?.draft_ticket);
  const second = await api(
    "/knowledge/structured-drafts",
    "POST",
    { ticket: proposal.draft_ticket, consent: true },
    201,
  );
  await page.goto(`${base}/app/structured-drafts?id=${second.id}`);
  await page.getByLabel("상태 값", { exact: true }).selectOption("운영");
  await page
    .getByRole("group", { name: "분류 값", exact: true })
    .getByRole("checkbox", { name: "장비", exact: true })
    .check();
  await page
    .getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    })
    .click();
  dialog = page.getByRole("dialog", {
    name: "대상 데이터베이스에 새 행 공유",
    exact: true,
  });
  await expect(dialog).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    )
    .toBe(true);
  await dialog
    .getByRole("checkbox", { name: /근거와 값을 확인했으며/ })
    .scrollIntoViewIfNeeded();
  await shot("structured-mobile-sharing");
  await api(`/databases/${db.id}`, "PUT", {
    properties: props.map((p) =>
      p.id === "state" ? { ...p, name: "운영 상태" } : p,
    ),
  });
  await expect(dialog).toHaveCount(0);
  await expect(page.getByText(/아래 구간은 과거 제안의 근거/)).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    }),
  ).toBeDisabled();
  await shot("structured-schema-changed");
  await api(`/databases/${db.id}`, "PUT", { properties: props });
  await expect(
    page.getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    }),
  ).toBeEnabled();
  await page
    .getByRole("button", {
      name: "선택한 값과 공유 대상 미리보기",
      exact: true,
    })
    .click();
  await expect(dialog).toBeVisible();
  await expect(
    dialog.getByRole("checkbox", { name: /근거와 값을 확인했으며/ }),
  ).not.toBeChecked();
  await api("/admin/users", "POST", {
    email: "structured-owner@example.test",
    name: "새 문서 소유자",
    role: "editor",
    password: "Structured-Owner-Password-2026!",
  });
  await api(`/workspaces/${workspace.id}/members`, "PUT", {
    email: "structured-owner@example.test",
    role: "editor",
  });
  const owner = (await api("/admin/users")).find(
      (u) => u.email === "structured-owner@example.test",
    ),
    beforeTransfer = await api(`/documents/${doc.id}`);
  assert.ok(owner);
  await api(`/documents/${doc.id}/knowledge`, "PUT", {
    version: beforeTransfer.version,
    owner_id: owner.id,
  });
  const other = await browser.newContext();
  try {
    await api(
      "/auth/login",
      "POST",
      {
        email: "structured-owner@example.test",
        password: "Structured-Owner-Password-2026!",
      },
      200,
      other,
    );
    const current = await api(
      `/documents/${doc.id}`,
      "GET",
      undefined,
      200,
      other,
    );
    await api(
      `/documents/${doc.id}`,
      "PUT",
      { version: current.version, visibility: "private" },
      200,
      other,
    );
    await expect(dialog).toHaveCount(0);
    await expect(
      page.getByLabel("수량 근거 인용", { exact: true }),
    ).toHaveCount(0);
    await expect(
      page.locator(".structured-review .recovery-notice"),
    ).toBeVisible();
    await shot("structured-access-removed");
    await api(
      `/knowledge/structured-drafts/${second.id}`,
      "GET",
      undefined,
      404,
    );
    assert.equal((await api(`/databases/${db.id}/rows`)).length, 1);
  } finally {
    await other.close();
  }
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  console.log(
    JSON.stringify({
      passed: true,
      source_preserved: true,
      explicit_shared_rows: 1,
      js_errors: errors.length,
      external_requests: external.length,
      current_acl_removal: true,
      live_schema_invalidation: true,
    }),
  );
} catch (error) {
  await shot("failure").catch(() => {});
  throw error;
} finally {
  await context.close();
  await browser.close();
}
