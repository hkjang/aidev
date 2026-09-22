import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import http from "node:http";
import { execFileSync } from "node:child_process";
const sourceDSN = process.env.MADI_SOURCE_TEST_ADMIN_DSN;
if (!sourceDSN)
  throw Error("Explicit disposable MADI_SOURCE_TEST_ADMIN_DSN is required");
const connection = new URL(sourceDSN);
assert.ok(
  connection.pathname.endsWith("_test"),
  "Only disposable *_test databases are accepted",
);
const stamp = Date.now(),
  schema = "madi_ui_ai_" + stamp,
  role = "madi_ui_ai_reader_" + stamp;
execFileSync(
  "psql",
  [
    sourceDSN,
    "-X",
    "-v",
    "ON_ERROR_STOP=1",
    "-q",
    "-c",
    `CREATE SCHEMA "${schema}";CREATE TABLE "${schema}".orders(name text,amount integer,status text);INSERT INTO "${schema}".orders VALUES('GPU 운영',300,'운영'),('문서 백업',100,'완료'),('권한 검토',200,'진행 중');CREATE ROLE "${role}" LOGIN;GRANT USAGE ON SCHEMA "${schema}" TO "${role}";GRANT SELECT ON "${schema}".orders TO "${role}";`,
  ],
  { stdio: ["ignore", "ignore", "pipe"] },
);
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080",
  output = path.resolve(
    process.env.MADI_SCREENSHOT_DIR || "test-results/text2sql",
  );
await mkdir(output, { recursive: true });
const model = http.createServer(async (req, res) => {
  let body = "";
  for await (const chunk of req) body += chunk;
  const input = JSON.parse(body);
  assert.equal(input.stream, true);
  assert.ok(!body.includes(role));
  const value = {
    name: "금액 높은 운영 업무",
    explanation:
      "허용된 orders 테이블에서 금액이 높은 두 업무의 이름, 금액, 상태를 조회하는 계획입니다. 아직 실제 데이터는 조회하지 않았습니다.",
    plan: {
      schema,
      table: "orders",
      columns: ["name", "amount", "status"],
      filters: [],
      order: [{ column: "amount", direction: "desc" }],
      limit: 2,
    },
  };
  res.writeHead(200, { "Content-Type": "text/event-stream" });
  const content = JSON.stringify(value);
  for (let i = 0; i < content.length; i += 60) {
    res.write(
      "data: " +
        JSON.stringify({
          choices: [{ delta: { content: content.slice(i, i + 60) } }],
        }) +
        "\n\n",
    );
    await new Promise((resolve) => setTimeout(resolve, 40));
  }
  res.end("data: [DONE]\n\n");
});
await new Promise((resolve) => model.listen(0, "127.0.0.1", resolve));
const browser = await chromium.launch({ headless: true }),
  context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
    locale: "ko-KR",
  }),
  page = await context.newPage(),
  errors = [];
