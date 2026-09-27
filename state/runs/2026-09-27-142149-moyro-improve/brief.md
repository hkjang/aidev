# 과제서 — 2026-09-27-142149-moyro-improve (moyro)

- 과제: 업로드 스텁 3개가 본문 상한 초과를 삼키고 200 + 거짓 감사 기록을 남기는 것 차단 (가치 3 / 위험 1 / 작업량 S)

- 왜: `compat_wave_handlers_final.go` 의 세 업로드 핸들러(`uploadTeamImage` 141, `uploadBotIcon` 610, `uploadChunk` 921)는
  `r.Body = http.MaxBytesReader(w, r.Body, N)` 로 상한을 걸어 두고 곧바로 `_, _ = io.Copy(io.Discard, r.Body)` /
  `n, _ := io.Copy(...)` 로 **그 상한 초과 오류를 버린다**. 그래서 상한을 넘긴 요청도 200 `{"status":"OK"}` 를 받고,
  더 나쁘게는 `audit.ActionTeamImageUpload` / `ActionBotIconUpload` / `ActionUploadChunk` 감사 기록이 남는다 —
  운영자가 감사 로그에서 "일어나지 않은 업로드"를 보게 된다. `uploadChunk` 는 여기에 더해 잘린 바이트 수를
  `{"file_offset": n}` 로 성공처럼 돌려주어, 청크 업로드를 이어 쓰는 공식 클라이언트에게 "50MB 를 받았다"고 거짓말한다.
  같은 저장소는 이미 같은 장애에 대한 정답을 갖고 있다 — `internal/httpapi/request_body.go:37 decodeCollectionBody` 가
  `errors.As(err, &tooLarge)`(`*http.MaxBytesError`)로 갈라 `http.StatusRequestEntityTooLarge`(413) 를 내고,
  `handlers.go:3315` 의 이모지 업로드도 413 `api.emoji.create.too_large` 를 낸다. 세 스텁만 이 규칙 밖에 있다.

