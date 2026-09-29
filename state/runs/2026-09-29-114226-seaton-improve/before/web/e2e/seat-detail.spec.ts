import { expect, test, type Page } from "@playwright/test";
import { csrfToken, login, mapCanvas } from "./helpers";

/**
 * 좌석 상세가 보여주는 값은 좌석 응답에서만 나와야 한다.
 *
 * 예전 상세는 '조직'에 좌석에 지정된 구역을 먼저 넣고, 직원 소속과 근무지는 왼쪽
 * 검색 결과에서 찾았다. 그래서 (1) 다른 팀 구역에 앉은 사람이 자기 팀이 아니라
 * 좌석 구역 이름으로 보였고 — 도면 색·툴팁은 직원 소속으로 칠하는데 상세만 달랐다 —
 * (2) 검색하지 않고 좌석을 고르면 소속과 근무지를 아예 읽지 못했다. 이 검증은
 * 한 번도 검색하지 않은 채 좌석을 골라 두 값을 확인한다.
 */

type Organization = { id: string; name: string };
type Employee = {
  id: string;
  employeeNo: string;
  name: string;
  email?: string;
  organizationId?: string;
  title?: string;
  position?: string;
  workplace?: string;
  status: string;
};
type Seat = {
  id: string;
  seatNo: string;
  organizationId?: string;
  employeeId?: string;
  employeeName?: string;
  employeeOrganizationId?: string;
  employeeWorkplace?: string;
};

const WORKPLACE = "판교 오피스";

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
const employee = async (page: Page, name: string): Promise<Employee> => {
  const response = await page.request.get(
    `/api/v1/employees?q=${encodeURIComponent(name)}`,
  );
  expect(response.ok()).toBeTruthy();
  return (await response.json()).items.find((e: Employee) => e.name === name);
};
/** 좌석 구역을 서버에서 바꾼다. 빈 문자열이 해제다. */
const setZone = async (page: Page, id: string, organizationId: string | null) =>
  expect(
    (
      await page.request.patch(`/api/v1/seats/${id}`, {
        headers: { "X-CSRF-Token": await csrfToken(page) },
        data: { organizationId: organizationId ?? "" },
      })
    ).status(),
  ).toBe(204);
/** 직원 등록은 사번 기준 덮어쓰기라, 되돌릴 때도 같은 본문을 그대로 다시 보낸다. */
const saveEmployee = async (page: Page, record: Employee) =>
  expect(
    (
      await page.request.post("/api/v1/employees", {
        headers: { "X-CSRF-Token": await csrfToken(page) },
        data: {
          id: record.id,
          employeeNo: record.employeeNo,
          name: record.name,
          email: record.email ?? "",
          organizationId: record.organizationId ?? null,
          title: record.title ?? "",
          position: record.position ?? "",
          workplace: record.workplace ?? "",
          status: record.status,
        },
      })
    ).ok(),
  ).toBeTruthy();

const detail = (page: Page) =>
  page.getByRole("region", { name: "좌석 상세", exact: true });
const value = (page: Page, label: string) =>
  detail(page).locator(`[data-info="${label}"]`);
const shape = (page: Page, id: string) =>
  mapCanvas(page).locator(`[data-seat-id="${id}"]`);
/** 검색창을 건드리지 않고 좌석만 고른다. */
const pick = async (page: Page, id: string) => {
  await page.goto("/");
  await mapCanvas(page).waitFor();
  await expect(page.getByPlaceholder("이름, 사번, 조직 검색")).toHaveValue("");
  await shape(page, id).click();
  await expect(detail(page)).toBeVisible();
};

test("검색하지 않아도 상세가 착석자 소속과 지정 구역을 구분해 보여준다", async ({
  page,
}) => {
  await login(page);
  const seat = (await seats(page)).find((s) => s.employeeName === "김개발")!;
  const orgs = await organizations(page);
  const own = orgs.find((o) => o.id === seat.employeeOrganizationId)!;
  const other = orgs.find((o) => o.id !== own.id)!;
  const before = await employee(page, "김개발");
  const zoneBefore = seat.organizationId ?? null;
  try {
    // 남의 팀 구역에 앉은 상태를 만든다. 직원 소속은 그대로다.
    await setZone(page, seat.id, other.id);
    await saveEmployee(page, { ...before, workplace: WORKPLACE });
    const refreshed = (await seats(page)).find((s) => s.id === seat.id)!;
    expect(refreshed.employeeOrganizationId).toBe(own.id);
    expect(refreshed.organizationId).toBe(other.id);
    // 서버가 근무지를 좌석 응답에 실어 주어야 화면이 검색 없이 읽을 수 있다.
    expect(refreshed.employeeWorkplace).toBe(WORKPLACE);

    await pick(page, seat.id);
    // '조직'은 좌석 구역이 아니라 실제로 앉은 사람의 소속이다.
    await expect(value(page, "조직")).toHaveText(own.name);
    await expect(value(page, "지정 구역")).toHaveText(
      `${other.name} · 구역 불일치`,
    );
    await expect(value(page, "근무지")).toHaveText(WORKPLACE);
    // 도면 툴팁도 같은 조직을 말해야 한다.
    await expect(shape(page, seat.id)).toHaveAttribute(
      "aria-label",
      new RegExp(`${own.name}.*구역 불일치\\(지정 ${other.name}\\)`),
    );

    // 구역을 직원 소속과 같게 두면 불일치는 사라지고 구역은 남는다.
    await setZone(page, seat.id, own.id);
    await pick(page, seat.id);
    await expect(value(page, "조직")).toHaveText(own.name);
    await expect(value(page, "지정 구역")).toHaveText(own.name);
  } finally {
    await setZone(page, seat.id, zoneBefore);
    await saveEmployee(page, before);
  }
});

test("구역을 해제하면 지정 구역 행이 사라지고 빈 좌석에도 구역이 보인다", async ({
  page,
}) => {
  await login(page);
  const all = await seats(page);
  const assigned = all.find((s) => s.employeeName === "김개발")!;
  const empty = all.find((s) => !s.employeeId && !s.organizationId)!;
  const org = (await organizations(page))[0];
  const zoneBefore = assigned.organizationId ?? null;
  try {
    await setZone(page, assigned.id, null);
    await pick(page, assigned.id);
    await expect(value(page, "조직")).toBeVisible();
    await expect(value(page, "지정 구역")).toHaveCount(0);

    // 구역은 좌석의 성질이므로 아무도 앉지 않은 좌석에서도 보여야 한다.
    await setZone(page, empty.id, org.id);
    await pick(page, empty.id);
    await expect(detail(page).getByText("배정된 직원이 없습니다")).toBeVisible();
    await expect(value(page, "지정 구역")).toHaveText(org.name);
    await expect(value(page, "조직")).toHaveCount(0);
  } finally {
    await setZone(page, assigned.id, zoneBefore);
    await setZone(page, empty.id, null);
  }
});
