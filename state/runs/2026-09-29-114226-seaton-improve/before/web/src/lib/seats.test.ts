import { describe, expect, it } from "vitest";
import type { Seat } from "../types";
import {
  commonSeatPrefix,
  deriveGrid,
  matchesFilter,
  needsReviewSeat,
  seatColor,
  seatHighlighted,
  nextSeatInDirection,
  seatLabelLayout,
  seatOrgDetail,
  seatOrgId,
  seatsInReadingOrder,
  seatUnavailable,
  shortSeatNo,
  zoomTier,
  zoneMismatched,
  type SeatFilter,
} from "./seats";

const seat = (over: Partial<Seat> = {}): Seat => ({
  id: over.id ?? "s1",
  floorMapId: "m1",
  seatNo: "A-01",
  type: "fixed",
  status: "available",
  x: 0.1,
  y: 0.1,
  width: 0.04,
  height: 0.05,
  rotation: 0,
  ...over,
});
const set = (...f: SeatFilter[]) => new Set<SeatFilter>(f);

describe("seatUnavailable", () => {
  it("type과 status 어느 쪽으로 지정해도 사용 불가로 본다", () => {
    expect(seatUnavailable(seat({ type: "unavailable" }))).toBe(true);
    expect(seatUnavailable(seat({ status: "unavailable" }))).toBe(true);
    expect(seatUnavailable(seat())).toBe(false);
  });
});

describe("seatColor", () => {
  it("사용 불가는 회색이 우선한다", () => {
    expect(seatColor(seat({ status: "unavailable", employeeId: "e1" }))).toBe(
      "#8796A1",
    );
  });
  it("배정·공용·빈 좌석을 구분한다", () => {
    expect(seatColor(seat({ employeeId: "e1" }))).toBe("#087E8B");
    expect(seatColor(seat({ type: "shared" }))).toBe("#3478C8");
    expect(seatColor(seat())).toBe("#FFFFFF");
  });
});

describe("matchesFilter", () => {
  it("필터가 없으면 전부 통과시킨다", () => {
    expect(matchesFilter(seat(), set())).toBe(true);
  });
  it("빈 좌석 필터는 색상 판정과 같은 기준을 쓴다", () => {
    // 회귀 방지: status로만 사용 불가인 좌석이 빈 좌석으로 강조되면 안 된다.
    const grey = seat({ status: "unavailable" });
    expect(seatColor(grey)).toBe("#8796A1");
    expect(matchesFilter(grey, set("available"))).toBe(false);
    expect(matchesFilter(seat({ type: "unavailable" }), set("available"))).toBe(
      false,
    );
    expect(matchesFilter(seat(), set("available"))).toBe(true);
  });
  it("배정 필터는 착석자가 있는 좌석만 고른다", () => {
    expect(matchesFilter(seat({ employeeId: "e1" }), set("assigned"))).toBe(
      true,
    );
    expect(matchesFilter(seat(), set("assigned"))).toBe(false);
  });
  it("검토 필요는 자동 승인선 아래만 고른다", () => {
    expect(matchesFilter(seat({ confidence: 0.9 }), set("review"))).toBe(true);
    expect(matchesFilter(seat({ confidence: 0.96 }), set("review"))).toBe(
      false,
    );
    expect(needsReviewSeat(seat())).toBe(false);
  });
  it("여러 필터는 합집합으로 동작한다", () => {
    const filters = set("assigned", "review");
    expect(matchesFilter(seat({ confidence: 0.8 }), filters)).toBe(true);
    expect(matchesFilter(seat({ employeeId: "e1" }), filters)).toBe(true);
    expect(matchesFilter(seat(), filters)).toBe(false);
  });
});

