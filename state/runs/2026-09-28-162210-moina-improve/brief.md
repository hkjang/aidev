- 과제: 수정 과제 — 지난 회차가 TIMEOUT(`fix-round: error: agent produced no result`)으로 끝나 main 에 들어오지 못한 `a4082e1`(resolveReport 500 분리)과 `93b0234`(설정 캐시 테스트 LISTEN 장벽)을 이번 브랜치에 재적용하고 CI `Source tests` 네 단계를 로컬에서 재현해 통과를 확인한다 (가치 4 / 위험 2 / 작업량 S)

- 왜: 이 저장소의 CI `Source tests` 잡 `Go formatting, tests and vet` 단계는 2026-09-20·09-23·09-28 네 번 간헐 실패했고, 지난 회차가 그 원인을 `TestPostgreSQLSettingChangeReachesAnotherInstance` 의 LISTEN 경합으로 특정해 `93b0234` 로 고쳤다. 그런데 회차 자체가 에이전트 TIMEOUT 으로 error 판정을 받아 **코드는 맞는데 아무것도 머지되지 않았다** — main 은 여전히 `d23f75d` 이고, flaky 테스트도 `5915beb` 의 신고 처리 계약 수정도 그대로 빠져 있다. 재적용하면 다음 회차마다 같은 간헐 실패에 발목잡히는 일이 끝나고, 관리자가 "신고를 찾을 수 없습니다" 오탐 토스트를 보는 결함도 함께 닫힌다.

- 확인한 사실 (이 세션에서 직접 열어 봄):
  - 로컬 브랜치 `auto/2026-09-28-1333`(origin 에도 있음)이 정확히 `d23f75d → a4082e1 → 93b0234` 선형 스택이다. 즉 이번 브랜치의 base 와 같은 부모라서 **cherry-pick 충돌이 없어야 한다**(`git log --oneline -4 93b0234` 로 확인).
  - `a4082e1` diffstat: `api/openapi.yaml` +3, `backend/internal/httpapi/admin.go` +27/-3, `backend/internal/httpapi/admin_report_resolve_postgres_integration_test.go` +234 (신규). 프로덕션 Go 파일은 `admin.go` **하나**.
  - `93b0234` diffstat: `backend/internal/httpapi/settings_cache_postgres_integration_test.go` +32 뿐. 프로덕션 파일 0개. 내용은 `waitForSettingListener` 헬퍼(`pg_stat_activity` 에서 `query='LISTEN moina_settings'` 인 다른 pid 가 생기기를 20ms 간격·20초 deadline 으로 대기)와 worker goroutine 기동 직후의 호출 한 줄.
  - 그 장벽이 의존하는 두 가지를 소스에서 확인했다: `backend/internal/store/store.go:421` `const settingChangeChannel = "moina_settings"`, `store.go:445` 가 `LISTEN `+settingChangeChannel 를 그대로 `Exec` 하므로 `pg_stat_activity.query` 문자열이 `LISTEN moina_settings` 와 정확히 일치한다. `store.go:109` 에 `func (s *Store) Pool() *pgxpool.Pool` 가 있어 테스트가 쓰는 `repository.Pool()` 도 실재한다. → 장벽은 그대로 두면 된다, 고칠 것 없음.
  - 장벽이 "다른 pid 의 listener" 를 세므로 남의 listener 를 오인할 가능성을 점검했다: `runSettingCacheWorker` 호출처는 `settings_cache.go:98`(정의)·`outbox.go:191`(프로덕션 백그라운드 그룹)·이 테스트 한 곳뿐이고, `RunBackground`/outbox 그룹을 켜는 테스트는 저장소에 없다(`grep -rn` 결과 0건). CI 가 `moina_ci` 한 DB 를 네 패키지에 공유하지만 이 worker 를 띄우는 테스트는 httpapi 의 이 하나라서 오인 위험이 없다.

- 수용 기준:
  1) 이번 브랜치에 `a4082e1` 과 `93b0234` 의 내용이 그대로 올라가 있고(순서 유지), 두 커밋이 건드린 4개 파일 외에 변경이 없다 — `git diff --stat d23f75d..HEAD` 가 `api/openapi.yaml`·`admin.go`·`admin_report_resolve_postgres_integration_test.go`·`settings_cache_postgres_integration_test.go` 네 줄만 보여야 한다.
  2) CI `Source tests` 잡의 네 단계를 로컬에서 재현해 전부 exit 0 이다(아래 검증 명령). 특히 `go test -race ./...` 에서 `--- FAIL` 0줄이고 `--- SKIP` 0줄이어야 한다(DSN 을 실제로 줬다는 증거).
  3) `a4082e1` 이 들고 온 테스트가 계약을 여전히 증명한다: `admin_report_resolve_postgres_integration_test.go` 의 6케이스가 6/6 PASS 이고, 최상위 `TestPostgreSQL*` 개수가 37건이다(v0.1.38 기준 36건 + 이 파일 1건).
  4) `TestPostgreSQLSettingChangeReachesAnotherInstance` 가 CI 를 흉내 낸 좁은 조건에서 반복 통과한다 — `taskset -c 0-3` + `GOMAXPROCS=4` 로 **5회** 연속 무실패. (지난 회차가 39회를 돌려 예산을 태웠다. 인과는 이미 결정론적으로 증명돼 원장에 남아 있으니 **재증명하지 말 것** — 회귀 확인만 하면 된다.)

