# 과제서 — 2026-09-29 GoalForge

- **과제**: MCP `activity_report` 의 `since` 가 스스로 광고하는 `7d` 를 실제로 거절하는 것을 고친다 — 일/주 단위 기간 파서를 `internal/model` 로 옮겨 CLI 와 MCP 가 같은 것을 읽게 한다 (가치 4 / 위험 1 / 작업량 S~M)

- **왜**: `internal/mcp/tools.go:73` 의 도구 스키마가 `since` 를 `"Duration such as 24h or 7d (default 24h)"` 라고 에이전트에게 광고하는데, 처리기(`internal/mcp/tools.go:657` `activityReport`)는 `time.ParseDuration` 을 쓴다. Go 의 파서는 `d` 를 모르므로 `7d` 는 반드시 `since must be a positive duration such as 24h, got "7d"` 로 거절된다 — 스키마가 시키는 대로 한 에이전트가 오류를 받는다. 어제 머지된 보관(retention) 기능이 CLI 쪽에만 `d`/`w` 를 아는 파서(`cmd/goalforge/main.go:3996 dayDuration`)를 넣었고, 그 커밋 메시지 자체가 "보관 기간을 말할 때 누구나 쓰는 단위" 라고 근거를 밝혔다. 같은 근거가 MCP 에도 그대로 적용되는데 파서가 `package main` 에 갇혀 있어 재사용되지 못했다. 고치면 광고와 동작이 일치하고, 앞으로 기간을 받는 자리가 한 곳을 공유하게 된다.

- **수용 기준**
  1. MCP `activity_report` 에 `since="7d"` 를 주면 오류 없이 7일 창으로 조회된다. `"2w"`, `"24h"`, `"90m"` 도 통과한다.
  2. `since="0h"`, `"-3d"`, `"soon"` 은 여전히 거절되고, 오류 문구가 받아들이는 형식(예: `24h`, `7d`)을 말해 준다. (양수 요구는 현재 동작 — `parsed <= 0` — 이므로 유지할 것. 이 축이 회귀하면 활동 보고가 미래 창을 조회한다.)
  3. CLI `storage prune --older-than 30d|12w|720h|1.5d` 의 기존 동작이 그대로다 — `cmd/goalforge/duration_test.go` 의 `TestDurationAcceptsDaysAndWeeks` / `TestUnreadableDurationSaysWhatItAccepts` 가 통과한다(위임으로 바꾸든 호출부를 바꾸든 이 두 테스트는 계속 통과해야 한다).
  4. `internal/model` 이 더는 `[no test files]` 가 아니다 — 새 파서의 단위 테스트가 표(테이블) 형태로 `d`/`w`/표준 단위/거절 케이스를 덮는다.

- **건드릴 파일** (프로덕션 3개 + 테스트 2~3개)
  - `internal/model/duration.go` — **신규**. `cmd/goalforge/main.go:3990-4015` 의 `dayDuration` 본문을 그대로 옮긴 공개 함수(이름 제안: `ParseWindow(value string) (time.Duration, error)`). `internal/model` 은 `fmt`/`strings`/`time` 만 쓰는 잎 패키지라 순환 의존이 없고, `internal/mcp` 와 `cmd/goalforge` 둘 다 이미 `model` 을 임포트한다(`internal/mcp/tools.go:301`, `cmd/goalforge/main.go:1594`). `strconv`/`errors` 임포트 추가가 필요하다.
  - `cmd/goalforge/main.go:3996 dayDuration` — 본문을 `return model.ParseWindow(value)` 한 줄로 줄이거나(가장 작은 diff, 기존 테스트 그대로 통과) 호출부 `storagePrune`(:3931)에서 직접 `model.ParseWindow` 를 부르고 `dayDuration` 을 지운다(후자면 `duration_test.go` 도 같이 고칠 것). **`storagePrune`:3935 의 `window <= 0` 가드는 지우지 말 것** — 보관 정리가 음수 창으로 도는 것을 막는 유일한 자리다.
  - `internal/mcp/tools.go:654 activityReport` — `time.ParseDuration(args.Since)` 를 `model.ParseWindow(args.Since)` 로 바꾼다. `parsed <= 0` 검사는 유지. 오류 문구를 받아들이는 단위와 맞춘다(`24h`, `7d`, `2w`). :73 의 스키마 설명은 이제 참이 되므로 그대로 둬도 되고, `2w` 를 덧붙여도 된다.
  - `internal/model/duration_test.go` — **신규**. 위 수용 기준 4.
  - `internal/mcp/server_test.go` (기존 파일, 회귀 케이스 추가) 또는 신규 `internal/mcp/activity_test.go` — `since="7d"` 가 받아들여지는 것을 처리기 수준에서 고정. 기존 `server_test.go` 가 `Server` 를 어떻게 세우는지 먼저 읽고 그 패턴을 따를 것(정찰은 파일 목록만 확인했고 내부 구조는 **미확인**).
  - `cmd/goalforge/duration_test.go` — `dayDuration` 을 지우는 선택지를 고른 경우에만.

