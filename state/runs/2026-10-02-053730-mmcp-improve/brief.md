# 과제서 (2026-10-02 정찰)

- 과제: `mattermost.User.DisplayName()` 이 한글 여부를 보지 않고 성+이름을 붙여 비한국어 이름을 "DoeJohn" 으로 만드는 것을 고치고, `internal/mattermost` 의 첫 테스트 파일을 추가 (가치 4 / 위험 2 / 작업량 S)

## 왜
이 저장소는 사람 이름을 만드는 경로가 두 개인데 계약이 서로 다르다. `internal/server/oidc.go:371 koreanName()` 은 `family_name` 의 첫 글자가 한글 음절(U+AC00–U+D7A3)일 때만 `family+given` 으로 뒤집고 아니면 `name` 클레임을 그대로 쓰며, 이 계약은 `TestKoreanName` 이 "John Doe" → "John Doe" 로 못 박고 있다. 반면 `internal/mattermost/client.go:160 User.DisplayName()` 은 한글 검사 없이 **항상** `LastName + FirstName` 을 구분자도 없이 붙이므로, Mattermost 계정의 FirstName="John"·LastName="Doe" 는 `"DoeJohn"` 이 된다.
이 값은 표시용으로만 쓰이지 않고 DM 미리보기(`dm.go:198` `recipient_display_name`), 승인 미리보기(`tools_mm.go:1309` `user_display_name`), 검색 결과(`tools_mm.go:265`), 그리고 `identity.go:386-387` 에서 `m.MMDisplayName` 으로 **DB에 저장**된다. 즉 AI 클라이언트가 사용자에게 "DoeJohn 에게 DM을 보낼까요?" 라고 물어보게 되고, 승인 화면의 이름이 깨진다. 고치면 두 경로가 같은 입력을 같은 규칙으로 읽는다.

## 수용 기준
1) `FirstName="John", LastName="Doe"` 인 `mattermost.User` 의 `DisplayName()` 이 `"John Doe"`(공백 포함, 라틴 어순)를 돌려준다. `"DoeJohn"` 이 아니다.
2) `FirstName="형국", LastName="장"` 은 지금처럼 `"장형국"`(한국 어순, 공백 없음)을 유지한다 — 한국어 동작은 **회귀하지 않는다**.
3) 이름이 둘 다 비어 있으면 `Nickname` 으로 떨어지는 현재 동작이 유지된다. 셋 다 비면 `""`.
4) `internal/mattermost/client_test.go` 가 새로 생겨 위 세 경우를 표로 검증하고, 같은 입력(given/family)에 대해 `server.koreanName()` 과 `User.DisplayName()` 이 **같은 읽기 결과**를 내는지 두 경로를 함께 확인한다(운영자 규칙: 한쪽만 고치지 말 것). 패키지가 다르므로 교차 검증 테스트는 `internal/server/unit_test.go` 에 두는 쪽이 자연스럽다 — 구현자가 판단.
5) `go test ./...` 가 전부 통과하고, `dm.go:79` 의 이름 매칭이 깨지지 않는다(아래 "위험" 참고).

