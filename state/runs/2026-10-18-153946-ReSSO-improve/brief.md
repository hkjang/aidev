# 과제서 — 2026-10-18-153946-ReSSO-improve

- **과제**: README 지표 표의 `resso_client_auth_errors_total` 행이 "401로 나간다"고 말하는 것을 세 Endpoint의 실제 계약(token 500 / revoke 503 / introspection 200 `active=false`)으로 고치고, 같은 규칙의 두 텔링이 다시 어긋나지 않게 실행 테스트로 고정 (가치 3 / 위험 1 / 작업량 S)

- **왜**: `README.md:191` 은 이 계열을 "401로 나가는 것은 Secret이 틀린 경우와 같고" 라고 설명하는데, 코드는 더 이상 그렇게 답하지 않는다 — `internal/httpserver/oidc.go:1010-1029` 의 주석이 "The three endpoints sharing this helper used to answer a failed client authentication identically; all three no longer do" 로 그 사실을 직접 적고 있고, token 은 500 `server_error`, revoke 는 503 `temporarily_unavailable`, introspection 은 **200 `{"active":false}`** 로 답한다. 이 계열이 존재하는 이유 자체가 "응답만 보면 장애가 보이지 않는다" 인데(introspection 쪽은 HTTP 200 이라 요청 카운터에서 정상 호출로 보인다) README 는 운영자에게 **없는 401 을 찾으라고** 안내하고 있어, 경보가 울렸을 때 가장 먼저 열리는 문서가 진단을 반대 방향으로 돌린다. `docs/operations.md:90` 의 같은 불릿은 이미 세 답을 정확히 적고 있으므로 이것은 정본이 둘로 갈린 상태이고, 고치면 두 문서가 한 가지를 말한다.

- **수용 기준**:
  1. `README.md:191` 의 `resso_client_auth_errors_total` 행이 더는 401 을 이 계열의 응답으로 말하지 않고, 세 Endpoint의 답을 각각 명시한다 — token 500 `server_error`, revoke 503 `temporarily_unavailable`, introspection 200 `{"active":false}`(그리고 그래서 introspection 쪽 장애는 `resso_introspection_errors_total{stage="client_auth"}` 에만 보인다는 사실). 401 `invalid_client` 은 **Secret 이 틀린 경우**(= `resso_client_auth_failures_total`)에만 붙는 말로 남는다.
  2. 그 행이 이미 맞게 적고 있는 두 사실은 **그대로 유지**한다 — ① `stage` 라벨이 `client`·`secret` 둘이라는 것 ② 검증 중에 호출자가 연결을 끊은 시도는 이 계열에도 세지 않고 Rate limit 예산도 깎지 않는다는 것. (②는 `oidc.go:1046-1047` 의 `if r.Context().Err() != nil { return true, false }` 로 확인했다.)
  3. `cmd/resso/pins_test.go` 에 새 테스트가 생기고, **README 수정 전에 실패하고 수정 후 통과한다**. 그 테스트는 README 표 행과 `docs/operations.md` 불릿 **두 텔링을 모두 찾아** 각각이 세 답을 다 명시하는지 검사하며, 하나도 못 찾으면 `t.Fatal` 로 공허한 통과를 막는다(`checked == 0` 가드 — `pins_test.go:298-300` 의 선례와 같은 모양). 즉 증명해야 하는 것은 "README 문자열이 바뀌었다" 가 아니라 "같은 규칙을 말하는 모든 자리가 세 답을 말한다" 이다.
  4. 기존 `cmd/resso` 테스트가 **무수정으로** 전부 통과한다 — 특히 README 를 함께 읽는 `cmd/resso/command_docs_test.go:35` 와 `cmd/resso/pins_test.go:202,273`.

- **건드릴 파일** (프로덕션 Go 코드 **0개**, 총 2개):
  - `README.md:191` — 지표 표의 `resso_client_auth_errors_total` 행 한 줄. 표의 다른 행은 손대지 않는다.
  - `cmd/resso/pins_test.go` — 새 테스트 1개 추가(이름 예: `TestEveryTellingOfTheUndecidedClientAuthNamesAllThreeAnswers`). `TestEveryTellingOfTheDigestKeyRuleNamesClientSecrets`(`:270-301`)의 구조를 그대로 쓴다: `filepath.Join("..", "..")` 루트, `os.ReadFile`, 블록 분할, `checked` 카운터, `checked == 0` → `t.Fatal`. README 는 표 **행**이므로 `\n\n` 블록 분할이 표 전체를 한 블록으로 묶는다 — 계열 이름이 든 `|` 로 시작하는 **줄 단위**로 찾는 편이 맞고, operations.md 는 `- ` 불릿 줄 단위로 찾는다. 두 자리의 추출 방식이 다르다는 점을 구현자가 직접 확인하고 맞출 것.

- **검증 명령** (이 저장소에서 실제로 도는 것):
  - `go test ./cmd/resso -count=1 -run '^TestEveryTellingOfTheUndecidedClientAuthNamesAllThreeAnswers$' -v` — README 수정 **전**에 먼저 돌려 red 를 기록하고, 수정 후 PASS.
  - `go test ./cmd/resso -count=1` — 기존 문서·핀 테스트 무수정 통과.
  - `gofmt -l cmd` 가 비어 있을 것.
  - `golangci-lint run ./cmd/... ./internal/...` (0 issues).
  - 전체: `test_env="$(scripts/test-services.sh)" && eval "$test_env" && make test` — 같은 셸에서. 빌드가 `webui/dist/index.html` 을 바꾸면 복원하고 `git status` 에 빌드 산출물을 남기지 말 것.