describe("zoneMismatched / seatOrgId", () => {
  it("지정 구역과 착석자 소속이 다를 때만 불일치다", () => {
    expect(
      zoneMismatched(
        seat({ organizationId: "A", employeeOrganizationId: "B" }),
      ),
    ).toBe(true);
    expect(
      zoneMismatched(
        seat({ organizationId: "A", employeeOrganizationId: "A" }),
      ),
    ).toBe(false);
    // 한쪽이 비면 비교할 수 없으므로 불일치로 보지 않는다.
    expect(zoneMismatched(seat({ organizationId: "A" }))).toBe(false);
    expect(zoneMismatched(seat({ employeeOrganizationId: "B" }))).toBe(false);
  });
  it("대표 조직은 착석자 소속이 우선이다", () => {
    expect(
      seatOrgId(seat({ organizationId: "A", employeeOrganizationId: "B" })),
    ).toBe("B");
    expect(seatOrgId(seat({ organizationId: "A" }))).toBe("A");
    expect(seatOrgId(seat())).toBeNull();
  });
});

describe("seatOrgDetail", () => {
  it("착석자 소속을 지정 구역으로 대체하지 않는다", () => {
    const detail = seatOrgDetail(
      seat({
        employeeId: "e1",
        organizationId: "A",
        organizationName: "영업팀",
        employeeOrganizationId: "B",
        employeeOrganizationName: "개발팀",
      }),
    );
    expect(detail.employee).toBe("개발팀");
  });
  it("착석자 소속을 모르면 지정 구역이 아니라 정보 없음이다", () => {
    const detail = seatOrgDetail(
      seat({ employeeId: "e1", organizationId: "A", organizationName: "영업팀" }),
    );
    expect(detail.employee).toBe("정보 없음");
  });
  it("지정 구역은 따로 내보내고 불일치를 함께 알린다", () => {
    expect(
      seatOrgDetail(
        seat({
          employeeId: "e1",
          organizationId: "A",
          organizationName: "영업팀",
          employeeOrganizationId: "B",
          employeeOrganizationName: "개발팀",
        }),
      ).zone,
    ).toBe("영업팀 · 구역 불일치");
    // 같은 조직이면 알릴 불일치가 없다.
    expect(
      seatOrgDetail(
        seat({
          employeeId: "e1",
          organizationId: "A",
          organizationName: "영업팀",
          employeeOrganizationId: "A",
          employeeOrganizationName: "영업팀",
        }),
      ).zone,
    ).toBe("영업팀");
  });
  it("지정 구역이 없으면 빈 문자열이라 행을 그리지 않는다", () => {
    expect(seatOrgDetail(seat()).zone).toBe("");
  });
});

describe("seatHighlighted", () => {
  it("필터와 조직 강조를 모두 만족해야 한다", () => {
    const s = seat({ employeeId: "e1", employeeOrganizationId: "DEV" });
    expect(seatHighlighted(s, set("assigned"), "DEV")).toBe(true);
    expect(seatHighlighted(s, set("assigned"), "SALES")).toBe(false);
    expect(seatHighlighted(s, set("available"), "DEV")).toBe(false);
    expect(seatHighlighted(s, set(), null)).toBe(true);
  });
});

