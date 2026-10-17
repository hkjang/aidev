## 2026-10-07
- 선택: 일·월 단위 쓰기 제한의 Retry-After 가 KST 경계를 9시간 넘겨 잡히는 것 고치기 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `quota.ts:retryAfterSeconds` 가 다음 창 전환 시각을 `windowStart()`(KST 날짜의 **UTC** 자정 라벨)로 계산해 `day`·`month` 제한에 걸린 사람에게 항상 9시간을 더 기다리라고 알려 주고 있었다. 전환 시각을 `kstDayStart(at)+1일` / `Date.UTC(kstY, kstM, 1) - 9h` 로 바로 계산하도록 바꿨고(`windowStart()` 는 DB 유일 키라 본문 한 줄도 건드리지 않았다), 테스트에서 직접 부를 수 있게 `retryAfterSeconds`·`windowStart` 에 `export` 를 붙였다. `rules.test.ts` 에 `describe('write limits')` 4개를 추가해 day 2건·month 2건(10→11월, 12→1월) 고정값, `minute` 분기 불변, `windowStart()` 의 ISO 문자열 4개를 못 박았다. `npm run check`(lint+typecheck+vitest 44개) 와 `npm run build` 통과. e2e 는 실 Postgres 가 필요해 돌리지 못했지만 grep 으로 `retry-after` 값을 단정하는 e2e 가 없음을 확인했다.
- 실패 재현: `AssertionError: expected 115200 to be 82800` (day, `2026-10-07T16:00:00Z`) / `AssertionError: expected 2620800 to be 2588400` (month, `2026-10-31T16:00:00Z`) — 같은 실행에서 `minute` 과 `windowStart` 테스트 2개는 이미 통과(회귀 가드).
- 보류 아이디어: util/text.ts:isSafeLink() 가 호출처 0인 죽은 코드이고 주석의 사설망 차단 약속을 지키지 않는다 / util/time.ts 에 단위 테스트가 전혀 없다(kstWindowOn·parseClock 47시간 클램프·backoffMs 의 "캡 전에 지터") / domain/catalog.ts 와 maskEmail·mailboxKey 의 엣지케이스 테스트 공백 / quota.ts:usage() 의 "one round trip" 주석이 실제(Promise.all 로 쿼리 n개)와 어긋난다 / rules.test.ts 한 파일에 단위 테스트가 전부 모여 있다(테스트가 세 배로 늘기 전에는 손대지 말 것).
- 과제서: 채택 — 과제서의 근거·수용 기준·손으로 한 검산이 지금 코드와 전부 일치했고 수정 범위도 프로덕션 파일 1개로 끝났다.

