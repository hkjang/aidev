import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../app/providers";
import { LoginPage } from "./auth-pages";

const REFUSAL_NOTICE = "자동으로 로그인하지 않았습니다";

function json(payload: unknown) {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

// Silent SSO is on, so the callback may send the browser back here with
// ?sso=none when the identity provider has no session for it.
function renderLogin(url: string, config: Record<string, unknown> = {}) {
  const fetchMock = vi.fn().mockImplementation((input: RequestInfo | URL) => {
    const target = String(input);
    if (target.includes("/auth/session")) {
      return Promise.resolve(
        json({ authenticated: false, bootstrapAvailable: false }),
      );
    }
    if (target.includes("/public/config")) {
      return Promise.resolve(
        json({
          siteName: "AppStore",
          oidcEnabled: true,
          oidcConfigured: true,
          oidcAutoLogin: true,
          ...config,
        }),
      );
    }
    return Promise.resolve(json({ version: "test" }));
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[url]}>
        <AuthProvider>
          <LoginPage />
        </AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function findSsoLink() {
  return screen.findByRole("link", { name: /회사 계정으로 SSO 로그인/ });
}

describe("Login page after a refused silent SSO", () => {
  it("explains why the visitor was not signed in automatically", async () => {
    renderLogin("/login?sso=none");

    await findSsoLink();
    const notice = await screen.findByRole("status");
    expect(notice).toHaveTextContent(REFUSAL_NOTICE);
  });

  it("stays quiet when the address carries no sso=none marker", async () => {
    renderLogin("/login?returnTo=%2Fsubmit");

    await findSsoLink();
    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByText(new RegExp(REFUSAL_NOTICE))).toBeNull();
  });

  it("keeps the SSO link carrying only returnTo", async () => {
    renderLogin("/login?sso=none&returnTo=%2Fmy%2Fapps");

    const link = await findSsoLink();
    expect(link).toHaveAttribute(
      "href",
      "/api/v1/auth/oidc/login?returnTo=%2Fmy%2Fapps",
    );
    expect(await screen.findByRole("status")).toHaveTextContent(REFUSAL_NOTICE);
  });

  it("drops the notice once SSO is switched off, even from an old address", async () => {
    renderLogin("/login?sso=none", { oidcEnabled: false });

    // Without SSO there is no button the notice could point at, so the
    // screen falls back to whatever login method remains.
    expect(
      await screen.findByText(/사용 가능한 로그인 방식이 없습니다/),
    ).toBeVisible();
    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByText(new RegExp(REFUSAL_NOTICE))).toBeNull();
  });
});
