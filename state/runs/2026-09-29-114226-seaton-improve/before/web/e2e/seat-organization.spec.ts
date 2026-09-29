import { expect, test, type Page } from "@playwright/test";
import { csrfToken, login, mapCanvas } from "./helpers";

type Organization = { id: string; name: string; color: string };
type Seat = {
  id: string;
  floorMapId: string;
  seatNo: string;
  rotation: number;
  organizationId?: string;
  employeeId?: string;
  employeeName?: string;
  employeeOrganizationId?: string;
};

const seats = async (page: Page): Promise<Seat[]> => {
  const response = await page.request.get("/api/v1/seats");
  expect(response.ok()).toBeTruthy();
  return (await response.json()).items;
};
const organizations = async (page: Page): Promise<Organization[]> => {
  const response = await page.request.get("/api/v1/organizations");
  expect(response.ok()).toBeTruthy();
  return (await response.json()).items;
};
const shape = (page: Page, id: string) =>
  mapCanvas(page).locator(`[data-seat-id="${id}"]`);
const field = (page: Page) =>
  page
    .getByRole("dialog")
    .getByRole("combobox", { name: "조직 구역", exact: true });
const choose = async (page: Page, name: string) => {
  await field(page).click();
  await page.getByRole("option", { name, exact: true }).click();
};
const edit = async (page: Page, id: string) => {
  await shape(page, id).click();
  await page.getByRole("button", { name: "편집", exact: true }).click();
  await expect(field(page)).toBeVisible();
};
const save = async (page: Page, method: "POST" | "PATCH", id?: string) => {
  const response = page.waitForResponse(
    (r) =>
      new URL(r.url()).pathname === `/api/v1/seats${id ? `/${id}` : ""}` &&
      r.request().method() === method,
  );
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "저장", exact: true })
    .click();
  const result = await response;
  expect(result.status()).toBe(method === "POST" ? 201 : 204);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  return result;
};
const reopen = async (page: Page, id: string, name: string) => {
  await page.reload();
  await mapCanvas(page).waitFor();
  await page.getByRole("button", { name: /배치 편집/ }).click();
  await edit(page, id);
  await expect(field(page)).toHaveText(name);
};
const cancel = async (page: Page) => {
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "취소", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
};

test.beforeEach(async ({ page }) => {
  await login(page);
  await mapCanvas(page).waitFor();
  await page.getByRole("button", { name: /배치 편집/ }).click();
});

for (const assigned of [true, false]) {
  test(`신규 좌석을 ${assigned ? "조직 지정" : "미지정"}으로 생성하고 다시 연다`, async ({
    page,
  }) => {
    const org = (await organizations(page))[0];
    const seatNo = `ORG-${Date.now()}-${assigned ? "A" : "N"}`;
    const headers = { "X-CSRF-Token": await csrfToken(page) };
    try {
      await page
        .getByRole("button", { name: "좌석 추가", exact: true })
        .click();
      await expect(field(page)).toHaveText("미지정");
      await expect(
        page.getByText("좌석에 지정할 구역이며 직원 소속은 바뀌지 않습니다"),
      ).toBeVisible();
      await page.getByRole("textbox", { name: "좌석 번호" }).fill(seatNo);
      // 기존 좌석과 겹치지 않는 도면 아래쪽에 만든다.
      await page
        .getByRole("spinbutton", { name: "y", exact: true })
        .fill("0.85");
      await choose(page, org.name);
      if (!assigned) await choose(page, "미지정");
      const response = await save(page, "POST");
      expect(response.request().postDataJSON().organizationId).toBe(
        assigned ? org.id : null,
      );
      const id = (await response.json()).id;
      const created = (await seats(page)).find((s) => s.id === id)!;
      expect(created.organizationId ?? null).toBe(assigned ? org.id : null);
      expect(created.employeeId).toBeFalsy();
      await page.getByText("조직 색", { exact: true }).click();
      await expect(shape(page, id).locator("rect").last()).toHaveAttribute(
        "fill",
        assigned ? org.color : "#DFE7EB",
      );
      await reopen(page, id, assigned ? org.name : "미지정");
    } finally {
      // 번호로 찾아 응답 확인 중 실패해도 생성한 좌석을 정리한다.
      for (const seat of (await seats(page)).filter(
        (s) => s.seatNo === seatNo,
      )) {
        expect(
          (
            await page.request.delete(`/api/v1/seats/${seat.id}`, { headers })
          ).status(),
        ).toBe(204);
      }
    }
  });
}

