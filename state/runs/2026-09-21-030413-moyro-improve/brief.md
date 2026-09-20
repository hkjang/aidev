- 과제: 사이드바 재정렬 PUT 응답·WebSocket 이벤트를 실제 저장된 전체 순서로 통일 (가치 3 / 위험 1 / 작업량 S)
- 왜: `updateSidebarCategoryOrder`는 `sidebar.UpdateOrder`가 중복을 제거하고 사용자·팀 밖의 ID를 무시한 뒤에도 입력 배열을 그대로 200 응답과 `sidebar_category_order_updated`에 실어 GET의 실제 순서와 다른 결과를 보낸다. 저장 후 `Order`로 조회한 동일 배열을 두 출력에 사용하면 부분 배열·중복·유령 ID 요청에서도 호출자와 다른 탭에 서버의 실제 순서를 전달할 수 있다.
- 수용 기준:
  1) `PUT /api/v4/users/{userID}/teams/{teamID}/channels/categories/order` 성공 후 `h.sidebar.Order(ctx, uid, teamID)`를 호출하고 그 반환값 하나를 JSON 응답과 이벤트 `data.order`에 함께 사용한다. 이벤트 이름·`data.team_id`·`Broadcast.UserID`·응답 배열 모양을 유지한다.
  2) 중복 ID, 없는 ID, 다른 사용자/팀 ID, 일부 카테고리 생략, 빈 배열을 다룬다. 응답에는 해당 사용자·팀의 전체 저장 카테고리가 한 번씩 나오고 후속 GET order와 일치한다. 타 소유자의 행은 바뀌지 않고, 생략된 행의 sort_order 유지·중복 첫 위치 우선이라는 기존 저장 규칙을 그대로 둔다.
  3) 저장 또는 되읽기 오류는 기존 `api.sidebar.order.app_error`의 500이며 성공 이벤트를 발행하지 않는다. 되읽기 실패 시 이미 끝난 UPDATE가 롤백된다고 주장하지 않는다. 기존 본문/개수 상한과 413/400 동작도 유지한다.
  4) 실제 PostgreSQL + sidebar.Service + 해당 PUT/GET 핸들러를 지나는 회귀 테스트로 응답·GET·DB 행을 비교한다. 실행 중인 실제 `ws.Hub`와 `ws.Client.Send`로 이벤트 JSON을 받아 응답과 배열이 정확히 같고 대상 사용자가 맞는지 검증한다. 이벤트를 따로 조립한 대역이나 소스 문자열 검사로 대신하지 않는다.
  5) 최소 한 회귀 사례(예: 유효 ID 두 개의 역순 + 중복 + 유령 ID + 생략된 기본 카테고리)는 변경 전 코드에서 실패해야 한다. 실패 경로는 실제 DB 오류로 검증한다. 되읽기만 실패시키려면 테스트 전용 DB 트리거 등으로 UPDATE 후 SELECT 시 오류가 나도록 하는 방법을 검토하되, 프로덕션 서비스에 테스트용 훅을 만들지 않는다. 시간 내 독립적인 되읽기 장애 재현이 어렵다면 미검증으로 명시하고 기본 저장 실패 500·이벤트 미발행은 반드시 증명한다.
- 건드릴 파일:
  - `server/internal/httpapi/compat_wave_handlers_early.go:updateSidebarCategoryOrder` — UpdateOrder 성공 직후 Order 조회·오류 처리; 같은 조회 결과를 응답·이벤트에 전달. 현재의 “one UPDATE per id” 주석은 이미 unnest 단일 UPDATE인 서비스에 맞춰 정정.
  - `server/internal/httpapi/sidebar_handlers_postgres_test.go` — 기존 `seedSidebarHandlerFixture`, `sidebarCategoriesRequest`를 참고해 재정렬용 요청과 실제 DB 회귀 테스트 추가(길면 별도 `sidebar_order_postgres_test.go`). `newOperationsTestDB`는 `native_operations_postgres_test.go`에 있으며 임의 스키마를 만들고 정리한다.
  - 읽기 참고: `server/internal/sidebar/service.go:UpdateOrder, Order, dedupeIDs, loadCategories` 및 `service_postgres_test.go:TestUpdateOrderOnlyTouchesTheCallersCategories`. 저장 로직 변경은 불필요하다.
  - 이벤트 검증 참고: `presence_scope_postgres_test.go:TestPresenceAndCustomStatusWebSocketAudiencePostgres`의 실제 hub.Run/Register/ClientCount 대기 패턴, `native_work_items_test.go:TestBroadcastWorkItemIntersectsRecipientsWithCurrentChannelMembership`의 Send 수신/JSON 디코딩 패턴. resolver 대역은 복사하지 않는다.
