## 2026-10-03
- 선택: 승인 검증의 fail-open 두 곳 닫기 — PR 버전 확인 실패 시 통과, 단일 사용 승인의 동시 재사용 (가치 5 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `internal/approval/approval.go:Check` 에서 (a) 승인이 PR 버전을 고정했는데 현재 버전을 못 가져오면(`currentPRVersion == nil`) `ErrStale` 로 거부하도록 분기를 추가하고 — 승인 레코드는 `approved` 로 남겨 업스트림 복구 후 재시도 가능 — (b) 승인 소비를 `UPDATE … WHERE id=$1 AND status='approved'` 조건부 UPDATE + `RowsAffected()==0 → ErrStale` 로 바꿨다. 프로덕션 파일 1개만 건드렸고, 증거는 기존 프로덕션 배선 하네스(`internal/tools/integration_test.go`)에 `fakeBitbucket.prFetchFails` 를 더해 테스트 2건으로 잡았다(새 목 없음). 검증: 로컬 postgres:16-alpine 컨테이너를 띄우고 `TEST_DATABASE_URL=… go test ./... -count=1 -p 1 -race` 전체 통과(테스트 패키지 9개 ok, skip 없음), `go build ./... && go vet ./...` 통과, 동시성 테스트는 `-count=5 -race` 반복도 통과.
- 실패 재현: `integration_test.go:515: expected APPROVAL_STALE, got BITBUCKET_ERROR (BITBUCKET_ERROR: Bitbucket 503: service unavailable)` / `integration_test.go:568: 5 of 20 concurrent checks consumed the same approval, want exactly 1` — 추가로 1)번의 핵심 증거인 레코드 상태도 수정 전에 확인했다: `approval status = "consumed", want "approved"`.
- 보류 아이디어: `internal/mcp/jsonrpc.go` 프레이밍 단위 테스트(테스트 0건, DB 불필요) / `internal/identity/mapper.go` 매핑 고정 규칙 테스트 / `internal/approval/redactArgs` 스크럽 누락 키(key, credential, authorization, pat, apikey) 보강 / `internal/permission/resolver.go` 캐시·fail-closed 경로 테스트 / `Executor.prVersion`·`prTargetBranch` 의 오류 삼킴 정리(브랜치 제한 경로까지 영향, 별 회차)
- 과제서: 채택 — 과제서의 근거 2건이 현재 코드와 정확히 일치하고 수용 기준 3개를 그대로 충족했다.

