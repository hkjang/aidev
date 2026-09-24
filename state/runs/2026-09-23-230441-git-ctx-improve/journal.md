# 회차 노트 2026-09-23-230441-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:04] base pinned — main@5120fb9
- [러너 23:04] autonomy release — 

## 정찰 노트
- 골랐다: 지난 회차가 mcp `cutAtBoundary` 만 고쳤고(1dd6363, HEAD 에 있음) 같은 텍스트의 상류인 `internal/search` 의 두 바이트 절단(ReadFile 192KiB·ExportContext 200000B)은 그대로다 — 운영자가 반복해 말한 "같은 값을 읽는 경로가 여럿이면 전부 맞추라"에 정확히 해당하고, S·저위험이며 실제 출력(REST JSON 의 U+FFFD)이 바뀐다.
- 제친 것: 마스킹·MCP OAuth 파생 항목은 이 HEAD 에 선행 작업이 미반영(placeholderRE·Restricted() grep 0건)이라 전제가 없고, 매니페스트 파서 확장은 반려 이력 유형, app flake 는 실패 이름 미확보.
- 추측으로 적은 것: 절단점이 한글 중간에 걸린다는 산수는 손계산이며 런타임 재현은 샌드박스가 `go run`/`python3` 을 막아 **미확인**이다. 구현자는 수정 전 실패를 반드시 먼저 재현할 것.
- 조심할 것: MCP 기본 예산 24KiB 때문에 MCP 경로에서는 손상이 가려질 수 있다(노출 경로는 REST 플레이그라운드). `runeSafeCut` 은 mcp 패키지라 임포트하면 순환이므로 search 안에 헬퍼를 따로 둘 것. 예산 상수와 Revision 은 건드리지 말 것.
- 프로필: 1일 전 것이 지금 코드와 어긋나지 않아 새로 쓰지 않았다(HEAD 만 5120fb9/v0.77.15 로 바뀜).
- [러너 23:08] scout done — search 쪽 바이트 절단(ReadFile 192KiB, ExportContext 200000B)이 UTF-8 글자를 반으로 자르지 않게 수정 (가치 3 / 위험 

## 구현 노트
- 무엇/왜: `internal/search/service.go` 의 ReadFile(192KiB)·ExportContext(200000B) 바이트 절단을 새 `cutAtRuneBoundary` 로 바꿔 글자 중간에서 자르지 않게 했다(commit 29e91ff). 예산 상수·`Truncated` 플래그·진단/공지 문구는 그대로.
- 확신 없는 곳: (1) 과제서의 선택 항목이던 REST 플레이그라운드 HTTP 왕복 1건은 하지 않았다 — 증거는 실제 store+Service 레벨과 JSON 왕복까지만이다. `internal/app/playground.go` 의 jsonOut 이 이 값을 그대로 직렬화한다는 것은 코드를 읽어 확인했을 뿐 런타임으로 밟지 않았다. (2) ExportContext 회귀는 같은 libraryID 를 20개 넘기지 않고 8개 대형 청크 한 라이브러리로 200000B 를 넘겼는데, 기본 FinalK=8 에 의존한다 — 설정 기본값이 바뀌면 이 테스트가 절단에 도달하지 못하고 조용히 약해질 수 있다.
- 검증 중 고친 것: 처음 쓴 `strings.Contains(json, "�")`(원문자)는 **항상 통과하는 무의미한 단언**이었다. `encoding/json` 은 invalid UTF-8 을 6바이트 이스케이프 `\ufffd` 로 쓴다(바이트 덤프로 확인). 이스케이프 검사 + 역직렬화 왕복 비교로 바꿨고, UTF-8 단언을 지운 채 수정 전 코드로 돌려 **JSON 단언만으로도 실패**함을 확인했다. 비평가는 이 지점을 먼저 볼 것.
- 일부러 안 한 것: mcp `runeSafeCut` 재사용(mcp→search 임포트라 순환), 예산 상수 변경, `contentsecurity.Revision()` 변경(응답 절단이라 재색인과 무관), 차선 후보(mcp.cacheKey) 착수.
- 다음 역할 주의: 새 테스트는 실 SQLite(`store.Open("sqlite", …)`)가 필요하고 `sqlite_fts5` 태그로 돌려야 한다. `internal/app` 은 통과해도 매번 ~100초다(이번 2회 모두 통과, flake 재현 없음). 같은 계약의 절단 함수가 이제 search·mcp 두 곳에 있으니 한쪽만 고치지 말 것.
- [러너 23:20] brief accepted — 채택 — 지정한 두 절단 자리와 근거가 현재 코드와 정확히 맞았고, 수정 전 실패(UTF-8 및 JSON 양쪽)를 재현해 수용 기준 1~
- [러너 23:21] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve(low, blocking 없음). 확인: main@5120fb9 을 detached worktree 로 꺼내 새 테스트만 복사해 돌려 **수정 전 FAIL**(truncation_test.go:72, :192)을 직접 재현했다 — 테스트는 무의미하지 않다. pad 0~3 이 3바이트 문자의 세 오프셋을 모두 덮는 것도 산수로 확인.
- 구현자의 불확실 2건 해소: playground.go:173·:613 모두 jsonOut 이라 주석의 U+FFFD 설명이 실제 경로와 맞고(HTTP 왕복 실행은 여전히 미확인), FinalK=8 걱정은 기우 — 절단 미도달 시 notice 접미부 단언이 조용히 약해지지 않고 곧바로 FAIL 한다.
- 남는 우려(다음 회차): assembleSourceQueryResults 의 `snippet[:16000]`(service.go:4793, 4726)과 `description[:4000]`(3039)은 여전히 바이트 절단이고, 4793 은 Query→ExportContext **상류**다. export 본문은 고쳐졌지만 그 안의 원격 스니펫은 아직 U+FFFD 를 만들 수 있다. 이번 커밋 범위 밖이라 차단하지 않았다.
- 중복: 새 cutAtRuneBoundary 는 같은 파일 1785줄 clipText 와 루프가 동치인데 주석은 mcp.runeSafeCut 만 언급한다. search 안에만 같은 계약 함수가 둘 — 통합 검토 권장(비차단).
- 안 본 것: ./internal/app(~100초), -race, 전체 suite, 실제 HTTP 왕복. 돌린 것은 gofmt/vet 무출력 + search·mcp·contentsecurity -count=1 전부 ok. 릴리즈 노트에는 "REST read-file·export 응답의 마지막 글자 깨짐 수정"으로 적으면 된다.
- [러너 23:24] review approved — 리뷰 승인 (risk=low)
- [러너 23:24] pr created — https://github.com/hkjang/git-ctx/pull/37
- [러너 23:32] ci passed — 검사 5개 모두 success
- [러너 23:32] merge done — c8cebc2
- [러너 23:47] release ci-blocked — 릴리즈 커밋 CI: failed — 성공이 아닌 검사: Admin console=failure (태그 보류)
