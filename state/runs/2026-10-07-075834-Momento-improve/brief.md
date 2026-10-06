# 과제서 — 2026-10-07 (Momento, base main@1ef81d6 / v0.34.58)

- 과제: 보존 정책 저장 실패를 서버의 영문 문장·pgx 원문 대신 한국어로 안내한다 (가치 2 / 위험 1 / 작업량 S)

- 왜: `RetentionAdmin` 의 Alert 이 `{save.error.message}`(web/src/pages/AdminPage.tsx:2957) 로 서버 문장을 그대로 띄운다 — 다섯 칸이 전부 `type="number"` 자유 입력이라 범위를 벗어난 값이 흔한데, 그때 올라오는 것은 `raw_event_months must be between 1 and 120`(internal/httpapi/advanced_analytics.go:82) 처럼 **화면에 없는 영문 컬럼명**이고, 500 으로 떨어지면 `err.Error()`(advanced_analytics.go:117)가 그대로 실려 pgx 원문이 브라우저까지 흐른다. v0.34.54~57 이 네 번 통과시킨 패턴(`adminErrors.ts` 에 `describe*` 를 하나 더 두고 Alert 하나만 바꾸기)을 그대로 적용하면, 읽는 사람이 `raw_event_months` 가 「Raw Event (개월)」 칸이라는 것을 스스로 번역하지 않아도 된다.

- 수용 기준:
  1) 다섯 가지 `INVALID_RETENTION` 거절이 **각각 자기 칸의 화면 라벨을 이름으로 가리키고** 허용 범위를 한국어로 말한다 — `raw_event_months`→「Raw Event (개월)」(1~120개월), `session_months`→「Session 요약 (개월)」(1~120개월), `aggregation_months`→「Aggregation (개월)」(비워 두거나 1~1200개월), `realtime_hours`→「Realtime (시간)」(1~168시간), `debug_days`→「Debugger / Dead Letter (일)」(1~90일). Alert 본문에 영문 컬럼명(`raw_event_months` 등)·`must be between` 이 남지 않는다.
  2) `RETENTION_SAVE_FAILED`(500) Alert **본문**에 Postgres 표지(`SQLSTATE`·`violates`·`relation`·`no rows in result set`)가 하나도 없고, 서버 원문은 지우지 않고 `detail` caption 으로만 남는다. `UNKNOWN_SITE`(404)·`INVALID_PAYLOAD`(400)·`REQUEST_FAILED` 도 한국어이고, 모르는 코드는 서버 메시지 그대로(`default` 되돌림).
  3) 테스트가 증명할 것: **지금 Alert 과 똑같이 동작하는 항등 스텁**(`error.message` 를 그대로 돌려주는 구현)을 끼우면 새 단언이 실패한다 — 즉 테스트가 "영문/원문이 본문에 샜다" 를 실제로 짚는다. 기존 33개 테스트는 스텁에서도 전부 통과해야 한다(새 테스트가 보존 정책 결함만 짚는다는 증거).

- 건드릴 파일 (프로덕션 2 + 테스트 1):
  - `web/src/pages/adminErrors.ts` — `describeRetentionError(error: unknown): UserErrorNotice` 를 **새로** export. 기존 `refusal`(66행)과 문장 상수 `UNREADABLE_PAYLOAD`·`LOST_REQUEST`·`PASS_TO_ADMIN`(79-84행)만 재사용하고, 네 개 `describe*`(86 user / 198 site / 273 network / 334 eventDefinition)와 **합치지 말 것** — 지난 네 회차의 판단이고 핸들러마다 코드 집합이 다르다. `INVALID_RETENTION` 은 코드 하나에 원인이 다섯이므로 `describeUserError` 의 `PASSWORD_PROBLEM`(37행) 선례대로 **메시지를 읽어 가른다**: 다섯 컬럼명(`raw_event_months`·`session_months`·`aggregation_months`·`realtime_hours`·`debug_days`)은 서로의 부분문자열이 아니라 단순 포함 검사로 안전하게 갈린다(`raw_event_months`·`session_months`·`aggregation_months` 는 `_months` 를 공유하지만 어느 것도 다른 것의 부분문자열이 아니다 — 확인했다). 다섯 개 중 어느 것도 안 맞으면(서버가 검사를 더 늘린 경우) 영문을 본문에 올리지 말고 중립 문구 + `detail` 로 되돌릴 것.
  - `web/src/pages/AdminPage.tsx` — (a) import 블록(79-84행)에 `describeRetentionError` 추가, (b) 네 개 `*ErrorAlert` 옆(3359-3370행 사이)에 `RetentionErrorAlert({error})` = `<AdminNoticeAlert notice={describeRetentionError(error)} />` 를 같은 모양으로 추가, (c) **2957행 한 줄만** `{save.error && <RetentionErrorAlert error={save.error} />}` 로 교체. 2413(설정 저장)·2510(rekey)·3112(dimensions)·1372(설치 코드)·2761(삭제) 의 Alert 은 **이번에 손대지 말 것**.
  - `web/test/adminErrors.test.mjs` — 맨 위 import(3-8행)에 `describeRetentionError` 추가하고 파일 끝에 보존 정책 단언을 덧붙인다. `apiError(status, code, message)` 헬퍼(22-27행)를 그대로 쓸 것 — `APIError` 를 import 하지 않고 shape 만 만드는 것이 이 파일의 관례다. 서버 문장은 advanced_analytics.go:82·85·88·91·94 에서 **글자 그대로** 옮길 것.

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (worktree 에 `web/node_modules` 가 없어 `npm ci` 선행, 수 분). 현재 테스트 수는 **미확인**(이 세션은 `npm ci` 를 돌리지 않았다) — `web/test/*.test.mjs` 39개 파일, `adminErrors.test.mjs` 에 `test(` 33개.
  - 서버는 손대지 않지만 인용한 행을 확인했다면 `go vet ./internal/httpapi/` 한 번.
  - **프로덕션 배선까지 증명할 것** — 지난 네 회차가 모두 이것으로 통과했다. `npm run build` 의 dist 를 `/api/v1/me`·`/api/v1/sites`·`/api/v1/sites/{id}/retention` 을 흉내 낸 임시 서버에 올리고 headless Chrome(`google-chrome` 설치돼 있음, `puppeteer-core` 는 없어 `/tmp` 에 따로 설치)으로 `/admin?section=retention` 의 칸을 실제로 채우고 「보존 정책 저장」 을 눌러 Alert DOM 을 읽는다. 에러 객체는 손으로 만든 대역이 아니라 `api()` 가 HTTP 응답에서 만든 진짜 `APIError` 여야 한다. 고치기 전 main dist 로 같은 하네스를 돌려 **실패를 먼저 보고** 시작할 것. 임시 하네스는 `/tmp` 에만 두고 `web/dist` 는 커밋 전에 지워 `git status` 로 3파일만 담긴 것을 확인할 것.

