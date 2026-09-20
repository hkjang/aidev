import { expect, test } from "@playwright/test";
import { installMockApi } from "./mock-api";

test("공개 탐색과 URL 검색 상태는 새로고침 후에도 유지된다", async ({
  page,
}) => {
  await installMockApi(page);
  await page.goto("/apps?q=agent&category=ai&sort=trending");
  await expect(page.getByRole("heading", { name: "모든 앱" })).toBeVisible();
  await expect(page.locator("#catalog-search")).toHaveValue("agent");
  await expect(page.getByLabel("카테고리")).toHaveValue("ai");
  await expect(page.getByLabel("정렬")).toHaveValue("trending");
  await expect(page.getByRole("heading", { name: "Agent Hub" })).toBeVisible();
  await page.reload();
  await expect(page).toHaveURL(/q=agent&category=ai&sort=trending/);
  await expect(page.locator("#catalog-search")).toHaveValue("agent");
});

test("비로그인 등록 접근은 원래 URL을 보존한 로그인 화면으로 이동한다", async ({
  page,
}) => {
  await installMockApi(page);
  await page.goto("/submit");
  await expect(page).toHaveURL(/\/login\?returnTo=%2Fsubmit/);
  await expect(
    page.getByRole("heading", { name: "AppStore 로그인" }),
  ).toBeVisible();
  await expect(page.getByText("AppStore v2.0.0")).toBeVisible();
});

test("일반 사용자는 관리자 화면에서 403 상태를 본다", async ({ page }) => {
  await installMockApi(page, {
    authenticated: true,
    roles: ["user", "contributor"],
  });
  await page.goto("/admin");
  await expect(
    page.getByRole("heading", { name: "접근 권한이 없습니다" }),
  ).toBeVisible();
});

test("관리자 사용자 목록은 가상화되고 route refresh를 견딘다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/users?q=developer");
  await expect(page.getByRole("heading", { name: "사용자" })).toBeVisible();
  await expect(page.getByText("150명")).toBeVisible();
  await expect(page.getByRole("list", { name: "사용자 목록" })).toBeVisible();
  await page.reload();
  await expect(page).toHaveURL(/\/admin\/users\?q=developer/);
  await expect(page.getByRole("heading", { name: "사용자" })).toBeVisible();
});

test("AI는 256K 설정과 SSE token streaming을 처리한다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/ai");
  const contextWindow = page.getByLabel("Context Window").first();
  await expect(contextWindow).toHaveValue("262144");
  await contextWindow.fill("262144");
  await page.getByLabel("Model", { exact: true }).selectOption("compact-model");
  await page.locator("#stream-max").fill("9999");
  await expect(page.locator("#stream-max")).toHaveValue("4096");
  await page.getByRole("button", { name: "Streaming 시작" }).click();
  await expect(page.getByText("스트리밍 응답", { exact: true })).toBeVisible();
  await expect(page.getByText(/totalTokens/)).toBeVisible();
});

test("관리자는 Key Permission과 Template을 편집할 수 있다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/security");
  await expect(
    page.getByRole("heading", { name: "Key Permission Definitions" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "편집" }).first().click();
  await page.getByLabel("표시 이름").fill("앱 카탈로그 읽기");
  await page.getByRole("dialog").getByRole("button", { name: "저장" }).click();
  await expect(page.getByRole("dialog")).not.toBeVisible();

  await page.getByRole("button", { name: "Template 추가" }).click();
  await page.getByLabel("이름").fill("AI Client");
  await page.getByLabel("AI 사용").check();
  await page.getByRole("dialog").getByRole("button", { name: "저장" }).click();
  await expect(page.getByRole("dialog")).not.toBeVisible();
});

