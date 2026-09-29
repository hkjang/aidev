/**
 * 가이드 문서용 화면 캡처.
 *
 * 실제로 띄운 SeatOn 을 headless Chromium 으로 열어 docs/assets/guide/*.png 를
 * 만든다. 목업이나 자리표시자를 쓰지 않기 위해 시드 데이터(seed.mjs)가 채운
 * 화면만 찍는다. 시드가 없으면 먼저 넣고, 이미 있으면 그대로 둔다.
 *
 * 대상 주소와 계정은 캡처 전용 환경 변수로만 받는다. e2e 와 변수를 공유하면
 * 다른 검증이 가리키는 배포를 실수로 찍게 되므로 값이 없으면 바로 멈춘다.
 *
 *   GUIDE_SHOT_BASE_URL=http://127.0.0.1:18790 \
 *   GUIDE_SHOT_USERNAME=admin GUIDE_SHOT_PASSWORD=... node e2e/guide-shots.mjs
 *
 * 화면을 위해 만드는 것은 API 키 하나뿐이며, 끝나면 폐기한다. 전역 설정은
 * 읽기만 하고 바꾸지 않는다.
 */
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";
import { seed } from "./seed.mjs";

const baseURL = process.env.GUIDE_SHOT_BASE_URL;
const username = process.env.GUIDE_SHOT_USERNAME;
const password = process.env.GUIDE_SHOT_PASSWORD;
if (!baseURL || !username || !password) {
  console.error(
    "GUIDE_SHOT_BASE_URL, GUIDE_SHOT_USERNAME, GUIDE_SHOT_PASSWORD 를 모두 지정하세요.",
  );
  process.exit(2);
}
// 버려도 되는 배포만 가리키게 한다. 운영 주소를 찍는 일은 없어야 한다.
if (!/^https?:\/\/(127\.0\.0\.1|localhost)(:\d+)?$/.test(baseURL)) {
  console.error(`GUIDE_SHOT_BASE_URL 은 로컬 주소여야 합니다: ${baseURL}`);
  process.exit(2);
}

const outDir = fileURLToPath(new URL("../../docs/assets/guide/", import.meta.url));
await mkdir(outDir, { recursive: true });

console.log("[guide] 시드", JSON.stringify(await seed({ baseURL, username, password })));

const browser = await chromium.launch();
const context = await browser.newContext({
  baseURL,
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 1,
  locale: "ko-KR",
  timezoneId: "Asia/Seoul",
});
const page = await context.newPage();

const settle = async () => {
  await page.waitForLoadState("networkidle");
  // MUI 전환 애니메이션이 끝난 뒤 찍는다.
  await page.waitForTimeout(400);
};
const shot = async (name, options = {}) => {
  await settle();
  await page.screenshot({ path: `${outDir}${name}.png`, ...options });
  console.log(`[guide] ${name}.png`);
};
const csrf = async () =>
  (await (await page.request.get("/api/v1/auth/me")).json()).csrfToken;

// 로그인
await page.goto("/login");
await page.fill('input[autocomplete="username"]', username);
await page.fill('input[autocomplete="current-password"]', password);
// 보호된 화면에서 튕겨 온 안내가 떠 있으면 닫고 찍는다.
const notice = page.getByRole("alert").getByRole("button");
if (await notice.count()) await notice.first().click();
await shot("login");
await page.click('button[type="submit"]');
await page.waitForURL((url) => !url.pathname.startsWith("/login"));

// 좌석맵 — 기본 상태
await page.goto("/");
await page.locator("svg[aria-label*='좌석 배치도']").waitFor();
await shot("seatmap");

// 좌석맵 — 검색으로 직원 찾기
const search = page.getByPlaceholder(/이름|사번/).first();
await search.fill("김개발");
await page.waitForTimeout(500);
await page.keyboard.press("Enter");
await page.waitForTimeout(800);
await shot("seatmap-search");

// 좌석맵 — 검토 필요 필터
const fullView = async () => {
  await page.getByRole("button", { name: /전체 보기/ }).click();
  await page.mouse.move(1300, 700);
};
await fullView();
await page
  .getByRole("group", { name: "좌석 필터" })
  .getByRole("button", { name: "검토 필요", exact: true })
  .click();
await shot("seatmap-filter-review");
await page
  .getByRole("group", { name: "좌석 필터" })
  .getByRole("button", { name: "검토 필요", exact: true })
  .click();

// 좌석맵 — 조직 색 + 구역
await page.getByRole("button", { name: "조직 색", exact: true }).click();
await page
  .getByRole("button", { name: "좌석에 지정된 조직 구역을 배경으로 표시" })
  .click();
