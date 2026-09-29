import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { login } from "./helpers";

/**
 * 직원 목록 CSV 내보내기.
 *
 * 이 검증은 서버 상태를 바꾸지 않는다 — 화면의 필터만 만지므로 되돌릴 것이 없다.
 * 칸 하나하나의 규칙(수식 방어·빈 칸·라벨)은 vitest
 * (`src/lib/employeeExport.test.ts`)가 증명한다. 여기서 보는 것은 실제 브라우저에서
 * 내려받기가 정말 일어나는지, 파일 이름이 맞는지, 그리고 표에 보이는 행이 그대로
 * 파일이 되는지다.
 */
test.describe("직원 CSV 내보내기", () => {
  test.beforeEach(async ({ page }) => login(page));

  test("표에 보이는 목록이 그대로 직원목록 파일로 내려온다", async ({
    page,
  }) => {
    await page.goto("/admin/employees");
    const button = page.getByRole("button", { name: "CSV 내보내기" });
    await expect(button).toBeEnabled();
    const rows = await page.locator("table tbody tr").count();
    expect(rows).toBeGreaterThan(0);

    // 내려받기는 클릭과 거의 동시에 일어나므로 기다림을 클릭 전에 걸어야 한다.
    const wait = page.waitForEvent("download");
    await button.click();
    const download = await wait;
    expect(download.suggestedFilename()).toMatch(
      /^직원목록_\d{4}-\d{2}-\d{2}\.csv$/,
    );

    const path = await download.path();
    const text = readFileSync(path!, "utf8");
    const lines = text.split("\r\n");
    expect(lines[0]).toBe(
      "﻿이름,사번,이메일,조직명,직급,직책,근무지,좌석,재직상태",
    );
    // 다시 조회하지 않으므로 표의 행 수와 파일의 데이터 행 수가 같다.
    expect(lines.length - 1).toBe(rows);
    // 재직상태 라벨은 표와 같은 함수에서 나온다. 코드(active)가 새어 나오면 안 된다.
    expect(text).toContain(",재직");
    expect(text).not.toContain("active");
  });

  test("걸러진 목록이 비면 내보내기를 누를 수 없다", async ({ page }) => {
    await page.goto("/admin/employees");
    const button = page.getByRole("button", { name: "CSV 내보내기" });
    await expect(button).toBeEnabled();
    // 시드 직원은 전원 좌석이 있어 "미배정"으로 거르면 목록이 빈다.
    await page
      .getByRole("combobox")
      .filter({ hasText: "전체 배정상태" })
      .click();
    await page.getByRole("option", { name: "미배정", exact: true }).click();
    await expect(page.getByText("조건에 맞는 직원이 없습니다.")).toBeVisible();
    await expect(button).toBeDisabled();
  });
});
