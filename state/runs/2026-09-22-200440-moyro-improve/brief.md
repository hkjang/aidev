# 과제서 (정찰, 2026-09-22, base main@960d3a1)

- 과제: `getPreferenceByName` 이 DB 장애까지 404 로 내는 것을 404/500 으로 분리 (가치 3 / 위험 1 / 작업량 S)

- 왜: `server/internal/httpapi/compat_wave_handlers_early.go:80 getPreferenceByName` 은
  `h.prefs.GetByName` 이 돌려준 오류를 종류와 무관하게 전부
  `writeError(w, 404, "api.preference.get.not_found", "preference not found")` 로 바꾼다.
  서비스(`server/internal/preferences/service.go:GetByName`)는 이미 "행 없음"만 `pgx.ErrNoRows`
  로 구분해 돌려주는데 핸들러가 그 구분을 버리므로, 연결 실패·테이블 장애·컨텍스트 취소도
  클라이언트에게는 "그런 설정은 없다"로 보이고 공식 클라이언트는 기본값으로 조용히 넘어가
  테마·사이드바 설정이 사라진 것처럼 렌더된다(같은 파일의 `listAllPreferences`·
  `listPreferencesInCategory` 는 이미 500 을 내므로 한 리소스 안에서도 읽기 경로마다 답이 다르다).
  고치면 장애가 장애로 보이고 404 는 진짜 "행 없음"만 뜻하게 된다.

- 수용 기준:
  1) `GET /api/v4/users/{user_id}/preferences/{category}/name/{name}` 이 행이 없을 때 지금과
     똑같이 404 + 오류 id `api.preference.get.not_found` + 메시지 `preference not found` 를 낸다
     (응답 본문 모양·오류 id 불변 — 클라이언트 계약이다).
  2) 같은 요청이 DB 장애(행 없음이 아닌 오류)를 만나면 500 을 낸다. 오류 id 는
     `api.preference.get.not_found` 그대로 두어도 되고(2026-09-20 의 `getSidebarCategory` 선례가
     그렇게 했다) 500 전용 id 를 새로 만들어도 되지만, 404 경로의 id 는 바꾸지 말 것.
  3) 401/403 게이트(`requireUserParamAccess`)와 200 성공 응답(`Preference` JSON)은 그대로다.
  4) 테스트가 증명할 것: 실제 PostgreSQL + 실제 `preferences.Service` + 실제 핸들러로
     (a) 없는 (category,name) → 404 와 기존 오류 id, (b) `DROP TABLE preferences CASCADE` 뒤
     같은 요청 → 500, (c) 있는 행 → 200 + 값 일치. 수정 전 코드에서 (b) 가 실제로 404 로
     실패하는 것을 확인해 붙일 것(러너 기록에 남기기).

- 건드릴 파일:
  - `server/internal/httpapi/compat_wave_handlers_early.go:80 getPreferenceByName` — 오류 분기를
    `errors.Is(err, pgx.ErrNoRows)` → 404, 그 외 → 500 으로. `errors` 는 이미 import 되어 있고
    `pgx` 는 이 파일에 아직 없으니 `github.com/jackc/pgx/v5` 를 추가한다(같은 패키지의
    `native_ai.go:220`, `native_keys.go:133`, `native_operations.go:199` 가 이미 같은 방식으로
    `errors.Is(err, pgx.ErrNoRows)` 를 쓰므로 새 관례가 아니다). 서비스 계약은 바꾸지 말 것 —
    `preferences.Service.GetByName` 은 지금처럼 `pgx.ErrNoRows` 를 그대로 돌려주면 된다
    (`server/internal/pluginhost/mattermost_api_compat.go:189 GetPreferenceForUser` 가 같은
    메서드를 쓰며 오류를 그대로 AppError 로 감싸므로, 센티널을 새로 만들면 그쪽까지 번진다).
  - 새 테스트: `server/internal/httpapi/preferences_errors_postgres_test.go` (신규 파일 권장).
    템플릿은 `server/internal/httpapi/sidebar_handlers_postgres_test.go:96
    TestGetSidebarCategoryReports404OnlyForMissingRows` 를 거의 그대로 따라가면 된다 —
    `newOperationsTestDB(t)`(`native_operations_postgres_test.go:86`) → `store.Migrate` →
    `seedSidebarHandlerFixture(t, ctx, db)`(`sidebar_handlers_postgres_test.go:171`, user-a 를
    만들어 준다) → `h := &handlers{prefs: preferences.New(db)}`. 요청은
    `sidebarCategoriesRequest` 처럼 `userIDKey` 컨텍스트 + chi `userID`/`category`/`name`
    URL 파라미터를 직접 붙여 만들면 되고, actor == target 이면 `requireUserParamAccess` 가
    `h.auth` 없이 통과하므로 다른 서비스는 주입할 필요가 없다. 테스트용 행 하나는
    `INSERT INTO preferences (...)` 로 직접 넣을 것(스키마 컬럼은 서비스 쿼리 기준
    user_id, category, name, value — update_at 등 추가 컬럼 유무는 마이그레이션에서 확인).

