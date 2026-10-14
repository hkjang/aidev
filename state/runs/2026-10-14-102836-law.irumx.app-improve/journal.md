# 회차 노트 2026-10-14-102836-law.irumx.app-improve — law.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:28] base pinned — main@c62b3a0
- [러너 10:28] autonomy release — 

## 정찰 노트
- 보류 1순위(pipeline.ts:489 dev 스택)를 확인하러 갔다가 더 센 것을 옆에서 찾았다 — :54 `Background.run` 의 catch 는 `env.dev` 게이트가 없어 **운영**에서 `String(err)` 를 원문으로 찍고, after-commit 작업 실패가 전부 이 경로로 모인다. dev 전용 1줄(가치 3)이 운영 로그 누출(가치 4)로 커졌고 여전히 프로덕션 1파일·3줄이라 S 를 유지했다. validate.ts 테스트(M, 프로덕션 0파일)는 결함이 아니라 공백 메우기여서 제쳤다.
- 추측으로 적은 것: after-commit 실패 문구에 실제로 DSN·토큰이 섞여 나오는 사례를 **재현해 보지 못했다**(핸들러 체인 jobs/handlers.ts 는 열지 않았다). 근거는 `redactSecrets` 가 DSN 규칙을 갖고 있다는 사실과 경로 구조뿐이다. 누출 가능성이 아니라 "가려지지 않는 출력 경로" 자체를 고치는 과제로 적었다.
- 샌드박스 제한으로 `npm run test`/`npx vitest` 를 실행하지 못했다 — 기준선 통과는 **미확인**(지난 회차와 동일). 구현자가 먼저 한 번 돌려 기준선을 잡을 것.
- 구현자가 조심할 것: ① redact 를 slice **앞**에 둘 것(먼저 자르면 패턴이 끊겨 빠져나간다) ② 테스트 파일명을 `redact-secrets.test.ts` 로 하지 말 것 — 미머지 선행 회차가 그 경로를 쓴다 ③ `redactSecrets` 정규식은 건드리지 말 것(호출 지점만 늘린다) ④ `logFailure` 는 비공개라 :111/:489 를 가짜 deps 로 시험하려 들지 말 것.
- [러너 10:32] scout done — 오류 stderr 로그 세 경로를 redactSecrets 로 통과시키기 — after-commit 실패 로그가 운영에서 원문을 찍는다 (가

## 구현 노트
- pipeline.ts 에서 오류를 stderr 로 쓰는 네 곳 중 세 곳이 redactSecrets 를 안 거쳤다. 세 곳에 호출을 붙였다 — :54 Background.run(운영 경로, env.dev 게이트 없음), :111 dev 한 줄, :489 원본 스택. 정규식은 손대지 않았다(호출 지점만 늘림). 프로덕션 1파일 3줄 + 테스트 1파일.
- :54 는 redact 를 slice **앞**에 뒀고, 그 순서가 왜 중요한지를 추측으로 남기지 않고 테스트로 실증했다 — 300자 경계에 DSN 을 걸쳐 놓으면 먼저 자를 때 뒤쪽 @ 가 잘려 패턴이 어긋나고 s3cret 이 로그에 남는다(수정 전 실제 출력 확인).
- 확신 없는 곳: **:111 과 :489 는 실행 테스트로 덮지 못했다.** logFailure 가 비공개 함수여서 handleRequest 전체 deps(Database·Settings·Router)를 가짜로 조립해야 하는데, 운영자 지시(손으로 만든 대역을 근거로 삼지 말 것)에 어긋나므로 일부러 하지 않았다. 둘은 dev 전용·각 1줄이고 코드 검토로 닫았다 — 비평가는 이 두 줄을 눈으로 확인해 주기 바란다. 치환 함수 자체는 선행 회차의 redact-secrets.test.ts(미머지)와 이번 Background 경로 시험이 받친다.
- 또 하나 검증 못 한 것: after-commit 실패 문구에 실제로 DSN·토큰이 섞여 나오는 **운영 사례는 재현하지 않았다**(jobs/handlers.ts 체인은 열지 않았다). 고친 것은 "가려지지 않는 출력 경로" 자체다.
- 일부러 하지 않은 것: 주민번호 패턴(:503) 과잉 차단 손대지 않음(사람 판단 대기, 지시대로 넓히지 않음). alertOperator(:145)는 이미 redact 를 거치므로 건드리지 않음. Background 의 pending/keepAlive/settled 동작은 그대로 — catch 안 detail 한 조각만 바꿨다.
- 다음 역할이 조심할 것: 새 테스트는 **DB 불필요**, `npx vitest run packages/core/src/__tests__/background-redact.test.ts` 로 단독 실행된다. 두 번째 시험이 로그 JSON 의 필드 이름·순서·라벨을 고정하므로, 나중에 request_id 같은 필드를 더하면 그 시험을 같이 갱신해야 한다(의도된 잠금이다).
- 검증: `npm run check` 통과 — lint 0 errors / 646 warnings(기준선 동일), typecheck 3개 워크스페이스, vitest 7 files 106 tests(기준선 6/103 + 신규 3). npm install 이 package-lock.json 을 흔들어 놔서(npm 버전 차이로 peer/libc 필드) 커밋 전에 git checkout 으로 되돌렸다 — 커밋은 의도한 2파일뿐이다.
- [러너 10:36] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(`:54` 에 `env.dev` 게이트 없음, `:111`/`:489`, `:487` 만 redact 통과), 수용
- [러너 10:37] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, 차단 없음). 테스트 유효성을 직접 재현했다 — pipeline.ts 만 main 으로 되돌리면 3개 중 2개가 실패하고(원문 `s3cret`·`abcdefgh12345678` 노출, 300자 경계에서 `@` 가 잘려 패턴 우회), HEAD 로 되돌리면 전부 통과. 실패 출력이 고치려는 증상과 일치한다. 원장에 `- 실패 재현:` 줄이 없었으므로 비평이 대신 남긴다.
- 구현자가 의심한 :113/:491 을 눈으로 닫았다 — :113 은 삼항식 전체를 감싸 의미 동일, :491 은 `err.stack` 진위 검사 뒤 string 을 넘긴다. pipeline.ts 의 stderr 출력 네 곳을 세어 전부 redact 를 거치는 것도 확인했고 정규식은 무수정이다. `npm run check` 재현 통과(lint 0 errors/646 warnings, 7 files/106 tests) — 원장 수치와 일치.
- 승인에도 남는 우려 ①: 두 번째 시험 제목은 'JSON 필드 이름·순서' 고정이라 말하지만 `JSON.parse` 후 `toEqual` 이라 **키 순서는 단언되지 않는다**. 다음 회차가 '순서 잠금' 으로 신뢰하면 안 된다 — 릴리즈 노트에 적을 값.
- 우려 ②: after-commit 실패 문구에 실제로 DSN·토큰이 섞여 나오는 운영 사례는 비평도 재현하지 못했다(jobs/handlers.ts 미열람). 고친 것은 출력 경로이고 그 경로의 존재는 코드로 확정되므로 지장 없으나, 릴리즈 노트는 "누출 사례 수정" 이 아니라 "가려지지 않던 경로 차단" 으로 써야 한다.
- 보지 않은 것: jobs/handlers.ts 핸들러 체인, pipeline.ts 외의 다른 파일 stderr 출력(이번 범위 밖), DB 필요한 시험 전부(e2e·acceptance·rls). 주민번호 패턴(:503) 과잉 차단은 이번 변경이 건드리지 않았고 여전히 사람 판단 대기 상태다.
- [러너 10:39] review approved — 리뷰 승인 (risk=low)
- [러너 10:40] pr created — https://github.com/hkjang/law.irumx.app/pull/4
- [러너 10:40] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 
