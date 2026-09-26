- 과제: gofmt 드리프트를 정리하고 CI 에 `gofmt -l` 게이트를 추가해 재드리프트를 막는다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 2026-09-24·09-25 두 회차가 `gofmt -l ./internal ./cmd` 를 실제로 돌려 `internal/oracle/profile.go`·`internal/oracle/oracle_test.go` 드리프트를 재현했는데 세 회차 연속 "범위 밖" 으로 미뤄졌고, 2026-09-25 에 신설된 `.github/workflows/ci.yml` 은 build→vet→test 만 돌려 포맷을 전혀 보지 않는다. 게이트를 넣으면 포맷 드리프트가 사람 눈 대신 CI 에서 걸리고, 매 회차 정찰이 같은 항목을 다시 보고하는 낭비가 끝난다.
- 수용 기준:
  1) `gofmt -l ./cmd ./internal` 출력이 빈 문자열이다(수정 전 출력과 수정 후 출력을 둘 다 회차 노트에 붙일 것 — 수정 전 목록이 이번 회차의 red 증거다).
  2) `.github/workflows/ci.yml` 에 gofmt 단계가 추가되고, 그 단계가 드리프트를 실제로 감지한다 — 검증 방법: 임의의 `.go` 파일에 공백을 넣어 일부러 깨뜨린 뒤 로컬에서 같은 셸 스니펫을 돌려 exit 1 과 파일 목록 출력을 확인하고, 원복 후 exit 0 을 확인한다(문자열 검사만으로 "게이트가 있다" 고 주장하지 말 것).
  3) 기존 `build-vet-test` 잡의 `go build ./...` → `go vet ./...` → `go test ./...` 순서와 트리거·permissions·timeout 은 그대로다. `go build ./...`·`go vet ./...`·`go test ./...` 가 모두 exit 0.
- 건드릴 파일 (프로덕션 3 개, 그중 2 개는 포맷 전용):
  - `internal/oracle/profile.go` — `gofmt -w` 결과만 반영. 로직·주석 문구·필드 순서는 한 글자도 바꾸지 말 것.
  - `internal/oracle/oracle_test.go` — 위와 동일.
  - `.github/workflows/ci.yml` — `go test` 단계 **뒤에** gofmt 단계 하나 추가. 권장 형태:
    ```yaml
      - name: gofmt
        run: |
          out=$(gofmt -l ./cmd ./internal)
          if [ -n "$out" ]; then
            echo "gofmt drift in:"; echo "$out"; exit 1
          fi
    ```
    (`gofmt -l` 은 드리프트가 있어도 exit 0 이므로 위처럼 출력 비어 있음을 직접 검사해야 한다. `./cmd ./internal` 로 범위를 명시하는 이유: `git ls-files '*.go' | cut -d/ -f1` 결과가 `cmd`, `internal` 뿐이라 루트 `.go` 파일이 없다 — 이번 회차 실측.)
    단계를 **마지막에** 두는 이유: GitHub Actions 는 첫 실패 단계에서 잡을 멈추므로, 앞에 두면 포맷 한 칸 때문에 build·vet·test 신호가 전부 가려진다. 순서를 바꾸고 싶으면 그 판단은 적어 둘 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l ./cmd ./internal` — 수정 전(비어 있지 않아야 함) / 수정 후(비어 있어야 함)
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...` (실측 기준: catalog 55~58 초, mcp 약 10 초, meta·oracle 캐시, cmd 3 개는 no test files)
  - 워크플로 문법: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml'))"` (2026-09-25 회차가 쓴 방법) — 가능하면 `actionlint` 도.
  - `git diff --check`, `git status data/kcb`(무변화 확인)
- 위험과 피할 것:
  - **드리프트가 이미 없을 수 있다.** 이번 정찰 세션은 Bash 승인 게이트로 `gofmt` 실행이 거부되어 드리프트를 직접 재현하지 **못했다**. 근거는 (a) 2026-09-25 회차의 실제 `gofmt -l` 출력, (b) `git log 2bf4e68..HEAD -- internal/oracle` 이 비어 있고 `internal/oracle` 을 마지막으로 건드린 커밋이 `c707512`(v0.6.0) 라는 이번 회차 실측 — 즉 그 이후 아무도 손대지 않았으므로 드리프트는 그대로 남아 있을 것이다. **먼저 `gofmt -l ./cmd ./internal` 을 돌려 확인하라.** 비어 있다면 수용 기준 1 은 "이미 충족" 으로 적고 게이트 추가와 기준 2·3 만 수행하면 과제는 그대로 성립한다(차선 후보로 갈 필요 없음).
  - `.github/workflows/ci.yml` 은 빌드 경로다 — 단계 추가 외에 트리거·`permissions`·`runs-on`·`timeout-minutes`·`setup-go` 설정을 손대지 말 것. 시크릿·외부 배포·새 잡 추가 금지.
  - `internal/oracle` 은 읽기 전용 정책 경계(`sqlguard.go`)와 같은 패키지다. 이번 과제는 `profile.go`·`oracle_test.go` **포맷만** 이다. `sqlguard.go`·`manager.go`·`explain.go` 는 열지도 말 것. 포맷 커밋에 로직 변경을 섞으면 비평에서 되돌려진다.
  - 기본 빌드는 `driverAvailable=false` 다. Oracle 관련 e2e 주장은 하지 말 것 — 이번 과제는 애초에 동작을 바꾸지 않으므로 "동작 무변화" 를 `go test ./...` 로만 주장하면 충분하다.
  - 미머지 '성공' 커밋 3 개(`1507e00` oracle TrimStatement, `caf7a00` eval -verbose MISS, `c712da1` timeparse 순서결정성)가 이번 base(`32a8811`)에 여전히 없다 — 이번 회차 `git merge-base --is-ancestor` 로 실측. 이 세 영역은 열린 PR 과 중복이니 손대지 말 것. 특히 `cmd/jasql-eval/main.go:64` 의 세 bool MISS 조건은 건드리지 말 것.
  - gofmt 게이트가 첫 실행부터 빨개지지 않도록 **포맷 수정과 게이트 추가를 같은 커밋(또는 같은 PR)** 에 담을 것. 게이트만 먼저 넣으면 CI 가 즉시 빨개진다.
- 차선 후보: **골든 평가 지표 하한을 회귀 테스트로 고정** — `cmd/jasql-eval` 이 계산하는 `table_selection_acc`/`column_recall_avg`/`join_path_acc`/`metric_lookup_acc` 를 `internal/catalog` 쪽 테스트에서 하한 단정(각각 ≥0.94 / ≥0.9 / ≥0.94 / ≥1 — 2026-09-25 회차 실측값)으로 고정한다. 지금은 `tokenize` 같은 공용 프리미티브를 건드릴 때마다 구현자가 손으로 `go run ./cmd/jasql-eval` 전후를 비교해야 하고, 잊으면 리콜 하락이 조용히 머지된다. `cmd/jasql-eval/main.go` 의 종료 가드는 `table_selection_acc < 0.7` 하나뿐이라(이번 회차 실측) 0.94 → 0.75 하락도 green 이다. 주의: `data/kcb` 골든셋을 읽는 무거운 테스트라 CI 시간이 늘고, 하한을 현재값에 딱 붙이면 정당한 개선도 빨개진다 — 하한은 "현재값" 이 아니라 "내려가면 안 되는 선" 으로 잡을 것.
