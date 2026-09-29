/**
 * MCP SSO(OAuth) 설정 카드의 순수 규칙. 서버(internal/app/mcpoauth.go)와 같은
 * 어휘와 같은 기본값을 쓴다 — 화면이 보여 주는 주소가 서버가 실제로 내는 것과
 * 달라서는 안 된다.
 */

/** 이 앱의 범위 어휘. api_keys.scopes 와 같다. */
export const MCP_SCOPES = ["read", "write", "mcp"] as const;

/**
 * 화면에 보여 줄 리소스 식별자. 설정값이 있으면 그것이고, 없으면 적을 값의
 * 제안으로 현재 오리진 + /mcp 를 보여 준다 — 서버는 빈 값을 요청 주소로 대신
 * 만들지 않으므로(Host 헤더가 허용 대상이 되면 안 된다) 제안일 뿐이다.
 */
export function mcpResource(configured: string, origin: string): string {
  const value = configured.trim();
  return value !== "" ? value : `${origin}/mcp`;
}

/** 거절된 클라이언트가 인증 서버를 찾으러 읽는 문서의 주소(RFC 9728). */
export function mcpMetadataURL(resource: string): string {
  try {
    const url = new URL(resource);
    return `${url.origin}/.well-known/oauth-protected-resource${url.pathname}`;
  } catch {
    return "";
  }
}

/**
 * 켜지는 조건: 스위치가 켜져 있고 Keycloak issuer 와 리소스 식별자가 있다.
 * 서버의 active() 와 같다. 스위치만 켜고 둘 중 하나가 없으면 꺼진 것처럼
 * 동작하므로 화면도 그렇게 말한다.
 */
export function mcpOAuthActive(values: Record<string, string>): boolean {
  return (
    values["mcp.oauth.enabled"] === "true" &&
    (values["oidc.issuer_url"] ?? "").trim() !== "" &&
    (values["mcp.oauth.resource"] ?? "").trim() !== ""
  );
}

/**
 * 저장 전에 화면에서 잡을 수 있는 문제. 서버 validate() 의 규칙 가운데 입력
 * 중에 바로 알려 줄 만한 것만 — 최종 판정은 서버가 한다.
 */
export function mcpOAuthProblem(values: Record<string, string>): string {
  const enabled = values["mcp.oauth.enabled"] === "true";
  const scopes = (values["mcp.oauth.scopes"] ?? "")
    .split(/\s+/)
    .filter(Boolean);
  const unknown = scopes.filter(
    (scope) => !(MCP_SCOPES as readonly string[]).includes(scope),
  );
  if (unknown.length > 0) {
    return `범위는 read, write, mcp 중에서 적습니다: ${unknown.join(", ")}`;
  }
  if (enabled && (values["oidc.issuer_url"] ?? "").trim() === "") {
    return "켜려면 위의 Keycloak Issuer URL 이 필요합니다";
  }
  if (enabled && !scopes.includes("mcp")) {
    return "범위에 mcp 가 있어야 합니다 — 없으면 토큰이 통과해도 /mcp 가 403 을 냅니다";
  }
  if (enabled && (values["mcp.oauth.resource"] ?? "").trim() === "") {
    return "켜려면 리소스 식별자가 필요합니다 — 요청 주소로 대신 만들지 않습니다";
  }
  return "";
}
