# 회차 노트 2026-09-22-220449-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:04] base pinned — main@e4a2ca0
- [러너 22:04] autonomy release — 
- [러너 22:11] scout done — 미래 시각으로 들어온 교류 기록을 거부하고, 교류 입력 검증을 이름 있는 순수 함수로 묶기 (가치 3 / 위�

## 구현 노트
- 무엇/왜: `createInteraction` 의 익명 구조체를 `interactionInput` 으로 올리고 검증을 `validateInteractionInput(in, now)` 로 뺀 뒤 `interactionFutureSkew = 5분` 을 넘는 미래 `occurred_at` 을 400 으로 막았다. 미래 한 건이 지표·last_interaction_at·Time Travel 에서 서로 다르게 읽히던 길을 입구에서 닫는다.
- 확신 없는 곳·검증 못 한 것: 실제 postgres 로 "미래 기록이 지표를 망가뜨린다"를 **재현하지 않았다**(이 세션에 DB 없음) — 근거는 코드 경로 읽기다. 다만 고치기 전 실제 핸들러가 nil store 에서 `data.go:437` 사람 존재 확인 쿼리까지 도달해 패닉나는 것은 확인했고(= 검증이 DB 앞에서 끝나지 않았다는 증거), 고친 뒤 같은 호출이 400 으로 끝난다. `time.Now()` 를 핸들러에서 넘기므로 서버 시계가 크게 어긋나면 정상 입력이 막힐 수 있다(5분 여유만 둠).
- 일부러 하지 않은 것: (1) 프런트 `InteractionDialog` 의 `max` — node_modules 가 없어 `npm ci`+build 없이는 MUI slotProps 타입까지 검증할 수 없고 페이지 테스트 기반도 없어 뺐다(서버 오류는 이미 같은 다이얼로그 Alert 로 보인다). (2) docs/API.md 한 줄 — API.md 에 교류 엔드포인트 절 자체가 없어 한 줄만 끼우면 붕 뜬다. (3) `weight` 묵음 보정과 `parseOrbitAt` 의 미래 거부 — 계약 변경이라 별건.
- 다음 역할이 조심할 것: 새 테스트(`internal/server/data_test.go`)는 DB 가 필요 없다. 통과 경로를 nil store 핸들러로 부르면 패닉나므로 그쪽은 순수 함수로만 검사했다 — 핸들러 테스트를 늘릴 때 같은 함정. `timetravel_db_test.go` 는 여전히 `ORBIT_TEST_DATABASE_URL` 없으면 skip 이다.
- [러너 22:16] brief accepted — 채택 — 근거(상한 없음, 세 경로가 같은 값을 다르게 읽음, INSERT 경로는 data.go 한 곳)가 지금 코드와 정확히 일치해 수용 
- [러너 22:16] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 reject. 서버 검증 자체는 옳고 테스트도 진짜다(main 에 얹으면 컴파일 실패/패닉 FAIL, 브랜치에서 7개 PASS, `go vet`·`go test ./...` 통과, 교류 INSERT 는 data.go:468 한 곳뿐).
- 막은 이유는 프런트 회귀다: `web/src/pages/PersonPage.tsx:434` 가 UTC 벽시계를 datetime-local 기본값으로 넣고 `:445` 에서 로컬로 다시 해석해, UTC 음수 오프셋 클라이언트는 기본값 그대로 저장하면 now+|offset|(NY 실측 +239.7분)을 보내 이제 항상 400 이다 — 수리는 이 파일부터 보고 기본값을 로컬 시각으로 만들면 표시 오류까지 같이 잡힌다.
- 못 본 것: 실제 postgres 재현(DSN 없음)과 웹 빌드/테스트(node_modules 없음). 타임존 결함은 node 로 날짜 계산만 실측했다.
- 남는 우려: 이미 저장된 미래 행은 정리되지 않으니 릴리즈 노트는 "앞으로 들어오는 입력만 막는다"로. 핸들러 테스트가 nil store 패닉에 기대는 구조라 회귀 시 패키지 테스트가 통째로 중단된다.
- [러너 22:20] review rejected — 리뷰 거절: web/src/pages/PersonPage.tsx:434,445 교류 기록 다이얼로그의 기본값이 UTC 벽시계를 datetime-local 필드에 넣고(`new Date().toISOString().slice(0,16)`) 저장할 때

## 수리 노트
- 비평이 맞았다: `PersonPage.tsx:435` 의 UTC 벽시계 기본값이 `:446` 에서 로컬로 되읽혀 NY 기준 now+239.9분을 보내고, 이번 커밋의 5분 상한과 만나 교류 기록 UI 가 통째로 막혔다(node 로 재현). 틀린 지적은 없었다.
- 구현 노트가 "node_modules 없어 프런트 검증 불가"라 했는데 이 워크트리에는 있었다 — `npm ci` 없이 바로 test/build 가 돌았다. 다음 역할은 없다고 단정하지 말 것.
- 고친 방법: 한 줄 인라인 대신 순수 함수 `web/src/datetimeLocal.ts`(로컬 구성요소 포맷, DST 안전)로 빼고 `datetimeLocal.test.ts` 3건을 붙였다 — 저장소 관례(순수 계산은 src/*.ts + 옆 테스트)를 따랐고, 옛 구현으로 되돌리면 실제로 FAIL 하는 것까지 확인했다.
- 검증: web 112건 PASS, `npm run build`(tsc -b 포함, `process` 타입이 없어 테스트는 `vi.stubEnv("TZ", …)` 로 작성) 성공, go vet/test PASS. 서버 쪽 변경은 손대지 않음.
- 확신 없는 곳: 실제 브라우저에서의 확인은 못 했다(jsdom 만). 이미 DB 에 들어간 미래 행은 그대로이며 릴리즈 노트는 "앞으로 들어오는 입력만 막는다"가 여전히 맞다.
- [러너 22:24] repair done — # 수리 요약 (수리 시도 2)  - 지적이 맞다. 재현: `TZ=America/New_York node -e` 로 `new Date().toISOString().slice(0,16)` 기본값을 `new Date(...)` 로 되읽으니 now+239.9분(Asia/

## 비평 노트
- 판정 approve. 확인한 것: `go vet`·`go test ./...` 통과, 새 Go 테스트 7건 PASS, 웹 112건 PASS, `npm run build`(tsc -b 포함) 성공, 워크트리 clean. 교류 INSERT 는 data.go:468 한 곳뿐이고 mcp.go·ai.go·export.go 에 쓰기 경로가 없음을 직접 확인했다.
- 구현자가 의심한 자리를 먼저 봤다: (1) 검증이 DB 앞에서 끝나는지 — 순서상 소유 확인(data.go:461)보다 앞이고, 400 이 사람 존재 여부와 무관하게 같아 열거 채널도 새로 안 생긴다. (2) 주석이 근거로 든 `math.Max(0, days)` 는 data.go:494 에 실제로 있어 설명과 코드가 일치한다. (3) 수리가 만든 TZ 테스트는 NY/Seoul 에 서로 다른 값을 단언하므로 `vi.stubEnv("TZ")` 가 실제로 먹힌다는 증거 — 옛 UTC 구현으로는 통과할 수 없다.
- 못 본 것: 실제 postgres 재현(DSN 없음, timetravel_db_test 는 SKIP)과 실제 브라우저 확인(jsdom 만). 보안·법무 차단 사유 없음 — 인증·권한·암호화·마이그레이션·워크플로 미변경, 새 의존성 없음, 오류 메시지에 입력이 되비치지 않음, 새로 수집·전송하는 개인정보 없음.
- 남는 우려(릴리즈 노트): 이미 저장된 미래 행은 정리되지 않으니 "앞으로 들어오는 입력만 막는다". 5분 여유는 서버 시계 기준이라 클라이언트 시계가 5분 넘게 앞서면 정상 입력도 400 이 된다.
- 다음 회차가 알 것: data_test.go 의 핸들러 테스트는 nil store 패닉에 기대므로 회귀 시 패키지 테스트가 통째로 중단된다. PersonPage.tsx:434 기본값은 여전히 마운트 때 한 번만 계산되어 다이얼로그 재개봉·저장 후 갱신되지 않는다(이번 변경 이전부터, 방향은 개선).
- [러너 22:28] review approved — 리뷰 승인 (risk=low)
- [러너 22:28] pr created — https://github.com/hkjang/orbit/pull/9
- [러너 22:31] ci passed — 검사 1개 모두 success
- [러너 22:31] merge done — 1dc3fe3
- [러너 22:37] release published — v0.6.3
- [러너 22:38] assets verified — v0.6.3 자산 1개 (이전 v0.6.2: 1)
