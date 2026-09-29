/**
 * 좌석맵 뷰포트 기하.
 *
 * 좌석맵은 SVG viewBox를 직접 계산해 화면 이동과 확대를 다룬다. 이 계산은
 * 컴포넌트 상태와 무관한 순수 함수라 여기 모아 두고 단위 테스트한다.
 *
 * 좌표계는 두 가지를 오간다.
 * - 비율 좌표: 도면 대비 0~1. 좌석이 저장되는 좌표계이며 view의 중심도 이 단위다.
 * - 캔버스 단위: viewBox가 쓰는 단위. 도면 원본 비율을 유지한 정규화 픽셀이다.
 */

export type Size = { width: number; height: number };
/** cx, cy는 화면 중심의 비율 좌표. zoom은 전체 보기를 1로 둔 상대 배율. */
export type MapView = { cx: number; cy: number; zoom: number };
export type Rect = { x: number; y: number; width: number; height: number };

export const FIT_VIEW: MapView = { cx: 0.5, cy: 0.5, zoom: 1 };
export const MIN_ZOOM = 1;
export const MAX_ZOOM = 12;
/** 검색으로 좌석을 찾았을 때 최소한 이 배율까지는 확대해 보여준다. */
export const FOCUS_ZOOM = 3;

/** 도면 전체가 화면에 꼭 맞는 배율. 이 값을 1배로 삼는다. */
export const fitScaleFor = (viewport: Size, canvas: Size) => {
  if (!viewport.width || !viewport.height || !canvas.width || !canvas.height)
    return 1;
  return Math.min(
    viewport.width / canvas.width,
    viewport.height / canvas.height,
  );
};

/**
 * 현재 배율에서 화면에 들어오는 도면 영역의 크기(캔버스 단위).
 * 뷰포트와 종횡비가 같으므로 viewBox에 그대로 쓰면 레터박스가 생기지 않는다.
 */
export const visibleSize = (
  viewport: Size,
  canvas: Size,
  zoom: number,
): Size => {
  const scale = fitScaleFor(viewport, canvas) * zoom;
  if (!viewport.width || !viewport.height || scale <= 0)
    return { width: canvas.width, height: canvas.height };
  return { width: viewport.width / scale, height: viewport.height / scale };
};

/**
 * 화면 중심을 도면 안쪽으로 제한한다. 도면이 화면보다 작으면 가운데로 고정해
 * 여백이 한쪽으로 몰리지 않게 한다.
 */
export const clampCenter = (
  cx: number,
  cy: number,
  visible: Size,
  canvas: Size,
): { cx: number; cy: number } => {
  const halfX = visible.width / canvas.width / 2,
    halfY = visible.height / canvas.height / 2;
  return {
    cx: halfX >= 0.5 ? 0.5 : Math.min(1 - halfX, Math.max(halfX, cx)),
    cy: halfY >= 0.5 ? 0.5 : Math.min(1 - halfY, Math.max(halfY, cy)),
  };
};

/** viewBox가 그리는 실제 화면 사각형(캔버스 단위). 미니맵 표시도 같은 값을 쓴다. */
export const viewRectFor = (
  view: MapView,
  viewport: Size,
  canvas: Size,
): Rect => {
  const visible = visibleSize(viewport, canvas, view.zoom);
  const { cx, cy } = clampCenter(view.cx, view.cy, visible, canvas);
  return {
    x: cx * canvas.width - visible.width / 2,
    y: cy * canvas.height - visible.height / 2,
    width: visible.width,
    height: visible.height,
  };
};

export const toViewBox = (rect: Rect) =>
  `${rect.x} ${rect.y} ${rect.width} ${rect.height}`;

/**
 * pivot(비율 좌표)이 화면에서 같은 자리에 남도록 배율을 바꾼다.
 * 커서 아래 지점을 고정한 휠 확대가 이 성질에 기댄다.
 */
export const zoomAround = (
  view: MapView,
  nextZoom: number,
  viewport: Size,
  canvas: Size,
  pivot?: { x: number; y: number },
): MapView => {
  const zoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, nextZoom));
  if (zoom === view.zoom) return view;
  const anchor = pivot ?? { x: view.cx, y: view.cy };
  const ratio = view.zoom / zoom;
  const cx = anchor.x - (anchor.x - view.cx) * ratio;
  const cy = anchor.y - (anchor.y - view.cy) * ratio;
  return {
    zoom,
    ...clampCenter(cx, cy, visibleSize(viewport, canvas, zoom), canvas),
  };
};

/** 화면 중심을 비율 좌표로 옮긴다. 미니맵 클릭과 방향키 이동이 함께 쓴다. */
export const centerOn = (
  view: MapView,
  cx: number,
  cy: number,
  viewport: Size,
  canvas: Size,
): MapView => ({
  ...view,
  ...clampCenter(cx, cy, visibleSize(viewport, canvas, view.zoom), canvas),
});

/** 좌석을 화면 중앙으로 가져오고 최소 FOCUS_ZOOM 까지 확대한다. */
export const focusOn = (
  view: MapView,
  box: { x: number; y: number; width: number; height: number },
  viewport: Size,
  canvas: Size,
): MapView => {
  const zoom = Math.min(MAX_ZOOM, Math.max(view.zoom, FOCUS_ZOOM));
  return {
    zoom,
    ...clampCenter(
      box.x + box.width / 2,
      box.y + box.height / 2,
      visibleSize(viewport, canvas, zoom),
      canvas,
    ),
  };
};
