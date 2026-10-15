- 과제: [수정 과제] PR #32(`auto/2026-10-16-0228`, 107eeca)의 CI 실패를 재현해 근본 원인을 고친다 — 새 `network_cidr_test.go` 가 공유 Postgres 에 catch-all `0.0.0.0/0` 행을 남기는 것이 1순위 가설 (가치 5 / 위험 2 / 작업량 M)

- 왜: 2026-10-16 회차가 `createNetwork` 의 CIDR 정규화를 구현했는데(프로덕션 1파일 + 테스트 1파일) CI 가 실패해 verify-failed 로 끝났고 PR #32 가 열린 채 남아 있다. 고치면 이미 옳은 서버 수정(호스트 비트가 켜진 CIDR 이 500 대신 정규화 저장)이 머지되고, 공유 DB 를 쓰는 테스트가 서로를 오염시키는 구조적 함정이 하나 닫힌다.

- **먼저 읽을 것 — 이 정찰은 아무것도 실행하지 못했다.** 이 세션은 Bash 실행 권한이 막혀 있었다: `go version`·`go vet`·`docker info`·`gh pr view`·`curl api.github.com` 이 전부 "requires approval" 로 거부됐다. 따라서 **실제 CI 로그를 보지 못했고, 어떤 단계가 왜 빨간불인지는 미확인이다.** 아래 가설은 전부 코드 읽기만으로 세운 것이다. 구현자의 **첫 번째 작업은 추측을 고치는 것이 아니라 실패를 재현해 진짜 원인을 읽는 것**이다.

- 재현 절차 (ci.yml 그대로):
  1. `git checkout auto/2026-10-16-0228` (또는 main 에 cherry-pick 107eeca).
  2. CI 와 같은 DB 를 띄운다:
     `docker run -d --name momento-ci-pg -e POSTGRES_PASSWORD=ci -e POSTGRES_DB=momento -p 5432:5432 postgres:17-alpine`
  3. `export MOMENTO_TEST_POSTGRES_DSN='postgres://postgres:ci@127.0.0.1:5432/momento?sslmode=disable'`
  4. **패키지 전체를, CI 와 똑같이 한 번에** 돌린다 — 패키지를 하나만 돌리면 1순위 가설이 재현되지 않는다:
     `go test -race ./cmd/... ./internal/...`
  5. 빨간불이 나오면 그 테스트 이름과 메시지를 그대로 기록하고, 그 다음에 고친다.
  6. 위 4번이 녹색이면 가설 H1·H2 는 기각이다. 그때는 남은 CI 단계를 순서대로 돌려 빨간불을 찾는다:
     `go vet ./cmd/... ./internal/...`
     `go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...`
     `(cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build)`
     `(cd web && npm ci && npm audit && npm run lint && npm test && npm run build)`
     `docker build --build-arg VERSION=ci --build-arg COMMIT=$(git rev-parse --short=12 HEAD) -t momento:ci .`

