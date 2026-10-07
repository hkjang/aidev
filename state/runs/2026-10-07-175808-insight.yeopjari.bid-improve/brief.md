- 과제: 일·월 단위 쓰기 제한의 Retry-After 가 KST 경계를 9시간 넘겨 잡히는 것 고치기 (가치 4 / 위험 2 / 작업량 S)
- 왜: `packages/core/src/quota.ts` 의 `retryAfterSeconds()` 는 "다음 창 시작"을 `windowStart()`(= KST 날짜 키의 **UTC 자정**)으로 계산하는데, 카운터 버킷이 실제로 넘어가는 순간은 KST 자정, 즉 그 UTC 자정보다 **9시간 이른** 시각이다. 그래서 `day`/`month` 제한에 걸린 사람에게 항상 9시간을 더 기다리라고 알려 준다. 고치면 429 응답의 `retry-after` 헤더(`http/respond.ts:68`)와 웹 클라이언트가 읽는 값(`apps/web/src/api/client.ts:181`)이 실제로 풀리는 시각과 맞는다.

- 수용 기준:
  1) `day`: `at = 2026-10-07T16:00:00Z` 일 때 **82800초(23시간)** 를 돌려준다. (현재 코드는 115200초 = 32시간. 손으로 검산: `kstDateKey(at)`=`2026-10-08` → 지금 코드의 "다음 시작"=`2026-10-09T00:00Z`; 실제 버킷 전환은 KST 자정 = `2026-10-08T15:00Z`.)
  2) `day`: `at = 2026-10-07T02:00:00Z` → **46800초(13시간)**. (현재 79200초 = 22시간.)
  3) `month`: `at = 2026-10-31T16:00:00Z` → 다음 KST 달 시작 `2026-11-30T15:00Z` 까지 = **29일 23시간**. (현재 30일 8시간. 12월→1월 넘김도 깨지지 않아야 하므로 `at = 2026-12-31T16:00:00Z` 도 테스트에 넣을 것.)
  4) `minute` 분기의 값은 바뀌지 않는다.
  5) **`windowStart()` 의 반환값은 한 글자도 바뀌지 않는다** — 테스트로 고정해 두면 더 좋다.
  6) 단위 테스트가 위 고정값들을 검증하고 `npm run test` 에서 통과한다.

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `packages/core/src/quota.ts:retryAfterSeconds` — 다음 창 전환 시각을 `util/time.js` 의 `kstDayStart()` 로 계산한다. `day`: `kstDayStart(at).getTime() + days(1)`. `month`: `kstParts(at)` 로 KST 연·월을 얻어 다음 달 1일의 KST 자정(`Date.UTC(y, m, 1) - KST_OFFSET_MINUTES*60000`, 12월이면 연도 +1 / 월 0)으로. 기존 `while` 루프 탐색은 지워도 된다(그 루프의 주석이 말하는 "31일 더하기 overshoot" 문제는 `Date.UTC` 월 산술이 알아서 처리한다).
  - `packages/core/src/quota.ts` — `retryAfterSeconds` 에 `export` 를 붙여 테스트가 직접 부를 수 있게. (DB 없이 검증할 수 있는 유일한 길. `consume()` 경로로 증명하려면 Postgres 가 필요하고 그것은 e2e 의 몫이다.)
  - `packages/core/src/util/time.ts` — 필요하면 `KST_OFFSET_MINUTES`·`kstParts`·`kstDayStart` 를 쓰기만 할 것. 이미 전부 `export` 되어 있으니 **수정할 필요 없다**.
  - `packages/core/src/__tests__/rules.test.ts` — 끝에 `describe('write limits', …)` 추가. 기존 파일의 관례를 따를 것: 한국어 서술형 `it(...)` 문장, `expect(...).toBe(...)` 고정값, 주석 최소.

- 검증 명령:
  - `npm run test` (= `vitest run`, `packages/**/*.test.ts` + `apps/**/*.test.ts`)
  - `npm run lint` · `npm run typecheck` · 전체는 `npm run check`
  - (참고: README 와 `.github/workflows/check.yml` 에 적힌 명령이다. **이 정찰 세션에서는 샌드박스가 node/npm 실행을 거부해 직접 돌려 보지 못했다 — 미확인.** CI 는 `npm run check` → `npm run build` → `node scripts/e2e.mjs` → `node scripts/screenshots.mjs` 순으로 돈다.)

- 위험과 피할 것:
  - **`windowStart()` 를 고치지 말 것.** 그 값은 `insight_priv.quota_counters` 의 유일 키 `(subject, action, window_kind, window_start)` 에 그대로 들어간다. UTC 자정 라벨이 KST 자정과 어긋나 보여도 **창의 길이와 전환 시점은 올바르다**(라벨만 9시간 밀려 있다). 바꾸면 운영 중인 카운터가 새 버킷으로 갈라져 모든 제한이 한 번 리셋된다. 이번 과제는 "언제 풀리는지 알려 주는 숫자" 하나만 고친다.
  - `release()` 와 `usage()` 도 같은 `windowStart()` 를 쓴다 — 그래서 더더욱 건드리면 안 된다.
  - `util/time.ts` 는 읽기만. `kstDateKey`/`kstDayStart` 는 새 연구 알림 묶음 등 다른 곳에서도 쓰인다.
  - 보호 경로 회피: `db/migrations/*`, `.github/workflows/*`, `packages/core/src/auth/*`, `routes/sso.ts`, `packages/core/vendor/*` 는 열지 않아도 끝난다.
  - 마이그레이션 없음, 라우트·응답 스키마 변경 없음. e2e 151개가 Retry-After 값을 단정하고 있는지는 확인하지 않았다(미확인) — `npm run check` 통과 뒤 CI 의 e2e 결과도 볼 것.
  - 겸사겸사 다른 것을 고치지 말 것. 특히 `usage()` 의 "one round trip" 주석(실제로는 `Promise.all` 로 쿼리 n개)은 이번 범위가 아니다.

- 차선 후보: **`packages/core/src/util/text.ts:isSafeLink()` 정리.** 2026-10-07 grep 으로 호출처가 0인 죽은 코드임을 확인했다(실제로 쓰이는 링크 검사는 `domain/eligibility.ts:isWebLink`, `routes/team.ts:278` 의 `parseLinks` 경유). 주석은 "obvious private ranges 차단"을 약속하지만 코드는 `169.254.169.254` 와 `metadata.google.internal` 두 호스트만 막는다. 선택지는 (a) 죽은 함수를 지우고 주석의 약속도 같이 지우기, (b) `isWebLink` 쪽에 사설 범위 차단을 **좁게** 넣고 `parseLinks` 테스트를 늘리기. **두 함수를 하나로 통합하려 하지 말 것** — 운영자 규칙(같은 종류의 파서는 좁히는 방향으로만).
