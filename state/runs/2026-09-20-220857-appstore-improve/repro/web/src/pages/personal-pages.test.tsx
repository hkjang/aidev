import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AuthProvider } from "../app/providers";
import { FavoritesProvider } from "../features/apps/favorites";
import { MyAppsPage, MyDashboardPage } from "./personal-pages";

function myApp(overrides: Record<string, unknown>) {
  return {
    id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    slug: "agent-hub",
    name: "Agent Hub",
    summary: "AI 에이전트 카탈로그",
    ...overrides,
  };
}

function renderMyApps(items: Record<string, unknown>[], path = "/my/apps") {
  const fetchMock = vi.fn().mockImplementation((input: RequestInfo | URL) => {
    const url = String(input);
    const payload = url.includes("/auth/session")
      ? { authenticated: true, user: { roles: ["user", "admin"] } }
      : url.includes("/me/apps")
        ? { items, total: items.length }
        : { items: [] };
    return Promise.resolve(
      new Response(JSON.stringify(payload), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <AuthProvider>
          <FavoritesProvider>
            {path === "/my" ? <MyDashboardPage /> : <MyAppsPage />}
          </FavoritesProvider>
        </AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("My applications", () => {
  it("labels each card with the status the owner cannot otherwise see", async () => {
    renderMyApps([
      myApp({ status: "draft" }),
      myApp({
        id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
        slug: "flow-studio",
        name: "Flow Studio",
        status: "pending_review",
      }),
      myApp({
        id: "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
        slug: "secure-vault",
        name: "Secure Vault",
        status: "archived",
      }),
    ]);

    const cards = await screen.findAllByRole("article");
    expect(cards).toHaveLength(3);
    expect(cards[0]).toHaveTextContent("초안");
    expect(cards[1]).toHaveTextContent("검토 대기");
    expect(cards[2]).toHaveTextContent("보관됨");
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("opens the edit screen from a card the public detail route would 404 on", async () => {
    renderMyApps([
      myApp({ status: "published", visibility: "public" }),
      myApp({
        id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
        slug: "flow-studio",
        name: "Flow Studio",
        status: "pending_review",
      }),
      myApp({
        id: "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
        slug: "secure-vault",
        name: "Secure Vault",
        status: "published",
        visibility: "private",
      }),
    ]);

    // A published, public app still opens its catalog page.
    const published = await screen.findByRole("heading", {
      name: "Agent Hub",
    });
    expect(published.querySelector("a")).toHaveAttribute(
      "href",
      "/apps/agent-hub",
    );
    expect(
      screen.getByRole("link", { name: "Agent Hub 상세 보기" }),
    ).toHaveAttribute("href", "/apps/agent-hub");

    // /apps/{slug} serves only published, public apps, so the other two have
    // no page there: the name link goes to the owner's edit screen and the
    // 자세히 button, with nothing to show, is left out.
    for (const [name, id] of [
      ["Flow Studio", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"],
      ["Secure Vault", "cccccccc-cccc-4ccc-8ccc-cccccccccccc"],
    ]) {
      const heading = screen.getByRole("heading", { name });
      expect(heading.querySelector("a")).toHaveAttribute(
        "href",
        `/my/apps/${id}/edit`,
      );
      expect(
        screen.queryByRole("link", { name: `${name} 상세 보기` }),
      ).toBeNull();
    }
  });

  it("shows why a rejected app was sent back", async () => {
    renderMyApps([
      myApp({
        status: "rejected",
        review: {
          id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
          appId: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
          status: "rejected",
          level: 1,
          reason: "서비스 URL이 사내망에서 열리지 않습니다.",
          reviewerName: "검토자 김",
          decidedAt: "2026-09-01T08:00:00Z",
        },
      }),
    ]);

    const notice = await screen.findByRole("alert");
    expect(notice).toHaveTextContent(
      "서비스 URL이 사내망에서 열리지 않습니다.",
    );
    expect(notice).toHaveTextContent("검토자 김");
    expect(notice).toHaveTextContent("1단계 검토");
  });

  it("keeps a superseded reason hidden once the app is back in review", async () => {
    renderMyApps([
      myApp({
        status: "pending_review",
        review: {
          id: "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
          appId: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
          status: "pending",
          level: 1,
        },
      }),
    ]);

    expect(
      await screen.findByRole("heading", { name: "Agent Hub" }),
    ).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
  });
});

describe.each(["/my/apps", "/my"])("Owner favorites on %s", (path) => {
  it.each([
    ["draft", "public", "초안"],
    ["pending_review", "public", "검토 대기"],
    ["rejected", "public", "반려"],
    ["archived", "public", "보관됨"],
    ["published", "private", "게시됨"],
  ])(
    "hides both favorite actions for %s/%s and preserves storage",
    async (status, visibility, label) => {
      // One saved and one unsaved app exercise both action labels through the
      // real provider. Keep whitespace to detect even a render-time rewrite.
      const saved = ' [ "saved-app", "unrelated-app" ] ';
      localStorage.setItem("appstore.favorites", saved);
      renderMyApps(
        [
          myApp({
            id: "saved-id",
            slug: "saved-app",
            name: "Saved App",
            status,
            visibility,
          }),
          myApp({
            id: "unsaved-id",
            slug: "unsaved-app",
            name: "Unsaved App",
            status,
            visibility,
          }),
        ],
        path,
      );

      for (const [name, id] of [
        ["Saved App", "saved-id"],
        ["Unsaved App", "unsaved-id"],
      ]) {
        const heading = await screen.findByRole("heading", { name });
        const card = within(heading.closest("article")!);
        expect(card.queryByRole("button", { name: /즐겨찾기/ })).toBeNull();
        expect(heading.querySelector("a")).toHaveAttribute(
          "href",
          `/my/apps/${id}/edit`,
        );
        expect(card.getByText(label)).toBeVisible();
        expect(
          card.queryByRole("link", { name: `${name} 상세 보기` }),
        ).toBeNull();
        expect(
          await card.findByRole("link", { name: `${name} 관리 설정 열기` }),
        ).toHaveAttribute("href", `/admin/apps/${id}`);
      }
      expect(localStorage.getItem("appstore.favorites")).toBe(saved);
    },
  );

  it("keeps the published public card's favorite toggle and detail links", async () => {
    renderMyApps([myApp({ status: "published", visibility: "public" })], path);
    const add = await screen.findByRole("button", {
      name: "Agent Hub 즐겨찾기 추가",
    });
    expect(add).toHaveAttribute("aria-pressed", "false");
    expect(
      screen.getByRole("heading", { name: "Agent Hub" }).querySelector("a"),
    ).toHaveAttribute("href", "/apps/agent-hub");
    expect(
      screen.getByRole("link", { name: "Agent Hub 상세 보기" }),
    ).toHaveAttribute("href", "/apps/agent-hub");
    await userEvent.click(add);
    const remove = screen.getByRole("button", {
      name: "Agent Hub 즐겨찾기 해제",
    });
    expect(remove).toHaveAttribute("aria-pressed", "true");
    expect(JSON.parse(localStorage.getItem("appstore.favorites")!)).toEqual([
      "agent-hub",
    ]);
    await userEvent.click(remove);
    expect(
      screen.getByRole("button", { name: "Agent Hub 즐겨찾기 추가" }),
    ).toHaveAttribute("aria-pressed", "false");
    expect(JSON.parse(localStorage.getItem("appstore.favorites")!)).toEqual([]);
  });
});
