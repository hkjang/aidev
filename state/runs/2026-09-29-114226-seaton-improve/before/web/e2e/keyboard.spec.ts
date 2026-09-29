import { expect, test } from "@playwright/test";
import { login, mapCanvas } from "./helpers";

test.describe("키보드 조작", () => {
  test("로그인 화면을 키보드만으로 지날 수 있다", async ({ page }) => {
    await page.goto("/login");
    await page.locator("body").click({ position: { x: 5, y: 5 } });
    const focused = () =>
      page.evaluate(
        () =>
          document.activeElement?.getAttribute("autocomplete") ??
          document.activeElement?.getAttribute("type") ??
          "",
      );
    const stops: string[] = [];
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press("Tab");
      stops.push(await focused());
    }
    expect(stops).toContain("username");
    expect(stops).toContain("current-password");
    expect(stops).toContain("submit");
  });

  test("검색 결과를 키보드로 골라 좌석을 볼 수 있다", async ({ page }) => {
    await login(page);
    await mapCanvas(page).waitFor();
    // 좌석은 도면 위 그림이라 탭으로 닿지 않는다. 키보드 사용자가 좌석을 고르는
    // 길은 이 검색 목록뿐이므로 목록이 초점을 받고 Enter로 열려야 한다.
    await page.fill('input[placeholder*="이름"]', "개발팀");
    await page.keyboard.press("Enter");
    await expect(page.getByText(/\d+명 검색됨/)).toBeVisible();

    const row = page.getByRole("button", {
      name: /이코딩 · 개발팀 .* 좌석 보기/,
    });
    await expect(row).toBeVisible();
    await row.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("heading", { name: /HQ-3F-/ })).toBeVisible();
    // 상세가 열리며 해당 좌석으로 확대된다.
    await expect(page.getByText(/^\d+%$/)).not.toHaveText("100%");
  });

  test("도면 위 좌석을 방향키로 옮겨 다닌다", async ({ page }) => {
    await login(page);
    await mapCanvas(page).waitFor();
    const focused = () =>
      page.evaluate(
        () => document.activeElement?.getAttribute("aria-label") ?? "",
      );

    // 좌석은 탭 순서에 딱 하나만 들어온다. 좌석 수백 개를 모두 훑게 하면 도면을
    // 지나 다음 조작으로 가는 데만 수백 번을 눌러야 한다.
    await page.locator("body").click({ position: { x: 5, y: 5 } });
    const stops: string[] = [];
    for (let i = 0; i < 30; i++) {
      await page.keyboard.press("Tab");
      stops.push(await focused());
    }
    expect(stops.filter((name) => name.startsWith("HQ-3F-"))).toHaveLength(1);

    await page.evaluate(() =>
      document.querySelector<SVGGElement>("[data-seat-id]")?.focus(),
    );
    expect(await focused()).toContain("HQ-3F-001");
    await page.keyboard.press("ArrowRight");
    expect(await focused()).toContain("HQ-3F-002");
    await page.keyboard.press("ArrowDown");
    expect(await focused()).toContain("HQ-3F-008");
    await page.keyboard.press("ArrowUp");
    expect(await focused()).toContain("HQ-3F-002");
    await page.keyboard.press("End");
    expect(await focused()).toContain("HQ-3F-030");
    await page.keyboard.press("Home");
    expect(await focused()).toContain("HQ-3F-001");

    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "HQ-3F-001" }),
    ).toBeVisible();
  });

  test("좌석에 초점이 있으면 방향키가 화면을 밀지 않는다", async ({ page }) => {
    await login(page);
    await mapCanvas(page).waitFor();
    // 좌석 초점이 옮겨 가는 동시에 화면까지 밀리면 두 번 움직여 어디를 보고
    // 있는지 잃는다.
    const viewBox = () => mapCanvas(page).getAttribute("viewBox");
    await page.evaluate(() =>
      document.querySelector<SVGGElement>("[data-seat-id]")?.focus(),
    );
    const before = await viewBox();
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowDown");
    expect(await viewBox()).toBe(before);
  });

  test("아이콘만 있는 조작에도 이름이 붙어 있다", async ({ page }) => {
    await login(page);
    await mapCanvas(page).waitFor();
    const unnamed = await page.evaluate(
      () =>
        Array.from(document.querySelectorAll("button, [role=button]")).filter(
          (el) =>
            !(
              el.getAttribute("aria-label") ||
              el.textContent?.trim() ||
              el.getAttribute("title")
            ),
        ).length,
    );
    expect(unnamed).toBe(0);
  });
});