- 수용 기준:
  1) 상한(각각 10MB / 256KB / 50MB)을 넘긴 본문으로 세 엔드포인트를 호출하면 **413**(`http.StatusRequestEntityTooLarge`)
     이 돌아온다. 오류 id 는 저장소 관례대로 `api.*.too_large` 계열로 새로 만들되(예: `api.team.image.too_large`
     — 단 이 id 는 이미 `handlers.go:891` 이 프로필 이미지 400 에 쓰고 있으니 **재사용하지 말고**
     `api.team.image.upload_too_large` 처럼 충돌 없는 id 를 쓸 것), 상한 이내 요청의 200 응답 본문 모양은 그대로 둔다.
  2) 413 로 끊긴 요청은 **감사 기록을 남기지 않는다**. `audit_logs` 에 해당 action 행이 0건임을 실제 DB 로 단언한다
     (`LogAsync` 는 비동기이므로 성공 케이스의 도착을 먼저 폴링으로 확인한 뒤 실패 케이스의 0건을 단언 — 선례:
     `internal/httpapi/channel_membership_errors_postgres_test.go:89-133` 의 "먼저 성공 감사를 관찰하고 그다음 count 단언").
  3) 상한 이내 정상 요청은 **변하지 않는다**: 세 경로 모두 200, `uploadTeamImage`·`uploadBotIcon` 은
     `{"status":"OK"}`, `uploadChunk` 는 `{"id":<uploadID>,"file_offset":<실제 바이트 수>}`, 감사 기록 1건.
     401/403 분기(`callerCanAdminTeam`, `callerIsSystemAdmin`, `uploadChunk` 의 빈 caller 401)는 건드리지 않는다.
  4) 테스트가 증명할 것: 수정 **전** 코드에서 세 오버사이즈 케이스가 "status = 200, want 413" 으로 실패하고
     감사 행이 1건 남는 것을 먼저 확인(RED)한 뒤, 수정 후 통과. 그리고 한 핸들러의 새 분기만 되돌리면
     그 핸들러의 서브테스트만 깨지는 것을 확인.
  5) (같은 파일·같은 줄의 곁가지, 분리 가능) `uploadBotIcon`(613)·`deleteBotIcon`(625)의 감사 target 이 항상 빈 문자열이다 —
     `router.go:957-959` 는 경로 파라미터를 `{botUserID}` 로 선언하는데 두 핸들러는 `chi.URLParam(r, "botID")` 를 읽는다.
     감사 로그가 "어느 봇인지"를 잃는다. `"botUserID"` 로 고치고 감사 행의 `target` 이 실제 bot id 와 같음을 단언한다.
     (`compat_wave_handlers.go:296`, `late.go`, `handlers.go:3004` 의 `"botID"` 는 `{botID}` 를 선언하는
     **다른 라우트**들이라 정상이다 — 건드리지 말 것.)

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `server/internal/httpapi/request_body.go` — 드레인 전용 헬퍼 1개 추가. 예:
    `func drainCappedBody(w http.ResponseWriter, r *http.Request, limit int64) (int64, bool)` —
    `http.MaxBytesReader` 로 감싸 `io.Copy(io.Discard, ...)` 하고, `errors.As(err, &tooLarge)`(`*http.MaxBytesError`)
    이면 `false` 를 돌려준다(응답은 호출자가 각자 id 로 쓰게 두는 편이 기존 `decodeCollectionBody` 와 대칭이 깨지지
    않는다면 헬퍼가 413 을 직접 써도 된다 — 단 **id 는 호출자가 넘기게** 할 것). 세 상한값은 호출자가 그대로 유지.
  - `server/internal/httpapi/compat_wave_handlers_final.go` —
    `uploadTeamImage`(135-148), `uploadBotIcon`(605-618), `uploadChunk`(915-927) 의 `io.Copy` 무시를 헬퍼 호출로 바꾸고
    **감사 호출보다 앞에서** early-return. 기준 5 를 같이 한다면 `uploadBotIcon`·`deleteBotIcon` 의 URL 파라미터 이름 수정.
  - `server/internal/httpapi/upload_stub_limits_postgres_test.go` (신규) — 실제 PostgreSQL 16 격리 스키마 +
    `store.Migrate` + 실제 `audit.New(db, slog.Default())` + 실제 `handlers` + 실제 chi 라우트로 6 케이스
    (3 엔드포인트 × 정상/초과). 손으로 만든 대역 금지.
    **라우팅 방식**: `NewRouter(cfg, db, hub, host, logger)`(router.go:72)는 전체 서비스 배선을 요구해 무겁다.
    이 패키지의 확립된 관례는 `chi.NewRouter()` 에 필요한 라우트만 등록하는 것이다
    (`sso_exchange_postgres_test.go:42`, `plugin_management_routes_test.go:46`, `oauth_flow_test.go:21` 이 이미 그렇게 한다).
    이 방식을 쓰되 **패턴 문자열은 `router.go:904/957-959` 에서 글자 그대로 복사**할 것 —
    기준 5 의 결함은 "패턴이 선언한 이름(`{botUserID}`)과 핸들러가 읽는 이름(`botID`)의 불일치" 이므로,
    테스트가 편의상 `{botID}` 로 등록하면 버그가 가려져 RED 가 나오지 않는다.
    셋업(격리 스키마 `newOperationsTestDB`, `store.Migrate`, `&handlers{...}` 직접 조립)은
    `custom_profile_values_errors_postgres_test.go:34-41`, `channel_membership_errors_postgres_test.go:30` 를 그대로 따를 것.
    `uploadTeamImage` 는 `h.callerCanAdminTeam`, 두 봇 경로는 `h.callerIsSystemAdmin` 을 통과해야 하므로
    핸들러 구조체에 필요한 필드(rbac/teams 등)를 무엇으로 채워야 하는지 구현자가 먼저 확인할 것 — **미확인**.

- 검증 명령 (server/ 에서):
  - `go build ./...`
  - `go vet ./...`
  - `MOYRO_TEST_POSTGRES_DSN=<dsn> go test -race -p 1 -count=1 ./internal/httpapi/`
    — DSN 이 없으면 DB 테스트가 **통째로 skip** 되므로 `ok` 만 보고 통과라고 하지 말 것.
      로컬 컨테이너 포트는 55432 가 점유된 적이 있어 **55433** 을 쓴다.
  - `MOYRO_TEST_POSTGRES_DSN=<dsn> go test -race -p 1 ./...` (전체, 수 분 — `-p 1` 유지)
  - `gofmt -l internal/httpapi/compat_wave_handlers_final.go internal/httpapi/request_body.go <신규 테스트>`
  - 루트에서 `bash scripts/check-source-sizes.sh`
    — `compat_wave_handlers_final.go` 는 2026-09-26 기준 72407/77000 이었다. 이번 변경은 수십 바이트라 여유가
      있지만 반드시 실행해 확인할 것.
  - 웹 변경 없음 — `webapp` 은 건드리지 말 것(tsconfig `types` 부재로 인한 TS2345 는 이 회차와 무관하다).

