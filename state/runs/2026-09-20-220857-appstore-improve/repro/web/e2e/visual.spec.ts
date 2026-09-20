import { expect, test, type Page, type TestInfo } from "@playwright/test";
import { installMockApi } from "./mock-api";

interface CaptureRoute {
  name: string;
  path: string;
  heading: string;
}

const publicRoutes: CaptureRoute[] = [
  { name: "today", path: "/", heading: "팀의 좋은 앱을" },
  { name: "today-alias", path: "/today", heading: "팀의 좋은 앱을" },
  { name: "apps", path: "/apps?q=agent&sort=trending", heading: "모든 앱" },
  { name: "app-detail", path: "/apps/agent-hub", heading: "Agent Hub" },
  { name: "categories", path: "/categories", heading: "카테고리" },
  { name: "category-detail", path: "/categories/ai", heading: "ai 앱" },
  { name: "search-alias", path: "/search?q=agent", heading: "모든 앱" },
  { name: "mcp-apps", path: "/apps?mcp=true", heading: "MCP 앱" },
  { name: "favorites", path: "/favorites", heading: "즐겨찾기" },
  { name: "forbidden", path: "/403", heading: "접근 권한이 없습니다" },
  {
    name: "not-found",
    path: "/missing-page",
    heading: "페이지를 찾을 수 없습니다",
  },
];

const personalRoutes: CaptureRoute[] = [
  { name: "my-home", path: "/my", heading: "내 AppStore" },
  { name: "my-apps", path: "/my/apps", heading: "내가 등록한 앱" },
  {
    name: "my-app-edit",
    path: "/my/apps/ffffffff-ffff-4fff-8fff-ffffffffffff/edit",
    heading: "앱 수정",
  },
  {
    name: "my-app-security",
    path: "/my/apps/ffffffff-ffff-4fff-8fff-ffffffffffff/security",
    heading: "보안 심의",
  },
  { name: "my-keys", path: "/my/keys", heading: "API · MCP 키" },
  { name: "my-profile", path: "/my/profile", heading: "내 프로필" },
  { name: "my-activity", path: "/my/activity", heading: "내 활동 내역" },
  { name: "my-settings", path: "/my/settings", heading: "개인 설정" },
  { name: "my-favorites-alias", path: "/my/favorites", heading: "즐겨찾기" },
  { name: "submit", path: "/submit", heading: "앱 등록" },
  { name: "review", path: "/review", heading: "검토 대기" },
  {
    name: "review-detail",
    path: "/review/dddddddd-dddd-4ddd-8ddd-dddddddddddd",
    heading: "Flow Studio",
  },
];

const adminRoutes: CaptureRoute[] = [
  { name: "admin-dashboard", path: "/admin", heading: "대시보드" },
  { name: "admin-apps", path: "/admin/apps", heading: "앱 관리" },
  { name: "admin-app-new", path: "/admin/apps/new", heading: "앱 추가" },
  {
    name: "admin-app-detail",
    path: "/admin/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    heading: "Agent Hub",
  },
  {
    name: "admin-categories",
    path: "/admin/categories",
    heading: "카테고리",
  },
  { name: "admin-users", path: "/admin/users", heading: "사용자" },
  { name: "admin-roles", path: "/admin/roles", heading: "역할·권한" },
  { name: "admin-reviews", path: "/admin/reviews", heading: "검토 대기" },
  { name: "admin-workflow", path: "/admin/workflow", heading: "승인 워크플로" },
  { name: "admin-ai", path: "/admin/ai", heading: "AI 공급자" },
  { name: "admin-api", path: "/admin/api", heading: "REST API" },
  { name: "admin-mcp", path: "/admin/mcp", heading: "MCP 서버" },
  { name: "admin-keys", path: "/admin/api-keys", heading: "API 키" },
  {
    name: "admin-authentication",
    path: "/admin/authentication",
    heading: "인증·SSO",
  },
  {
    name: "admin-security",
    path: "/admin/security",
    heading: "보안·키 정책",
  },
  {
    name: "admin-security-check",
    path: "/admin/security-check",
    heading: "보안 심의 연동",
  },
  {
    name: "admin-analytics",
    path: "/admin/analytics",
    heading: "방문 추적",
  },
  { name: "admin-audit", path: "/admin/audit", heading: "감사 로그" },
  {
    name: "admin-settings",
    path: "/admin/settings",
    heading: "시스템 설정",
  },
];

async function capture(
  page: Page,
  testInfo: TestInfo,
  routes: CaptureRoute[],
): Promise<void> {
  for (const route of routes) {
    await page.goto(route.path);
    await expect(
      page.getByRole("heading", { name: new RegExp(route.heading) }).first(),
    ).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath("screens", `${route.name}.png`),
      fullPage: true,
      animations: "disabled",
    });
  }
}

test("공개 주요 화면 캡처", async ({ page }, testInfo) => {
  await installMockApi(page);
  await capture(page, testInfo, publicRoutes);
  await page.goto("/login?returnTo=%2Fsubmit");
  await expect(
    page.getByRole("heading", { name: "AppStore 로그인" }),
  ).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath("screens", "login.png"),
    fullPage: true,
    animations: "disabled",
  });
  await page.goto("/admin/bootstrap");
  await expect(
    page.getByRole("heading", { name: "AppStore 로그인" }),
  ).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath("screens", "admin-bootstrap-alias.png"),
    fullPage: true,
    animations: "disabled",
  });
  await page.goto("/");
  await page.getByRole("button", { name: "라이트 모드로 전환" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.screenshot({
    path: testInfo.outputPath("screens", "today-light.png"),
    fullPage: true,
    animations: "disabled",
  });
});

test("개인화·검토 주요 화면 캡처", async ({ page }, testInfo) => {
  await installMockApi(page, { authenticated: true });
  await capture(page, testInfo, personalRoutes);
});

test("관리자 전체 주요 화면 캡처", async ({ page }, testInfo) => {
  await installMockApi(page, { authenticated: true });
  await capture(page, testInfo, adminRoutes);
});
