# PR 처리기 노트 2026-10-03-085759-git-ctx-shepherd — git-ctx PR #44
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-03-074735-git-ctx-improve)
# 회차 노트 2026-10-03-074735-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:47] base pinned — main@04a1db6
- [러너 07:47] autonomy release — 

## 구현 노트
- 무엇을/왜: `internal/mcp/budget.go` 의 `sectionCount`·`cutAtBoundary` 가 펜스 안 내용(read-file·get-symbol-context 가 감싼 파일/심볼 본문)의 `### `·`- ` 를 답변 구조로 읽어, 예산의 1/3을 버리고(12,000B 요청 → 8,104B) `mcp_calls.result_count` 에 문서 제목 수를 기록했다. `closeOpenFence` 의 줄 단위 스캐너를 `lineFence` 로 떼어내 공유했다. 프로덕션 1파일.
- 확신 없는 곳: (1) 코드 검색 `#### ` 히트 집계를 일부러 펜스 비인식으로 **남겼다** — 펜스 인식/비인식 중 어느 쪽이 맞는지는 `formatCodeSearch` 가 스니펫을 펜스로 감싸지 않는다는 현 설계에 달려 있다. 비평가는 이 비대칭을 먼저 보시길(근거는 `sectionCount` 주석과 대조군 테스트). (2) `clampResponse` 의 `LastIndex("\n### Notes\n")` 는 같은 원인인데 재현 못 해 손대지 않았다. (3) 펜스가 걸친 채 끊긴 청크가 코드 검색 답변의 마크다운을 깨는 것(상류 원인)은 그대로다.
- 일부러 하지 않은 것: `format.go` 는 한 줄도 안 건드렸다(포매터를 바꾸면 집계 계약이 같이 움직인다). `c.txt`·`server.log` 위생 작업도 뺐다(범위 분리).
- 기존 테스트 수정 1건: `result_count_test.go:200` 의 "변경 없는 `### ` 카운트" 단언은 2026-10-01 회차의 영향범위 고정용이었고 이번 계약 변경과 정면으로 충돌해, 같은 테스트 안에서 값을 1로 못박는 새 단언으로 바꿨다(약화 아님). 이것이 이번 diff 에서 설명이 필요한 유일한 테스트 변경이다.
- 다음 역할 주의: 새 테스트는 전부 실제 SQLite fixture + `Server.ServeHTTP` 왕복이라 외부 DB 는 필요 없고, 공유 in-memory SQLite 이름을 쓰므로 `t.Parallel()` 을 붙이면 안 된다. 릴리즈는 하지 않았다.
- [러너 08:03] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- reject. 구현자가 먼저 보라고 한 비대칭이 바로 결함이었다: `### ` 규칙을 전역으로 펜스 인식하게 만들었지만 `formatSemanticSearch`(format.go:257)·`formatRunbooks`(format.go:485) 는 청크 내용을 **펜스 없이** `### ` 아래 생prose 로 쓴다. 따라서 budget.go:79-83 의 "이 규칙이 세는 포매터만 펜스를 쓴다" 는 전제가 사실이 아니다. 수리는 이 두 포매터부터 볼 것.
- 실제 `Server.ServeHTTP` 왕복으로 재현(검증용 테스트는 지웠다): 펜스가 안 닫힌 청크 하나가 섞이면 `search-semantic` 답변의 `### ` 9개가 `result_count=1` 로, `find-runbook` 7개가 2로 기록된다(main 은 9/7). `maxBytes:2500` 에선 4,686B→2,239B 로 잘리면서 공지가 "1 of 1 result sections are included" 라 거짓말을 하고, 컷이 히트 중간(`### …docs/zf4.md` 아래 스니펫·Source 없음)에 떨어진 뒤 ``` 하나가 붙는다 — budget.go:188-192 주석이 막겠다고 한 바로 그 실패.
- 전제는 가설이 아니다: `parse`(internal/indexer/indexer.go:1110-1131) 가 펜스 상태 없이 `^#{1,6}\s+` 로 쪼개므로 ```bash 예제 안의 `# comment` 줄에서 블록이 갈려 펜스 불균형 청크가 일상적으로 생긴다. 런북·셋업 문서가 정확히 그 모양이다.
- 확인했으나 문제없음: `lineFence` 추출은 분기별로 대조해 동작 동일, `unfencedLines`/`lastUnfenced` 는 빈 입력·말미 개행·오프셋 의미까지 기존 `LastIndex("\n### ")` 와 일치. 보안·법무 차단 사유 없음(인증·ACL·마이그레이션·비밀값·개인정보 무관, 의존성 추가 없음). 못 본 것: 전체 `./...`·`-race`, `clampResponse` 의 `LastIndex("\n### Notes\n")`(같은 펜스 맹점, 여전히 미재현).
- 승인했어도 남을 우려로 기록: 릴리즈 노트는 "예산 절단이 내용의 마크다운을 구조로 읽지 않는다" 를 read-file·get-symbol-context 로 한정해 써야 한다. 상류(`parse` 의 펜스 무인식 분할)는 범위 밖이지만 이 계열 버그의 뿌리이고, 고치면 재색인 지문이 바뀐다.
- [러너 08:10] review rejected — 리뷰 거절: internal/mcp/budget.go:90 `countUnfenced(text, "### ")` regresses `search-semantic` and `find-runbook`. Both formatters write indexed chunk content as *unfenced prose* under `### ` head

