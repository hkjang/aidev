## 2026-10-07
- 선택: 일·월 단위 쓰기 제한의 Retry-After 가 KST 경계를 9시간 넘겨 잡히는 것 고치기 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `quota.ts:retryAfterSeconds` 가 다음 창 전환 시각을 `windowStart()`(KST 날짜의 **UTC** 자정 라벨)로 계산해 `day`·`month` 제한에 걸린 사람에게 항상 9시간을 더 기다리라고 알려 주고 있었다. 전환 시각을 `kstDayStart(at)+1일` / `Date.UTC(kstY, kstM, 1) - 9h` 로 바로 계산하도록 바꿨고(`windowStart()` 는 DB 유일 키라 본문 한 줄도 건드리지 않았다), 테스트에서 직접 부를 수 있게 `retryAfterSeconds`·`windowStart` 에 `export` 를 붙였다. `rules.test.ts` 에 `describe('write limits')` 4개를 추가해 day 2건·month 2건(10→11월, 12→1월) 고정값, `minute` 분기 불변, `windowStart()` 의 ISO 문자열 4개를 못 박았다. `npm run check`(lint+typecheck+vitest 44개) 와 `npm run build` 통과. e2e 는 실 Postgres 가 필요해 돌리지 못했지만 grep 으로 `retry-after` 값을 단정하는 e2e 가 없음을 확인했다.
- 실패 재현: `AssertionError: expected 115200 to be 82800` (day, `2026-10-07T16:00:00Z`) / `AssertionError: expected 2620800 to be 2588400` (month, `2026-10-31T16:00:00Z`) — 같은 실행에서 `minute` 과 `windowStart` 테스트 2개는 이미 통과(회귀 가드).
- 보류 아이디어: util/text.ts:isSafeLink() 가 호출처 0인 죽은 코드이고 주석의 사설망 차단 약속을 지키지 않는다 / util/time.ts 에 단위 테스트가 전혀 없다(kstWindowOn·parseClock 47시간 클램프·backoffMs 의 "캡 전에 지터") / domain/catalog.ts 와 maskEmail·mailboxKey 의 엣지케이스 테스트 공백 / quota.ts:usage() 의 "one round trip" 주석이 실제(Promise.all 로 쿼리 n개)와 어긋난다 / rules.test.ts 한 파일에 단위 테스트가 전부 모여 있다(테스트가 세 배로 늘기 전에는 손대지 말 것).
- 과제서: 채택 — 과제서의 근거·수용 기준·손으로 한 검산이 지금 코드와 전부 일치했고 수정 범위도 프로덕션 파일 1개로 끝났다.

- 릴리즈: v0.1.1 (2026-10-07, run 2026-10-07-175808-insight.yeopjari.bid-improve)
