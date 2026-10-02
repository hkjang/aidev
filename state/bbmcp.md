## 2026-10-03
- 선택: 승인 검증의 fail-open 두 곳 닫기 — PR 버전 확인 실패 시 통과, 단일 사용 승인의 동시 재사용 (가치 5 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `internal/approval/approval.go:Check` 에서 (a) 승인이 PR 버전을 고정했는데 현재 버전을 못 가져오면(`currentPRVersion == nil`) `ErrStale` 로 거부하도록 분기를 추가하고 — 승인 레코드는 `approved` 로 남겨 업스트림 복구 후 재시도 가능 — (b) 승인 소비를 `UPDATE … WHERE id=$1 AND status='approved'` 조건부 UPDATE + `RowsAffected()==0 → ErrStale` 로 바꿨다. 프로덕션 파일 1개만 건드렸고, 증거는 기존 프로덕션 배선 하네스(`internal/tools/integration_test.go`)에 `fakeBitbucket.prFetchFails` 를 더해 테스트 2건으로 잡았다(새 목 없음). 검증: 로컬 postgres:16-alpine 컨테이너를 띄우고 `TEST_DATABASE_URL=… go test ./... -count=1 -p 1 -race` 전체 통과(테스트 패키지 9개 ok, skip 없음), `go build ./... && go vet ./...` 통과, 동시성 테스트는 `-count=5 -race` 반복도 통과.
- 실패 재현: `integration_test.go:515: expected APPROVAL_STALE, got BITBUCKET_ERROR (BITBUCKET_ERROR: Bitbucket 503: service unavailable)` / `integration_test.go:568: 5 of 20 concurrent checks consumed the same approval, want exactly 1` — 추가로 1)번의 핵심 증거인 레코드 상태도 수정 전에 확인했다: `approval status = "consumed", want "approved"`.
- 보류 아이디어: `internal/mcp/jsonrpc.go` 프레이밍 단위 테스트(테스트 0건, DB 불필요) / `internal/identity/mapper.go` 매핑 고정 규칙 테스트 / `internal/approval/redactArgs` 스크럽 누락 키(key, credential, authorization, pat, apikey) 보강 / `internal/permission/resolver.go` 캐시·fail-closed 경로 테스트 / `Executor.prVersion`·`prTargetBranch` 의 오류 삼킴 정리(브랜치 제한 경로까지 영향, 별 회차)
- 과제서: 채택 — 과제서의 근거 2건이 현재 코드와 정확히 일치하고 수용 기준 3개를 그대로 충족했다.

- 릴리즈: v0.2.2 (2026-10-03, run 2026-10-03-070743-bbmcp-improve)
