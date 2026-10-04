- 과제: 이벤트 목록 API의 실효 조회 상한과 창 포화 여부 명시 (가치 2 / 위험 1 / 작업량 S)
- 왜: `GET /admin/k8s/events?limit=1000`은 `handleK8sEvents`에서 1000을 전달하지만 `ListK8sEvents`는 500으로 제한하며 응답에는 events만 있어 실제 조회 범위를 알 수 없다. 기존 목록에 실효 limit과 창 포화 메타데이터를 더해 API 이용자가 일부 이벤트를 전체로 오인하지 않게 한다.
- 수용 기준: 1) 기존 `events` 배열, `last_seen DESC`, `cluster_id` 필터, 인증·메서드·DB 오류 응답을 유지하고, 성공 응답에 정수 `limit`(실효 상한)과 불리언 `window_full`을 항상 추가한다. 생략/공백/문자열/0/음수 limit은 기존대로 100, 1~500은 그대로, 501 이상은 500이다. 2) `window_full = len(events) >= limit`; true일 때만 비어 있지 않은 `window_notice`를 추가하여 “조회 상한에 도달했으며 더 오래된 이벤트가 있을 수 있음”을 안내한다. 정확히 500행과 501행 모두 limit=500에서는 true이며 누락 확정·정확한 전체 건수·has_more를 주장하지 않는다. 3) 실제 SQLite와 `Server.Routes()`를 거치는 HTTP 테스트가 limit=1000에서 최대 500행/limit=500/full=true, 정확히 상한인 경우 true, 상한 미만/빈 결과 false(빈 events는 []), 비정상 limit의 기본값, 다른 클러스터 행의 비혼입 및 최신순 반환을 증명한다. HTTP가 보고한 실효 limit과 스토어에서 실제 읽은 행 수의 계약을 함께 검증한다.
- 건드릴 파일: `internal/store/k8s.go:ListK8sEvents` 및 신규 `K8sEventQueryLimit(limit int) int` — 기존 `boundedLimit(limit,100,500)`을 작은 공유 헬퍼로 추출하여 스토어와 핸들러가 같은 정규화 함수를 사용(동작·시그니처 무변경); `internal/proxy/admin_k8s.go:handleK8sEvents` — 기존 `intParam` 결과를 공유 헬퍼로 정규화하여 조회와 응답에 사용, 포화 메타데이터 추가; `internal/proxy/admin_k8s_events_window_test.go`(신규) — 위 HTTP 회귀 테스트; `docs/K8S_OPERATIONS_HUB.md`의 기존 GET `/admin/k8s/events` 행 및 인접 설명 — 필드·기본값·상한·포화의 보수적 의미 기록. 프로덕션 파일 2개, 테스트 1개, 문서 1개 이내.
- 검증 명령: 저장소 루트에서 `go test ./internal/proxy -run 'K8sEvents|NotifyScan' -count=1` (정찰 실행 PASS, 1.987s; 지금은 신규 K8sEvents 테스트가 없으므로 기존 NotifyScan 테스트의 기준선이다). 구현 후 신규 테스트 이름을 `TestK8sEvents...`로 시작하고 같은 명령을 실행한다. 이어 `gofmt -l internal/store/k8s.go internal/proxy/admin_k8s.go internal/proxy/admin_k8s_events_window_test.go` (출력 없어야 함), `go build ./...`, `go vet ./...`, `go test ./... -count=1`. 후자의 전체 게이트는 정찰에서 재실행하지 않았으며 최근 회차는 proxy 약 98초였으니 시간을 남긴다.
- 위험과 피할 것: 조회 상한 500, SQL 정렬, 다른 events/revisions 호출자, notify/PSS/권한/auth/DDL/workflows/릴리즈 파일은 변경하지 않는다. 전역 `intParam`·`boundedLimit`의 계약도 바꾸지 않는다. `limit+1`은 스토어 상한에 잘리므로 정확한 누락 검출처럼 사용하지 않는다. count 쿼리·페이지네이션·UI 표시까지 확장하지 않는다. 같은 정규화 함수를 두 경로가 호출하게 하되 범용 파서 통합은 하지 않는다. Fake store/손으로 만든 Server 대역/소스 문자열 검사로 결함을 증명하지 않는다. 사람이 반려한 2026-09-07 접근의 상세는 미확인이라 재사용 근거로 삼지 않는다.
- 차선 후보: `/admin/k8s/inventory`의 실효 상한·창 포화 표시 (가치 2 / 위험 1 / 작업량 S) — 1순위가 이미 구현됐다고 확인되는 경우에만, 같은 `admin_k8s.go:handleK8sInventory`와 `store/k8s.go:ListK8sInventory`의 기본값 200·상한 10000 계약을 대상으로 별도 계획을 갱신한다. 이벤트와 한 회차에 묶지 않는다.

범위와 확인 근거
- main@386f21b. `admin_k8s.go:471-488`의 핸들러, `:1159 intParam`, `store/k8s.go:674 ListK8sEvents`, `:1082 boundedLimit`, `server.go:474`의 실제 mux 등록을 읽었다. `admin_ui.go:10693`은 limit=20으로 이 API를 호출하므로 기존 events 배열을 그대로 남겨야 한다.
- `admin_k8s_test.go:TestK8sAdminFlowRegistersClusterAndIngestsSnapshot`의 배선(openTestStore + NewAsyncLogger + NewServer(testConfig) + httptest.NewServer(Routes))을 따라 테스트한다. `k8s_notify_window_test.go:warningEvent/insertEvents`는 실제 `store.K8sEvent`를 `InsertK8sEvent`로 넣는 재사용 가능한 fixture다. c1 501건과 c2 소수 행, 구별되는 last_seen을 사용하여 상한과 격리를 함께 확인한다. Mattermost webhook은 이 GET 테스트에 불필요하다.
- 현 결함은 소스 경로로 확인했고 신규 HTTP 재현 테스트는 정찰 역할상 작성하지 않았다. 구현자가 추가 필드 단언이 기존 코드에서 실패하는 것을 먼저 확인해야 한다. 실 PostgreSQL·브라우저 호환은 미확인이다.

