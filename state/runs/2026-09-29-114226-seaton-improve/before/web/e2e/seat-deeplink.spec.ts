import { expect, test } from "@playwright/test";
import { login, mapCanvas } from "./helpers";

/**
 * 좌석맵이 깊은 링크를 만드는지 본다.
 *
 * 좌석맵은 오래 `?q=`·`?map=` 을 읽기만 했다. 화면에서 검색해도 주소는 `/` 그대로여서
 * 새로고침하면 검색 결과와 고른 좌석이 사라지고, 주소를 복사해 줘도 상대는 다른 화면을
 * 봤다. 여기서 보는 것은 (1) 제출이 주소에 `q` 를 남기는지 (2) 그 주소를 다시 열면
 * 같은 화면이 되는지 (3) 제출이 같은 검색을 두 번 보내지 않는지다.
 *
 * 도면(`?map=`) 쪽은 시드에 도면이 한 개뿐이어서 선택란 자체가 화면에 나오지 않는다 →
 * 그쪽 규칙은 src/lib/seatMapLink.test.ts 가 증명한다.
 */

const PERSON = "김개발";
type Page = import("@playwright/test").Page;
const searchBox = (page: Page) =>
  page.getByPlaceholder("이름, 사번, 조직 검색");
/** 검색 결과 목록에서 그 사람 줄. 좌석이 배정돼 있어야 눌러 볼 수 있는 단추다. */
const resultRow = (page: Page) =>
  page.getByRole("button", { name: new RegExp(`^${PERSON} · `) });

test.describe("좌석맵 깊은 링크", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await mapCanvas(page).waitFor();
  });

  test("검색을 제출하면 주소에 남고 새로고침해도 그대로다", async ({
    page,
  }) => {
    await searchBox(page).fill(PERSON);
    await searchBox(page).press("Enter");

    await expect(page).toHaveURL(/\?q=/);
    expect(
      decodeURIComponent(new URL(page.url()).searchParams.get("q") ?? ""),
    ).toBe(PERSON);
    // 검색이 실제로 나갔고 찾은 사람의 좌석이 선택됐다.
    await expect(resultRow(page)).toHaveCount(1);
    const detail = page.getByRole("region", { name: "좌석 상세" });
    await expect(detail.getByText(PERSON)).toBeVisible();
    const seatNo = await detail.getByRole("heading").first().textContent();
    expect(seatNo?.trim()).toBeTruthy();

    // 주소만 들고 다시 열어도 같은 화면이어야 한다.
    await page.reload();
    await mapCanvas(page).waitFor();
    await expect(searchBox(page)).toHaveValue(PERSON);
    await expect(resultRow(page)).toHaveCount(1);
    await expect(
      page.getByRole("region", { name: "좌석 상세" }).getByText(PERSON),
    ).toBeVisible();
    expect(
      (
        await page
          .getByRole("region", { name: "좌석 상세" })
          .getByRole("heading")
          .first()
          .textContent()
      )?.trim(),
    ).toBe(seatNo?.trim());
  });

  test("제출이 같은 검색을 두 번 보내지 않고, 지우면 주소에서 사라진다", async ({
    page,
  }) => {
    // 제출이 주소에 q 를 쓰면 q 를 감시하는 effect 가 깨어나 같은 검색을 한 번 더
    // 보낼 수 있다. 실제로 나간 요청을 세어 그것이 없음을 본다.
    const calls: string[] = [];
    page.on("request", (req) => {
      const url = new URL(req.url());
      if (url.pathname === "/api/v1/employees" && url.searchParams.get("q"))
        calls.push(url.searchParams.get("q")!);
    });

    await searchBox(page).fill(PERSON);
    await searchBox(page).press("Enter");
    await expect(page).toHaveURL(/\?q=/);
    await expect(resultRow(page)).toHaveCount(1);
    // 주소가 바뀐 뒤 effect 가 한 번 더 돌 여유를 준다.
    await page.waitForTimeout(1000);
    expect(calls.filter((q) => q === PERSON)).toHaveLength(1);

    // 지우고 제출하면 q 가 주소에서 빠진다.
    await searchBox(page).fill("");
    await searchBox(page).press("Enter");
    await expect(page).not.toHaveURL(/\?q=/);
    await expect(page.getByText("사람 또는 조직을 검색하세요")).toBeVisible();
  });

  test("이미 주소에 있던 edit=1 은 검색해도 지워지지 않는다", async ({
    page,
  }) => {
    await page.goto("/?edit=1");
    await mapCanvas(page).waitFor();
    await expect(page.getByText(/Shift로 다중 선택|개 선택/)).toBeVisible();

    await searchBox(page).fill(PERSON);
    await searchBox(page).press("Enter");

    await expect(page).toHaveURL(/edit=1/);
    await expect(page).toHaveURL(/q=/);
  });
});
