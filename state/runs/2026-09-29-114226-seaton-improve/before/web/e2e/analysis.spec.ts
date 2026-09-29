import { expect, test } from "@playwright/test";
import { login, watchConsole } from "./helpers";

test.describe("도면 업로드와 AI 분석", () => {
  test("고를 것이 하나뿐인 목록은 미리 골라 둔다", async ({ page }) => {
    await login(page);
    await page.goto("/admin/maps");
    // 사업장이 하나뿐인데 목록을 펼쳐 고르게 하면, 저장이 왜 꺼져 있는지만
    // 헷갈린다.
    await page.getByRole("button", { name: "층", exact: true }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("combobox")).toHaveText(/본사/);
    await dialog.getByRole("button", { name: "취소" }).click();

    await page.getByRole("button", { name: /도면 업로드/ }).click();
    await expect(page.getByRole("dialog").getByRole("combobox")).toHaveText(
      /본사 · 3층/,
    );
  });

  test("도면을 올리고 AI 분석까지 화면에서 끝낼 수 있다", async ({ page }) => {
    test.setTimeout(180_000);
    const problems = watchConsole(page);
    await login(page);
    await page.goto("/admin/maps");
    await page.getByRole("button", { name: /도면 업로드/ }).click();

    const dialog = page.getByRole("dialog");
    const version = `t${Date.now().toString().slice(-6)}`;
    await dialog.getByLabel("도면 버전").fill(version);
    await dialog
      .locator("input[type=file]")
      .setInputFiles("e2e/fixtures/plan.png");
    await dialog.getByRole("button", { name: "업로드" }).click();

    const card = page.locator(`[data-map-version="${version}"]`);
    await expect(card).toContainText("분석 전", { timeout: 30_000 });

    // 분석은 비동기 작업이다. 요청 즉시 끝나지 않으므로 화면이 진행 상태를
    // 보여주고, 끝나면 인식된 좌석 수가 카드에 나타나야 한다.
    await card.getByRole("button", { name: "AI 분석" }).click();
    await expect(card).not.toContainText("분석 전", { timeout: 150_000 });
    await expect(card).toContainText(/[1-9]\d*석/);

    // 확인용으로 올린 버전은 지우고 끝낸다. 남겨 두면 실행할 때마다 도면 카드가
    // 쌓여 다른 검증이 어느 카드를 보는지 흐려진다.
    await card.getByRole("button", { name: "삭제" }).click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "삭제" })
      .click();
    await expect(card).toHaveCount(0);
    expect(problems()).toHaveLength(0);
  });
});