describe("deriveGrid", () => {
  it("좌석이 하나면 간격을 알 수 없어 실패한다", () => {
    expect(deriveGrid([seat()])).toBeNull();
  });
  it("가로·세로로 떨어진 좌석에서 간격을 읽는다", () => {
    const grid = deriveGrid([
      seat({ id: "a", x: 0.1, y: 0.2 }),
      seat({ id: "b", x: 0.25, y: 0.2 }),
      seat({ id: "c", x: 0.1, y: 0.32 }),
    ]);
    expect(grid).not.toBeNull();
    expect(grid!.pitchX).toBeCloseTo(0.15, 6);
    expect(grid!.pitchY).toBeCloseTo(0.12, 6);
  });
  it("가장 좁은 간격을 주기로 삼는다", () => {
    const grid = deriveGrid([
      seat({ id: "a", x: 0.1, y: 0.1 }),
      seat({ id: "b", x: 0.2, y: 0.1 }),
      seat({ id: "c", x: 0.5, y: 0.1 }),
      seat({ id: "d", x: 0.1, y: 0.3 }),
    ]);
    expect(grid!.pitchX).toBeCloseTo(0.1, 6);
  });
  it("원점은 주기 안으로 접힌다", () => {
    const grid = deriveGrid([
      seat({ id: "a", x: 0.35, y: 0.5 }),
      seat({ id: "b", x: 0.45, y: 0.5 }),
      seat({ id: "c", x: 0.35, y: 0.7 }),
    ]);
    // 0.35 를 주기 0.1 로 접으면 0.05 근처가 된다.
    expect(grid!.originX).toBeGreaterThanOrEqual(0);
    expect(grid!.originX).toBeLessThan(grid!.pitchX);
    expect(grid!.originY).toBeLessThan(grid!.pitchY);
  });
  it("한 축으로만 늘어선 좌석은 좌석 크기를 주기로 되돌린다", () => {
    const grid = deriveGrid([
      seat({ id: "a", x: 0.1, y: 0.2, width: 0.05, height: 0.06 }),
      seat({ id: "b", x: 0.3, y: 0.2, width: 0.05, height: 0.06 }),
    ]);
    expect(grid!.pitchX).toBeCloseTo(0.2, 6);
    expect(grid!.pitchY).toBeCloseTo(0.06, 6);
  });
  it("간격이 최소 주기보다 좁으면 격자를 만들지 않는다", () => {
    expect(
      deriveGrid([
        seat({ id: "a", x: 0.1, y: 0.1, width: 0.001, height: 0.001 }),
        seat({ id: "b", x: 0.1005, y: 0.1, width: 0.001, height: 0.001 }),
      ]),
    ).toBeNull();
  });
});

describe("commonSeatPrefix", () => {
  it("구분자 단위로 공통 접두사를 찾는다", () => {
    expect(commonSeatPrefix(["HQ-3F-001", "HQ-3F-002", "HQ-3F-010"])).toBe(
      "HQ-3F-",
    );
  });

  it("토큰 중간에서 자르지 않는다", () => {
    // 공통 문자열은 "A-1"이지만 "A-"까지만 접두사로 인정해야 한다.
    expect(commonSeatPrefix(["A-11", "A-12"])).toBe("A-");
  });

  it("공통 부분이 없으면 빈 문자열", () => {
    expect(commonSeatPrefix(["A-01", "B-01"])).toBe("");
  });

  it("구분자가 없으면 떼지 않는다", () => {
    expect(commonSeatPrefix(["1001", "1002"])).toBe("");
  });

  it("좌석이 하나뿐이면 떼지 않는다", () => {
    expect(commonSeatPrefix(["HQ-3F-001"])).toBe("");
  });

  it("접두사를 떼면 빈 이름이 되는 좌석이 있으면 떼지 않는다", () => {
    expect(commonSeatPrefix(["A-", "A-1"])).toBe("");
  });
});

describe("shortSeatNo", () => {
  it("접두사를 뗀다", () => {
    expect(shortSeatNo("HQ-3F-001", "HQ-3F-")).toBe("001");
  });

  it("접두사가 없으면 원래 번호", () => {
    expect(shortSeatNo("HQ-3F-001", "")).toBe("HQ-3F-001");
    expect(shortSeatNo("B-01", "HQ-3F-")).toBe("B-01");
  });
});

describe("zoomTier", () => {
  it("확대하지 않았으면 1단계", () => {
    expect(zoomTier(1)).toBe(1);
    expect(zoomTier(0.4)).toBe(1);
    expect(zoomTier(1.3)).toBe(1);
  });

  it("가장 가까운 2의 거듭제곱으로 뭉친다", () => {
    expect(zoomTier(1.5)).toBe(2);
    expect(zoomTier(3)).toBe(4);
    expect(zoomTier(8.7)).toBe(8);
  });

  it("최대 8단계에서 멈춘다", () => {
    expect(zoomTier(12)).toBe(8);
  });
});

