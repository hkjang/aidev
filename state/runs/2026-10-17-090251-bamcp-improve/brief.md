# 과제서 — 2026-10-17-090251-bamcp-improve (bamcp)

- **과제: CI 4개 잡 전부 실패하는 원인을 로그에서 특정해 고치기 (PR #1)** (가치 5 / 위험 2 / 작업량 M)

- **왜**: PR #1(`auto/2026-10-17-0659`, 커밋 `30928ef`, `internal/tools/args.go`+테스트 2개만 수정)에서 CI 4개 잡이 **전부** 실패했다 — 러너 기록: `[러너 07:15] ci failed — 성공이 아닌 검사: Bamboo 권한 플러그인 빌드=failure, 도커 이미지 빌드=failure, Go 테스트=failure, 웹 콘솔 빌드=failure`. 그런데 그 diff는 `internal/tools/` 밖을 전혀 건드리지 않으므로 웹·플러그인·도커 잡이 깨질 이유가 코드에는 없다. 즉 **PR 코드가 아니라 워크플로/저장소/계정 수준의 공통 원인**일 가능성이 높고, 이것을 특정하지 못하면 앞으로 어떤 PR도 머지되지 않는다.

## ⚠ 이 과제서의 한계 (먼저 읽을 것)
이 정찰 세션에서는 `gh`, `go`, `npm`, `docker`, `mvn` 실행이 **전부 권한 거부**되었다(`git`/`ls`/`cat`/`grep`만 허용). 따라서 **실패 로그를 못 봤고, 어떤 잡의 어떤 단계가 왜 깨졌는지는 미확인**이다. 아래 가설은 전부 정적 읽기만으로 세운 것이다. **추측을 믿지 말고 1단계(로그 확보)부터 하라.**

## 1단계 — 실제 실패 로그부터 받아라 (이것이 과제의 절반이다)
```
gh run list --repo hkjang/bamcp --limit 10
gh run view <RUN_ID> --repo hkjang/bamcp --log-failed | head -200
```
4개 잡의 **첫 실패 단계 이름과 에러 문구**를 적은 뒤에 고치기 시작할 것. 로그에서 분기:

- **(A) 4개 잡이 모두 "The job was not started…" / 결제·쿼터·러너 배정 실패**
  → 저장소에서 고칠 수 있는 문제가 아니다. **워크플로를 건드리지 말고** 원장에 "CI 실패 원인=계정/러너 수준, 코드 무관"으로 기록하고 **차선 후보**로 넘어가라. 이것이 가장 유력한 단일 가설이다(4개 잡이 서로 독립인데 동시에, 두 번 같은 이유로 깨졌다).
- **(B) 각 잡이 서로 다른 단계에서 깨짐** → 아래 잡별 후보를 보라. 이 경우 CI가 이 저장소에서 **한 번도 통과한 적이 없을** 가능성이 크다(워크플로는 `245f3ef` 에서 다른 전부와 함께 한 번에 들어왔고 그 뒤 검증 커밋이 없다). 그러면 **이번 회차에는 한 잡만 고치고 나머지는 보류 아이디어로 남겨라** — 4개를 한 번에 고치면 파일 수가 6개를 넘는다.
- **(C) Go 테스트만 실질 실패이고 나머지는 취소/연쇄** → `internal/tools` 의 `toInt` 엄격화가 기존 테스트를 깬 것이다. 아래 Go 후보를 보라.

## 잡별 후보 (정적 읽기 기준, 전부 미검증)
- **Go 테스트** (`.github/workflows/ci.yml:17-40`): 0659 회차 구현자는 도커 임시 Postgres로 `go build ./... && go vet ./... && go test ./... -count=1 -p 1` 전부 통과를 주장했다. 같은 명령이므로 통과해야 정상. 다만 0659 **정찰**은 `go build`/`go vet` 를 "미확인"으로 남겼다. 의심 지점: `args.go:139 const maxExclusive = 1 << 63` (타입 없는 상수 — float64 비교에서만 쓰여 컴파일은 되지만 `go vet`/린터가 걸 수 있는 유일한 새 구문), 그리고 `analysis.go:397 splitResultKey` 가 `toInt` 엄격화로 같이 좁아진 것(0659 비평가가 남긴 우려).
- **웹 콘솔 빌드** (`ci.yml:42-57`): `npm run check` = `tsc -b --noEmit`(`web/package.json:10`). TS는 락파일에서 5.9.3이라 `-b --noEmit` 조합 자체는 지원된다. 의심 지점: `web/tsconfig.app.tsbuildinfo`·`web/tsconfig.node.tsbuildinfo` 가 **저장소에 커밋되어 있다**. 신규 체크아웃에서 `tsc -b` 가 이 스테일 buildinfo로 up-to-date 판정/재빌드 판정을 다르게 하며, `check` 가 buildinfo를 덮어쓴 뒤 `build` 가 다시 도는 순서다. 커밋된 `*.tsbuildinfo` 2개를 지우고 `web/.gitignore` 에 넣는 것은 **워크플로를 느슨하게 만드는 것이 아니라 산출물을 소스에서 빼는 정당한 수정**이다.
- **Bamboo 권한 플러그인 빌드** (`ci.yml:59-71`): `mvn -B -ntp package`, Temurin 8. `plugin/pom.xml` 존재 확인함. 의심 지점: Atlassian 사설 저장소(`packages.atlassian.com`) 접근, Java 8에서 도는 Maven/플러그인 버전.
- **도커 이미지 빌드** (`ci.yml:73-107`): `docker build` 후 `기동 점검`. 기동 점검은 `healthz` 를 30회 재시도하지만 **루프가 실패해도 종료 코드를 보지 않고** 다음 `curl -fsS .../api/config` 에서 처음으로 깨진다 — 로그에 그렇게 찍혔다면 원인은 "컨테이너가 안 떴다"이지 curl이 아니다. `docker logs bamcp` 를 반드시 읽을 것.

## 수용 기준
1) 4개 잡의 실제 실패 단계와 에러 문구가 원장/저널에 **그대로 인용**되어 있다 (추측 금지).
2) 원인이 저장소에서 고칠 수 있는 것이면, 그중 **한 잡**을 골라 고치고 **같은 검증을 로컬에서 재현해 통과**를 확인했다 (아래 검증 명령).
3) 고치기 전 상태로 되돌리면 같은 실패가 재현된다는 것을 보였다 (실패 → 수정 → 통과 순서).
4) `.github/workflows/ci.yml` 의 **검증 강도를 낮추지 않았다** — 단계 삭제, `continue-on-error`, `|| true`, 테스트 제외, 버전 낮추기 전부 금지. 워크플로 수정은 "깨진 설정을 올바른 설정으로" 일 때만 허용하고, 그 이유를 커밋 메시지에 쓴다.
5) 원인이 계정/러너 수준이면 코드를 바꾸지 않고 그 사실을 기록한 뒤 차선 후보를 수행했다.

