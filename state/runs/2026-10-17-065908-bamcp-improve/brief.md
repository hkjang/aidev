# 과제서 — 2026-10-17-065908-bamcp-improve

- **과제**: MCP 도구 인자 정수 변환(`internal/tools/args.go` 의 `toInt`) 엄격화 + `internal/tools` 첫 단위 테스트 신설 (가치 4 / 위험 1 / 작업량 S)

- **왜**: `Args.Int`/`Args.OptInt` 는 `toInt` 를 쓰는데, `case float64: return int(t), true` 이므로 JSON 숫자 `12.9` 가 조용히 `12` 이 된다. MCP 클라이언트(LLM)가 보내는 JSON 숫자는 전부 `float64` 로 디코드되므로 `analysis.go:116,120` 의 `a.Int("baseBuildNumber")`·`a.Int("headBuildNumber")` 가 **사용자가 요청하지 않은 빌드를 비교하고도 오류를 내지 않는다**. 문자열 분기도 `fmt.Sscanf(s, "%d", &n)` 로 읽어 전체 소비를 요구하지 않는다(= `"12abc"` 류를 받아들일 것으로 보임 — 아래 "미확인" 참조). 게다가 `internal/tools` 패키지에는 DB 없이 도는 단위 테스트가 **하나도 없다**(기존 `integration_test.go`·`testdb_test.go` 는 `TEST_DATABASE_URL` 이 없으면 전부 skip). 순수 함수에 테스트를 깔고 변환을 엄격하게 바꾸면, 잘못된 인자가 승인 스냅샷·Bamboo 호출로 흘러가는 대신 호출 시점에 거부된다.

- **수용 기준**:
  1. `internal/tools/args_test.go` 가 새로 생기고 `go test ./internal/tools/ -count=1 -run 'TestArgs|TestToInt'` 가 DB 없이 통과한다. 테스트는 **표 기반(table-driven)** 으로 `String`/`OptString`/`Int`/`OptInt`/`OptBool`/`StringSlice` 를 덮는다.
  2. `toInt` 가 다음을 **거부**한다(`ok == false`): 정수가 아닌 부동소수(`12.9`, `-3.7`), 뒤에 쓰레기가 붙은 문자열(`"12abc"`, `"12 34"`, `"1e3"`), 빈 문자열, `int` 범위를 벗어나는 부동소수(`1e19`). 다음은 **계속 받아들인다**: `int`/`int32`/`int64`, 정수값 `float64`(`12.0`, `0`, `-5`), 공백만 감싼 10진 문자열(`" 7 "`), `"0"`, 음수 문자열(`"-5"`).
  3. 거부 시 `Args.Int` 는 기존 메시지 `인자 %s 는 정수여야 합니다` 를 그대로 반환하고, `Args.OptInt` 는 기존대로 `def` 를 반환한다(시그니처·에러 문구 변경 없음).
  4. 테스트가 증명해야 하는 것: **느슨한 변환이 조용히 성공하지 않는다**는 것. 즉 `Args{"n": 12.9}.Int("n")` 는 `err != nil` 이고, `Args{"n": "12abc"}.OptInt("n", 25)` 는 `25` 다.
  5. `go test ./internal/tools/ ./internal/permission/ -count=1` 이 계속 통과한다(회귀 없음).

- **건드릴 파일** (프로덕션 2개 + 테스트 1개):
  - `internal/tools/args.go:toInt` — `float64`/`float32` 분기에 정수성·범위 검사 추가, `string` 분기를 `strconv.Atoi` 로 교체(전체 소비 요구). `fmt` 임포트가 더는 필요 없으면 정리(에러 포매팅에 여전히 쓰이므로 남을 것).
  - `internal/tools/args_test.go` — **신규**. `package tools`(내부 테스트)로 만들면 `toInt` 를 직접 테스트할 수 있다. 기존 테스트 파일은 `package tools_test` 이므로 두 패키지가 공존하는 형태가 된다(Go 는 허용).
  - `internal/tools/analysis.go:116-131` — `baseBuildNumber`/`headBuildNumber` 가 `<= 0` 이면 거부(한 줄짜리 가드 2곳). 이것까지만 하고 다른 핸들러는 건드리지 말 것.