- **검증 명령** (이 저장소에서 실제로 도는 것)
  ```
  go build ./...
  go vet ./internal/model/ ./internal/mcp/ ./cmd/goalforge/
  go test ./internal/model/ ./internal/mcp/ -v
  go test ./cmd/goalforge/
  go test ./...        # 약 25초
  ```

- **위험과 피할 것**
  - **기준선이 전부 통과가 아니다.** 이 워크트리(main@dc28e64, 트리 깨끗)에서 방금 찍은 실측: `internal/mcp` ok, `internal/model` `[no test files]`, **`cmd/goalforge` 2건 실패** — `TestCLIFullLifecycle`(main_e2e_test.go:115, 커밋 저자가 `hkjang` 으로 잡힘)과 `TestRestoreVerifiesRecordsAndSettlesOutsideWork`(main_e2e_test.go:576, `fatal: 'DISABLED' does not appear to be a git repository`). 둘 다 환경 사유이고 이 과제와 무관하다. 시작 전에 직접 기준선을 한 번 찍고 비교할 것.
  - 저 두 실패를 고치려 하지 말 것. 저자 신원 건은 2026-09-28 회차가 이미 수정을 냈고 아직 머지 전(review-pending)이라 base 에 없다 — 다시 하면 충돌한다. push 실패는 하네스가 `GIT_CONFIG_COUNT`/`remote.origin.pushurl=DISABLED` 를 주입해 push 를 일부러 막은 것이다. **우회 금지.**
  - `internal/notify/suppress.go:52`(`EnvRepeatWindow` 환경변수)와 `internal/api/setup.go:285,304` 도 `time.ParseDuration` 을 쓴다. **이번 회차 범위 밖이다** — 파일 예산을 지키고, 환경변수/HTTP 설정은 기본값 폴백 의미가 달라 별도 판단이 필요하다. 아이디어 파일에 남겨 뒀다.
  - `internal/store/sqlite/store.go` 스키마, `internal/policy/*`, `internal/gitops/commit.go`, `.github/` 는 건드리지 말 것(보호 경로).
  - `ParseWindow` 의 접미사 판정은 `strings.CutSuffix` 라 `"1d12h"` 같은 복합 표기는 `d` 분기에 걸리지 않고 `time.ParseDuration` 으로 넘어가 거절된다. 기존 CLI 동작과 같으므로 **고치지 말고 그대로 옮길 것** — 동작 확장은 이 과제가 아니다.
  - 커밋 메시지는 이 저장소 관례대로 무엇이 **달라지는지**를 한국어 한 줄로(최근 커밋: "상태 데이터베이스가 무한히 자라던 것을 정리할 수 있게 한다"). 브랜치는 `improve/<주제>`.

- **차선 후보**: `internal/model/model.go:94 ParseCriterion` 회귀 테스트 추가 (가치 2 / 위험 1 / 작업량 S). `internal/model` 이 `[no test files]` 이고 `ParseCriterion` 은 CLI 두 곳(`main.go:1594`, `:3347`)과 MCP(`tools.go:301`)에서 사용자 입력을 받는 유일한 파서인데 계약이 고정돼 있지 않다. 덮을 것: `"a=b"`, `"a@journey=b"`(kind 소문자화), `"=b"`/`"a="`/`"ab"`(거절), `"a@=b"`/`"@j=b"`(거절), `"a@b@c=d"`(kind 가 `b@c` 가 되는 현재 동작 — 잘못된 kind 는 `internal/store/sqlite/store.go:565` 의 `policy.ValidGateKind` 가 뒤에서 거절하므로 파서에서 고칠 필요 없음. 현재 동작을 기록만 할 것). 1순위와 파일이 겹치므로(둘 다 `internal/model`) 둘 중 하나만 할 것.
