# PR 처리기 노트 2026-10-16-031104-Momento-shepherd — Momento PR #32
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-16-022842-Momento-improve)
# 회차 노트 2026-10-16-022842-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@0beff4c
- [러너 02:28] autonomy release — 

## 정찰 노트
- 고른 이유: 최근 여섯 회차가 전부 web 문구·helperText 였고 그 중 둘이 verify-failed 였다. 이번엔 서버의 실제 결함을 골랐다 — `admin.go:createNetwork` 가 `net.ParseCIDR` 의 반환값 두 개를 버리고 원문을 `cidr cidr NOT NULL` 컬럼(001_initial.sql:67)에 넣는다. 프로덕션 1파일로 끝나고, `ci.yml:17-30` 에 postgres:17-alpine + `MOMENTO_TEST_POSTGRES_DSN` 이 **이미 있어** DB 테스트가 CI 에서 실제로 돈다(이전 프로필은 "DB 통합 skip" 만 적어 로컬 이야기로 읽혔다).
- 제친 후보: Custom Dimension helperText 와 README SDK 줄은 **머지 대기 PR 과 같은 줄**이다 — pinned main 에 `describeDimensionError` 가 없고 AdminPage.tsx:3122 는 아직 `{save.error.message}` 임을 확인했으므로, 재구현이 아니라 회피로 처리했다. DataTable CSV 는 csvExport.ts 주석이 원시값을 계약으로 못 박아 rejected, 표 검색은 열여덟 회차 미선정으로 rejected.
- 확신 없는 곳: PostgreSQL `cidr` 이 호스트 비트를 거절한다는 것은 **문서 근거이고 이 환경에서 실행으로 재현하지 못했다**(docker/psql 가용성 확인이 권한으로 막혔다). 과제서에 "원문 INSERT 가 성공하면 전제가 거짓이니 밀지 말고 차선으로" 를 명시했다. `s.audit` 가 principal 없이 도는지도 미확인으로 남겼다.
- 구현자가 조심할 것: 400 거절로 돌아서지 말 것(cidrRule.ts:104-107 이 막지 않기로 한 이유를 적고 있다), 같은 정규화를 다른 핸들러·`trusted_proxy_cidrs` 로 번지게 하지 말 것, migrations 와 cidrRule.ts 는 이번에 손대지 말 것, 저장만 보지 말고 `listNetworks` 반환값까지 읽을 것.
- 프로필은 7일 전(14일 미만)이고 열어 본 자리가 전부 맞아 다시 쓰지 않았다. 다만 "CI 에 postgres 서비스와 DSN 이 있다" 는 사실이 빠져 있으니 다음 프로필 갱신 때 넣을 것.
- [러너 02:34] scout done — `createNetwork` 가 `net.ParseCIDR` 로 파싱한 대역을 버려서, 호스트 비트가 켜진 CIDR 이 Postgres `cidr` 컬럼에 막혀 

