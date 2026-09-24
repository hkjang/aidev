# 과제서 (2026-09-24, base main@4ca2ff9 / v0.2.35)

- **과제**: 채널 북마크 patch/delete 가 URL 의 채널을 무시하고 bookmarkID 만으로 동작하는 교차채널 권한 구멍 막기 (가치 4 / 위험 1 / 작업량 S)

- **왜**: `PATCH|DELETE /api/v4/channels/{channelID}/bookmarks/{bookmarkID}` 는 caller 가 **URL 의 channelID** 멤버(또는 채널 관리자)인지만 확인한 뒤 `bookmarks.Patch(ctx, bid, …)` / `bookmarks.Delete(ctx, bid)` 를 부르는데, 서비스의 두 SQL 은 `WHERE id=$1 AND delete_at=0` 뿐이라 채널 조건이 없다 — 즉 아무 공개 채널 하나의 멤버가 bookmark UUID 만 알면 **자신이 속하지 않은(비공개 포함) 다른 채널의 북마크를 고치거나 지울 수 있다**. 같은 이유로 성공 후 WS 이벤트가 실제 소유 채널이 아니라 URL 채널로 방송돼, 진짜 그 북마크를 보고 있는 사람들의 화면은 갱신되지 않고 엉뚱한 채널에 유령 이벤트가 간다. 권한 경계를 리소스의 실제 채널로 맞추면 두 문제가 같이 없어진다.

- **수용 기준**:
  1. 채널 B 의 북마크 id 를 `PATCH /channels/A/bookmarks/{id}` 로 보내면 **404 `api.bookmark.not_found`** 가 나가고 DB 의 그 행은 `display_name`·`update_at` 까지 그대로다(호출자가 A 의 멤버여도, A 의 channel_admin 이어도, system_admin 이어도 동일 — 경로가 가리키는 리소스가 A 에 없다는 사실은 권한과 무관하다).
  2. `DELETE /channels/A/bookmarks/{id}`(id 는 B 소유)도 같은 404 이고 그 행의 `delete_at` 은 0 으로 남는다. 감사 로그(`audit.ActionBookmarkUpdate`/`ActionBookmarkDelete`)도 남지 않는다.
  3. 올바른 채널로 부른 patch/delete/reorder 는 지금과 똑같이 200 + 기존 응답 모양 + 기존 이벤트 이름(`channel_bookmark_updated`/`channel_bookmark_deleted`)으로 동작하고, 없는 bookmarkID 는 여전히 404 `api.bookmark.not_found`(상태 코드·id 문자열·본문 모양 불변).
  4. 테스트는 **수정 전 코드에서 실제로 실패**해야 한다 — 즉 지금은 1)·2) 자리에서 200 이 나오고 B 의 행이 바뀌거나 지워지는 것을 먼저 확인한 뒤 수정할 것. 실제 PostgreSQL + 실제 `bookmarks.Service` + 실제 핸들러로 증명하고, 손으로 만든 대역 서비스로 대체하지 말 것.
  5. `bookmarks.Service` 레벨에서 막았다면 서비스 단위로도 "다른 채널 id 를 주면 ErrNotFound + 행 불변" 이 증명돼야 한다.

- **건드릴 파일**:
  - `server/internal/bookmarks/service.go:Patch` — 시그니처를 `Patch(ctx, channelID, id string, p Patch)` 로 바꾸고 `WHERE id=$1 AND channel_id=$2 AND delete_at=0` 으로 채널을 조인(이미 같은 파일의 `Reorder` 가 정확히 이 모양이다 — 관례를 맞추는 것이지 새로 만드는 게 아니다). `RowsAffected()==0` → 기존 `ErrNotFound` 그대로.
  - `server/internal/bookmarks/service.go:Delete` — 동일하게 `Delete(ctx, channelID, id string)` 으로 채널 조건 추가.
  - (선택) `service.go:Get` — 호출처가 delete 핸들러 하나뿐이다. 시그니처를 바꾸지 말고 그대로 두고 핸들러가 `b.ChannelID != cid` 를 보게 해도 되고, 아래 핸들러 수정으로 충분하면 손대지 않아도 된다.
  - `server/internal/httpapi/compat_wave_handlers_late.go:1142 patchChannelBookmark` — `h.bookmarks.Patch` 호출에 `cid` 를 넘긴다. 기존 `errors.Is(err, bookmarks.ErrNotFound) → 404` 분기가 그대로 새 케이스를 받는다(핸들러 분기 구조를 새로 짜지 말 것).
  - `server/internal/httpapi/compat_wave_handlers_late.go:1190 deleteChannelBookmark` — 이미 `h.bookmarks.Get(ctx, bid)` 로 행을 읽고 있으므로 `b.ChannelID != cid` 면 404 `api.bookmark.not_found` 로 조기 반환(소유자/채널관리자 판정 **이전에**). 그리고 `h.bookmarks.Delete` 에 `cid` 를 넘긴다.
  - `server/internal/httpapi/bookmark_scope_postgres_test.go` (신규) — 이 패키지의 기존 관례를 그대로 따를 것: `newOperationsTestDB(t)`(`native_operations_postgres_test.go:86`, DSN 없으면 skip, 격리 스키마) + `store.Migrate` + `seedSidebarHandlerFixture(t, ctx, db)`(`sidebar_handlers_postgres_test.go:171` — user-a / team-main / chan-alpha 를 만든다). 두 번째 채널(chan-beta)과 그 멤버십은 테스트 안에서 직접 INSERT 한다. 참고 형태는 `preferences_errors_postgres_test.go`.
  - (선택) `server/internal/bookmarks/service_test.go` (신규) — 지금 이 패키지에는 테스트 파일이 **하나도 없다**. 수용 기준 5 를 여기서 증명하면 값이 크다.