- 검증 명령:
  - 루트: `bash scripts/check-source-sizes.sh`
  - `server/`: `go test -count=1 ./internal/sidebar ./internal/httpapi` (정찰에서 통과. DSN 미설정이라 DB 테스트는 skip이며 통합 검증 증거가 아님.)
  - 구현 검증은 별도 테스트 PostgreSQL DSN을 `MOYRO_TEST_POSTGRES_DSN`에 설정한 뒤 `server/`에서 `go test -race -p 1 -count=1 ./internal/sidebar ./internal/httpapi`, 이어 `go vet ./...`. 최종 전체 게이트는 동일 DSN으로 `go test -race -p 1 ./...`.
  - DSN 준비가 필요하면 운영/다른 프로젝트 DB를 쓰지 말고 일회용 PostgreSQL 16을 사용한다. 현재 55432는 다른 프로젝트가 사용 중이다. 이번 정찰은 DB 컨테이너를 생성하지 않았고 DB 통합 실행 결과는 미확인이다.
- 위험과 피할 것: auth/session, migrations, workflows, ws 허브 구현, 웹앱, preferences·reminders의 미머지/반려 여부 미확인 접근은 건드리지 않는다. `Order`는 기본 카테고리를 초기화할 수 있으므로 새 사용자 빈 PUT도 기존 GET과 같은 결과를 기대한다. `sort_order, create_at` 동률은 완전한 정렬 키가 아니므로 회귀 fixture의 create_at을 서로 다르게 두고 동률 정렬 정책 변경은 별도 후보로 남긴다. 동시 요청 사이의 엄격한 스냅샷·이벤트 순서 보장은 이번 범위가 아니다. early.go는 53860/58000 bytes로 여유가 있고 대규모 분리는 불필요하다.
- 차선 후보: `compat_wave_handlers_final.go`의 `listChannelViews`, `createChannelView`, `listChannelTimezones` 세 곳에서 `IsMember` DB 오류를 500으로 분리 (가치 2 / 위험 1 / 작업량 S). 실제 비멤버 403·멤버 기존 응답·DB 오류 500을 실제 DB로 증명하며, 관리자 예외가 있는 나머지 두 함수와 auth 공통 헬퍼는 제외한다. 1순위가 이미 해결된 경우에만 전환한다.

실행 계획·추정: 근거 재확인/fixture 5분 → 핸들러 5분 → 응답·GET·실제 hub 회귀 15분 → 검증 10분 + 예비 10분 = 45분. 비교안은 입력 정규화만 하기(저장에 남은 생략 항목을 반영하지 못함), 서비스 API/트랜잭션 확장(이번 목적에는 과도함), 저장 후 기존 Order 재사용(선택)이다. DB 준비나 전체 테스트가 오래 걸리면 범위를 추가하지 말고 실행/미실행 결과를 구분해 남긴다.

정찰 근거와 한계: main@9acbddd, git log -30, CLAUDE.md·README·docs/roadmap.md·docs/development.md·docs/index.md·CI 및 관련 소스/테스트를 읽었다. 작업 트리 변경 없음. 요청된 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`은 현재 호출 가능한 Skill 도구/카탈로그 및 검색한 로컬 경로에서 찾지 못했으며 원문 절차·반환 형식 적용은 미확인이다. 위 계획과 추정은 읽지 못한 스킬을 적용했다고 주장하는 것이 아니라 이 과제의 자체 판단이다.
