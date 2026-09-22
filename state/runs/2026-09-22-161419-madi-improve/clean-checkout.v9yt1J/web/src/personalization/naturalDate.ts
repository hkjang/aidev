export type DateSuggestion = {
  date: string;
  explanation: string;
  timezone: string;
};
const weekdays = ["월", "화", "수", "목", "금", "토", "일"];
function iso(date: Date) {
  return date.toISOString().slice(0, 10);
}
function calendarDate(year: number, month: number, day: number) {
  if (
    year < 1000 ||
    year > 9999 ||
    month < 1 ||
    month > 12 ||
    day < 1 ||
    day > 31
  )
    return null;
  const date = new Date(Date.UTC(year, month - 1, day, 12));
  return date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
    ? date
    : null;
}
// Deterministic calendar suggestions only. Never parses with Date.parse or sends text to AI.
export function suggestDate(
  input: string,
  timezone = "Asia/Seoul",
  now = new Date(),
): DateSuggestion | null {
  const text = input.trim().replace(/\s+/g, "");
  if (!text || text.length > 80) return null;
  let parts: Intl.DateTimeFormatPart[];
  try {
    parts = new Intl.DateTimeFormat("en-CA", {
      timeZone: timezone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).formatToParts(now);
  } catch {
    return null;
  }
  const part = (type: string) =>
    Number(parts.find((p) => p.type === type)?.value);
  const year = part("year"),
    base = calendarDate(year, part("month"), part("day"));
  if (!base) return null;
  const relative: Record<string, number> = {
    오늘: 0,
    내일: 1,
    모레: 2,
    어제: -1,
  };
  let date: Date | null = null,
    explanation = "";
  if (Object.prototype.hasOwnProperty.call(relative, text)) {
    date = new Date(base);
    date.setUTCDate(date.getUTCDate() + relative[text]);
    explanation = `${iso(base)} 기준 ${text}`;
  } else {
    const week = /^(이번주|다음주)?([월화수목금토일])(?:요일)?$/.exec(text);
    const full = /^(\d{4})(?:년|-)(\d{1,2})(?:월|-)(\d{1,2})일?$/.exec(text);
    const short = /^(\d{1,2})월(\d{1,2})일?$/.exec(text);
    if (week) {
      const day = weekdays.indexOf(week[2]),
        today = (base.getUTCDay() + 6) % 7;
      const offset =
        week[1] === "다음주"
          ? 7 - today + day
          : week[1] === "이번주"
            ? day - today
            : (day - today + 7) % 7;
      date = new Date(base);
      date.setUTCDate(date.getUTCDate() + offset);
      explanation = week[1]
        ? `월요일 시작 ${week[1]} ${week[2]}요일`
        : `오늘을 포함한 다음 ${week[2]}요일`;
    } else if (full || short) {
      date = full
        ? calendarDate(+full[1], +full[2], +full[3])
        : calendarDate(year, +short![1], +short![2]);
      explanation = full
        ? "입력한 연도·월·일"
        : `연도를 생략하여 올해 ${year}년으로 제안합니다`;
    }
  }
  if (!date || date.getUTCFullYear() < 1000 || date.getUTCFullYear() > 9999)
    return null;
  if (iso(date) < iso(base)) explanation += " · 오늘보다 이전 날짜입니다";
  return { date: iso(date), explanation, timezone };
}
