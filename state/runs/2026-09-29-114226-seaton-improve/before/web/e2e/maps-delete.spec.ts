import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { csrfToken, login, watchConsole } from "./helpers";

test.describe("도면 버전 삭제", () => {
  test("잘못 올린 버전을 지울 수 있다", async ({ page }) => {
    const problems = watchConsole(page);
    await login(page);
    await page.goto("/admin/maps");
    await page.getByRole("button", { name: /도면 업로드/ }).click();

    const dialog = page.getByRole("dialog");
    const version = `x${Date.now().toString().slice(-6)}`;
    await dialog.getByLabel("도면 버전").fill(version);
    await dialog
      .locator("input[type=file]")
      .setInputFiles("e2e/fixtures/plan.png");
    await dialog.getByRole("button", { name: "업로드" }).click();

    const card = page.locator(`[data-map-version="${version}"]`);
    await expect(card).toBeVisible({ timeout: 30_000 });

    await card.getByRole("button", { name: "삭제" }).click();
    const confirm = page.getByRole("dialog");
    await expect(confirm).toContainText(version);
    await confirm.getByRole("button", { name: "삭제" }).click();

    await expect(card).toHaveCount(0);
    expect(problems()).toHaveLength(0);
  });

  test("게시를 내리면 그때부터 지울 수 있다", async ({ page }) => {
    test.setTimeout(120_000);
    await login(page);
    const headers = { "X-CSRF-Token": await csrfToken(page) };
    const listed = await page.request.get("/api/v1/floor-maps");
    const maps = await listed.json();
    const active = maps.items?.find((m: { active?: boolean }) => m.active);
    expect(
      active,
      `${listed.status()} ${JSON.stringify(maps).slice(0, 300)}`,
    ).toBeTruthy();

    // 그 층의 유일한 버전이면 다른 버전을 게시해 밀어낼 수도 없어, 게시를 내리는
    // 길이 없으면 잘못 올린 도면이 영영 남는다.
    const version = `u${Date.now().toString().slice(-6)}`;
    const draft = await (
      await page.request.post("/api/v1/floor-maps", {
        headers,
        multipart: {
          floorId: active.floorId,
          version,
          file: {
            name: "plan.png",
            mimeType: "image/png",
            buffer: readFileSync("e2e/fixtures/plan.png"),
          },
        },
      })
    ).json();
    try {
      await page.request.post(`/api/v1/floor-maps/${draft.id}/publish`, {
        headers,
      });
      await page.goto("/admin/maps");
      const card = page.locator(`[data-map-version="${version}"]`);
      await expect(card).toBeVisible({ timeout: 30_000 });
      // 게시 중에는 삭제가 없다.
      await expect(card.getByRole("button", { name: "삭제" })).toHaveCount(0);
      await card.getByRole("button", { name: "게시 내림" }).click();
      await expect(page.getByText("게시를 내렸습니다")).toBeVisible();

      await card.getByRole("button", { name: "삭제" }).click();
      await page
        .getByRole("dialog")
        .getByRole("button", { name: "삭제" })
        .click();
      await expect(card).toHaveCount(0);
    } finally {
      // 도중에 어긋나도 원래 버전을 다시 게시해 좌석맵을 되돌린다. 게시가 내려간
      // 채로 남으면 다음 검증이 통째로 무너진다.
      await page.request.delete(`/api/v1/floor-maps/${draft.id}`, { headers });
      await page.request.post(`/api/v1/floor-maps/${active.id}/publish`, {
        headers,
      });
    }
  });

  test("게시 중인 도면은 지울 수 없다", async ({ page }) => {
    await login(page);
    await page.goto("/admin/maps");
    // 서비스 중인 도면을 실수로 지우면 좌석맵이 통째로 비어 버린다. 화면에는
    // 삭제 단추 자체가 없어야 하고, 서버도 거절해야 한다.
    const published = page.locator('[data-map-version="v1"]');
    await expect(published.getByRole("button", { name: "삭제" })).toHaveCount(
      0,
    );

    const maps = await (await page.request.get("/api/v1/floor-maps")).json();
    const active = maps.items.find((m: { active?: boolean }) => m.active);
    const me = await (await page.request.get("/api/v1/auth/me")).json();
    const response = await page.request.delete(
      `/api/v1/floor-maps/${active.id}`,
      { headers: { "X-CSRF-Token": me.csrfToken } },
    );
    expect(response.status()).toBe(409);
    expect(await response.text()).toContain("게시 중인 도면");
  });
});
