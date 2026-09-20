import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AdminAnalyticsPage, snippetBytes } from "./admin-pages";

const settings = {
  enabled: true,
  provider: "custom",
  momentoUrl: "",
  momentoSiteId: "",
  momentoProxy: true,
  measurementId: "",
  matomoUrl: "",
  matomoSiteId: "",
  customSnippet: '<script src="https://t.corp.example/t.js"></script>',
  allowedHosts: "",
  includeAdmin: false,
  placement: "head",
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

function renderAnalytics() {
  const calls: { url: string; method: string; body?: string }[] = [];
  const fetchMock = vi
    .fn()
    .mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";
      calls.push({ url, method, body: init?.body as string | undefined });
      if (url.endsWith("/admin/analytics/violations")) {
        return Promise.resolve(
          json({
            items: [
              {
                origin: "https://pixel.corp.example",
                directive: "img-src",
                page: "https://store.corp.example/apps",
                count: 4,
                firstSeen: "2026-09-12T05:00:00Z",
                lastSeen: "2026-09-12T05:10:00Z",
                allowed: false,
              },
              {
                origin: "https://t.corp.example",
                directive: "script-src-elem",
                page: "https://store.corp.example/",
                count: 1,
                firstSeen: "2026-09-12T05:00:00Z",
                lastSeen: "2026-09-12T05:00:00Z",
                allowed: true,
              },
            ],
          }),
        );
      }
      if (url.endsWith("/admin/analytics/allowed-hosts")) {
        return Promise.resolve(
          json({ ...settings, allowedHosts: "https://pixel.corp.example" }),
        );
      }
      return Promise.resolve(json(settings));
    });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/analytics"]}>
        <AdminAnalyticsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return calls;
}

describe("Analytics settings", () => {
  it("lists blocked origins and allows one with a click", async () => {
    const calls = renderAnalytics();
    const user = userEvent.setup();

    expect(await screen.findByText("https://pixel.corp.example")).toBeVisible();
    expect(screen.getByText("허용됨")).toBeVisible();
    const buttons = screen.getAllByRole("button", { name: "허용에 추가" });
    expect(buttons).toHaveLength(1);

    await user.click(buttons[0]!);
    await waitFor(() =>
      expect(
        calls.some(
          (call) =>
            call.method === "POST" &&
            call.url.endsWith("/admin/analytics/allowed-hosts") &&
            call.body ===
              JSON.stringify({ origin: "https://pixel.corp.example" }),
        ),
      ).toBe(true),
    );
  });

  it("shows the pasted snippet with its byte budget", async () => {
    renderAnalytics();
    const textarea = await screen.findByLabelText("추적 스니펫");
    expect(textarea).toHaveValue(settings.customSnippet);
    expect(
      screen.getByText(
        new RegExp(`${settings.customSnippet.length} / 8,192 bytes`),
      ),
    ).toBeVisible();
  });

  it("measures a snippet in encoded bytes", () => {
    expect(snippetBytes("abc")).toBe(3);
    expect(snippetBytes("한글")).toBe(6);
  });
});