test("light/dark 테마 선택은 새로고침 후 유지된다", async ({ page }) => {
  await installMockApi(page);
  await page.goto("/");
  await page.evaluate(() => localStorage.setItem("appstore.theme", "light"));
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.getByRole("button", { name: "다크 모드로 전환" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("모바일 메뉴는 키보드와 명시적 버튼으로 접근 가능하다", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "mobile", "모바일 프로젝트에서만 실행");
  await installMockApi(page);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "메뉴 열기" });
  await menu.click();
  await expect(
    page.getByRole("complementary", { name: "주 메뉴" }),
  ).toHaveClass(/open/);
  await expect(page.getByRole("link", { name: /MCP 앱/ })).toBeVisible();
});

test("관리자는 앱 상세를 수정하고 삭제 확인 후 목록으로 돌아온다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/apps");
  await page.getByRole("link", { name: "Agent Hub", exact: true }).click();
  await expect(page).toHaveURL(
    "/admin/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  );
  await expect(page.getByLabel("앱 이름")).toHaveValue("Agent Hub");
  await page.getByLabel("게시 상태").selectOption("archived");
  await page.getByRole("button", { name: "변경 저장" }).click();
  await expect(page.getByText("앱 정보가 저장되었습니다.")).toBeVisible();

  await page.getByRole("button", { name: "앱 삭제" }).click();
  const dialog = page.getByRole("dialog", { name: "앱을 영구 삭제할까요?" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "영구 삭제" }).click();
  await expect(page).toHaveURL("/admin/apps");
});

test("SSO 연결 테스트는 discovery endpoint를 표시한다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/authentication");
  await page.getByRole("button", { name: "SSO 연결 테스트" }).click();
  await expect(page.getByText("discovery 문서를 정상적으로")).toBeVisible();
  await expect(
    page.getByText(
      "https://sso.example.internal/realms/company/protocol/openid-connect/token",
    ),
  ).toBeVisible();
  await expect(page.getByText("지원", { exact: true })).toBeVisible();
});

test("스토어 메뉴는 Apps와 MCP Apps를 동시에 선택하지 않는다", async ({
  page,
}) => {
  await installMockApi(page);
  const menu = page.getByRole("navigation").or(page.locator(".sidebar-scroll"));
  await page.goto("/apps");
  await expect(
    menu.getByRole("link", { name: "전체 앱", exact: true }),
  ).toHaveClass(/active/);
  await expect(menu.getByRole("link", { name: "MCP 앱" })).not.toHaveClass(
    /active/,
  );
  await page.goto("/apps?mcp=true");
  await expect(menu.getByRole("link", { name: "MCP 앱" })).toHaveClass(
    /active/,
  );
  await expect(
    menu.getByRole("link", { name: "전체 앱", exact: true }),
  ).not.toHaveClass(/active/);
});

test("SSO를 설정해도 복구용 관리자 로그인을 계속 사용할 수 있다", async ({
  page,
}) => {
  await installMockApi(page);
  await page.goto("/login");
  await expect(
    page.getByRole("link", { name: "회사 계정으로 SSO 로그인" }),
  ).toBeVisible();

  const toggle = page.getByRole("button", { name: "관리자 계정으로 로그인" });
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByLabel("Bootstrap 관리자")).toBeHidden();

  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByLabel("Bootstrap 관리자")).toBeVisible();
  await expect(page.getByLabel("비밀번호")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "관리자 로그인" }),
  ).toBeVisible();
});

test("거절된 silent SSO 뒤 로그인 화면은 이유를 알리고 SSO 링크는 그대로다", async ({
  page,
}) => {
  await installMockApi(page);
  await page.goto("/login?sso=none&returnTo=%2Fmy%2Fapps");
  await expect(page.getByRole("status")).toContainText(
    "자동으로 로그인하지 않았습니다",
  );
  await expect(
    page.getByRole("link", { name: "회사 계정으로 SSO 로그인" }),
  ).toHaveAttribute("href", "/api/v1/auth/oidc/login?returnTo=%2Fmy%2Fapps");

  await page.goto("/login");
  await expect(page.getByRole("status")).toHaveCount(0);
});

