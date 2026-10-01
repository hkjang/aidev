# 회차 노트 2026-10-01-214215-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:42] base pinned — main@f1de053
- [러너 21:42] autonomy release — 

## 정찰 노트
- 보류 목록의 [2/2/S] "sectionCount가 #### 히트를 세지 못함" 을 골랐다. 다른 pending 은 가치 2 이하거나 재현 경로가 미입증(formatRepositoryMap 의 JSON 펜스, LastIndex Notes 오인)인데, 이것만 임시 프로브로 **감사 열과 공지 숫자가 동시에 거짓**임을 실측해 확정했다: 50건 중 19건 전송에 result_count=3, 공지 "2 of 2", 본문 끝은 Source 인용 없는 반쪽 히트. 프로브는 삭제해 작업 트리는 깨끗하다.
- 범위를 budget.go 두 함수로 못박았다. format.go 의 제목 수준을 바꾸는 쪽은 모든 에이전트의 렌더링을 바꾸면서도 개수는 여전히 2만큼 틀리므로 제외했다.
- 추측으로 남긴 것: 수정 후 find-symbol 계열의 result_count 가 그대로인지는 미측정이다(read-file=0 은 실측). 수용 기준 4로 구현자가 관측할 것.
- 조심할 것: 기존 truncation_test.go:57 은 히트를 `### ` 로 쓰는 프로덕션에 없는 모양을 합성해 통과한다 — 그 파일을 고쳐 맞추지 말고 실제 HTTP 왕복으로 새 테스트를 쓸 것. mcp fixture 는 공유 in-memory SQLite 라 t.Parallel() 금지.
- 펜스 안쪽 제목을 집계에서 빼는 일반화는 의도적으로 다음 회차로 미뤘다(ideas.json 에 pending 으로 기록).
- [러너 21:48] scout done — 결과 개수 집계와 절단 경계가 `#### ` 로 쓰인 search-code 히트를 보게 하기 — `sectionCount`·`cutAtBoundary` (가치 

## 구현 노트
- 무엇/왜: `internal/mcp/budget.go` 의 `sectionCount`·`cutAtBoundary` 사다리 맨 앞에 `\n#### ` 한 단을 붙였다(프로덕션 1파일). `formatCodeSearch` 가 히트를 `#### ` 로 쓰는데 두 함수가 `\n### ` 만 봐서 `mcp_calls.result_count`=3 고정, 공지 "2 of 2", 마지막 히트가 `Source:` 없이 잘리는 세 증상이 같은 원인이었다. commit f7d57cc.
- 확신 없는 곳: (1) 이번 변경으로 **파일 본문**의 `#### ` 도 집계에 들어간다 — fixture 의 read-file 대상 파일에는 `#### ` 가 없어 대조군이 0 으로 통과했지만, `#### ` 제목이 든 마크다운을 read-file 하면 result_count 가 본문 제목 수가 된다. 수정 전에도 `### ` 로 같은 성질이 있었으므로 성질은 선존이고 노출 범위만 넓어졌다(ideas.json 에 pending 으로 승격). (2) 대조군 기대값 `find-symbol=2`·`get-symbol-context=4`·`read-file=0` 은 이 fixture 의 실측값을 박은 것이라 fixture 데이터를 늘리면 이 테스트가 깨진다. (3) 포매터 중 `#### ` 를 쓰는 곳은 `format.go:118`(formatCodeSearch) 한 곳뿐이고 대조군 세 도구의 응답에 `#### ` 가 없음도 테스트가 단언하지만, 그것은 포매터가 쓰는 제목 이야기일 뿐 (1)의 파일 본문 유래 `#### ` 와는 별개다.
- 일부러 안 한 것: `format.go` 의 제목 수준은 그대로 뒀다(모든 에이전트의 렌더링이 바뀌고 개수는 여전히 2만큼 틀림). 히트 0건 응답을 0 으로 만드는 일, 펜스 안쪽 제목 제외 일반화, `responseNoticeBytes` 조정은 범위 밖. 기존 `truncation_test.go`·`fence_test.go` 는 한 줄도 고치지 않았다.
- 다음 역할이 조심할 것: 새 테스트 `internal/mcp/result_count_test.go` 는 sqlite 공유 in-memory fixture 를 쓰므로 `t.Parallel()` 금지(그래서 넣지 않았다), `-tags sqlite_fts5` 필수. 절단 케이스는 `limit:50, maxBytes:4000` 과 60개 패딩 청크의 **길이**에 의존해 경계가 잡히므로, 패딩 문구를 바꾸면 수치가 아니라 관계식만 유효하다(테스트는 숫자를 하드코딩하지 않고 응답에서 센다). 외부 DB·Vault·Docker·실브라우저·govulncheck 는 미실행이고 릴리즈는 하지 않았다.
- [러너 21:55] brief accepted — 채택 — 지정한 두 함수·줄 번호·근거가 현재 코드와 정확히 맞았고, 과제서가 예측한 세 증상(result_count=3, 공지 "2 of 2", 
