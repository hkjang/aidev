import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { AppSecurityCheckPage } from "./owner-page";
import { SecurityVerifiedBadge } from "./verified-badge";

const appId = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const binding =
  "[APPSTORE-SECURITY-CHECK:v1]\napp_id=" +
  appId +
  "\n[/APPSTORE-SECURITY-CHECK]";

function view(overrides: Record<string, unknown> = {}) {
  return {
    enabled: true,
    configured: true,
    appId,
    appName: "Agent Hub",
    appStatus: "draft",
    bindingText: binding,
    newReviewUrl: "https://seccheck.example.internal/reviews/new",
    status: "UNVERIFIED",
    verified: false,
    ...overrides,
  };
}

function renderOwnerPage(
  initial: Record<string, unknown>,
  verified = view({ verified: true, status: "APPROVED" }),
) {
  const calls: { url: string; method: string; body?: unknown }[] = [];
  const fetchMock = vi
    .fn()
    .mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      calls.push({ url, method: init?.method ?? "GET", body: init?.body });
      const payload = url.includes("/security-check/verify")
        ? verified
        : url.includes("/security-check")
          ? initial
          : url.includes("/submit")
            ? { id: appId, status: "pending_review" }
            : {};
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
      <MemoryRouter initialEntries={[`/my/apps/${appId}/security`]}>
        <Routes>
          <Route
            path="/my/apps/:id/security"
            element={<AppSecurityCheckPage />}
          />
          <Route path="/my/apps" element={<p>내 앱</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return calls;
}

describe("App security check", () => {
  it("shows the block to paste into SecCheck and withholds submission", async () => {
    renderOwnerPage(view());
    expect(
      await screen.findByText(/APPSTORE-SECURITY-CHECK/),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /SecCheck에서 심의 등록/ }),
    ).toHaveAttribute("href", "https://seccheck.example.internal/reviews/new");
    // Nothing may be submitted before an approval is confirmed.
    expect(screen.queryByRole("button", { name: /검토 제출/ })).toBeNull();
    // The badge and the SecCheck status row both read 미확인 before a review.
    expect(screen.getAllByText("미확인").length).toBeGreaterThan(0);
  });

  it("verifies a review and then opens submission", async () => {
    const calls = renderOwnerPage(view());
    const user = userEvent.setup();
    await screen.findByText(/APPSTORE-SECURITY-CHECK/);
    await user.type(
      screen.getByLabelText("심의 ID"),
      "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    );
    await user.click(screen.getByRole("button", { name: /심의 결과 확인/ }));

    const submit = await screen.findByRole("button", { name: /검토 제출/ });
    const verifyCall = calls.find((call) => call.url.includes("/verify"));
    expect(verifyCall?.method).toBe("POST");
    expect(String(verifyCall?.body)).toContain(
      "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    );

    await user.click(submit);
    await waitFor(() =>
      expect(calls.some((call) => call.url.endsWith("/submit"))).toBe(true),
    );
  });

  it("explains why a review was refused", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockImplementation((_input: RequestInfo | URL, init?: RequestInit) => {
          if ((init?.method ?? "GET") === "POST") {
            return Promise.resolve(
              new Response(
                JSON.stringify({
                  error: {
                    code: "SECURITY_CHECK_BINDING_MISMATCH",
                    message: "심의 설명에 이 앱의 연동 정보가 없습니다.",
                  },
                }),
                {
                  status: 409,
                  headers: { "content-type": "application/json" },
                },
              ),
            );
          }
          return Promise.resolve(
            new Response(JSON.stringify(view()), {
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
        <MemoryRouter initialEntries={[`/my/apps/${appId}/security`]}>
          <Routes>
            <Route
              path="/my/apps/:id/security"
              element={<AppSecurityCheckPage />}
            />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    const user = userEvent.setup();
    await screen.findByText(/APPSTORE-SECURITY-CHECK/);
    await user.type(
      screen.getByLabelText("심의 ID"),
      "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
    );
    await user.click(screen.getByRole("button", { name: /심의 결과 확인/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      /연동 정보가 없습니다/,
    );
  });
});

describe("Security verified badge", () => {
  it("labels a cleared app and stays away otherwise", () => {
    const { rerender } = render(
      <SecurityVerifiedBadge app={{ securityVerified: true }} />,
    );
    expect(screen.getByText(/보안 심의 완료/)).toBeInTheDocument();
    rerender(<SecurityVerifiedBadge app={{ securityVerified: false }} />);
    expect(screen.queryByText(/보안 심의 완료/)).toBeNull();
  });
});
