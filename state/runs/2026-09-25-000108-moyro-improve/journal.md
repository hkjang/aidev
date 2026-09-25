# 회차 노트 2026-09-25-000108-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:01] base pinned — main@1b6aa79
- [러너 00:01] autonomy release — 

## 구현 노트
- 무엇/왜: 글 **생성**은 `postcommand.stripReservedProps` 로 클라이언트가 보낸 예약 props(`override_username`, `from_plugin`/`plugin_id`, `_moyro_post_type`, `approval_request_id` 등)를 떨어뜨리는데, **편집** 두 경로(`PUT /posts/{id}`, `PUT /posts/{id}/patch`)는 본문 props 를 그대로 `posts.Update` 에 넘겨 작성자가 그 키들을 되집어넣거나(출처 위조·승인 유니크 인덱스 선점) 신뢰 어댑터가 찍은 출처를 지울 수 있었다. 새 `postcommand.MergeEditedProps` 가 두 경로의 규칙을 한 곳에서 소유한다. 커밋 a248f19.
- 확신 없는 곳·검증 못 한 것: (1) `updatePost` 에 **선행 `posts.Get` 한 번이 늘었다** — 편집 핫패스의 추가 왕복이고, 없는 행은 기존 403 에 맡기고 진짜 장애만 500 으로 내도록 `errors.Is(pgx.ErrNoRows)` 로 갈랐지만 이 500 분기 자체는 장애 주입으로 검증하지 못했다(기존 테스트와 새 테스트 모두 정상 경로만 탄다). (2) `MergeEditedProps` 는 예약 키가 없으면 클라이언트 맵을 **그대로 돌려준다**(기존 `stripReservedProps` 와 같은 관례). 호출자가 그 맵을 이후 변형하면 요청 본문의 맵을 건드리는 셈 — 현재 두 호출자 모두 변형하지 않지만 계약으로 못 박지는 않았다. (3) 두 경로의 `post_edited` 이벤트 인코딩이 서로 다르다(`updatePost` 는 `data.post` 가 JSON **문자열**, `patchPost` 는 **객체**). 기존 불일치라 손대지 않았고, 테스트는 두 모양을 다 읽는다 — 웹 클라이언트가 둘 다 처리하는지는 확인 안 했다.
- 일부러 하지 않은 것: `pluginhost.UpdatePost`(mattermost_api_compat.go:471)는 `plugin_id`/`from_plugin`/`_moyro_*` 만 떨어뜨리는 **더 좁은** 목록을 쓴다 — 플러그인은 출처를 정당하게 찍는 신뢰 호출자라 `posts.Update` 직행을 유지했다. 목록을 통일할지는 플러그인 신뢰 경계를 먼저 정해야 해서 ideas.json 에 별건으로 남겼다. 같은 파일에서 본 `patchPost` 의 `updated, _ = h.posts.Get(...)`(되읽기 실패 → 200 + 본문 `null` + 이벤트 누락)도 결정론적 장애 주입 방법을 못 찾아 손대지 않았다.
- 다음 역할이 조심할 것: 새 `server/internal/httpapi/post_edit_props_postgres_test.go` 3건은 **`MOYRO_TEST_POSTGRES_DSN` 이 있어야 돈다**(없으면 `newOperationsTestDB` 가 통째로 skip → `ok` 만 보고 통과라고 하면 안 된다). 이번 검증은 로컬 `moyro-pg-improve`(postgres:16-alpine, 55433)로 했다. `TestPostEditRoutesDropClientSuppliedReservedProps` 는 두 경로가 **같은 위조 `approval_request_id`** 를 보내므로, 수정이 되돌아가면 두 번째 경로가 유니크 인덱스 충돌 500 으로 깨진다 — 그건 테스트 결함이 아니라 버그의 증상이다.
- [러너 00:14] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 테스트가 진짜인지 되돌려 확인했다: 두 핸들러 파일만 main 판으로 바꾸고 돌리니 응답·저장행·WS 이벤트 세 곳에서 위조 키로 FAIL, HEAD 에서는 3건 PASS(실 DB 55433, -v 로 skip 아님 확인). 복원 후 트리 clean.
- 구현 노트의 의심 3개 모두 확인: (1) posts.Get 은 주석과 달리 pgx.ErrNoRows 를 올리므로 errors.Is 분기는 없는 글의 403 을 지키는 데 필수 — 방향은 맞다(다만 500 분기 자체는 여전히 미주입). (2) patchPost 의 "identity" 주석은 정확하다(req.Props 가 nil 이면 props 가 existing.Props 로 채워진 뒤 들어간다). (3) 이벤트 인코딩 불일치는 기존 것이라 그대로 두는 게 맞다.
- 안 본 것: go test -race ./... 전체, 웹 typecheck·build(웹 변경 없음), 브라우저 e2e. 돌린 것은 vet·build·httpapi/postcommand/posts 실 DB·check-source-sizes.
- 승인이어도 남는 우려(릴리즈 노트감): pluginhost/mattermost_api_compat.go:471 은 좁은 제외 목록이라 플러그인 API 로는 approval_request_id·from_webhook·override_username 을 여전히 찍을 수 있다. 그리고 PUT /posts/{id} 에 props 키가 없으면 비예약 props 는 계속 날아간다(usePostActions.ts:188 이 그 경로) — 회귀는 아니고 별건.
- 다음 회차 별건 후보: posts/service.go:634 의 "Returns nil,nil if missing" 주석이 사실과 다르다(다음 호출자가 믿으면 없는 글에 500 을 낸다).
- [러너 00:18] review approved — 리뷰 승인 (risk=low)
- [러너 00:18] pr created — https://github.com/hkjang/moyro/pull/25
- [러너 00:30] ci passed — 검사 3개 모두 success
- [러너 00:30] merge done — a248f19
- [러너 01:06] release published — v0.2.38
- [러너 01:24] assets verified — v0.2.38 자산 1개 (이전 v0.2.37: 1)
