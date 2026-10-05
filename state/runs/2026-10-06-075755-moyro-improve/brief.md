# 과제서 — 2026-10-06 (moyro, base main@4005197 / v0.2.47)

- **과제: `postsByIDsReactions` 가 DB 장애를 "이 게시글에는 반응이 없다" 로 보고하는 것 차단 — 형제 핸들러 `listReactions` 가 이미 쓰는 500 으로 맞추기 (가치 3 / 위험 2 / 작업량 S)**

- **왜:** `server/internal/httpapi/compat_wave_handlers_late.go:304-350`(`postsByIDsReactions`)의 루프는 `h.reactions.ChannelForPost` / `h.channels.IsMember` / `h.reactions.ListForPost` 세 호출의 오류를 전부 `continue` 로 삼키고 마지막에 `writeJSON(w, 200, out)` 을 한다. 그래서 DB 순단 중에는 **200 + `{}`** 가 나가고, 공식 Mattermost 클라이언트는 그것을 "이 게시글들에 반응이 하나도 없다" 로 읽어 화면의 반응을 지운다 — 거부도 아니고 재시도도 안 하는, 조용히 틀린 데이터다. 같은 세 서비스 메서드를 같은 순서로 읽는 **형제 단건 핸들러** `handlers.go:2488-2507`(`listReactions`)은 `ListForPost` 오류를 이미 `500 api.reaction.list.app_error` 로 내고 있다. 두 경로가 한 값을 다르게 읽던 것을 맞추는 일이다(2026-10-02 `createPost` vs `fireIncomingWebhook` 회차와 같은 모양).
  - 부수적으로 같은 함수의 `body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))`(:311)이 읽기 오류를 버리고 64 KiB 에서 **조용히 자른다** — 잘린 JSON 은 `400 "expected ids array or {ids}"`("본문 모양이 틀렸다")를 받지만 실제 원인은 "너무 크다". `request_body.go:11-19` 는 "every collection-shaped body reads through decodeCollectionBody" 라고 스스로 적어 두었는데 이 한 곳이 예외다.

- **수용 기준:**
  1. `ListForPost` 가 실패하는 요청은 200 `{}` 가 아니라 **500** 이고, id 는 형제가 이미 쓰는 **`api.reaction.list.app_error`** 를 재사용한다(새 id 를 만들지 말 것).
  2. `ChannelForPost` 가 **진짜 장애** 오류를 올리면 500. `pgx.ErrNoRows`(= 없는 게시글 / `delete_at!=0` 인 삭제된 게시글) 와 `ch == ""` 는 **지금처럼 조용히 건너뛴다** — 핸들러 주석(:299-303)이 선언한 가시성 필터 계약이다. 판정은 `errors.Is(err, pgx.ErrNoRows)` 로 한다(이 패키지의 기존 관용구: `compat_wave_handlers.go:223/230`, `handlers.go:1794`).
  3. `h.channels.IsMember` 가 오류를 올리면 500. **멤버가 아니어서(`ok == false`, 오류 없음) 걸러지는 기존 동작은 바이트 단위로 그대로** — 이 핸들러에는 system-admin 예외가 없으므로(북마크 핸들러와 달리) 정책 선행 결정이 필요 없다.
  4. 64 KiB(또는 새 상한) 를 넘는 본문은 400 이 아니라 **413**. `io.LimitReader` 를 `http.MaxBytesReader(w, r.Body, …)` 로 바꿔 절단이 `*http.MaxBytesError` 로 드러나게 하고 `errors.As` 로 413 / 그 외 읽기 오류 400 으로 나눈다. **권장: 상한을 패키지 상수 `collectionBodyMaxBytes`(1 MiB)로 올려 위 패키지 주석의 계약에 맞출 것** — 오늘 잘못된 400 을 받던 65 KiB~1 MiB 본문만 동작이 달라지고, 하류의 `len(ids) > 200` 절단이 작업량을 그대로 묶어 둔다. 상한 변경이 범위를 넓힌다고 판단하면 64 KiB 를 유지하고 413 만 고쳐도 기준 4 는 충족이다(둘 중 무엇을 골랐는지 커밋 메시지에 적을 것).
  5. **보존**: `[...]` 와 `{ids:[...]}` 두 `json.Unmarshal` 폴백, `len(ids)==0` → `200 {}`, `len(ids) > 200` → `ids[:200]` 절단, 멤버십 캐시(`memberOf`) 동작, 정상 요청의 응답 바이트.
  6. **테스트가 증명해야 하는 것**: 실제 PostgreSQL 격리 스키마 위에서 ① 정상 2개 게시글 → 200 + 반응 맵(양성 대조) ② `reactions` 테이블만 숨겨 `ListForPost` 를 실패시키면 500 ③ `posts` 를 숨겨 `ChannelForPost` 를 실패시키면 500 ④ `channel_members` 를 숨겨 `IsMember` 를 실패시키면 500 ⑤ 삭제된 게시글(`delete_at!=0`) 은 **여전히** 조용히 빠지고 나머지 id 는 200 으로 돌아온다 ⑥ 비회원 채널의 게시글은 **여전히** 조용히 빠진다 ⑦ 상한 초과 본문 413 ⑧ 깨진 JSON 400 ⑨ 빈 배열 200 `{}`. 그리고 **형제 경로 대조**: 같은 `reactions` 숨김 상태에서 단건 `GET /posts/{postID}/reactions` 도 500 `api.reaction.list.app_error` 를 내는 것을 같은 테스트 안에서 단언해, 두 경로가 같은 입력을 같은 값으로 읽는 것을 end-to-end 로 고정할 것.
  7. **인과 역방향 확인**: 수정한 분기를 하나씩 되돌리면 해당 서브테스트만 깨지고 보존 단언(⑤⑥⑨, 정상 응답)은 하나도 깨지지 않는 것을 확인해 과잉 수정이 아님을 보일 것.

