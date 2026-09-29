import { expect, test, type Page } from "@playwright/test";
import { fetchSeats, login, mapCanvas, restoreSeat } from "./helpers";

/**
 * 좌석 사각형의 도면 좌표. 화면 좌표는 편집 모드에서 도구 줄이 늘어나며 함께
 * 움직이므로, 레이아웃과 무관한 이 값으로 이동을 확인한다.
 */
const seatAt = (page: Page, name: string) =>
  page.evaluate((label) => {
    const svg = document.querySelector("svg[aria-label*='좌석 배치도']")!;
    const group = Array.from(svg.querySelectorAll("g")).find(
      (node) => node.querySelector("text")?.textContent === label,
    )!;
    // 선택된 좌석은 앞에 강조 테두리가 하나 더 붙는다. 좌석 본체는 마지막 것.
    const rects = Array.from(group.querySelectorAll("rect"));
    const rect = rects[rects.length - 1];
    return {
      x: Number(rect.getAttribute("x")),
      y: Number(rect.getAttribute("y")),
    };
  }, name);

test.describe("좌석 배치 편집", () => {
  // 이 검증은 좌석 위치를 실제로 바꾼다. 되돌리지 않으면 반복 실행할수록 좌석이
  // 한쪽으로 밀려 서로 겹치고, 다음 실행이 엉뚱한 이유로 깨진다.
  let origin: Array<{ id: string; x: number; y: number }> = [];

  test.beforeEach(async ({ page }) => {
    await login(page);
    origin = (await fetchSeats(page)).map((seat) => ({
      id: seat.id,
      x: seat.x,
      y: seat.y,
    }));
    await mapCanvas(page).waitFor();
    await page.getByRole("button", { name: /배치 편집/ }).click();
  });

  test.afterEach(async ({ page }) => {
    const now = await fetchSeats(page);
    for (const seat of now) {
      const was = origin.find((item) => item.id === seat.id);
      if (was && (was.x !== seat.x || was.y !== seat.y)) {
        await restoreSeat(page, seat.id, was);
      }
    }
  });

  test("방향키로 옮긴 자리가 서버에 남는다", async ({ page }) => {
    const before = await seatAt(page, "김개발");
    await select(page, "김개발");
    await page.keyboard.press("ArrowRight");
    await expect
      .poll(async () => (await seatAt(page, "김개발")).x)
      .toBeGreaterThan(before.x);
    const moved = await seatAt(page, "김개발");

    // 화면에서만 움직이고 서버에 남지 않으면 다음 접속에서 제자리로 돌아간다.
    await page.reload();
    await mapCanvas(page).waitFor();
    await expect
      .poll(async () => (await seatAt(page, "김개발")).x)
      .toBeCloseTo(moved.x, 1);
  });

  test("실행 취소로 옮기기 전 자리로 돌아간다", async ({ page }) => {
    const before = await seatAt(page, "이코딩");
    const savedBefore = await saved(page, "이코딩");
    await select(page, "이코딩");
    await page.keyboard.press("ArrowDown");
    await expect
      .poll(async () => (await seatAt(page, "이코딩")).y)
      .toBeGreaterThan(before.y);
    // 저장이 끝나기 전에 실행 취소를 누르면 두 요청이 앞뒤로 엇갈린다. 서버가
    // 옮긴 자리를 받아들인 것을 확인한 뒤에 되돌린다. 서버는 비율 좌표를 쓰므로
    // 화면 좌표와 섞어 비교하지 않는다.
    await expect
      .poll(async () => saved(page, "이코딩"))
      .toBeGreaterThan(savedBefore);

    await page.keyboard.press("ControlOrMeta+z");
    await expect
      .poll(async () => (await seatAt(page, "이코딩")).y)
      .toBeCloseTo(before.y, 1);

    await page.reload();
    await mapCanvas(page).waitFor();
    expect((await seatAt(page, "이코딩")).y).toBeCloseTo(before.y, 1);
  });

  test("끌어서 옮겨도 자리가 바뀐다", async ({ page }) => {
    const before = await seatAt(page, "박서버");
    const seat = mapCanvas(page)
      .locator("g")
      .filter({ hasText: "박서버" })
      .first();
    const box = (await seat.boundingBox())!;
    // 좌석이 겹치지 않도록 도면 아래쪽 빈 곳으로 내린다.
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2 + 120, {
      steps: 12,
    });
    await page.mouse.up();
    await expect
      .poll(async () => (await seatAt(page, "박서버")).y)
      .toBeGreaterThan(before.y);
    const moved = await seatAt(page, "박서버");

    await page.reload();
    await mapCanvas(page).waitFor();
    await expect
      .poll(async () => (await seatAt(page, "박서버")).y)
      .toBeCloseTo(moved.y, 1);
  });
});

/** 서버에 저장된 좌석의 세로 위치(비율 좌표). */
const saved = async (page: Page, name: string) => {
  const seat = (await fetchSeats(page)).find(
    (item) => item.employeeName === name,
  );
  return seat ? seat.y : -1;
};

/** 좌석을 눌러 선택한다. 방향키 미세 이동은 선택된 좌석에만 걸린다. */
const select = async (page: Page, name: string) => {
  await mapCanvas(page).locator("g").filter({ hasText: name }).first().click();
  await expect(page.getByText(/1개 선택|개 선택/)).toBeVisible();
};