- 건드릴 파일: 새로 쓸 코드는 없다. cherry-pick 으로 들어오는 4개가 전부다.
  - `backend/internal/httpapi/admin.go:resolveReport` — UPDATE·`moderation_actions` INSERT·commit 을 한 transaction 으로 묶고 실패를 500 `storage_error`("신고를 처리할 수 없습니다")로, `RowsAffected()==0` 은 기존 404·기존 문구로 (`a4082e1` 그대로).
  - `api/openapi.yaml` — PATCH `/admin/reports/{reportID}`·POST `.../resolve`·`.../reject` 세 경로에 `description` 한 줄씩. `responses` 목록은 늘리지 않는다(저장소 관례).
  - `backend/internal/httpapi/admin_report_resolve_postgres_integration_test.go` — 신규 6케이스. 저장 실패는 대역이 아니라 테스트 전용 `BEFORE UPDATE ON reports` / `BEFORE INSERT ON moderation_actions` 트리거가 sentinel 신고 id 만 거부하는 방식.
  - `backend/internal/httpapi/settings_cache_postgres_integration_test.go:waitForSettingListener` — LISTEN 장벽 (`93b0234` 그대로).

- 검증 명령 (이 저장소에서 실제로 도는 것. 저장소 루트 = worktree 루트):
  ```
  # throwaway DB — CI source 잡과 같은 PostgreSQL 16 을 쓸 것 (image 잡만 17)
  docker run -d --name moina-recon-pg -p 127.0.0.1:55432:5432 \
    -e POSTGRES_DB=moina_ci -e POSTGRES_USER=moina -e POSTGRES_PASSWORD=moina-ci-password \
    postgres:16-alpine
  export MOINA_TEST_POSTGRES_DSN='postgres://moina:moina-ci-password@127.0.0.1:55432/moina_ci?sslmode=disable'

  # CI Source tests 4단계
  make check                                   # 1단계 (OpenAPI route 120개 유지 확인)
  make fmt && (cd backend && go test -race ./... && go vet ./...)   # 2단계 — 여기가 간헐 실패하던 곳
  (cd backend && go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...)
  npm ci --prefix frontend && npm run lint --prefix frontend && npm test --prefix frontend

  # 수용 기준 4 — 경합 조건 5회
  cd backend && for i in 1 2 3 4 5; do \
    GOMAXPROCS=4 taskset -c 0-3 go test -race -count=1 -run TestPostgreSQLSettingChangeReachesAnotherInstance ./internal/httpapi/ || break; done
  ```
  `--- SKIP` 이 한 줄이라도 나오면 DSN 이 안 걸린 것이므로 그 실행은 증거가 못 된다.

- 위험과 피할 것:
  - **가장 큰 위험은 코드가 아니라 검증 예산이다.** 지난 회차는 코드가 맞았는데도 에이전트 TIMEOUT 으로 error 가 됐다. 인과 증명(LISTEN 전 NOTIFY 유실 probe, interleaving 재현, 39회 반복)은 이미 끝나 원장에 있다 — 다시 하지 말고 수용 기준 2·4 만 채우고 끝낼 것.
  - `93b0234` 을 "느슨하게" 만들지 말 것: `t.Skip`·재시도·deadline 연장·`//lint:ignore`·`continue-on-error` 는 금지다. `ci.yml`·`release.yml` 은 **읽기만** 한다.
  - `93b0234` 의 장벽 문자열 `query='LISTEN moina_settings'` 는 `store.go:421`·`445` 에 묶여 있다. store 쪽 채널 이름이나 LISTEN 문장을 손보면 장벽이 조용히 20초 t.Fatal 로 바뀐다 — 이번 회차에 store 는 건드리지 말 것.
  - `api/openapi.yaml` 의 `responses` 를 늘리지 말 것(관례: 오류 계약은 `description` 문장으로만).
  - 프런트(`AdminReportsPage.tsx`)는 `readableError` 로 서버 message 를 그대로 띄우므로 무변경이다 — vitest·시각 회귀·e2e 는 이 회차 범위 밖. 시각 회귀 베이스라인은 24개 화면이 허용치에 아슬아슬하므로 건드리지 말 것.
  - 보호 경로 회피: auth.go·oidc.go·mcp_oauth.go·store/migrations 는 이번 변경에 포함되지 않는다. `admin.go` 는 관리자 권한 경로이지만 권한 판정은 미들웨어에 있고 `a4082e1` 은 handler 본문만 바꾼다.
  - 미확인: 이번 정찰 샌드박스에서는 `node`·`go`·`curl` 실행이 전부 권한 거부라 **아무 명령도 실행하지 못했다**. 위 검증 명령은 지난 회차들이 실제로 돌린 것을 근거로 적었고, 이 세션에서 재확인한 것은 아니다. cherry-pick 이 정말 무충돌인지도 git 메타데이터(선형 스택)로 추론한 것이며 실제 `git cherry-pick` 은 실행하지 않았다.

- 차선 후보: CI `Go formatting, tests and vet` 단계에 `if: failure()` 로 `--- FAIL` 줄만 `$GITHUB_STEP_SUMMARY` 에 적기 (가치 4 / 위험 2 / S). 1순위가 이미 머지돼 있거나 cherry-pick 이 예상과 달리 크게 충돌하면 이것을 고를 것 — 이번 회차 이전의 가장 큰 비용이 "어느 테스트가 깨졌는지 알아내기" 였고(Actions 로그는 admin 권한 없이 못 읽는다), `ab36254` 가 e2e 쪽에 같은 접근을 이미 썼다. 진단 출력만 더하므로 워크플로 완화가 아니다. 주의: `go test` 출력을 `tee` 로 붙잡을 때 `set -o pipefail` 과의 상호작용을 확인할 것.
