- 과제: Resolver 권한 캐시의 무제한 증가를 막고 TTL 캐시 경로에 첫 테스트를 붙인다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/permission/resolver.go` 의 `cache map[string]cacheEntry` 는 만료 항목을 **그 키를 다시 조회할 때만** 지운다(`cached()` 안의 `delete(r.cache, key)` 가 유일한 삭제 지점이고, `Reset()` 외에는 다른 회수 경로가 없다). 존재하지 않는 project/repository 이름으로 호출해도 `restRepository` 는 `None` 결정을 **오류 없이** 돌려주므로 `store_` 가 기본 TTL 60초로 그 키를 캐시하는데, 그 키는 두 번 다시 조회되지 않아 항목이 프로세스 수명 내내 남는다 — 인증된 실행자 1명이 `repository` 인자만 바꿔 읽기 도구를 반복 호출하는 것으로 게이트웨이 메모리를 무한히 늘릴 수 있다. 고치면 캐시 메모리가 키 카디널리티와 무관하게 상한을 가지며, 두 회차 연속 차선 후보였던 "Resolver TTL 캐시 경로 테스트 0건" 공백도 같은 변경으로 닫힌다.

- 수용 기준:
  1) `store_` 로 상한보다 많은 서로 다른 키를 (ttl > 0 으로) 넣어도 `len(r.cache)` 가 상한 이하로 유지된다 — 지금은 넣은 수만큼 그대로 증가한다.
  2) 기존 캐시 의미가 그대로다: `ttl <= 0` 이면 캐시하지 않고, 만료된 키는 `cached()` 가 `false` 를 돌려주며 맵에서 사라지고, 회수(eviction)가 어떤 항목의 만료 시각도 **늘리지 않으며** 만료된 결정을 되살리지 않는다. 즉 회수의 유일한 결과는 "다음 호출에서 다시 조회" 이고, 캐시 때문에 허용이 늘어나는 방향의 변화는 없다.
  3) `Reset()` 후 맵이 비고 이후 `cached()` 가 어떤 이전 키에도 `false` 를 돌려준다.
  4) 테스트는 `internal/permission` 패키지 안(`package permission`)에서 `store_`/`cached`/`Reset` 을 직접 불러 1~3을 증명한다 — DB·HTTP·목 없이 돈다. 수정 전에 1)이 FAIL 하는 red 로그를 먼저 남길 것.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/permission/resolver.go:store_` — 락 안에서 삽입 전에 (a) 맵 크기가 상한에 닿았을 때 만료 항목을 쓸어내고, (b) 그래도 상한이면 만료가 가장 이른 항목 하나를 비우고 삽입. 패키지 수준 상수(예: `maxCacheEntries = 10000`)를 같은 파일에 선언. `cached()`·`Reset()`·`Repository()`·`Project()` 의 기존 동작은 바꾸지 말 것(`store_` 호출 지점·인자도 그대로).
  - `internal/permission/resolver_test.go` (신규) — `package permission`. `NewResolver(nil, nil)` 로 충분하다(`NewResolver` 는 store/provider 를 역참조하지 않고 `NewPluginClient(store)` 에 보관만 하며, `Reset()` 도 plugin 의 뮤텍스 필드만 건드린다 — 코드로 확인). 만료 테스트는 `store_(key, dec, 20*time.Millisecond)` 처럼 짧은 ttl 로 하고 `decide("u","P","r", Read, "test", "")` 로 Decision 을 만들면 된다.

- 검증 명령 (이 저장소에서 실제로 확인함):
  - `go test ./internal/permission -count=1 -v` — 이번 정찰에서 실행해 `ok github.com/hkjang/bbmcp/internal/permission 0.004s` 를 받았다. DB 불필요.
  - `go test ./internal/permission -count=5 -race` — 락 아래 회수 로직이므로 경쟁 확인 필수.
  - `go build ./... && go vet ./...`
  - 회귀 전체(이 변경에 DB 는 필요 없지만 회귀 확인용): `docker start bbmcp-test-pg` → `export TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable'` → `go test ./... -count=1 -p 1 -v` 에서 `--- SKIP` 0건 확인(TEST_DATABASE_URL 없으면 통합 테스트가 조용히 건너뛰므로 exit 0 을 통과로 읽지 말 것).

- 위험과 피할 것:
  - `internal/tools/integration_test.go:newFixture` 의 `perm.CacheTTLSec = 0`(246행)을 **바꾸지 말 것**. 그 한 줄이 공유 fixture 의 모든 권한 변경 시나리오 전제다 — 이번 테스트는 패키지 내부 단위 테스트로 가므로 fixture 를 건드릴 이유가 없다.
  - `Repository`/`Project` 의 fail-closed 분기(94·121행)와 `cached()` 의 조회 선행 순서는 계약이다. 캐시 적중이 설정 조회 뒤·업스트림 호출 앞에 오는 순서를 바꾸지 말 것.
  - 백그라운드 janitor 고루틴을 만들지 말 것 — 수명 관리라는 새 움직이는 부품이 생기고 `Resolver` 에 종료 경로가 없다.
  - 캐시 키에 `cfg.Mode`/`cfg.FailClosed` 를 넣는 별건(보류 아이디어 2/2/S)과 섞지 말 것. 이번 범위는 메모리 상한 + 테스트뿐이다.
  - 보호 경로 회피: `internal/auth/*`, `internal/api/oauth.go`, `internal/approval/*`, `internal/tools/executor.go`, `migrations/*`, `.github/workflows/*` 는 전혀 건드리지 않는다.
  - 과거 교훈: 이 저장소의 반복 결함은 "업스트림 오류를 삼켜 보호를 건너뛴다" 였다. 이번 변경은 그 반대 방향(캐시를 더 적게 신뢰)이어야 한다 — 상한에 닿았을 때 **삽입을 건너뛰는** 선택도 허용되지만, 만료 항목을 남겨 두고 유효 항목을 유지하는 쪽이 성능상 낫다.

- 차선 후보: `approval.Create` 의 `json.Marshal`·`scan()` 의 `json.Unmarshal` 오류 삼킴 정리 (가치 2 / 위험 1 / 작업량 S) — `internal/approval/approval.go` 의 `body, _ := json.Marshal(redacted)` 와 `_ = json.Unmarshal(args, &r.Arguments)` 가 오류를 버려 승인 화면 인자가 조용히 비어 보일 수 있다. 승인 보호 자체(해시·버전 고정·1회 소비)에는 영향이 없고 표시용 필드만 바뀌는 좁은 변경이다.
