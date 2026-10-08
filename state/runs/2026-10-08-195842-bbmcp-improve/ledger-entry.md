## 2026-10-08
- 선택: PR 대상 브랜치 조회 실패 시 실행 도구의 브랜치 검사 생략을 막는다 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `internal/tools/executor.go`의 prTargetBranch가 오류를 전달하고 기존 접두사 정규화 결과가 비면 거부하여, RiskExecute PR 호출이 승인 검사 전에 고정 설명의 PERMISSION_UNKNOWN으로 종료되도록 했다. 기존 newFixture의 실제 Registry→Executor→Resolver→HTTP 클라이언트와 PostgreSQL로 503/빈 ref/접두사만 있는 ref에서 최초 승인 행 0개, 기존 승인 approved 유지·merge 미실행, 복구 시 BRANCH_RESTRICTED 후 같은 승인으로 merge 성공·consumed를 검증했으며 nil-current-version 방어는 comment 경로로 옮기고 merge 버전 증가·인자 변경·동시 소비 테스트를 보존했다. 수정 전 실패→수정 후 통과→executor만 원복 시 재실패→수정본 복구 후 최종 검증을 완료했고, 전용 DSN `postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable`의 TEST_DATABASE_URL로 `go test ./internal/tools -count=1 -p 1 -v`(24 PASS, 하위 테스트 포함) 및 `go test ./... -count=1 -p 1 -v`(9개 패키지, 131 PASS, SKIP/FAIL 0), `go build ./...`, `go vet ./...`가 성공했다; 커밋 `13b44b9`, 프로덕션 1개+테스트 1개 변경.
- 실패 재현: `integration_test.go:641: expected PERMISSION_UNKNOWN, got APPROVAL_REQUIRED` / `integration_test.go:648: approval count = 1, want 0` — 수정 전 첫 tools 전체 실행에서 새 테스트만 실패(tools-red.log). 확장 테스트에서도 승인된 빈 ref 두 경우 오류가 nil이었고, 503은 기존 APPROVAL_STALE 방어로 거부됐다(tools-expanded-red.log).
- 보류 아이디어: Resolver.Reset 후 캐시 권한 재조회 통합 테스트 (가치 4 / 위험 1 / 작업량 S) — 이번에는 대상 조회 결함 수정으로 범위 고정.
- 보류 아이디어: approval.redactArgs 중첩 map/slice 스크럽 (가치 3 / 위험 2 / 작업량 S) — 도구 계약·재귀 한도 확인 필요.
- 보류 아이디어: MCP 8MiB 초과 본문의 잘린 접두사 처리 거부 (가치 3 / 위험 2 / 작업량 S) — 정찰의 코드 추론이며 런타임 재현 미실행.
- 보류 아이디어: 승인 화면 긴 한국어 인자의 UTF-8 절단 경계 보존 (가치 3 / 위험 1 / 작업량 S) — 별도 승인 저장 경계 과제.
- 과제서: 채택 — 현재 코드와 503 최초 승인 행 생성 재현이 과제서와 일치하여 지정 범위와 수용 기준을 그대로 구현했다.