- 가설 (증거가 센 순서. 재현이 말해 주는 것이 이 순서보다 우선한다):
  - **H1 — 새 테스트가 공유 Postgres 에 catch-all `0.0.0.0/0` 을 띄워 둔다.**
    `internal/httpapi/network_cidr_test.go` 의 `TestCreateNetworkStoresTheNetworkAddressOfACIDRWithHostBits` 는 표에 `{"정규화 핸들러 확인 전체", "0.0.0.0/0", "0.0.0.0/0"}` 를 두고 실제로 `network_ranges` 에 INSERT 한다. 수집기는 `internal/service/collector.go:431` 에서
    `SELECT name,internal FROM network_ranges WHERE $1::inet <<= cidr ORDER BY masklen(cidr) DESC LIMIT 1`
    로 읽으므로, 그 행이 살아 있는 동안 **모든 IPv4 주소가 내부망으로 분류된다**(`internal=true`, 이름 '정규화 핸들러 확인 전체').
    CI 는 `MOMENTO_TEST_POSTGRES_DSN` 하나를 **네 패키지가 공유**한다 — `internal/httpapi`(`integration_test.go`·`load_test.go`), `internal/service`(`inbox_drain_test.go`·`maintenance_test.go`), `internal/database`(`migrate_parallel_test.go`·`upgrade_test.go`). `go test ./...` 는 패키지를 **동시에** 돌리므로, 같은 패키지 안의 순차 실행 순서는 보호가 되지 못한다. `internal/service` 의 드레인 테스트가 `is_internal`/`network_name` 을 보는 동안 `internal/httpapi` 의 새 테스트가 catch-all 을 띄워 두면 그쪽이 빨간불이 된다. 같은 이유로 **재현이 간헐적일 수 있다** — "같은 이유로 두 번 실패" 와도 어긋나지 않는다(같은 테스트가 두 번 졌을 수 있다).
    고치는 방향(느슨하게 만들지 말 것): `0.0.0.0/0`·`::/0` 를 **DB 에 넣는 사례에서 빼고**, 전체 대역 정규화는 DB 없는 `TestNormalizeCIDRAgreesWithParseCIDR`(이미 `0.0.0.0/0`·`::/0` 를 담고 있다)에 맡긴다. 핸들러 경유 확인은 `10.20.30.1/24`·`10.20.31.0/24`·`10.20.32.1/32`·`2001:db8::1/64` 로도 수용 기준을 똑같이 증명한다. **단언을 지우지 말고 사례만 좁히는 것**이 핵심이다.
  - **H2 — 같은 테스트가 `listNetworks` 로 테이블 전체를 읽는다.** 같은 함수 끝에서 `server.listNetworks` 를 호출해 `network_ranges` 의 **모든 행**을 받아 이름으로 찾는다. 다른 패키지가 동시에 넣고 지우는 행과 섞이고, `integration_*_test.go:41/109` 의 `테스트 사내망`(`192.0.2.0/24`)이 같은 테이블에 있다. 이름이 겹치지는 않으므로 H1 보다 약하지만, 목록 대신 `SELECT cidr::text FROM network_ranges WHERE name=$1` 로 좁히면 이 결합이 사라진다.
  - **H3 — 환경 게이트(`govulncheck` 또는 `npm audit`)가 새 advisory 로 빨간불.** 이 저장소의 전례다(`fcf17ce`: x/text v0.41.0 으로 GO-2026-6629 해소). PR 의 Go diff 와 무관하게 실패하므로, H1·H2 를 고쳐도 CI 가 여전히 빨갛다면 여기다. 고치는 방향은 **게이트 완화가 아니라 의존성 상향**이다(`go get`/`npm update` 로 해당 패키지만, 메이저 업그레이드 금지).
  - **H4 — `go vet` / 컴파일.** `normalizeCIDR` 는 패키지에 중복 이름이 없고(`grep normalizeCIDR` 결과 이 diff 뿐), `cidr, err := …` 뒤 `err = s.DB.QueryRow(…)` 로 바꾼 것도 문법상 옳다. 가능성은 낮지만 재현 2단계에서 공짜로 가려진다.

- 수용 기준:
  1) 재현 절차 4번(`go test -race ./cmd/... ./internal/...`, 실제 Postgres 연결)이 PR 브랜치에서 **실패하는 것을 먼저 직접 관측**하고 그 출력(테스트 이름 + 메시지)을 원장에 그대로 적는다. 관측하지 못했으면 그 사실을 적고, 무엇을 대신 확인했는지 적는다.
  2) 고친 뒤 같은 명령이 exit 0 이고, 같은 명령을 **연속 2회** 돌려도 녹색이다(H1 이 원인이면 간헐적이므로 1회로는 증명되지 않는다).
  3) `createNetwork` 의 수용 기준이 그대로 남아 있다: 호스트 비트가 켜진 `10.20.30.1/24` 가 201 로 저장되고 `network_ranges.cidr` 이 `10.20.30.0/24` 이며, `net.ParseCIDR` 이 거절하는 값은 여전히 400 `INVALID_CIDR` + `CIDR is invalid` 다. 이 단언을 **약화하거나 삭제하지 않았다**.
  4) `.github/workflows/*.yml` 을 **바꾸지 않았다**(워크플로를 느슨하게 해서 통과시키는 것은 금지). 바꿔야만 한다면 그 이유를 원장에 적고 게이트를 줄이지 않는 변경만 한다.
  5) `go vet ./cmd/... ./internal/...` 이 exit 0.

