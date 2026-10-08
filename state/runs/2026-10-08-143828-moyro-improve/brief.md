- 과제: 실제 WebSocket 연결·해제가 수동 DND/away 상태를 보존하는 통합 회귀 테스트 (가치 3 / 위험 1 / 작업량 M)
- 왜: 수동 상태를 보존하는 `userstatus.Service.SetAuto`와 `NewRouter`의 연결·해제 콜백은 존재하지만, 기존 presence 테스트는 직접 만든 핸들러와 `hub.Register`를 사용하여 이 배선을 통과하지 않는다. 실제 인증·라우터·WebSocket·PostgreSQL을 연결한 테스트로 재접속 때 사용자의 상태 선택이 사라지는 회귀를 막는다.
- 수용 기준:
  1) 실제 HTTP `PUT /api/v4/users/{userID}/status`로 설정한 `dnd/manual:true`와 `away/manual:true`가 각각 실제 WebSocket 연결 → 해제 → 재연결 뒤에도 유지된다. 각 단계에서 관찰자의 `status_change` 이벤트를 받은 뒤 대상 사용자 자신의 GET 응답과 DB의 `status, manual`을 확인한다.
  2) `offline/manual:false`로 시작하는 양성 대조 사용자는 실제 연결 때 `online/manual:false`, 그 사용자의 유일한 소켓 해제 때 `offline/manual:false`가 된다. 이벤트의 `data.status` 및 JSON 문자열인 `data.payload` 내부 상태가 GET/DB와 일치한다. 이 대조는 수동 상태 테스트가 콜백 미실행 때문에 거짓 통과하는 것을 막는다.
  3) 테스트는 `NewRouter`, 실제 `auth.Register/Login`, `httptest.NewServer`, `gorilla/websocket`, 실제 PostgreSQL 격리 스키마를 사용한다. `OnConnect/OnDisconnect` 직접 호출·대체, 테스트 인증 미들웨어, 손으로 만든 `ws.Client`, 소스 문자열 검사는 사용하지 않는다.
  4) 콜백 완료를 `ClientCount`나 고정 sleep으로 추측하지 않는다. 관찰자 소켓의 이벤트를 한 reader에서 읽고, 대상 user_id와 event 종류를 구분하며 읽기 deadline을 둔다. PUT에서 생긴 이벤트를 먼저 소비한 다음 연결 이벤트를 기다려 이전 이벤트를 새 콜백의 증거로 오인하지 않는다.
  5) 새 테스트가 실제 DB에서 skip 없이 통과하고 race 검사도 통과한다. 프로덕션 코드 변경은 0파일이다. 현재 결함이 증명된 과제가 아니라 테스트 공백 보강이므로 수정 전 RED를 억지로 만들지 않는다.
- 건드릴 파일:
  - `server/internal/httpapi/presence_lifecycle_postgres_test.go` (신규): `TestPresenceSocketLifecyclePreservesManualStatus` 및 파일 내부 fixture/event/assertion 헬퍼. 변경은 이 테스트 1파일로 제한한다.
  - 읽기만 할 근거: `server/internal/httpapi/router.go:80`의 실제 audience resolver, `:301-334`의 `OnConnect/OnDisconnect`, `:476-480`의 상태 API, `:1530`의 WebSocket 등록.
  - 읽기만 할 근거: `server/internal/userstatus/service.go:Service.SetAuto`는 `WHERE user_statuses.manual = false`로 수동 상태를 보호하고 `Get`으로 현재 상태를 반환한다. `server/internal/ws/hub.go:Hub.Run`은 첫 연결·마지막 해제에서 비동기 콜백을 실행한다.
  - 배선 선례: `server/internal/httpapi/invite_email_route_postgres_test.go:TestInviteByEmailRouteKeepsTeamAdminGate`의 config/secrets/auth/pluginhost/httptest 구성과 `loginForInviteTest`; `native_operations_postgres_test.go:newOperationsTestDB`의 격리 스키마를 재사용한다. 공통 헬퍼 추출 리팩터는 하지 않는다.
  - 기존 커버리지 경계: `presence_scope_postgres_test.go:TestPresenceAndCustomStatusWebSocketAudiencePostgres`는 직접 `handlers`와 가짜 소켓 client를 만들고 PUT 핸들러를 호출한다. 이번 과제와 중복되지 않는다.
