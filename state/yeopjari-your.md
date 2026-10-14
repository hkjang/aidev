## 2026-10-06
- 선택: 브라우저 오류 알림의 중복 묶음 서명에 오류 종류·내용을 넣는다 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `client-errors.ts` 가 모든 브라우저 보고를 평범한 `new Error` 로 넘겨 `pipeline.ts` 의 `code` 가 늘 `'internal'` 이 되고, 서명이 `CLIENT <경로> internal` 하나로 뭉쳐 한 화면의 첫 오류가 창(window) 동안 그 화면의 다른 모든 오류 메일을 막았다. 서명 조립을 순수 함수 `alertSignature(method, path, code, extra='')` 로 꺼내고(extra 가 빈 문자열이면 예전 문자열과 바이트 동일 — 공백도 안 붙음), `alertOperator` 에 기본값 `''` 인 마지막 인자를 더하고, client-errors 가 스크럽된 메시지와 `kind` 로 만든 고정 8자 hex 지문(FNV-1a, `clientErrorFingerprint`)을 그 인자로 넘긴다. 서명 조립은 한 곳으로 모으고 폴딩된 경로는 `folded` 변수 하나로 묶어 DB `path` 칼럼·메일이 서명과 어긋날 수 없게 했다(지문은 `path` 칼럼에 섞지 않음 — 실제 DB 행으로 확인). 검증: `npm run check`(lint+타입+단위 39개 통과), `npm run build`, 그리고 Docker PostgreSQL 16 위 `scripts/e2e.mjs` 전체 — 고치기 전 159 passed / 1 failed(새 검사만 실패), 고친 뒤 **160 passed / 0 failed**.
- 실패 재현: `FAIL  같은 화면의 서로 다른 브라우저 오류는 서로 다른 묶음으로 (종류·내용이 서명에) :: [{"signature":"CLIENT /today internal","occurrences":4,"last_message":"[브라우저 render] assignment to constant variable"}]` — `/today` 로 보낸 메시지·kind 가 서로 다른 4건이 전부 한 행으로 뭉쳤다(고친 뒤 같은 입력이 서명 3행 + 중복 1건은 occurrences=2). 단위 테스트 쪽은 새 export 가 없어 `TypeError: clientErrorFingerprint is not a function` 로 6건 실패했다 — 결함을 증명한 것은 위의 e2e(실제 배선) 쪽이다.
- 보류 아이디어: util/validate.ts·util/text.ts 단위 테스트 공백 메우기(차선 후보, 3/1/S) / util/text.ts 의 isSafeLink·tidyBody·excerpt 는 호출부 0개인 죽은 코드이고 주석이 과장된 보호를 주장한다(2/1/S) / v.instant 가 '2026'·'Dec 25' 같은 비 ISO-8601 문자열도 통과시킨다(2/1/S) / launch.ts 의 waitingFor 가 published_at IS NULL 인 회원에게 전원 '1번'을 줄 수 있다(미확인 가설, 3/3/M) / client-errors 의 IGNORED 가 'Failed to fetch'·'NetworkError' 를 전부 버려 API 전면 장애가 조용해진다(2/2/S) / 새로: e2e.mjs 가 가짜 서버 포트 18998·18997 을 하드코딩해 같은 기계의 다른 세션과 충돌한다(2/1/S).
- 과제서: 채택 — 근거(code 가 늘 'internal', 서명 PK upsert 가 창 안에서 send=false)가 지금 코드와 정확히 맞고 e2e 로 재현됐다.

- 릴리즈: 0.1.1 (2026-10-06, run 2026-10-06-225811-yeopjari-your-improve)
## 2026-10-14
- 선택: 프로필의 비워 둘 수 있는 항목이 실제로 비워지지 않는다 — `v.optional` 이 빈 문자열을 "안 보낸 것"으로 삼킨다 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `v.optional` 이 `''` 를 fallback(undefined)으로 끊고 `parse` 가 그 키를 빼면, `PUT /api/v1/profile` 의 부분 갱신이 현재 행을 그대로 두어 회원이 지운 항목이 성공 응답과 함께 예전 값으로 되돌아왔다. `''` 를 안쪽 검증기에 그대로 넘기는 `v.clearable` 래퍼를 더하고(`util/validate.ts`) 흡연·음주·종교·MBTI·하는 일·소개 여섯 개만 옮겼다(`routes/profile.ts:209-215`) — `v.optional` 은 호출부 30여 곳이 `''` 를 "안 보낸 것"으로 쓰므로 손대지 않았고 `region`·`job_family`·`height_cm` 도 그대로 뒀다. 검증: `npm run check`(lint+타입+단위 50개 통과), `npm run build`, 그리고 Docker PostgreSQL 16 위 실제 배선 e2e — 고치기 전 **161 passed / 1 failed**(새 검사만), 고친 뒤 **163 passed / 0 failed**(검사 셋 다 추가한 상태).
- 실패 재현: `FAIL  비워 둘 수 있는 항목은 빈 문자열로 보내면 실제로 지워진다 :: 채움=no 지움="no" db="no"` — `{ smoking: '' }` 를 보내도 응답 본문과 `your.profiles.smoking` 둘 다 예전 값 `no` 로 남았다. 같은 회차의 단위 검사는 고치기 전 `TypeError: v.clearable is not a function` 으로 에러가 났을 뿐이라(새 export 가 없어서) 결함을 증명한 것은 위의 e2e 쪽이다.
- 보류 아이디어: util/validate.ts·util/text.ts 단위 테스트 공백 메우기(3/1/S, `v.clearable` 만 덮었고 `v.int('0x10')`·`v.string` 의 UTF-16 길이 등은 그대로 공백) / 지울 수 있는 필드 목록을 한곳에 모아 같은 결함의 재발을 막기(2/1/S) / `scripts/e2e.mjs` 의 가짜 서버 포트 18998·18997 하드코딩을 환경변수로(2/1/S — 이번 회차에도 다른 프로젝트의 고아 프로세스가 18998 을 쥐고 있어 실제로 막혔다) / `v.instant` 가 ISO-8601 이 아닌 `'2026'`·`'Dec 25'` 도 통과시킨다(2/1/S) / `util/text.ts` 의 `isSafeLink`·`tidyBody`·`excerpt` 는 호출부 0개인 죽은 코드이고 주석이 과장된 보호를 주장한다(2/1/S)
- 과제서: 채택 — 지목한 줄(`validate.ts:225`, `parse:29`, `profile.ts:252-255`)이 지금 코드와 정확히 맞고 e2e 로 재현·수정·재검증까지 됐다. 정찰이 "추측으로 적은 것"으로 남긴 수용 기준 3(공개 중 프로필에서 `intro` 비우기 거부)도 `checklist()` 의 `intro ≥ 20자` 요구를 확인하고 e2e 검사로 못 박았다.

- 릴리즈: 0.1.4 (2026-10-14, run 2026-10-14-090824-yeopjari-your-improve)