## 건드릴 파일 (6개 이내 — 분기에 따라 하나의 묶음만)
- 분기 (A): **없음**. 기록만.
- 분기 웹: `web/tsconfig.app.tsbuildinfo`, `web/tsconfig.node.tsbuildinfo` (삭제), `web/.gitignore` (신규 또는 추가), 필요시 `web/package.json:10` (`check` 스크립트)
- 분기 Go: `internal/tools/args.go`, 필요시 `internal/tools/analysis.go:397 splitResultKey`, `internal/tools/args_test.go`
- 분기 플러그인: `plugin/pom.xml`, 필요시 `.github/workflows/ci.yml:59-71`
- 분기 도커: `Dockerfile`, 필요시 `.github/workflows/ci.yml:94-107` (기동 점검 루프의 실패를 삼키지 않도록 — **강도를 올리는 방향만**)

## 검증 명령 (이 저장소에서 CI가 실제로 쓰는 것과 동일)
```
# Go 잡 (DB 필수 — 없으면 테스트가 skip 되어 통과처럼 보인다)
docker run -d --name bamcp-pg -e POSTGRES_USER=bamcp -e POSTGRES_PASSWORD=bamcp \
  -e POSTGRES_DB=bamcp_test -p 5432:5432 postgres:16-alpine
export TEST_DATABASE_URL='postgres://bamcp:bamcp@localhost:5432/bamcp_test?sslmode=disable'
go build ./... && go vet ./... && go test ./... -count=1 -p 1

# 웹 잡 (반드시 깨끗한 상태에서 — git clean 으로 node_modules/tsbuildinfo 제거 후)
cd web && npm ci --no-audit --no-fund && npm run check && npm run build

# 플러그인 잡
cd plugin && mvn -B -ntp package     # 느림

# 도커 잡
docker build --build-arg VERSION="$(cat VERSION)" -t "bamcp:v$(cat VERSION)" .
```
웹을 고쳤으면 `internal/webui/dist` 임베드 산출물이 바뀌는지 확인하고, 바뀌면 그것도 커밋해야 `go build` 결과에 반영된다.

## 위험과 피할 것
- **워크플로를 느슨하게 만들어 통과시키는 것은 금지** (운영자 지시). `continue-on-error`, 단계 제거, `-p 1` 제거, 테스트 skip 확대 전부 반려 사유다.
- `internal/api/oauth.go`·`internal/auth/`·`internal/permission/resolver.go`·`internal/identity/mapper.go`·`internal/database/migrations/` 는 이 과제와 무관하다. 손대지 말 것.
- `go mod tidy` 를 무심코 돌리지 말 것 — `go.mod` 의 chi/go-oidc/golang-jwt/uuid 가 직접 의존인데 `// indirect` 로 적혀 있어 diff 가 크게 벌어진다. 이번 과제 범위 밖이다.
- 4개 잡을 한 회차에 다 고치려 들지 말 것. 파일 10개 이상 변경은 47%가 사람 손을 다시 탄다.
- `TEST_DATABASE_URL` 없이 `go test ./...` 는 통과처럼 보인다. 그것을 근거로 "통과"라고 쓰지 말 것.

## 차선 후보
**`internal/tools/read.go:190,482` 의 raw `start`/`limit` 를 Bamboo 전달 전 클램프** (가치 3 / 위험 2 / S). 두 곳만 `OptInt("start")`/`OptInt("limit")` 를 `clamp()` 없이 Bamboo REST 로 넘긴다 — 같은 파일의 다른 호출부는 전부 `clamp()` 를 거친다. 음수·거대값이 그대로 나가 Bamboo 측 오류나 과다 응답을 유발할 수 있다. 수정 범위는 `internal/tools/read.go` 1개 파일 + `internal/tools/` 테스트 1개. 분기 (A)(계정/러너 수준 실패)일 때 이쪽을 하라.