- 구현 순서와 추정(45분):
  - 10분: 위 NewRouter 선례로 fixture를 구성하고 실제 로그인한 일반 사용자 관찰자 1명과 대상 사용자를 만든다. NewRouter가 콜백을 설정한 뒤 hub.Run을 시작한다. 관찰자는 시나리오 동안 연결을 유지한다.
  - 15분: `handlers.go:websocket`이 지원하는 `Authorization: Bearer ...` 헤더로 `/api/v4/websocket`에 Dial한다. URL query에 토큰을 넣지 않는다. 일반 활성 관찰자는 `ws/audience.go:DatabaseAudienceResolver`의 subject 범위 이벤트를 볼 수 있으므로 팀/채널/게스트 fixture는 필요 없다. 관찰자 자신의 최초 online 이벤트를 먼저 처리한다.
  - 10분: 수동 DND/away와 자동 상태의 표 기반 시나리오를 순차 실행하고 PUT 이벤트 → connect 이벤트 → disconnect 이벤트를 분리한다. 이전 콜백 완료를 확인한 뒤 다음 소켓을 열어 빠른 재접속 경쟁을 이 과제로 끌어들이지 않는다. 마지막 정리 때 소켓·reader·hub goroutine을 종료한 뒤 DB를 닫도록 cleanup 순서를 관리한다.
  - 5분: 아래 focused/race 명령 및 diff 확인. 5분 예비: 동기화·정리 문제 해결. 다중 탭·급속 재접속·custom status·guest 가시성까지 늘리지 않는다.
- 검증 명령:
  - 정찰에서 실제 실행, exit 0: `cd server && go test -count=1 ./internal/httpapi ./internal/userstatus ./internal/bookmarks ./internal/ws` (httpapi 0.167s, ws 0.033s; userstatus/bookmarks는 [no test files], DSN 없는 DB 테스트는 통과가 아니라 skip).
  - 정찰에서 실제 DSN으로 실행, skip 0 / exit 0: `cd server && go test -count=1 -v ./internal/httpapi -run 'TestInviteByEmailRouteKeepsTeamAdminGate|TestPresenceAndCustomStatusWebSocketAudiencePostgres'` (3.161s). 기존 NewRouter·presence fixture가 이 환경에서 작동함을 확인했다.
  - 구현 후 실행: `cd server && go test -race -count=1 -v ./internal/httpapi -run 'TestPresenceSocketLifecyclePreservesManualStatus|TestPresenceAndCustomStatusWebSocketAudiencePostgres|TestInviteByEmailRouteKeepsTeamAdminGate'`.
  - 이어서 저장소 DB 회귀: `cd server && go test -race -p 1 -count=1 ./...` (수 분; 이번 정찰에서는 미실행). 두 DB 명령 모두 `MOYRO_TEST_POSTGRES_DSN`을 설정하고 새 테스트의 PASS/skip 0을 직접 확인한다.
  - 루트 `bash scripts/check-source-sizes.sh`는 정찰에서 exit 0. 구현 후 `gofmt -l server/internal/httpapi/presence_lifecycle_postgres_test.go`와 `git diff --check`도 확인한다.
  - 환경: 현재 `moyro-pg-improve` / 호스트 55433이 실행 중이며 위 실제 DB 테스트로 접속을 확인했다. 컨테이너 환경에서 자격증명을 읽어 자식 go 프로세스 환경에만 전달했고 비밀은 출력하지 않았다. DSN을 문서·로그에 복사하지 않는다.
- 위험과 피할 것: auth/session, migrations, workflows, 상태 API, audience 정책, hub 구현은 수정하지 않는다. 09-07~09-09/10-06의 presence 가시성·사이드바 개선을 다시 구현하는 과제가 아니다. `Hub.Run`의 콜백은 goroutine이므로 소켓 등록 수만으로 완료를 판단하면 거짓 통과·플레이크가 생긴다. 정상 코드로 새 시나리오를 실행한 결과는 아직 미확인이다. 실패가 실제 제품 결함으로 밝혀지면 증거를 기록하고 별도 아이디어로 남기며 이 테스트 과제에 운영 코드 수리를 섞지 않는다.
- 차선 후보: `newOperationsTestDB` 주석에 격리 스키마 장애 주입 관용구 문서화 (가치 2 / 위험 1 / S) — 구현 시 동등한 lifecycle 통합 테스트가 이미 있음을 발견하거나 예비 시간 안에 결정론적인 동기화를 확보하지 못할 때만 선택. 프로덕션 0파일, `native_operations_postgres_test.go`의 헬퍼 주석만 변경하고 `file_search_rows_fault_postgres_test.go`의 RENAME+VIEW/ALTER TYPE 및 `incoming_webhook_member_errors_postgres_test.go`의 DROP TABLE 사례를 정확한 테스트 이름과 연결한다. DB 연결 자체가 불가능해도 테스트를 무조건 skip시켜 완료로 보고하지 않는다.

진행 제약: 요청한 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`은 현재 호출 가능한 Skill 도구/리소스와 로컬 검색에서 발견하지 못했다. 해당 스킬의 반환 형식을 준수했다고 주장하지 않는다. 대신 사용자 지정 형식 안에 대안 비교, 작업 분해, 45분 추정 및 예비·차선 계획을 명시했다.