test("관리자는 스토어 목록과 상세에서 관리 설정으로 바로 이동한다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/apps");
  const cardShortcut = page
    .getByRole("link", { name: "Agent Hub 관리 설정 열기" })
    .first();
  await expect(cardShortcut).toBeVisible();

  await page.goto("/apps/agent-hub");
  await page.getByRole("link", { name: "Agent Hub 관리 설정 열기" }).click();
  await expect(page).toHaveURL(
    "/admin/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  );
  await expect(page.getByRole("heading", { name: "Agent Hub" })).toBeVisible();
});

test("일반 사용자에게는 관리 설정 바로가기가 보이지 않는다", async ({
  page,
}) => {
  await installMockApi(page, {
    authenticated: true,
    roles: ["user", "contributor"],
  });
  await page.goto("/apps/agent-hub");
  await expect(page.getByRole("heading", { name: "Agent Hub" })).toBeVisible();
  await expect(page.getByRole("link", { name: /관리 설정 열기/ })).toHaveCount(
    0,
  );
});

test("관리자는 앱 관리에서 새 앱을 등록한다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/apps");
  await page.getByRole("link", { name: "앱 추가" }).click();
  await expect(page).toHaveURL("/admin/apps/new");
  await expect(page.getByRole("heading", { name: "앱 추가" })).toBeVisible();
  await expect(page.getByRole("button", { name: "앱 삭제" })).toHaveCount(0);

  await page.getByLabel("앱 이름").fill("새 앱");
  await page.getByLabel("Slug").fill("new-app");
  await page.getByLabel("한 줄 설명").fill("관리자가 직접 등록한 앱");
  await page.getByLabel("서비스 URL").fill("https://new.internal.example");
  await page.getByLabel("카테고리").selectOption({ index: 1 });
  await page.getByLabel("상세 설명").fill("설명입니다.");
  await page.getByRole("button", { name: "앱 등록" }).click();

  await expect(page).toHaveURL("/admin/apps/new-app-id");
});

test("빠른 이동 팔레트로 메뉴와 앱으로 이동한다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/");

  await page.keyboard.press("Control+k");
  const palette = page.getByRole("dialog", { name: "빠른 이동" });
  await expect(palette).toBeVisible();

  // Menus are listed before anything is typed.
  await expect(
    palette.getByRole("option", { name: /카테고리/ }).first(),
  ).toBeVisible();

  await palette.getByRole("combobox").fill("agent");
  await expect(
    palette.getByRole("option", { name: /Agent Hub/ }).first(),
  ).toBeVisible();
  // An administrator also gets the app's admin record.
  await expect(
    palette.getByRole("option", { name: /Agent Hub · 관리 설정/ }),
  ).toBeVisible();

  await page.keyboard.press("Enter");
  await expect(page).toHaveURL("/apps/agent-hub");
  await expect(palette).toHaveCount(0);
});

test("빠른 이동 팔레트는 esc로 닫히고 최근 항목을 기억한다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/");
  await page.getByRole("button", { name: /빠른 이동/ }).click();
  const palette = page.getByRole("dialog", { name: "빠른 이동" });
  await palette.getByRole("combobox").fill("카테고리");
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL("/categories");

  await page.keyboard.press("Control+k");
  await expect(
    page.getByRole("dialog", { name: "빠른 이동" }).getByText("최근 이동"),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "빠른 이동" })).toHaveCount(0);
});

test("관리자는 로고와 파비콘을 업로드하고 주소로 가져온다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/settings");
  await expect(
    page.getByRole("heading", { name: "시스템 설정" }),
  ).toBeVisible();

  const png = Buffer.from(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
    "base64",
  );
  const uploads: string[] = [];
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      request.url().includes("/api/v1/admin/branding/")
    ) {
      uploads.push(request.url());
    }
  });

  await page
    .getByLabel("로고 파일 선택")
    .setInputFiles({ name: "logo.png", mimeType: "image/png", buffer: png });
  await expect
    .poll(() => uploads.filter((url) => url.endsWith("/logo")).length)
    .toBe(1);

  await page
    .getByLabel("파비콘 이미지 주소")
    .fill("https://cdn.example.com/favicon.png");
  await page.getByRole("button", { name: "주소에서 가져오기" }).last().click();
  await expect
    .poll(() => uploads.filter((url) => url.endsWith("/favicon")).length)
    .toBe(1);
});

