# 과제서 — 2026-10-06-225806-weekly-improve

- 과제: 임베딩 현황이 실패한 집계 질의를 "코퍼스가 비었다"로 답하지 않게 한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `embeddingStatus`(internal/app/semantic.go:431)가 세 개의 count 를 한 번에 읽는 `QueryRow(...).Scan(&result.Items,&result.Embedded,&result.Stale)` 의 오류를 `_ =` 로 버리므로, 질의가 실패하면 `Items/Embedded/Stale` 이 Go 의 제로값 그대로 남고 핸들러는 `writeData(w, 200, result)` 로 **`items:0, embedded:0, stale:0`** 을 내보냅니다 — 관리자 화면에서 "임베딩할 항목이 하나도 없다"(=할 일 없음)와 "집계를 읽지 못했다"가 글자 그대로 같아집니다. 같은 파일의 `pendingEmbeddingCount`(446행)도 오류를 버리고 `0` 을 돌려주는데, 이 값은 `rebuildEmbeddings`(485행)가 "남은 건수"로 화면에 쓰는 숫자라서, 질의가 실패하면 backlog 가 10만 건이어도 **"남은 것 없음"** 으로 보고합니다(그 함수의 주석이 바로 이 조용한 상한을 고치며 "it now reports what is left" 라고 적어 둔 자리입니다). 고치면 운영자가 "커버리지 0%"를 보고 임베딩을 다시 돌리는 헛수고 대신, 읽히지 않았다는 사실을 보게 됩니다.