describe("seatLabelLayout", () => {
  it("확대해도 글자가 좌석을 뒤덮지 않는다", () => {
    const fit = seatLabelLayout("김개발", 40, 22, 1);
    const zoomed = seatLabelLayout("김개발", 40, 22, 4);
    // 화면상 크기(= fontSize × 배율)는 커지되 배율만큼 커지지는 않는다.
    expect(zoomed.fontSize * 4).toBeGreaterThan(fit.fontSize);
    expect(zoomed.fontSize * 4).toBeLessThan(fit.fontSize * 2.5);
    // 좌석 대비로는 반드시 작아져야 글자가 좌석을 뒤덮지 않는다.
    expect(zoomed.fontSize).toBeLessThan(fit.fontSize);
  });

  it("확대하면 잘렸던 글자가 더 보인다", () => {
    const fit = seatLabelLayout("최고참프론트엔드팀", 40, 22, 1);
    const zoomed = seatLabelLayout("최고참프론트엔드팀", 40, 22, 4);
    expect(fit.text).toContain("…");
    expect(zoomed.text).toBe("최고참프론트엔드팀");
  });

  it("너무 작은 좌석에는 라벨을 그리지 않는다", () => {
    expect(seatLabelLayout("A", 8, 5, 1).show).toBe(false);
  });

  it("작은 좌석도 확대하면 라벨이 나타난다", () => {
    expect(seatLabelLayout("A1", 12, 8, 4).show).toBe(true);
  });
});

describe("nextSeatInDirection", () => {
  // 3×2 격자. 좌석 크기 0.1×0.1, 간격 0.2.
  const grid = [0, 1].flatMap((row) =>
    [0, 1, 2].map((col) =>
      seat({
        id: `r${row}c${col}`,
        x: 0.1 + col * 0.2,
        y: 0.1 + row * 0.2,
        width: 0.1,
        height: 0.1,
      }),
    ),
  );

  it("오른쪽 옆자리로 간다", () => {
    expect(nextSeatInDirection(grid, "r0c0", "right")?.id).toBe("r0c1");
  });

  it("아래 줄로 간다", () => {
    expect(nextSeatInDirection(grid, "r0c1", "down")?.id).toBe("r1c1");
  });

  it("끝에서는 더 가지 않는다", () => {
    expect(nextSeatInDirection(grid, "r0c2", "right")).toBeNull();
    expect(nextSeatInDirection(grid, "r0c0", "up")).toBeNull();
  });

  it("줄이 살짝 어긋나도 옆자리로 이어진다", () => {
    const skewed = [
      seat({ id: "a", x: 0.1, y: 0.1, width: 0.05, height: 0.05 }),
      seat({ id: "b", x: 0.3, y: 0.12, width: 0.05, height: 0.05 }),
      seat({ id: "c", x: 0.32, y: 0.4, width: 0.05, height: 0.05 }),
    ];
    expect(nextSeatInDirection(skewed, "a", "right")?.id).toBe("b");
  });

  it("모르는 좌석에서 시작하면 첫 좌석을 준다", () => {
    expect(nextSeatInDirection(grid, "없음", "right")?.id).toBe("r0c0");
  });
});

describe("seatsInReadingOrder", () => {
  it("위에서 아래, 왼쪽에서 오른쪽으로 정렬한다", () => {
    const seats = [
      seat({ id: "b", x: 0.5, y: 0.1, width: 0.1, height: 0.1 }),
      seat({ id: "c", x: 0.1, y: 0.4, width: 0.1, height: 0.1 }),
      seat({ id: "a", x: 0.1, y: 0.1, width: 0.1, height: 0.1 }),
    ];
    expect(seatsInReadingOrder(seats).map((s) => s.id)).toEqual([
      "a",
      "b",
      "c",
    ]);
  });
});