test("앱 관리 목록에서 서비스 주소로 바로 이동한다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/apps");
  const open = page.getByRole("link", { name: "Agent Hub 서비스 열기" });
  await expect(open).toHaveAttribute("href", "https://agent.internal.example");
  await expect(open).toHaveAttribute("target", "_blank");
});

test("첫 화면 배너는 관리자가 저장한 문구를 사용한다", async ({ page }) => {
  await installMockApi(page, {
    config: {
      heroEyebrow: "사내 서비스 카탈로그",
      heroTitle: "필요한 도구를\n바로 찾으세요.",
      siteDescription: "팀이 만든 서비스를 한 곳에 모았습니다.",
      heroPrimaryLabel: "지금 둘러보기",
    },
  });
  await page.goto("/");
  const title = page.getByRole("heading", { level: 1 });
  // The line break the administrator typed is kept as a real break.
  await expect(title).toContainText("필요한 도구를");
  await expect(title).toContainText("바로 찾으세요.");
  await expect(title.locator("br")).toHaveCount(1);
  await expect(page.getByText("사내 서비스 카탈로그")).toBeVisible();
  await expect(
    page.getByText("팀이 만든 서비스를 한 곳에 모았습니다."),
  ).toBeVisible();
  await expect(page.getByRole("link", { name: "지금 둘러보기" })).toBeVisible();
  // Wording the administrator left alone keeps the shipped default.
  await expect(page.getByRole("link", { name: "모든 앱 탐색" })).toBeVisible();
});

test("추천 진열은 최신순이 아니라 관리자가 정한 우선순위를 따른다", async ({
  page,
}) => {
  await installMockApi(page);
  await page.goto("/");
  const shelf = page.getByRole("region", { name: "에디터 추천" });
  // Flow Studio is the older app but carries priority 1, so it leads.
  const names = shelf.locator(".app-card-name");
  await expect(names.first()).toHaveText("Flow Studio");
  await expect(names.nth(1)).toHaveText("Agent Hub");
});

test("관리자는 앱 관리에서 추천 우선순위를 지정하거나 지운다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa");
  await expect(page.getByText("추천 2순위")).toBeVisible();
  const rank = page.getByLabel("추천 우선순위");
  await expect(rank).toHaveValue("2");

  const savedRank = async () => {
    const request = page.waitForRequest(
      (candidate) =>
        candidate.url().includes("/api/v1/admin/apps/") &&
        candidate.method() === "PUT",
    );
    await page.getByRole("button", { name: "변경 저장" }).click();
    return (await request).postDataJSON() as { featuredRank: number | null };
  };

  await rank.fill("1");
  expect((await savedRank()).featuredRank).toBe(1);

  // Clearing the box means "no priority", which the store reads as
  // most-recently-changed.
  await rank.fill("");
  expect((await savedRank()).featuredRank).toBeNull();
});

test("가이드 문서는 비로그인 방문자도 앱 상세에서 내려받는다", async ({
  page,
}) => {
  await installMockApi(page);
  await page.goto("/apps/agent-hub");
  await expect(
    page.getByRole("heading", { name: "가이드 문서" }),
  ).toBeVisible();
  const download = page.getByRole("link", { name: /내려받기/ });
  await expect(download).toHaveAttribute(
    "href",
    "/api/v1/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/documents/f1f1f1f1-f1f1-4f1f-8f1f-f1f1f1f1f1f1",
  );
  await expect(page.getByText("agent-hub-운영-가이드.pdf")).toBeVisible();
});

