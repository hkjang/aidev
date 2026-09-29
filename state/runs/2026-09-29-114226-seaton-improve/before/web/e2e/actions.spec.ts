import { expect, test } from "@playwright/test";
import { login } from "./helpers";

test.describe("처리 필요", () => {
  test("상단 배지와 작업 큐 건수가 어긋나지 않는다", async ({ page }) => {
    await login(page);
    await page.goto("/admin/actions");
    // 배지와 목록은 서버에서 서로 다른 질의로 계산한다. 한쪽만 고치면 화면에
    // 7건이라 적혀 있는데 목록에는 6건만 나오는 상태가 된다.
    const badge = page.getByRole("navigation").getByText(/^\d+$/).first();
    const queue = page.getByText(/^\d+개 항목$/);
    await expect(queue).toBeVisible();
    const listed = Number((await queue.textContent())!.replace(/\D/g, ""));
    expect(Number((await badge.textContent())!)).toBe(listed);

    // 갈래별 탭을 모두 더하면 전체와 같아야 한다.
    let sum = 0;
    const tabs = [
      { name: "미배정", kind: "unassigned_employee" },
      { name: "퇴직자", kind: "retired_assignment" },
      { name: "조직 불일치", kind: "organization_mismatch" },
      { name: "AI 확인", kind: "low_confidence" },
    ];
    for (const { name, kind } of tabs) {
      // 탭을 누르면 서버에 다시 묻는다. 응답을 기다리지 않고 세면 이전 탭의
      // 목록을 세게 된다.
      const [response] = await Promise.all([
        page.waitForResponse((res) => res.url().includes(`kind=${kind}`)),
        page.getByRole("tab", { name }).click(),
      ]);
      const items = (await response.json()).items as Array<{ kind: string }>;
      expect(items.every((item) => item.kind === kind)).toBe(true);
      await expect(page.getByText(`${items.length}개 항목`)).toBeVisible();
      sum += items.length;
    }
    expect(sum).toBe(listed);
  });

  test("조직 영역 불일치를 화면에서 바로 맞춘다", async ({ page }) => {
    await login(page);
    const seats = await (await page.request.get("/api/v1/floor-maps")).json();
    const active = seats.items.find((m: { active?: boolean }) => m.active);
    const before = await (
      await page.request.get(`/api/v1/seats?floorMapId=${active.id}`)
    ).json();
    const mismatched = before.items.find(
      (s: { organizationId?: string; employeeOrganizationId?: string }) =>
        s.organizationId &&
        s.employeeOrganizationId &&
        s.organizationId !== s.employeeOrganizationId,
    );
    expect(mismatched, "구역이 어긋난 좌석이 시드에 있어야 한다").toBeTruthy();

    await page.goto("/admin/actions");
    await page.getByRole("tab", { name: "조직 불일치" }).click();
    const fix = page.getByRole("button", { name: "영역 맞춤" }).first();
    await expect(fix).toBeVisible();
    await fix.click();

    // 맞추고 나면 그 항목은 큐에서 사라지고 좌석 구역이 직원 소속을 따라간다.
    await expect(page.getByRole("button", { name: "영역 맞춤" })).toHaveCount(
      0,
    );
    const after = await (
      await page.request.get(`/api/v1/seats?floorMapId=${active.id}`)
    ).json();
    const fixed = after.items.find(
      (s: { id: string }) => s.id === mismatched.id,
    );
    expect(fixed.organizationId).toBe(mismatched.employeeOrganizationId);

    // 다음 실행도 같은 상태에서 시작하도록 되돌린다.
    const me = await (await page.request.get("/api/v1/auth/me")).json();
    await page.request.patch(`/api/v1/seats/${mismatched.id}`, {
      headers: { "X-CSRF-Token": me.csrfToken },
      data: { organizationId: mismatched.organizationId },
    });
  });
});