## 수리 노트
- 맞았던 지적: 1~4 전부. `formatSemanticSearch`·`formatRunbooks`·`formatChangeRequests` 는 청크를 생prose 로 쓰므로 `### ` 규칙을 전역 펜스 인식으로 만든 것이 회귀였다. 실제 왕복으로 재현: semantic 12→1, runbook 9→1, 컷이 스니펫·`Source:` 없는 헤딩에서 끊김, 공지 "1 of 1". 틀린 지적은 없었다.
- 고친 방법: 추측을 없앴다. `fencesContent(tool)` 이 `toolcatalog.ReadFile`·`GetSymbolContext` 만 참 — `contentFence` 로 내용을 감싸는 유일한 두 포매터다. 나머지는 main 과 같은 `strings.Count`/`LastIndex`. `sectionCount`·`cutAtBoundary`·`clampResponse` 가 `tool` 을 받고 dispatch.go:375 가 넘긴다. 마크다운 스니핑이 아니라 호출된 도구 이름을 쓰므로 포매터/소비자 계약이 한 단계 느슨해졌다.
- 테스트: `unfenced_content_test.go` 신규(생prose 포매터 2개의 감사 카운트 + 잘린 semantic 답변의 히트 완결성·공지 두 숫자). `fencesContent` 를 `return true` 로 되돌려 네 지적이 모두 재현되는지 확인한 뒤 복원했다.
- 여전히 확신 없는 곳: (1) 불균형 펜스가 섞인 답변은 그 아래 히트들이 마크다운상 코드블록으로 렌더되고 `closeOpenFence` 가 끝에 ``` 를 붙인다 — main 과 동일한 상류(`parse` 의 펜스 무인식 분할) 문제로 범위 밖. (2) `clampResponse` 의 `LastIndex("\n### Notes\n")` 는 여전히 펜스 비인식(미재현, 손대지 않음). (3) `#### ` 규칙은 `codeSearchHits` 의 `## Code Search` 접두 판별로 두었다 — 도구 이름으로 옮기지 않았다.
- [러너 08:22] repair done — 비평 지적은 전부 맞았다. 재현(실제 `Server.ServeHTTP` 왕복): 청크 하나가 ```bash 를 열고 닫지 않으면 `search-semantic` 의 `### ` 12개가 `result_count=1`, `find-runbook`

## 비평 노트
- approve. 테스트가 변경을 실제로 검증한다 — 직접 돌연변이로 확인: `fencesContent` 를 항상 `true` 로 하면 `TestUnfencedHitsArePastAChunkThatOpensAFence`(두 서브테스트)·`TestTruncatedUnfencedHitsEndOnAWholeHit` 가 깨지고, 항상 `false`(=main) 로 하면 `TestReadFileCut/CountIgnoresHeadingsInsideTheFencedContent`·`TestSymbolContextCountIgnoresListItemsInsideTheFencedContent`·`TestReadFileCountIgnoresContentSubsectionHeadings` 가 깨진다. 양방향이 다 고정돼 있다.
- 검증: `gofmt -l ./cmd ./internal`(빈 출력), `go build`·`go vet -tags sqlite_fts5 ./...`, `go test -tags sqlite_fts5 -count=1 ./...` 전체 통과(이전 회차가 못 본 범위), `go test -race ./internal/mcp` 통과. `lineFence` 추출은 분기별 대조로 main 과 동일, `unfencedLines`/`lastUnfenced` 오프셋 의미는 빈 입력·선두 개행(at=0 → `enough` 거짓)·말미 개행까지 `LastIndex("\n### ")` 와 일치. `contentFence`(format.go:207)가 content 최장 백틱런+1 을 쓰므로 `lineFence` 가 펜스 안에서 조기 종료되지 않음을 직접 확인 — 전제가 성립한다. `tool` 은 `registry[i].name` = `toolcatalog.*` 상수 그대로 전달(별칭 없음).
- 보안·법무 차단 사유 없음: 인증·세션·ACL·마이그레이션·비밀값·개인정보·의존성 무관, 신규 엔드포인트 없음, 손으로 만든 비교/서명 없음. 신규 import 는 기존 내부 패키지 `toolcatalog` 뿐. 되돌리기는 코드 전용 revert.
- 승인해도 남는 우려(상세는 review.json): (1) `fencesContent` 의 도구 이름 허용목록이 `contentFence` 사용처와 **계약으로 묶여 있지 않다** — 세 번째 포매터가 `contentFence` 를 쓰기 시작하면 아무 테스트도 안 깨지고 그 도구만 조용히 구동작으로 남는다. (2) budget.go:131-133 주석의 "다른 포매터는 모두 생prose" 는 `formatRepositoryMap`(format.go:379)의 ```json 펜스를 빼먹었다(실해 없음 — MarshalIndent 는 `### `/`- ` 로 시작하는 줄을 내지 않고 그 경로는 main 과 동일). (3) `clampResponse` 의 `LastIndex("\n### Notes\n")` 는 여전히 펜스 비인식 — main 과 동일함을 확인했고 악화는 없다. (4) `#### ` 는 여전히 `## Code Search` 접두로 판별 — 한 파일에 판별자가 둘이다.
- 못 본 것: 외부 Postgres/pgvector/Vault 통합, 실브라우저, `govulncheck`. 릴리즈 노트는 read-file·get-symbol-context 로 한정해 쓰고, 상류(`internal/indexer/indexer.go:1110` 의 펜스 무인식 분할 = 불균형 청크의 원인, 고치면 재색인 지문 변경)는 다음 회차 몫으로 남겨야 한다.
- [러너 08:30] review approved — 리뷰 승인 (risk=low)
- [러너 08:30] pr created — https://github.com/hkjang/git-ctx/pull/44
- [러너 08:37] ci failed — 성공이 아닌 검사: Known vulnerabilities=failure · 실패한 검사: ? 잡: Known vulnerabilities 
