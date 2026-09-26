# 회차 노트 2026-09-26-131233-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:12] base pinned — main@511778c
- [러너 13:12] autonomy release — 

## 정찰 노트
- 1순위로 oidcCORS를 고른 이유: 이 저장소에 이미 있는 `realmLookupFailed`(oidc.go:60)를 discovery·jwks·authorization·revocation·logout 다섯이 쓰는데 `realmFromPath` 호출자 중 `oidcCORS`만 안 쓴다 — 결함이 코드 안의 기존 관례와 대조되어 근거가 짧고, 응답을 바꾸지 않으므로 위험이 낮다. 차선인 userinfo 카운터를 제친 이유는 userinfo의 401/500이 이미 `resso_http_requests_total{route,status}`에 보여 보류 노트의 가치가 부풀려져 있었기 때문이고, client_id 접근 로그(5회 연속 보류)는 미검증 입력 스크럽과 context 왕복 배선이 S를 넘겨서다.
- 추측으로 적은 것(과제서에도 "미확인"으로 명시): `WebOriginAllowed`(store/clients.go:348)가 읽는 테이블 이름을 열어 보지 않았다 — 수용 기준 (c)의 RENAME 대상이 `clients`인지 별도 origin 테이블인지 구현자가 먼저 확인해야 한다. 그 외 인용한 행 번호·함수·테스트 이름(integration_test.go:6904 lockedBuffer, :9344 JWKS 캐시 테스트)은 모두 직접 열어 확인했다.
- 구현자가 조심할 것: (1) 미등록 오리진에는 로그를 남기면 안 된다 — `oidcCORS`는 프로토콜 라우트 전체에 걸리고(server.go:108) Origin 헤더는 인증 없이 바꿀 수 있어 로그 폭주 자리다. 이것은 부수 조건이 아니라 수용 기준이다. (2) 조회 실패에 CORS를 허용하는 쪽으로 기울면 기록 과제가 정책 변경이 되어 위험 등급이 달라진다 — fail-closed 유지. (3) 오리진 원문을 로그에 넣지 말 것(운영자 규칙: 로그로 넘기는 값은 식별자로).
- 이번 회차는 코드를 한 줄도 바꾸지 않았고 테스트도 돌리지 않았다(읽기만). 작업 트리 깨끗.
- [러너 13:17] scout done — oidcCORS가 조회 실패로 CORS 헤더를 뺀 것을 기록하기 — `realmFromPath` 오류와 `WebOriginAllowed` 오류 (가치 3 / 위�

## 구현 노트
- 무엇을·왜: `oidcCORS`(middleware.go:87)의 `err == nil`/`allowErr == nil && allowed`를 풀어 써, 이쪽 장애 두 경우에만 Error 한 줄을 남겼다 — `realmFromPath`의 비-ErrNotFound 오류는 기존 `realmLookupFailed(r,"cors",err)`, `WebOriginAllowed` 오류는 전용 한 줄(`trace_id`·`realm`·`error`). 응답·헤더·fail-closed는 그대로다(커밋 5b260c7).
- 확신 없는 곳·검증 못 한 것: (1) 전용 로그 줄의 **문구**를 테스트가 부분 문자열(`"a CORS origin could not be checked"`)로 단언한다 — 문구를 고치면 테스트가 깨진다(의도적이지만 취약한 결합이다). (2) `WebOriginAllowed` 실패를 **`clients` 테이블 RENAME** 으로만 재현했다. 다른 종류의 store 오류(컨텍스트 취소, 풀 고갈)는 재현하지 않았다. (3) 부하 상황에서 이 Error 줄이 얼마나 나오는지는 측정하지 않았다 — 장애가 지속되면 요청마다 한 줄이다(미등록 오리진과 달리 외부에서 유발할 수는 없다). (4) `realm=master` 단언은 slog TextHandler 출력 형식에 의존한다.
- 일부러 하지 않은 것: 미등록 오리진(`allowErr == nil && !allowed`)에는 아무것도 기록하지 않았다 — 인증 없는 호출자가 Origin 헤더만 바꿔 채울 수 있는 자리라 과제서의 수용 기준 2다. 오리진 원문도 로그에 넣지 않았고(테스트가 `spa.example.com` 미노출을 단언), 지표·감사는 건드리지 않았다. 문서 변경 없음(운영자가 보는 계약이 바뀌지 않았다).
- 다음 역할이 조심할 것: 새 테스트 `TestIntegrationCORSSaysWhenItCouldNotCheckAnOrigin`은 **실제 PostgreSQL 이 있어야 돈다**(`eval "$(scripts/test-services.sh)"` 후 같은 셸). 테스트가 `clients`·`realms` 테이블을 RENAME 했다가 되돌리므로 **병렬 실행에 안전하지 않다**(t.Parallel 없음, 기존 로그아웃 테스트와 같은 관례). 중간에 죽으면 `clients_hidden`/`realms_hidden`이 남는다.
- [러너 13:30] brief accepted — 채택 — 근거(98·99행의 버려진 `err`/`allowErr`, `realmFromPath` 호출자 중 유일하게 `realmLookupFailed`를 쓰지 않음)가 코드와 그대�
- [러너 13:31] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: `git show main:…/middleware.go`를 임시 복원해 새 테스트를 base에서 돌려 (c)(e) 단언 세 개가 실제로 FAIL함을 확인(HEAD에서는 PASS), 이후 파일 복원 — 작업 트리는 원래부터 더러웠던 `webui/dist/index.html`만 남음. `go test -race ./internal/httpserver` 전체 109.975s ok, gofmt·go build 깨끗. 응답·헤더·fail-closed 불변, `WebOriginAllowed`가 `clients`를 읽는 것(store/clients.go:355)까지 열어 확인.
- 거절 사유 하나: `docs/operations.md:33` — "`clients` 조회가 멈추면 **다음 두 줄**을 찾으세요"가 이제 거짓이다(새 줄이 세 번째다). 수리가 가장 먼저 볼 파일은 **docs/operations.md** 이고, 그 표에 한 행(메시지 / 어디서=oidcCORS / 답 불변·CORS 헤더만 빠짐, 증상=브라우저는 불투명 CORS 실패인데 접근 로그는 200) 추가로 끝난다. 직전 동종 커밋 ee75261·87f4ce9·a578657이 모두 같은 커밋에서 이 파일을 고쳤다.
- 못 본 것: 부하·장기 장애에서의 실제 로그량, LDAP·다중 Realm 조합, 컨텍스트 취소로 인한 `allowErr` 재현(추론만).
- 승인 후에도 남는 우려(릴리즈 노트/다음 회차): middleware.go:120-135 주석의 "외부에서 유발할 수 없다"는 단정은 과하다 — 클라이언트가 연결을 끊으면 context 취소로 같은 Error 줄이 난다(기존 성질이라 차단 아님). realms 장애 중에는 한 요청이 `endpoint=cors`+핸들러 두 줄을 남긴다. 테스트는 로그 문구에 부분 문자열로 결합되어 있어 문구 변경 시 함께 고쳐야 한다.
- 보안·법무 차단 없음: 인가·세션·정책 변경 없음, 로그 신규 값은 경로 realm·trace_id·store 오류 문자열뿐이고 Origin 원문 미기록(테스트가 단언). 개인정보 신규 수집·전송 없음.
- [러너 13:37] review rejected — 리뷰 거절: docs/operations.md:33 `clients` 조회가 멈추면 "다음 두 줄을 찾으세요. 둘 다 …" 라고 못 박은 표가 이 변경으로 사실과 달라졌다. 새 로그 줄(intern

## 수리 노트
- 맞았던 지적: docs/operations.md:33 의 "다음 두 줄". `WebOriginAllowed`가 `clients`를 읽는다는 것(store/clients.go:355)을 직접 열어 확인했으므로 새 로그 줄은 같은 장애의 세 번째 줄이 맞고, 운영자가 보는 증상(브라우저는 불투명 CORS 실패·접근 로그는 200)이 정본 표에 없던 것도 맞다. 틀린 지적은 없었다.
- 고친 방법: 문서만 고쳤다 — "두 줄/둘 다" → "세 줄/셋 다", 표에 한 행 추가. 코드·테스트·응답 계약은 그대로(커밋 cea3c36).
- 재검증: gofmt·go build·go vet 깨끗, 실제 PostgreSQL로 CORS 테스트 PASS(-v 로 SKIP 아님 확인), httpserver 전체 109s ok.
- 여전히 확신 없는 곳: 새 행의 "어디서" 를 "프로토콜 Endpoint 앞단 CORS 검사(`/realms/...` 전체)" 로 적었는데 다른 두 행이 Endpoint 이름을 쓰는 것과 표기 층이 다르다(미들웨어라 Endpoint 하나로 좁힐 수 없다). 표 밖 남은 사실 하나: realms 장애 중에는 한 요청이 `endpoint=cors` 줄과 핸들러 줄을 함께 남기는데 이는 문서화하지 않았다.
- 손대지 않은 것: `webui/dist/index.html`(회차 시작부터 더러운 빌드 산출물), 지표·감사·미등록 Origin 무기록 정책.
- [러너 13:41] repair done — # 수리 요약 (시도 1)  - 지적은 맞았다: `WebOriginAllowed`가 읽는 것이 `clients`(internal/store/clients.go:355)임을 직접 확인했고, 그래서 5b260c7이 추가한 `a CORS origin

## 비평 노트
- 직접 확인: main 의 middleware.go 를 임시로 되돌려 새 테스트를 돌려 (c) 두 줄과 (e) 한 줄이 실제로 FAIL 함을 눈으로 봤다(HEAD 에서는 PASS, 0.66s). 실제 PostgreSQL 로 `go test -race ./internal/httpserver ./internal/store` 117s/81s 모두 ok, gofmt·go vet 깨끗, 작업 트리는 회차 시작부터 더럽던 `webui/dist/index.html` 만 남음. 응답·헤더·fail-closed 불변((c) 가 단언), `oidcCORS` 가 `/realms/{realm}` 전체에 걸리는 것(server.go:107-108)과 `WebOriginAllowed` 가 `clients` 를 읽는 것(store/clients.go:355)까지 열어 확인. 테스트는 스키마별 격리(integration_test.go:265)라 RENAME 이 다른 테스트를 건드리지 않는다.
- 승인이어도 남는 우려 ①(릴리즈 노트 아님, 다음 회차): middleware.go:124-130 주석의 "A store that stopped answering is not reachable that way" 는 과하다 — 클라이언트가 연결을 끊으면 context 취소로 같은 Error 줄이 난다. 비대칭(미등록 Origin 무기록 / store 오류 기록)의 결론은 여전히 옳고 `realmLookupFailed` 가 이미 다섯 Endpoint 에서 같은 성질을 갖지만, 단정 문구는 다음에 손볼 때 완화할 값어치가 있다. 접근 로그가 어차피 요청당 한 줄이라 증폭은 2배에 그친다.
- 승인이어도 남는 우려 ②: docs/operations.md:38 의 "접근 로그에는 200으로 남습니다" 는 `clients` 조회만 흔들릴 때 참이다. clients 테이블이 통째로 멈추면 Token·UserInfo 는 핸들러가 스스로 500 을 답하므로 200 이 아니다 — 이 행이 설명하려는 혼동스러운 경우(JWKS·discovery)에는 정확하니 거절 사유로 보지 않았다.
- 승인이어도 남는 우려 ③: 테스트가 로그 문구에 부분 문자열(`"a CORS origin could not be checked"`, `"endpoint=cors"`, `"realm=master"`)로 결합돼 있어 문구나 slog 핸들러를 바꾸면 함께 고쳐야 한다. 테스트 중간에 죽으면 그 테스트 전용 스키마에 `clients_hidden`/`realms_hidden` 이 남지만 스키마가 테스트마다 새로 만들어져 영향 없음.
- 못 본 것: 부하·장기 장애에서의 실제 로그량 측정, context 취소·풀 고갈로 인한 `allowErr` 실측(추론만), LDAP·다중 Realm 조합. 보안·법무 차단 없음 — 인가·세션·정책 변경 없고, 새로 남기는 값은 조회에 성공한 경로 realm·trace_id·store 오류 문자열뿐이며 Origin 원문 미기록을 테스트가 단언한다. 개인정보 신규 수집·전송 없음.
- [러너 13:46] review approved — 리뷰 승인 (risk=low)
- [러너 13:46] pr created — https://github.com/hkjang/ReSSO/pull/29
