# 회차 노트 2026-09-28-162210-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:22] base pinned — main@d23f75d
- [러너 16:22] autonomy release — 

## 정찰 노트
- 우선 과제(지난 회차 error)를 그대로 받았다. 확인한 핵심은 "고칠 코드가 남아 있지 않다"는 것 — CI 간헐 실패의 원인·수정(`93b0234`)과 계약 수정(`a4082e1`)이 브랜치 `auto/2026-09-28-1333`에 d23f75d 위 선형 스택으로 온전히 있고, 회차가 error가 된 이유는 코드가 아니라 에이전트 TIMEOUT이었다. 그래서 과제를 "재적용 + CI 4단계 로컬 재현"으로 좁혔다. 새 아이디어(Go 테스트 실패 요약)를 제친 것은 그것이 진단 편의이고 이번 우선 과제는 이미 진단이 끝난 유실 복구여서다.
- `93b0234`의 장벽은 읽어 검증했다: `store.go:421/445`가 `LISTEN moina_settings`를 그대로 Exec하므로 `pg_stat_activity.query` 문자열 일치가 성립하고, `store.go:109`에 `Pool()`이 있고, `runSettingCacheWorker`를 켜는 테스트가 이 하나뿐이어서(outbox 그룹을 켜는 테스트 0건) 남의 listener를 오인할 여지도 없다. 장벽을 손볼 필요 없음.
- 추측으로 남긴 것: cherry-pick 무충돌은 git 메타데이터(선형 스택)로 추론했고 실제 `git cherry-pick`은 돌리지 않았다. 검증 명령도 지난 회차 실적을 근거로 적은 것이며 이 세션에서 재확인하지 못했다.
- 이번 샌드박스는 `node`·`go`·`curl`·`docker` 실행이 전부 권한 거부여서 **아무 명령도 실행하지 못했다**(git 읽기·grep·파일 읽기만). GitHub API도 못 봤으므로 실패 run 번호는 지난 회차 기록을 그대로 인용했다.
- 구현자가 조심할 것: 검증을 늘리지 말 것. 인과 증명은 끝났으니 경합 조건 반복은 5회로 끊고, `ci.yml`·`release.yml`은 읽기만 하고, `t.Skip`·재시도·deadline 연장으로 통과시키는 것은 금지다.
- [러너 16:28] scout done — 수정 과제 — 지난 회차가 TIMEOUT(`fix-round: error: agent produced no result`)으로 끝나 main 에 들어오지 못한 `a4082e1`

