import { expect, test, type Page } from "@playwright/test";
import http from "node:http";
import type { AddressInfo } from "node:net";
import { login } from "./helpers";

/**
 * 방문 추적 스니펫과 CSP 를 실제 브라우저로 확인한다.
 *
 * 스니펫을 붙이는 일의 어려운 쪽은 정책이다. 브라우저가 nonce 없는 스크립트를
 * 조용히 막으면 화면은 멀쩡한데 수집만 안 된다. 그래서 가짜 Momento 수집기를
 * 세워 로더가 실제로 실행되고 수집 요청이 같은 오리진 프록시를 지나 도착하는
 * 것까지 본다. 가짜 수집기는 이 검증 프로세스 안에서 뜨고, 서버가 그곳에 닿을
 * 주소는 E2E_COLLECTOR_HOST 로 준다 — CI 처럼 --network host 면 기본값
 * 127.0.0.1 이고, Docker Desktop 의 브리지 네트워크면 host.docker.internal 이다.
 */

type Hit = { method: string; path: string; cookie: string; body: string };

const TRACKING_KEYS = [
  "tracking.enabled",
  "tracking.provider",
  "tracking.momento_url",
  "tracking.momento_site_id",
  "tracking.momento_proxy",
  "tracking.custom_snippet",
  "tracking.allowed_hosts",
  "tracking.include_admin",
  "tracking.placement",
];

const BASE_POLICY =
  "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'";

const putSettings = async (page: Page, settings: Record<string, string>) => {
  const me = await (await page.request.get("/api/v1/auth/me")).json();
  const response = await page.request.put("/api/v1/settings", {
    headers: { "X-CSRF-Token": me.csrfToken },
    data: { settings },
  });
  return response;
};

// 가짜 수집기: tracker.js 는 data-endpoint 로 이벤트 하나를 보내는 최소 로더다.
const fakeCollector = () => {
  const hits: Hit[] = [];
  const server = http.createServer((request, response) => {
    let body = "";
    request.on("data", (chunk) => (body += chunk));
    request.on("end", () => {
      hits.push({
        method: request.method ?? "",
        path: request.url ?? "",
        cookie: request.headers.cookie ?? "",
        body,
      });
      if (request.url === "/tracker.js") {
        response.writeHead(200, {
          "Content-Type": "application/javascript",
          "Set-Cookie": "momento_probe=1",
        });
        response.end(
          `(function(){var s=document.currentScript;var e=s.dataset.endpoint||"";fetch(e+"/collect/v1/events",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({site_id:s.dataset.siteId,events:[{name:"page_view",url:location.pathname}]})}).then(function(){window.__momentoSent=true;});})();`,
        );
        return;
      }
      response.writeHead(202, { "Content-Type": "application/json" });
      response.end('{"accepted":1}');
    });
  });
  return { server, hits };
};

