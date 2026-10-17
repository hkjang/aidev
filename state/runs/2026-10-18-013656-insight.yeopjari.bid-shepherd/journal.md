# PR 처리기 노트 2026-10-18-013656-insight.yeopjari.bid-shepherd — insight.yeopjari.bid PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-18-005904-insight.yeopjari.bid-improve)
# 회차 노트 2026-10-18-005904-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:59] base pinned — main@601659d
- [러너 00:59] autonomy release — 

## 정찰 노트
- 우선 과제가 배정돼 새 아이디어 대신 CI 실패를 봤다. 10-17 회차가 .github·과금·주석을 이미 배제했으므로 남은 창은 739ea33(마지막 녹색)→601659d 뿐이고, 두 PR 이 모두 601659d 의 자식이라 원인이 PR 이 아니라 main 에 있다 — 이 창에서 npm run check 에 새로 들어온 것은 db-breaker.test.ts(신규) 하나다.
- 그 안의 벽시계 단정 4곳을 과제로 골랐다. 특히 :127 의 50ms 는 db.ts:250-252 의 I/O 없는 경로를 재므로 러너 속도만 재고, 바로 위 dropped===tried 가 같은 것을 이미 증명한다(삭제해도 증거 손실 없음).
- 확신 없는 곳: **인과를 확인하지 못했다**. 이 세션에서 gh 와 npm ci 가 둘 다 승인 거부돼 CI 로그도, 로컬 실행도 못 했다. 실패 단계 이름은 미확인이며, 과제서에 분기 A(check 단계)/B(다른 단계)/C 로 나눠 적었다.
- 배제한 것은 직접 확인했다: lock 은 버전 문자열만 변경(npm ci 비동기화 아님), .env.example 도 버전 한 줄, functions/ 는 어느 typecheck 에도 없고 TS 의 no-undef 는 꺼져 있다.
- 구현자가 조심할 것: check.yml 을 한 줄도 바꾸지 말 것(타임아웃 늘리기도 약화로 본다), 차단기 상수(10s/120s/maxAttempts)를 건드리지 말 것, 로컬 main 은 5fffd04 로 낡아 origin/main(601659d) 기준으로 비교할 것.
- [러너 01:04] scout done — CI 가 PR #4·#5 에서 같은 이유로 죽는 자리를 `npm run check` 안에서 확정하고, db-breaker 테스트의 벽시계(wall-clo

