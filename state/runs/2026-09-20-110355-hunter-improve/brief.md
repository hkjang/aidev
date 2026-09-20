# 과제서 2026-09-20 — hunter

- 과제: 넘기기 표(handoff claim) 발급에 사용자별 미사용 표 개수 상한(20개) 추가 (가치 3 / 위험 1 / 작업량 S)
- 왜: `POST /api/v1/handoff/claims` 는 권한 있는 사용자가 호출할 때마다 보고서 본문을 통째로 암호화해 `handoff_claims` 에 넣고, 정리는 "만료 행을 다음 발급 때 삭제" 뿐이라 한 사용자가 5분 안에 반복 호출하면 표(row)와 암호화 본문이 무한히 쌓인다(`internal/app/handoff.go:252-265` 트랜잭션에는 COUNT 검사가 없음 — 확인). 사용자당 살아 있는 표를 20개로 묶으면 실수·스크립트 반복 호출이 DB 를 부풀리지 못하고, 정상 사용(메뉴에서 한 번 클릭 → 표 1개 → 즉시 소비)은 전혀 달라지지 않는다.
- 수용 기준:
  1) 같은 사용자가 유효한(만료 전·미소비) 표를 20개 가진 상태에서 21번째 `POST /api/v1/handoff/claims` 는 `429` 와 한국어 메시지(예: "발급했지만 아직 쓰지 않은 표가 너무 많습니다. 잠시 후 다시 시도하세요")로 거절되고 행을 만들지 않는다(감사 기록 `agent.handoff` 도 남기지 않음).
  2) 표 하나를 `GET /api/v1/handoff/claims/{claim}` 으로 소비하거나 만료되면 다시 발급할 수 있다(정리 DELETE 와 COUNT 가 같은 트랜잭션 안에서 순서대로 실행).
  3) 상한은 **사용자별**이다 — 다른 사용자(테스트의 `handoff-other` 같은 계정이 읽을 수 있는 실행 기준)는 영향받지 않는다.
  4) 테스트가 증명할 것: 실제 HTTP 경로로 20개 발급(201) → 21번째 429 → 하나 소비(200) → 다시 201 → `handoff_claims` 의 해당 user_id 행 수가 20. 손으로 주입한 카운터가 아니라 실제 DB 행과 `testApp` 서버로 검증한다.
- 건드릴 파일:
  - `internal/app/handoff.go` — 상수 `handoffClaimTTL` 옆에 `handoffClaimsPerUser = 20` 추가. `issueHandoffClaim`(201행~)의 트랜잭션 안에서 `DELETE FROM handoff_claims WHERE expires_at<=now()` 직후 `SELECT count(*) FROM handoff_claims WHERE user_id=$1 AND expires_at>now()` 를 실행하고 상한 이상이면 `fail(w, 429, …)` 후 return(defer Rollback 이 정리). INSERT 는 그 뒤에. 동시 발급 경합은 이 상한의 목적(부풀림 억제)상 정확히 20 을 보장할 필요가 없으므로 `FOR UPDATE` 나 advisory lock 을 추가하지 말 것 — 주석에 "근사 상한" 이라고 적는다.
  - `internal/app/handoff_test.go` — `TestHandoffClaimsAreSingleUseBoundAndOffByDefault` 뒤에 새 함수 `TestHandoffClaimsPerUserCap` 을 추가(같은 헬퍼 `testApp`, `loginTest`, `reportFixture`, `mustRequest`, `str`, `digest` 사용). 기존 테스트는 손대지 않는다. 20회 반복은 루프로.
  - `internal/app/openapi.json` — `/api/v1/handoff/claims` POST 응답에 `429` 항목 한 줄 추가.
  - `docs/admin-guide.md`(§5.7 "다른 서비스로 보내기" — 파일명은 `docs/` 에서 `handoff` 로 grep 해 확인, 미확인) 한 문장: "사용자당 아직 쓰지 않은 표는 20개까지이며 초과 시 429". `docs/validation.md` 24행 표의 `handoff_test.go` 항목에 "사용자별 상한" 추가. 가이드 md 를 바꿨으면 `node scripts/render-guides.mjs` 로 html/pdf 재생성.
- 검증 명령:
  - `gofmt -l internal/app/handoff.go internal/app/handoff_test.go` (출력 없어야 함)
  - `go vet ./internal/app && go build ./cmd/hunter`
  - 임시 PostgreSQL(예: `docker run -d --rm -e POSTGRES_PASSWORD=pw -p 5433:5432 postgres:17`) 후
    `HUNTER_TEST_DSN='postgres://postgres:pw@localhost:5433/postgres?sslmode=disable' go test -race -count=1 -run 'TestHandoff' ./internal/app`
    — DSN 없으면 skip 되므로 skip 을 통과로 보고하지 말 것. **전체 스위트(650초+)는 돌리지 말 것**(예산 교훈).
  - 문서를 바꿨으면 `node scripts/check-docs.mjs`, 마지막에 `git diff --check`.
- 위험과 피할 것:
  - `redeemHandoffClaim`(274행~)·설정 검증·`handoffTargets`·프런트(`web/src`) 는 건드리지 않는다. 프런트 변경이 없으므로 `internal/webassets/dist` 재복사도 불필요.
  - 429 는 `agentReportSlots` 확보(231행) 뒤 `reportRun` 전에 넣으면 보고서 렌더 비용을 아끼지만, 그러면 "읽을 수 없는 실행은 404" 가 상한보다 먼저 판정되지 않는다 — 열거 위험은 없으므로 **트랜잭션 안(정리 DELETE 직후) 한 곳**에만 두어 순서 논쟁을 피한다.
  - 새 환경변수·설정 그룹 키를 만들지 않는다(상수로 충분). auth·migrations(`CREATE TABLE` 46행)·workflows 는 손대지 않는다. 기존 인덱스 `handoff_claims_expiry` 만으로 COUNT 는 충분하고 새 인덱스는 추가하지 않는다(사용자당 최대 20행).
  - 범위 밖 한 줄 수정(2026-09-19 비평에서 걸린 유형)을 섞지 않는다. 커밋은 검증 직후 **먼저** 만든다(배경 테스트 대기 중 유실 교훈).
- 차선 후보: `oidcReturnTo`(Go, `internal/app/auth_oidc*.go`)와 `safeReturnPath`(TS, `web/src/auth-flow.ts`)의 return_to 규칙을 공유 JSON 벡터 파일 하나로 양쪽 테스트에서 읽어 교차 검증 (3/1/M) — 코드 동작 변경 없이 테스트만 추가하므로 위험이 낮지만 벡터 파일 위치(Go 테스트가 `web/tests` 를 읽을지)를 먼저 정해야 함.
