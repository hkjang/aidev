import { render, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AppProviders } from "./providers";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("AppProviders silent SSO", () => {
  it("sends a signed-out visitor to the silent login with the page kept as returnTo", async () => {
    sessionStorage.clear();
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const path = new URL(String(input), "http://localhost").pathname;
        if (path === "/api/v1/public/config")
          return Promise.resolve(
            jsonResponse({
              siteName: "AppStore",
              publicMode: true,
              oidcEnabled: true,
              oidcConfigured: true,
              oidcAutoLogin: true,
              workflowEnabled: true,
              theme: "system",
            }),
          );
        if (path === "/api/v1/auth/session")
          return Promise.resolve(jsonResponse({ authenticated: false }));
        return Promise.resolve(jsonResponse({}, 404));
      }),
    );
    const assign = vi.fn();
    vi.spyOn(window, "location", "get").mockReturnValue({
      ...window.location,
      assign,
    } as Location);

    render(
      <MemoryRouter initialEntries={["/apps/agent-hub?tab=docs"]}>
        <AppProviders>
          <div />
        </AppProviders>
      </MemoryRouter>,
    );

    await waitFor(() =>
      expect(assign).toHaveBeenCalledWith(
        "/api/v1/auth/oidc/login?prompt=none&returnTo=%2Fapps%2Fagent-hub%3Ftab%3Ddocs",
      ),
    );
    // One attempt per tab session: the flag is set before the browser leaves.
    expect(assign).toHaveBeenCalledTimes(1);
    expect(sessionStorage.getItem("appstore.sso.silentAttempted")).toBe("true");
    sessionStorage.clear();
  });
});