let workspace, originalConnectorPolicy;
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`);
});
async function api(endpoint, method = "GET", data, client = context) {
  const response = await client.request.fetch(base + "/api/v1" + endpoint, {
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
    fullPage: !(await page.getByRole('dialog').count()),
    animations: 'disabled',
    style: '.toast{visibility:hidden!important}',
  });
}
try {
  await page.goto(base + "/login");
  await api("/auth/login", "POST", {
    email: process.env.MADI_TEST_EMAIL || "admin@example.test",
    password: process.env.MADI_TEST_PASSWORD || "Browser-Test-Password-2026!",
  });
  workspace = await api("/workspaces", "POST", {
    name: "Text2SQL UI 검증 " + stamp,
  });
  await page.evaluate(
    (id) => localStorage.setItem("madi.workspace", id),
    workspace.id,
  );
  const email = `sql-ai-ui-${stamp}@example.test`,
    service = await api("/admin/users", "POST", {
      email,
      name: "AI 데이터 읽기 계정",
      role: "editor",
      kind: "service",
    });
  await api(`/workspaces/${workspace.id}/members`, "PUT", {
    email,
    role: "editor",
  });
  const policy = await api("/admin/connectors/settings");
  originalConnectorPolicy = policy;
  await api("/admin/connectors/settings", "PUT", {
    enabled: true,
    allowed_hosts: [
      ...new Set([...(policy.allowed_hosts || []), connection.hostname]),
    ],
  });
  const source = await api("/data-sources", "POST", {
    workspace_id: workspace.id,
    name: "운영 비용 데이터",
    kind: "postgres",
    space_id: "",
    service_account_id: service.id,
    enabled: true,
    credentials: { username: role, password: "" },
    config: {
      host: connection.hostname,
      port: Number(connection.port),
      database: connection.pathname.slice(1),
      tables: [schema + ".orders"],
      timeout_seconds: 15,
      max_rows: 100,
      allow_plaintext: true,
      acknowledge_readonly: true,
      acknowledge_acl: true,
      allow_ai: true,
    },
  });
  await api(`/data-sources/${source.id}/inspect`, "POST", {});
  const wsSettings = await api(`/workspaces/${workspace.id}/settings`);
  await api(`/workspaces/${workspace.id}/settings`, "PUT", {
    version: wsSettings.version,
    data: {
      ai_enabled: true,
      ai_base_url: `http://127.0.0.1:${model.address().port}/v1`,
      ai_model: "UI 계획 검증",
      ai_max_tokens: 262144,
    },
  });
  const reviewerEmail = `sql-ai-reviewer-${stamp}@example.test`,
    reviewer = await api("/admin/users", "POST", {
      email: reviewerEmail,
      name: "데이터 조회 검토자",
      role: "viewer",
      password: "SQL-UI-review-password-2026!",
    });
  await api(`/workspaces/${workspace.id}/members`, "PUT", {
    email: reviewerEmail,
    role: "viewer",
  });
  await api("/admin/approval/policies", "POST", {
    workspace_id: workspace.id,
    resource_kind: "sql_query_plan",
    name: "UI 조회 계획 확인",
    enabled: true,
    stages: [
      {
        name: "테이블과 조건 확인",
        mode: "all",
        gates: [{ name: "지정 검토자", kind: "user", id: reviewer.id }],
      },
    ],
  });
  const tableURL =
    base + `/app/data-sources?source=${source.id}&table=${schema}.orders`;
  await page.goto(tableURL);
  await page
    .getByLabel("AI 데이터 조회 질문", { exact: true })
    .fill("금액이 높은 업무 두 건을 이름과 상태와 함께 보고 싶어요.");
  await page
    .getByRole("button", { name: "조회 계획 제안받기", exact: true })
    .click();
  await page.getByRole("dialog").waitFor();
  await page
    .getByText("이 내용은 실제 데이터 조회 결과가 아닙니다.", { exact: false })
    .waitFor();
  await shot("text2sql-plan");
  const proposalURL = page.url();
  await page.reload();
  await page.getByRole("dialog").waitFor();
  assert.equal(page.url(), proposalURL);
  await page
    .getByLabel("SQL 계획 확인 (CREATE QUERY 입력)", { exact: true })
    .fill("CREATE QUERY");
  await page.setViewportSize({ width: 390, height: 844 });
  await shot("text2sql-plan-mobile");
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    "plan mobile overflow",
  );
  await page
    .getByRole("button", { name: /^확인한 (조회 등록|계획 검토 요청)$/ })
    .click();
  await page
    .getByLabel("SQL 계획 확인 (CREATE QUERY 입력)", { exact: true })
    .waitFor({ state: "hidden" });
  let proposals = await api(`/data-sources/${source.id}/ai/proposals`);
  const proposal = proposals.proposals[0];
  if (proposal.status === "review") {
    const rc = await browser.newContext();
    await api(
      "/auth/login",
      "POST",
      { email: reviewerEmail, password: "SQL-UI-review-password-2026!" },
      rc,
    );
    const request = await api(
      `/approvals/requests/${proposal.approval_id}`,
      "GET",
      undefined,
      rc,
    );
    await api(
      `/approvals/requests/${proposal.approval_id}/decisions`,
      "POST",
      {
        action: "approve",
        request_version: request.version,
        gate_index: 0,
        comment: "화면에서 조회 범위 확인",
      },
      rc,
    );
    await rc.close();
  }
  await page.goto(tableURL);
  await page.getByRole("button", { name: "조회 실행", exact: true }).waitFor();
  await shot("text2sql-registered-mobile");
  await page.getByRole("button", { name: "조회 실행", exact: true }).click();
  await page.getByRole("cell", { name: "GPU 운영", exact: true }).waitFor();
  await shot("text2sql-result-mobile");
  await page.setViewportSize({ width: 1440, height: 1000 });
  await shot("text2sql-result");
  assert.deepEqual(errors, []);
  console.log(
    JSON.stringify({
      ok: true,
      workspace_id: workspace.id,
      source_id: source.id,
      source_schema: schema,
      readonly_role: role,
      screenshots: output,
    }),
  );
} finally {
  if (workspace) {
    try {
      const v = await api(`/workspaces/${workspace.id}/settings`);
      await api(`/workspaces/${workspace.id}/settings`, "PUT", {
        version: v.version,
        data: { ai_enabled: false },
      });
    } catch {}
  }
  try {
    if (originalConnectorPolicy) {
      await api("/admin/connectors/settings", "PUT", {
        enabled: originalConnectorPolicy.enabled,
        allowed_hosts: originalConnectorPolicy.allowed_hosts || [],
      });
    }
  } finally {
    await browser.close();
    await new Promise((resolve) => model.close(resolve));
  }
}
