import { expect, test, type Page } from "@playwright/test";
import { csrfToken, login } from "./helpers";

/** 개인 API 키를 만들고 원문을 돌려준다. 화면을 거치지 않는 준비 단계에 쓴다. */
const issueKey = async (page: Page, name: string, scopes: string[]) => {
  const created = await page.request.post("/api/v1/api-keys", {
    headers: { "X-CSRF-Token": await csrfToken(page) },
    data: { name, scopes },
  });
  expect(created.ok(), await created.text()).toBe(true);
  return (await created.json()) as { id: string; key: string; prefix: string };
};

const asKey = (page: Page, key: string) => ({
  headers: { Authorization: `Bearer ${key}` },
});

test.describe("개인 API 키", () => {
  test("원문은 만들 때 한 번만 보여준다", async ({ page }) => {
    await login(page);
    await page.goto("/profile/keys");
    await page
      .getByRole("button", { name: /키 만들기/ })
      .first()
      .click();

    const dialog = page.getByRole("dialog");
    const name = `확인용 ${Date.now().toString().slice(-6)}`;
    await dialog.getByLabel("키 이름").fill(name);
    await dialog.getByRole("button", { name: "생성" }).click();

    // 서버에는 복원할 수 없는 해시만 남으므로, 이 화면을 지나면 다시 볼 수 없다.
    const secret = page.getByText(/^seat_[A-Za-z0-9_-]{20,}$/);
    await expect(secret).toBeVisible();
    const raw = (await secret.textContent())!;
    await page.getByRole("button", { name: "보관 완료" }).click();

    await page.reload();
    await expect(page.getByText(raw)).toHaveCount(0);
    const row = page.getByRole("row", { name: new RegExp(name) });
    await expect(row).toBeVisible();
    // 상태는 글로도 읽혀야 한다. 흐리게만 표시하면 화면 낭독기에는 아무것도
    // 전해지지 않고, 눈으로도 폐기된 키인지 알 수 없다.
    await expect(row).toContainText("사용 중");

    // 뒷정리: 방금 만든 키를 지운다.
    const keys = await (await page.request.get("/api/v1/api-keys")).json();
    const mine = keys.items.find(
      (item: { name: string }) => item.name === name,
    );
    await page.request.delete(`/api/v1/api-keys/${mine.id}`, {
      headers: { "X-CSRF-Token": await csrfToken(page) },
    });
  });

  test("범위를 넘는 요청은 막는다", async ({ page }) => {
    await login(page);
    const readOnly = await issueKey(page, `읽기 ${Date.now()}`, ["read"]);
    try {
      // 읽기 전용 키로 쓰기가 되면, 조회용으로 나눠 준 키가 데이터를 바꿀 수 있다.
      expect(
        (
          await page.request.get("/api/v1/employees", asKey(page, readOnly.key))
        ).status(),
      ).toBe(200);
      expect(
        (
          await page.request.post("/api/v1/organizations", {
            ...asKey(page, readOnly.key),
            data: { externalId: "SCOPE-TEST", name: "범위 확인" },
          })
        ).status(),
      ).toBe(403);
      // mcp 범위가 없으면 MCP 연결도 막힌다.
      expect(
        (
          await page.request.post("/mcp", {
            ...asKey(page, readOnly.key),
            data: { jsonrpc: "2.0", id: 1, method: "tools/list" },
          })
        ).status(),
      ).toBe(403);
    } finally {
      await page.request.delete(`/api/v1/api-keys/${readOnly.id}`, {
        headers: { "X-CSRF-Token": await csrfToken(page) },
      });
    }
  });

  test("화면에서 폐기하면 그 키는 즉시 막힌다", async ({ page }) => {
    await login(page);
    const name = `폐기 ${Date.now().toString().slice(-6)}`;
    const key = await issueKey(page, name, ["read"]);
    expect(
      (
        await page.request.get("/api/v1/employees", asKey(page, key.key))
      ).status(),
    ).toBe(200);

    await page.goto("/profile/keys");
    const row = page.getByRole("row", { name: new RegExp(name) });
    await row.getByRole("button", { name: "폐기" }).click();
    // 되돌릴 수 없는 조작이라 어떤 키인지 보여주고 한 번 더 확인받는다.
    const confirm = page.getByRole("dialog");
    await expect(confirm).toContainText(name);
    await confirm.getByRole("button", { name: "폐기" }).click();

    await expect
      .poll(async () =>
        (
          await page.request.get("/api/v1/employees", asKey(page, key.key))
        ).status(),
      )
      .toBe(401);
  });

  test("회전하면 유예가 끝난 옛 키는 막힌다", async ({ page }) => {
    await login(page);
    const headers = { "X-CSRF-Token": await csrfToken(page) };
    const settings = (payload: Record<string, string>) =>
      page.request.put("/api/v1/settings", {
        headers,
        data: { settings: payload },
      });

    // 유예를 0으로 두면 회전 즉시 옛 키가 막혀야 한다. 이 조건이 무너지면 유출된
    // 키를 회전으로 갈아 끼워도 옛 키가 계속 살아 있다.
    await settings({ "security.rotation_grace_hours": "0" });
    const key = await issueKey(page, `회전 ${Date.now()}`, ["read"]);
    try {
      const rotated = await (
        await page.request.post(`/api/v1/api-keys/${key.id}/rotate`, {
          headers,
        })
      ).json();
      expect(
        (
          await page.request.get("/api/v1/employees", asKey(page, key.key))
        ).status(),
      ).toBe(401);
      expect(
        (
          await page.request.get("/api/v1/employees", asKey(page, rotated.key))
        ).status(),
      ).toBe(200);
      await page.request.delete(`/api/v1/api-keys/${rotated.id}`, { headers });
    } finally {
      await settings({ "security.rotation_grace_hours": "24" });
    }
  });
});