- **검증 명령** (server/ 에서):
  - `go vet ./...`
  - `go build ./...`
  - 실제 DB: `MOYRO_TEST_POSTGRES_DSN=...` 를 준 뒤 `go test -race -p 1 -count=1 ./internal/bookmarks/ ./internal/httpapi/`
    — **DSN 이 없으면 새 테스트는 통째로 skip 되고 `ok` 만 나온다. 그 `ok` 를 통과 근거로 쓰지 말 것.** 프로필대로 로컬 `postgres:16-alpine` 컨테이너를 포트 **55433** 으로 띄울 것(55432 는 점유된 적 있음).
  - 전체: `go test -race -p 1 ./...` (수 분, `-p 1` 유지)
  - 루트에서 `bash scripts/check-source-sizes.sh` — `compat_wave_handlers_late.go` 가 커지므로 상한을 넘지 않는지 확인(이 파일 1603줄, 상한 58000바이트 기준은 early.go 가 54325/58000 로 가장 빡빡하다).

- **위험과 피할 것**:
  - **응답 계약을 넓히지 말 것.** 새 상태 코드나 새 오류 id 를 만들지 말고 기존 `404 api.bookmark.not_found` 를 재사용한다. "다른 채널 것" 을 403 으로 내면 북마크 id 의 존재 여부가 새어 나가므로 404 가 맞다.
  - `listChannelBookmarks`·`createChannelBookmark`·`reorderChannelBookmark` 는 **이미 채널로 스코프돼 있다**(Create 는 cid 로 INSERT, Reorder 의 SQL 은 `channel_id=$2`). 건드리지 말 것.
  - 같은 네 핸들러에 있는 `ok, _ := h.channels.IsMember(...)`(late.go:1058/1090/1150/1241)의 **DB 오류 403 위장은 이번 범위 밖**이다. 최근 회차들이 연달아 오류매핑만 손봤으므로 같이 끼워 넣지 말 것 — 별건이고, 아래 차선 후보로 남긴다.
  - `channel_bookmarks` 스키마·`store/migrations` 는 건드리지 않는다. 마이그레이션 없이 끝나는 과제다.
  - 웹(`webapp/`)은 이 과제와 무관하다. 손대지 말 것 — 교훈에 있듯 `webapp/tsconfig.json` 의 `types` 부재 때문에 typecheck 가 남의 이유로 깨질 수 있다.
  - 운영자 교훈: 대역(fake service)·소스 문자열 검사로 증명하지 말고 실제 배선(실제 핸들러 → 실제 서비스 → 실제 DB)을 통과시킬 것. 그리고 "같은 값을 읽는 경로가 여럿" 규칙대로, patch 와 delete **양쪽 모두** 를 고치고 양쪽 모두 테스트할 것(한쪽만 막으면 다른 쪽으로 같은 공격이 그대로 된다).
  - **미확인**: 이번 정찰에서는 `go vet`/`go test` 를 실제로 돌리지 못했다(샌드박스가 해당 명령을 거부). 위 근거는 전부 소스 읽기로 확인한 것이며, 코드 인용(`WHERE id=$1 AND delete_at=0`, 핸들러 라인 번호)은 직접 열어 본 내용이다. 빌드 상태 자체는 미확인이니 구현자가 먼저 baseline `go build ./...` 를 한 번 돌릴 것.

- **차선 후보**: 채널 북마크 4개 핸들러(late.go:1058/1090/1150/1241)의 `ok, _ := h.channels.IsMember` 가 DB 오류를 403 으로 위장하는 것을 500 으로 분리 — `callerIsSystemAdmin` 예외가 붙어 있지만 판단은 단순하다(멤버십을 읽지 못하면 권한을 알 수 없으므로 500). 선례 761c3ae(`api.context.permissions.app_error` id 유지 + 고정 메시지 500)를 그대로 따르면 S. 1순위가 성립하지 않을 때만 고를 것.
