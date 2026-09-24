# 회차 노트 2026-09-25-062952-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:29] base pinned — main@e9f3fb2
- [러너 06:29] autonomy release — 

## 정찰 노트
- CI 를 골랐다: 과거 5 회차의 "성공" 커밋이 pinned base e9f3fb2 에 하나도 없고(timeparse.go:114 map-range 가 그대로 남아 있음), 러너 자신이 "워크플로가 없어 PR 이 전부 CI 검사 없음으로 막힌다"고 [5/1/S] 로 적었다. 코드 결함 후보(glossary 빈 토큰, stripKoreanSuffix '별로' 도달 불가)는 각각 영향 없음·랭킹 위험으로 제쳤다.
- 확신 없는 곳: 이번 회차에 `go build/vet/test` 와 `gofmt` 를 **한 번도 실행하지 못했다**(샌드박스가 승인 없이 go/gofmt 실행을 막음). 현재 HEAD 가 green 이라는 것은 2026-09-22 프로필 기록에 기댄 추측이다. gofmt 드리프트 2 파일도 재확인 못 했다.
- 구현자가 조심할 것: 워크플로를 쓰기 전에 세 명령을 먼저 로컬에서 돌려 green 을 확인할 것. 빨간 게 있으면 `continue-on-error` 로 덮지 말고 사실대로 적고 차선(docs/development.md 동기화)으로 전환.
- 보호 경로 PR 이므로 diff 는 `.github/workflows/ci.yml` 한 파일로 끝낼 것 — gofmt 정리·문서 수정·매트릭스 확장을 같이 넣으면 승인이 막힌다.
- 프로필은 2026-09-22 판이 HEAD(e9f3fb2)와 일치해 다시 쓰지 않았다.
- [러너 06:33] scout done — GitHub Actions CI 워크플로 추가 — build·vet·test 를 PR 마다 자동 실행 (가치 5 / 위험 1 / 작업량 S)

## 구현 노트
- `.github/workflows/ci.yml` 한 파일만 추가했다(커밋 `run build, vet and test on every push and pull request`). push(main)+pull_request 에서 checkout@v4 + setup-go@v5(`go-version-file: go.mod`) 로 Go 를 깔고 build → vet → test 를 이 순서로 돌린다. 러너 PR 이 전부 "CI 검사 없음" 으로 막히던 것을 풀기 위한 변경이다.
- 실행으로 증명한 것: 워크플로를 쓰기 **전에** 현재 HEAD(e9f3fb2)에서 `go build ./...`(exit 0) · `go vet ./...`(exit 0) · `go test ./...`(exit 0, catalog 54.365s / mcp 9.626s / meta·oracle cached / cmd 3 개 no test files) 를 직접 돌려 green 을 확인했다. 워크플로 파일은 `yaml.safe_load` 파싱과 `actionlint`(설치 후 실행, exit 0) 로 확인했다.
- **확신 없는 곳**: GitHub Actions 러너에서 실제로 돌려보지 못했다(이 세션은 push 권한 없음). 특히 (a) ubuntu-latest 러너가 go.mod 의 `1.25.0` 툴체인을 받는 경로, (b) `internal/catalog` 54s 가 러너에서 얼마나 느려질지 — timeout-minutes 15 는 로컬 대비 여유를 본 값이지 측정값이 아니다. 로컬 Go 는 1.26.7 이라 1.25.0 툴체인에서의 vet/test 결과도 미확인이다.
- **일부러 하지 않은 것**: gofmt 게이트(=`gofmt -l ./internal ./cmd` 가 `internal/oracle/oracle_test.go`, `internal/oracle/profile.go` 드리프트를 **실제로 재현** — 넣었으면 첫 실행부터 빨갛다), `-tags oracle`(cgo + Instant Client 필요), PG 서비스 컨테이너/`JASQL_TEST_PG`(unset 이면 통합 테스트 skip), OS·Go 매트릭스, `actions/cache`(setup-go 기본 캐시로 충분), docs/development.md 수정(보호 경로 diff 오염 방지 — 범위 누적 금지).
- **다음 역할이 조심할 것**: 보호 경로 PR 이라 사람 승인이 필요하고 diff 는 의도적으로 33 줄 한 파일이다 — 여기에 gofmt 정리나 문서 수정을 얹지 말 것. 첫 CI 실행이 빨갛게 나오면 `continue-on-error` 로 덮지 말고 원인을 그대로 보고할 것. gofmt 드리프트 2 파일 정리는 별도 회차 과제로 ideas.json 에 pending 으로 남겼다.
- [러너 06:36] brief accepted — 채택 — 근거(`.github` 부재, origin 이 실제 GitHub 저장소, 5 회차의 '성공' 커밋이 pinned base 에 하나도 없음)가 현재 코드와 정
- [러너 06:36] verify passed — 검증 3개 통과 (auto)
- [러너 06:36] pr created — https://github.com/hkjang/jasql/pull/6
- [러너 06:36] guard held — .github/workflows/ci.yml 