- **위험과 피할 것**:
  - `docs/operations.md:90` 의 긴 불릿은 **이미 정확하다**. 다시 쓰거나 줄이지 말 것 — 수용 기준 3 의 테스트는 그 불릿이 **지금 상태로 통과**하는 것이 정상이고(README 쪽만 red), 그것이 테스트가 형태를 가린다는 증거다.
  - `internal/httpserver/oidc.go`·`metrics.go` 를 **고치지 말 것**. 이 과제는 문서와 테스트뿐이고, 세 Endpoint의 응답 계약은 이미 사람이 받아들인 상태다. 응답 상태를 "통일" 하려는 변경은 프로필의 설계 관례("Endpoint 계약은 개별적이다. 공통 응답 처리로 통합하지 않는다")와 정면으로 어긋난다.
  - README 표의 **다른 행**(`resso_token_errors_total` 등)이 operations.md 보다 짧다는 이유로 함께 늘리지 말 것. 표는 요약이고 operations.md 가 정본이다 — 이번에 고치는 것은 "짧다" 가 아니라 "**틀렸다**" 인 한 행이다.
  - `client_id` 를 로그·지표 라벨·감사 detail 로 흘리는 설명을 새로 쓰지 말 것. `oidc.go:1038-1041` 이 "clientID is not recorded" 로 이유까지 적어 두었고 이것은 운영자 규칙이다.
  - `make lint` 는 이 환경에서 PATH 문제로 깨진다(`Makefile:63-64` 의 `PATH="$(go env GOPATH)/bin:$PATH"` 가 `goX.Y.Z` 심을 툴체인으로 고르게 한다 — 지난 회차가 재현해 `ideas.json` 에 남긴 별건). **이번 과제에서 손대지 말고**, 대신 위처럼 `golangci-lint` 를 직접 돌릴 것.
  - 보호 경로는 전부 피한다 — `internal/store/migrations`, `.github/workflows/`, auth/session 코드 어디도 건드리지 않는다.

- **차선 후보**: `scripts/test-services.sh` 의 포트 선점 실패 안내에 `RESSO_TEST_POSTGRES_PORT` 로 우회할 수 있다는 한 줄을 더하기 (2/1/S) — 현재 안내가 "`--stop` 하고 다시 실행" 뿐인데 포트를 다른 프로젝트 컨테이너가 쥐고 있으면 `--stop` 후에도 같은 실패가 반복된다. 분기는 `scripts/test-services.sh` 의 `:68`·`:88`·`:356` 부근으로 **지난 회차 기록에 적힌 위치이며 이번 정찰에서 다시 열어 확인하지 않았다(미확인)** — 구현자가 먼저 줄 번호를 확인할 것.

## 이번 정찰이 실제로 확인한 것 / 미확인

**확인(파일을 열어 읽음)**
- `README.md:191` 의 문구 — "401로 나가는 것은 Secret이 틀린 경우와 같고, 이 시도는 Rate limit 예산을 깎지 않습니다…" 를 직접 읽었다. 표는 `README.md:183` 에서 시작하고 그 행이 191 번째 줄이다.
- `internal/httpserver/oidc.go:1010-1029`(세 Endpoint가 더는 같게 답하지 않는다는 주석 + 세 답), `:1042-1053`(`clientAuthUndecided` 본문 — `ErrNotFound` 제외, context 취소 시 지표/로그 없이 반환, `stage` 라벨), `:1057-1070`(`writeClientAuthError` 는 401/429 를 쓰는 **별개** 헬퍼), 호출부 `:520`·`:1288`·`:1458`.
- `docs/operations.md:90` 의 `resso_client_auth_errors_total` 불릿 — 세 답(500/503/200 `active=false`)과 `stage` 두 값, Rate limit 이력까지 정확히 적혀 있다.
- `cmd/resso/pins_test.go:270-301` — 두 문서에서 같은 규칙의 텔링을 찾아 비교하고 `checked == 0` 이면 `t.Fatal` 하는 선례. `cmd/resso/command_docs_test.go:35`, `pins_test.go:202,273`, `testservices_test.go:165` 가 README 를 읽는 자리들.
- `Makefile:63-64` 의 `PATH="$(go env GOPATH)/bin:$PATH"` 두 줄과 `:77` 의 수정된 `test_env=…` 안내.

**미확인(구현자가 직접 확인할 것)**
- **이번 정찰에서 명령을 하나도 실행하지 못했다.** 이 세션의 Bash 가 작업 디렉터리 밖 접근과 복합 명령을 막아 `go test`·`go version`·`golangci-lint` 를 전혀 돌리지 못했다. 위 검증 명령은 **과거 회차 원장에서 실제로 돌았던 것**을 옮긴 것이며 이번에 재확인한 결과가 아니다.
- README 표 행을 `\n\n` 블록 분할로 잡을 수 있는지(표 전체가 한 블록일 가능성이 높다 — 줄 단위 추출을 권한다). 실제 분할 결과는 구현자가 확인할 것.
- `webui/dist/index.html` 이 이번 트리에서 깨끗한지. 회차 시작 시 `git status` 는 clean 이었다(확인), 빌드 후 상태는 미확인.
- 지난 회차가 보고한 `npm audit` 5건·`govulncheck` 미호출 모듈 3건의 **현재** 건수.
