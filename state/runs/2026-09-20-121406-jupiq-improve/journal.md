# 회차 노트 2026-09-20-121406-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:14] base pinned — main@52fb7f9
- [러너 12:14] autonomy release — 

## 정찰 노트
- 선택 이유: main@52fb7f9에는 메일·MCP OAuth·Discovery 캐시 브랜치가 아직 머지되지 않아 관련 보류 항목 5건이 모두 막혀 있고, 캡처류는 앱 기동·Playwright 승인 비용이 커 예산 홀드 위험이 있음. queryInt→400은 코드로 확인된 실제 결함(hub_id=abc가 "모든 Hub")이고 같은 저장소의 /users/{username}에 이미 같은 규칙의 선례(boundedUserDetailQueryInt)가 있어 S 크기로 끝남.
- 확신 없는 곳: OpenAPI 400 응답 추가가 계약 테스트에 걸리는지(경로 존재만 검사하는 것으로 보이나 미확인), ADMIN_GUIDE에 API 오류 코드 표가 있는지(미확인 — 과제서에 grep으로 확인하라고 적음), 외부 자동화 클라이언트가 음수 page를 보내는지(프런트는 안 보냄, 외부는 미확인).
- 구현자 주의: store의 pageBounds·limit 보정은 그대로 두고 API 층에서만 거부(0은 유지, 음수·비정수만 400). 가짜 Store 주입 대신 nil Store로 "store 이전 400"을 증명. core_handlers.go는 미머지 브랜치 셋과 겹치므로 호출 줄만 최소 변경. 러너 환경에서 awk·sort -t·리다이렉션 등은 승인 필요 — 검증은 go test 한두 번으로.
- 프로필을 새로 씀: 어제 프로필이 main에 없는 mcp_oauth.go·authtest를 있는 것처럼 적고 있어 바로잡음.
- [러너 12:18] scout done — 목록·지표 API의 정수 질의 파라미터(page·page_size·limit·hub_id)가 정수가 아니거나 음수이면 400 `invalid_query`�

## 구현 노트
- 바꾼 것: `queryInt`를 `(int, error)`로 바꾸고 `queryIntOrReject`·`pageQuery`(400 `invalid_query` 직접 응답)를 얹어 users·servers·local-users·audit·metrics·usage/consumption·자원 목록 7개 핸들러의 호출 줄만 수정(6a246f9). OpenAPI에 `InvalidQuery` 응답 컴포넌트와 각 경로 `'400'` 추가. 별도 커밋 695591e는 HEAD부터 있던 `openapi.yaml` 117·134행의 인용 없는 `?`·`,`를 따옴표로 감싼 것(yaml.v3·PyYAML이 그 줄에서 파싱 실패하던 문제) — 문장은 동일.
- 확신 없는 곳·검증 못 한 것: 외부 자동화 클라이언트가 음수 page나 비정수 값을 보내고 있었는지는 확인할 수 없음(프런트는 항상 숫자). `Atoi`가 `+7`을 7로 받는 것은 종전과 같아 그대로 둠. 프런트 `npm test`는 node_modules가 없어 돌리지 않았음(프런트 파일 무변경). `/metrics` limit 스키마에 `minimum: 0`을 추가한 것은 서버 실제 동작(0 허용→store 보정)에 맞춘 문서 변경.
- 일부러 하지 않은 것: store의 `pageBounds`·limit 보정과 상한 초과(page_size 200 등)의 400 전환 — 정수 입력 응답 무변경 조건 때문. `boundedUserDetailQueryInt`(`invalid_pagination`) 재구성 — 동작·메시지 유지가 조건이고 미머지 브랜치와의 충돌을 피하기 위해 손대지 않음. 가이드·PDF — API 오류 코드 표가 없음(`grep invalid_pagination docs/*.md` 0건).
- 다음 역할이 조심할 것: `TestScopedRBACHTTPFailClosedIntegration`의 새 단언(`?page=abc`→400)은 `JUPIQ_INTEGRATION_TEST_DSN`이 있어야 돌고 없으면 skip — 이번 회차엔 임시 postgres:16-alpine(127.0.0.1:55499)으로 실제 통과시켰고 컨테이너는 지웠음. `query_params_test.go`는 Store nil인 `Server{}`를 쓰므로 누군가 핸들러에서 파싱 순서를 store 호출 뒤로 옮기면 panic으로 실패한다(의도된 설계). openapi.yaml을 다시 편집할 때 flow mapping `{ description: … }` 안에 `?`·`,`·`:`를 넣으면 다시 파싱이 깨지니 따옴표로 감쌀 것.
- [러너 12:28] brief accepted — 채택 — 과제서의 진단(queryInt가 Atoi 실패를 기본값으로 바꿈, 선례 boundedUserDetailQueryInt)이 코드와 일치했고 수용 기준 5개
- [러너 12:29] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: `git diff main...HEAD` 전체(Go 5개·테스트 3개·openapi.yaml), queryInt 호출처 전수(helpers.go 외 없음), 프런트 page/page_size/limit/hub_id 생성부(항상 숫자), `/approvals`~`/costs`가 resourceList 경유임, `go test ./internal/api/` 통과. 못 본 것: DSN 통합 단언 실행, openapi.yaml YAML 파서 검증(계약 테스트는 텍스트 파서).
- 테스트 유효성: query_params_test.go는 Store nil이라 구 코드에선 nil 역참조 panic → 실패, helpers_test.go는 구 시그니처와 컴파일 불가. 둘 다 변경을 실제로 핀한다.
- 승인이어도 남는 우려: (1) resource_handlers.go:43 owner_user_id=abc는 여전히 조용히 무시 — hub_id와 같은 부류, 다음 회차 후보. (2) openapi의 page_size minimum: 1 vs 서버 0 허용 불일치 잔존. (3) Atoi 범위 초과 메시지가 원인을 덮음.
- 릴리스 노트: 음수·비정수 page/page_size/limit/hub_id → 400 invalid_query는 외부 자동화 클라이언트에 동작 변경.
- 보안·법무 차단 없음 — 입력 검증을 좁히는 변경, 인가 순서(scopedAccess → 파싱) 유지, 개인정보 신규 수집 없음.
- [러너 12:30] review approved — 리뷰 승인 (risk=low)
- [러너 12:30] pr created — https://github.com/hkjang/jupiq/pull/20
- [러너 12:34] ci passed — 검사 3개 모두 success
- [러너 12:34] merge done — 695591e
- [러너 12:40] release published — v1.7.2
- [러너 12:44] assets verified — v1.7.2 자산 1개 (이전 v1.7.1: 1)