- **건드릴 파일 (프로덕션 1개):**
  - `server/internal/httpapi/compat_wave_handlers_late.go` — `postsByIDsReactions`(:304-350) 본문만. 본문 읽기(:311-320)를 `http.MaxBytesReader` + `errors.As(*http.MaxBytesError)` 로, 루프(:329-350)의 세 `continue` 를 "장애는 500 / `pgx.ErrNoRows`·비회원·빈 channel_id 는 continue" 로. `pgx` import 추가 필요(`github.com/jackc/pgx/v5` — 같은 패키지 `compat_wave_handlers.go` 가 이미 import 하므로 go.mod 변경 없음). `errors`·`io`·`net/http` 는 이미 import 되어 있다(:10-17 확인).
  - `server/internal/httpapi/posts_ids_reactions_fault_postgres_test.go` (신규) — 위 9+1 서브테스트.
  - **이 둘 뿐이다.** `handlers.go`, `request_body.go`, `router.go`, 서비스 패키지는 **한 글자도 건드리지 말 것**.

- **검증 명령:**
  - `cd server && go test -race -p 1 -count=1 ./internal/httpapi/` — DSN 없이 돌리면 DB 테스트가 skip 되어 0.2초에 끝난다(= DB 통과로 오인 금지). 실제 DB 로 돌릴 것: `MOYRO_TEST_POSTGRES_DSN` 을 주면 이 패키지는 50초대가 정상. `-run TestPostsByIDsReactions -v` 로 **skip 0** 을 눈으로 확인할 것.
  - `cd server && go build ./... && go vet ./...`
  - `cd server && gofmt -l internal/httpapi/compat_wave_handlers_late.go internal/httpapi/posts_ids_reactions_fault_postgres_test.go` (출력 없어야 함 — 패키지 전체 `gofmt -l` 은 기존 9파일을 출력하니 그것으로 자기 변경을 판정하지 말 것)
  - `bash scripts/check-source-sizes.sh`
  - 웹 변경이 없으므로 webapp 명령은 불필요하다. 다만 러너 게이트가 webapp typecheck 를 돌린다면 `@types/node`/`useDraft.test.tsx` TS2345 는 **이 변경과 무관한 기존 환경 결함**이다(교훈 2026-09-09).