- 릴리즈: v0.2.2 (2026-10-03, run 2026-10-03-070743-bbmcp-improve)
## 2026-10-04
- 선택: 승인 레코드의 인자 스크럽 누락 키 보강 (`internal/approval/redactArgs`) (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `redactArgs` 가 `token|secret|password` 세 조건문만 보던 것을 패키지 수준 `sensitiveKeyParts` 슬라이스 순회로 바꾸고 `credential`/`authorization`/`bearer`/`apikey`/`api_key`/`accesskey`/`access_key`/`passwd`/`privatekey`/`private_key` 를 추가했다 — 이 이름들로 들어온 값이 `approval_requests.arguments_redacted` 에 원문으로 저장돼 `GET /api/admin/approvals` 로 관리 콘솔에 나가던 경로를 닫았다. 감사 쪽 `redactKeys` 는 베끼지 않았다(승인자는 diff/content/text 본문을 봐야 하고, `pat` 부분일치는 실제 도구 인자 `path` 를 가린다). 프로덕션 파일 1개 + 테스트 1건. 검증은 docker postgres:16-alpine 을 띄우고 `go build ./... && go vet ./...` 통과, `TEST_DATABASE_URL=… go test ./... -count=1 -p 1` 전체 통과(테스트 패키지 9개 ok), `-v` 로 `--- SKIP` 0건 확인.
- 실패 재현: `integration_test.go:463: Arguments["apiKey"] = super-secret-pat, want "[redacted]"` / `integration_test.go:463: Arguments["authorization"] = Bearer super-secret-pat-3, want "[redacted]"` (수정 전, 8개 키 중 7개가 DB 를 거쳐 원문으로 돌아왔다 — `bearerToken` 만 기존 `token` 조건에 걸려 통과)
- 보류 아이디어: `internal/mcp/jsonrpc.go` 프레이밍 단위 테스트(테스트 0건, DB 불필요) / `internal/identity/mapper.go` 매핑 고정 규칙 테스트 / `internal/permission/resolver.go` 캐시·fail-closed 경로 테스트 / `Executor.prVersion`·`prTargetBranch` 의 오류 삼킴 정리(브랜치 제한 경로까지 영향, 별 회차) / `internal/audit` 의 `pat` 부분일치가 `path` 를 과다 스크럽
- 과제서: 채택 — 근거(`redactArgs` 3개 하드코딩 조건, 미필터 args map, `scan()`→`GET /api/admin/approvals` 노출)가 현재 코드와 정확히 일치했고 수용 기준 3개를 그대로 충족했다.

- 릴리즈: v0.2.3 (2026-10-04, run 2026-10-04-132233-bbmcp-improve)
## 2026-10-05
- 선택: MCP `initialize` 응답에 `Mcp-Session-Id` 헤더를 실어 세션 추적을 살린다 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `internal/mcp/server.go` 의 `dispatch` 에 `http.ResponseWriter` 를 넘겨(`dispatch(w, r, req)`; 단건 `:132`·배치 `:115` 두 호출부) `initialize` 가 `openSession` 에서 받은 세션 UUID 를 `w.Header().Set(SessionHeader, sessionID)` 로 내보내게 했다 — 저장소 전체에 `w.Header().Set(SessionHeader, …)` 이 0건이어서 클라이언트가 세션 id 를 알 길이 없었고, 그 결과 `touchSession`/`closeSession` 이 한 번도 돌지 않아 `last_seen_at` 은 INSERT 기본값에 머물고 `closed_at` 은 영구 NULL 이었다(관리 콘솔 활성 세션 지표가 두 방향으로 틀림). 프로덕션 파일 1개 + 기존 e2e 하네스(`mcpoauth_e2e_test.go` `newGateway`) 확장으로 테스트 2건. 새 목은 만들지 않았고 증거는 실제 HTTP 왕복과 실제 `mcp_sessions` 행이다(`gateway` 에 `pool` 필드를 노출해 `created_at`/`last_seen_at`/`closed_at` 직접 SELECT). 검증: docker postgres:16-alpine 을 띄우고 `go build ./... && go vet ./...` 통과, `TEST_DATABASE_URL=… go test ./... -count=1 -p 1 -v` 에서 `--- PASS` 89건 / `--- SKIP` 0건 / `--- FAIL` 0건, 새 테스트는 `-count=3 -race` 반복도 통과.
- 실패 재현: `mcpoauth_e2e_test.go:463: initialize 응답에 Mcp-Session-Id 헤더가 없습니다 (헤더: map[Content-Length:[986] Content-Type:[application/json; charset=utf-8] … X-Frame-Options:[SAMEORIGIN]])` / `--- FAIL: TestMCPSessionIDIsIssuedAndTracked (0.28s)` — 수정 전 헤더 목록에 `Mcp-Session-Id` 가 아예 없어 1)에서 멈췄고 2)의 touch/close 단정에는 도달조차 못 했다. 기준 3) 테스트(`TestMCPSessionIDNotIssuedWithoutAuth`)는 수정 전에도 PASS 였다 — 과제서가 예측한 대로 `initialize` 가 인증 실패 시 `openSession` 전에 반환하므로 결함이 아니라 회귀 방지 가드다. 이 사실을 숨기지 않고 적는다.
- 보류 아이디어: `tools/call` 응답 KB 절단이 UTF-8 문자 중간을 자른다(`server.go:310` `string(text[:limit*1024])`, 차선 후보였음) / `handlePost` 의 401 판정이 한국어 메시지 substring(`strings.Contains(…, "인증")`)에 의존 / `internal/identity/mapper.go` 매핑 고정·재바인딩 거부 테스트(이제 `newGateway` 하네스로 가능) / `internal/permission/resolver.go` 캐시·fail-closed 경로 테스트 / `approval.redactArgs` 의 중첩 map/slice 미스크럽(`server.go:279` 로 중첩 맵 경로가 열려 있음)
- 과제서: 채택 — 근거 3건(`SessionHeader` 참조 3곳 전부 읽기 전용, `gateway` 에 pool 필드 없음, `initialize` 의 인증 선행 반환)이 현재 코드와 정확히 일치했고 수용 기준 3개를 그대로 충족했다.

