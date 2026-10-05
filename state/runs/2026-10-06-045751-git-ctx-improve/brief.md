- 과제: CODEOWNERS 패턴에서 **뒤에 슬래시를 붙인 디렉터리 이름이 루트에만 적용되는** 비대칭 고치기 — `internal/search/codeowners.go:codeownersMatch` (가치 4 / 위험 1 / 작업량 S)
- 왜: `codeownersMatch`(codeowners.go:121)는 `directoryOnly` 패턴에 `/**` 를 **붙인 뒤에** `strings.Contains(pattern, "/")` 로 "중첩 패턴인가" 를 판정한다(`:134-137`). 그래서 `docs/` 같은 비앵커 디렉터리 이름이 `docs/**` 가 되어 항상 "슬래시를 포함" 하게 되고, 앵커 분기로 흘러 **루트의 `docs/` 만** 소유한다 — 같은 파일의 계약 주석("a bare name matches at any depth", `:120`)과 기존 테스트가 고정한 파일 쪽 대칭(`codeowners_test.go:21` 의 `{"service.go", "internal/service.go", true}`)을 모두 어긴다. 결과적으로 **디렉터리 이름을 더 명시적으로 쓴 쪽(`docs/`)이 덜 매칭된다**: `docs` 는 `src/docs/readme.md` 를 잡는데 `docs/` 는 놓친다. 고치면 MCP `find-code-owner`(tools.go:420 `handleFindCodeOwner` → `FindOwners`, ownership.go:69·:83)와 의존성 영향 분석의 소유자 표기(dependencies.go:189·:475)가 CODEOWNERS 선언을 놓치고 커밋 작성자 추정으로 떨어지는 일이 없어진다 — 파일 머리말이 "선언은 추정이 아니다" 라고 적어 둔 바로 그 가치다.

- 수용 기준:
  1) `codeownersMatch("docs/", "src/docs/readme.md") == true`, `codeownersMatch("internal/", "cmd/internal/app/auth.go") == true`, `codeownersMatch("docs/", "src/docs") == true` — 이 세 가지가 수정 전 `false`, 수정 후 `true`.
  2) 과대 매칭이 생기지 않는다: `docs/` 는 `mydocs/readme.md`·`src/mydocs/readme.md`·`src/docsx/readme.md` 를 잡지 않고, 앵커 형태 `/docs/` 는 여전히 `src/docs/readme.md` 를 잡지 않으며, 슬래시를 스스로 가진 다중 세그먼트 패턴 `docs/api/` 도 여전히 `src/docs/api/x.md` 를 잡지 않는다(앵커 유지).
  3) 테스트가 증명할 것: `codeowners_test.go:13` 의 기존 표(14줄)는 **한 줄도 바꾸지 말고** 그대로 통과해야 하고, 거기에 위 1)·2) 케이스를 덧붙여 "파일 이름과 디렉터리 이름이 깊이에 대해 같게 동작한다" 는 대칭을 고정한다. 추가로 **대역 없이** 실제 `store.Open` 픽스처로 `declaredOwners`/`FindOwners` 를 한 번 왕복해 `docs/ @team` 선언이 `src/docs/readme.md` 의 `Declared` 에 실제로 실리는 것을 단언한다(기존 선례: `codeowners_test.go:73` 의 `TestDeclaredOwnersAnswerWithoutTheSourceServer` 가 이미 store 픽스처를 쓴다 — 그 모양을 그대로 재사용할 것).