## 구현 노트
- 무엇·왜: `admin.go:createNetwork` 가 `net.ParseCIDR` 의 대역 반환값을 버리고 원문을 `cidr cidr NOT NULL` 컬럼에 넣어 `10.20.30.1/24` 가 500 + pgx 원문이 됐다. 같은 파일 안 헬퍼 `normalizeCIDR` 을 두어 `(*net.IPNet).String()` 을 INSERT `$2` 와 `s.audit` 의 `cidr` 에 쓴다. 프로덕션 1파일, 테스트 1파일(신규).
- 전제는 추측이 아니라 실행으로 확인했다: 이 환경에 docker 가 있어 `postgres:17-alpine` 을 직접 띄웠고(포트 55432), 원문 INSERT 가 `invalid cidr value … Value has bits set to right of mask` 로 막히는 것을 먼저 봤다. 항등 스텁으로 핸들러가 500 을 돌려주는 red 를 찍은 뒤 고쳤다.
- 확신 없는 곳: (a) `s.audit` 가 principal 없이도 핸들러를 깨뜨리지 않는 것은 확인했으나(`Exec` 가 `_, _ =` 로 오류를 삼킨다) **감사 행이 실제로 쌓이는지는 단언하지 않았다** — FK(`actor_id REFERENCES users`) 때문에 테스트에서는 아마 쌓이지 않았을 것이다. (b) `net.ParseCIDR` 이 IPv4-mapped IPv6 같은 희귀 표기에서 내놓는 정규형은 표로 확인한 10개 밖까지 보지 않았다. (c) govulncheck·web·sdk 게이트는 **돌리지 않았다**(의존성·web·sdk 무변경).
- 일부러 하지 않은 것: 400 거절로 돌아서지 않았다(`cidrRule.ts:104-107` 이 막지 않기로 한 이유를 적고 있다); `cidrRule.ts`·`migrations/**`·`internal/auth`·워크플로·`docs/openapi.yaml` 무변경; 같은 정규화를 `deleteNetwork`·`trusted_proxy_cidrs`·`segment`/`insight` 로 번지게 하지 않았다(헬퍼 주석에 경계 명시); 감사 `details` 에 새 원문·`err.Error()` 를 더하지 않았다.
- 다음 역할이 조심할 것: `network_cidr_test.go` 의 네 테스트 중 **셋은 DB 가 있어야 돈다**(`MOMENTO_TEST_POSTGRES_DSN` 없으면 skip — skip 을 통과로 읽지 말 것). CI 는 `ci.yml:17-30` 의 postgres 서비스로 실제 실행한다. 로컬에서 확인하려면 `docker run -d -e POSTGRES_PASSWORD=ci -e POSTGRES_DB=momento -p 55432:5432 postgres:17-alpine` 후 DSN 을 주면 된다. 테스트는 `name` 으로 자기 행만 `t.Cleanup` 에서 지우므로 공유 DB 를 더럽히지 않는다. `admin.go` 는 `gofmt` 가 한 번 걸려(새 주석 앞 빈 줄) 고친 뒤 amend 했다 — 현재 `gofmt -l internal/httpapi/` 는 비어 있다.
- [러너 02:40] brief accepted — 채택 — 인용한 자리(`admin.go:800-828` 의 버려진 반환값, `001_initial.sql:67` 의 `cidr cidr NOT NULL`, `integration_test.go:35` 의 `testPool`, 
- [러너 02:40] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 구현자가 의심한 자리를 직접 돌려 봤다: 따로 띄운 postgres:17-alpine(56321)에서 새 테스트 5개 PASS, `admin.go` 를 main 판 + 항등 스텁으로 되돌리면 `network_cidr_test.go:120` 이 `500 NETWORK_CREATE_FAILED … SQLSTATE 22P02` 로 red — 원장의 `- 실패 재현:` 줄과 글자까지 일치한다. 테스트가 바뀐 경로를 정말 지난다.
- 게이트 실측: `go vet ./cmd/... ./internal/...` exit 0, 실제 DSN 을 준 `go test -race ./cmd/... ./internal/...` 전 패키지 exit 0. `gofmt -l` 이 짚는 `internal/segment/segment_test.go` 는 ad878a1 선존이고 이번 diff 밖이다. 범위·인가·마이그레이션 무변경(server.go:115-117 그대로), 되돌리기 안전.
- 못 본 것: govulncheck·npm audit·web/sdk·Docker 빌드는 돌리지 않았다(의존성·web·sdk 무변경이라 생략). 감사 행이 실제로 쌓이는지는 나도 단언하지 않는다(테스트는 principal 없이 돌고 `audit` 이 오류를 삼킨다).
- 승인이어도 남는 우려 — 릴리즈 노트/다음 회차: (1) `network_cidr_test.go:124-128` 누출 검사는 :121 의 `continue` 때문에 201 응답에서만 돌아 사실상 죽은 단언이다. (2) 핸들러 테스트가 테스트 동안 `0.0.0.0/0`(internal=true) 포괄 행을 넣는다 — 지금은 직렬+Cleanup 으로 안전하지만 이 패키지에 `t.Parallel()` 로 트래픽 분류를 단언하는 테스트가 생기면 뒤집힐 수 있다. (3) createNetwork 의 500 은 아직 `err.Error()` 원문을 돌려준다(선존, orgAdmin 전용).
- 다음 회차 후보: `cidrRule.ts:104-107`·`adminErrors.ts:300-307` 주석이 아직 "Postgres cidr 컬럼 거동 미확인" 이라고 적고 있는데 이번에 실측됐다. 또 이제 서버가 조용히 대역 시작 주소로 바꿔 저장하므로, 폼이 hint 대신 바뀔 값을 미리 보여 주는 쪽이 사용자에게 정직하다.
- [러너 02:45] review approved — 리뷰 승인 (risk=low)
- [러너 02:45] pr created — https://github.com/hkjang/Momento/pull/32
- [러너 02:49] ci failed — 성공이 아닌 검사: test=failure · 실패한 검사: ? 잡: test 

## 수리 노트
- 지적은 맞았다 — 다만 PR 과 무관하다. CI 가 멈춘 자리는 `govulncheck` 단계뿐이고(`go test -race`·`go vet` 는 그 앞에서 통과), 원인은 CIDR 코드가 아니라 고정 `toolchain go1.26.7` 의 stdlib 권고 10건(`net/http` 8·`net/textproto` 1·`crypto/tls` 1, GO-2026-6603~6617)이 전부 **호출 경로 확인**으로 뜬 것이다. 호출 지점은 선존 코드(`automation.go:506`, `secret/cipher.go:155`, `collector.go:1341`)이고 107eeca 가 만든 자리가 아니다. pre-fix 트리에서 CI 와 같은 명령으로 같은 10건·같은 추적줄을 먼저 재현했다(red).
- 고친 방법: `go.mod` 의 `toolchain` 과 Dockerfile 빌더 이미지를 1.26.9 로. 선례 5a9e55e 와 같은 두 자리이고, `go 1.26.0` 지시자·의존성·`go.sum` 은 건드리지 않았다. 빌더 이미지를 함께 올린 이유는 go.mod 만 올리면 릴리즈 이미지가 여전히 취약한 stdlib 로 링크되기 때문이다. 새 테스트는 두지 않았다 — toolchain 핀을 단언하는 테스트는 go.mod 를 복창하는 것이라 govulncheck 자체가 회귀 게이트다.
- 실측(go1.26.9): govulncheck exit 0(호출 0건), `go vet` exit 0, `gofmt -l internal/httpapi/ cmd/` 비어 있음, 실제 postgres:17-alpine DSN 을 준 `go test -race ./cmd/... ./internal/...` 전 패키지 ok. PR 의 CIDR 테스트 7개는 **skip 없이** PASS 확인(`-run 'CIDR|Network' -v`). sdk·web `npm audit` 0건. `docker build` exit 0 — `golang:1.26.9-bookworm` 으로 실제 빌드되고 sdk/web `npm ci && npm run build` 도 그 안에서 돌았다.
- 여전히 확신 없는 곳: (a) web `npm run lint`·`npm test` 와 sdk `typecheck`·`test` 는 Node 24 가 아닌 로컬에서 **따로 돌리지 않았다**(내 diff 가 go.mod·Dockerfile 뿐이라 영향 없음, Docker 안의 build 만 확인). (b) govulncheck 는 advisory DB 를 매번 새로 읽으므로 이번 통과가 다음 실행의 통과를 보장하지 않는다 — 1.26.10 이 나올 만한 새 권고가 뜨면 같은 자리에서 또 멈춘다. (c) 비평가가 남긴 세 우려(죽은 누출 단언, `0.0.0.0/0` 포괄 행, 500 의 `err.Error()` 원문)는 이번 수리 범위가 아니어서 **그대로 두었다**.
- 릴리즈 역할이 볼 것: 선례 5a9e55e 는 같은 성격의 변경에 `internal/version/version.go`·CHANGELOG·`.github/release-notes/` 를 함께 넣었다. 나는 CI 게이트 복구만 했으므로 버전·릴리즈 노트는 손대지 않았다.

## 심사 노트
- 확인한 것: 실제 postgres:17-alpine(55433) DSN 으로 CIDR 테스트 7개 skip 없이 PASS, 전체 `go test -race` ok, `go vet`·`gofmt` 깨끗, govulncheck exit 0(원래 실패 10건 소멸), `docker pull golang:1.26.9-bookworm` 성공, ci.yml:34-37 의 `go-version-file: go.mod` 가 toolchain 1.26.9 를 실제로 집는 것. normalizeCIDR 을 항등 스텁으로 바꿔 :47·:120·:155 red 를 직접 찍고 원본 복원(코드 미변경).
- 찾은 결함(reject/fix): admin.go:809-815 가 IPv4-mapped IPv6 에서 주소 체계를 바꾸고 대역을 넓힌다 — `::ffff:0:0/96` → `0.0.0.0/0`, `::ffff:10.0.0.0/104` → `10.0.0.0/8`. psql 로 main 은 이 값들을 그대로 저장했음을 확인했으니 호스트 비트와 무관한 회귀이고, collector.go:431 의 `<<=` 라벨링이 이를 소비해 모든 IPv4 방문자를 internal 로 만든다('203.0.113.9' 매칭이 f → t). 주석·커밋이 말하는 "속한 대역"·"판정 불변" 과도 어긋난다.
- 못 본 것: web/sdk npm 게이트, 전체 docker build, release.yml 오프라인 smoke. govulncheck 는 advisory DB 의존이라 이번 통과가 다음 통과를 보장하지 않는다.
- 비평 노트 우려 확인: :123-127 누출 검사는 201 본문이 `{"id":...}` 뿐이라 영구히 참인 죽은 단언(차단 아님). 0.0.0.0/0 포괄 행은 t.Parallel() 테스트가 직렬 Cleanup 뒤 재개되므로 지금은 안전(전체 -race 로 실증).
- 권고 근거: 결함이 `::ffff:` 표기 하나에 갇혀 있어 107eeca 본론과 6cef4b3 의 CI 복구는 살릴 수 있다 — 체계 보존 분기 + 표에 `::ffff:` 두 줄을 더하면 끝나므로 human 이 아니라 fix. network_ranges.internal 은 분석 라벨 전용이고 인가에 쓰이지 않아 보안 차단 소견은 없다(blocking 비움), 다만 잘못된 라벨이 이벤트 행에 박혀 revert 로 돌아오지 않으므로 risk=medium.
