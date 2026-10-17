## 2026-10-17
- 선택: MCP 정수 인자의 조용한 절삭·오해석 차단 (`internal/tools/args.go` toInt 엄격화) (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: MCP 로 들어오는 JSON 숫자는 전부 float64 라 `toInt` 가 소수부를 그대로 절삭했고, 문자열 경로의 `fmt.Sscanf` 는 `"42 (최신)"` 의 뒤따르는 문자를 무시하고 42 를 돌려줬다. 이 값이 `schema.go:resultTarget` 을 거쳐 `buildNumber` 가 되므로 조회·중지·릴리스가 조용히 다른 빌드로 나갈 수 있었다. 실수는 정수로 정확히 표현될 때만, 문자열은 `strconv.Atoi` 로 전체가 10진 정수일 때만 허용하도록 고치고 NaN·무한대·int 범위 초과도 거부했다(`json.Number` 도 같은 규칙). 검증: 도커로 임시 Postgres 를 띄워 `TEST_DATABASE_URL` 을 준 상태로 `go build ./... && go vet ./... && go test ./... -count=1 -p 1` 전부 통과 — DB 의존 테스트가 실제로 돌았다(`internal/tools` 0.005s → 5.4s).
- 실패 재현: `args_test.go:34: 소수 123.7 을 123 로 조용히 절삭했습니다` / `args_test.go:57: "123 (최신)" 을 123 로 받아들였습니다` / `args_test.go:41: int 범위를 넘는 1e30 을 -9223372036854775808 로 받아들였습니다`. 실행기 전체를 지나는 통합 테스트도 수정 전 코드로 되돌려 재현 확인: `integration_test.go:554: buildNumber=42 (최신) 가 거부되지 않았습니다: <nil>`(즉 42번 빌드 결과가 그대로 반환됐다).
- 보류 아이디어: (1) `read.go:190,482` 의 raw start/limit 을 Bamboo 로 넘기기 전 클램프 — 현재 음수·거대값이 그대로 전달됨 (가치 3/위험 2/S). (2) `internal/approval`·`internal/operation` 단위 테스트 공백 보강 — 승인 상태 전이가 전혀 테스트되지 않음 (가치 4/위험 2/M). (3) `go.mod` 의 직접 의존성(chi, go-oidc, golang-jwt, uuid)이 `// indirect` 로 잘못 표기된 것 정정 (가치 2/위험 1/S). (4) `Args.OptBool` 도 `"true"`/`"1"` 문자열을 받지 못하고 조용히 기본값으로 빠짐 — Int 와 같은 수준의 명시적 처리 필요 (가치 2/위험 1/S).
- 과제서: 채택 — 정찰이 지목한 `toInt` 절삭이 실제 결함이었고, 확인 못 했다던 `Sscanf` 의 뒤따르는 문자 허용도 재현으로 확정됐다.

## 2026-10-17
- 선택: 해석 못한 원천 빌드 키를 권한 제외(filteredCount)로 집계하지 않음 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 과제서의 CI 실패는 **저장소 밖 원인으로 확정**했다 — 체크런 4개가 2~3초 만에 스텝 실행 전 실패했고, 이 환경 저장소 28개의 공개/비공개 여부와 CI 결과가 **28/28 정확히 일치**한다(비공개 10개 전부 2~3초 startup 실패, 공개 18개 전부 스텝 실행). 쌍둥이 저장소 bbmcp 는 같은 분(22:14:39Z)에 **동일한 액션 버전(checkout@v4·setup-go@v5·setup-node@v4)** 으로 통과했으므로 앞 회차 수리의 "deprecated 액션" 가설은 반증됐다. GitHub Actions 는 공개 저장소만 무료이고 비공개는 분(minutes) 과금이므로 원인은 **계정 Actions 분 소진/지출 한도**다. 저장소에서 고칠 것이 없어(워크플로 느슨화는 금지) 차선으로, 앞 회차 비평이 지목하고 미수정으로 남긴 실제 결함을 고쳤다: `bamboo_release_context` 가 `splitResultKey` 해석 실패를 권한 제외와 같은 분기에서 `c.Filtered++` 로 집계해, Bamboo 가 예상 밖 키를 주면 배포 전 점검에서 원천 빌드가 조용히 빠지고 원인이 "권한 부족" 으로 오보고됐다. 해석 실패는 경고로 분리했고 제외 자체는 유지해 fail-closed 를 지켰다. 검증: 임시 Postgres 로 `TEST_DATABASE_URL` 을 준 `go build ./... && go vet ./... && go test ./... -count=1 -p 1` 전부 통과(`internal/tools` 7.711s — DB 통합 테스트 실제 실행), Go 1.24 컨테이너 빌드·vet 통과, CI 4개 잡(go·web·plugin·image + 기동 점검) 전부 로컬 재현 통과.
- 실패 재현: `integration_test.go:642: 파싱 실패를 권한 제외로 집계했습니다: filteredCount=1` / `integration_test.go:652: 해석 실패를 알리는 경고가 없습니다: []`. 수정 후 통과, 수정만 되돌리면 같은 두 줄로 다시 실패(인과 확인).
- 보류 아이디어: (1) `read.go:190,482` raw start/limit 클램프 — **기각**: `client.max()`(≤500, 0→PageSize)와 `pageQuery()`(start≤0 생략)가 프로덕션 배선에서 이미 클램프하므로 실패하는 테스트를 만들 수 없다. (2) `internal/approval`·`internal/operation` 단위 테스트 공백 보강(가치 4/위험 2/M). (3) `go.mod` 직접 의존성 `// indirect` 오표기 정정(가치 2/위험 1/S) — `-mod=mod` 가 붙은 명령마다 diff 가 튄다. (4) `Args.OptBool` 의 `"true"`/`"1"` 문자열 처리 명시(가치 2/위험 1/S). (5) `approval.Arguments` 평문 JSON 저장 경로에 logmask 적용 여부 점검(가치 3/위험 2/S).
- 과제서: 기각 — 정찰은 과제서를 남기지 못했고, 배정된 CI 실패는 두 번의 독립 재현(앞 회차 수리 + 이번 회차)에서 로컬 4개 잡 전부 통과하며 공개/비공개 28/28 상관으로 계정 Actions 과금 문제로 확정됐다. 저장소 쪽에 고칠 결함이 없다.

