import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../../app/providers";
import { FavoritesProvider } from "./favorites";
import { AppCard } from "./app-card";
import type { StoreApp } from "../../types";

const app: StoreApp = {
  id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  slug: "agent-hub",
  name: "Agent Hub",
  summary: "팀의 AI Agent를 발견하고 실행하는 통합 허브",
  status: "published",
};

function renderCard(roles: string[]) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation((input: RequestInfo | URL) => {
      const url = String(input);
      const payload = url.includes("/auth/session")
        ? { authenticated: !!roles.length, user: { roles } }
        : {};
      return Promise.resolve(
        new Response(JSON.stringify(payload), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
      );
    }),
  );
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AuthProvider>
          <FavoritesProvider>
            <AppCard app={app} />
          </FavoritesProvider>
        </AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("App card", () => {
  it("spells out the detail link for a visitor", async () => {
    renderCard([]);
    expect(
      await screen.findByRole("link", { name: "Agent Hub 상세 보기" }),
    ).toHaveTextContent("자세히");
    expect(
      screen.queryByRole("link", { name: "Agent Hub 관리 설정 열기" }),
    ).toBeNull();
  });

  it("reduces the detail link to an icon for an administrator", async () => {
    renderCard(["user", "admin"]);
    // An administrator's footer also carries the admin shortcut, so the detail
    // link drops its wording and keeps only the icon.
    expect(
      await screen.findByRole("link", { name: "Agent Hub 관리 설정 열기" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Agent Hub 상세 보기" }).textContent,
    ).toBe("");
  });

  it("points the whole card at the app through the name link", () => {
    renderCard([]);
    const name = screen.getByRole("heading", { name: "Agent Hub" });
    const link = name.querySelector("a");
    // The overlay that makes the summary and the badges clickable is drawn
    // from this link in styles.css (.app-card-name a::after).
    expect(link).toHaveAttribute("href", "/apps/agent-hub");
    expect(link?.closest(".app-card")).not.toBeNull();
  });
});
