# 과제서 — 2026-10-16 (base main@879082e)

- 과제: 채널 북마크 `Reorder` 가 주석이 약속한 dense 재배열을 하지 않고 한 행만 UPDATE 해, `sort_order` 가 중복되고 북마크가 요청한 자리에 놓이지 않는 것 수정 (가치 3 / 위험 2 / 작업량 S)

- 왜: `server/internal/bookmarks/service.go:181-196` 의 `Reorder` 주석은 "re-shuffles the rest of the channel's bookmarks so positions are dense and conflict-free … a simple two-shot update" 라고 적혀 있지만, 실제 본문은 **대상 1행의 `sort_order` 만 바꾸는 단일 `UPDATE`** 뿐이다(두 번째 shot 이 없다). 그래서 같은 채널 안에 같은 `sort_order` 가 둘 이상 생기고, 순서는 `List`(`service.go:43-62`)의 타이브레이크 `ORDER BY sort_order ASC, create_at ASC` 가 결정한다 — 즉 **먼저 만들어진 북마크가 항상 앞에 남아** 사용자가 끌어다 놓은 위치가 무시되거나 한 칸씩 어긋난다. 고치면 `POST /api/v4/channels/{channelID}/bookmarks/{bookmarkID}/sort_order`(`router.go:844`) 가 실제로 요청한 자리에 북마크를 놓고, 채널의 `sort_order` 가 중복 없는 조밀한 값으로 유지된다.

  구체 실패(확인된 코드로부터 연역, 아직 테스트로 관찰하지 않음 — 구현자가 RED 로 먼저 확정할 것):
  A(create_at=1, sort=1), B(2, 2), C(3, 3) 인 채널에서 C 를 맨 앞으로 옮기려고 `sort_order=0`(또는 1) 을 보내면,
  - `sort_order=1` 인 경우: A=1, C=1, B=2 → List 는 `(1,t1)A, (1,t3)C, (2,t2)B` = **A, C, B**. 요청은 C, A, B.
  - 반대로 A 를 맨 뒤(3)로 보내면 A=3, C=3 → List 는 B, **A**, C. 요청은 B, C, A.

- 수용 기준:
  1. 재배열 호출 뒤 그 채널의 활성(`delete_at=0`) 북마크 `sort_order` 집합이 **중복 없이 조밀**하다(0..n-1).
  2. 대상 북마크가 요청한 위치에 실제로 놓인다 — `List` 가 돌려주는 **순서**로 관찰한다. 앞으로 옮기기, 뒤로 옮기기, 제자리 유지 세 경우를 모두 고정할 것.
  3. 범위 격리: 같은 DB 의 **다른 채널** 북마크와 **소프트 삭제된**(`delete_at>0`) 행의 `sort_order`·`update_at` 은 한 글자도 바뀌지 않는다.
  4. 없는 id / 다른 채널의 id 는 여전히 `bookmarks.ErrNotFound` → 핸들러 404 `api.bookmark.not_found`(`compat_wave_handlers_late.go:1257-1261`). 기존 `bookmark_scope_postgres_test.go` 가 고정한 채널 스코프 계약을 깨지 말 것.
  5. 범위를 벗어난 `sort_order`(음수, n 이상)는 에러가 아니라 양 끝으로 **클램프**된다(현재도 에러가 아니므로 응답 계약 유지).
  6. 테스트가 증명할 것: 수정 **전** 에 2·1 이 RED 이고(위 A/B/C 고정표본으로 실제 순서를 출력), 수정 후 GREEN 이며, 재배열 SQL 을 원래 단일 UPDATE 로 되돌리면 같은 서브테스트만 같은 출력으로 다시 RED 가 된다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/bookmarks/service.go:Reorder` — 본문을 트랜잭션 기반 dense 재배열로 교체하고, **주석을 실제 구현과 일치시킬 것**(지금은 주석이 거짓이다). 권장 구현은 이 저장소에 이미 있는 관용구를 그대로 따르는 것:
    `server/internal/sidebar/service.go:573-591` 의 `UpdateOrder` 가 `UPDATE … SET sort_order=x.sort_order, update_at=$1 FROM unnest($2::text[], $3::int[]) AS x(id, sort_order) WHERE … AND <소유자 스코프>` 한 문장으로 전체 순서를 다시 쓴다. 북마크 쪽은 (a) `s.db.Pool.Begin(ctx)` 으로 tx 를 열고 (b) `SELECT id FROM channel_bookmarks WHERE channel_id=$1 AND delete_at=0 ORDER BY sort_order ASC, create_at ASC`(= `List` 와 **같은** 정렬) 로 현재 순서를 읽고 — `rows.Err()` 를 반드시 확인할 것, 이 저장소가 10-08 에 고친 바로 그 함정이다 — (c) 대상 id 가 없으면 `ErrNotFound` 로 조기 반환, (d) 슬라이스에서 빼서 클램프한 위치에 다시 끼운 뒤 (e) 위 `unnest` 한 문장으로 `0..n-1` 을 쓰고 `WHERE b.channel_id=$4 AND b.delete_at=0` 로 스코프를 걸고 (f) commit. `store.DB` 는 `Pool *pgxpool.Pool`(`store/store.go:36-38`) 하나뿐이므로 `Pool.Begin` 이 바로 쓸 수 있다.
  - 신규 `server/internal/bookmarks/service_postgres_test.go` — **이 패키지의 첫 테스트**(`go test ./internal/bookmarks` 는 지금 `[no test files]`). 하네스는 `server/internal/sidebar/service_postgres_test.go:661-701` 의 `newSidebarTestDB` 를 그대로 베껴 올 것: `MOYRO_TEST_POSTGRES_DSN` 없으면 `t.Skipf`, 고유 스키마 생성 + `search_path` + `t.Cleanup` 의 `DROP SCHEMA … CASCADE`. 마이그레이션은 `store.Migrate` 로 올린다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd server && go test -race -count=1 -v ./internal/bookmarks` — DSN 없이 돌리면 skip 된다. **skip 을 통과로 착각하지 말 것**: `-v` 로 skip 0 을 눈으로 확인할 것.
  - 실제 DB: 컨테이너 `moyro-pg-improve`(포트 55433)는 **지금 `Exited (255)` 상태**다(이번에 `docker ps -a` 로 확인). `docker start moyro-pg-improve` 로 되살린 뒤 `MOYRO_TEST_POSTGRES_DSN` 을 **자식 프로세스 환경에만** 주어 돌릴 것(DSN 을 출력·문서·커밋에 남기지 않는다). 같은 이름의 `moyro-pg-55433`(Created) 와 55433 포트가 겹치니 둘 다 띄우지 말 것.
  - `cd server && go test -race -p 1 -count=1 ./...` (수 분; 직전 회차 기준 50패키지 ok)
  - `cd server && go build ./... && go vet ./...`, `gofmt -l internal/bookmarks`, `git diff --check`, 루트에서 `bash scripts/check-source-sizes.sh`
  - 웹은 건드리지 않으므로 webapp 검증 불필요(그리고 `webapp/tsconfig.json` 의 `@types/node` TS2345 오탐은 이 회차와 무관하다 — 2026-09-09 교훈).