- 릴리즈: v0.1.1 (2026-10-07, run 2026-10-07-175808-insight.yeopjari.bid-improve)
## 2026-10-08
- 선택: 과제 링크가 500자를 넘으면 잘라 저장하지 말고 입력 오류로 거절 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: parseLinks의 검증 전 500자 절단이 긴 URL을 다른 주소로 저장하는 원인이었다. 기존 text 정규화를 절단 없이 적용하고 500자 초과이면 원래 인덱스의 validation_failed 오류를 수집하도록 프로덕션 1개 파일만 수정했으며, 오류에는 URL 원문을 넣지 않았다. 단위 경계·정규화 회귀 및 실제 PUT→GET e2e로 500자 완전 보존, 501자 HTTP 400/field_errors, 실패 시 제목·링크와 연구 전체 조회 상태 보존을 검증했다.
- 실패 재현: `AssertionError: expected function to throw an error, but it didn't` (501/700자); `Tests  3 failed | 29 passed (32)` — 수정 전 실제 공개 parseLinks 테스트, 수정만 되돌려도 동일 실패 재확인.
- 보류 아이디어: util/text.ts:isSafeLink() 죽은 코드·주석 정리 (가치 3 / 위험 2 / S)
- 보류 아이디어: maskEmail·mailboxKey 경계 테스트 (가치 2 / 위험 1 / S; catalog와 별도 범위)
- 보류 아이디어: quota usage one round trip 주석 확인 (가치 2 / 위험 2 / S; 측정 없는 성능 변경 금지)
- 보류 아이디어: judgeSubmission URL 답변 절단 방지 (가치 3 / 위험 2 / S; 이번 파서와 통합하지 않고 독립 재현부터)
- 과제서: 채택 — 기준 코드에서도 절단 결함이 재현되었고 지정 3개 파일과 수용 기준 그대로 구현했다.
- 검증: `npm ci --no-audit --fund=false` 성공(기존 eslint deprecated 경고, 의존성 변경 없음); `npm run test -- packages/core/src/__tests__/rules.test.ts` 수정 전 3 실패→수정 후 32 통과; 수정 되돌림 3 실패→복구 후 `npm run check` 린트·3 workspace 타입 검사·5 파일/63 테스트 통과; `npm run build` 성공; e2e 테스트 비교 수정 후 최종 `npm run check`도 63 통과.
- 실서버 검증: `BASE=http://localhost:18789 DATABASE_URL=postgres://postgres@127.0.0.1:55482/insight APP_DATABASE_URL=postgres://insight_app@127.0.0.1:55482/insight node scripts/e2e.mjs` → `176 passed, 0 failed`. 별도 폐기용 Postgres 16 컨테이너, 4개 정식 migration, 정식 빌드 서버/irumx_gateway→insight_app 역할 전환, 기존 가짜 OAuth :18999를 사용했다. .env.example을 프로세스 메모리로 읽어 포트만 분리했으며 .env 파일 작성 없이 실행했다. 기존 개발 DB/서버는 변경하지 않았다.
- 시행착오: 테스트 DB 준비 중 잘못 추정한 insight_gateway 역할명이 없어 SQL 실패; 소스의 실제 irumx_gateway를 확인해 폐기용 DB에서 바로잡고 migration을 재실행했다. 첫 e2e는 175 통과/1 실패: 직접 저장→조회로 URL 길이 500·원문 일치·라벨 일치를 확인했고, JSONB 객체 키 순서(url,label)와 입력 순서(label,url)가 원인이었다. 문자열 비교를 isDeepStrictEqual로 고쳐 값·구조 전체 비교를 유지하고 176개 모두 재실행해 통과했다.
- 범위/정리: 변경 3개(프로덕션 1개), lockfile 무변경, 빌드 산출물은 기존 .gitignore로 제외. 검증용 서버·컨테이너 종료, e2e 기존 운영자 지정 테스트가 만든 임시 credential fixture 2개 삭제. `git diff --check` 통과 및 `git status --short`로 지정 파일만 확인. UI 스크린샷·운영 배포 검증은 범위 밖이라 미실행.
- 스킬: Skill 호출 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/ 아래 completion-verification, systematic-debugging, test-driven-development의 SKILL.md를 직접 읽고 실패→최소 수정→성공→수정 되돌림 실패→최종 검증 절차를 적용했다.
- 커밋: `fde293d` — 과제 링크: 500자 초과 주소를 자르지 않고 입력 오류로 거절 (작성자 hkjang, 트레일러 없음). 커밋 뒤 git diff --check 통과, git status --short 출력 없음.

