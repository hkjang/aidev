# PR 처리기 노트 2026-10-17-093709-bamcp-shepherd — bamcp PR #1
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-17-065908-bamcp-improve)
# 회차 노트 2026-10-17-065908-bamcp-improve — bamcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:59] base pinned — main@70ee0f6
- [러너 06:59] autonomy release — 

## 정찰 노트
- `args.go` 를 고른 이유: TODO/FIXME 가 0건이고 권한·OAuth 쪽은 테스트가 두껍다. 반면 `internal/tools` 는 2800줄인데 DB 없이 도는 단위 테스트가 전무하고, 그 안에서 `toInt` 의 `float64` → `int` 절삭은 JSON 숫자가 전부 float64 로 오는 MCP 환경에서 **조용히 잘못된 빌드를 비교하는** 실제 결함이다. 승인·운영 패키지 테스트 공백(아이디어 8)이 가치는 더 크지만 전부 DB 의존 M 이라 제쳤다.
- 추측으로 적은 것: `fmt.Sscanf(" 12abc ", "%d", &n)` 가 err==nil 로 12 를 주는지 **실행 확인 못 했다**(샌드박스가 스크래치 `go run` 을 거부). 과제서는 "테스트를 먼저 써서 현재 동작을 기록한 뒤 엄격화" 순서를 명시했고, Sscanf 가 이미 엄격하면 float64 검사와 테스트만으로 수용 기준이 충족된다.
- 구현자가 조심할 것: `Int`/`OptInt` 호출부 대부분이 `clamp()` 로 감싸져 있어 엄격화의 체감 영향이 작다. 영향 있는 곳은 `analysis.go:116,120` 과 `read.go:190,482` 뿐이고, **`read.go` 의 raw start/limit 클램프는 이번 범위가 아니다**(아이디어 3으로 분리). `internal/tools/schema.go`(MCP 클라이언트 계약)와 `oauth`·`auth`·`migrations` 는 건드릴 필요가 없다.
- 확인한 검증 명령: `go test ./internal/tools/ ./internal/permission/ -count=1` → 둘 다 ok(DB 없이). `go build ./...`·`go vet ./...` 은 이 세션에서 실행 승인을 못 받아 **미확인**이지만 CI 가 쓰는 명령이다.
- [러너 07:04] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `internal/tools/args.go` 의 `toInt` 만 고쳤다(프로덕션 파일 1개). MCP JSON 숫자는 전부 float64 라 소수부가 조용히 절삭됐고, 문자열 경로의 `fmt.Sscanf` 는 `"42 (최신)"` 에서 42 만 떼어 갔다. 이 값이 `schema.go:resultTarget` 을 거쳐 `buildNumber` 가 되므로 **엉뚱한 빌드에 중지·릴리스가 나갈 수 있었다**.
- 확신 없는 곳: 엄격화로 **기존 클라이언트가 깨질 여지**. 웹 콘솔·MCP 클라이언트가 `limit`/`start` 를 어떤 타입으로 보내는지 실제 트래픽으로 확인하지 못했다. `OptInt` 는 거부 시 기본값으로 빠지므로 조회 쪽은 무해하지만, `Int`(analysis.go:116,120 의 base/headBuildNumber)는 이제 에러를 낸다. 숫자·숫자문자열은 그대로 통과하니 정상 클라이언트는 영향 없을 것으로 본다.
- `json.Number` 지원을 추가했지만 이 저장소에 그 경로로 들어오는 호출자는 **없다**(디코더가 `map[string]any`). 방어적으로만 넣었고 테스트도 없다.
- 일부러 하지 않은 것: `read.go:190,482` 의 raw start/limit 클램프(아이디어 1로 분리), `schema.go` 의 MCP 클라이언트 계약 문구, `OptBool` 엄격화. 파일 수를 늘리지 않으려고 뺐다.
- 다음 역할이 조심할 것: `TestMalformedBuildNumberRejected` 는 **DB 가 있어야 돈다**. `TEST_DATABASE_URL` 없이 `go test ./...` 하면 skip 되고 통과처럼 보인다. 이번엔 도커로 임시 Postgres(`postgres:16-alpine`)를 띄워 전체 스위트를 실제로 돌렸다 — `internal/tools` 가 0.005s 가 아니라 5.4s 로 나와야 진짜 돈 것이다.
- [러너 07:08] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 구현자가 "확신 없다" 한 두 곳을 모두 직접 확인했다. (1) 재현: `main` 의 `args.go` 만 되돌린 사본에 새 테스트를 넣어 돌려 원장의 실패 출력을 줄번호까지 그대로 재현했다 — 특히 `integration_test.go:554` 에서 수정 전에는 `buildNumber="42 (최신)"` 가 `te==nil` 로 42번 빌드 결과를 그대로 돌려줬다. 테스트가 바뀐 경로를 실제로 지난다. (2) 호환성: 인자 경로는 `internal/mcp/server.go:290` 하나뿐이고 `toInt` 가 받던 타입 집합은 줄지 않았다(좁아진 건 소수부·NaN·범위 초과·뒤따르는 문자뿐). 정상 클라이언트 영향 없다는 판단에 동의한다.
- 검증도 독립 실행했다. 임시 Postgres 로 `TEST_DATABASE_URL` 을 주고 `go test ./... -count=1 -p 1` 전부 ok, `internal/tools` 5.413s(구현자 주장과 일치). `go build`·`go vet` 통과. 다만 **러너 verify.txt 는 DB 없이 돌아 `internal/tools` 0.007s** 였으니, 러너 게이트만 보면 통합 테스트는 돌지 않은 셈이다. `floatToInt` 의 반개구간 `[-2^63, 2^63)` 과 32비트 `int` 축소 검사도 손으로 따져 봤고 오류 없다.
- 못 본 것: 실서버 Bamboo 6.10.4 호출, 플러그인 JAR, 도커 이미지 기동. 코드 변경이 stdlib 파싱 한 함수라 필요 없다고 봤다.
- 승인이어도 남는 우려 2개(릴리즈 노트에 쓸 것): `json.Number` 분기(`args.go:126`)는 **도달 불가 + 무테스트** 이고 나중에 `UseNumber()` 를 켜면 `1e3`·`123.0` 을 거부해 float 경로와 규칙이 갈린다. 그리고 `analysis.go:397 splitResultKey` 도 같이 엄격해져, Bamboo 가 숫자 아닌 접미사 키를 주면 `analysis.go:258` 에서 **권한 필터(`c.Filtered`)로 집계되며 조용히 빠진다** — 모의 서버로는 안 잡히는, 이 저장소의 상습 실패 부류다. 실서버에서 배포 버전 소스 빌드가 비면 여기를 먼저 보라.
- 다음 회차용: 작업 트리의 `internal/webui/dist/index.html` 수정은 verify 의 `npm run build` 산물이며 추적된 index.html 은 의도된 placeholder(Dockerfile 이 재빌드) — PR diff 에 없으니 같이 커밋하지 말 것. 보류 아이디어 (1)(`read.go:190,482` raw start/limit 클램프)은 `OptInt` 가 잘못된 값을 조용히 기본값으로 바꾸는 문제와 같은 뿌리라 묶어서 처리하는 편이 낫다.
- [러너 07:14] review approved — 리뷰 승인 (risk=low)
- [러너 07:14] pr created — https://github.com/hkjang/bamcp/pull/1
- [러너 07:15] ci failed — 성공이 아닌 검사: Bamboo 권한 플러그인 빌드=failure, 도커 이미지 빌드=failure, Go 테스트=failure, 웹 콘솔 빌드=failure · 실패한 검사: ? 잡: 웹 콘솔 빌드 
