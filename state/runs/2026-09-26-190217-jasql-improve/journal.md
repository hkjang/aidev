# 회차 노트 2026-09-26-190217-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:02] base pinned — main@32a8811
- [러너 19:02] autonomy release — 

## 정찰 노트
- gofmt 드리프트 정리 + CI gofmt 게이트를 골랐다. 3 회차 연속 1 순위로 적혔는데 아무도 안 했고, `git log 2bf4e68..HEAD -- internal/oracle` 이 비어 있어(마지막 수정 커밋 `c707512`, v0.6.0) 2026-09-25 의 `gofmt -l` 결과가 그대로 유효하다. 파일 3 개(2 개는 포맷 전용)로 끝난다.
- 제친 후보: 골든 평가 하한 고정(M, `data/kcb` 읽는 무거운 테스트 + 하한 설정 판단이 필요해 45 분 초과 위험) → 차선으로 남김. 한국어 접미사 그림자 규칙 추가 조사는 목록 전체를 손으로 점검해 남은 쌍이 없음을 확인하고 `rejected` 로 내렸다. `caf7a00`·`c712da1`·`1507e00` 영역은 이번에도 미머지라 회피.
- 추측으로 적은 것: **드리프트 존재를 이번 세션에서 직접 재현하지 못했다** — Bash 승인 게이트로 `gofmt` 실행이 거부됐다. 과제서에 "먼저 `gofmt -l ./cmd ./internal` 로 확인하라, 비어 있어도 게이트 추가만으로 과제는 성립" 이라고 명시했다. CI 실행 시간·테스트 소요 숫자도 이전 회차 값이다.
- 구현자가 조심할 것: 포맷 수정과 게이트 추가를 **같은 PR** 에 담을 것(게이트만 먼저 넣으면 첫 실행부터 빨갛다). `gofmt -l` 은 드리프트가 있어도 exit 0 이라 출력 비어 있음을 직접 검사해야 한다. `internal/oracle` 의 `sqlguard.go`·`manager.go` 는 열지 말 것 — 포맷 커밋에 로직이 섞이면 되돌려진다.
- ci.yml 은 빌드 경로라 단계 추가 외에는 손대지 말 것. 게이트가 실제로 잡는지는 문자열 검사가 아니라 일부러 파일을 깨뜨려 exit 1 을 보고 원복하는 방식으로 증명할 것.
- [러너 19:09] scout done — gofmt 드리프트를 정리하고 CI 에 `gofmt -l` 게이트를 추가해 재드리프트를 막는다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `gofmt -w` 로 `internal/oracle/profile.go`·`oracle_test.go` 정렬 드리프트를 정리하고, `.github/workflows/ci.yml` 의 `go test` 뒤에 gofmt 단계를 추가해 재드리프트를 CI 가 잡게 했다(a0dd8b6, 프로덕션 3 파일). 둘을 같은 커밋에 담아 게이트 첫 실행이 green 이다.
- 게이트는 문자열 검사가 아니라 ci.yml 에서 `run` 블록을 파싱해 추출한 뒤 실제로 돌려 증명했다: 깨끗한 트리 exit 0 → `cmd/jasql-eval/main.go` 를 일부러 깨뜨려 exit 1 + 파일 목록 → 원복 후 exit 0. `gofmt -l` 은 드리프트가 있어도 exit 0 이라는 점을 수정 전 실행으로 직접 확인했고, 그래서 종료코드가 아니라 출력 공백 여부를 검사한다.
- 확신 없는 곳: **GitHub Actions 러너에서의 실제 실행은 검증 못 했다** — 이 세션은 push 권한이 없어 로컬 셸(`sh`)로만 돌렸다. 러너의 `bash -e` 기본 셸과 POSIX `sh` 차이는 이 스니펫 범위(`$( )`, `[ -n ]`, `exit 1`)에선 없다고 판단했으나 실측은 아니다. `actionlint`(go run, exit 0)가 shellcheck 까지 돌려 경고 없음은 확인했다.
- 일부러 하지 않은 것: `internal/oracle` 의 `sqlguard.go`·`manager.go`·`explain.go` 는 열지 않았다(포맷 커밋에 로직 혼입 방지). ci.yml 의 트리거·`permissions`·`runs-on`·`timeout-minutes`·`setup-go` 와 기존 3 단계 순서는 손대지 않았다 — diff 는 순수 추가 11 줄. 차선 후보(골든 지표 하한 고정)는 범위 밖으로 남겼다.
- 다음 역할이 조심할 것: Go 변경은 공백 전용임을 `git diff -w` 빈 출력로 증명했으니 로직 리뷰는 불필요하다. `go test ./...` 에서 `internal/oracle` 이 `(cached)` 로 나와 변경 패키지가 실제로 안 돈 것처럼 보였다 — `-count=1 -v` 로 9 개 테스트 통과를 따로 찍었다. 이후 회차가 `./cmd`·`./internal` 밖(예: 루트)에 `.go` 를 추가하면 게이트 범위를 함께 넓혀야 한다(현재 최상위 Go 디렉터리는 `cmd`, `internal` 둘뿐 — 이번 회차 `git ls-files` 실측).
- [러너 19:14] brief accepted — 채택 — 과제서가 "직접 재현 못 했다" 고 남긴 드리프트를 첫 명령으로 재현했고(두 파일 그대로), 수용 기준 1~3 을 그대�
- [러너 19:14] verify passed — 검증 3개 통과 (auto)
- [러너 19:14] pr created — https://github.com/hkjang/jasql/pull/9
- [러너 19:14] guard held — .github/workflows/ci.yml 
