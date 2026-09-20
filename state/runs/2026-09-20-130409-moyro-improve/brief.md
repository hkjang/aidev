# 과제서 2026-09-20 — moyro

- 과제: 사이드바 카테고리 일괄 PUT(`updateSidebarCategoriesBulk`)을 단일 트랜잭션으로 — 두 번째 항목이 거부되면 첫 항목 변경이 남는 부분 적용 제거 (가치 3 / 위험 1 / 작업량 S~M)

- 왜: `PUT /api/v4/users/{uid}/teams/{tid}/channels/categories` 는 Mattermost 드래그-드롭 재정렬이 "새 상태 전체"를 한 번에 보내는 경로인데, 현재 핸들러(`server/internal/httpapi/compat_wave_handlers_early.go:704-739`)가 항목마다 `h.sidebar.Update` 를 따로 호출하고 `Update` 는 호출마다 자기 tx 를 `Begin/Commit` 한다(`server/internal/sidebar/service.go:412-459`). 그래서 배열의 N번째 항목이 404(없는/남의 카테고리)·400(sorting 값 오류, custom 빈 이름)·DB 오류로 실패하면 1..N-1 항목의 display_name·sorting·muted·collapsed·채널 배치는 이미 커밋된 채 오류 응답이 나가고, `sidebar_categories_updated` 이벤트도 안 나가 다른 탭은 그 절반 상태를 모른다. 지난 회차(0665ae3)가 넣은 sentinel(`ErrNotFound`/`ErrInvalid`)·`writeSidebarError` 가 그대로 쓰이므로 응답 코드 규칙은 바뀌지 않고, "전부 적용되거나 아무것도 적용되지 않음"만 새로 보장된다.

- 수용 기준:
  1) `sidebar.Service` 에 `UpdateMany(ctx, userID, teamID string, cats []Category) ([]Category, error)` 가 생기고, 배열 전체가 하나의 tx 안에서 처리된다. 어느 항목이든 `ErrNotFound`/`ErrInvalid`/DB 오류를 내면 tx 가 롤백되어 **앞 항목의 행도 update_at 포함 그대로**이고, 반환 오류는 그 항목의 오류(sentinel 유지 — `errors.Is` 로 판별 가능)다.
  2) 성공 시 반환 슬라이스는 입력 순서와 같고, 각 원소는 커밋 뒤 `Get` 으로 되읽은 값이다(기존 `Update` 가 `s.Get` 을 돌려주는 것과 동일한 모양·`channel_ids` 필터링). 빈 배열이면 tx 를 열지 않고 빈 슬라이스를 돌려준다(현재 핸들러가 빈 배열에 `200 []` 을 내는 동작 유지).
  3) 기존 단건 `Update` 는 시그니처·동작 그대로다(내부만 `updateTx(ctx, tx, …)` 로 뽑아 두 경로가 같은 검증·UPDATE·`replaceChannelsTx` 를 공유). 기존 sidebar 테스트 7개(`TestUpdateRejectsBadFieldsAndKeepsDefaultNames`, `TestUpdateAndDeleteReportMissingCategoriesAsNotFound` 등)가 그대로 통과한다.
  4) 핸들러 `updateSidebarCategoriesBulk` 는 for 루프 대신 `UpdateMany` 한 번을 부르고, 오류는 지금처럼 `writeSidebarError(w, "api.sidebar.bulk.app_error", err)` 로 404/400/500 을 고른다. 응답 JSON 모양·오류 id·`tooManyBatchItems` 두 겹 가드·브로드캐스트 이벤트 이름은 바뀌지 않는다. 함수 위 주석("apply each one inside its own service.Update transaction")을 새 동작에 맞게 고칠 것.
  5) 새 Postgres 통합 테스트(`server/internal/sidebar/service_postgres_test.go`)가 다음을 증명한다: (a) 자기 custom 카테고리 A 의 유효한 변경 + 존재하지 않는 id 를 한 배열로 `UpdateMany` → `ErrNotFound` 이고 `categoryRow(t, ctx, db, A.ID)` 의 display_name·update_at 이 호출 전과 같음, (b) A 유효 변경 + sorting `"bogus"` → `ErrInvalid` 이고 A 그대로, (c) 두 카테고리를 한 번에 바꾸면 둘 다 반영되고 반환 순서가 입력 순서와 같음. 수정 전 코드(루프 방식)에 (a)/(b) 를 대면 실패해야 한다 — 구현자는 이를 한 번 실제로 확인할 것(예: `UpdateMany` 를 임시로 "항목마다 `Update`" 로 바꿔 보기).