- 위험과 피할 것:
  - **인덱스 base 결정이 이 과제의 유일한 설계 판단이다.** Mattermost 의 같은 엔드포인트는 0-based 인덱스이고 CLAUDE.md 가 "Mattermost 호환 경로·응답 모양 보존" 을 요구하므로 **0-based 로 가라**. 다만 `Create`(`service.go:93-98`)는 `COALESCE(MAX(sort_order),0)+1` 이라 **첫 북마크가 1 로 시작**한다 — 재배열 전 채널은 1-based, 재배열 후에는 0-based 조밀이 된다. 이 비대칭은 `List` 가 상대 순서만 쓰므로 무해하지만 **과제서의 가정이므로 코드 주석에 적어 둘 것**. (여기가 틀리면 깨지는 자리다. 구현자가 1-based 가 낫다고 판단하면 상수 하나만 바꾸면 되지만, 그 판단 근거를 커밋 메시지에 남길 것.)
  - `Create` 의 `MAX+1` 채번, `List` 의 정렬·타이브레이크, `Patch` 의 `sort_order` 직접 설정 경로는 **건드리지 말 것**. 이번 과제는 `Reorder` 한 함수다. `Patch` 로 `sort_order` 를 직접 넣는 경로도 충돌을 만들 수 있지만 그건 별개 후보다(범위 확대 금지).
  - 핸들러(`compat_wave_handlers_late.go:1239-1276`)는 **고치지 말 것**: 응답 `{"status":"OK"}`, 감사 `audit.ActionBookmarkReorder` 의 `details{channel_id, sort_order}`, WS `channel_bookmark_sort_changed` 의 `data{bookmark_id, sort_order}` 는 모두 요청값을 그대로 echo 하므로 서비스 수정으로 바뀌지 않는다. 같은 파일의 `ok, _ := h.channels.IsMember` 403 위장은 **이번 범위 밖**이다(보류 아이디어로 남아 있다).
  - 최근 네 회차 중 셋이 httpapi 상태코드/본문 디코드 계열이었다. 이 과제는 **서비스 계층 동작 수정**이므로 그 계열을 반복하지 않는다 — 상태코드 재분류를 끼워 넣지 말 것.
  - 보호 경로(auth/session/migrations/.github/workflows) 와 `webapp/`, `docs/` 는 건드리지 않는다. 스키마 변경 없음(기존 `channel_bookmarks.sort_order` 컬럼만 쓴다).
  - `update_at`: 지금은 대상 1행만 bump 되지만 dense 재배열 뒤에는 **위치가 바뀐 모든 행**이 bump 된다. 의도된 변화이므로 테스트에서 "움직이지 않은 다른 채널 행은 unchanged" 로만 고정하고, 같은 채널 안의 update_at 변화는 깨지는 단언으로 만들지 말 것.

- 차선 후보: **`postsByIDsReactions`(`compat_wave_handlers_late.go:304-356`)의 세 `continue` 중 `ListForPost` 장애만 500 `api.reaction.list.app_error` 로 분리** — DB 장애가 200 `{}`("반응 없음") 으로 보고되는 것. 단, 10-04·10-05·10-08 과 같은 "장애를 정상 응답으로 접는 httpapi 핸들러" 계열이므로 1순위가 성립하지 않을 때만 고를 것. 그보다 먼저 고려할 것: 격리 스키마 장애 주입 관용구(`newOperationsTestDB` + RENAME/VIEW, `ALTER TYPE`, `DROP TABLE`)를 `httpapi` 테스트 헬퍼 주석에 문서화 (가치 2 / 위험 1 / S, 프로덕션 0파일).
