import { expect, test, type Page } from "@playwright/test";
import { login, mapCanvas, seatLabels, watchConsole } from "./helpers";

test.describe("좌석맵", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await mapCanvas(page).waitFor();
  });

  test("도면 전체가 보이고 좌석 라벨이 읽힌다", async ({ page }) => {
    const labels = await seatLabels(page);
    expect(labels.length).toBeGreaterThan(5);
    // 좌석 번호는 도면 안에서 구분되는 뒷자리만 남아야 한다. 접두사를 그대로 두면
    // "HQ-3F-0…"처럼 잘려 어느 자리인지 알 수 없다.
    const numbers = labels.filter((t) => /^\d+$/.test(t));
    expect(numbers.length).toBeGreaterThan(3);
    expect(labels.filter((t) => t.includes("…"))).toHaveLength(0);
  });

  test("조작 패널이 도면을 가리지 않는다", async ({ page }) => {
    const toolbar = page.getByRole("group", { name: "좌석 필터" });
    const bar = await toolbar.boundingBox();
    const canvas = await mapCanvas(page).boundingBox();
    expect(bar).not.toBeNull();
    expect(canvas).not.toBeNull();
    expect(bar!.y + bar!.height).toBeLessThanOrEqual(canvas!.y + 1);
  });

  test("휠로 확대·축소되고 라벨이 좌석을 뒤덮지 않는다", async ({ page }) => {
    const zoom = page.getByText(/^\d+%$/);
    await expect(zoom).toHaveText("100%");
    const fit = await labelHeight(page);

    const box = (await mapCanvas(page).boundingBox())!;
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    for (let i = 0; i < 6; i++) await page.mouse.wheel(0, -240);
    await expect(zoom).not.toHaveText("100%");
    const zoomedText = await zoom.textContent();
    expect(Number(zoomedText!.replace("%", ""))).toBeGreaterThan(300);

    // 글자가 배율만큼 커지면 좌석 한 칸을 글자가 가득 채워 아무것도 못 읽는다.
    const zoomed = await labelHeight(page);
    expect(zoomed).toBeGreaterThan(fit * 0.8);
    expect(zoomed).toBeLessThan(fit * 3);

    await page.getByRole("button", { name: "전체 보기 (도면 맞춤)" }).click();
    await expect(zoom).toHaveText("100%");
  });

  test("편집 도구가 확대 조작과 도면을 가리지 않는다", async ({ page }) => {
    await page.getByRole("button", { name: /배치 편집/ }).click();
    const tools = page.getByText(/Shift로 다중 선택|개 선택/);
    await expect(tools).toBeVisible();
    // 편집 도구가 도면 위에 떠 있으면 확대 배율 표시와 좌석을 함께 덮는다.
    const bar = (await tools.boundingBox())!;
    const canvas = (await mapCanvas(page).boundingBox())!;
    expect(bar.y + bar.height).toBeLessThanOrEqual(canvas.y + 1);
    await expect(page.getByText(/^\d+%$/)).toBeVisible();
    const zoom = (await page.getByText(/^\d+%$/).boundingBox())!;
    expect(overlaps(bar, zoom)).toBe(false);
  });

  test("1280px 노트북 화면에서 세 칸이 모두 들어간다", async ({ page }) => {
    // 세 칸의 최소 폭 합이 화면을 넘겨 상세 패널이 오른쪽으로 잘려 나간 적이 있다.
    await page.setViewportSize({ width: 1280, height: 800 });
    await expect(page.getByText("좌석을 선택하면")).toBeVisible();
    const detail = (await page.getByText("좌석을 선택하면").boundingBox())!;
    expect(detail.x + detail.width).toBeLessThanOrEqual(1280);
    const canvas = (await mapCanvas(page).boundingBox())!;
    expect(canvas.width).toBeGreaterThan(300);
  });

  test("화면을 옮겨도 좌석을 다시 그리지 않는다", async ({ page }) => {
    // 화면 이동과 확대는 viewBox만 바꾸고 좌석 좌표는 건드리지 않는다. 좌석을
    // 옮기는 방식으로 되돌리면 좌석 수백 개를 매 프레임 다시 그려 큰 도면에서
    // 화면이 끊긴다. 500석 도면에서도 60fps가 유지되는 근거가 이것이다.
    const seatCoords = () =>
      page.evaluate(() =>
        Array.from(
          document.querySelectorAll(
            "svg[aria-label*='좌석 배치도'] [data-seat-id] rect",
          ),
        )
          .slice(0, 10)
          .map((node) => `${node.getAttribute("x")},${node.getAttribute("y")}`)
          .join(" "),
      );
    // 전체 보기에서는 도면이 다 보여 화면이 움직이지 않는다. 먼저 확대한다.
    const box = (await mapCanvas(page).boundingBox())!;
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.wheel(0, -400);
    await expect(page.getByText(/^\d+%$/)).not.toHaveText("100%");

    const before = await seatCoords();
    const viewBox = await mapCanvas(page).getAttribute("viewBox");
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    for (let i = 1; i <= 10; i++)
      await page.mouse.move(
        box.x + box.width / 2 - i * 4,
        box.y + box.height / 2 - i * 3,
      );
    await page.mouse.up();

    expect(await mapCanvas(page).getAttribute("viewBox")).not.toBe(viewBox);
    expect(await seatCoords()).toBe(before);
  });

  test("확대하면 미니맵이 나타난다", async ({ page }) => {
    const minimap = page.getByLabel("도면 전체 미니맵");
    await expect(minimap).toBeHidden();
    const box = (await mapCanvas(page).boundingBox())!;
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.wheel(0, -240);
    await expect(minimap).toBeVisible();
  });

  test("좌석을 누르면 상세가 열린다", async ({ page }) => {
    // 화면 이동을 위해 포인터를 잡는 처리가 클릭을 삼킨 적이 있다.
    const seat = mapCanvas(page)
      .locator("g", { has: page.locator("text") })
      .first();
    await seat.click();
    await expect(
      page.getByRole("heading", { name: /김개발|\d+/ }),
    ).toBeVisible();
  });

  test("조직 색과 구역 표시를 켤 수 있다", async ({ page }) => {
    const problems = watchConsole(page);
    await page.getByRole("button", { name: "조직 색", exact: true }).click();
    await page
      .getByRole("button", { name: "좌석에 지정된 조직 구역을 배경으로 표시" })
      .click();
    await expect(page.getByText("개발팀 6")).toBeVisible();
    expect(problems()).toHaveLength(0);
  });

  test("필터로 빈 좌석만 강조한다", async ({ page }) => {
    await page.getByRole("button", { name: "빈 좌석", exact: true }).click();
    await expect(page.getByText(/\d+ \/ \d+석 강조/)).toBeVisible();
  });
});

/** 두 사각형이 한 점이라도 겹치는지. */
const overlaps = (a: Box, b: Box) =>
  a.x < b.x + b.width &&
  b.x < a.x + a.width &&
  a.y < b.y + b.height &&
  b.y < a.y + a.height;

type Box = { x: number; y: number; width: number; height: number };

/** 화면에 그려진 좌석 라벨의 실제 픽셀 높이. */
const labelHeight = async (page: Page) => {
  const box = await mapCanvas(page).locator("text").first().boundingBox();
  return box!.height;
};
