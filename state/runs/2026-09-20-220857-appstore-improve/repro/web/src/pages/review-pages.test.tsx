import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { ReviewDetailPage } from "./review-pages";

const reviewId = "dddddddd-dddd-4ddd-8ddd-dddddddddddd";
const appId = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";

function detail(overrides: Record<string, unknown> = {}) {
  return {
    id: reviewId,
    appId,
    appName: "Agent Hub",
    appSlug: "agent-hub",
    submitterId: "11111111-1111-4111-8111-111111111111",
    submitterName: "김개발",
    team: "AI Platform",
    level: 1,
    status: "pending",
    createdAt: "2026-09-16T01:00:00Z",
    app: {
      id: appId,
      slug: "agent-hub",
      name: "Agent Hub",
      summary: "팀의 AI Agent를 발견하고 실행하는 통합 허브",
      description: "검증된 에이전트를 한곳에서 탐색합니다.",
      serviceUrl: "https://agent.internal.example",
      category: { id: "ai", slug: "ai", name: "AI · Automation" },
      tags: ["AI", "Agent"],
      screenshots: [],
      language: "Go",
      framework: "React",
      supportsMcp: true,
      supportsApi: true,
      team: "AI Platform",
      version: "2.4.0",
      visibility: "public",
      status: "pending_review",
      updatedAt: "2026-09-16T00:00:00Z",
    },
    documents: [
      {
        id: "f1f1f1f1-f1f1-4f1f-8f1f-f1f1f1f1f1f1",
        appId,
        title: "운영 가이드",
        fileName: "guide.pdf",
        contentType: "application/pdf",
        size: 2048,
        createdAt: "2026-09-15T00:00:00Z",
        downloadUrl: `/api/v1/apps/${appId}/documents/f1f1f1f1-f1f1-4f1f-8f1f-f1f1f1f1f1f1`,
      },
    ],
    securityCheck: {
      enabled: true,
      configured: true,
      appId,
      appName: "Agent Hub",
      appStatus: "pending_review",
      bindingText: "",
      status: "APPROVED",
      reviewNumber: "SEC-2026-0042",
      approvedAt: "2026-09-15T02:00:00Z",
      verified: true,
    },
    history: [
      {
        id: reviewId,
        appId,
        level: 1,
        status: "pending",
        createdAt: "2026-09-16T01:00:00Z",
      },
      {
        id: "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
        appId,
        level: 1,
        status: "rejected",
        reason: "서비스 URL이 열리지 않습니다.",
        reviewerName: "박검토",
        createdAt: "2026-09-14T01:00:00Z",
        decidedAt: "2026-09-14T02:00:00Z",
      },
    ],
    ...overrides,
  };
}

function renderDetail(payload: Record<string, unknown>) {
  const calls: { url: string; method: string; body?: unknown }[] = [];
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
        calls.push({
          url: String(input),
          method: init?.method ?? "GET",
          body: init?.body,
        });
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
      <MemoryRouter initialEntries={[`/review/${reviewId}`]}>
        <Routes>
          <Route path="/review/:id" element={<ReviewDetailPage />} />
          <Route path="/review" element={<p>검토 대기</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return calls;
}

describe("Review detail", () => {
  it("shows the app, its guides, its security state and earlier decisions", async () => {
    renderDetail(detail());
    expect(
      await screen.findByText(/검증된 에이전트를 한곳에서/),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /서비스 URL 열기/ }),
    ).toHaveAttribute("href", "https://agent.internal.example");
    expect(screen.getByText("운영 가이드")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /내려받기/ })).toBeInTheDocument();
    expect(screen.getByText("보안 심의 완료")).toBeInTheDocument();
    expect(screen.getByText("SEC-2026-0042")).toBeInTheDocument();
    // The earlier rejection and its note, but not the review being decided.
    expect(
      screen.getByText("서비스 URL이 열리지 않습니다."),
    ).toBeInTheDocument();
    expect(screen.getByText("박검토", { exact: false })).toBeInTheDocument();
  });

  it("sends the reviewer's comment with an approval", async () => {
    const calls = renderDetail(detail());
    const user = userEvent.setup();
    await screen.findByText(/검증된 에이전트를 한곳에서/);
    await user.type(
      screen.getByLabelText("검토 의견"),
      "사내망에서 확인했습니다.",
    );
    await user.click(screen.getByRole("button", { name: /승인 및 게시/ }));
    await waitFor(() =>
      expect(calls.some((call) => call.url.endsWith("/approve"))).toBe(true),
    );
    const approval = calls.find((call) => call.url.endsWith("/approve"));
    expect(String(approval?.body)).toContain("사내망에서 확인했습니다.");
  });

  it("refuses to reject without a comment and sends it once written", async () => {
    const calls = renderDetail(detail());
    const user = userEvent.setup();
    await screen.findByText(/검증된 에이전트를 한곳에서/);
    await user.click(screen.getByRole("button", { name: /반려/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      /사유를 입력하세요/,
    );
    expect(calls.some((call) => call.url.endsWith("/reject"))).toBe(false);

    await user.type(screen.getByLabelText("검토 의견"), "URL을 고쳐 주세요.");
    await user.click(screen.getByRole("button", { name: /반려/ }));
    await waitFor(() =>
      expect(calls.some((call) => call.url.endsWith("/reject"))).toBe(true),
    );
    expect(
      String(calls.find((call) => call.url.endsWith("/reject"))?.body),
    ).toContain("URL을 고쳐 주세요.");
  });

  it("closes the decision once the review is settled", async () => {
    renderDetail(
      detail({
        status: "approved",
        reviewerName: "박검토",
        decidedAt: "2026-09-16T03:00:00Z",
        reason: "확인했습니다.",
      }),
    );
    expect(
      await screen.findByText(/이미 처리된 검토입니다/),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /승인 및 게시/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /반려/ })).toBeDisabled();
  });
});
