import { describe, expect, it } from "vitest";
import { toCSV } from "./format";
import {
  EMPLOYEE_CSV_HEADERS,
  employeeCsvRows,
  employeeSeatLabel,
  employeeStatusLabel,
} from "./employeeExport";
import type { Employee } from "../types";

const person = (over: Partial<Employee> = {}): Employee => ({
  id: "e1",
  employeeNo: "100001",
  name: "홍길동",
  email: "hong@example.com",
  organizationName: "개발팀",
  title: "책임",
  position: "팀원",
  workplace: "본사",
  status: "active",
  seatId: "s1",
  seatNo: "HQ-3F-001",
  ...over,
});

describe("employeeStatusLabel", () => {
  it("재직상태를 화면과 같은 한글 라벨로 바꾼다", () => {
    expect(employeeStatusLabel("active")).toBe("재직");
    expect(employeeStatusLabel("leave")).toBe("휴직");
    expect(employeeStatusLabel("retired")).toBe("퇴직");
  });

  it("모르는 값은 표가 해 온 대로 퇴직으로 읽는다", () => {
    // 표는 active/leave 가 아니면 모두 "퇴직"으로 보여 왔다. 이 함수로 옮기면서
    // 그 갈래를 바꾸면 같은 데이터가 화면에서 다르게 읽히므로 그대로 둔다.
    expect(employeeStatusLabel("")).toBe("퇴직");
    expect(employeeStatusLabel("unknown")).toBe("퇴직");
  });
});

describe("employeeSeatLabel", () => {
  it("좌석이 없으면 미배정이다", () => {
    expect(employeeSeatLabel(undefined)).toBe("미배정");
    expect(employeeSeatLabel("")).toBe("미배정");
  });

  it("좌석이 있으면 좌석 번호 그대로다", () => {
    expect(employeeSeatLabel("HQ-3F-001")).toBe("HQ-3F-001");
  });
});

describe("EMPLOYEE_CSV_HEADERS", () => {
  it("좌석을 뺀 모든 열 이름이 직원 양식과 같은 말이다", () => {
    // 같은 값이 화면·직원 양식·파일에서 같은 이름으로 읽히게 한다. 이름을 새로
    // 지으면(표의 머리글 "조직" 처럼) 양식과 어긋나 같은 열을 두 이름으로 읽게
    // 된다. 파일 자체는 가져오기 입력이 아니다 — 조직코드 열이 없어 그대로
    // 올리면 같은 이름의 조직이 새로 생긴다(employeeExport.ts 머리주석).
    expect(EMPLOYEE_CSV_HEADERS).toEqual([
      "이름",
      "사번",
      "이메일",
      "조직명",
      "직급",
      "직책",
      "근무지",
      "좌석",
      "재직상태",
    ]);
  });
});

describe("employeeCsvRows", () => {
  it("열 순서대로 한 사람을 한 행으로 만든다", () => {
    // 직급은 title, 직책은 position 이다. 서버 파서와 직원 양식이 그렇게 읽는다.
    expect(employeeCsvRows([person()])).toEqual([
      [
        "홍길동",
        "100001",
        "hong@example.com",
        "개발팀",
        "책임",
        "팀원",
        "본사",
        "HQ-3F-001",
        "재직",
      ],
    ]);
  });

  it("행 수가 목록의 사람 수와 같다", () => {
    const rows = employeeCsvRows([
      person({ id: "a", employeeNo: "1" }),
      person({ id: "b", employeeNo: "2" }),
      person({ id: "c", employeeNo: "3" }),
    ]);
    expect(rows).toHaveLength(3);
    expect(rows.every((row) => row.length === EMPLOYEE_CSV_HEADERS.length)).toBe(
      true,
    );
  });

  it("좌석 없는 사람은 미배정이고 나머지 빈 값은 빈 칸이다", () => {
    // 표는 빈 값에 "-" 를 보여주지만 그 자리 표시를 파일에 넣으면 안 된다.
    // csvCell 이 "-" 로 시작하는 값을 수식으로 보고 "'-" 로 바꿔 버린다.
    const [row] = employeeCsvRows([
      person({
        email: undefined,
        organizationName: undefined,
        title: undefined,
        position: undefined,
        workplace: undefined,
        seatId: undefined,
        seatNo: undefined,
        status: "leave",
      }),
    ]);
    expect(row).toEqual([
      "홍길동",
      "100001",
      "",
      "",
      "",
      "",
      "",
      "미배정",
      "휴직",
    ]);
  });

  it("표의 상태 칸과 같은 함수에서 재직상태를 얻는다", () => {
    const people = [
      person({ status: "active" }),
      person({ status: "leave" }),
      person({ status: "retired" }),
    ];
    expect(employeeCsvRows(people).map((row) => row.at(-1))).toEqual(
      people.map((employee) => employeeStatusLabel(employee.status)),
    );
  });
});

describe("toCSV 를 거친 직원 목록", () => {
  const csv = (items: Employee[]) =>
    toCSV(EMPLOYEE_CSV_HEADERS, employeeCsvRows(items));

  it("BOM 으로 시작하고 줄을 CRLF 로 잇는다", () => {
    const text = csv([person()]);
    expect(text.startsWith("﻿")).toBe(true);
    expect(text.split("\r\n")).toHaveLength(2);
    expect(text).toContain("﻿이름,사번,이메일,조직명,직급,직책,근무지,좌석,재직상태");
  });

  it("수식으로 읽힐 값은 막고 쉼표가 든 값은 감싼다", () => {
    const text = csv([
      person({ name: "=HYPERLINK(1)", organizationName: "영업, 2팀" }),
    ]);
    expect(text).toContain("'=HYPERLINK(1)");
    expect(text).toContain('"영업, 2팀"');
  });
});
