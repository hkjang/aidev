/**
 * 직원 목록 조회 쿼리를 짓는 규칙.
 *
 * 화면의 필터(검색어·재직상태·배정상태·조직)는 여러 자리에서 조회를 부른다 —
 * 첫 로드, 검색 제출, Select 세 개, 가져오기 뒤 새로고침. 그 자리마다
 * `new URLSearchParams` 을 손으로 조립하면 한쪽만 키 이름을 틀리거나 빈 값을
 * `?status=` 로 흘려 서버가 다르게 읽는다 — 이 저장소에서 반복된 어긋남이다.
 * 그래서 키 이름과 빈 값 규칙을 여기 한 곳에 두고 부르는 쪽이 모두 이것을 쓴다.
 * `seatMapLink.ts`·`silentSso.ts`·`employeeExport.ts` 가 같은 꼴의 짝이다.
 *
 * 키 이름은 마음대로 짓지 않는다. 서버 `listEmployees`
 * (`internal/app/employees.go:59-63`)가 읽는 `q`·`organizationId`·`status`·
 * `assignment` 를 그대로 쓴다.
 */

/**
 * 한 번에 받아 오는 최대 인원.
 *
 * 서버는 `limit` 을 500 까지만 받고(`employees.go:65-67`) 전체 건수는 돌려주지
 * 않는다. 그래서 화면은 언제나 상한으로 받고 "최대 500명 표시" 를 안내한다 —
 * 조건에 상관없이 같은 값이어야 표에 보이는 것과 CSV 가 어긋나지 않는다.
 */
export const EMPLOYEE_QUERY_LIMIT = "500";

export interface EmployeeFilters {
  /** 이름·사번·이메일·조직명을 함께 훑는 자유 검색어. */
  q?: string;
  /** 재직상태 코드(active|leave|retired). 빈 값은 전체. */
  status?: string;
  /** 배정상태(assigned|unassigned). 빈 값은 전체. */
  assignment?: string;
  /** 조직 id. 빈 값은 전체 조직. 조직명이 아니라 id 다. */
  organizationId?: string;
}

/**
 * 주어진 필터로 `GET /api/v1/employees` 의 쿼리를 짓는다.
 *
 * 빈 값(공백만 있는 것도 포함)은 키를 아예 넣지 않는다. 서버는 `$1=''` 로 빈
 * 값을 전체로 치므로 넣어도 결과는 같지만, 키가 없으면 주소가 짧아 어떤 조건이
 * 걸려 있는지 눈으로 읽힌다.
 */
export function employeeQuery(filters: EmployeeFilters): URLSearchParams {
  const params = new URLSearchParams({ limit: EMPLOYEE_QUERY_LIMIT });
  const put = (key: string, value: string | undefined) => {
    const trimmed = value?.trim();
    if (trimmed) params.set(key, trimmed);
  };
  put("q", filters.q);
  put("organizationId", filters.organizationId);
  put("status", filters.status);
  put("assignment", filters.assignment);
  return params;
}