- **검증 명령** (이 저장소에서 실제로 확인함):
  ```
  go test ./internal/tools/ ./internal/permission/ -count=1      # DB 없이 통과 확인됨 (0.004s / 0.003s)
  go vet ./internal/tools/
  go build ./...
  ```
  CI(`.github/workflows/ci.yml`)는 `go build ./...` → `go vet ./...` → `TEST_DATABASE_URL` 를 주고 `go test ./... -count=1 -p 1` 을 돈다. 이 과제는 DB 의존 테스트를 건드리지 않으므로 로컬에서 위 세 명령으로 충분하다.

- **위험과 피할 것**:
  - **호출부 전수 확인이 선행 조건.** `grep -n '\.Int(\|\.OptInt(' internal/tools/*.go` 로 확인한 결과 대부분은 `clamp(...)` 로 감싸져 있어 엄격화의 영향이 없다. 영향이 있는 곳은 `analysis.go:116,120`(raw `Int`), `read.go:190`, `read.go:482`(raw `OptInt("start")`, `OptInt("limit")` 를 `Adapter.Results`/`Adapter.*` 로 그대로 전달). 이 raw 전달 자체를 클램프하는 것은 **이번 과제 범위가 아니다**(아이디어 목록에 별건으로 올려 둠).
  - `OptInt` 의 기본값 복귀 동작을 바꾸지 말 것. 엄격화로 "거부"가 늘면 `OptInt` 는 조용히 `def` 로 가므로 사용자가 보기엔 동작이 유지된다. `Int` 만 에러가 늘어난다.
  - **`internal/tools/schema.go` 의 입력 스키마를 손대지 말 것.** MCP 클라이언트에 노출되는 계약이고, 이 과제와 독립이다.
  - 보호 경로 회피: `internal/api/oauth*.go`, `internal/auth/`, `internal/database/migrations/`, `.github/workflows/` 는 **건드리지 않는다**. 이 과제는 그럴 필요가 없다.
  - `StringSlice` 는 현재 프로덕션 코드에서 호출되는 곳이 없다(grep 무결과). 테스트는 써도 되지만 동작을 바꾸지는 말 것.

- **미확인 (구현자가 테스트로 먼저 확인할 것)**:
  - `fmt.Sscanf(" 12abc ", "%d", &n)` 가 `err == nil` 로 12 를 주는지 **직접 실행해 확인하지 못했다**(샌드박스가 스크래치 프로그램 실행을 거부). 수용 기준 1의 표 기반 테스트를 **먼저 작성해 현재 동작을 기록한 뒤** 엄격화하는 순서로 진행할 것. 만약 Sscanf 가 이미 거부한다면 문자열 분기 변경은 생략하고 `float64` 정수성 검사와 테스트만 남기면 된다(그것만으로도 수용 기준 2·4를 만족한다).
  - `int(float64(1e19))` 의 결과값은 Go 명세상 구현 정의(implementation-defined)다. 테스트는 구체적 값을 단정하지 말고 "거부된다"만 단정할 것.

- **차선 후보**: `internal/permission/capability.go` 의 권한 매핑 단위 테스트 신설 — `KindOf`/`IsRead`/`capsFor`/`fromBambooPlan`/`fromBambooDeployment`/`fromBambooEnvironment`/`newDecision` 은 모두 순수 함수인데 테스트가 없다. 특히 `IsRead` 는 `Resolver.Require` 에서 `fresh`(캐시 우회) 여부를 결정하므로 쓰기 권한이 읽기로 분류되면 **낡은 캐시로 빌드 실행이 승인될 수 있다**. 신규 파일 `internal/permission/capability_test.go` 하나로 끝나고(프로덕션 파일 0개) 위험이 사실상 없다. 1순위가 성립하지 않으면(= 현재 변환이 이미 엄격하면) 이것을 하라.