- 건드릴 파일 (프로덕션 0~1, 테스트 1 — 네 개 안쪽):
  - `internal/httpapi/network_cidr_test.go:TestCreateNetworkStoresTheNetworkAddressOfACIDRWithHostBits` — DB 에 넣는 사례에서 전체 대역(`0.0.0.0/0`)을 빼고, 끝의 `listNetworks` 전체 읽기를 이름으로 좁힌 조회로 바꾼다. (H1·H2)
  - `internal/httpapi/network_cidr_test.go:TestPostgresRefusesACIDRWithHostBitsAndAcceptsTheNormalizedOne` — 정리(`t.Cleanup`)가 실패 경로에서도 반드시 도는지 확인. 지금은 이름 하나만 지운다.
  - `internal/httpapi/admin.go:normalizeCIDR` / `createNetwork`(797-845) — **원인이 서버 쪽으로 밝혀진 경우에만** 손댄다. 그렇지 않으면 그대로 둔다.
  - (H3 이면) `go.mod`/`go.sum` 또는 `web/package.json`·`sdk/package.json` 중 advisory 가 가리킨 **한 패키지만**.

- 검증 명령 (이 저장소에서 실제로 CI 가 도는 것):
  - `docker run -d --name momento-ci-pg -e POSTGRES_PASSWORD=ci -e POSTGRES_DB=momento -p 5432:5432 postgres:17-alpine`
  - `MOMENTO_TEST_POSTGRES_DSN='postgres://postgres:ci@127.0.0.1:5432/momento?sslmode=disable' go test -race ./cmd/... ./internal/...` (두 번)
  - `go vet ./cmd/... ./internal/...`
  - 필요할 때만: `go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...`, `(cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build)`, `(cd web && npm ci && npm audit && npm run lint && npm test && npm run build)`
  - 끝나면 `docker rm -f momento-ci-pg`

- 위험과 피할 것:
  - **워크플로 파일(`.github/workflows/ci.yml`·`release.yml`·`release-reconcile.yml`)을 느슨하게 만들지 말 것.** `-race` 제거, `npm audit` 제거, `continue-on-error`, `-skip` 추가 전부 금지. 이 회차의 명시적 조건이다.
  - `internal/auth`·`internal/database/migrations` 는 건드리지 말 것. 이번 원인과 무관하다.
  - `web/src/pages/cidrRule.ts` 의 「대역의 시작 주소는 …」 안내 문구는 **이번 범위가 아니다**(별 회차로 미뤄 둔 항목이고, `web/test/cidrRule.test.mjs` 가 문장을 글자 그대로 본다).
  - `integration_traffic_class_test.go:41`·`integration_channel_internal_test.go:109` 의 `192.0.2.0/24` 고정 대역을 바꾸지 말 것 — httptest 가 모든 요청을 `192.0.2.1` 에서 보내는 것에 묶여 있다.
  - 교훈: 이 저장소의 과거 회차가 "로컬에서 되니 CI 도 될 것" 으로 verify-failed 를 반복했다. **패키지 하나만 돌려 녹색을 본 것으로 끝내지 말 것** — H1 은 패키지 동시 실행에서만 보인다.
  - `0beff4c`(main)와 PR 브랜치의 차이는 커밋 하나뿐이다. 다른 기능을 같이 넣지 말 것.

- 차선 후보: **H3 경로 — 의존성 advisory 하나를 올려 CI 를 녹색으로 되돌린다.** H1·H2 가 재현되지 않고 `govulncheck` 또는 `npm audit` 이 빨간불이면, `fcf17ce`(x/text v0.41.0) 와 똑같은 모양으로 지목된 패키지 하나만 최소 버전으로 올리고(메이저 금지), 같은 게이트를 로컬에서 돌려 exit 0 을 확인한다. 그 경우 `network_cidr_test.go` 는 손대지 않는다.
