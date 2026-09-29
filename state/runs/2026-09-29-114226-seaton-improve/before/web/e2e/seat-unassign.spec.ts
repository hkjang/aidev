import { expect, test, type Page } from "@playwright/test";
import { csrfToken, keepingSeats, login, mapCanvas } from "./helpers";

/** 해제 검증의 기준이 되는 직원. 시드가 자리를 잡아 준다. */
const PERSON = "김개발";

type Seat = {
  id: string;
  seatNo: string;
  employeeId?: string;
  employeeName?: string;
};

const seats = async (page: Page): Promise<Seat[]> => {
  const response = await page.request.get("/api/v1/seats");
  expect(response.ok()).toBeTruthy();
  return (await response.json()).items;
};
const shape = (page: Page, id: string) =>
  mapCanvas(page).locator(`[data-seat-id="${id}"]`);
/** 상세 패널. 도면 위의 좌석 라벨과 같은 글자가 겹치므로 여기로 좁혀 본다. */
const detail = (page: Page) =>
  page.getByRole("region", { name: "좌석 상세" });
const unassignButton = (page: Page) =>
  detail(page).getByRole("button", { name: "배정 해제", exact: true });
const removeButton = (page: Page) =>
  detail(page).getByRole("button", { name: "삭제", exact: true });

test.beforeEach(async ({ page }) => {
  await login(page);
  await mapCanvas(page).waitFor();
});

test("배정된 좌석을 화면에서 해제하면 빈 좌석이 되고 상세는 열린 채 남는다", async ({
  page,
}) => {
  await keepingSeats(page, [PERSON], async () => {
    const before = (await seats(page)).find((s) => s.employeeName === PERSON)!;
    expect(before.employeeId).toBeTruthy();

    // (a) 해제 전에는 상세에 직원 이름이 보인다. 편집 모드를 켜지 않아도
    // 배정은 다룰 수 있어야 하므로 그대로 좌석을 고른다.
    await shape(page, before.id).click();
    await expect(detail(page).getByText(PERSON, { exact: true })).toBeVisible();
    await expect(
      detail(page).getByText("배정됨", { exact: true }),
    ).toBeVisible();

    // (b) 확인 창을 승인하면 DELETE 가 나가고 204 로 끝난다.
    page.once("dialog", (dialog) => {
      expect(dialog.message()).toContain(before.seatNo);
      void dialog.accept();
    });
    const call = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname ===
          `/api/v1/seat-assignments/${before.id}` &&
        r.request().method() === "DELETE",
    );
    await unassignButton(page).click();
    expect((await call).status()).toBe(204);

    // 서버가 실제로 배정을 지웠다.
    await expect
      .poll(async () => {
        const now = (await seats(page)).find((s) => s.id === before.id)!;
        return [now.employeeId ?? null, now.employeeName ?? null];
      })
      .toEqual([null, null]);
    expect((await seats(page)).some((s) => s.employeeName === PERSON)).toBe(
      false,
    );

    // (c) 상세 패널은 닫히지 않고 같은 좌석을 빈 좌석으로 보여 준다.
    await expect(
      detail(page).getByText("빈 좌석", { exact: true }),
    ).toBeVisible();
    await expect(
      detail(page).getByText("배정된 직원이 없습니다."),
    ).toBeVisible();
    await expect(
      detail(page).getByText(before.seatNo, { exact: true }),
    ).toBeVisible();
    await expect(detail(page).getByText(PERSON)).toHaveCount(0);
    await expect(unassignButton(page)).toHaveCount(0);

    // (d) 배정된 동안 막혀 있던 삭제가 풀린다 = 막다른 길이 사라졌다.
    await page.getByRole("button", { name: "배치 편집" }).click();
    await expect(removeButton(page)).toBeEnabled();
  });

  // (5) keepingSeats 가 자리를 돌려놓았는지 확인한다.
  await expect
    .poll(async () =>
      (await seats(page)).some((s) => s.employeeName === PERSON),
    )
    .toBe(true);
});

test("이미 해제된 좌석을 해제하면 서버 오류가 화면에 보인다", async ({
  page,
}) => {
  await keepingSeats(page, [PERSON], async () => {
    const before = (await seats(page)).find((s) => s.employeeName === PERSON)!;
    await shape(page, before.id).click();
    await expect(unassignButton(page)).toBeVisible();

    // 다른 관리자가 먼저 해제한 상황. 화면은 아직 배정된 좌석을 들고 있다.
    expect(
      (
        await page.request.delete(`/api/v1/seat-assignments/${before.id}`, {
          headers: { "X-CSRF-Token": await csrfToken(page) },
        })
      ).status(),
    ).toBe(204);

    page.once("dialog", (dialog) => void dialog.accept());
    const call = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname ===
          `/api/v1/seat-assignments/${before.id}` &&
        r.request().method() === "DELETE",
    );
    await unassignButton(page).click();
    expect((await call).status()).toBe(404);
    await expect(
      page.getByRole("alert").filter({ hasText: "배정된 직원이 없습니다" }),
    ).toBeVisible();
  });
});

test("빈 좌석 상세에는 배정 해제 단추가 없다", async ({ page }) => {
  const empty = (await seats(page)).find((s) => !s.employeeId)!;
  expect(empty).toBeTruthy();
  await shape(page, empty.id).click();
  await expect(
    detail(page).getByText("배정된 직원이 없습니다."),
  ).toBeVisible();
  await expect(unassignButton(page)).toHaveCount(0);
});
