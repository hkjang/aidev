- 과제: YAML 리스트 블록 스칼라가 형제 필드까지 과마스킹하지 않게 수정 (가치 3 / 위험 2 / 작업량 M)
- 왜: `maskBlockScalars`가 `- password: |`의 깊이를 키가 아닌 대시 앞 공백으로 계산하여 같은 항목의 `name`·`port`까지 비밀 본문으로 가린다. 키의 실제 시작 열에서 블록 경계를 판단하면 비밀은 계속 숨기면서 검색·ReadFile에 필요한 설정 구조를 보존한다.
- 수용 기준:
  1) 아래 재현 입력은 비밀 본문 `hunter2`만 `[REDACTED]`로 바뀌고 헤더·`name`·`port`·다음 항목은 바이트 그대로 남는다. 본문 없이 바로 형제 `name`이 오는 두 번째 재현 입력은 원문 그대로이고 finding도 빈 문자열이어야 한다.
  2) 일반 매핑, 리스트 항목 첫 키, 리스트의 후속 비밀 키, 중첩 리스트, `-   password:`처럼 대시 뒤 공백이 여러 개인 입력에서 키 열 기준 경계가 맞는다. 기존 `|`, `>-`, `|2` 지원과 본문 마스킹을 유지하고 빈 줄·LF/CRLF·마지막 개행 유무를 보존한다. 비밀 본문 자체에 `name: literal`처럼 콜론이 있는 줄은 여전히 전부 가려야 한다.
  3) 전체 출력 비교 회귀 테스트가 수정 전 실패/수정 후 통과하며, `Sanitize`를 두 번 적용해도 형제 필드가 보존된다. 기존 `TestABlockScalarStatesItsValueOnTheFollowingLines`, `TestMaskingNeverChangesTheLineCount`가 통과하고 `Revision()`은 현재 `edeca363cffe`와 달라지되 반복 호출 시 안정적이다. `ReadFile`의 기존 테스트 픽스처를 이용한 사례 하나로 최종 반환 Content에도 형제 필드가 남고 비밀이 없는지 증명한다.
- 건드릴 파일:
  - `internal/contentsecurity/sanitize.go:blockScalarSecretRE, maskBlockScalars, Revision` — 헤더에서 키 앞 접두부(선행 공백 + 선택적 대시와 그 뒤 공백)를 캡처하고 그 길이를 깊이로 쓰는 최소 수정을 권장한다. 캡처 정규식 자체가 바뀌면 현재 Revision 해시에도 반영된다. `blockIndent`는 본문 들여쓰기 계산용으로 유지할 수 있다.
  - `internal/contentsecurity/credential_shapes_test.go:TestABlockScalarStatesItsValueOnTheFollowingLines, TestMaskingNeverChangesTheLineCount, TestTheMaskingRevisionTracksTheRules` — 위 경계·원문 보존 사례를 테이블 기반 테스트로 추가한다. 기존 전역 정규식 교체 테스트에는 t.Parallel을 붙이지 않는다.
  - `internal/search/service_test.go:TestReadFileServesIndexedAndUnindexedFiles` — 기존 SQLite/가짜 source 패턴을 재사용한 작은 회귀 사례를 추가한다. 운영 코드 `internal/search/service.go:ReadFile`은 읽기 참고용이며 수정 불필요.
  - `docs/operations.md:데이터 보호` — 리스트에서는 대시 앞 들여쓰기가 아니라 키의 열이 블록 경계임을 한 문장 보완한다.
- 검증 명령: 저장소 루트에서 `go test -tags sqlite_fts5 ./internal/contentsecurity ./internal/indexer ./internal/search ./internal/mcp` (정찰 실행 exit 0; search 1.278s, 나머지 캐시); 구현 후 회귀 비캐시 확인 `go test -tags sqlite_fts5 -count=1 ./internal/contentsecurity ./internal/search`; `go test -tags sqlite_fts5 -race ./internal/contentsecurity ./internal/search`; `go test -tags sqlite_fts5 ./...`; `go vet ./...`; `go build -tags sqlite_fts5 ./...`; `gofmt -l internal/contentsecurity internal/search` (출력 없음이 성공). 뒤의 race·전체·vet·build는 CI에서 사용하는 명령 형식이며 이번 정찰에서는 미실행.
- 위험과 피할 것: 전체 YAML 파서·새 의존성 도입, 새 비밀 키 명명 규칙, malformed YAML 처리 확대, 인증/ACL·store migration·.github/workflows·버전 파일은 범위 밖이다. 콜백만 고쳐 Revision이 그대로 남는 실수를 피한다. `${VAR}` 예외·Authorization 개행·curl 접두부·OAuth 캠페인은 기록상 성공했으나 cf3b598에는 없으므로 재구현하거나 묶어 넣지 않는다. parser 확장 반려와 app flake 재현 실패를 반복하지 않는다. ReadFile은 자체적으로 CRLF를 LF로 정규화하므로 CRLF 보존 검증은 Sanitize 경계에서 한다.
- 차선 후보: `mcp.cacheKey`가 호출자의 ACLPrincipals 슬라이스를 정렬하지 않게 복사 (가치 2 / 위험 1 / S) — 1순위가 이미 해결된 기준에서만 선택. `internal/mcp/cache.go:cacheKey`에서 principalACLs 결과를 복사한 뒤 정렬하고 `internal/mcp/cache_test.go`에 원본 불변·순서가 다른 동일 ACL 집합의 동일 키를 검증한다. `principalACLs`와 `WithUnrestricted`의 권한 의미나 filterLibraries까지 함께 바꾸지 않는다. `go test -tags sqlite_fts5 ./internal/mcp`로 검증.

실행으로 확인한 근거 (cf3b598):
```text
입력:  "items:\n  - password: |\n      hunter2\n    name: public-service\n    port: 8080\n  - name: next\n"
현재:  "items:\n  - password: |\n      [REDACTED]\n    [REDACTED]\n    [REDACTED]\n  - name: next\n"
기대:  "items:\n  - password: |\n      [REDACTED]\n    name: public-service\n    port: 8080\n  - name: next\n"
입력2: "items:\n  - password: |\n    name: public-service\n  - name: next\n"
현재2: "items:\n  - password: |\n    [REDACTED]\n  - name: next\n", finding="credential_assignment"
```
원본 sanitize.go를 산출물 assets/sanitize-probe.go에 복사하고 package/main 진입점만 붙여 실행했다(저장소 변경 없음). 재실행: `go run /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-135414-git-ctx-improve/assets/sanitize-probe.go`. 이 복사본은 수정 전 증거이므로 수정 후 검증은 저장소 테스트로 한다.

진행 순서·예산: 재현 테스트 8분 → 접두부 캡처/깊이 계산과 지문 확인 10분 → ReadFile 회귀·문서 10분 → 검증 12분 + 여유 5분 = 45분. 15분 내 최소 수정이 성립하지 않으면 범위를 늘리지 말고 원인을 기록한다. 새 YAML 문법 지원은 후속 아이디어로 남긴다.

조사 제한: 요청한 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 도구 목록 및 로컬 .codex/.claude/.agents·aidev 스킬 검색에서 발견되지 않았고 Skill 호출 도구도 없다. 해당 절차·반환 형식은 미확인이며 이 과제서는 사용자 지정 형식을 따른다. 실제 YAML 라이브러리와 대조·외부 DB 통합·전체 suite는 이번 미검증이다.