- 릴리즈: v0.1.4 (2026-10-08, run 2026-10-08-133829-insight.yeopjari.bid-improve)
## 2026-10-08
- 선택: 제출 답변 URL이 500자를 넘으면 자르지 말고 입력 오류로 거절하기 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: judgeSubmission이 URL을 500자로 절단한 뒤 검사해 다른 주소로 성공 저장하는 원인을 확인하고, URL만 text(value, Infinity)로 정규화한 뒤 빈 값·500자 초과·http(s) 순으로 검증하도록 수정했다. 경계·정규화·UTF-16 길이·필수/선택·여러 질문 오류 수집·text(500)/long(4000) 유지 단위 테스트를 추가하고, 실제 제출 API와 참여자/기업 GET으로 500자 완전 보존 및 501자 거절 시 전체 답변/제출 시각 보존을 확인했다. 오류에는 URL 원문을 넣지 않으며 parseLinks·공통 text/isWebLink·라우트·UI는 바꾸지 않았다.
- 실패 재현: `AssertionError: expected function to throw an error, but it didn't` / `Tests  7 failed | 60 passed (67)` — 수정 전 공개 parseForm→judgeSubmission 실행. 수정만 되돌려 동일 7실패를 재확인했고 복구 후 67개 통과.
- 보류 아이디어: isSafeLink 죽은 코드·사설망 차단 주석 축소 (가치 3 / 위험 2 / S)
- 보류 아이디어: maskEmail·mailboxKey 현재 계약 경계 테스트, catalog와 분리 (가치 2 / 위험 1 / S)
- 보류 아이디어: quota usage one round trip 주석을 실제 측정과 비교 (가치 2 / 위험 2 / S)
- 보류 아이디어: README 로컬 설치 안내를 npm ci로 맞추기 (가치 2 / 위험 1 / S)
- 보류 아이디어: API client 204 쓰기의 캐시 무효화 누락, 실제 소비 경로 확인 우선 (가치 2 / 위험 1 / S)
- 보류 아이디어: README·architecture의 낡은 테스트 개수 안내 정리 (가치 2 / 위험 1 / S)
- 보류 아이디어: 월 quota purge의 실제 DB 회귀 확인, 운영 사용 여부부터 확인 (가치 2 / 위험 3 / M)
- 과제서: 채택 — 현재 코드에서도 검증 전 URL 절단이 재현되어 지정한 프로덕션 1개와 테스트 2개 파일 범위 그대로 구현했다.
- 검증: `npm ci --no-audit --fund=false` 성공(기존 eslint deprecated 경고, lockfile 무변경); 기준 `npm run test -- packages/core/src/__tests__/rules.test.ts` 32통과→추가 후 7실패/60통과→수정 후 67통과→수정 되돌림 7실패→복구 후 67통과. 최종 `npm run check`는 lint·3 workspace typecheck·5파일/98테스트 통과, `npm run build` 성공, `git diff --check` 통과.
- 실서버 검증: `BASE=http://localhost:18790 DATABASE_URL=postgres://postgres@127.0.0.1:55483/insight APP_DATABASE_URL=postgres://insight_app@127.0.0.1:55483/insight node scripts/e2e.mjs`(실행 시 개발용 자격증명과 TMPDIR를 프로세스 환경으로 전달) → `180 passed, 0 failed`. 새 PostgreSQL 16 컨테이너에 정식 migration 4개 적용, irumx_gateway→insight_app 전환을 쓰는 정식 빌드 서버, 하네스의 fake OAuth :18999 사용. 제출은 평점 오류·500자 성공·501자 실패 3회로 분당 제한 안에서 실행했다.
- 시행착오: 첫 E2E 실행은 기존 `submitted` 변수와 새 GET 응답 변수 이름이 겹쳐 SyntaxError, 같은 상태의 lint도 no-redeclare 실패했다. 새 변수만 submittedTask로 바꾸고 실제 E2E 전체 및 최종 check를 재실행해 모두 통과했다.
- 범위/정리: 3개 파일(프로덕션 1개)만 커밋, 빌드 산출물은 기존 .gitignore로 제외. 이번 검증 서버·컨테이너/익명 볼륨 종료·삭제, 기존 E2E 운영자 지정 테스트가 생성한 임시 credential fixture 디렉터리를 삭제했다. 기존 개발 설정 파일과 서비스는 변경하지 않았다. 브라우저 화면·운영 배포는 범위 밖이라 미실행.
- 스킬: Skill 호출 도구가 제공되지 않아 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/ 아래 completion-verification, systematic-debugging, test-driven-development의 SKILL.md를 직접 읽고 실패→최소 수정→성공→수정 되돌림 실패→복구→전체 검증을 적용했다.
- 커밋: `739ea33` — 제출 답변: 500자 초과 URL을 자르지 않고 입력 오류로 거절 (작성자 hkjang, 트레일러 없음). 커밋 후 작업 트리 깨끗함.

