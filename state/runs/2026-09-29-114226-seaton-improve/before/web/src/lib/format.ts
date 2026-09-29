/**
 * 화면 공통 표시 규칙.
 *
 * 날짜·라벨·내보내기 형식은 화면 없이 검증할 수 있어야 하므로 여기 모아 둔다.
 */

/**
 * 좌석 변경이 어떤 경로로 일어났는지. 원문 값을 그대로 보여주지 않는다.
 * 키는 서버가 seat_history.source 에 실제로 넣는 값과 일치해야 한다.
 * 이력 화면의 "방식" 목록이 이 표에서 만들어지므로, 없는 값을 넣으면
 * 고를 수는 있지만 결과가 없는 항목이 생기고 실제 값은 고를 수 없다.
 */
export const SOURCE_LABELS: Record<string, string> = {
  manual: "수동 배정",
  bulk: "일괄 등록",
  dashboard: "처리필요에서 조치",
  dashboard_bulk: "처리필요 일괄 조치",
  hr_sync: "인사 동기화",
  mcp: "MCP 연동",
};

export const sourceLabel = (source: string) =>
  SOURCE_LABELS[source] || source || "-";

/**
 * 방금 일어난 일은 상대 시간이 읽기 쉽고, 오래된 일은 절대 날짜가 정확하다.
 * 하루가 넘어가면 절대 표기로 바꾼다.
 */
export const relativeTime = (value: string | Date, now: Date = new Date()) => {
  const at = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(at.getTime())) return "-";
  const seconds = Math.round((now.getTime() - at.getTime()) / 1000);
  if (seconds < 0) return "방금";
  if (seconds < 60) return "방금";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}분 전`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}시간 전`;
  if (seconds < 86400 * 7) return `${Math.floor(seconds / 86400)}일 전`;
  return at.toLocaleDateString("ko-KR");
};

export const absoluteTime = (value: string | Date) => {
  const at = value instanceof Date ? value : new Date(value);
  return Number.isNaN(at.getTime()) ? "-" : at.toLocaleString("ko-KR");
};

/**
 * CSV 한 칸을 안전하게 감싼다. 쉼표·따옴표·줄바꿈이 들어와도 열이 밀리지 않고,
 * =, +, -, @ 로 시작하는 값은 스프레드시트에서 수식으로 실행되지 않도록 막는다.
 */
export const csvCell = (value: unknown) => {
  const text = value === null || value === undefined ? "" : String(value);
  const guarded = /^[=+\-@\t\r]/.test(text) ? `'${text}` : text;
  return /[",\n\r]/.test(guarded)
    ? `"${guarded.replace(/"/g, '""')}"`
    : guarded;
};

/** 헤더와 행을 CSV 본문으로 만든다. Excel이 한글을 깨뜨리지 않도록 BOM을 붙인다. */
export const toCSV = (headers: string[], rows: unknown[][]) =>
  "﻿" +
  [headers, ...rows].map((row) => row.map(csvCell).join(",")).join("\r\n");

/** 파일 이름에 쓸 수 없는 문자를 걷어낸다. */
export const safeFileName = (name: string) =>
  name.replace(/[\\/:*?"<>|]/g, "-").slice(0, 120);

/**
 * 날짜 입력(YYYY-MM-DD)을 조회에 쓸 시각으로 바꾼다.
 *
 * 사용자는 자기 시간대의 달력에서 날짜를 고르는데, 서버가 그 문자열을 자기
 * 시간대의 자정으로 해석하면 두 시간대가 다를 때 하루가 어긋난다. 실제로
 * KST 오전에 만든 기록이 "오늘"로 조회되지 않았다. 경계는 시간대를 아는
 * 브라우저가 계산해 시각으로 넘긴다.
 *
 * 시작일은 그 날 00:00:00, 종료일은 다음 날 00:00:00(열린 구간)이다.
 */
export const startOfLocalDay = (date: string): Date | null => {
  const parts = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date.trim());
  if (!parts) return null;
  const [, y, m, d] = parts;
  const at = new Date(Number(y), Number(m) - 1, Number(d), 0, 0, 0, 0);
  return Number.isNaN(at.getTime()) ? null : at;
};

export const dayRangeToISO = (from: string, to: string) => {
  const start = from ? startOfLocalDay(from) : null;
  const endDay = to ? startOfLocalDay(to) : null;
  // 종료일을 포함하려면 다음 날 자정 직전까지여야 하므로 하루를 더한다.
  const end = endDay ? new Date(endDay.getTime()) : null;
  if (end) end.setDate(end.getDate() + 1);
  return {
    from: start ? start.toISOString() : "",
    to: end ? end.toISOString() : "",
  };
};

/**
 * 파일 이름에 쓸 오늘 날짜(YYYY-MM-DD). toISOString 은 UTC 기준이라
 * 사용자가 고른 기간과 하루 어긋난 이름이 붙을 수 있어 로컬 달력을 쓴다.
 */
export const localDateStamp = (now: Date = new Date()) => {
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
};