## 구현 노트
- 바꾼 것: `packages/core/src/__tests__/db-breaker.test.ts` 한 파일(+47/-8, 프로덕션 0개). 시계 단정 4곳 중 `:127`(50ms)은 삭제하고, 나머지 3곳을 `BUDGET_CEILING_MS = 4 × RETRY_BUDGET_MS(2500)` 한 상수로 모아 "무엇의 상한인가" 를 주석으로 적었다. 같은 의도는 `MOST_CONNECTS(4×3)`·`held.size` 증가·`AppError.cause` 의 `CONNECTION_CLOSED`/`CONNECT_TIMEOUT`·차단기 거절은 `cause` 없음·`retryAfter ≤ 10` 으로 옮겼다.
- 확신 없는 곳: **CI 가 왜 죽는지는 끝내 모른다 — step 이름 미확인**(`gh` 미인증, 비공개 저장소). 과제서가 기댄 인과(이 벽시계가 CI 를 깬다)는 **틀렸다**: 1코어·스피너 8개(9배 과포화)에서도 11개 전부 통과하고 첫 테스트 여유가 19배였다. 따라서 이 변경이 CI 를 녹색으로 만든다고 주장하지 않는다 — 경화일 뿐이다.
- 대신 확인한 것(앞 두 회차가 못 한 것): `npm ci` 가 이 세션에서는 돌아가서 CI 의 모든 step 을 `601659d` 에서 로컬 재현해 전부 통과했다 — check 109, build, migrate 0001~0004, 서버 기동, **e2e 180/0**, **screenshots 85**. PR #5 머지 트리(`cb269c5`, 부모가 `601659d` 라 fast-forward)도 별도 `npm ci` 후 check **121 통과**. 저장소 내용 가설은 닫혔다고 본다.
- 일부러 안 한 것: `db.ts`·`errors.ts`·`check.yml` 무변경(차단기 상수·게이트 손대지 않음). 테스트 전용 변경이라 변경 **후** e2e·screenshots 는 다시 돌리지 않았다. 차선 후보(`noContent()` 204 정리)는 분기 A 가 성립했으므로 건드리지 않았다.
- 다음 역할이 조심할 것: 새 단정의 효력은 변이 검증으로만 증명된다 — `maxAttempts 4→8`, 차단기 `retryAfter` `/1000→/100`, `isConnectionLost` 에 `CONNECT_TIMEOUT` 추가 중 아무거나 넣으면 새 파일은 잡고 **옛 파일은 11개 전부 통과**한다. 이 테스트는 DB 가 필요 없지만 포트 0 에 소켓 서버 2개를 띄우고 모듈 전역 `dropped`·`held` 를 테스트 순서대로 공유한다(`dropped = 0` 초기화 위치를 옮기지 말 것). 검증용 Postgres 컨테이너·서버·`/tmp/pr5` 는 모두 지웠다.
- [러너 01:18] brief accepted — 채택 — 분기 A 의 벽시계 4곳은 과제서가 읽은 그대로였고 지정 파일 1개만 고쳤다. 다만 과제서가 기대한 인과(이 단정�
- [러너 01:18] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 판정 **reject** (단일 사유, 수리가 먼저 볼 파일: `packages/core/src/__tests__/db-breaker.test.ts:149`). 구현자의 변이 검증 주장을 그대로 재현해 봤다 — `maxAttempts 4→8` 은 새 상한 `dropped<=12` 에 24로 걸려 실제로 잡힌다(옛 파일 통과). 상한 12 의 근거도 `PATCHED.md 5`·`connection.js:477`(연속 3드롭→CONNECTION_CLOSED, `errored()` 가 `initial=null` 로 재연결 폭주 차단)에서 확인했다. 여기까지는 주장대로다.
- 다만 구현자가 보고하지 않은 **반대 방향 변이**가 하나 통과한다: `db.ts:345` 의 `healthy()` 를 `retrying()` 우회로 바꾸면 **옛 파일은 실패하고 새 파일은 11개 전부 통과**한다. `dropped===tried` 단정을 `again.healthy()` 뒤에서 앞으로 옮긴 탓이고, `db.ts:340-345` 주석이 명시한 "health 도 차단기를 거친다" 가 무방비가 됐다. 수리는 :153 뒤에 한 줄 추가(옮기기 아님)면 끝.
- 승인이어도 남을 우려: 벽시계 3곳이 전부 `BUDGET_CEILING_MS`(10s) 한 상수로 **느슨해졌다**(5s/4s/3s→10s). SILENT 테스트는 connectTimeout 1초인데 상한 10초다 — 지금은 `CONNECT_TIMEOUT`·`held.size+1` 이 기제를 잡지만 상한을 또 키우면 시간 성분은 남지 않는다.
- 못 본 것: `npm run check`/build/e2e/screenshots 전체는 재실행하지 않았다(db-breaker 11개 1.45s 통과만 확인). CI 실패 원인은 이 세션에서도 미확인 — **이 PR 은 CI 수리가 아니다**(구현 노트가 과제서 인과를 스스로 기각). 릴리즈 노트에 'CI 고침' 으로 적지 말 것. 보안·법무 차단 사유 없음(테스트 전용, 가짜 로컬 자격증명, revert 안전). 로컬 `main` 은 5fffd04 로 낡아 `origin/main`(601659d) 기준으로 비교했다.
- [러너 01:22] review rejected — 리뷰 거절: packages/core/src/__tests__/db-breaker.test.ts:149 회귀 탐지 손실 — `expect(dropped).toBe(tried)` 가 `again.healthy()` 호출 '뒤'(옛 :126)에서 '앞'으로 옮겨져, healt

## 수리 노트
- 지적은 **맞았다**. 직접 재현: 옛 파일(601659d)은 `dropped===tried` 를 `again.healthy()` **뒤**에 뒀는데 새 파일이 앞으로 옮겨, `db.ts:345` 를 `retrying()` 우회로 변이시켜도 11개 전부 통과했다. 틀린 지적은 없었고, 제시된 수리법(`:153` 뒤에 한 줄 추가, 옮기기 아님)도 그대로 유효했다.
- 고친 방법: 테스트 파일 1개 +4줄 — `expect(await again.healthy()).toBe(false)` 뒤에 `expect(dropped).toBe(tried)` 추가(앞의 것 유지)와 "왜 두 번인가" 주석. `db.ts`·`errors.ts`·`check.yml` 무변경.
- 검증: 같은 변이가 이제 `1 failed / 10 passed` 로 잡힌다(`expected 15 to be 12`, 옛 파일과 같은 자리). 변이 후 `db.ts` 복구를 `git diff` 로 확인. `npm run check` 109 통과.
- 일부러 안 한 것: 비평가가 "승인이어도 남을 우려" 로 적은 `BUDGET_CEILING_MS`(5s/4s/3s→10s 완화)는 손대지 않았다 — 거절 사유가 아니었고 범위를 넘는다. e2e·screenshots 는 테스트 전용 변경이라 재실행하지 않았다.
- 여전히 확신 없는 곳: CI 가 왜 죽었는지는 이번에도 모른다(구현 노트가 과제서 인과를 이미 기각). 릴리즈 노트에 'CI 고침' 으로 쓰지 말 것.
- [러너 01:24] repair done — # 수리 요약 (수리 시도 2)  - 문제: `db-breaker.test.ts` 에서 `expect(dropped).toBe(tried)` 가 `again.healthy()` 뒤에서 앞으로 옮겨져, "health 도 차단기를 거쳐 연결 없�

## 비평 노트
- 판정 **approve**. 변경은 `packages/core/src/__tests__/db-breaker.test.ts` 1개(+50/-7, 프로덕션 0), `npm run check` 109 통과(eslint+typecheck 3+테스트), db-breaker 5회 반복 11/11 안정.
- 변이로 실제 검증했다(옛 파일=origin/main 과 비교): `maxAttempts 4→8` 과 차단기 `retryAfter /1000→/100` 은 **새 파일만** 잡고(옛 파일 11개 통과), 수리가 넣은 `healthy()` 우회 변이도 새 파일이 잡는다(`expected 15 to be 12`). `connect_timeout` 옵션 무시(15s)·`isConnectionLost` 에 `CONNECT_TIMEOUT` 추가는 양쪽 다 잡는다. 순 탐지력은 더 강하다.
- 승인이어도 남을 우려 2개: (1) SILENT 테스트 상한 3s→10s 완화로 **3~10초 구간의 회귀는 실제로 놓친다** — `connect_timeout: 8`(고정) 변이는 옛 파일만 잡았다. 운영은 `functions/[[path]].ts:70` 에서 `connectTimeout: 5` 이므로 이 구간은 비어 있지 않다. 다음 회차에 SILENT 만 connectTimeout 배수(예 `< 2_000`)로 되돌릴 것. (2) 테스트2 의 `dropped - before > 3` 는 러너가 한 시도에 2.5초를 넘기면 3 이 나와 깨진다 — 탈플레이크 목적에 역행하는 작은 위험(옛 파일 테스트1 에도 같은 단정이 있었음).
- 못 본 것: e2e·screenshots·build 는 재실행하지 않았다(테스트 전용 변경). CI 실패 원인은 이 세션에서도 미확인 — **릴리즈 노트에 'CI 고침' 으로 적지 말 것**. `RETRY_BUDGET_MS 2500→25000` 변이는 옛·새 둘 다 못 잡는다(기존 공백, 이번 회귀 아님).
- 보안·법무 차단 사유 없음: 테스트 전용, 자격증명은 루프백 가짜값, 신규 의존성·개인정보·인증/암호 경로 무변경, revert 안전.
- [러너 01:29] review approved — 리뷰 승인 (risk=low)
- [러너 01:29] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/6
- [러너 01:30] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 지적은 **틀렸다** — CI `check` 실패는 이 PR 의 결함이 아니다. 잡이 3초에 죽고 실패 step 이름이 없는데(`실패한 검사: ?`, 로그 비어 있음), `npm ci` 만 12초다. 서비스 컨테이너 초기화·checkout·setup-node 를 더하면 3초 안에는 `npm run check` 를 포함한 **어느 step 도 시작조차 못 한다** — 저장소 내용이 닿는 자리가 아니다. 그래서 고치지 않고 커밋 없이 끝냈다.
- 확인한 것: HEAD(2f37e63)에서 `npm ci` 0 · `npm run check` 0(109 테스트) · `npm run build` 0. `merge-base origin/main HEAD == origin/main` 이라 `refs/pull/6/merge` 는 HEAD 와 같은 트리 — CI 가 보는 것을 돌렸다. `.github/` 는 마지막 녹색(739ea33) 이후 0줄 변경.
- 앞 비평의 유일한 내용 가설도 기각: 테스트2 `dropped - before > 3` 는 2코어 고정 + 스피너 6개(4배 과포화) 5회에서 11/11 통과(본문 1.26~1.39초, 상한 10초). 느린 러너 플레이크로 설명되지 않는다.
- 남은 설명은 저장소 밖(`Set up job`: Actions 과금·지출 한도, 이미지 pull 제한, 러너 할당). 서로 다른 세 diff(#4·#5·#6)와 main@601659d 가 똑같이 죽는 모양이 이를 가리킨다.
- 확신 없는 곳: 어느 저장소 밖 원인인지는 **끝내 못 좁혔다** — 이 세션도 `gh` 미인증·비공개(API 404)로 로그를 못 봤다. 사람이 잡 117445410932 의 "Set up job" 출력을 봐야 한다. 릴리즈 노트에 'CI 고침' 으로 적지 말 것.