- 릴리즈: v0.2.4 (2026-10-05, run 2026-10-05-100729-bbmcp-improve)
## 2026-10-06
- 선택: `tools/call` 응답 KB 절단이 UTF-8 문자 중간을 자른다 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `internal/mcp/server.go:callTool` 의 응답 상한이 `string(text[:limit*1024])` 로 바이트 오프셋에서 잘라 3바이트 한국어 문자가 세 번 중 두 번 반쪽만 남았고, `json.Marshal` 이 그 깨진 바이트를 `�` 로 바꿔 클라이언트가 손상된 마지막 글자를 받았다. `trimToRune` (`utf8.DecodeLastRune` 역방향 트림) 으로 잘린 뒤쪽의 불완전한 시퀀스를 떼어냈다 — `bitbucket.Client.File` 이 이미 줄 경계에서 자르는 것과 같은 원칙. 프로덕션 파일 1개(+17/-2), 테스트 파일 1개. 증거는 새 목이 아니라 기존 프로덕션 배선 하네스(`internal/api/mcpoauth_e2e_test.go` `newGateway`) 의 실제 HTTP 왕복이다: `MaxResponseKB=1` 로 두고 6KB 한국어 설명을 가진 프로젝트를 `bitbucket_projects` 로 읽으며, 1바이트씩 밀린 패딩 3종(`""`/`"A"`/`"AA"`)으로 오프셋 1024 의 mod 3 잉여류를 모두 덮어 "운 좋게 경계에 맞아 통과" 를 배제했다. 검증: docker `postgres:16-alpine`(기존 `bbmcp-test-pg`, 포트 15532) 을 띄우고 `go build ./... && go vet ./...` 통과, `TEST_DATABASE_URL=… go test ./... -count=1 -p 1 -v` 에서 `--- PASS` 90건 / `--- SKIP` 0건 / `--- FAIL` 0건(패키지 9개 ok), 새 테스트는 `-count=3 -race` 반복도 통과. 웹(`web/`)은 파일을 전혀 건드리지 않아 `npm run check/build` 는 돌리지 않았다.
- 실패 재현: `mcpoauth_e2e_test.go:616: pad 0B: 절단된 응답 1023번째 바이트에 U+FFFD 가 있습니다 — 문자 중간에서 잘렸습니다 (…"한한한한�\n\n… 응"…)` / `mcpoauth_e2e_test.go:616: pad 2B: 절단된 응답 1022번째 바이트에 U+FFFD 가 있습니다 — 문자 중간에서 잘렸습니다 (…"한한한한��\n\n… "…)` — 수정 전 3종 중 2종이 실패하고 `pad 1B` 만 우연히 깨끗한 경계에 맞아 통과했다(예측한 대로 mod 3 중 하나는 결함이 드러나지 않는다). 이 사실을 숨기지 않고 적는다.
- 보류 아이디어: `internal/identity/mapper.go` 매핑 고정·재바인딩 거부 테스트(`newGateway` + `keycloakStub.token(t, mutate)` 로 가능) / `internal/permission/resolver.go` 캐시·fail-closed 경로 테스트(`newFixture` 가 `CacheTTLSec=0` 으로 캐시를 끄고 돌아 미검증) / `handlePost` 의 401 판정이 한국어 메시지 substring(`strings.Contains(…, "인증")`)에 의존 / `approval.redactArgs` 의 중첩 map/slice 미스크럽(`server.go:285` 로 경로가 열려 있음) / `Executor.prVersion`·`prTargetBranch` 의 오류 삼킴 정리(브랜치 제한 경로까지 의미가 바뀌므로 단독 회차)
- 과제서: 기각 — 이번 회차에는 정찰 과제서가 없었고(회차 노트에 러너의 base pin 2줄만 있음), 보류 목록의 1순위 항목을 현재 코드로 재확인해 그대로 골랐다.

- 릴리즈: v0.2.5 (2026-10-06, run 2026-10-06-033747-bbmcp-improve)
