# 회차 노트 2026-10-02-053730-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:37] base pinned — main@60fca20
- [러너 05:37] autonomy release — 

## 정찰 노트
- 이 과제를 고른 이유: 순수 테스트 공백 보강(2순위)보다 **실제 결함**이 낫고, 운영자가 반복해 지적한 "같은 값을 읽는 파서가 둘인데 한쪽만 다르다" 패턴에 정확히 들어맞는다 — `oidc.go:371 koreanName()`(한글 가드 있음, `TestKoreanName` 이 라틴 이름 보존을 못 박음)과 `mattermost/client.go:160 DisplayName()`(가드 없음, 항상 성+이름 연결)이 같은 이름을 다르게 읽는다. 프로덕션 파일 1개 + 테스트 1~2개로 끝나고 Postgres 없이 검증된다.
- 제친 후보: `ChannelMembers` 페이지네이션 누락은 호출자 의도를 못 읽어 타당성 미확인(위험 3). `logbuf` 테스트는 결함 없는 순수 보강이라 차선으로 남겼다.
- 추측으로 적은 것: ① 로컬 Postgres 가용성 **미확인** — `go test ./...` 가 0.005s 로 끝났으므로 통합 테스트는 skip 됐다. ② `LastName+FirstName` 무조건 연결이 "한국어 전용 의도" 일 가능성은 남지만, 같은 저자가 `koreanName()` 에 한글 가드를 넣고 테스트로 고정했으므로 저장소의 계약은 가드 쪽이라고 판단했다.
- 구현자가 조심할 것: `dm.go:79` 의 이름 매칭 후보 목록은 건드리지 말 것(양쪽 어순을 이미 열거하므로 회귀 없음 — 확인함). 두 함수를 공용 헬퍼로 합치지 말 것(입력 계약이 다르다). `identity.go:386` 이 이 값을 DB 에 저장하므로 기존 행을 마이그레이션으로 고치려 들지 말 것.
- [러너 05:42] scout done — `mattermost.User.DisplayName()` 이 한글 여부를 보지 않고 성+이름을 붙여 비한국어 이름을 "DoeJohn" 으로 만드는 �

## 구현 노트
- `mattermost.User.DisplayName()` 이 한글 여부를 보지 않고 항상 `LastName+FirstName` 를 붙여 "John Doe" 를 "DoeJohn" 으로 만들던 것을, `server.koreanName()` 과 같은 한글 음절 범위(U+AC00–U+D7A3)로 분기하도록 고쳤다. 신규 `internal/mattermost/client_test.go` + `internal/server/unit_test.go` 교차 검증. 프로덕션 파일 1개.
- 고치기 전에 실패를 먼저 확인했다: `client_test.go:28: DisplayName() = "DoeJohn", want "John Doe"`, `unit_test.go:102: User{"John" "Doe"}.DisplayName() = "DoeJohn", want "John Doe"`.
- 정찰이 "미확인" 으로 남긴 로컬 Postgres: `mmcp-pg` 컨테이너(127.0.0.1:15499)가 떠 있었다. 통합 테스트가 `DROP SCHEMA public CASCADE` 를 하므로 **개발 DB 를 쓰지 않고** 스크래치 DB `mmcp_itest` 를 만들어 `TEST_POSTGRES_DSN=... go test -race -count=1 ./...` 를 돌리고 끝나고 DROP 했다. `-v` 출력의 SKIP 0건 — 통합 테스트가 실제로 돌았고 그 안에 `mattermost_send_dm` 의 승인·발송 경로가 포함된다(= dm.go:79 매칭이 프로덕션 배선으로 통과).
- 확신 없는 곳: `startsWithHangulSyllable` 은 `koreanName()` 과 똑같이 **첫 음절만** 본다. 한자 성(U+4E00–)이나 'Kim 형국' 같은 혼합 표기는 라틴 어순으로 떨어진다 — 기존 OIDC 경로와 같은 한계를 일부러 그대로 맞췄고, 넓히지 않았다. 또 Nickname 폴백은 예전처럼 TrimSpace 하지 않고 원문을 돌려준다(동작 유지).
- 일부러 하지 않은 것: 두 함수의 공용 헬퍼 통합(입력 계약이 다름 — 운영자 규칙), `dm.go:79` 매칭 목록 수정(이미 양쪽 어순을 열거하므로 불필요), `identity_map.mm_display_name` 의 기존 행 마이그레이션(보호 경로), `New()` 의 query/fragment 보존 버그(범위 밖 — ideas.json 에 pending).
- 다음 역할이 조심할 것: `internal/mattermost` 테스트는 DB 없이 돈다. `internal/server` 의 통합 테스트는 `TEST_POSTGRES_DSN` 이 필요하고 **그 DSN 의 public 스키마를 지운다** — 반드시 스크래치 DB 를 가리킬 것. 프런트·릴리즈 경로는 건드리지 않았다.
- [러너 05:47] brief accepted — 채택 — 과제서의 근거(client.go:163 라틴 분기가 죽은 코드, dm.go:79 매칭이 LastName+FirstName 를 따로 가지고 있어 회귀하지 않�
- [러너 05:47] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