## 구현 노트
- 무엇을 왜: 지난 회차가 TIMEOUT error 로 끝나 유실된 `a4082e1`(resolveReport 를 한 transaction 으로 묶고 저장 실패를 404 → 500 `storage_error` 로 분리) + `93b0234`(설정 캐시 전파 테스트의 LISTEN 장벽)을 cherry-pick 으로 재적용했다(`c5e7cf9`·`a2a0f65`). 새로 쓴 코드는 없다. `git diff 93b0234 HEAD` 가 비어 원래 검증된 스택과 트리가 byte-identical 이고, base 대비 변경은 예상한 4개 파일뿐(프로덕션은 `admin.go` 하나).
- 확신 없는 곳·검증 못 한 것: (1) `93b0234` 의 flaky 수정은 **확률적** 결함이라 5회 회귀 통과로만 확인했다 — 인과 증명은 지난 회차 것을 인용했고 이번에 재현하지 않았다(예산이 지난 회차의 실패 원인이었다). CI 의 진짜 부하는 로컬 `taskset` 흉내와 다를 수 있다. (2) e2e·시각 회귀·`make image` 는 돌리지 않았다 — 이 회차 범위 밖이고 프런트는 무변경이다. (3) `waitForSettingListener` 는 `pg_stat_activity.query == 'LISTEN moina_settings'` 문자열 일치에 묶여 있다. 읽어서 `store.go:421`(채널명)·`445`(LISTEN Exec) 와 맞는 것을 확인했지만, store 쪽 채널명이나 LISTEN 문장을 나중에 손대면 이 장벽이 조용히 20초 `t.Fatal` 로 바뀐다.
- 일부러 하지 않은 것: 워크플로 완화 일체 없음 — `t.Skip`·재시도·deadline 연장·`//lint:ignore`·`continue-on-error` 를 하나도 넣지 않았고 `ci.yml`·`release.yml`·`store` 는 읽기만 했다. OpenAPI 는 관례대로 `responses` 를 늘리지 않고 `description` 세 줄만. 같이 눈에 띈 결함(새 500 출구 4곳이 pg err 를 버려 운영자 로그에 원인이 없음)은 파일 수를 늘리지 않으려고 `ideas.json` 에 다음 회차 후보로 남겼다.
- 다음 역할이 조심할 것: httpapi·store 의 `TestPostgreSQL*` 37건은 **`MOINA_TEST_POSTGRES_DSN` 이 있어야 돈다**. 없으면 조용히 `t.Skip` 되고 `go test` 는 그대로 exit 0 이므로 증거가 못 된다 — `-v` 로 `--- SKIP` 0줄을 확인할 것. 그리고 첫 실행에서 대부분 패키지가 `(cached)` 로 나왔다: **캐시된 `ok` 는 증거가 아니다**, `-count=1` 로 다시 돌려야 한다(이번에 실제로 그렇게 해서 httpapi 5.5초 실행을 확인했다). CI source 는 PostgreSQL **16**(image 잡만 17)이고 `moina_ci` 한 DB 를 네 패키지가 `-p 4` 로 공유한다.
- [러너 16:40] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(선형 스택이라 cherry-pick 이 실제로 무충돌, 장벽이 의존하는 `store
- [러너 16:40] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 이번 샌드박스는 `go`·`node`·`docker` 가 모두 열려 있어서 실제로 돌렸다: throwaway `postgres:16-alpine` DSN 으로 backend 전체 `go test -count=1 -race -p 4 ./...` 통과, `scripts/check-openapi-routes.mjs` 120 method/path 통과, `go vet`·`gofmt -l` 무출력. 실제 diff 는 4파일 293줄이다(워크트리 로컬 `main` ref 가 v0.1.28 로 낡아 `git diff main...HEAD` 는 80파일로 보인다 — base 는 `d23f75d`=origin/main).
- 구현자가 안 했다고 적은 인과 증명을 대신 재현했다: `d23f75d` 로 별도 worktree 를 만들어 새 테스트만 얹으니 `admin_report_resolve_..._test.go:205` 에서 주장한 그대로 실패했다(poison UPDATE → 404 not_found, poison moderation_actions → 200 에 report 는 resolved·action 0행). HEAD 에서는 둘 다 500 storage_error 로 PASS. 테스트는 변경 경로를 확실히 지난다.
- 못 본 것: 설정 캐시 flake 자체는 확률적이라 재현하지 않았고, e2e·시각 회귀·`make image` 는 돌리지 않았다. 대신 장벽을 코드로 확인했다 — `store.go:445` 의 `LISTEN moina_settings` 문자열 일치, `settings_cache.go:117~118` 의 `invalidateAll` 이 `ListenSettingChanges`(119) 앞이라는 주석 주장까지 사실이다.
- 승인이어도 남는 우려 3개: (1) 장벽 질의가 `current_database()` 의 아무 backend 나 세는데 CI 는 `moina_ci` 를 네 패키지가 공유한다 — 지금은 listener 를 켜는 테스트가 이 하나뿐이라 안전하지만 `RunBackground`(outbox.go:191)를 켜는 테스트가 생기면 남의 listener 로 장벽이 만족되어 flake 가 되돌아온다. (2) 장벽의 20초 `t.Fatal` 은 조용한 flake 를 단단한 실패로 바꾼 것이므로 pool 지연에 민감하다. (3) 새 500 출구 4곳이 pgx err 를 버리고 `writeError`(server.go:694)는 로그를 안 남겨 운영자는 원인 없는 `storage_error` 만 본다 — 기존 관례와 같고 이미 ideas.json 에 있다.
- 릴리즈가 알아야 할 것: VERSION 이 아직 v0.1.38 이라 bump 가 필요하다. OpenAPI PATCH description 은 `400 invalid_resolution` 사유로 상태 allowlist 만 적었고 빈 `resolution`·2000 rune 초과는 빠졌다(alias 두 곳은 적혀 있다) — 불완전하지만 이전엔 description 자체가 없었으니 후퇴는 아니다. migration·프런트 무변경이라 revert 는 코드 되돌리기로 완결된다.
- [러너 16:45] review approved — 리뷰 승인 (risk=low)
- [러너 16:45] pr created — https://github.com/hkjang/moina/pull/35
- [러너 16:54] ci passed — 검사 2개 모두 success
- [러너 16:54] merge done — a2a0f65
- [러너 17:10] release published — v0.1.39
- [러너 17:16] assets verified — v0.1.39 자산 1개 (이전 v0.1.38: 1)
