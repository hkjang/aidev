import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PublicConfig } from "../../types";
import {
  beginSilentSso,
  clearSilentSsoState,
  isSilentSsoPath,
  markSignedOut,
  shouldAttemptSilentSso,
  silentSsoUrl,
} from "./silent-sso";

const autoLoginConfig: PublicConfig = {
  siteName: "AppStore",
  publicMode: true,
  oidcEnabled: true,
  oidcConfigured: true,
  oidcAutoLogin: true,
  workflowEnabled: true,
};

function decide(overrides: Partial<PublicConfig> = {}, path = "/my/apps") {
  const [pathname = "/", search = ""] = path.split("?");
  return shouldAttemptSilentSso({
    config: { ...autoLoginConfig, ...overrides },
    pathname,
    search: search ? `?${search}` : "",
  });
}

describe("silent SSO rules", () => {
  beforeEach(() => sessionStorage.clear());
  afterEach(() => sessionStorage.clear());

  it("attempts only when the administrator turned auto_login on", () => {
    expect(decide()).toBe(true);
    expect(decide({ oidcAutoLogin: false })).toBe(false);
    expect(decide({ oidcAutoLogin: undefined })).toBe(false);
    expect(decide({ oidcEnabled: false })).toBe(false);
    expect(decide({ oidcConfigured: false })).toBe(false);
    expect(
      shouldAttemptSilentSso({ config: undefined, pathname: "/", search: "" }),
    ).toBe(false);
  });

  it("never retries in the same tab session once an attempt was made", () => {
    const assign = vi.fn();
    vi.spyOn(window, "location", "get").mockReturnValue({
      ...window.location,
      assign,
    } as Location);
    expect(decide()).toBe(true);
    beginSilentSso("/my/apps");
    expect(assign).toHaveBeenCalledWith(
      "/api/v1/auth/oidc/login?prompt=none&returnTo=%2Fmy%2Fapps",
    );
    // A reload after the provider refused must not bounce again.
    expect(decide()).toBe(false);
  });

  it("honours the refusal marker the callback leaves in the address", () => {
    expect(decide({}, "/login?sso=none")).toBe(false);
    expect(decide({}, "/?sso=none")).toBe(false);
    expect(decide({}, "/?sso=error")).toBe(false);
    expect(decide({}, "/?tab=drafts")).toBe(true);
  });

  it("stays quiet after a deliberate sign-out until a session exists again", () => {
    markSignedOut();
    expect(decide()).toBe(false);
    clearSilentSsoState();
    expect(decide()).toBe(true);
  });

  it("treats unreadable storage as already attempted", () => {
    // Private modes and blocked site data throw on access; reading that as
    // "not yet attempted" would start a redirect loop.
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });
    expect(decide()).toBe(false);
  });

  it("does not attempt from the login, error, or non-page paths", () => {
    for (const path of [
      "/login",
      "/403",
      "/admin/bootstrap",
      "/api/v1/auth/oidc/callback",
      "/api/v1/apps",
      "/mcp",
      "/health/ready",
      "/healthz",
      "/readyz",
      "/docs",
      "/openapi.json",
      "/momento/collect",
    ]) {
      expect(isSilentSsoPath(path), path).toBe(false);
      expect(decide({}, path), path).toBe(false);
    }
    for (const path of ["/", "/apps/agent-hub", "/my/apps", "/admin/users"]) {
      expect(isSilentSsoPath(path), path).toBe(true);
    }
  });

  it("carries the deep link back but only within this origin", () => {
    expect(silentSsoUrl("/apps/agent-hub?tab=docs")).toBe(
      "/api/v1/auth/oidc/login?prompt=none&returnTo=%2Fapps%2Fagent-hub%3Ftab%3Ddocs",
    );
    expect(silentSsoUrl("//evil.example/phish")).toBe(
      "/api/v1/auth/oidc/login?prompt=none&returnTo=%2F",
    );
    expect(silentSsoUrl("https://evil.example/")).toBe(
      "/api/v1/auth/oidc/login?prompt=none&returnTo=%2F",
    );
  });
});