test.describe("방문 추적", () => {
  const { server, hits } = fakeCollector();
  let collectorURL = "";
  let original: Record<string, string> = {};

  test.beforeAll(async () => {
    await new Promise<void>((resolve) => server.listen(0, "0.0.0.0", resolve));
    const host = process.env.E2E_COLLECTOR_HOST ?? "127.0.0.1";
    collectorURL = `http://${host}:${(server.address() as AddressInfo).port}`;
  });
  test.afterAll(async () => {
    await new Promise<void>((resolve) => server.close(() => resolve()));
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    const data = await (await page.request.get("/api/v1/settings")).json();
    original = Object.fromEntries(
      (data.items as { key: string; value: string }[])
        .filter((item) => TRACKING_KEYS.includes(item.key))
        .map((item) => [item.key, item.value]),
    );
    hits.length = 0;
  });
  test.afterEach(async ({ page }) => {
    // 다른 검증은 추적이 꺼진 서버를 전제한다. 반드시 원래대로 돌린다.
    await putSettings(page, original);
  });

  test("꺼져 있으면 문서와 정책이 원래대로다", async ({ page }) => {
    const response = await page.goto("/");
    expect(response?.headers()["content-security-policy"]).toBe(BASE_POLICY);
    expect(await response?.text()).not.toContain("tracker.js");
    const proxied = await page.request.get("/momento/tracker.js");
    expect(proxied.status()).toBe(404);
  });

  test("Momento 프록시로 켜면 nonce 붙은 로더가 실행되고 수집이 같은 오리진을 지나 도착한다", async ({
    page,
  }) => {
    const saved = await putSettings(page, {
      "tracking.enabled": "true",
      "tracking.provider": "momento",
      "tracking.momento_url": collectorURL,
      "tracking.momento_site_id": "SITE_E2E",
      "tracking.momento_proxy": "true",
      "tracking.include_admin": "false",
    });
    expect(saved.ok(), await saved.text()).toBe(true);

    const blocked: string[] = [];
    page.on("console", (message) => {
      if (message.text().includes("Content Security Policy"))
        blocked.push(message.text());
    });
    const response = await page.goto("/");
    const policy = response?.headers()["content-security-policy"] ?? "";
    const html = (await response?.text()) ?? "";
    const nonce = /'nonce-([^']+)'/.exec(policy)?.[1];
    expect(nonce, policy).toBeTruthy();
    expect(policy).not.toMatch(/script-src[^;]*'unsafe-inline'/);
    expect(policy).toContain("report-uri /api/v1/tracking/csp-report");
    expect(policy).not.toContain(new URL(collectorURL).host);
    expect(html).toContain(
      `<script nonce="${nonce}" async src="/momento/tracker.js" data-site-id="SITE_E2E" data-environment="prd" data-contract-version="1" data-endpoint="/momento"></script>\n</head>`,
    );

    // 로더가 실제로 실행되어 수집 요청을 보냈다 — 정책이 막지 않았다는 증거.
    await page.waitForFunction(
      () => (window as { __momentoSent?: boolean }).__momentoSent === true,
    );
    expect(blocked).toEqual([]);
    const loader = hits.find((hit) => hit.path === "/tracker.js");
    const collect = hits.find((hit) => hit.path === "/collect/v1/events");
    expect(loader?.method).toBe("GET");
    expect(collect?.method).toBe("POST");
    expect(collect?.body).toContain('"site_id":"SITE_E2E"');
    // 세션 쿠키는 수집기에 가지 않고, 수집기의 쿠키는 브라우저에 앉지 않는다.
    expect(loader?.cookie).toBe("");
    expect(collect?.cookie).toBe("");
    const cookies = await page.context().cookies();
    expect(cookies.some((cookie) => cookie.name === "momento_probe")).toBe(
      false,
    );

    // include_admin 이 꺼져 있으면 관리 화면 문서에는 붙지 않는다.
    const admin = await page.goto("/admin/settings");
    expect(admin?.headers()["content-security-policy"]).toBe(BASE_POLICY);
    expect(await admin?.text()).not.toContain("tracker.js");
  });

  test("정책이 막은 출처가 화면에 보이고 한 번 눌러 허용에 더한다", async ({
    page,
  }) => {
    // 문자열을 이어 붙여 만든 주소는 스니펫에서 읽히지 않으므로 정책에 없고,
    // 브라우저가 막은 뒤 report-uri 로 신고한다.
    const saved = await putSettings(page, {
      "tracking.enabled": "true",
      "tracking.provider": "custom",
      "tracking.custom_snippet": `<script>fetch("https://" + "blocked.e2e.invalid/collect").catch(function(){});</script>`,
      "tracking.allowed_hosts": "",
      "tracking.include_admin": "false",
    });
    expect(saved.ok(), await saved.text()).toBe(true);
    await page.goto("/");
    await expect
      .poll(async () => {
        const list = await (
          await page.request.get("/api/v1/settings/tracking/violations")
        ).json();
        return (list.items as { origin: string }[]).map((item) => item.origin);
      })
      .toContain("https://blocked.e2e.invalid");

    await page.goto("/admin/settings");
    await page.getByRole("tab", { name: "방문 추적" }).click();
    const row = page
      .getByRole("table", { name: "정책이 차단한 출처" })
      .getByRole("row")
      .filter({ hasText: "https://blocked.e2e.invalid" });
    await expect(row).toBeVisible();
    await expect(row).toContainText("connect-src");
    await row.getByRole("button", { name: "허용에 추가" }).click();
    await expect(page.getByText(/허용 목록에 더했습니다/)).toBeVisible();
    await expect(page.getByLabel("추가로 허용할 출처")).toHaveValue(
      "https://blocked.e2e.invalid",
    );
    await expect(row.getByText("허용됨")).toBeVisible();

    // 허용한 출처는 다음 문서의 정책에 들어간다.
    const response = await page.goto("/");
    expect(response?.headers()["content-security-policy"]).toContain(
      "connect-src 'self' https://blocked.e2e.invalid",
    );
  });

  test("8KB 를 넘는 스니펫은 저장되지 않는다", async ({ page }) => {
    const rejected = await putSettings(page, {
      "tracking.provider": "custom",
      "tracking.custom_snippet":
        "<script>" + "x".repeat(8 * 1024) + "</script>",
    });
    expect(rejected.status()).toBe(400);
    expect(await rejected.text()).toContain("8192");
  });
});