## 2026-10-17
- 선택: 지급 목록 CSV 내려받기의 DB 장애·네트워크 실패를 공통 점검 안내로 전달하기 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `client.ts:234 download()` 는 `request()` 와 달리 503 `db_unavailable` 에서 `reportDegraded()` 를 부르지 않고 fetch 실패를 날 `TypeError` 로 흘려보내, DB 가 닿지 않는 동안 「지급 목록 내려받기」만 공통 점검 배너를 띄우지 않았다. fetch 를 try/catch 로 감싸 `request()` 와 같은 `network` `ApiError`(status 0, 같은 한국어 문구)로 바꾸고, `!response.ok` 분기에서 만든 error 의 `code === 'db_unavailable'` 일 때 리스너를 먼저 부른 뒤 같은 error 를 그대로 던지도록 프로덕션 1개 파일만 고쳤다. 검증은 모킹 없이 테스트 안에서 띄운 node `http` 서버(포트 0)에 프로덕션 `api.download` 를 그대로 호출해 리스너 횟수·`ApiError` 의 status/code/message·저장된 파일 이름을 단정했고, 수정만 되돌려 같은 2건이 다시 실패하는 것으로 인과를 확인했다.
- 실패 재현: `AssertionError: expected +0 to be 1 // Object.is equality` (503 db_unavailable 알림 0회) / `AssertionError: expected TypeError: fetch failed to be an instance of ApiError` — 같은 실행에서 `Tests  2 failed | 2 passed (4)`, 성공 저장 테스트 2개는 이미 통과(회귀 가드).
- 보류 아이디어: download() 에 요청 취소(signal)·x-insight-stale 처리까지 request() 와 맞추기 (가치 2 / 위험 2 / S — 단, 두 함수를 한 함수로 통합하지는 말 것) / util/text.ts:isSafeLink() 죽은 코드·사설망 차단 주석 정리 (가치 3 / 위험 2 / S) / apps/web 의 나머지 client 가드(defaultMessage·CSRF 헤더 선택) 테스트 보강 (가치 3 / 위험 1 / S — 이번에 첫 web 테스트 부트스트랩이 생겼으니 후속이 싸다) / maskEmail·mailboxKey 경계 테스트 (가치 2 / 위험 1 / S) / README 로컬 설치 안내를 npm ci 로 맞추기 (가치 2 / 위험 1 / S)
- 과제서: 채택 — 과제서가 지목한 두 결함이 현재 코드에서 그대로 재현되었고, 지정한 프로덕션 1개 + 테스트 1개 범위와 수용 기준 4개를 바꾸지 않고 끝냈다.
- 검증: `npm ci --no-audit --fund=false` 종료 코드 0, `git status --short` 에 lockfile 변동 없음. 단일 테스트 수정 전 2실패/2통과 → 수정 후 4통과 → 수정만 되돌림 2실패/2통과 → 복구 후 4통과. 최종 `npm run check` 는 eslint + core/server/web 타입 검사 + `Test Files 7 passed (7) / Tests 113 passed (113)` 통과(직전 6파일 109개에서 web 1파일 4개 추가). `npm run build` 성공, `git diff --check` 통과, `git status --short` 에 지정 2개 파일만.
- 미실행: `node scripts/e2e.mjs` 와 `scripts/screenshots.mjs` 는 실 Postgres·Chromium 이 필요하고 이번 변경은 브라우저 측 알림 배선이라 돌리지 않았다. 브라우저에서 `.in-degraded` 배너가 실제로 뜨는 것까지는 확인하지 못했고, 알림(`onServiceDegraded`)이 정확히 1회 나가는 데까지만 증명했다. 기존 개발 DB·서버·설정 파일은 건드리지 않았다(`dev-reset.sh`·`ci-env.mjs` 미실행).
- 범위/정리: 서버 코드(`db.ts`·`errors.ts`·`http/pipeline.ts`·`routes/admin.ts`)·워크플로 파일·배너 UI·`request()` 본문은 읽기만 하고 바꾸지 않았다. `request()`/`download()` 통합과 204 캐시 무효화는 묶지 않았다. 새 의존성·jsdom 없이 기존 vitest node 환경만 썼고, 테스트 응답 본문은 수령 정보 원문 없이 더미 CSV 한 줄(`id,status` / `rw_1,ready`)로 했다. 브라우저 전역(window/document/URL.createObjectURL)은 `afterEach` 에서 원래 값으로 복원하고 `onServiceDegraded` 해제 함수도 매번 부른다.
- 커밋: `b55ab2d` — 지급 목록 내려받기도 DB 장애·네트워크 실패를 공통 점검 안내로 알리기 (작성자 hkjang, 트레일러 없음). 커밋 후 작업 트리 깨끗함.

