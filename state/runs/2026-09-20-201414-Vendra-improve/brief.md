- 과제: 업무 관제탑의 결재 항목도 상신 당시 단계 스냅샷을 따르게 하기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `productivity.go:workInbox`는 현재 `workflow_definitions.steps`만 읽지만 승인함과 승인 처리는 `workflow_instances.context`의 상신 당시 단계를 우선하므로, 정의를 수정하면 같은 결재가 관제탑에서 사라지거나 다른 역할에게 표시될 수 있다. 기존 `instanceSteps`를 관제탑에서도 사용하면 결재할 사람과 단계 설명이 실제 승인 경로와 일치한다.
- 수용 기준:
  1) 두 단계로 상신해 두 번째 단계로 진행한 뒤 정의를 한 단계로 줄여도, 원래 담당자의 `GET /api/v1/me/work-inbox`에 `key=approval:<instanceId>` 항목이 남고 설명에 원래 두 번째 단계 이름이 나온다. 같은 사용자의 `GET /api/v1/approvals`의 `currentStepDefinition`과 일치해야 한다.
  2) 진행 중 단계의 역할·이름을 정의에서 다른 역할·이름으로 바꿔도, 원래 역할 사용자에게 두 조회 API가 같은 결재를 보여주고 새 역할만 가진 사용자에게는 둘 다 보여주지 않는다. `*` 관리자만으로 검사하면 역할 필터 오류를 놓치므로 일반 내부 사용자 두 명을 사용한다. 스냅샷 없는 구형 인스턴스는 기존 정의로 되돌아가는 동작을 유지한다.
  3) 실제 PostgreSQL과 `newTestApp(t)`의 `app.Handler()`를 통과하는 회귀 테스트가 수정 전 관제탑 누락/잘못된 역할 표시로 실패하고 수정 후 통과한다. 상신·정의 수정·승인 진행은 실제 API로 수행해 스냅샷 생성 배선까지 확인하고, 최종 승인 후 두 목록에서 사라지는 것도 확인한다. 구형 행에 한해서 SQL로 context를 비우는 준비를 허용한다. 응답 JSON을 파싱해 특정 ID를 검사하며 소스 문자열 검사를 증거로 삼지 않는다.
- 건드릴 파일:
  - `internal/httpapi/productivity.go:workInbox` — 승인 조회 SELECT에 `i.context` 추가, Scan에 대응 바이트 슬라이스 추가, 직접 `json.Unmarshal(steps, &definitions)`하던 곳을 `instanceSteps(snapshot, steps)`로 교체. 단계 범위 확인·역할 필터·설명 모두 반환된 같은 목록을 읽게 한다. 이 파일 다른 함수가 encoding/json을 사용하므로 import를 무작정 지우지 않는다.
  - `internal/httpapi/work_inbox_snapshot_integration_test.go` (신규 제안) — 위 API 통합 테스트. 기존 파일을 늘려도 되나 테스트 목적은 한 가지로 유지한다.
  - 읽기 참고(변경 불필요): `internal/httpapi/workflows.go:instanceSteps,listApprovals,workflowAction,updateWorkflow`, `internal/httpapi/objects.go:submitObject`, `internal/httpapi/login_integration_test.go:newTestApp,postLogin,TestEditingAWorkflowDoesNotStrandInFlightApprovals,TestWorkInboxDoesNotLoseWorkToTypesTheReaderCannotRead`, `internal/httpapi/workflowsteps_test.go`의 실제 상신·PATCH 예시, `internal/httpapi/workflow_snapshot_test.go`의 fallback 단위 테스트, `internal/httpapi/app.go` 라우팅.
- 검증 명령:
  - 저장소 루트: `go test ./internal/... ./cmd/... -count=1` (정찰에서 실제 통과: httpapi 1.444초. 세 DSN 모두 unset이어서 DB 통합 검증은 실행되지 않음.)
  - 구현 검증은 반드시 별도 PostgreSQL 테스트 DB를 준비하고 `VENDRA_TEST_DSN`을 지정한다. `VENDRA_TEST_MIGRATE_DSN`, `VENDRA_TEST_UPGRADE_DSN`은 각각 별도의 빈 DB를 지정한다. README와 `.github/workflows/ci.yml`에 같은 세 DB 구성이 있다. Docker daemon 29.7.2는 정찰에서 사용 가능함을 확인했다.
  - `go test ./internal/httpapi -run 'TestWorkInbox.*Snapshot|TestEditingAWorkflowDoesNotStrandInFlightApprovals|TestInstanceSteps|TestWorkInboxDoesNotLoseWorkToTypesTheReaderCannotRead' -count=1 -v` — 신규 테스트 이름은 `TestWorkInboxUsesSubmissionSnapshot` 등 정규식에 맞출 것. 출력에 SKIP이 없어야 한다.
  - 세 DSN을 설정한 뒤 `go test ./internal/... ./cmd/... -count=1`, `go vet ./internal/... ./cmd/...`, `gofmt -l internal cmd`. gofmt 출력은 비어야 한다. 프런트 변경이 없으므로 npm 설치·빌드는 이 과제에 필요 없다.
- 위험과 피할 것: auth/session, `internal/db/migrations`, `.github/workflows`, 통화·금액 정책은 건드리지 않는다. helper의 fallback 계약·승인 권한·데이터 스코프·자가결재 정책을 바꾸지 않는다. 별도 발견한 LIMIT 200 선적용 문제, 워크플로 이름 스냅샷, 은행계좌 결재 조회 문제는 이번 범위 밖이다. 설정을 바꾸면 원래 값을 보관하고 cleanup에서 복구하며, cleanup DB 작업에는 `context.Background()`를 쓴다. 생성한 데이터는 FK 역순으로 제거하고 공용 시드 역할·정의를 훼손하지 않는다. 새 비밀번호 리터럴을 추가하지 말고 기존 테스트 도우미/상수를 재사용한다.
- 차선 후보: 사용자 가이드 MCP 도구표를 실제 `tools/list` 응답과 대조하는 가드 (가치 2 / 위험 1 / 작업량 S) — 주 과제의 전제가 이미 다른 변경으로 해결된 때만 선택한다. `internal/httpapi/integrations.go:mcp`의 tools/list 분기와 `docs/USER_GUIDE.md` 4.6 표가 대상이며, 서버 소스 문자열 대신 실제 응답 JSON 이름 집합을 양방향 비교한다. 보호 경로 변경은 불필요하다.

구현 순서와 여유: 재현 fixture/API 회귀 테스트 15분 → 조회/Scan/helper 연결 5분 → 역할·구형 fallback·승인 완료 검증 15분 → 전체 검증/정리 여유 10분, 총 45분(M). 새 추상화나 SQL 정책 재작성 없이 기존 helper 재사용을 선택한다. 단순 d.steps 유지 방식은 승인함과 불일치를 남기며, SQL에서 JSON fallback을 새로 구현하면 같은 계약이 이중화된다.

확인 한계: SQL과 세 읽기 경로 차이는 직접 확인했으나 이번 정찰에서는 새 재현 테스트 작성 및 DB 실행을 하지 않았다. 구현자가 위 실패를 먼저 확인해야 한다. CLAUDE.md/AGENTS.md는 현재 체크아웃 검색에서 없었고, TODO/FIXME 검색 결과도 없었다. 지정된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 Skill/skills 도구가 노출되지 않았고 로컬 스킬 경로 검색에서도 찾지 못했다. 따라서 해당 스킬 절차/반환 형식 준수는 미확인이며, 위 추정·대안 비교·검증 계획은 프롬프트 기준으로 작성했다.