test("기존 좌석의 지정·변경·해제와 직원 소속 우선 색을 유지한다", async ({
  page,
}) => {
  const original = (await seats(page)).find(
    (s) => s.employeeName === "김개발",
  )!;
  const orgs = await organizations(page);
  const own = orgs.find((o) => o.id === original.employeeOrganizationId)!;
  const other = orgs.find((o) => o.id !== own.id)!;
  const headers = { "X-CSRF-Token": await csrfToken(page) };
  const employee = async () => {
    const response = await page.request.get(
      `/api/v1/employees?q=${encodeURIComponent(original.employeeName!)}`,
    );
    expect(response.ok()).toBeTruthy();
    return (await response.json()).items.find(
      (e: { id: string }) => e.id === original.employeeId,
    );
  };
  const employeeBefore = await employee();
  try {
    for (const org of [null, own, other, null]) {
      await edit(page, original.id);
      await choose(page, org?.name ?? "미지정");
      const response = await save(page, "PATCH", original.id);
      expect(response.request().postDataJSON().organizationId).toBe(
        org?.id ?? "",
      );
      const allSeats = await seats(page);
      const saved = allSeats.find((s) => s.id === original.id)!;
      expect(saved.organizationId ?? null).toBe(org?.id ?? null);
      expect(saved.employeeId).toBe(original.employeeId);
      expect(saved.employeeOrganizationId).toBe(own.id);
      expect(await employee()).toEqual(employeeBefore);
      if (org === other)
        await expect(shape(page, original.id)).toHaveAttribute(
          "aria-label",
          /구역 불일치/,
        );
      else
        await expect(shape(page, original.id)).not.toHaveAttribute(
          "aria-label",
          /구역 불일치/,
        );
      await page.getByText("조직 색", { exact: true }).click();
      await expect(
        shape(page, original.id).locator("rect").last(),
      ).toHaveAttribute("fill", own.color);
      await page.getByText("구역", { exact: true }).click();
      for (const organization of orgs) {
        const count = allSeats.filter(
          (s) =>
            s.floorMapId === original.floorMapId &&
            s.organizationId === organization.id,
        ).length;
        const label = mapCanvas(page).getByText(
          `${organization.name} · ${count}석`,
          { exact: true },
        );
        if (count) await expect(label).toBeVisible();
        else
          await expect(
            mapCanvas(page)
              .locator("text")
              .filter({ hasText: `${organization.name} ·` }),
          ).toHaveCount(0);
      }
      await reopen(page, original.id, org?.name ?? "미지정");
      await cancel(page);
    }
  } finally {
    expect(
      (
        await page.request.patch(`/api/v1/seats/${original.id}`, {
          headers,
          data: { organizationId: original.organizationId ?? "" },
        })
      ).status(),
    ).toBe(204);
  }
});

test("다른 필드만 수정하면 구역을 보존하고 취소는 저장하지 않는다", async ({
  page,
}) => {
  const original = (await seats(page)).find(
    (s) => s.employeeName === "김개발",
  )!;
  const org = (await organizations(page))[0];
  const headers = { "X-CSRF-Token": await csrfToken(page) };
  try {
    await edit(page, original.id);
    await choose(page, org.name);
    await save(page, "PATCH", original.id);
    await reopen(page, original.id, org.name);
    await page.getByRole("spinbutton", { name: "회전 각도" }).fill("5");
    await save(page, "PATCH", original.id);
    const changed = (await seats(page)).find((s) => s.id === original.id)!;
    expect(changed.organizationId).toBe(org.id);
    expect(changed.rotation).toBe(5);
    await reopen(page, original.id, org.name);
    const writes: string[] = [];
    page.on("request", (r) => {
      if (
        ["POST", "PATCH", "DELETE"].includes(r.method()) &&
        new URL(r.url()).pathname.startsWith("/api/v1/seats")
      )
        writes.push(r.method());
    });
    await choose(page, "미지정");
    await cancel(page);
    expect((await seats(page)).find((s) => s.id === original.id)).toEqual(
      changed,
    );
    await reopen(page, original.id, org.name);
    await cancel(page);
    await page.getByRole("button", { name: "좌석 추가", exact: true }).click();
    await choose(page, org.name);
    await cancel(page);
    expect(writes).toEqual([]);
  } finally {
    expect(
      (
        await page.request.patch(`/api/v1/seats/${original.id}`, {
          headers,
          data: {
            organizationId: original.organizationId ?? "",
            rotation: original.rotation,
          },
        })
      ).status(),
    ).toBe(204);
  }
});
