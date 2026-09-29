/**
 * 좌석맵 화면과 주소(쿼리)를 잇는 규칙.
 *
 * 좌석맵은 `?map=`·`?q=`·`?edit=1` 로 들어오는 깊은 링크를 읽고, 검색·도면
 * 선택을 다시 주소에 적는다. 읽는 쪽과 쓰는 쪽이 각자 `searchParams.get(...)`
 * 을 부르면 한쪽만 다듬어(예: 공백) 같은 주소를 다르게 읽는 일이 생긴다 — 이
 * 저장소에서 반복된 어긋남이다. 그래서 규칙을 여기 한 곳에 두고 양쪽이 쓴다.
 * `silentSso.ts` 의 `loginPathFor`/`returnToFrom` 이 같은 꼴의 짝이다.
 */

export interface SeatMapParams {
  /** 볼 도면(floor map) id. 없으면 빈 문자열. */
  mapId: string;
  /** 검색어. 앞뒤 공백을 떼고, 공백만 있으면 없는 것으로 친다. */
  query: string;
  /** 편집 모드로 열라는 표시. 읽기만 하고 화면에서 다시 쓰지는 않는다. */
  edit: boolean;
}

/** 주소에서 좌석맵이 보는 값만 추린다. */
export function readSeatMapParams(params: URLSearchParams): SeatMapParams {
  return {
    mapId: params.get("map")?.trim() ?? "",
    query: params.get("q")?.trim() ?? "",
    edit: params.get("edit") === "1",
  };
}

/**
 * 주어진 값만 갈아 끼운 새 쿼리.
 *
 * 준 키만 손대고 나머지(`edit`·아직 없는 미래의 키)는 그대로 둔다. 값이 비면
 * 키를 지워 `?q=` 같은 빈 껍데기를 주소에 남기지 않는다.
 */
export function writeSeatMapParams(
  prev: URLSearchParams,
  next: { mapId?: string; query?: string },
): URLSearchParams {
  const params = new URLSearchParams(prev);
  const put = (key: string, value: string | undefined) => {
    if (value === undefined) return;
    const trimmed = value.trim();
    if (trimmed) params.set(key, trimmed);
    else params.delete(key);
  };
  put("map", next.mapId);
  put("q", next.query);
  return params;
}

/**
 * "이 검색은 이미 했다"를 가리키는 키.
 *
 * 검색 제출이 주소에 `q` 를 쓰면 `q` 를 감시하는 effect 가 다시 깨어나 같은
 * 검색을 한 번 더 보낸다. 제출 경로가 이 키를 먼저 세워 두면 effect 는 조용히
 * 지나간다 — 그러려면 두 경로가 같은 입력에서 반드시 같은 키를 내야 한다.
 */
export function seatSearchKey(mapId: string, query: string): string {
  return `${mapId}:${query.trim()}`;
}