test("소유자는 앱 수정 화면에서 가이드 문서를 첨부하고 삭제를 예약한다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa");
  await expect(page.getByText("Agent Hub 운영 가이드")).toBeVisible();
  await page.getByLabel("가이드 문서 파일 선택").setInputFiles({
    name: "설치 안내.pdf",
    mimeType: "application/pdf",
    buffer: Buffer.from("%PDF-1.4 install guide"),
  });
  await expect(page.getByText("저장 시 업로드됩니다")).toBeVisible();
  await page
    .getByRole("button", { name: "삭제", exact: false })
    .first()
    .click();
  await expect(page.getByText("저장 시 삭제됩니다")).toBeVisible();
  await page.getByRole("button", { name: "변경 저장" }).click();
  await expect(page.getByText("앱 정보가 저장되었습니다.")).toBeVisible();
});

test("앱 카드는 이름 밖을 눌러도 앱 상세로 이동한다", async ({ page }) => {
  await installMockApi(page);
  await page.goto("/apps");
  const card = page.locator(".app-card").filter({ hasText: "Agent Hub" });
  // The card's own padding: no text, no button — only the overlay the app name
  // link draws over the card.
  await card.first().click({ position: { x: 8, y: 8 } });
  await expect(page).toHaveURL(/\/apps\/agent-hub$/);
  await expect(page.getByRole("heading", { name: "Agent Hub" })).toBeVisible();
});

test("게시되지 않은 내 앱의 카드는 수정 화면으로 열린다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  const saved = ' [ "release-radar", "unrelated-app" ] ';
  await page.goto("/my/apps");
  await page.evaluate(
    (value) => localStorage.setItem("appstore.favorites", value),
    saved,
  );
  // Both owner pages use the production AppCard and FavoritesProvider.
  const card = page.locator(".app-card").filter({ hasText: "Release Radar" });
  for (const path of ["/my", "/my/apps"]) {
    await page.goto(path);
    await expect(card).toBeVisible();
    await expect(card.getByRole("button", { name: /즐겨찾기/ })).toHaveCount(0);
    await expect(card.getByText("반려", { exact: true })).toBeVisible();
    await expect(
      card.getByRole("link", { name: "Release Radar 상세 보기" }),
    ).toHaveCount(0);
    await expect(
      card.getByRole("link", { name: "Release Radar 관리 설정 열기" }),
    ).toBeVisible();
    expect(
      await page.evaluate(() => localStorage.getItem("appstore.favorites")),
    ).toBe(saved);
  }
  await card.getByRole("link", { name: "Release Radar", exact: true }).click();
  await expect(page).toHaveURL(
    /\/my\/apps\/ffffffff-ffff-4fff-8fff-ffffffffffff\/edit$/,
  );
  await expect(page.getByRole("heading", { name: "앱 수정" })).toBeVisible();
});

test("관리자의 앱 카드는 자세히 글자 없이 아이콘만 보여 준다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/apps");
  const card = page.locator(".app-card").filter({ hasText: "Agent Hub" });
  await expect(
    card.getByRole("link", { name: "Agent Hub 상세 보기" }),
  ).toHaveText("");
  await expect(
    card.getByRole("link", { name: "Agent Hub 관리 설정 열기" }),
  ).toBeVisible();
  await expect(card.getByText("자세히")).toHaveCount(0);
});

test("보안 심의를 마쳐야 앱을 제출할 수 있다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/my/apps/ffffffff-ffff-4fff-8fff-ffffffffffff/security");
  await expect(page.getByRole("heading", { name: "보안 심의" })).toBeVisible();
  // The block an owner pastes into SecCheck, and no way past it yet.
  await expect(page.getByText("[APPSTORE-SECURITY-CHECK:v1]")).toBeVisible();
  await expect(page.getByRole("button", { name: /검토 제출/ })).toHaveCount(0);

  await page.getByLabel("심의 ID").fill("77777777-7777-4777-8777-777777777777");
  await page.getByRole("button", { name: /심의 결과 확인/ }).click();
  await expect(page.getByText(/보안 심의가 확인되었습니다/)).toBeVisible();
  await expect(page.getByRole("button", { name: /검토 제출/ })).toBeVisible();
});

