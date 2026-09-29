import {
  expect,
  test,
  type APIRequestContext,
  type Page,
} from "@playwright/test";
import { createSign, generateKeyPairSync } from "node:crypto";
import http from "node:http";
import type { AddressInfo } from "node:net";
import { login } from "./helpers";

/**
 * MCP 를 Keycloak 액세스 토큰으로 여는 길을 실서버로 확인한다.
 *
 * 단위 테스트는 검증기(서명·발급자·만료·대상)를 보고, 여기서는 나머지 —
 * 실제 라우팅과 데이터베이스를 지나는 계정 조회 — 를 본다. 가짜 Keycloak 을
 * 이 프로세스 안에 세워(discovery·JWKS·authorize·token) 웹 SSO 로그인으로
 * 계정을 등록한 뒤, 같은 사람의 액세스 토큰이 /mcp 를 여는지, 등록 전·비활성·
 * 다른 앱용 토큰은 거절되는지, REST 는 여전히 받지 않는지 본다. 서버가 가짜
 * IdP 에 닿을 주소는 tracking.spec 과 같이 E2E_COLLECTOR_HOST 로 준다(기본
 * 127.0.0.1). 브리지 네트워크에서 host.docker.internal 을 주면(컨테이너에
 * --add-host=host.docker.internal:host-gateway 필요) 브라우저는 그 이름을 모르므로
 * Chromium 의 host-resolver-rules 로 같은 이름을 127.0.0.1 에 붙인다 — 웹 SSO
 * 로그인이 issuer 와 같은 주소의 로그인 화면으로 가야 하기 때문이다.
 */

const IDP_HOST = process.env.E2E_COLLECTOR_HOST ?? "127.0.0.1";

const SSO_USER = "sso-mcp-user";
const WEB_CLIENT = "seaton-web";
const MCP_CLIENT = "claude-mcp";

const SETTING_KEYS = [
  "oidc.enabled",
  "oidc.issuer_url",
  "oidc.client_id",
  "oidc.auto_login",
  "mcp.oauth.enabled",
  "mcp.oauth.resource",
  "mcp.oauth.audience",
  "mcp.oauth.scopes",
];

const base64url = (value: Buffer | string) =>
  Buffer.from(value).toString("base64url");

// 가짜 Keycloak: 실제 RSA 키로 서명하고, 웹 로그인의 authorize/token 도 흉내낸다.
const fakeKeycloak = () => {
  const { privateKey, publicKey } = generateKeyPairSync("rsa", {
    modulusLength: 2048,
  });
  const jwk = publicKey.export({ format: "jwk" });
  let issuer = "";
  let lastNonce = "";
  const sign = (claims: Record<string, unknown>) => {
    const header = base64url(
      JSON.stringify({ alg: "RS256", typ: "JWT", kid: "e2e" }),
    );
    const body = base64url(JSON.stringify(claims));
    const signer = createSign("RSA-SHA256");
    signer.update(`${header}.${body}`);
    return `${header}.${body}.${signer.sign(privateKey).toString("base64url")}`;
  };
  // Keycloak 26 의 액세스 토큰 모양: typ=Bearer, aud=["account"], 클라이언트는 azp.
  const accessToken = (overrides: Record<string, unknown> = {}) => {
    const now = Math.floor(Date.now() / 1000);
    return sign({
      iss: issuer,
      sub: "e2e-subject-1",
      aud: ["account"],
      azp: MCP_CLIENT,
      typ: "Bearer",
      iat: now,
      exp: now + 300,
      preferred_username: SSO_USER,
      email: `${SSO_USER}@example.com`,
      scope: "openid profile email",
      ...overrides,
    });
  };
  const server = http.createServer((request, response) => {
    const url = new URL(request.url ?? "/", issuer);
    const json = (status: number, body: unknown) => {
      response.writeHead(status, { "Content-Type": "application/json" });
      response.end(JSON.stringify(body));
    };
    if (url.pathname === "/.well-known/openid-configuration") {
      return json(200, {
        issuer,
        jwks_uri: `${issuer}/jwks`,
        authorization_endpoint: `${issuer}/auth`,
        token_endpoint: `${issuer}/token`,
        id_token_signing_alg_values_supported: ["RS256"],
      });
    }
    if (url.pathname === "/jwks") {
      return json(200, {
        keys: [{ ...jwk, kid: "e2e", use: "sig", alg: "RS256" }],
      });
    }
    if (url.pathname === "/auth") {
      // 화면 없이 곧바로 인가 코드를 돌려준다 — 로그인은 이 검증의 관심사가 아니다.
      lastNonce = url.searchParams.get("nonce") ?? "";
      const back = new URL(url.searchParams.get("redirect_uri") ?? "");
      back.searchParams.set("code", "e2e-code");
      back.searchParams.set("state", url.searchParams.get("state") ?? "");
      response.writeHead(302, { Location: back.toString() });
      return response.end();
    }
    if (url.pathname === "/token") {
      const now = Math.floor(Date.now() / 1000);
      return json(200, {
        token_type: "Bearer",
        access_token: accessToken(),
        id_token: sign({
          iss: issuer,
          sub: "e2e-subject-1",
          aud: WEB_CLIENT,
          exp: now + 300,
          iat: now,
          nonce: lastNonce,
          preferred_username: SSO_USER,
          name: "SSO MCP 검증 사용자",
          email: `${SSO_USER}@example.com`,
        }),
      });
    }
    json(404, { error: "not_found" });
  });
  return {
    server,
    accessToken,
    setIssuer: (value: string) => (issuer = value),
  };
};