- **테스트 배선 (지시: 손으로 만든 대역 금지):**
  - `newOperationsTestDB(t)` + `store.Migrate` 로 격리 스키마를 잡고, **실제** `reactions.New(db)` / `channels.New(...)` 를 주입한다. chi 라우터는 `router.go:770-773` 의 등록(`r.Post("/posts/ids/reactions", h.postsByIDsReactions)`)과 단건 `GET /posts/{postID}/reactions` 등록을 글자 그대로 복사한다.
  - 장애 주입은 **`ALTER TABLE <t> RENAME TO <t>_hidden`** 을 쓸 것 — `DROP TABLE` 은 FK 때문에 거부될 수 있고, RENAME 은 FK 를 건드리지 않으며 숨긴 테이블을 그대로 조회해 "실패한 요청이 아무것도 쓰지 않았다" 까지 단언할 수 있다(2026-10-02 회차가 확정한 관용구).
  - 기존 형제 테스트가 `h.audit` 를 nil 로 두는 함정이 있지만 **이 핸들러는 감사 행을 남기지 않으므로** audit 서비스는 불필요하다 — 감사 단언을 쓰지 말 것(이 경로에는 없다).
  - 기존 `internal/reactions/service_postgres_test.go:82-94`(`TestChannelForPostIgnoresDeletedPosts`) 가 "삭제된 게시글은 오류" 를 이미 고정하고 있다 — 기준 ⑤ 의 전제이므로 이 테스트를 깨지 말 것.

- **위험과 피할 것:**
  - `request_body.go` 의 공용 헬퍼(`decodeCappedBody`/`decodeCollectionBody`/`decodeOptionalCollectionBody`/`drainCappedBody`/`tooManyBatchItems`) 를 **수정하거나 추가하지 말 것**. 이 핸들러는 `[...]` 와 `{ids:[...]}` 두 모양을 받으려고 원시 바이트가 필요하므로 공용 디코더로 치환할 수 없다 — 분기는 핸들러 안에 둔다. 나머지 `_ = decodeCappedBody` 호출자 14곳은 이번 범위 밖이다.
  - `listReactions`(handlers.go:2488-2507) 의 **자기 결함은 고치지 말 것**: 거기서는 `ChannelForPost` 장애가 404 로, `IsMember` 장애가 403 으로 접힌다. 그것까지 손대면 파일 2개 + 지난 5회차가 연속으로 다룬 404/403 재분류 계열을 다시 여는 것이 된다. 이번에는 **읽기만 하고 수용 기준 ⑥ 의 대조 단언으로 현재 동작을 고정**하고, 분리는 ideas.json 의 별도 pending 항목으로 남겼다.
  - 보호 경로를 피할 것: `auth`/`session`/`oidcauth`, `store/migrations`, `.github/workflows` — 이번 과제는 전부 불필요하다.
  - 새 오류 id 를 만들지 말 것(기준 1). 상태 코드가 두 실패를 가르는 것이고 id 는 형제와 공유한다.
  - `memberOf` 캐시의 `ok, cached := memberOf[ch]` 2값 읽기를 단순화하지 말 것 — `false` 를 캐시하는 것이 의도다.
  - **미확인**: DB 컨테이너 `moyro-pg-improve`/호스트 55433 이 지금 살아 있는지 이번 세션에서 재확인하지 않았다(이전 두 회차에서는 `docker start moyro-pg-improve` 로 살아났다). 수정 전 실제 응답이 정확히 `200 {}` 인지도 실행으로 확인하지 않았다 — 코드 읽기에서 나온 추론이다. 먼저 RED 를 관찰해 이 추론을 확정할 것.

- **차선 후보:** `patchCustomProfileField`(`compat_wave_handlers_final.go:1245`)의 `_ = decodeCappedBody` 로 인한 no-op 패치 거짓 성공 — 모든 포인터 필드가 nil 이 되어 200 + `ActionCustomFieldPatch` 감사행이 남는다. 단 **선행 결정이 필요**하다: 빈 본문 `{}`(현재 200 no-op) 과 깨진 본문을 응답에서 구별할 계약을 먼저 정해야 하고, 2026-10-05 머지분과 같은 파일이다. 그 결정이 부담되면 3순위로 `postsByIDsReactions` 의 본문 상한만(수용 기준 4 단독) 떼어 내 제출할 것.