await page.mouse.move(1300, 700);
await shot("seatmap-org-color");
await page.getByRole("button", { name: "상태 색", exact: true }).click();

// 좌석맵 — 배치 편집 모드
await page.getByRole("button", { name: /배치 편집/ }).click();
await page.mouse.move(1300, 700);
await shot("seatmap-edit");
await page.getByRole("button", { name: /편집 완료/ }).click();

// 프로필 메뉴 — 내 API 키와 빌드 버전이 여기에 있다.
await page.getByRole("button", { name: /admin|프로필|계정/ }).last().click();
await page.getByRole("menuitem", { name: "내 API 키" }).waitFor();
await shot("profile-menu");
await page.keyboard.press("Escape");

// 처리필요
await page.goto("/admin/actions");
await page.getByRole("heading", { name: "처리필요" }).waitFor();
await shot("admin-actions");
// 작업 큐와 운영 준비도가 보이도록 아래로 내린다.
await page.mouse.move(840, 600);
await page.mouse.wheel(0, 600);
await page.waitForTimeout(300);
await shot("admin-actions-queue");

// 도면 · 좌석
await page.goto("/admin/maps");
await page.getByText("본사 · 3층").first().waitFor();
await shot("admin-maps");
await page.getByRole("button", { name: "사업장", exact: true }).click();
await page.getByRole("dialog").waitFor();
await shot("admin-maps-building");
await page.keyboard.press("Escape");
await page.getByRole("dialog").waitFor({ state: "hidden" });
await page.getByRole("button", { name: /도면 업로드/ }).first().click();
await page.getByRole("dialog").waitFor();
await shot("admin-maps-upload");
await page.keyboard.press("Escape");

// 직원
await page.goto("/admin/employees");
await page.getByRole("heading", { name: "직원" }).waitFor();
await shot("admin-employees");

// 변경 이력
await page.goto("/admin/history");
await page.getByText(/전체 \d+건 중 \d+건/).waitFor();
await shot("admin-history");

// 사용자 권한
await page.goto("/admin/users");
await page.getByRole("cell", { name: /admin/ }).first().waitFor();
await shot("admin-users");

// 시스템 설정
await page.goto("/admin/settings");
await page.getByRole("heading", { name: "시스템 설정" }).waitFor();
await shot("admin-settings");
await page.getByRole("tab", { name: "Keycloak SSO" }).click();
await shot("admin-settings-sso");
// MCP SSO(OAuth) 카드는 같은 탭 아래쪽에 있다 — 카드만 잘라 찍는다.
await page.getByTestId("mcp-oauth-card").scrollIntoViewIfNeeded();
await settle();
await page
  .getByTestId("mcp-oauth-card")
  .screenshot({ path: `${outDir}admin-settings-mcp-sso.png` });
console.log("[guide] admin-settings-mcp-sso.png");
await page.getByRole("tab", { name: "보안 · 키" }).click();
await shot("admin-settings-security");
await page.getByRole("tab", { name: "AI 분석" }).click();
await shot("admin-settings-ai");
await page.getByRole("tab", { name: "방문 추적" }).click();
await page.getByRole("heading", { name: "정책이 차단한 출처" }).waitFor();
await shot("admin-settings-tracking");

// 내 API 키 — 만들고, 원문 화면을 찍고, 폐기한다.
await page.goto("/profile/keys");
await page.getByRole("heading", { name: "내 API 키" }).waitFor();
await page.getByRole("button", { name: /키 만들기/ }).first().click();
const dialog = page.getByRole("dialog");
await dialog.getByLabel("키 이름").fill("사내 AI 비서");
await shot("profile-keys-create");
await dialog.getByRole("button", { name: "생성" }).click();
const secret = page.getByText(/^seat_[A-Za-z0-9_-]{20,}$/);
await secret.waitFor();
// 문서에 실제 키 원문이 남지 않도록 화면의 글자만 가린다. 키는 아래에서 폐기한다.
await secret.evaluate((node) => {
  node.textContent = "seat_" + "•".repeat(38);
});
await shot("profile-keys-created");
await page.getByRole("button", { name: "보관 완료" }).click();
await shot("profile-keys");
const keys = await (await page.request.get("/api/v1/api-keys")).json();
for (const key of keys.items.filter((k) => k.name === "사내 AI 비서")) {
  await page.request.delete(`/api/v1/api-keys/${key.id}`, {
    headers: { "X-CSRF-Token": await csrf() },
  });
}

await browser.close();
