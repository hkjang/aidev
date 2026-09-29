import { describe, expect, it } from "vitest";
import {
  centerOn,
  clampCenter,
  FIT_VIEW,
  fitScaleFor,
  focusOn,
  MAX_ZOOM,
  MIN_ZOOM,
  toViewBox,
  viewRectFor,
  visibleSize,
  zoomAround,
} from "./mapView";

// 세로로 긴 도면과 가로로 긴 뷰포트를 섞어 종횡비 처리를 함께 검증한다.
const canvas = { width: 714, height: 1000 }; // A4 세로 비율
const wideCanvas = { width: 1000, height: 400 };
const viewport = { width: 900, height: 600 };

describe("fitScaleFor", () => {
  it("도면 전체가 들어가는 쪽 배율을 고른다", () => {
    // 세로 도면은 높이가 병목이다: min(900/714, 600/1000) = 0.6
    expect(fitScaleFor(viewport, canvas)).toBeCloseTo(0.6, 10);
    // 납작한 도면은 너비가 병목이다: min(900/1000, 600/400) = 0.9
    expect(fitScaleFor(viewport, wideCanvas)).toBeCloseTo(0.9, 10);
  });
  it("측정 전(크기 0)에는 1로 물러난다", () => {
    expect(fitScaleFor({ width: 0, height: 0 }, canvas)).toBe(1);
  });
});

describe("visibleSize", () => {
  it("전체 보기에서는 도면이 한 축을 가득 채운다", () => {
    const visible = visibleSize(viewport, canvas, 1);
    // 높이가 병목이므로 세로는 도면 전체, 가로는 여유가 생긴다.
    expect(visible.height).toBeCloseTo(canvas.height, 6);
    expect(visible.width).toBeGreaterThan(canvas.width);
  });
  it("뷰포트와 종횡비가 같아 레터박스가 생기지 않는다", () => {
    const visible = visibleSize(viewport, canvas, 2.5);
    expect(visible.width / visible.height).toBeCloseTo(
      viewport.width / viewport.height,
      10,
    );
  });
  it("확대하면 보이는 영역이 반비례로 줄어든다", () => {
    const one = visibleSize(viewport, canvas, 1);
    const four = visibleSize(viewport, canvas, 4);
    expect(one.width / four.width).toBeCloseTo(4, 10);
  });
});

describe("clampCenter", () => {
  it("도면이 화면보다 크면 가장자리를 넘지 않는다", () => {
    const visible = visibleSize(viewport, canvas, 4);
    const clamped = clampCenter(-3, 9, visible, canvas);
    const halfX = visible.width / canvas.width / 2;
    const halfY = visible.height / canvas.height / 2;
    expect(clamped.cx).toBeCloseTo(halfX, 10);
    expect(clamped.cy).toBeCloseTo(1 - halfY, 10);
  });
  it("도면이 화면보다 작은 축은 가운데로 고정한다", () => {
    // 전체 보기에서 가로는 도면보다 시야가 넓다 → 항상 0.5
    const visible = visibleSize(viewport, canvas, 1);
    expect(clampCenter(0.1, 0.5, visible, canvas).cx).toBe(0.5);
  });
});

describe("zoomAround", () => {
  it("커서 아래 지점이 화면에서 같은 자리에 남는다", () => {
    const pivot = { x: 0.3, y: 0.72 };
    const before = { cx: 0.5, cy: 0.5, zoom: 2 };
    const after = zoomAround(before, 5, viewport, canvas, pivot);
    // 화면상의 위치 = (지점 - 중심) * 배율. 이 값이 보존되어야 한다.
    const screenBefore = {
      x: (pivot.x - before.cx) * before.zoom,
      y: (pivot.y - before.cy) * before.zoom,
    };
    const screenAfter = {
      x: (pivot.x - after.cx) * after.zoom,
      y: (pivot.y - after.cy) * after.zoom,
    };
    expect(screenAfter.x).toBeCloseTo(screenBefore.x, 10);
    expect(screenAfter.y).toBeCloseTo(screenBefore.y, 10);
  });
  it("배율 상한과 하한을 넘지 않는다", () => {
    expect(zoomAround(FIT_VIEW, 999, viewport, canvas).zoom).toBe(MAX_ZOOM);
    expect(
      zoomAround({ ...FIT_VIEW, zoom: 4 }, 0.01, viewport, canvas).zoom,
    ).toBe(MIN_ZOOM);
  });
  it("배율이 그대로면 같은 뷰를 돌려준다", () => {
    const view = { cx: 0.4, cy: 0.6, zoom: 3 };
    expect(zoomAround(view, 3, viewport, canvas)).toBe(view);
  });
  it("축소해 전체가 보이면 중심이 가운데로 돌아온다", () => {
    const zoomed = { cx: 0.9, cy: 0.9, zoom: 8 };
    const out = zoomAround(zoomed, MIN_ZOOM, viewport, canvas);
    expect(out.cx).toBe(0.5);
    expect(out.cy).toBeCloseTo(0.5, 10);
  });
});

describe("viewRectFor", () => {
  it("전체 보기에서 도면을 모두 담는다", () => {
    const rect = viewRectFor(FIT_VIEW, viewport, canvas);
    expect(rect.x).toBeLessThanOrEqual(0);
    expect(rect.y).toBeCloseTo(0, 6);
    expect(rect.x + rect.width).toBeGreaterThanOrEqual(canvas.width);
    expect(rect.y + rect.height).toBeCloseTo(canvas.height, 6);
  });
  it("확대해도 사각형이 도면 밖으로 나가지 않는다", () => {
    const rect = viewRectFor({ cx: 0.98, cy: 0.02, zoom: 6 }, viewport, canvas);
    expect(rect.x).toBeGreaterThanOrEqual(-0.001);
    expect(rect.y).toBeGreaterThanOrEqual(-0.001);
    expect(rect.x + rect.width).toBeLessThanOrEqual(canvas.width + 0.001);
    expect(rect.y + rect.height).toBeLessThanOrEqual(canvas.height + 0.001);
  });
  it("viewBox 문자열은 네 값을 순서대로 잇는다", () => {
    expect(toViewBox({ x: 1, y: 2, width: 3, height: 4 })).toBe("1 2 3 4");
  });
});

describe("centerOn / focusOn", () => {
  it("centerOn은 배율을 바꾸지 않는다", () => {
    const view = { cx: 0.5, cy: 0.5, zoom: 3 };
    expect(centerOn(view, 0.2, 0.8, viewport, canvas).zoom).toBe(3);
  });
  it("focusOn은 좌석 중심을 화면 중앙에 두고 최소 배율까지 확대한다", () => {
    const seat = { x: 0.4, y: 0.4, width: 0.04, height: 0.03 };
    const out = focusOn(FIT_VIEW, seat, viewport, canvas);
    expect(out.zoom).toBeGreaterThanOrEqual(3);
    expect(out.cx).toBeCloseTo(seat.x + seat.width / 2, 10);
    expect(out.cy).toBeCloseTo(seat.y + seat.height / 2, 10);
  });
  it("focusOn은 이미 더 확대돼 있으면 배율을 낮추지 않는다", () => {
    const seat = { x: 0.5, y: 0.5, width: 0.04, height: 0.03 };
    expect(
      focusOn({ cx: 0.5, cy: 0.5, zoom: 7 }, seat, viewport, canvas).zoom,
    ).toBe(7);
  });
});