- 수용 기준:
  1) `report_item_embeddings` 를 읽을 수 없는 상태에서 `GET /api/v1/admin/embeddings` 가 `items:0, embedded:0, stale:0` 을 **사실처럼** 내보내지 않는다. 응답은 집계를 읽지 못했다는 것을 기계가 읽을 수 있는 필드로 말한다(권장: 이 저장소의 기존 관례인 "읽지 못함" 플래그 — `rollup_handlers.go:227` 의 `view.IssueClearance.Unread = true` + `a.logger.Warn(...)`, 시험은 `issueclearance_test.go:169`. 예: `status` 구조체에 `CountsUnread bool \`json:"countsUnread"\`` 추가). **200 을 유지하는 쪽을 권장**합니다 — 500 으로 바꾸면 `vectorAvailable/enabled/model/reason` 까지 같이 사라져 화면이 "벡터 없음"과 구분되지 않고, 프런트가 이 카드를 어떻게 렌더하는지는 이번 정찰에서 **미확인**이기 때문입니다.
  2) 질의가 정상일 때의 응답 본문은 한 글자도 바뀌지 않는다(`vectorAvailable`,`enabled`,`model`,`items`,`embedded`,`stale`,`reason` 의 기존 값·이름·0 값 생략 규칙 그대로). 새 필드는 추가만 하고, 정상 경로에서는 `false` 다.
  3) `pendingEmbeddingCount` 가 질의 실패를 호출자에게 알리고, `rebuildEmbeddings`(semantic.go:485 `remaining := a.pendingEmbeddingCount(...)`)가 **읽지 못한 backlog 를 "0 건 남음"으로 보고하지 않는다**. 시그니처를 `(int, error)` 로 바꾸거나 `(int, bool)` 로 하되, `embeddingbacklog_test.go:156` 의 기존 호출(`server.app.pendingEmbeddingCount(server.ctx(), "rebuild-1")`)도 같이 고쳐야 합니다 — 그 시험은 `// guards: rebuildEmbeddings, pendingEmbeddingCount` 로 보호 대상을 선언해 둔 자리입니다.
  4) 시험은 손으로 만든 대역 없이 **프로덕션 `app.Handler()` + 실제 PostgreSQL**로 증명한다: 이 저장소에 이미 있는 기법대로 테이블 이름을 바꿔 질의만 실패시키고(`confluence_test.go:379 TestTheForceSyncRefusesAnUnreadableState` 가 같은 기법을 씀) 끝에 반드시 복원한다. 고치기 전 실패 → 고친 뒤 통과, 그리고 수정 한 줄을 되돌려 같은 실패가 다시 나는 것까지 확인한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/app/semantic.go:embeddingStatus` — 431행의 `_ =` 를 `if err := ...; err != nil` 로 바꾸고, 실패 시 집계 세 값을 응답에서 "읽지 못함"으로 표시(새 bool 필드) + `a.logger.Warn("embedding status counts", "error", err)`. 성공 시 동작은 그대로.
  - `internal/app/semantic.go:pendingEmbeddingCount` — 446행의 `_ =` 를 오류 반환으로 바꾸고, `rebuildEmbeddings`(485행)의 호출부에서 읽지 못한 경우를 "남은 건수 0"과 구분해 보고.
  - `internal/app/embeddingbacklog_test.go:156` — 시그니처 변경에 따른 호출 수정만(동작 단정은 그대로 유지).
  - 새 시험 1개 파일(예: `internal/app/embeddingstatusunread_test.go`) — 위 수용 기준 1·3. 파일 머리에 `// guards: embeddingStatus, pendingEmbeddingCount`.
  - 프런트: **먼저 `grep -rn "admin/embeddings" frontend/src` 로 이 응답을 실제로 렌더하는 곳이 있는지 확인**하세요. 이번 정찰은 `embeddingStatus`/`embedding/status` 로만 찾아 **프런트 소비 지점을 확인하지 못했습니다(미확인)**. 렌더하는 곳이 없으면 프런트는 한 줄도 바꾸지 말고 `npm` 검증도 돌리지 마세요. 있으면 새 플래그를 표시하는 최소 한 줄만 더하고, 이 저장소에 AdminPage 렌더 시험이 없다는 점을 근거로 **그 한 줄은 "타입 검사·빌드까지만 확인" 이라고 명시**하세요(타입 검사·grep 을 UI 행동 증거로 제출하지 말 것).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/app` (빈 출력), `go build ./...`, `go vet ./...`
  - DB 시험에는 DSN 이 필요합니다. 셸에 `WEEKLY_TEST_POSTGRES_DSN` 이 없고, **DSN 없이 돌린 SKIP 는 성공이 아닙니다.** 이전 회차들은 `weekly-test-pg`(pgvector/pgvector:pg16, 포트 15434)를 `docker start` 하고 `docker inspect` 로 자격을 읽어 **자식 프로세스 환경에만** DSN 을 구성했습니다(비밀번호를 출력·기록하지 마세요).
  - 지정: `go test ./internal/app -count=1 -run 'Embedding|Semantic' -v`
  - 전체: `go test ./... -count=1` (internal/app 약 150~185초)
  - 계약: `python3 scripts/openapi-check.py`(경로 119개), `python3 scripts/paging-check.py`(목록 10곳). **응답에 새 필드를 더하므로 `GET /api/v1/admin/embeddings` 의 OpenAPI 스키마를 같이 갱신해야 할 수 있습니다** — openapi-check 가 요구하는지 먼저 돌려서 확인하세요.
  - 커버리지: `python3 scripts/guard-check.py --changed 75542d2`
  - `git diff --check`

- 위험과 피할 것:
  - `a.capabilities.Vector` 가 false 면 431행 블록 자체가 실행되지 않습니다. 새 시험은 pgvector 가 있는 DB(위 `weekly-test-pg`/CI 이미지)에서만 의미가 있으니, 벡터 미지원이면 `t.Skip` 하고 그 사실을 로그로 남기세요 — 벡터 없는 환경의 통과를 증거로 내지 마세요.
  - 테이블 rename 은 **반드시 `t.Cleanup` 으로 복원**하세요. 시험이 중간에 실패해도 되돌아가야 합니다. `newTestServer` 가 scratch DB 를 만들지만 전역 state 디렉터리를 바꾸므로 `t.Parallel()` 은 금지입니다.
  - `migrations/`·`.github/workflows/`·`auth.go`·`crypto.go` 는 건드리지 마세요. 이번 과제와 무관합니다.
  - 범위를 넓히지 마세요: 같은 파일의 다른 `_ =` 나 `searchSemantic`·`rebuildEmbeddings` 의 배치 상한 로직은 이번 과제가 아닙니다. 프로덕션 파일은 `semantic.go` 하나로 끝내세요(필요하면 프런트 1파일까지).
  - 응답 필드 이름을 **기존 필드 rename 으로 처리하지 마세요**(프런트 소비 지점 미확인 → 이름을 바꾸면 조용히 깨집니다). 추가만 하세요.
  - `mutation-check`·`authz-check` 는 소스를 제자리에서 바꾸므로 커밋 직전이나 다른 빌드와 병행해 돌리지 마세요.
  - 공유 DB 오염으로 마이그레이션 수 시험이 환경적으로 흔들릴 수 있고, `TestTheOperatorSeesEveryQueueTheRelayTouches`(mail-queues)는 전체 실행에서만 흔들립니다 — 실패하면 단독 재실행으로 먼저 확인하세요.

- 차선 후보: `adminUsers` 의 역할 필터가 알 수 없는 값을 받으면 필터를 조용히 버려 전체 사용자를 돌려준다 (가치 2 / 위험 1 / 작업량 S) — `internal/app/admin.go:481` 의 `for _, role := range strings.Split(query.Get("role"), ",")` 가 `validRole(role)` 아닌 값을 모두 버리고, 남은 `roles` 가 비면 `role = ANY(...)` 조건을 아예 넣지 않습니다. 같은 핸들러 주석이 "검토 책임자 picker 가 리뷰어만 필요하다" 고 적어 둔 바로 그 용도에서, 오타 하나가 전체 디렉터리를 picker 에 붓습니다. 또한 `frontend/src/pages/AdminPage.tsx:204` 는 `role=TEAM_LEADER,ORG_MANAGER,ADMIN&limit=500` 으로 요청하는데 500 은 `userPageMaximum` 과 같은 값이라, 리뷰어가 500 명을 넘는 배포에서는 picker 가 조용히 잘립니다(응답의 `total` 을 picker 가 쓰는지는 **미확인**). 1순위가 성립하지 않으면 이쪽을 "알 수 없는 역할은 400 으로 거절한다" 한 가지로 좁혀 잡으세요.
