# 회차 노트 2026-09-28-074216-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:42] base pinned — main@83657da
- [러너 07:42] autonomy release — 

## 구현 노트
- 무엇/왜: `internal/mcp/format.go` 에 `contentFence` 를 두고 `formatFileContent`·`formatSymbolContext` 가 함께 쓰게 했다. 저장소 파일 자체가 코드 펜스를 담고 있으면(파이썬 docstring 예제, 4백틱 안 3백틱 README) 답변이 연 블록이 내용 한가운데서 닫혀 `Source:`·`### Notes`·절단 공지가 파일 본문으로 읽혔다. commit 82ceb79 (프로덕션 파일 1개).
- 확신 없는 곳: (1) `closeOpenFence`(budget.go:111-119)는 여는 펜스 길이를 모른 채 항상 `"\n```"` 를 붙인다 — 4백틱 이상으로 감싼 답변이 **절단되는** 경우는 여전히 닫히지 않는다. 선존 결함(수정 전에도 read-file 은 4백틱을 냈다)이지만 이제 symbol-context 도 그 경로에 들어간다. 비평가가 여기를 먼저 보면 된다. 재현은 큰 read-file 응답 + 작은 maxBytes. (2) `formatRepositoryMap`(:350)의 고정 ```json 은 같은 결함이나 summary_json 에 백틱 세 개가 실제로 들어오는 색인 경로를 확인 못 해 뺐다.
- 일부러 안 한 것: `closeOpenFence` 는 줄 단위 스캔 재작성이 필요하고 기존 절단 테스트 3건을 지나므로 별도 회차로 미뤘다. `item.Documentation` 을 펜스 없이 산문으로 넣는 문제는 렌더 계약이 정해져 있지 않아 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트 `internal/mcp/fence_test.go` 는 공유 in-memory SQLite fixture 를 쓰므로 `t.Parallel()` 을 붙이면 안 되고, 고정 ID(`s9`,`c9`,`c10`)를 넣으므로 같은 ID 를 다른 테스트에서 재사용하면 충돌한다. `markdownBlocks`/`backtickRun` 은 테스트 전용 헬퍼다(프로덕션 아님). 검증 실측: `./internal/mcp` ok 0.864s, `-race` ok 4.163s, 전체 `go test -tags sqlite_fts5 ./...` 전 패키지 ok(internal/app 120.491s), gofmt/vet/build 및 `node --check web/app.js`·`test/web/*.test.js` 전부 exit 0. 외부 PostgreSQL·pgvector·Vault·Docker·실브라우저·릴리즈는 이 환경에서 미검증.
- [러너 08:04] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: `git checkout main -- internal/mcp/format.go` 로 되돌려 신규 테스트가 원장의 실패 재현과 같은 메시지로 FAIL 하는 것을 직접 봤고, 수정본에서 `./internal/mcp` ok 0.855s·vet·gofmt·`go build -tags sqlite_fts5 ./...` 모두 깨끗. 단언 3개가 실제 JSON-RPC 응답 텍스트를 보므로 무의미 단언이 아니다.
- 확인함: 구현자가 지목한 `closeOpenFence`(budget.go:111) 절단 공백은 **회귀가 아니다** — `contentFence`+`closeOpenFence` 를 복제한 독립 프로그램으로 3백틱/4백틱을 비교했고, 내용에 `` ``` `` 줄이 있는 응답이 절단되면 수정 전후 모두 펜스가 열린 채 남는다. 선존 결함이고 악화 없음. 가장 긴 연속이 정확히 3인 내용은 종전과 같이 4백틱이라 read-file 의 흔한 경우 출력이 안 바뀐다.
- 남는 우려(승인이어도): 릴리즈 노트에 "절단 공지가 더 이상 파일 본문으로 읽히지 않는다" 고 쓰면 사실과 다르다 — **절단되지 않은 응답**에 한정해 써야 한다. `contentFence` 는 펜스 길이 상한이 없어 백틱이 길게 이어지는 병적 파일은 펜스 한 줄이 예산을 먹을 수 있다(다음 회차 상한 권장).
- 못 본 것: 외부 PostgreSQL·pgvector·Vault·Docker·실브라우저·릴리즈, `-race` 와 전체 `./...`(구현자 실측만 신뢰), 실제 MCP 클라이언트의 5백틱 이상 렌더.
- 부서 소견: security·legal 모두 차단 없음. 펜스 탈출은 파일이 `Source:`·`### Notes` 틀을 위조하는 프롬프트 인젝션 면이었고 이 변경은 그것을 좁힌다. 이 PR 밖 위생 항목: main 에 이미 추적된 `c.txt`(쿠키 항목 0개)·`server.log` — 값 확인해 비밀값 없음, 차단 아님.
- [러너 08:12] review approved — 리뷰 승인 (risk=low)
- [러너 08:12] pr created — https://github.com/hkjang/git-ctx/pull/41
- [러너 08:19] ci passed — 검사 5개 모두 success
- [러너 08:19] merge done — 82ceb79
- [러너 08:37] release published — v0.77.20
- [러너 08:51] assets verified — v0.77.20 자산 2개 (이전 v0.77.19: 2)
