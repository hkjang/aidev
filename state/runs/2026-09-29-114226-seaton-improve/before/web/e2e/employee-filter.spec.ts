import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { login } from "./helpers";

/**
 * 직원 화면의 조직 필터.
 *
 * 서버 `listEmployees` 는 `organizationId` 를 이미 걸러 주지만(employees.go:62,69)
 * 화면에 그것을 고를 자리가 없었다. 여기서 보는 것은 화면의 Select 가 정말 그
 * 쿼리를 실어 보내는지(표가 그 조직만 남는지)와, 걸러진 목록이 그대로 CSV 로
 * 내려오는지다. 쿼리 조립 규칙 자체는 vitest(`src/lib/employeeQuery.test.ts`)가
 * 증명한다.
 *
 * 이 검증은 서버 상태를 바꾸지 않는다 — 화면의 필터만 만지므로 되돌릴 것이 없다.
 * 시드는 조직 3개와 직원 10명(개발팀 6·영업팀 2·인사팀 2)을 만든다(seed.mjs).
 */

/** 표의 '조직' 칸. 머리글 순서는 직원·사번·조직·직책/직급·좌석·상태·작업. */
const ORG_COLUMN = 2;

const chooseOrganization = async (
  page: import("@playwright/test").Page,
  name: string,
) => {
  await page.getByRole("combobox", { name: "조직 필터" }).click();
  await page.getByRole("option", { name, exact: true }).click();
};

test.describe("직원 조직 필터", () => {
  test.beforeEach(async ({ page }) => login(page));

  test("조직을 고르면 그 조직의 직원만 표에 남는다", async ({ page }) => {
    await page.goto("/admin/employees");
    const rows = page.locator("table tbody tr");
    // 거르기 전에는 시드 직원 10명이 모두 보인다.
    await expect(rows).toHaveCount(10);

    await chooseOrganization(page, "영업팀");

    await expect(rows).toHaveCount(2);
    const orgs = await rows
      .locator(`td:nth-child(${ORG_COLUMN + 1})`)
      .allTextContents();
    expect(orgs).toEqual(["영업팀", "영업팀"]);
  });

  test("조직 필터는 검색어를 지우지 않는다", async ({ page }) => {
    await page.goto("/admin/employees");
    const search = page.getByPlaceholder("이름, 사번, 이메일, 조직 검색");
    // 개발팀에만 있는 이름. 영업팀으로 거르면 두 조건이 겹쳐 아무도 남지 않는다.
    await search.fill("김개발");
    // exact 가 없으면 AppShell 의 "직원 빠른 검색" 단추까지 잡혀 strict 위반이 된다.
    await page.getByRole("button", { name: "검색", exact: true }).click();
    await expect(page.locator("table tbody tr")).toHaveCount(1);

    await chooseOrganization(page, "영업팀");

    await expect(search).toHaveValue("김개발");
    await expect(page.getByText("조건에 맞는 직원이 없습니다.")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "CSV 내보내기" }),
    ).toBeDisabled();
  });

  test("걸러진 목록만 CSV 로 내려온다", async ({ page }) => {
    await page.goto("/admin/employees");
    await chooseOrganization(page, "영업팀");
    await expect(page.locator("table tbody tr")).toHaveCount(2);

    const button = page.getByRole("button", { name: "CSV 내보내기" });
    await expect(button).toBeEnabled();
    // 내려받기는 클릭과 거의 동시에 일어나므로 기다림을 클릭 전에 걸어야 한다.
    const wait = page.waitForEvent("download");
    await button.click();
    const download = await wait;

    const text = readFileSync((await download.path())!, "utf8");
    const lines = text.split("\r\n");
    // 머리글 한 줄 + 영업팀 2명. 거르기 전 10명이 담기면 안 된다.
    expect(lines.length - 1).toBe(2);
    expect(text).not.toContain("개발팀");
    expect(text).toContain("영업팀");
  });
});
