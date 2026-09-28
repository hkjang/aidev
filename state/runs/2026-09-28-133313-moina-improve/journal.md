# 회차 노트 2026-09-28-133313-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:33] base pinned — main@d23f75d
- [러너 13:33] autonomy release — 

## 정찰 노트
- 우선 과제가 자동 배정돼 선택의 여지는 없었다. PR #34 = branch `auto/2026-09-28-1202` = `5915beb`(로컬에 있음)이고, 실패 신호는 앞 회차 journal의 `[러너 12:31] ci failed — 성공이 아닌 검사: Source tests=failure` 한 줄뿐이다.
- 확인한 것: `ci.yml`의 `source` 잡 step 순서와 env(PostgreSQL **16**, `MOINA_TEST_POSTGRES_DSN` 주입), `Dependency audit`이 `continue-on-error: true`라 범인이 아님, `5915beb`의 전체 diff(파일 3개), 새 테스트 234줄, `pgQuoteLiteral`이 `posts_update_postgres_integration_test.go:176`에 실재함, docker 사용 가능.
- **추측으로 남긴 것(구현자가 재현으로 판정할 것)**: 실패한 step이 무엇인지. `gh`가 이 샌드박스에서 권한 거부돼 CI 로그를 못 읽었고, staticcheck 실행도 권한 거부로 못 돌렸다. 1순위 가설은 staticcheck(앞 회차 비평 노트가 "미설치로 미실행"이라고 명시한 유일한 단계), 2순위는 검증에 쓴 PostgreSQL 17 ↔ CI 16 차이다. 둘 다 미확인.
- 구현자가 조심할 것: 워크플로를 느슨하게 만들어 통과시키는 것은 금지다(단계 제거·`continue-on-error`·테스트 제외·`//lint:ignore`). 새 테스트는 DSN이 없으면 조용히 `t.Skip` 하므로 `--- SKIP` 0줄을 반드시 확인하고, 다른 프로젝트의 docker 컨테이너(`ptium-chart-pg` 등)가 돌고 있으니 새 이름·새 포트를 쓸 것. `git stash`는 다른 세션과 스택을 공유하므로 쓰지 말 것.
- 프로필은 2일 전 것이고 이번에 확인한 모든 사실(CI 16, e2e 스크립트 부재, 통합 404 위치)과 어긋나지 않아 새로 쓰지 않았다.
- [러너 13:38] scout done — PR #34(`5915beb`, resolveReport transaction화)의 CI `Source tests` 실패를 원인대로 고친다 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 무엇을 왜: CI `Source tests` 실패는 `5915beb` 탓이 아니었다. 공개 api.github.com 으로 깨진 step 이 `Go formatting, tests and vet` 하나임을(frontend·staticcheck·audit 은 skipped) 확인하고, 최근 CI 25건에서 **같은 step 이 main 에서도 간헐 실패**(85740223·7d626a33·4fb2c6dc·5915beb)함을 확인했다. 범인은 `TestPostgreSQLSettingChangeReachesAnotherInstance` — worker 의 `LISTEN moina_settings` 확립을 기다리지 않고 NOTIFY 를 보내 경합 시 알림이 영구 유실된다. `pg_stat_activity` 장벽 하나로 고쳤다(`93b0234`, 테스트 파일 1개·프로덕션 0개). `5915beb` 은 결백하지만 main 진입이 필요해 cherry-pick 유지(`a4082e1`).
- 확신 없는 곳·검증 못 한 것: (1) CI 로그를 **한 줄도 읽지 못했다** — 로그 API 는 403 `Must have admin rights`, 웹은 `Sign in to view logs`. 따라서 "CI 에서 깨진 테스트가 정말 이것"은 직접 증거가 아니라 step·소요시간(26초 실패 vs 31초 성공)과 로컬 재현의 일치에 근거한 추론이다. 같은 step 에 다른 flaky 가 더 있을 가능성을 배제하지 못했다. (2) 검증 도중 `go test -race -count=1 ./...` 한 번이 httpapi 에서 FAIL 했는데 출력을 grep 으로 흘려 **어느 테스트인지 확인하지 못했다**; 직후 39회 연속 무실패였고 유력한 후보는 아래 (3) 이지만 미확인이다. (3) 이 샌드박스(WSL2)의 host wall clock 이 40초에 2회, 최대 971ms **뒤로** 점프하는 것을 측정했고 그 때문에 `TestPostgreSQLSecurityNoticesReachTheOwner` 가 경합 20회 중 2회 `ORDER BY created_at,id` 순서 역전으로 실패했다 — GitHub runner 는 이런 점프가 없으므로 CI 원인이 아니라고 판단해 손대지 않았다. 이 판단이 틀렸다면 CI 실패의 두 번째 원인이 남아 있다.
- 일부러 하지 않은 것: `ci.yml` 은 읽기만 했다(`continue-on-error`·재시도·테스트 제외·`t.Skip`·`//lint:ignore` 전부 안 씀). 프로덕션 코드는 건드리지 않았다 — `runSettingCacheWorker` 가 "재연결 시 알림 유실은 30초 TTL 로 한정된다"고 스스로 문서화한 설계이고, 프로덕션 NOTIFY 는 부팅 한참 뒤 관리자 조작으로 나가므로 결함이 아니라고 봤다. 위 (3) 의 시계 취약 테스트도 고치지 않았다(ideas.json 에 남김).
- 다음 역할이 조심할 것: 이 테스트들은 `MOINA_TEST_POSTGRES_DSN` 이 있어야 돌고, 없으면 조용히 `t.Skip` 된다 — 증거로 쓸 때는 `--- SKIP` 0줄을 확인할 것(최종 확인: 전 패키지 0줄, 새 6케이스 6/6 PASS). 재현하려면 CI 와 같은 조건이 필요하다: `postgres:16-alpine` + `taskset -c 0-3` + `GOMAXPROCS=4` + 컨테이너 `--cpuset-cpus 0-3`. 단독 패키지만 돌리면 이 경합은 보이지 않는다. 새 장벽은 `pg_stat_activity.query='LISTEN moina_settings'` 문자열에 의존하므로 pgx 가 LISTEN 을 보내는 방식이나 store 의 채널 이름이 바뀌면 20초 뒤 `설정 변경 LISTEN이 준비되지 않았습니다` 로 실패한다(조용히 넘어가지는 않는다).
- [러너 14:20] improve timeout — 단계 제한 시간 초과
- [러너 14:20] brief accepted — 채택 — 과제서의 "수용 기준 1"(네 단계 전부 로컬 재현)을 그대로 수행한 결과 정찰의 가설 순위가 전부 틀렸음이 출력�
- [러너 14:20] improve error — error: agent produced no result (TIMEOUT )
- [러너 14:20] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(low). base 를 `origin/main`(d23f75d)으로 잡아 실제 diff 4파일을 전부 읽었고, throwaway `postgres:16-alpine`로 직접 돌려 확인했다: `go vet` exit 0, 대상 두 테스트 PASS(하위 6/6), `go test -race -count=1 ./...` 전 패키지 ok, httpapi `--- SKIP` 0줄, `make check` 120 route 통과.
- 테스트가 변경을 정말 지나는지 확인했다 — `admin_report_resolve_...:189,196` 두 케이스는 수정 전 코드에서 각각 404/200 을 받으므로 반드시 실패한다(진짜 회귀 테스트). `settings_cache_...` 장벽은 `store.go:421·445`의 `LISTEN moina_settings` 문자열과 `settings_cache.go:98`의 `invalidateAll`→LISTEN 순서를 코드로 확인해 주장이 성립함을 검증했다. tx 패턴은 `posts.go:221`·`social.go:89`와 동일.
- 못 본 것: frontend·staticcheck·e2e·image 잡, CI 로그(구현자와 같은 이유로 접근 불가). 즉 "CI 실패의 진짜 범인이 이 테스트 하나뿐인가"는 나도 독립 확인하지 못했다 — 머지 후 `Source tests` 가 몇 회 연속 녹색인지 다음 회차가 반드시 볼 것.
- 승인이어도 남는 우려(상세는 review.json): ① 장벽이 "이 DB에 LISTEN 중인 backend 존재"만 보므로 같은 DSN 에 실제 인스턴스가 붙어 있으면 무력화된다 — 기준선 count 증가를 기다리는 편이 단단하다. ② 프로덕션 worker 에는 같은 부팅 창이 남아 있다(30초 TTL 로 한정, 의도적 보류). ③ `api/openapi.yaml:641` PATCH 설명이 "메모 없으면 400" 규칙을 빠뜨렸다(alias 두 곳에는 있음).
- 되돌리기: migration·외부 상태 변경 없음. 두 커밋 모두 revert 로 온전히 복구된다.
- [러너 14:25] review approved — 리뷰 승인 (risk=low)
