## 2026-10-17
- 선택: MCP 정수 인자의 조용한 절삭·오해석 차단 (`internal/tools/args.go` toInt 엄격화) (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: MCP 로 들어오는 JSON 숫자는 전부 float64 라 `toInt` 가 소수부를 그대로 절삭했고, 문자열 경로의 `fmt.Sscanf` 는 `"42 (최신)"` 의 뒤따르는 문자를 무시하고 42 를 돌려줬다. 이 값이 `schema.go:resultTarget` 을 거쳐 `buildNumber` 가 되므로 조회·중지·릴리스가 조용히 다른 빌드로 나갈 수 있었다. 실수는 정수로 정확히 표현될 때만, 문자열은 `strconv.Atoi` 로 전체가 10진 정수일 때만 허용하도록 고치고 NaN·무한대·int 범위 초과도 거부했다(`json.Number` 도 같은 규칙). 검증: 도커로 임시 Postgres 를 띄워 `TEST_DATABASE_URL` 을 준 상태로 `go build ./... && go vet ./... && go test ./... -count=1 -p 1` 전부 통과 — DB 의존 테스트가 실제로 돌았다(`internal/tools` 0.005s → 5.4s).
- 실패 재현: `args_test.go:34: 소수 123.7 을 123 로 조용히 절삭했습니다` / `args_test.go:57: "123 (최신)" 을 123 로 받아들였습니다` / `args_test.go:41: int 범위를 넘는 1e30 을 -9223372036854775808 로 받아들였습니다`. 실행기 전체를 지나는 통합 테스트도 수정 전 코드로 되돌려 재현 확인: `integration_test.go:554: buildNumber=42 (최신) 가 거부되지 않았습니다: <nil>`(즉 42번 빌드 결과가 그대로 반환됐다).
- 보류 아이디어: (1) `read.go:190,482` 의 raw start/limit 을 Bamboo 로 넘기기 전 클램프 — 현재 음수·거대값이 그대로 전달됨 (가치 3/위험 2/S). (2) `internal/approval`·`internal/operation` 단위 테스트 공백 보강 — 승인 상태 전이가 전혀 테스트되지 않음 (가치 4/위험 2/M). (3) `go.mod` 의 직접 의존성(chi, go-oidc, golang-jwt, uuid)이 `// indirect` 로 잘못 표기된 것 정정 (가치 2/위험 1/S). (4) `Args.OptBool` 도 `"true"`/`"1"` 문자열을 받지 못하고 조용히 기본값으로 빠짐 — Int 와 같은 수준의 명시적 처리 필요 (가치 2/위험 1/S).
- 과제서: 채택 — 정찰이 지목한 `toInt` 절삭이 실제 결함이었고, 확인 못 했다던 `Sscanf` 의 뒤따르는 문자 허용도 재현으로 확정됐다.

