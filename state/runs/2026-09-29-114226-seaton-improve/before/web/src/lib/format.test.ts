import { describe, expect, it } from "vitest";
import {
  absoluteTime,
  csvCell,
  dayRangeToISO,
  localDateStamp,
  startOfLocalDay,
  relativeTime,
  safeFileName,
  SOURCE_LABELS,
  sourceLabel,
  toCSV,
} from "./format";

describe("sourceLabel", () => {
  it("원문 값을 사람이 읽는 말로 바꾼다", () => {
    expect(sourceLabel("manual")).toBe("수동 배정");
    expect(sourceLabel("hr_sync")).toBe("인사 동기화");
  });
  it("서버가 실제로 쓰는 값을 모두 다룬다", () => {
    // 이 목록이 서버와 어긋나면 관리자가 고를 수 없는 방식이 생긴다.
    // 값 출처: seats.go, sync.go, mcp.go, dashboard.go 의 source 인자.
    for (const source of [
      "manual",
      "bulk",
      "dashboard",
      "dashboard_bulk",
      "hr_sync",
      "mcp",
    ])
      expect(SOURCE_LABELS[source]).toBeTruthy();
  });
  it("모르는 값은 그대로 보여주고, 빈 값은 대시로 바꾼다", () => {
    expect(sourceLabel("unknown_source")).toBe("unknown_source");
    expect(sourceLabel("")).toBe("-");
  });
});

describe("relativeTime", () => {
  const now = new Date("2026-08-26T12:00:00Z");
  it("최근일수록 상대 시간으로 보여준다", () => {
    expect(relativeTime("2026-08-26T11:59:30Z", now)).toBe("방금");
    expect(relativeTime("2026-08-26T11:30:00Z", now)).toBe("30분 전");
    expect(relativeTime("2026-08-26T09:00:00Z", now)).toBe("3시간 전");
    expect(relativeTime("2026-08-24T12:00:00Z", now)).toBe("2일 전");
  });
  it("일주일이 넘으면 절대 날짜로 바꾼다", () => {
    expect(relativeTime("2026-07-01T12:00:00Z", now)).toContain("2026");
  });
  it("미래 시각과 잘못된 값을 안전하게 다룬다", () => {
    expect(relativeTime("2026-08-26T12:05:00Z", now)).toBe("방금");
    expect(relativeTime("not-a-date", now)).toBe("-");
  });
});

describe("absoluteTime", () => {
  it("잘못된 값은 대시로 물러난다", () => {
    expect(absoluteTime("nope")).toBe("-");
  });
});

describe("csvCell", () => {
  it("쉼표·따옴표·줄바꿈이 있으면 감싸고 이스케이프한다", () => {
    expect(csvCell("a,b")).toBe('"a,b"');
    expect(csvCell('그는 "말했다"')).toBe('"그는 ""말했다"""');
    expect(csvCell("두\n줄")).toBe('"두\n줄"');
  });
  it("평범한 값은 그대로 둔다", () => {
    expect(csvCell("A-01")).toBe("A-01");
    expect(csvCell(42)).toBe("42");
    expect(csvCell(null)).toBe("");
  });
  it("수식으로 해석될 수 있는 값 앞을 막는다", () => {
    // 스프레드시트에서 =HYPERLINK(...) 같은 값이 실행되지 않아야 한다.
    expect(csvCell("=1+1")).toBe("'=1+1");
    expect(csvCell("+82-10")).toBe("'+82-10");
    expect(csvCell("-5")).toBe("'-5");
    expect(csvCell("@user")).toBe("'@user");
  });
});

describe("toCSV", () => {
  it("헤더와 행을 CRLF로 잇고 BOM을 붙인다", () => {
    const csv = toCSV(["이름", "좌석"], [["김개발", "A-01"]]);
    expect(csv.startsWith("﻿")).toBe(true);
    expect(csv).toContain("이름,좌석\r\n김개발,A-01");
  });
});

describe("safeFileName", () => {
  it("파일 이름에 쓸 수 없는 문자를 걷어낸다", () => {
    expect(safeFileName("보고서/2026:최종?.csv")).toBe("보고서-2026-최종-.csv");
  });
});

describe("startOfLocalDay / dayRangeToISO", () => {
  it("사용자 시간대의 그 날 00:00을 가리킨다", () => {
    const at = startOfLocalDay("2026-08-26");
    expect(at).not.toBeNull();
    // 어느 시간대에서 돌려도 로컬 달력으로는 그 날 자정이어야 한다.
    expect(at!.getFullYear()).toBe(2026);
    expect(at!.getMonth()).toBe(7);
    expect(at!.getDate()).toBe(26);
    expect(at!.getHours()).toBe(0);
    expect(at!.getMinutes()).toBe(0);
  });
  it("종료일은 다음 날 자정까지를 뜻해 그 날 전체가 포함된다", () => {
    const { from, to } = dayRangeToISO("2026-08-26", "2026-08-26");
    const start = new Date(from);
    const end = new Date(to);
    expect(end.getTime() - start.getTime()).toBe(24 * 3600 * 1000);
    // 그 날 늦은 시각도 구간 안에 들어와야 한다.
    const evening = new Date(2026, 7, 26, 23, 59, 0);
    expect(evening.getTime()).toBeGreaterThanOrEqual(start.getTime());
    expect(evening.getTime()).toBeLessThan(end.getTime());
  });
  it("빈 값과 잘못된 형식은 빈 문자열로 물러난다", () => {
    expect(dayRangeToISO("", "")).toEqual({ from: "", to: "" });
    expect(startOfLocalDay("2026-8-2")).toBeNull();
    expect(startOfLocalDay("어제")).toBeNull();
  });
});

describe("localDateStamp", () => {
  it("UTC가 아니라 로컬 달력의 날짜를 쓴다", () => {
    // KST 오전 8시는 UTC로는 전날이다. 파일 이름이 사용자가 고른 기간과
    // 하루 어긋나면 안 된다.
    const at = new Date(2026, 7, 26, 8, 0, 0);
    expect(localDateStamp(at)).toBe("2026-08-26");
  });
  it("월과 일을 두 자리로 채운다", () => {
    expect(localDateStamp(new Date(2026, 0, 5))).toBe("2026-01-05");
  });
});
