/**
 * 직원 목록을 표와 파일로 내보내는 규칙.
 *
 * 같은 직원을 표의 칸과 CSV 의 열이 각자 문장으로 만들면(예: 표는 삼항으로
 * "재직", 내보내기는 다른 표) 같은 데이터가 화면과 파일에서 다르게 읽힌다 —
 * 이 저장소에서 반복된 어긋남이다. 그래서 라벨과 행 만들기를 여기 한 곳에 두고
 * `EmployeesPage` 의 표와 CSV 가 모두 이것을 쓴다. `seatMapLink.ts`·
 * `silentSso.ts` 가 같은 꼴의 짝이다.
 *
 * 열 이름은 마음대로 짓지 않는다. 같은 화면의 직원 양식
 * (`EmployeesPage.downloadTemplate`)과 서버 파서(`internal/app/employees.go:159`
 * 의 `find`)가 쓰는 말을 그대로 써서, 같은 값이 화면·양식·파일에서 같은 이름으로
 * 읽히게 한다. 그래서 직급은 title, 직책은 position 이고 조직 열은 표의
 * 머리글("조직")이 아니라 "조직명"이다.
 *
 * 다만 이 파일 자체는 가져오기 입력이 아니다. 조직을 가리키는 "조직코드"
 * (organizations.external_id)가 없어 그대로 올리면 `saveEmployee`
 * (employees.go:109-120)가 external='import:<조직명>' 으로 ON CONFLICT 을 놓쳐
 * **같은 이름의 새 조직을 만들고** 직원을 그리로 옮긴다(organizations.name 에
 * UNIQUE 가 없다 — migrations.sql:14-22). 조직을 바꿀 때는 조직코드가 있는 직원
 * 양식을 쓴다. USER_GUIDE 3.4 절이 그렇게 안내한다.
 */

import type { Employee } from "../types";

/** 재직상태 코드 → 화면 라벨. 서버가 employees.status 에 넣는 값이 키다. */
const STATUS_LABELS: Record<string, string> = {
  active: "재직",
  leave: "휴직",
  retired: "퇴직",
};

/**
 * 재직상태 한 칸. 표와 CSV 가 같은 값을 보게 하는 유일한 출처다.
 *
 * 모르는 값은 "퇴직"이다 — 표가 active/leave 가 아닌 값을 모두 그렇게 보여
 * 왔기 때문이고, 여기서 갈래를 바꾸면 같은 직원이 화면에서 다르게 읽힌다.
 */
export const employeeStatusLabel = (status: string) =>
  STATUS_LABELS[status] ?? "퇴직";

/** 좌석 한 칸. 좌석이 없는 것은 빈 칸이 아니라 "미배정"이라고 적는다. */
export const employeeSeatLabel = (seatNo?: string) => seatNo || "미배정";

/** 파일의 머리글. "좌석"은 파서가 모르는 열이지만 배정 현황을 눈으로 보려고 넣는다. */
export const EMPLOYEE_CSV_HEADERS = [
  "이름",
  "사번",
  "이메일",
  "조직명",
  "직급",
  "직책",
  "근무지",
  "좌석",
  "재직상태",
];

/**
 * 화면에 보이는 목록을 그대로 CSV 행으로 옮긴다. 다시 조회하지 않으므로 표의
 * 행 수와 파일의 데이터 행 수가 같다.
 *
 * 빈 값은 빈 칸으로 둔다. 표가 보여주는 자리 표시 "-" 를 넣으면 `csvCell` 이
 * 수식으로 시작하는 값으로 보고 "'-" 로 바꿔 스프레드시트에 그대로 남는다.
 */
export const employeeCsvRows = (items: Employee[]): string[][] =>
  items.map((employee) => [
    employee.name,
    employee.employeeNo,
    employee.email ?? "",
    employee.organizationName ?? "",
    employee.title ?? "",
    employee.position ?? "",
    employee.workplace ?? "",
    employeeSeatLabel(employee.seatNo),
    employeeStatusLabel(employee.status),
  ]);