const putSettings = async (page: Page, settings: Record<string, string>) => {
  const me = await (await page.request.get("/api/v1/auth/me")).json();
  return page.request.put("/api/v1/settings", {
    headers: { "X-CSRF-Token": me.csrfToken },
    data: { settings },
  });
};

const mcp = (
  request: APIRequestContext,
  token: string,
  method = "tools/list",
  params?: unknown,
) =>
  request.post("/mcp", {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    data: { jsonrpc: "2.0", id: 1, method, params },
  });

// 파일 최상위여야 한다 — describe 안의 launchOptions 는 워커를 새로 띄운다.
test.use({
  launchOptions:
    IDP_HOST === "127.0.0.1"
      ? {}
      : { args: [`--host-resolver-rules=MAP ${IDP_HOST} 127.0.0.1`] },
});

test.describe("MCP SSO(OAuth) 인증", () => {
  const idp = fakeKeycloak();
  let original: Record<string, string> = {};
  let resource = "";
  let issuer = "";

  test.beforeAll(async () => {
    await new Promise<void>((resolve) =>
      idp.server.listen(0, "0.0.0.0", resolve),
    );
    issuer = `http://${IDP_HOST}:${(idp.server.address() as AddressInfo).port}`;
    idp.setIssuer(issuer);
  });
  test.afterAll(async () => {
    await new Promise<void>((resolve) => idp.server.close(() => resolve()));
  });

  test.beforeEach(async ({ page, baseURL }) => {
    await login(page);
    resource = `${baseURL}/mcp`;
    const data = await (await page.request.get("/api/v1/settings")).json();
    original = Object.fromEntries(
      (data.items as { key: string; value: string }[])
        .filter((item) => SETTING_KEYS.includes(item.key))
        .map((item) => [item.key, item.value]),
    );
  });
  test.afterEach(async ({ page }) => {
    // 다른 검증은 SSO 가 꺼진 서버를 전제한다. 반드시 원래대로 돌린다.
    await putSettings(page, original);
  });

  test("켜기 전에는 메타데이터가 404 이고 토큰은 키 전용 때와 같이 거절된다", async ({
    browser,
  }) => {
    // 세션 쿠키가 없는 컨텍스트로 — MCP 클라이언트가 그렇다.
    const anonymous = await browser.newContext();
    const metadata = await anonymous.request.get(
      "/.well-known/oauth-protected-resource/mcp",
    );
    expect(metadata.status()).toBe(404);
    const refused = await mcp(
      anonymous.request,
      idp.accessToken({ aud: [resource] }),
    );
    expect(refused.status()).toBe(401);
    expect(refused.headers()["www-authenticate"]).toBeUndefined();
    expect((await refused.json()).error.code).toBe("authentication_required");
    await anonymous.close();
  });

  test("켜려면 Issuer 가 있어야 하고 범위에 mcp 가 있어야 한다", async ({
    page,
  }) => {
    const noIssuer = await putSettings(page, {
      "oidc.issuer_url": "",
      "mcp.oauth.enabled": "true",
    });
    expect(noIssuer.status()).toBe(400);
    expect((await noIssuer.json()).error.message).toContain("Issuer");
    const noMCP = await putSettings(page, {
      "oidc.issuer_url": issuer,
      "mcp.oauth.enabled": "true",
      "mcp.oauth.scopes": "read",
    });
    expect(noMCP.status()).toBe(400);
    expect((await noMCP.json()).error.message).toContain("mcp");
    const badResource = await putSettings(page, {
      "mcp.oauth.resource": "seaton.intra/mcp",
    });
    expect(badResource.status()).toBe(400);
    // 리소스 식별자 없이는 켤 수 없다 — 요청 주소로 대신 만들면 Host 헤더가
    // 허용 대상을 정하게 된다.
    const noResource = await putSettings(page, {
      "oidc.issuer_url": issuer,
      "mcp.oauth.enabled": "true",
      "mcp.oauth.scopes": "read mcp",
      "mcp.oauth.resource": "",
    });
    expect(noResource.status()).toBe(400);
    expect((await noResource.json()).error.message).toContain("리소스 식별자");
  });

  test("Keycloak 토큰으로 /mcp 가 열리고, 등록 전·다른 앱·비활성·REST 는 거절된다", async ({
    page,
    browser,
    baseURL,
  }) => {
    // 이전 실행이 남긴 계정이 있으면 되살린다 — SSO 로그인은 active 를 건드리지 않는다.
    const csrf = (await (await page.request.get("/api/v1/auth/me")).json())
      .csrfToken as string;
    const findUser = async () => {
      const users = await (await page.request.get("/api/v1/users")).json();
      return (
        users.items as { id: string; username: string; active: boolean }[]
      ).find((u) => u.username === SSO_USER);
    };
    const existing = await findUser();
    if (existing && !existing.active) {
      await page.request.patch(`/api/v1/users/${existing.id}`, {
        headers: { "X-CSRF-Token": csrf },
        data: { active: true },
      });
    }

    const saved = await putSettings(page, {
      "oidc.enabled": "true",
      "oidc.issuer_url": issuer,
      "oidc.client_id": WEB_CLIENT,
      "oidc.auto_login": "false",
      "mcp.oauth.enabled": "true",
      "mcp.oauth.resource": resource,
      "mcp.oauth.audience": "",
      "mcp.oauth.scopes": "read mcp",
    });
    expect(saved.status()).toBe(200);

    // 1. 메타데이터는 인증 없이 맨 JSON 으로, CORS 로 열린다.
    const anonymous = await browser.newContext();
    const metadata = await anonymous.request.get(
      "/.well-known/oauth-protected-resource/mcp",
    );
    expect(metadata.status()).toBe(200);
    expect(metadata.headers()["access-control-allow-origin"]).toBe("*");
    const doc = await metadata.json();
    expect(doc.resource).toBe(resource);
    expect(doc.authorization_servers).toEqual([issuer]);
    expect(doc.bearer_methods_supported).toEqual(["header"]);
    expect(doc.scopes_supported).toEqual(["read", "mcp"]);
    expect(doc.error).toBeUndefined();

    // 2. 토큰 없는 /mcp 401 은 메타데이터 주소를 가리키고, REST 401 은 그러지 않는다.
    const challenge = await mcp(anonymous.request, "");
    expect(challenge.status()).toBe(401);
    expect(challenge.headers()["www-authenticate"]).toBe(
      `Bearer realm="SeatOn", resource_metadata="${baseURL}/.well-known/oauth-protected-resource/mcp"`,
    );
    const rest = await anonymous.request.get("/api/v1/auth/me");
    expect(rest.status()).toBe(401);
    expect(rest.headers()["www-authenticate"]).toBeUndefined();

    // 3. 아직 웹으로 로그인한 적 없는 사람의 토큰은 거절되고 계정도 생기지 않는다.
    const token = idp.accessToken({ aud: ["account", resource] });
    if (!existing) {
      const unregistered = await mcp(anonymous.request, token);
      expect(unregistered.status()).toBe(401);
      const body = await unregistered.json();
      expect(body.error.code).toBe("account_not_registered");
      expect(body.error.message).toContain("웹으로");
      expect(unregistered.headers()["www-authenticate"]).toContain(
        'error="invalid_token"',
      );
      expect(await findUser()).toBeUndefined();
    }

    // 4. 웹으로 SSO 로그인 — 등록되는 순간이다.
    const ssoContext = await browser.newContext();
    const ssoPage = await ssoContext.newPage();
    await ssoPage.goto("/login");
    await ssoPage.getByRole("link", { name: "사내 SSO로 로그인" }).click();
    await ssoPage.waitForURL((url) => !url.pathname.startsWith("/login"));
    const me = await (await ssoPage.request.get("/api/v1/auth/me")).json();
    expect(me.user.username).toBe(SSO_USER);
    expect(me.user.source).toBe("oidc");
    await ssoContext.close();

    // 5. 같은 사람의 액세스 토큰이 이제 /mcp 를 연다 — 세션도 키도 없이.
    const listed = await mcp(anonymous.request, token);
    expect(listed.status()).toBe(200);
    const tools = (await listed.json()).result.tools as { name: string }[];
    expect(tools.map((t) => t.name)).toContain("search_employees");
    const searched = await mcp(anonymous.request, token, "tools/call", {
      name: "search_employees",
      arguments: { query: "김" },
    });
    expect(searched.status()).toBe(200);
    expect((await searched.json()).result.isError).toBe(false);

    // 6. 다른 앱용 토큰(aud=account, azp=다른 클라이언트)은 본 값과 고칠 값을 말하며 거절된다.
    const foreign = idp.accessToken({ azp: "other-app" });
    const rejected = await mcp(anonymous.request, foreign);
    expect(rejected.status()).toBe(401);
    const reason = await rejected.json();
    expect(reason.error.code).toBe("invalid_token");
    expect(reason.error.message).toContain('azp="other-app"');
    expect(reason.error.message).toContain(resource);

    // 7. 관리자가 허용 대상에 그 클라이언트를 적으면 매퍼 없이 통과한다.
    await putSettings(page, { "mcp.oauth.audience": "other-app" });
    expect((await mcp(anonymous.request, foreign)).status()).toBe(200);

    // 8. 유효한 토큰이라도 REST 는 받지 않는다.
    const restWithToken = await anonymous.request.get("/api/v1/employees", {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(restWithToken.status()).toBe(401);

    // 9. 비활성 계정의 토큰은 거절된다 — 정지된 계정이 MCP 로 되살아나지 않는다.
    const user = await findUser();
    expect(user).toBeDefined();
    await page.request.patch(`/api/v1/users/${user!.id}`, {
      headers: { "X-CSRF-Token": csrf },
      data: { active: false },
    });
    const inactive = await mcp(anonymous.request, token);
    expect(inactive.status()).toBe(401);
    expect((await inactive.json()).error.code).toBe("account_not_registered");
    await page.request.patch(`/api/v1/users/${user!.id}`, {
      headers: { "X-CSRF-Token": csrf },
      data: { active: true },
    });

    // 10. 키 페이지는 켜져 있을 때만 "키 없이 SSO 로 연결" 안내를 보인다.
    await page.goto("/profile/keys");
    await expect(page.getByTestId("mcp-sso-hint")).toContainText(resource);

    // 11. 끄면 메타데이터가 404 로 돌아가고 토큰은 키 전용 때처럼 거절된다.
    await putSettings(page, { "mcp.oauth.enabled": "false" });
    expect(
      (
        await anonymous.request.get("/.well-known/oauth-protected-resource/mcp")
      ).status(),
    ).toBe(404);
    const off = await mcp(anonymous.request, token);
    expect(off.status()).toBe(401);
    expect(off.headers()["www-authenticate"]).toBeUndefined();
    expect((await off.json()).error.code).toBe("authentication_required");
    await anonymous.close();
  });

  test("설정 화면의 카드가 MCP 주소와 메타데이터 주소를 보여 준다", async ({
    page,
    baseURL,
  }) => {
    await page.goto("/admin/settings");
    await page.getByRole("tab", { name: "Keycloak SSO" }).click();
    const card = page.getByTestId("mcp-oauth-card");
    await expect(card).toContainText("MCP SSO(OAuth) 인증");
    await expect(card.getByLabel("MCP 주소 (클라이언트에 줄 값)")).toHaveValue(
      `${baseURL}/mcp`,
    );
    await expect(card.getByLabel("메타데이터 주소")).toHaveValue(
      `${baseURL}/.well-known/oauth-protected-resource/mcp`,
    );
    await card
      .getByLabel("리소스 식별자 (resource)")
      .fill("https://seats.example.com/mcp");
    await expect(card.getByLabel("메타데이터 주소")).toHaveValue(
      "https://seats.example.com/.well-known/oauth-protected-resource/mcp",
    );
    await card.getByLabel("SSO 토큰에 주는 범위").fill("read admin");
    await expect(card).toContainText("read, write, mcp 중에서");
  });
});
