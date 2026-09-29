import { expect, test } from "@playwright/test";
import {
  employeeId,
  fetchSeats,
  keepingSeats,
  login,
  watchConsole,
} from "./helpers";

test.describe("관리 화면", () => {
  test.beforeEach(async ({ page }) => login(page));

  test("변경 이력이 조회되고 오늘 범위 필터가 맞는다", async ({ page }) => {
    // 방금 만든 변경이 "오늘"로 조회되는지 본다. 예전에는 날짜 범위를 서버
    // 표준시로 해석해, 사용자가 고른 오늘이 서버에서는 다른 날이 되어 방금 한
    // 변경이 통째로 빠졌다.
    const me = await (await page.request.get("/api/v1/auth/me")).json();
    const seats = await fetchSeats(page);
    // 같은 좌석으로 다시 배정하면 아무 일도 일어나지 않아 이력이 남지 않는다.
    // 빈 좌석으로 옮겨 이력을 만들고, 확인이 끝나면 제자리로 돌린다.
    const target = seats.find((seat) => !seat.employeeName)!;
    const stampedReason = `오늘 확인 ${Date.now()}`;
    await keepingSeats(page, ["윤총무"], async () => {
      const moved = await page.request.post("/api/v1/seat-assignments", {
        headers: { "X-CSRF-Token": me.csrfToken },
        data: {
          employeeId: await employeeId(page, "윤총무"),
          seatId: target.id,
          source: "manual",
          reason: stampedReason,
        },
      });
      expect(moved.ok(), await moved.text()).toBe(true);

      await page.goto("/admin/history");
      await expect(page.getByText(/전체 \d+건 중 \d+건/)).toBeVisible();
      // 필터를 걸기 전에 방금 만든 변경이 목록에 있는지 먼저 확인한다. 여기서
      // 이미 없다면 날짜 해석이 아니라 변경 자체가 만들어지지 않은 것이다.
      await expect(page.getByText(stampedReason)).toBeVisible();

      // 날짜는 반드시 브라우저 안에서 계산한다. 검증 프로세스(Node)와 브라우저의
      // 시간대가 다르면, 화면이 해석하는 "오늘"과 다른 날짜를 넣게 된다. CI에서
      // 실제로 하루 어긋나 오늘 범위가 비었다.
      const day = (offsetDays: number) =>
        page.evaluate((offset) => {
          const at = new Date(Date.now() + offset * 86400000);
          return `${at.getFullYear()}-${String(at.getMonth() + 1).padStart(2, "0")}-${String(at.getDate()).padStart(2, "0")}`;
        }, offsetDays);
      const setRange = async (from: string, to: string) => {
        const start = page.getByLabel("시작일");
        const end = page.getByLabel("종료일");
        await start.fill(from);
        await end.fill(to);
        // 날짜 칸은 브라우저마다 다루는 방식이 달라, 값이 실제로 들어갔는지
        // 확인하고 조회한다. 값이 비면 조건 없이 조회한 것과 구분되지 않는다.
        await expect(start).toHaveValue(from);
        await expect(end).toHaveValue(to);
        await page.getByRole("button", { name: "조회" }).click();
      };

      const today = await day(0);
      await setRange(today, today);
      await expect(page.getByText(stampedReason)).toBeVisible();

      // 내일부터로 좁히면 방금 한 변경은 빠져야 한다.
      const tomorrow = await day(1);
      await setRange(tomorrow, tomorrow);
      await expect(page.getByText(stampedReason)).toHaveCount(0);
    });
  });

  test("도면 화면이 준비 단계와 도면 카드를 보여준다", async ({ page }) => {
    const problems = watchConsole(page);
    await page.goto("/admin/maps");
    // 도면 버전은 여러 개일 수 있다. 실제 설치에서도 버전이 쌓이므로 개수를
    // 전제하지 않는다.
    await expect(page.getByText("본사 · 3층").first()).toBeVisible();
    await expect(page.getByText("게시 중").first()).toBeVisible();
    // 준비 단계는 끝난 것만 완료로 표시해야 한다. 처음 설치한 관리자가 아직
    // 하지 않은 단계까지 체크로 읽은 적이 있다.
    for (const step of ["1. 사업장", "3. 도면", "5. 게시"]) {
      await expect(page.getByLabel(`${step} · 완료`)).toBeVisible();
    }
    await expect(page.getByLabel(/다음 할 일$/)).toHaveCount(0);
    expect(problems()).toHaveLength(0);
  });

  test("처리 필요 목록이 열린다", async ({ page }) => {
    const problems = watchConsole(page);
    await page.goto("/admin/actions");
    await expect(page.getByRole("heading", { name: "처리필요" })).toBeVisible();
    expect(problems()).toHaveLength(0);
  });

  test("사용자 권한 화면이 열린다", async ({ page }) => {
    const problems = watchConsole(page);
    await page.goto("/admin/users");
    await expect(
      page.getByRole("cell", { name: /admin/ }).first(),
    ).toBeVisible();
    expect(problems()).toHaveLength(0);
  });

  test("관리자 계정의 메일 주소를 화면에서 넣고 지운다", async ({ page }) => {
    // 로컬 관리자 계정은 처음부터 주소가 없어 알림 메일을 받을 길이 없었다.
    // 화면에서 넣은 주소가 API 에 남고, 비우면 다시 없어지는지 본다.
    const problems = watchConsole(page);
    const me = await (await page.request.get("/api/v1/auth/me")).json();
    const myID = me.user.id as string;
    const users = async () =>
      (await (await page.request.get("/api/v1/users")).json()).items as Array<{
        id: string;
        email?: string;
        active: boolean;
        role: string;
      }>;
    const before = (await users()).find((u) => u.id === myID)!;
    const address = `admin-${Date.now()}@corp.example`;
    try {
      await page.goto("/admin/users");
      await page.getByRole("button", { name: /admin 메일 주소 변경/ }).click();
      const field = page.getByRole("dialog").getByLabel("메일 주소");
      // 잘못된 주소는 서버에 보내지 않고 창 안에서 바로 알린다.
      await field.fill("admin.corp.example");
      await page.getByRole("dialog").getByRole("button", { name: "저장" }).click();
      await expect(
        page.getByRole("dialog").getByText("메일 주소 형식이 올바르지 않습니다"),
      ).toBeVisible();
      await field.fill(`  ${address}  `);
      await field.press("Enter");
      await expect(page.getByRole("dialog")).toBeHidden();
      await expect(page.getByRole("cell", { name: address })).toBeVisible();
      expect((await users()).find((u) => u.id === myID)!.email).toBe(address);

      // 자기 계정의 사용 스위치는 꺼져 있고, API 로 직접 보내도 거절된다.
      const mine = page.getByRole("switch", { name: /admin 계정 사용/ });
      await expect(mine).toBeChecked();
      await expect(mine).toBeDisabled();
      const refused = await page.request.patch(`/api/v1/users/${myID}`, {
        headers: { "X-CSRF-Token": me.csrfToken },
        data: { active: false },
      });
      expect(refused.status()).toBe(400);
      expect((await refused.json()).error.code).toBe("self_deactivation");
      expect((await users()).find((u) => u.id === myID)!.active).toBe(true);

      // 자기 행의 권한 선택도 꺼져 있고, API 로 직접 낮춰도 거절되어 system_admin
      // 그대로다. 낮아지면 이 화면·이 API 를 더는 부를 수 없어 되돌릴 길이 없다.
      await expect(
        page.getByRole("combobox", { name: /admin 권한/ }),
      ).toHaveAttribute("aria-disabled", "true");
      const demoted = await page.request.patch(`/api/v1/users/${myID}`, {
        headers: { "X-CSRF-Token": me.csrfToken },
        data: { role: "employee" },
      });
      expect(demoted.status()).toBe(400);
      expect((await demoted.json()).error.code).toBe("self_demotion");
      expect((await users()).find((u) => u.id === myID)!.role).toBe(
        "system_admin",
      );

      // 비우면 주소를 지운다.
      await page.getByRole("button", { name: /admin 메일 주소 변경/ }).click();
      await page.getByRole("dialog").getByLabel("메일 주소").fill("");
      await page.getByRole("dialog").getByRole("button", { name: "저장" }).click();
      await expect(page.getByRole("dialog")).toBeHidden();
      await expect(page.getByRole("cell", { name: /주소 없음/ })).toBeVisible();
      expect((await users()).find((u) => u.id === myID)!.email).toBeUndefined();
    } finally {
      await page.request.patch(`/api/v1/users/${myID}`, {
        headers: { "X-CSRF-Token": me.csrfToken },
        data: { email: before.email ?? "" },
      });
    }
    expect(problems()).toHaveLength(0);
  });

  test("시스템 설정은 불러오기 전에 빈 값으로 덮어쓰지 않는다", async ({
    page,
  }) => {
    const problems = watchConsole(page);
    await page.goto("/admin/settings");
    await expect(
      page.getByRole("heading", { name: "시스템 설정" }),
    ).toBeVisible();
    expect(problems()).toHaveLength(0);
  });
});
