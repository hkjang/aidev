import { expect, type Page } from "@playwright/test";

/** 시드가 만드는 층. 검증은 이 층의 좌석을 기준으로 한다. */
export const SEED_FLOOR = "3층";

const USERNAME = process.env.E2E_USERNAME ?? "admin";
const PASSWORD = process.env.E2E_PASSWORD ?? "e2e-verify-pass-123";

/**
 * 이미 열려 있는 로그인 화면의 입력란을 채워 보낸다. 깊은 링크로 밀려나 온
 * 로그인 화면은 주소에 returnTo 를 달고 있으므로 /login 으로 다시 가면 안 된다.
 */
export const submitLoginForm = async (page: Page) => {
  await page.fill('input[autocomplete="username"]', USERNAME);
  await page.fill('input[autocomplete="current-password"]', PASSWORD);
  await page.click('button[type="submit"]');
};

export const login = async (page: Page) => {
  await page.goto("/login");
  await submitLoginForm(page);
  await page.waitForURL((url) => !url.pathname.startsWith("/login"));
};

export const mapCanvas = (page: Page) =>
  page.locator("svg[aria-label*='좌석 배치도']");

export const seatLabels = async (page: Page) =>
  mapCanvas(page).locator("text").allTextContents();

/**
 * 좌석 목록을 API로 읽는다. 화면을 거치지 않으므로 검증 전후 상태를 확인하고
 * 되돌리는 데 쓴다.
 */
export const fetchSeats = async (page: Page) => {
  const maps = await (await page.request.get("/api/v1/floor-maps")).json();
  // 도면 버전은 여러 개이고 층도 여러 개일 수 있다. 검증이 다루는 것은 시드가
  // 만든 층의 게시된 버전이므로, 그 층을 이름으로 집어 고른다. 그러지 않으면
  // 나중에 추가된 다른 층의 좌석을 읽어 "김개발이 없다"는 식으로 깨진다.
  type MapItem = { id: string; active?: boolean; floorName?: string };
  const items = maps.items as MapItem[];
  const map =
    items.find((item) => item.active && item.floorName === SEED_FLOOR) ??
    items.find((item) => item.active) ??
    items[0];
  const mapId = map.id;
  const seats = await (
    await page.request.get(`/api/v1/seats?floorMapId=${mapId}`)
  ).json();
  return seats.items as Array<{
    id: string;
    seatNo: string;
    x: number;
    y: number;
    employeeName?: string;
  }>;
};

/**
 * 좌석 위치를 원래대로 되돌린다. 배치 편집 검증은 서버 상태를 실제로 바꾸므로,
 * 되돌리지 않으면 반복 실행할수록 좌석이 밀려 다음 실행이 깨진다.
 */
export const restoreSeat = async (
  page: Page,
  id: string,
  at: { x: number; y: number },
) => {
  const me = await (await page.request.get("/api/v1/auth/me")).json();
  await page.request.patch(`/api/v1/seats/${id}`, {
    headers: { "X-CSRF-Token": me.csrfToken },
    data: { x: at.x, y: at.y },
  });
};

/** 로그인한 세션의 CSRF 토큰. 변경 요청에 필요하다. */
export const csrfToken = async (page: Page) =>
  (await (await page.request.get("/api/v1/auth/me")).json())
    .csrfToken as string;

export const employeeId = async (page: Page, name: string) => {
  const found = await (
    await page.request.get(`/api/v1/employees?q=${encodeURIComponent(name)}`)
  ).json();
  return found.items[0].id as string;
};

/**
 * 직원의 좌석을 건드리는 검증을 감싼다.
 *
 * 배정을 바꾸는 검증이 원래대로 돌려놓지 않으면, 다음 검증이 "누가 어디 앉아
 * 있다"는 전제 위에서 엉뚱한 이유로 깨진다. 실제로 한 검증이 옮겨 놓은 직원
 * 때문에 다른 세 검증이 무너졌다.
 */
export const keepingSeats = async (
  page: Page,
  names: string[],
  body: () => Promise<void>,
) => {
  const before = await fetchSeats(page);
  const home = new Map(
    names.map((name) => [
      name,
      before.find((seat) => seat.employeeName === name)?.id ?? null,
    ]),
  );
  try {
    await body();
  } finally {
    const token = await csrfToken(page);
    const after = await fetchSeats(page);
    for (const [name, seatId] of home) {
      const now = after.find((seat) => seat.employeeName === name);
      if (now?.id === seatId) continue;
      if (now) {
        await page.request.delete(`/api/v1/seat-assignments/${now.id}`, {
          headers: { "X-CSRF-Token": token },
        });
      }
      if (seatId) {
        await page.request.post("/api/v1/seat-assignments", {
          headers: { "X-CSRF-Token": token },
          data: {
            employeeId: await employeeId(page, name),
            seatId,
            source: "manual",
            reason: "검증 정리",
          },
        });
      }
    }
  }
};

/**
 * 브라우저 콘솔 오류를 모은다. 타입 검사와 단위 테스트로는 드러나지 않는
 * 런타임 오류를 화면마다 붙잡는 용도다.
 */
export const watchConsole = (page: Page) => {
  const problems: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") problems.push(message.text());
  });
  page.on("pageerror", (error) => problems.push(String(error)));
  return () => problems.filter((p) => !p.includes("401"));
};

export const expectNoConsoleErrors = async (page: Page, path: string) => {
  const problems = watchConsole(page);
  await page.goto(path);
  await expect(page.locator("main, [role=main]").first()).toBeVisible();
  expect(problems()).toHaveLength(0);
};
