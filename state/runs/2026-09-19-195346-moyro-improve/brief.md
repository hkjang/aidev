# 과제서 — 2026-09-19 moyro

- 과제: 사이드바 카테고리 Update/Delete 가 검증 없이 저장하고 없는 행을 400 으로 내던 것 정리 (가치 3 / 위험 1 / 작업량 S)
- 왜: `sidebar.Update` 는 클라이언트가 보낸 `display_name`(빈 문자열 포함)·`sorting`(임의 문자열)을 검증 없이 UPDATE 하고, 기본 카테고리(favorites/channels/direct_messages)의 이름도 덮어쓰게 허용한다(`Create` 는 빈 이름을 거부하고 Mattermost 는 비 custom 타입의 display_name 을 보존한다). 또 `Update`/`Delete` 가 없는·남의 카테고리를 만나면 `pgx.ErrNoRows` 를 그대로 올려 핸들러가 400 `"no rows in result set"` 으로 답하고(같은 ID 로 `Get` 은 404), DB 장애도 400 으로 나간다. 웹앱은 `sorting: "alpha"|"recent"|"manual"` 유니온으로 읽으므로 저장된 이상 값은 화면 쪽을 깨뜨릴 수 있다(웹앱의 실제 폴백 동작은 미확인).
- 수용 기준:
  1) `PUT …/channels/categories/{id}` 에 `display_name: "   "` (custom 카테고리) 또는 `sorting: "bogus"` 를 보내면 400 이고 행이 바뀌지 않는다(`update_at` 포함).
  2) 기본 타입 카테고리(favorites/channels/direct_messages)에 다른 `display_name` 을 보내도 저장된 이름은 그대로이고 나머지 필드(sorting/muted/collapsed/channel_ids)는 적용된다 — Mattermost `updateSidebarCategories` 와 같은 정책. 응답 200.
  3) 존재하지 않거나 다른 사용자·다른 팀의 category id 로 `PUT`·`DELETE` 하면 404 (`Get` 과 동일). 기본 카테고리 `DELETE` 는 여전히 400. 그 밖의 DB 오류는 500.
  4) `sorting` 이 빈 문자열이면 400 이 아니라 기존 값 유지(Mattermost 는 `""` 도 IsValid 통과 — 미확인이면 "빈 값 = 유지" 로 하고 주석에 적을 것).
  5) 테스트: `server/internal/sidebar/service_postgres_test.go` 에 통합 테스트 1~2건 — (a) 빈 이름·bogus sorting 이 `ErrInvalid`(새 sentinel) 로 거부되고 행이 그대로인지, 기본 카테고리 이름이 보존되는지, (b) 남의/없는 id 가 `ErrNotFound`(또는 `pgx.ErrNoRows`) 로 오는지. 핸들러 쪽은 `errors.Is` 분기가 404/400/500 을 고르는 것을 기존 `request_body_test.go` 의 `sidebarOrderRequest` 방식(nil 서비스 대신 실제 DB 없이 안 되면 통합 테스트로만) 으로 최소 1건.
- 건드릴 파일:
  - `server/internal/sidebar/service.go:Update` — 시작부에 검증: `cat.Sorting` ∈ {"", alpha, recent, manual} 아니면 `fmt.Errorf("%w: sorting", ErrInvalid)`; UPDATE 를 `display_name = CASE WHEN type='custom' THEN $1 ELSE display_name END`, `sorting = COALESCE(NULLIF($2,''), sorting)` 로 바꾸고, custom 인데 `strings.TrimSpace(cat.DisplayName)==""` 이면 UPDATE 전에 타입을 읽어 거부(타입은 `SELECT type … FOR UPDATE` 한 번, 없으면 `ErrNotFound`). 새 sentinel `var ErrInvalid = errors.New("sidebar: invalid category")`, `ErrNotFound = errors.New("sidebar: category not found")` 를 파일 상단 const 블록 옆에 두고, `Create` 의 빈 이름 오류와 `Delete` 의 "only custom" 오류도 `ErrInvalid` 로 감싸기, `Delete`/`Get` 의 `pgx.ErrNoRows` 는 `ErrNotFound` 로 바꾸기(`Get` 을 쓰는 핸들러는 이미 무조건 404 라 영향 없음).
  - `server/internal/httpapi/compat_wave_handlers_early.go:updateSidebarCategory`, `updateSidebarCategoriesBulk`, `deleteSidebarCategory`, `createSidebarCategory` — 오류 분기 도우미 `writeSidebarError(w, prefix, err)` 하나 추가: `errors.Is(err, sidebar.ErrNotFound)` → 404, `sidebar.ErrInvalid` → 400, 그 외 500. 오류 ID 문자열(`api.sidebar.update.app_error` 등)은 유지.
  - `server/internal/sidebar/service_postgres_test.go` — 기존 `seedSidebarFixture`/`newSidebarTestDB` 재사용해 테스트 추가.
- 검증 명령 (저장소 루트 기준, `server/` 에서):
  - `go vet ./...`
  - `docker run -d --name moyro-sidebar-pg -e POSTGRES_USER=moyro -e POSTGRES_PASSWORD=moyro -e POSTGRES_DB=moyro -p 55432:5432 postgres:16-alpine` 뒤 `MOYRO_TEST_POSTGRES_DSN='postgres://moyro:moyro@127.0.0.1:55432/moyro?sslmode=disable' go test -race -p 1 ./internal/sidebar/ ./internal/httpapi/` (DSN 없이 돌리면 통합 테스트가 조용히 skip 되므로 반드시 DSN 을 넣을 것; 전체는 `go test -race -p 1 ./...`, 수 분 걸림). 끝나면 컨테이너 제거.
  - `bash scripts/check-source-sizes.sh` (루트에서)
  - 웹 변경 없음 → webapp 빌드 불필요.
- 위험과 피할 것:
  - `Update` 의 채널 목록 교체(`replaceChannelsTx`)·`UpdateOrder` 는 2026-09-17 회차가 막 고친 자리 — 로직을 건드리지 말고 호출만 그대로 둘 것.
  - `updateSidebarCategoriesBulk` 는 카테고리마다 별도 tx 라 중간 실패 시 앞 항목이 남는다(패키지 주석의 "all-or-nothing" 과 어긋남). 이번엔 고치지 말고 과제 밖으로 — 서비스 시그니처 변경이 필요해 M 이 된다.
  - 미머지 브랜치 auto/2026-09-07-0240(preferences)·auto/2026-09-08-2109(reminders)·auto/2026-09-09-0201(ws presence)·auto/2026-09-16-0812(mail, 28파일) 는 sidebar·`compat_wave_handlers_early.go` 를 건드리지 않음(diff --stat 로 확인) — 충돌 없음.
  - `muted` 카테고리가 소속 채널을 실제로 mute 하는 Mattermost 부수효과는 이 저장소에 없으며 과제 밖.
  - 응답 JSON 모양·경로는 그대로(`/api/v4` 호환 유지). 오류 ID 문자열도 유지.
- 차선 후보: `postacks`/channel-views 등 30여 곳의 `ok, _ := h.channels.IsMember` 가 DB 오류를 403 으로 위장 — `requireChannelMember(w, r, cid, uid) bool` 도우미 하나로 500/403 분리 (전 사이트 일괄이면 L 이라, `compat_wave_handlers_final.go` 5곳 + 통합 테스트 1건으로 S 범위 한정 가능).
