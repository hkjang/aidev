import assert from "node:assert/strict";
import { suggestDate } from "../web/src/personalization/naturalDate.ts";
const now = new Date("2026-09-08T16:00:00Z");
assert.equal(suggestDate("오늘", "Asia/Seoul", now).date, "2026-09-09");
assert.equal(suggestDate("오늘", "UTC", now).date, "2026-09-08");
assert.equal(suggestDate("내일", "Asia/Seoul", now).date, "2026-09-10");
assert.equal(
  suggestDate("다음주 월요일", "Asia/Seoul", now).date,
  "2026-09-14",
);
assert.equal(
  suggestDate("이번주 월요일", "Asia/Seoul", now).date,
  "2026-09-07",
);
assert.match(
  suggestDate("9월 1일", "Asia/Seoul", now).explanation,
  /이전 날짜/,
);
assert.equal(suggestDate("9월 20일", "UTC", now).date, "2026-09-20");
assert.equal(suggestDate("2028년 2월 29일", "UTC", now).date, "2028-02-29");
for (const input of [
  "2026-02-29",
  "2026-13-01",
  "내일 오전 9시",
  "언젠가",
  "",
  "<script>",
  "2026-00-01",
])
  assert.equal(suggestDate(input, "UTC", now), null, input);
assert.equal(suggestDate("오늘", "invalid-timezone", now), null);
assert.equal(
  suggestDate("내일", "America/New_York", new Date("2026-03-08T06:30:00Z"))
    .date,
  "2026-03-09",
);
assert.equal(
  suggestDate("다음주 월요일", "UTC", new Date("2026-09-07T12:00:00Z")).date,
  "2026-09-14",
);
console.log(
  "PASS Korean date suggestions: timezone, DST-safe calendar, year/week boundaries, ambiguity rejection; no automatic save.",
);