접근 비교와 결정
- 선택: 메타데이터만 추가하고 작은 limit 헬퍼를 공유한다. 조회 비용/배열 계약은 유지하고 두 소비자가 같은 정규화를 쓴다. 가장 중요한 가정은 API 소비자가 추가 JSON 필드를 허용한다는 점이며 확인한 UI 소비자는 events를 그대로 사용할 수 있다.
- 정확한 total/has_more나 페이지네이션: 실제 누락을 정밀하게 알 수 있지만 새 SQL/API 계약·동시 갱신 의미를 다뤄야 하므로 이번 45분 범위에서 제외한다.
- 문서만 500 상한 안내하거나 현 상태 유지: 코드 비용은 낮지만 실제 응답별 창 포화를 식별할 수 없다. 상한을 높이는 안도 데이터 증가 시 재발하므로 선택하지 않는다.
- 기존 후보 중 알림 quiet 판정 이동은 오류 우선순위를 바꾸고 바로 전 회차와 같은 함수를 다시 건드린다. PSS/RBAC/마스킹 조정은 보안 신호나 정보 노출 경계가 변한다. 이보다 이벤트 API 한 경로의 부가 메타데이터가 위험이 낮다.

구현 순서 / 검증 / 체크포인트 (아래 구현 체크포인트 참조)
1. 위 신규 HTTP 테스트를 만들고 `go test ./internal/proxy -run K8sEvents -count=1`로 메타데이터가 없는 기존 코드에서 실패함을 확인한다. 의도한 실패만 재현되는 자동 체크포인트이며 별도 사람 승인은 필요 없다. 컴파일 실패는 재현 증거가 아니다.
2. 공유 limit 헬퍼와 핸들러 응답을 한 덩어리로 구현하고 `go test ./internal/proxy -run 'K8sEvents|NotifyScan' -count=1`를 통과시킨다. 다른 호출자 결과는 그대로인지 자동 확인 후 다음으로 진행한다.
3. 기존 운영 가이드 API 설명을 갱신하고 위 gofmt/build/vet/전체 test 게이트를 통과시킨다. 가정과 코드가 다르면 이 과제서를 먼저 갱신하고 범위를 확대하지 않는다. 단계 완료·검증 결과를 구현자가 이 파일에 기록한다.

작업 추정과 여유
- 방법: bottom-up. fixture/회귀 재현 8~10분 + 공유 헬퍼/응답 6~8분 + 문서 2~3분 + 검증/검토 7~9분 = 기본 23~30분. 알려진 변동(501행 SQLite fixture 비용·기존 게이트 수정 필요 여부)에 예비 5~10분을 별도로 두어 28~40분을 예상한다. 자신도는 중간이며 통계적 80% 보장 같은 실측 신뢰구간은 아니다.
- 이전 10-01 회차의 단일 응답 메타데이터 추가+실 DB 테스트 범위와 유사하나 실제 작업 시간 기록은 없어 시간 배율로 환산하지 않았다. 근거 제공자는 이번 정찰의 코드 확인과 사용자 제공 성공 기록이다.
- 관리 예비는 이번 자율 회차에 배정하지 않는다(0분). 45분을 넘길 것으로 보이면 새 기능/다른 엔드포인트를 더하지 말고 범위와 추정을 갱신한다. DB 마이그레이션·외부 서비스·릴리즈 변경은 추정 밖이다.

적용 스킬
- 전용 Skill 도구는 이 세션에 노출되지 않아 로컬 원문을 읽어 적용: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. PMO의 references/sources.md도 읽었으며 외부 기관의 추정 수치/확률을 인용하지 않았다.


구현 체크포인트 (2026-10-05)
- 1단계 완료: 실제 SQLite + NewServer + Server.Routes HTTP 테스트 2개(성공 창 18케이스, 오류 3케이스)를 먼저 추가. 기존 코드에서 `response is missing effective limit; requested "&limit=1000", store returned 500 rows`로 실패 확인(0.317s). 최초 오류 테스트는 Routes 미들웨어의 기존 코드가 authentication_required인 것을 확인해 기대값을 정정했고, 이후 실패는 메타데이터 부재뿐이었다.
- 2단계 완료: K8sEventQueryLimit을 스토어·핸들러가 공유하고 limit/window_full/조건부 window_notice 추가. 범위는 프로덕션 2파일, 테스트 1파일, 문서 1파일. `go test ./internal/proxy -run 'K8sEvents|NotifyScan' -count=1` 통과(2.309s).
- 인과 확인 완료: 응답 맵에서 limit/window_full만 잠시 제거했을 때 같은 HTTP 테스트가 다시 실패(0.181s); 즉시 복원 후 좁은 회귀 통과(2.314s).
- 3단계 완료: 기존 운영 가이드의 API 행 및 인접 설명 갱신. gofmt 출력 없음, go build ./... 및 go vet ./... 종료 0. go test ./... -count=1 전체 통과(실패 0, proxy 67.557s, store 16.214s). 커밋 0c79c53, 작업 트리 깨끗함. 실제 PostgreSQL·브라우저·외부 엄격 JSON 클라이언트 호환은 미검증; 릴리즈는 별도 역할.
