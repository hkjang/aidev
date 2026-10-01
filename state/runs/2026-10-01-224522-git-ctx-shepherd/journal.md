# PR 처리기 노트 2026-10-01-224522-git-ctx-shepherd — git-ctx PR #43
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-01-214215-git-ctx-improve)
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
- [러너 21:56] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 확인한 것: budget.go 를 main 으로 되돌려 새 테스트가 **실제로 수정 전 실패**함을 재현했다(원장 `실패 재현:` 네 줄과 출력이 정확히 일치, 대조군 3개는 수정 전에도 PASS). 테스트는 진짜 `Server.ServeHTTP` 왕복이고 숫자를 응답에서 센다 — 항상 참인 단언 없음. gofmt·vet·build·`./internal/app ./internal/mcp ./internal/search` 전부 green(app 100.6s). 임시 프로브는 모두 삭제, 작업 트리 clean.
- **거절 사유(수리가 먼저 볼 파일: `internal/mcp/budget.go:109`)**: 새 `\n#### ` 절단 단이 무조건이라 **파일 본문**의 `#### ` 에도 걸린다. `#### ` 소제목이 중간에, `### ` 가 뒤에 오는 마크다운을 read-file 하면 기본 24KiB 예산에서 본문이 main 21,775B → HEAD 16,325B 로 **25% 줄었다**(예산 3,400 에서도 2,725→2,154 로 재현). budget.go:107 주석은 블래스트 반경을 "`#### ` 없는 응답은 그냥 통과" 로만 적어 코드와 어긋난다. 수리 방향: 두 단 모두 `### Source Matches (` 이후 구간으로 한정.
- 못 본 것: 전체 `./...`·`-race`·외부 Postgres/pgvector/Vault/Docker·실브라우저·govulncheck.
- 승인했어도 남을 우려(다음 회차): `clampResponse` 의 `shown`(`### ` 단위로 떨어질 수 있음) 과 `total`(`#### ` 단위) 이 **단위가 다르다** — 리포지터리 머리말이 길면 히트 0건인데 "1 of 50 included" 라 말한다. main 도 같은 자리에서 "1 of 2" 로 똑같이 거짓이라 회귀는 아니지만, 이번 수정이 `total` 만 고쳐 거짓말이 더 또렷해졌다. 단일 히트가 room 의 40% 를 넘으면 여전히 히트 중간에서 잘려 `Source:` 가 사라지는 것도 main 과 동일(= "last entry is whole" 은 일반 보장이 아님).
- 보안·법무: 차단 없음. ACL 필터를 이미 거친 텍스트에서 바이트를 빼기만 하므로 권한을 넓히지 않고, `result_count` 는 콘솔 표시·CSV·AVG 전용(mcpadmin.go:223,440,479)이라 제어 흐름 소비자가 없다. 새 개인정보 수집·보존·이전 없음, 의존성·라이선스 변경 없음.
- [러너 22:03] review rejected — 리뷰 거절: internal/mcp/budget.go:109 — the new `\n#### ` branch in `cutAtBoundary` is unconditional, so it also fires on `#### ` lines that are *file content*, not code-search hits. Measured on
- [러너 22:03] pr created — https://github.com/hkjang/git-ctx/pull/43

## 수리 노트
- 지적 두 건 **모두 맞았다**. 합성 없이 실측으로 확인: `#### ` 소제목 24개 뒤에 산문과 `### ` 가 오는 마크다운을 read-file 하면 cut 이 본문 `#### ` 를 골라 예산 3400 에서 2,715→2,371B, 24KiB 에서 23,798→23,780B 로 줄었다. 주석의 "`#### ` 없는 응답은 그냥 통과" 도 본문에 `#### ` 가 있으면 거짓이었다.
- 고친 방법: `codeSearchHits` 한 함수를 추가해 `## Code Search\n` 접두 + `\n### Source Matches (` **이후 구간**에서만 `#### ` 를 세고 자르도록 두 단을 모두 한정했다(비평가가 지시한 방향에 접두 검사 하나를 더 보탰다 — 두 문자열은 `formatCodeSearch` 만 쓰므로 앵커가 유일하다). 주석도 실제 범위대로 다시 썼다. commit d286e60.
- 테스트는 먼저 빨갛게 봤다: 새 2개가 HEAD 에서 FAIL, budget.go 를 main 으로 되돌리면 PASS(회귀 양방향 확인). 수정 후 read-file 전달 바이트가 main 과 정확히 일치하고, search-code 테스트 2개는 main 의 budget.go 에서 여전히 FAIL — 원래 버그는 계속 잡힌다. `./...` 전체 green(app 101.0s), gofmt·vet·build 통과.
- 여전히 확신 없는 곳: (1) 비평가가 남긴 `shown`/`total` **단위 불일치**는 손대지 않았다 — 범위 밖이고 main 과 동일하다. 단, `total` 이 이제 코드서치에서만 `#### ` 단위이므로 불일치는 코드서치 응답에 국한된다. (2) `### Source Matches (` 가 cut window 밖이면 `#### ` 단을 건너뛰고 `### ` 단으로 내려가는데, 그 경우 창 안에 히트가 없으므로 손실은 없다고 판단했다(인자가 `window` 인 이유). (3) 본문 유래 `#### ` 를 세던 **선존** 성질 중 `### ` 쪽은 그대로다 — read-file 의 result_count 는 여전히 본문 `### ` 수이고, 테스트는 그 main 값을 고정한다.
- 미실행: `-race`, 외부 Postgres/pgvector/Vault, Docker, 실브라우저, govulncheck, JS 계약 테스트(Go 만 건드렸다). push 하지 않았다.

## 심사 노트
- 권고 **merge**. 거절 사유 두 건 모두 해소를 실측으로 확인: `#### ` 소제목 160개와 늦은 `### ` 를 가진 37,290B 마크다운을 read-file 로 왕복시켜 전달 바이트를 세 버전에서 비교했다 — 24KiB 예산에서 main 23,603B / HEAD 23,603B(동일) / 거절된 f7d57cc 17,434B. 비평가의 25% 손실은 사라졌다. 프로브는 삭제, 작업 트리 clean.
- 코드로도 확인: `codeSearchHits` 는 `## Code Search\n` 접두 **와** `\n### Source Matches (` 를 둘 다 요구하고, 두 문자열은 `formatCodeSearch`(format.go:94,109)만 쓰며 호출자는 tools.go:100 한 곳뿐이다 — 코드서치 아닌 응답의 절단 경로는 main 과 동일하다. 주석 범위도 코드와 일치한다.
- 테스트 양방향: origin/main 으로 되돌리면 search-code 2개 FAIL, f7d57cc 로 되돌리면 read-file 가드 2개 FAIL, HEAD 는 5개 PASS. 실제 `ServeHTTP` 왕복이고 기대값을 응답에서 센다. gofmt·vet·build·`go test -tags sqlite_fts5 ./...` 전부 green.
- 남는 선존 성질(회귀 아님, notes 로 남김): 히트 **스니펫 본문**의 `#### ` 는 여전히 코드서치 집계·절단을 흔들 수 있고, `shown`/`total` 단위 불일치와 거대 단일 히트의 중간 절단은 main 과 동일하다.
- 못 본 것: `-race`, 외부 Postgres/pgvector/Vault, Docker, 실브라우저, govulncheck, JS 계약 테스트. 보호 파일(auth·session·migration) 미접촉이라 추가 항목은 해당 없음. risk low, blocking 없음.