## 건드릴 파일 (프로덕션 1개 + 테스트 1~2개)
- `internal/mattermost/client.go:160 func (u User) DisplayName()` — 한글 음절 판정을 넣어 분기. `koreanName()` 과 같은 범위(U+AC00–U+D7A3)를 쓰되 **두 함수를 합치려 하지 말 것**(입력 계약이 다르다: 하나는 OIDC 클레임 맵, 하나는 MM 사용자 구조체). 참고로 현재 163행 `TrimSpace(FirstName+" "+LastName)` 분기는 도달해도 항상 빈 문자열이다(161행이 빈 문자열이면 두 필드가 모두 공백이므로) — 고치면서 이 죽은 분기가 실제로 라틴 어순을 담당하게 만들면 된다.
- `internal/mattermost/client_test.go` — **신규**. 이 패키지는 현재 테스트 파일이 0개다(`go test ./...` 가 `[no test files]` 로 보고). `DisplayName` 표 테스트 + 여력이 있으면 `httptest.Server` 에 실제 `*Client`(`mattermost.New`)를 붙여 ① 비2xx 응답이 `*APIError` 로 매핑되고 `StatusOf(err)` 가 그 상태코드를 돌려주는지 ② `message` 가 빈 본문이면 `http.StatusText` 로 대체되는지 ③ `New("https://mm.example.com/api/v4")` 의 `BaseURL()` 이 `/api/v4` 를 떼어 내는지까지 추가. 손으로 만든 대역이 아니라 실제 타입·실제 HTTP 배선으로 검증한다.
- (선택) `internal/server/unit_test.go` — 위 4)의 교차 검증 한 케이스.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
go test ./internal/mattermost/        # 새 테스트
go test ./...                         # 전체 (Postgres 없으면 통합 테스트는 자동 skip)
gofmt -l .                            # 출력이 비어 있어야 한다
go vet ./...
```
- 전체 통합 테스트까지 돌리려면 `TEST_POSTGRES_DSN=postgres://... go test -race -count=1 ./...` (CI 가 하는 것). `internal/server/integration_test.go:192` 가 이 변수가 없으면 `t.Skip` 한다 — 로컬에서 `ok ... 0.005s` 가 나오면 통합 테스트는 **돌지 않은 것**이다. 수용 기준 5)는 이 DSN 을 띄워서 확인하는 쪽이 안전하다. (로컬에 Postgres 가 떠 있는지는 **미확인**.)
- `make test` 는 web vitest 도 돌린다. 이번 변경은 프런트를 건드리지 않으므로 Go 쪽만 돌려도 된다.

## 위험과 피할 것
- **가장 큰 위험: `internal/server/dm.go:79`.** 수신자 이름 매칭이 `[]string{FirstName+LastName, LastName+FirstName, Nickname, DisplayName()}` 를 후보로 돌린다. 이 목록에 `LastName+FirstName` 가 **이미 따로 들어 있으므로** `DisplayName()` 의 반환이 바뀌어도 매칭 범위는 줄지 않는다(확인함). 그래도 이 줄을 같이 수정하지 말 것 — 매칭은 `norm()` 으로 공백을 제거해 비교하므로 공백이 생겨도 영향이 없다.
- `internal/server/identity.go:386-387` 은 이 값을 `identity_map.mm_display_name` 으로 **저장**한다. 기존 행은 다음 갱신 때까지 옛 값을 유지한다 — 마이그레이션으로 기존 데이터를 고치려 하지 말 것(`internal/store/migrations` 는 보호 경로).
- `koreanName()`(oidc.go)과 `DisplayName()`을 공용 헬퍼로 통합하지 말 것. 운영자 규칙: 계약이 다른 파서는 통합하지 않는다. 각자 같은 한글 범위를 쓰되 따로 둔다.
- `internal/server/` 의 auth·oauth·oidc·identity 의 **로직**은 건드리지 말 것(테스트 추가는 예외). `.github/workflows`, `scripts/package-offline.sh`, `VERSION` 도 건드리지 말 것 — 릴리즈 경로다.
- 소스 문자열 grep 을 증거로 제출하지 말 것. 반드시 테스트 실행 출력으로 보일 것.

## 차선 후보
`internal/logbuf` 링 버퍼 테스트 추가 (가치 3 / 위험 1 / S). 129줄, 테스트 0개. `New(size)` → `add` 로 용량 초과 시 오래된 항목을 버리는지, `Snapshot()` 이 오래된 순서로 돌려주는지, `Subscribe()` 가 돌려주는 해제 함수를 호출한 뒤 채널에 더 쓰지 않는지(구독자 누수·블로킹)를 `go test -race ./internal/logbuf/` 로 검증. 1순위가 "의도된 한국 어순" 으로 판단되어 성립하지 않을 때 이것을 고를 것.