- 건드릴 파일 (프로덕션 1개):
  - `internal/search/codeowners.go:codeownersMatch` — `directoryOnly` 접미사를 붙이기 **전에** 중첩 여부를 정한다. 이 정찰이 실제로 돌려 본 모양(현 코드 `:130-137` 을 그대로 두고 한 줄 추가 + 조건 교체):
    ```go
    directoryOnly := strings.HasSuffix(pattern, "/")
    pattern = strings.TrimSuffix(pattern, "/")
    anchored := strings.HasPrefix(pattern, "/")
    pattern = strings.TrimPrefix(pattern, "/")
    nested := strings.Contains(pattern, "/")   // 추가: /** 를 붙이기 전에 판정
    if directoryOnly {
        pattern += "/**"
    }
    if anchored || nested {                    // 교체: strings.Contains(pattern, "/") → nested
    ```
    `:118-120` 의 계약 주석에 디렉터리 이름도 같은 규칙임을 한 줄로 명시하면 좋다. `globMatch`·`cutPrefixSegments`·`matchCodeowners`·`declaredOwners` 는 **건드릴 필요가 없다**(이 정찰이 수정본 복제로 18케이스를 돌려 확인).
  - `internal/search/codeowners_test.go` — 기존 표에 케이스 추가 + store 픽스처 왕복 테스트 1개.