## 2026-10-17
- 선택: PR #4 의 3초 CI 실패 원인 추적(수정 과제) → 저장소 원인 없음으로 분기 B 기록 + 차선 후보 구현: API client 의 세션·오류 계약 테스트와 204 쓰기의 캐시 무효화 누락 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 수정 과제는 **코드를 고치지 않고 분기 B 로 끝냈다** — 주석 원문은 이번에도 못 받았지만(비공개 저장소, `gh` 미인증, 익명 API 는 repo·run·annotations 모두 404), 과제서가 "유일한 결정적 증거" 로 지목한 `annotations_count: 2` 가 **실패 신호가 아님을 반증했다**: 같은 워크플로의 기록된 성공 체크런 5건(098cc43·bfbb56dd·c6a33997·fde293d3·739ea33, 10-07~08, 각 219~236초, conclusion success)도 전부 `annotations_count: 2` 다. 2 는 이 워크플로가 쓰는 마켓플레이스 액션 수(checkout@v4·setup-node@v4)와 같아 상시 deprecation 경고로 보이며, 따라서 다음 회차는 주석을 더 쫓을 필요가 없다. 또 `.github/` 는 마지막 녹색 CI(739ea33)부터 b55ab2d 까지 **바이트 동일**(`git diff 739ea33 601659d -- .github/` 빈 출력)이라 워크플로 설정 오류가 아니고, 같은 날 같은 계정의 다른 비공개 저장소 잡들이 06:36 전후로 정상 실행(jikim 07:29 48s/156s, kanpic 08:19 444s, moyro 10:28 639s, postra 13:32)돼 **계정 과금·스펜딩 리밋·러너 전면 장애도 배제**된다. b55ab2d 의 부모가 정확히 origin/main(601659d)이라 머지 레퍼런스 경합도 근거가 약하다. 3초에 `failure` 가 난 이유 자체는 **밝히지 못했다** — 추정으로 단정하지 않고 `check.yml` 은 한 줄도 건드리지 않았다(게이트 약화 없음). 남은 시간에 과제서가 허용한 차선 후보를 했다: 지금까지 web client 에는 테스트가 하나도 없어, 실제 node `http` 서버(포트 0)에 프로덕션 `api.get/post/put/del` 을 그대로 호출해 ① 회원/운영 화면에 따른 `yi_csrf`/`yi_acsrf` 선택과 `x-insight-session`, ② 읽기에는 CSRF 를 붙이지 않음, ③ 상태 코드 8개의 `defaultMessage`·서버 메시지 우선·`retry-after`·401 알림을 못 박았다. 그 과정에서 `request()` 가 **204 응답에서 `invalidateAll()` 앞서 조기 반환**해 cache.ts 가 문서로 약속한 첫 규칙("성공한 모든 쓰기는 캐시를 비운다")을 어기는 것을 테스트로 먼저 재현하고 프로덕션 1개 파일 5줄로 고쳤다. 검증: `npm run check`(eslint + core/server/web 타입 검사 + **7파일 121개 통과**, 직전 6파일 109개), `npm run build` 성공.
- 실패 재현: `AssertionError: expected 'old list' to be undefined` — 204 DELETE 뒤에도 캐시 값이 남음(`Tests 1 failed | 11 passed (12)`, 나머지 11개는 기존 계약 회귀 가드로 이미 통과). 수정만 되돌리니 같은 1건이 다시 실패, 복구하니 12개 통과. 계약 가드 쪽은 결함 수정이 아니라 기존 동작 고정이라 red 단계가 없으므로 변이 검증으로 대신했다: `csrfToken()` 의 쿠키 선택·`isAdminSurface()`·403 문구를 일부러 틀리게 바꾸니 해당 3개 테스트가 바로 실패(`Tests 3 failed | 9 passed`), 되돌리니 12개 통과.
- 보류 아이디어: download() 에 signal·x-insight-stale 까지 request() 와 맞추기 (가치 2 / 위험 2 / S — PR #4 와 같은 줄을 건드리므로 머지 후에) / util/text.ts:isSafeLink() 죽은 코드·사설망 주석 정리 (가치 3 / 위험 2 / S) / maskEmail·mailboxKey 경계 테스트 (가치 2 / 위험 1 / S) / README·architecture 의 낡은 검증 개수 안내를 범위 기술로 바꾸기 (가치 2 / 위험 1 / S) / respond.ts:noContent() 호출처 0 — 204 계약을 서버 쪽에서 쓸지 지울지 결정 (가치 2 / 위험 1 / S)
- 과제서: 차선 — 과제서가 결정적 증거로 지목한 주석 2건은 성공 회차에도 똑같이 2건이어서 실패 신호가 아니고(새로 반증함), 저장소 쪽 원인은 워크플로 동일·타 저장소 정상·fast-forward PR 로 전부 배제돼 분기 B(코드 미수정)로 끝낸 뒤 과제서가 지정한 차선 후보를 구현했다.

## 2026-10-18
- 선택: db-breaker 테스트의 벽시계 단정을 러너 속도와 무관한 증거로 바꾸기 (수정 과제) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: **실패한 CI step 이름은 미확인이다**(`gh` 미인증·비공개 저장소라 로그를 못 봤다). 대신 이 세션에서는 `npm ci` 가 돌아가서 **CI 의 모든 step 을 로컬에서 그대로 재현했고 전부 통과했다**: `601659d` 에서 `npm ci`(12초)·`npm run check`(109개 통과)·`npm run build`·migrate(0001~0004)·서버 기동(`/api/v1/health` ok)·`node scripts/e2e.mjs` **180 passed / 0 failed**·`node scripts/screenshots.mjs` **85 screenshots**, 그리고 PR #5 머지 내용(`cb269c5`, 부모가 정확히 `601659d` 라 fast-forward = CI 가 본 트리)을 `git archive` 로 꺼내 별도 `npm ci` 후 `npm run check` → **121개 통과**. 즉 저장소 내용 쪽 원인은 이번에 **실행으로** 배제됐다(앞선 두 회차는 정적 분석까지였다). 과제서 분기 C 지시대로 분기 A 의 벽시계 정리는 끝냈다: `db-breaker.test.ts` 의 시계 단정 4곳 중 `:127`(50ms, 차단기 열렸을 때 I/O 없는 경로 = 순수 CPU 시간)은 삭제하고, 나머지 3곳은 `RETRY_BUDGET_MS=2500` 을 이름으로 적은 `BUDGET_CEILING_MS = 4 × 2500` 한 상수로 바꿔 "무엇의 상한인가"를 주석으로 못 박았다. 같은 의도는 속도와 무관한 증거로 옮겼다 — 시도 횟수(`dropped`, `MOST_CONNECTS = 4×3`), `held.size` 증가(SILENT 는 한 번만 연결), `AppError.cause` 의 드라이버 코드(`CONNECTION_CLOSED` / `CONNECT_TIMEOUT`), 차단기가 거절한 오류는 `cause` 가 없다는 것, `retryAfter ≤ 10`(첫 대기 10초). 프로덕션 파일은 0개, `check.yml` 은 무변경(`git diff --stat` 이 테스트 1파일만).
- 실패 재현: **못 함 — CI 실패를 재현하지 못했다.** 1 코어에 스피너 8개를 붙여(`taskset -c 0`) 돌려도 11개 전부 통과했고(첫 테스트 266ms / 상한 5000ms, 여유 19배), 부하 없이도 통과한다. 그래서 이 변경은 결함 수정이 아니라 테스트 경화이고, 새 단정이 실제로 무는지를 변이 검증으로 증명했다: `db.ts` 의 `maxAttempts 4→8` 과 차단기 `retryAfter` 단위 오류(`/1000→/100`)를 넣으니 새 파일은 `AssertionError: expected 24 to be less than or equal to 12` / `expected 100 to be less than or equal to 10` 로 실패하고, `isConnectionLost` 에 `CONNECT_TIMEOUT` 을 넣으니 `expected 3 to be 1`. **같은 세 변이를 넣고 옛 파일(HEAD 버전)을 돌리면 `Tests 11 passed (11)`** — 즉 지운 시계 단정은 이 회귀들을 하나도 잡지 못했고, 바뀐 뒤에 잡는다. 게이트는 느슨해지지 않고 조여졌다.
- 보류 아이디어: respond.ts:noContent() 호출처 0 — 204 계약을 지울지 쓸지 정리 (가치 2 / 위험 1 / S) / download() 에 signal·x-insight-stale 까지 request() 와 맞추기 (가치 2 / 위험 2 / S, PR #4 머지 후) / util/text.ts:isSafeLink() 죽은 코드·사설망 주석 정리 (가치 3 / 위험 2 / S) / maskEmail·mailboxKey 경계 테스트 (가치 2 / 위험 1 / S) / README·architecture 의 검증 안내를 고정 개수 대신 범위 기술로 (가치 2 / 위험 1 / S)
- 과제서: 채택 — 분기 A 의 벽시계 4곳은 과제서가 읽은 그대로였고 지정 파일 1개만 고쳤다. 다만 과제서가 기대한 인과(이 단정이 CI 를 깬다)는 **성립하지 않는다**: 1 코어·9배 과포화에서도 여유가 19배였다. 저장소 내용은 e2e·screenshots·PR #5 의 check 까지 전부 로컬 실행으로 통과하므로, 다음 정찰은 저장소 쪽을 더 파지 말고 **CI 로그 접근(`gh` 인증)이나 러너/서비스 컨테이너 초기화 단계**를 확보하는 것부터 하기를 권한다.
- 검증: `npm ci --no-audit --fund=false` 종료 0, lockfile 무변동. 단일 테스트 수정 전 11통과 → 수정 후 11통과(변이 3건으로 red 확인) → 부하(1코어·스피너 8개)에서도 11통과. 최종 `npm run check`(eslint + core/server/web 타입 검사 + 6파일 109개)·`npm run build` 통과, `git diff --check` 통과, `git diff --stat` 은 테스트 1파일(+47/-8)뿐이고 `.github/` diff 는 빈 출력.
- 미실행/정리: 내 변경은 테스트 전용이라 변경 **후** e2e·screenshots 는 다시 돌리지 않았다(변경 전 `601659d` 에서 180/85 통과를 확인했다). 검증용 Postgres 16 컨테이너(`insight-ci-repro-pg`, 포트 55499)·서버(18791)·`/tmp/pr5`·스크린샷 디렉터리는 모두 지웠고, `.env` 파일은 만들지 않고 환경변수로만 넘겼다(`ci-env.mjs`·`dev-reset.sh` 미실행). Chromium 은 이 런의 임시 홈(`$HOME/.cache/ms-playwright`)에 설치됐다.
- 커밋: `137520a` — db-breaker 테스트: 러너 속도 대신 시도 횟수·드라이버 오류·차단기 상태로 단정 (작성자 hkjang, 트레일러 없음). 커밋 후 작업 트리 깨끗함.