- 검증 명령 (server/ 디렉터리에서):
  - `go vet ./...`
  - `go build ./...`
  - DSN 없이: `go test -count=1 ./internal/httpapi/` (DB 테스트는 skip — 이것만으로 "통과"라고
    주장하지 말 것)
  - 실제 DB: `docker run -d --rm -e POSTGRES_PASSWORD=postgres -p 55433:5432 postgres:16-alpine`
    후 `MOYRO_TEST_POSTGRES_DSN=postgres://postgres:postgres@127.0.0.1:55433/postgres` 를 설정하고
    `go test -race -p 1 -count=1 ./internal/httpapi/` → 이어서 `go test -race -p 1 ./...`
    (전체는 수 분. 포트 55432 는 과거 다른 컨테이너가 점유한 적이 있으니 55433 부터 시도)
  - 저장소 루트에서 `bash scripts/check-source-sizes.sh`
    (`compat_wave_handlers_early.go` 는 직전 회차 기준 53860/58000 으로 여유가 크지 않다 —
     새 테스트는 별도 파일로 두면 이 검사와 무관하다)
  - 웹 변경이 없으므로 webapp 빌드·typecheck 는 돌릴 필요 없음.

- 위험과 피할 것:
  - **같이 건드리지 말 것**: 같은 파일의 `upsertPreferences`(109행)·`deletePreferences`(134행) 의
    400→500 분리. 그쪽은 서비스에 `ErrInvalid` 센티널을 새로 만들어야 하고, 미머지 로컬 브랜치
    `auto/2026-09-07-0240`(`fix: bound what a client can park in the preferences table`)이 그 변경을
    값 상한·`.github/workflows/ci.yml` 수정과 한 덩어리로 들고 있다. 상한과 CI 워크플로는 절대
    건드리지 말 것. (그 브랜치는 origin 에 푸시된 적이 없어 PR 자체가 없다 — 사람이 반려한 것은
    아니지만, 이번 회차는 읽기 경로 한 곳으로만 좁힌다.)
  - **사이드바 재정렬(`updateSidebarCategoryOrder`, 같은 파일 575행) 은 손대지 말 것.**
    응답이 저장된 순서가 아니라 입력 배열을 에코하는 문제는 남아 있지만, 이미 구현된 커밋
    `bddb463` 이 `origin/auto/2026-09-21-0304` 에 있고 main 에 아직 안 들어왔다. 재구현 금지.
  - 보호 경로(auth/session/oidcauth/store/migrations/.github/workflows) 는 이 과제에서 전혀
    필요 없다. `requireUserParamAccess` 안의 `ok, _ := h.auth.HasRole`(handlers.go:240) 도
    이번 범위 밖이다.
  - 효과 없는 변경 금지: 오류 id 나 404 본문을 "정리"하겠다고 바꾸면 클라이언트 계약이 깨진다.
    실제로 달라지는 것은 "행 없음이 아닌 오류일 때의 상태 코드" 하나여야 한다.
  - 검증 함정: 로컬에 `MOYRO_TEST_POSTGRES_DSN` 이 없으면 DB 테스트가 통째로 skip 된다
    (정찰이 `go test -count=1 ./internal/httpapi/` 를 돌려 ok 를 받았지만 그건 skip 포함이다).
    `DROP TABLE` 은 `newOperationsTestDB` 가 잡아 주는 격리 스키마 안에서만 해야 한다.
    웹 typecheck 에서 `useDraft.test.tsx` 의 Timeout/number 오류가 보이면 그건 알려진 기존
    함정이지 이 변경 탓이 아니다(이번엔 웹을 안 건드리므로 돌릴 일도 없다).

- 차선 후보: **`uploadTeamImage` 가 10MB 초과 본문에도 200 + 업로드 감사 로그를 남기는 것**
  (`server/internal/httpapi/compat_wave_handlers_final.go:135`). `http.MaxBytesReader` 로 자른 뒤
  `_, _ = io.Copy(io.Discard, r.Body)` 로 오류를 버리고 `{"status":"OK"}` 와
  `audit.ActionTeamImageUpload` 를 남긴다 → 초과 업로드를 413 으로 돌려주고 감사 기록을 남기지
  않게 한다. 다만 이 핸들러는 애초에 아무것도 저장하지 않는 스텁이라 "성공 200" 자체가 이미
  사실이 아니므로, 비평에서 "효과 없는 정리"로 몰릴 소지가 있다. 1순위가 성립하지 않을 때만
  고르고, 고른다면 감사 로그가 실제로 남지 않는 것까지 테스트로 증명할 것.