- 위험과 피할 것:
  - **서버(`internal/httpapi/advanced_analytics.go`)를 손대지 말 것.** 영문 문장을 바꾸면 통합 테스트와 이 과제의 메시지 매칭이 동시에 흔들린다. `validateRetention` 의 범위도 그대로 둔다.
  - **`RETENTION_SAVE_FAILED` 의 원인을 특정하지 말 것 — 문구는 중립으로.** `retention_policies` 의 INSERT 는 `ON CONFLICT(site_id) DO UPDATE`(advanced_analytics.go:115) 라 '이미 등록된 정책' 은 거짓이고, 같은 500 을 DB 장애도 쓴다. 이 환경에 Postgres 가 없고 `MOMENTO_TEST_POSTGRES_DSN` 없으면 통합 테스트가 **조용히 skip** 되므로 DB 로 재현할 수 없다 — 재현하지 않은 원인을 문구로 단정하는 것이 지난 회차들이 피해 온 자리다.
  - **화면 쪽에 입력 차단(`disabled`)을 새로 넣지 말 것.** 이번 과제는 '실패를 읽히게 한다' 하나다. CIDR 회차처럼 helperText 검증까지 하려면 별 회차로 쪼갤 것(아래 보류 아이디어에 남아 있다).
  - 순수 `.ts` 모듈끼리의 **값** import 는 `.ts` 확장자가 필요하다(adminErrors.ts:26-27 이 그래서 `./passwordRule.ts`·`./cidrRule.ts`). 새 상수를 다른 모듈로 빼게 되면 확장자를 빼지 말 것.
  - 기존 테스트가 **문장을 글자 그대로 단언**한다 — 공용 상수 `PASS_TO_ADMIN`·`UNREADABLE_PAYLOAD`·`LOST_REQUEST` 의 글자를 바꾸면 기존 33건이 깨진다. 재사용만 하고 수정하지 말 것.
  - `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 의존성·CI 단계가 없고 main 의 20여 파일이 이미 실패한다).
  - 보호 경로(internal/auth, internal/database/migrations, .github/workflows, docs/openapi.yaml)는 건드리지 않는다 — 이 과제는 거기 닿지 않는다.
  - **미확인으로 남긴 것**: (1) `UNKNOWN_SITE`(404)가 화면에서 실제로 도달 가능한지 — `RetentionAdmin` 은 `useSite()` 의 사이트로만 요청하므로 사이트가 지워진 직후에만 난다. 도달성은 확인하지 않았으니 코드만 받아 두고 문구는 '사이트를 찾을 수 없다 + 새로 고침' 으로 둘 것. (2) `aggregation_months` 를 빈 칸으로 두면 `null` 이 가고 `validateRetention` 이 통과시키는 것은 **코드로만** 확인했고 브라우저로는 확인하지 않았다. (3) 현재 web 테스트 총 개수.

- 차선 후보: 「JSON Schema」 칸에 입력 중 JSON 구문 helperText 검증을 붙인다 (가치 2 / 위험 1 / S) — `describeEventDefinitionError`(adminErrors.ts:334~)가 이미 한국어 문장을 갖고 있으므로 판정만 순수 모듈로 빼면 둘이 같은 문장을 쓴다(CIDR 회차가 `CIDR_RULE` 로 한 것과 같은 구조). 다만 다섯 줄 자유 입력이라 onBlur/디바운스 설계가 작업의 본체다.