- 위험과 피할 것:
  - **"효과 없는 정리" 로 몰릴 위험이 이 과제의 주된 위험이다.** 세 핸들러는 저장하지 않는 스텁이므로 상태 코드만
    바꾸면 비평에서 기각될 수 있다. **413 과 "감사 기록 0건"(그리고 `uploadChunk` 의 거짓 `file_offset` 제거)을
    함께 증명해야** 실제 결함 수정이 된다. 기준 2 를 빼지 말 것.
  - 기존 오류 id 재사용 금지: `api.user.image.too_large` 는 `handlers.go:891` 이 프로필 이미지 **400** 에 쓰는 id 다.
    같은 id 로 다른 상태 코드를 내면 클라이언트 분기가 흔들린다. 새 id 를 쓰되 `/api/v4` 응답 **모양**(`{"id":...,"message":...}`)
    은 `writeError` 로 유지.
  - 성공 경로의 상한값(10MB/256KB/50MB)을 조정하지 말 것 — 상한 변경은 별개의 정책 결정이다.
  - `uploadTeamImage` 에 `denyGuestMutation` 이 없고 형제 `deleteTeamImage`(118)에는 있다 — 실제 비대칭이지만
    **이번 과제 범위 밖**이다(게스트 정책 결정이 선행). 손대지 말고 다음 회차 아이디어로 남길 것.
  - 보호 경로(auth/session/oidcauth, `store/migrations`, `.github/workflows`) 무변경. 마이그레이션 불필요.
  - `internal/httpapi` 의 다른 `MaxBytesReader` 호출자들(`native_tracking.go:156`, `handlers.go:2919/3141/3290`,
    `admin_compat_handlers.go:357`)까지 넓히지 말 것 — 파일 수가 늘고 계약이 다른 경로가 섞인다. 이번엔 3개만.
  - 감사 payload 에 본문 내용을 넣지 말 것(식별자만). 기존 `{"bytes": n}` 형태 유지.

- 차선 후보: **`patchPost` 의 `file_ids` 되읽기 실패 시 200 + 본문 `null` + `post_edited` 이벤트 누락**
  (`compat_wave_handlers_early.go:524` 부근 `updated, _ = h.posts.Get(...)`, 가치 3 / 위험 1 / S).
  고치는 방향은 2026-09-21 사이드바 재정렬 선례(되읽기 오류 → 기존 id 의 500, 성공 이벤트 차단) 그대로.
  **다만 막힌 곳은 주입 방법이다**: Update 는 성공하고 그 뒤 `Get` 만 실패시키는 결정론적 방법을 아직 못 찾았다
  (`DROP TABLE posts` 는 첫 `posts.Get` 부터 죽고, 2026-09-26 의 `{}` no-op 트릭은 posts 에 대응물이 없다).
  구현자가 이 주입을 30분 안에 못 풀면 차선도 접고 1순위로 돌아올 것.

- 이번 정찰이 실제로 확인한 것 / 미확인:
  - 확인: `compat_wave_handlers_final.go:135-148/605-618/915-927` 세 핸들러의 `io.Copy` 오류 무시 + 감사 호출 + 200
    (파일을 열어 읽음). `request_body.go:37-49` 의 `*http.MaxBytesError` → 413 선례. `handlers.go:3315` 의 413.
    `handlers.go:891` 이 `api.user.image.too_large` 를 400 에 쓰고 있음. `router.go:957-959` 의 `{botUserID}` 와
    핸들러의 `chi.URLParam(r,"botID")` 불일치(양쪽 모두 grep+파일로 확인). `internal/httpapi` 에 이 세 핸들러를
    이름으로 언급하는 테스트가 0건. 작업 트리 clean, base `main@c098d04`.
    `go test -count=1 -run TestUploadStubNothing ./internal/httpapi/` = ok (패키지·테스트 패키지 컴파일 확인).
  - **미확인**: 실제 DB 회귀. 이 정찰 세션에는 `MOYRO_TEST_POSTGRES_DSN` 이 없어 DB 테스트가 전부 skip 된다 —
    413/감사-0건을 실제로 재현해 보지 못했다. `go vet ./...`·전체 `go test`·`check-source-sizes.sh` 도 이번엔 미실행.
    `&handlers{}` 에 어떤 필드를 채우면 `callerCanAdminTeam`/`callerIsSystemAdmin` 이 통과하는지도 미확인.
