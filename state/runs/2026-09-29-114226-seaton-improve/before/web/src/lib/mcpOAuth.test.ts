import { describe, expect, it } from "vitest";
import {
  mcpMetadataURL,
  mcpOAuthActive,
  mcpOAuthProblem,
  mcpResource,
} from "./mcpOAuth";

describe("mcpResource", () => {
  it("설정값이 있으면 그것, 없으면 현재 오리진 + /mcp", () => {
    expect(mcpResource("https://seats.example.com/mcp", "http://x")).toBe(
      "https://seats.example.com/mcp",
    );
    expect(mcpResource("  ", "https://seaton.intra")).toBe(
      "https://seaton.intra/mcp",
    );
  });
});

describe("mcpMetadataURL", () => {
  it("리소스 식별자에서 RFC 9728 문서 주소를 만든다", () => {
    expect(mcpMetadataURL("https://seaton.intra/mcp")).toBe(
      "https://seaton.intra/.well-known/oauth-protected-resource/mcp",
    );
    expect(mcpMetadataURL("not a url")).toBe("");
  });
});

describe("mcpOAuthActive", () => {
  it("스위치·issuer·리소스 식별자가 모두 있어야 켜진다", () => {
    expect(mcpOAuthActive({ "mcp.oauth.enabled": "true" })).toBe(false);
    expect(
      mcpOAuthActive({
        "mcp.oauth.enabled": "true",
        "oidc.issuer_url": "https://keycloak.intra/realms/company",
      }),
    ).toBe(false);
    expect(
      mcpOAuthActive({
        "mcp.oauth.enabled": "true",
        "oidc.issuer_url": "https://keycloak.intra/realms/company",
        "mcp.oauth.resource": "https://seaton.intra/mcp",
      }),
    ).toBe(true);
    expect(
      mcpOAuthActive({
        "mcp.oauth.enabled": "false",
        "oidc.issuer_url": "https://keycloak.intra/realms/company",
        "mcp.oauth.resource": "https://seaton.intra/mcp",
      }),
    ).toBe(false);
  });
});

describe("mcpOAuthProblem", () => {
  const issuer = "https://keycloak.intra/realms/company";
  it("꺼져 있고 범위가 어휘 안이면 문제 없음", () => {
    expect(mcpOAuthProblem({ "mcp.oauth.scopes": "read mcp" })).toBe("");
  });
  it("어휘 밖 범위, issuer 없음, mcp 빠짐, 리소스 식별자 없음을 차례로 잡는다", () => {
    expect(mcpOAuthProblem({ "mcp.oauth.scopes": "read admin" })).toContain(
      "admin",
    );
    expect(
      mcpOAuthProblem({
        "mcp.oauth.enabled": "true",
        "mcp.oauth.scopes": "mcp",
      }),
    ).toContain("Issuer");
    expect(
      mcpOAuthProblem({
        "mcp.oauth.enabled": "true",
        "oidc.issuer_url": issuer,
        "mcp.oauth.scopes": "read",
      }),
    ).toContain("mcp");
    expect(
      mcpOAuthProblem({
        "mcp.oauth.enabled": "true",
        "oidc.issuer_url": issuer,
        "mcp.oauth.scopes": "read write mcp",
      }),
    ).toContain("리소스 식별자");
    expect(
      mcpOAuthProblem({
        "mcp.oauth.enabled": "true",
        "oidc.issuer_url": issuer,
        "mcp.oauth.scopes": "read write mcp",
        "mcp.oauth.resource": "https://seaton.intra/mcp",
      }),
    ).toBe("");
  });
});