test("관리자는 SecCheck 연동을 설정하고 연결을 확인한다", async ({ page }) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/admin/security-check");
  await expect(
    page.getByRole("heading", { name: "보안 심의 연동" }),
  ).toBeVisible();
  await expect(page.getByLabel("SecCheck 주소")).toHaveValue(
    "https://seccheck.internal.example",
  );
  // The stored key is never sent back to the screen.
  await expect(page.getByLabel("API Key")).toHaveValue("");
  await page.getByRole("button", { name: "연결 테스트" }).click();
  await expect(page.getByText(/SecCheck에 연결했습니다/)).toBeVisible();
});

test("검토자는 앱 내용과 이전 검토를 보고 의견과 함께 결정한다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  await page.goto("/review/dddddddd-dddd-4ddd-8ddd-dddddddddddd");
  // The app's name is both the page title and the heading of its card, so the
  // page's own heading is the one to look at.
  await expect(
    page.getByRole("heading", { name: "Flow Studio", level: 1 }),
  ).toBeVisible();
  // What the decision rests on: the app itself, its guide, its security state
  // and what the last reviewer said.
  await expect(
    page.getByText("조직의 API와 이벤트를 연결해 업무 흐름을 구성합니다."),
  ).toBeVisible();
  await expect(page.getByText("Flow Studio 운영 가이드")).toBeVisible();
  await expect(page.getByText("보안 심의 완료")).toBeVisible();
  await expect(page.getByText(/사내망에서 열리지 않았습니다/)).toBeVisible();

  // A rejection needs a note; an approval may carry one.
  await page.getByRole("button", { name: /반려/ }).click();
  await expect(page.getByText(/사유를 입력하세요/)).toBeVisible();
  await page.getByLabel("검토 의견").fill("사내망 접속을 확인했습니다.");
  await page.getByRole("button", { name: /승인 및 게시/ }).click();
  await expect(page).toHaveURL(/\/review$/);
});

test("공개 앱 즐겨찾기는 소유자와 공개 목록에서 추가하고 즐겨찾기에서 해제한다", async ({
  page,
}) => {
  await installMockApi(page, { authenticated: true });
  for (const path of ["/my/apps", "/my", "/apps"]) {
    await page.goto(path);
    const add = page.getByRole("button", { name: "Agent Hub 즐겨찾기 추가" });
    await expect(add).toHaveAttribute("aria-pressed", "false");
    await add.click();
    await expect(
      page.getByRole("button", { name: "Agent Hub 즐겨찾기 해제" }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      await page.evaluate(() =>
        JSON.parse(localStorage.getItem("appstore.favorites")!),
      ),
    ).toEqual(["agent-hub"]);
    await page.getByRole("link", { name: "즐겨찾기", exact: true }).click();
    await expect(page).toHaveURL("/favorites");
    await expect(
      page.getByRole("heading", { name: "Agent Hub" }),
    ).toBeVisible();
    const remove = page.getByRole("button", {
      name: "Agent Hub 즐겨찾기 해제",
    });
    await expect(remove).toHaveAttribute("aria-pressed", "true");
    await remove.click();
    await expect(page.getByRole("heading", { name: "Agent Hub" })).toHaveCount(
      0,
    );
    expect(
      await page.evaluate(() =>
        JSON.parse(localStorage.getItem("appstore.favorites")!),
      ),
    ).toEqual([]);
    await page.reload();
    await expect(
      page.getByRole("heading", { name: "즐겨찾기한 앱이 없습니다" }),
    ).toBeVisible();
  }
});