- 건드릴 파일:
  - `server/internal/sidebar/service.go:412-459` `Update` — 본문을 `updateTx(ctx context.Context, tx pgx.Tx, userID, teamID string, cat Category) error` 로 뽑고(검증 → `SELECT type … FOR UPDATE` → UPDATE → `replaceChannelsTx`), `Update` 는 Begin → updateTx → Commit → `s.Get`. 새 `UpdateMany` 는 Begin → 각 항목 updateTx → Commit → 각 항목 `s.Get`. sorting 검증(`switch cat.Sorting`)은 tx 열기 전에 전 항목 먼저 돌려도 되고 updateTx 안에서 해도 된다(어느 쪽이든 롤백되므로 결과는 같음; 단순한 쪽으로).
  - `server/internal/httpapi/compat_wave_handlers_early.go:699-739` `updateSidebarCategoriesBulk` — 루프를 `h.sidebar.UpdateMany` 호출로 교체, 주석 갱신. `out` 은 `[]sidebar.Category` 그대로(UpdateMany 가 값 슬라이스를 돌려주면 그대로 씀).
  - `server/internal/sidebar/service_postgres_test.go` — 위 5) 테스트 1~2개 추가. 기존 헬퍼 `newSidebarTestDB`, `sidebarTestContext`, `seedSidebarFixture`, `categoryRow`(`sidebarRow` 에 update_at 필드가 있는지는 미확인 — 없으면 SELECT 로 직접 읽을 것) 재사용.
  - (선택) `server/internal/httpapi/sidebar_errors_test.go` 는 손댈 필요 없음.

- 검증 명령 (저장소 `server/` 에서):
  - `go vet ./...`
  - `go test ./internal/sidebar/ ./internal/httpapi/` (DSN 없으면 DB 테스트가 조용히 skip 되므로 아래가 진짜 검증)
  - 로컬 Postgres: 55432·55434·55444·55450 은 다른 컨테이너가 쓰고 있음(2026-09-20 기준). 지난 회차처럼 `docker run -d --name moyro-test-pg -e POSTGRES_PASSWORD=postgres -p 55433:5432 postgres:16-alpine` 뒤
    `MOYRO_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55433/postgres?sslmode=disable' go test -race -p 1 ./...` (수 분; `-p 1` 필수 — 공유 DB). DSN 의 사용자/비밀번호 형식은 이전 회차 기록 기준이며 헬퍼(`newSidebarTestDB`)가 마이그레이션을 직접 적용하는지는 미확인 — 실패하면 `server/internal/sidebar/service_postgres_test.go:17` 부근의 헬퍼를 읽을 것.
  - 루트: `bash scripts/check-source-sizes.sh` (compat_wave_handlers_early.go·service.go 줄 수 상한 — 지금 594줄인 service.go 가 상한에 닿는지 미확인, 넘으면 `UpdateMany` 를 같은 패키지의 새 파일 `service_bulk.go` 로).
  - 웹 변경 없음 → webapp typecheck/build 는 돌리지 않아도 됨.

- 위험과 피할 것:
  - `replaceChannelsTx`·`UpdateOrder`·`Create`·`Delete`·`Get` 의 SQL 은 건드리지 말 것(지난 두 회차가 이미 정리했고 검증됨). 같은 배열에 같은 category id 가 두 번 오면 같은 tx 안에서 순서대로 두 번 적용돼 마지막이 이김 — 이건 현재 루프와 동일하므로 dedupe 를 추가하지 말 것(불필요한 동작 변경).
  - 단건 `PUT …/categories/{id}` 의 응답·상태 코드·`/api/v4` 경로·오류 id 문자열은 그대로 유지(Mattermost 호환).
  - `Get` 을 tx 안에서 부르지 말 것 — `Get`/`fillChannelIDs` 는 `s.db.Pool` 을 쓰므로 tx 미커밋 상태를 못 본다. 커밋 뒤에 읽어야 한다.
  - 트랜잭션 안에서 항목마다 `SELECT … FOR UPDATE` 를 여러 행에 거는 순서는 클라이언트 배열 순서다 — 같은 사용자의 동시 bulk 요청 둘이 반대 순서로 잠그면 데드락이 이론상 가능하지만 Postgres 가 한쪽을 오류로 끊어 500 이 되고 tx 는 롤백된다. 이번 과제 범위에서는 정렬로 회피하지 말고(순서가 곧 의미) 그대로 둔다. 과제서 밖.
  - 미머지 원격 브랜치 `origin/auto/2026-09-16-0812`(메일)는 sidebar·이 핸들러 파일을 건드리지 않음(diff --stat 확인). 로컬 전용 브랜치 `auto/2026-09-07-0240`(preferences) 은 같은 파일 `compat_wave_handlers_early.go` 의 preferences 구간(42~150줄)만 손대므로 sidebar 구간과 충돌하지 않음.
  - 지난 회차 교훈: webapp typecheck 가 `window.setInterval`/@types/node 로 깨지면 이번 변경 탓이 아님(웹은 손대지 않음).
  - 효과 없는 변경 금지 규칙 — 이 과제는 실제 동작(부분 적용 → 전무/전부)이 바뀌고 테스트 (a)/(b) 가 수정 전 코드에서 실패하므로 해당하지 않음. 그 확인을 꼭 할 것.

- 차선 후보: `getSidebarCategory`(`compat_wave_handlers_early.go:619-632`) 가 DB 장애까지 404 로 내는 것을 `writeSidebarError(w, "api.sidebar.get.not_found", err)` 로 바꿔 `ErrNotFound` 만 404, 그 외 500 (`Get` 이 이미 `ErrNotFound` 를 돌려주므로 한 줄; `sidebar_errors_test.go` 표 테스트에 get 오류 id 케이스 1건 추가). 1순위가 성립하지 않을 때 이걸 고르되, 1순위와 함께 넣어도 위험이 없으니 시간이 남으면 같은 PR 에 포함해도 됨.
