import { chromium } from "../../tests/node_modules/playwright/index.mjs";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
const base = process.env.MADI_BASE_URL;
assert.ok(base);
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1512, height: 1080 },
  locale: "ko-KR",
  timezoneId: "Asia/Seoul",
  reducedMotion: "reduce",
});
const page = await context.newPage();
const issues = [],
  external = [];
page.on("pageerror", (e) => issues.push(e.message));
page.on("response", (r) => {
  if (r.status() >= 500) issues.push(`${r.status()} ${r.url()}`);
});
await context.route("**/*", (route) => {
  if (
    !route.request().url().startsWith(base) &&
    !route.request().url().startsWith("data:")
  ) {
    external.push(route.request().url());
    return route.abort();
  }
  return route.continue();
});
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${path}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
async function shot(name) {
  const out = new URL("../../docs/screenshots/", import.meta.url);
  await mkdir(out, { recursive: true });
  await page.screenshot({
    path: new URL(name + ".png", out).pathname,
    animations: "disabled",
  });
}
try {
  await page.goto(base + "/login");
  await page
    .getByRole("button", { name: "디렉터리 계정으로 로그인", exact: true })
    .click();
  await shot("identity-directory-login");
  await page.getByLabel("디렉터리 사용자명", { exact: true }).fill("worker");
  await page
    .getByLabel("디렉터리 비밀번호", { exact: true })
    .fill("directory-secret");
  await page
    .getByRole("button", { name: "디렉터리 로그인", exact: true })
    .click();
  await page.waitForURL("**/app");
  assert.equal((await api("/auth/me")).email, "ldap@example.test");
  console.log("PASS real LDAP TLS bind through Korean login UI");
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const wid = (await api("/workspaces"))[0].id;
  await page.goto(base + "/admin/identity");
  await page
    .getByRole("heading", { name: "인증·디렉터리 관리", exact: true })
    .waitFor();
  await page.getByRole("button", { name: "그룹 매핑", exact: true }).click();
  await page.getByLabel("매핑 1 제공자", { exact: true }).selectOption("scim");
  await page
    .getByLabel("매핑 1 그룹", { exact: true })
    .fill("Browser Directory");
  await page
    .getByLabel("매핑 1 워크스페이스", { exact: true })
    .selectOption(wid);
  await page
    .getByLabel("매핑 1 역할", { exact: true })
    .selectOption("commenter");
  await page
    .getByRole("button", { name: "인증 설정 저장", exact: true })
    .click();
  await page.waitForFunction(async () =>
    (
      await (await fetch("/api/v1/admin/settings")).json()
    ).identity_group_mappings.some(
      (m) => m.group === "Browser Directory" && m.role === "commenter",
    ),
  );
  await shot("identity-group-mapping");
  await page
    .getByRole("button", { name: "LDAP · Active Directory", exact: true })
    .click();
  await page.reload();
  await page.getByLabel("디렉터리 URL", { exact: true }).waitFor();
  assert.ok(
    (
      await page.getByLabel("디렉터리 URL", { exact: true }).inputValue()
    ).startsWith("ldaps://"),
  );
  assert.equal(
    await page.getByLabel("검색 계정 비밀번호", { exact: true }).inputValue(),
    "",
  );
  await shot("identity-ldap");
  await page.getByRole("button", { name: "SAML SSO", exact: true }).click();
  await shot("identity-saml");
  console.log(
    "PASS group selects save correctly, tab refresh persists, stored secret is redacted",
  );
  await page.getByRole("button", { name: "SCIM 동기화", exact: true }).click();
  await page
    .getByLabel("SCIM 사용자·그룹 동기화 사용", { exact: true })
    .check();
  await page
    .getByLabel("서비스 계정 키에 SCIM 관리 권한 발급 허용", { exact: true })
    .check();
  await page
    .getByRole("button", { name: "인증 설정 저장", exact: true })
    .click();
  await page.waitForFunction(
    async () =>
      (await (await fetch("/api/v1/admin/settings")).json()).scim_enabled,
  );
  const service = await api("/admin/users", "POST", {
    email: "browser-scim@example.test",
    name: "브라우저 SCIM 서비스",
    role: "viewer",
    kind: "service",
  });
  await api("/workspaces/" + wid + "/members", "PUT", {
    email: service.email,
    role: "editor",
  });
  await page.getByRole("button", { name: "새로고침", exact: true }).click();
  await page
    .getByLabel("서비스 계정", { exact: true })
    .selectOption(service.id);
  await page.getByRole("button", { name: "키 발급", exact: true }).click();
  await page.getByLabel("키 워크스페이스", { exact: true }).selectOption(wid);
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "저장", exact: true })
    .click();
  await page.getByLabel("API 비밀 키", { exact: true }).waitFor();
  const first = await page
    .getByLabel("API 비밀 키", { exact: true })
    .inputValue();
  assert.ok(first.startsWith("madi_"));
  await page
    .getByRole("button", { name: "안전하게 보관했어요", exact: true })
    .click();
  const provision = await context.request.post(base + "/api/v1/scim/v2/Users", {
    headers: { Authorization: "Bearer " + first },
    data: {
      schemas: ["urn:ietf:params:scim:schemas:core:2.0:User"],
      userName: "browser-provisioned",
      emails: [{ value: "browser-provisioned@example.test" }],
      active: true,
    },
  });
  assert.equal(provision.status(), 201, await provision.text());
  page.on("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "회전", exact: true }).click();
  await page.getByLabel("API 비밀 키", { exact: true }).waitFor();
  const second = await page
    .getByLabel("API 비밀 키", { exact: true })
    .inputValue();
  assert.notEqual(first, second);
  const stale = await context.request.get(base + "/api/v1/scim/v2/Users", {
    headers: { Authorization: "Bearer " + first },
  });
  assert.equal(stale.status(), 401);
  const fresh = await context.request.get(base + "/api/v1/scim/v2/Users", {
    headers: { Authorization: "Bearer " + second },
  });
  assert.equal(fresh.status(), 200);
  await page
    .getByRole("button", { name: "안전하게 보관했어요", exact: true })
    .click();
  console.log(
    "PASS admin-only SCIM key issuance and rotation revoke old secret immediately",
  );
  await shot("identity-scim");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.reload();
  await page
    .getByRole("heading", { name: "인증·디렉터리 관리", exact: true })
    .waitFor();
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth + 1,
    ),
    "mobile page overflows",
  );
  await shot("mobile-identity");
  assert.deepEqual(issues, []);
  assert.deepEqual(external, []);
} catch (error) {
  await page
    .screenshot({ path: "/tmp/madi-identity-failure.png", fullPage: true })
    .catch(() => {});
  console.error("ISSUES", issues);
  console.error(await page.locator("body").innerText());
  throw error;
} finally {
  await browser.close();
}