- 검증 명령:
  - `go test -tags sqlite_fts5 -count=1 ./internal/search` (이 정찰 실측 기준 패키지 전체는 수 초대; `-run TestCodeowners` 만은 0.01s 미만)
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/search`
  - `go test -tags sqlite_fts5 -count=1 ./...` (전 패키지. `internal/app` 이 ~56s 로 벽시계를 지배)
  - `gofmt -l ./cmd ./internal` (빈 출력) · `go vet ./...` · `go build -tags sqlite_fts5 ./...`
  - `bash scripts/verify-version-sync.sh`

- 위험과 피할 것:
  - **기존 표(`codeowners_test.go:14-32`)의 14줄을 약화하거나 수정하지 말 것.** 이 정찰이 그 14줄 중 관련 항목(`{"internal/", "internal/app/auth.go", true}`·`{"internal/", "web/app.js", false}`·`{"/web/**", "internal/web/app.js", false}`·`{"docs/*.md", "docs/ko/guide.md", false}`)을 수정본으로 다시 돌려 전부 그대로임을 확인했다. 값이 바뀌면 고친 방향이 틀린 것이다.
  - **`globMatch` 를 일반화하려 들지 말 것.** `**` 처리와 `cutPrefixSegments` 는 이 비대칭과 무관하고, 손대면 위 네 줄이 움직인다. 이번 변경은 `codeownersMatch` 한 함수의 판정 순서뿐이다.
  - **`matchCodeowners` 의 "마지막 매칭이 이긴다" 순서와 섹션·인라인 주석 파싱(`:96-111`)은 범위 밖.** 매칭이 넓어지면 `last match wins` 로 결정되는 소유자가 바뀔 수 있으므로, `codeowners_test.go:42` 의 `TestCodeownersParsingKeepsTheDecidingRuleLast` 가 무수정 통과하는지 꼭 확인할 것(넓어진 매칭이 그 픽스처의 결정 규칙을 바꾸면 그 사실을 보고할 것).
  - `internal/auth`·`internal/store` migration·`.github/workflows`·`internal/version`·릴리즈 노트·CHANGELOG 는 건드리지 말 것(세션 규칙: 버전 올리기·릴리즈 금지).
  - `dependencies.go:450 declaredOwnersFor` 와 `ownership.go:83` 은 호출자일 뿐이므로 수정 대상이 아니다. 단 `./internal/search` 전체 테스트(특히 `dependencies_test.go`·`ownership_test.go`)가 통과하는지 확인할 것 — 소유자 문자열이 더 많이 채워지면서 기존 단언이 움직일 수 있는 유일한 자리다.
  - 과거 교훈: 같은 값을 읽는 경로가 둘이면 한쪽만 고치지 말 것 — 여기서는 판정 함수가 `codeownersMatch` 하나뿐임을 `grep` 아닌 호출자 열거로 확인했다(`matchCodeowners:108` 이 유일한 호출처).

- 차선 후보: 추적된 디버그 산출물 `c.txt`·`server.log` 제거 + `.gitignore` 등록 — 이번 정찰이 두 파일을 **실제로 열어 비밀값 없음을 확인했다**(`c.txt` 는 libcurl 쿠키 파일 헤더 3줄·쿠키 행 0개, `server.log` 는 `GIT_CTX_RECOVERY_KEY is required and must contain at least 32 characters` 한 줄로 키 값이 아니다). 8회차 연속 보류였던 "내용 확인 선행" 조건이 이로써 해소됐으므로 1순위가 성립하지 않으면 바로 집을 수 있다. 삭제 전 두 이름이 코드·스크립트·워크플로에 참조되지 않는지 확인할 것.

---
## 이 과제의 근거 — 정찰이 실제로 돌려 본 결과
수정 전 현재 코드와 "한 줄 재배치" 수정본을 같은 세션에서 18케이스로 나란히 돌렸다(프로덕션 파일은 건드리지 않고 `internal/search` 에 임시 테스트를 넣어 실행한 뒤 삭제했다 — `git status --short` 빈 출력 확인). 결과:

```
pattern="docs/"     path="src/docs/readme.md"       want=true  current=false(CURRENT-WRONG) fixed=true
pattern="docs"      path="src/docs/readme.md"       want=true  current=true  fixed=true
pattern="internal/" path="cmd/internal/app/auth.go" want=true  current=false(CURRENT-WRONG) fixed=true
pattern="internal"  path="cmd/internal/app/auth.go" want=true  current=true  fixed=true
pattern="docs/"     path="src/docs"                 want=true  current=false(CURRENT-WRONG) fixed=true
pattern="docs/"     path="docs/readme.md"           want=true  current=true  fixed=true
pattern="internal/" path="internal/app/auth.go"     want=true  current=true  fixed=true
pattern="internal/" path="web/app.js"               want=false current=false fixed=false
pattern="docs/"     path="mydocs/readme.md"         want=false current=false fixed=false
pattern="docs/"     path="src/mydocs/readme.md"     want=false current=false fixed=false
pattern="docs/"     path="src/docsx/readme.md"      want=false current=false fixed=false
pattern="/docs/"    path="src/docs/readme.md"       want=false current=false fixed=false
pattern="/docs/"    path="docs/readme.md"           want=true  current=true  fixed=true
pattern="/docs"     path="src/docs/readme.md"       want=false current=false fixed=false
pattern="docs/api/" path="src/docs/api/x.md"        want=false current=false fixed=false
pattern="docs/api/" path="docs/api/x.md"            want=true  current=true  fixed=true
pattern="a/b/"      path="a/b/c/d.md"               want=true  current=true  fixed=true
→ current implementation disagrees with the gitignore contract in 3 of 18 cases
```
즉 **틀린 3건이 모두 고쳐지고 나머지 15건(과대 매칭 방지·앵커 유지 대조군)은 한 건도 움직이지 않는다.** 구현자는 이 표를 그대로 `codeowners_test.go:13` 의 기존 표 아래에 붙이면 수용 기준 1)·2)를 동시에 만족한다.

### 미확인으로 남긴 것
- 이 저장소에 색인된 실제 CODEOWNERS 파일 중 비앵커 디렉터리 패턴(`docs/` 형태)을 쓰는 것이 실제로 몇 건인지 — 외부 GitLab·Bitbucket 데이터가 없어 확인하지 못했다. 즉 **영향 규모는 미확인**이고, 결함 자체는 위 실측으로 확정이다.
- `find-code-owner` MCP 왕복까지의 end-to-end 재현은 하지 않았다(코드 경로 `handleFindCodeOwner`→`FindOwners`→`declaredOwners`→`matchCodeowners`→`codeownersMatch` 는 호출자 열거로 확인했으나 실행은 안 했다). 구현자는 수용 기준 3)대로 store 픽스처 왕복 테스트로 이 구간을 메울 것.
