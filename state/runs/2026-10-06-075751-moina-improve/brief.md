# 과제서 (2026-10-06)

- 과제: `social.go`의 `storage_error` 500 출구 29곳을 `writeStorageError`로 이관한다 (가치 3 / 위험 1 / 작업량 M)

- 왜: v0.1.42가 `writeStorageError`(`storage_error.go:22`)를 만들고 v0.1.43이 `posts.go` 41곳을 이관해 관례가 확립됐는데, `social.go`에는 `err`를 그대로 버리는 `writeError(w, http.StatusInternalServerError, "storage_error", …)` 출구가 29곳 남아 있다(`followTopic`만 2026-10-04에 이관됨 — `:286`). 중복 message가 특히 심한 파일이어서("Link를 저장할 수 없습니다" 4곳 = `followUser`의 Begin·Exec·알림 enqueue·Commit, "Moim에서 나갈 수 없습니다" 2곳, "Moim을 만들 수 없습니다" 2곳, "Moim 목록을 불러올 수 없습니다" 2곳, "Topic을 불러올 수 없습니다" 2곳, "알림을 불러올 수 없습니다" 2곳) 운영자는 500을 받고도 `handler`·`cause_type`·`pg_code`가 없어 어느 쿼리가 거절됐는지 알 수 없다.

- 수용 기준:
  1) `grep -c 'StatusInternalServerError, "storage_error"' backend/internal/httpapi/social.go` 가 **0**이 되고, 29곳이 전부 `writeStorageError(w, r, "<감싼 함수 이름>", err, "<원래 message 그대로>")` 가 된다.
  2) **응답 불변**: status 500·`code":"storage_error"`·message 문자열이 한 글자도 바뀌지 않는다. `git show HEAD:backend/internal/httpapi/social.go` 와 현재 파일의 한글 문자열 리터럴 multiset을 비교해 **HEAD 에만 있는 리터럴이 0개**임을 출력으로 보일 것(v0.1.43 이 쓴 절차 재사용). 아래 3-나 분할 때문에 늘어나는 리터럴은 허용된다.
  3) 테스트가 증명해야 하는 것: 실제 핸들러 경로에서 로그가 **없다가 생긴다**. `moim_join_postgres_integration_test.go` 의 `"failed INSERT is 500 storage_error"` subtest(`:193`)에 로그 단언만 먼저 넣고 `social.go` 수정 전에 돌려 red 를 관측한 뒤(= `"error_code":"storage_error"`·`"handler":"joinMoim"`·`"cause_type":"*pgconn.PgError"`·`"pg_code":"P0001"` 누락), 이관 후 green. **같은 red 출력에서 응답 단언(500 + `storage_error` + message)은 통과해야 한다** — red 의 원인이 응답이 아니라 없는 관측성임이 그 출력으로 분리된다.
  4) `RowsAffected()==0` 404 분기(`:289`·`:579`·`:598`·`:1068`), `store.IsConflict` 409(`:526`), `owner_cannot_leave` 409(`:614`), `store.ErrNotFound` 404(`:607`), `not_found`·`invalid_*` 분기는 전부 그대로다. OpenAPI route **120개** 유지.

- 건드릴 파일 (프로덕션 **1개**):
  - `backend/internal/httpapi/social.go` — 29곳 치환. 감싼 함수에서 `handler` 이름을 **기계적으로 유도**할 것(v0.1.43 이 쓴 `grep -n '^func '` 경계 방식). 수기 매핑 금지. 확인한 매핑: `writeProfile`(:60) · `followUser`(:91,:97,:103,:108) · `unfollowUser`(:118) · `blockUser`(:133,:138,:142) · `unblockUser`(:152) · `muteUser`(:167) · `unmuteUser`(:176) · `listTopics`(:239,:247) · `unfollowTopic`(:299) · `search`(:365) · `listNotifications`(:398,:406) · `readNotifications`(:486) · `createMoim`(:516,:528) · `listMoims`(:544,:552) · `joinMoim`(:576) · `leaveMoim`(:595,:611) · `uploadMedia`(:711) · `deleteMedia`(:1065) · `createReport`(:1108).
    - **(3-가) `:60` 은 `writeProfile` 로 적을 것.** `writeProfile`(:42)은 라우트 핸들러가 아니라 두 경로가 공유하는 렌더 헬퍼다(`social.go:39` getProfile, `auth.go:483` `getProfileByUser`). v0.1.43 이 `writePostError` 에 했던 것처럼 `handler` 파라미터를 뚫으면 `auth.go` + integration 테스트(`social_search_profile_postgres_integration_test.go:172` 가 `writeProfile` 을 직접 호출)까지 번지는데, 그 출구는 함수 안에 **하나뿐**이고 카운터 쿼리를 유일하게 지목하므로 이름만으로 충분하다. 파일 수를 1개로 묶는 쪽을 택한다.
    - **(3-나) 합친 `Exec 실패 || Commit 실패` 두 곳은 그대로 이관할 수 없다 — 분할해야 한다.** `blockUser:141` `if _, err := tx.Exec(…); err != nil || tx.Commit(r.Context()) != nil {` 와 `createMoim:524` `if err != nil || tx.Commit(r.Context()) != nil {`. 그 `if` 본문에서 `err` 는 Exec 의 오류라 **Commit 이 거절된 경우 `nil`** 이고, 그대로 넘기면 `cause_type:"<nil>"`·`pg_code:""` 가 찍혀(`storage_error.go:48` 가 `nil` 을 `"<nil>"` 로 반환) 이 과제가 없애려는 "원인 없는 500" 을 새로 만든다. v0.1.43 이 `deletePost`·`deleteRemoin` 에 한 것과 같은 분할: **단락 평가가 주던 순서(Exec 실패면 Commit 미호출)와 응답·message·분기를 그대로 두고** 두 `if` 로 나눈다. `createMoim` 은 `store.IsConflict(err)` 409 가 **Exec 오류에만** 걸려 있는 현재 동작을 보존할 것(Commit 오류는 지금 `IsConflict(nil)==false` 로 500 이므로 분할 후에도 500).
  - `backend/internal/httpapi/moim_join_postgres_integration_test.go` — `"failed INSERT is 500 storage_error"`(:193)에 로그 단언 추가. 이 파일은 `CREATE FUNCTION … RAISE EXCEPTION`(:97~103) + `CREATE TRIGGER … BEFORE INSERT ON moim_members`(:106)로 실제 pg 실패를 만들고 `server.Handler()`(:111)를 지나므로, **`Handler()` 가 `slog.Default()` 를 캡처한다**는 성질로 로그를 되읽을 수 있다. 캡처 배선은 `posts_update_postgres_integration_test.go:108~112` 를 그대로 복사하면 된다 — `logs := &lockedBuffer{}` / `previousLogger := slog.Default()` / `slog.SetDefault(slog.New(slog.NewJSONHandler(logs, nil)))` / `t.Cleanup(…)`, 그리고 `server := New(…)` **앞**에 둘 것. `lockedBuffer` 는 `storage_error_test.go:22` 에 이미 있으니 새로 만들지 말 것. `RAISE EXCEPTION` 에 ERRCODE 가 없으므로 SQLSTATE 는 **P0001**, 타입은 `*pgconn.PgError` 다(v0.1.43 에서 같은 트리거 기법으로 리터럴 단언이 통과해 확인됨).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  docker run -d --rm --name moina-recon-pg -e POSTGRES_PASSWORD=moina -p 55432:5432 postgres:16-alpine
  cd backend
  export MOINA_TEST_POSTGRES_DSN='postgres://postgres:moina@127.0.0.1:55432/postgres?sslmode=disable'
  go build ./... && go vet ./...
  go test -race -count=1 -v ./... 2>&1 | tail -60
  gofmt -l .
  make check          # 저장소 루트에서. OpenAPI route 120개
  make fmt            # 변경 없음이어야 함
  ```
  - **DSN 이 실제로 걸렸다는 증거**: `-v` 출력에 `--- SKIP` **0줄**, `--- FAIL` 0줄, 최상위 `TestPostgreSQL*` PASS **44건** 유지.
  - 응답이 불변이므로 프런트(`npm test`)·e2e·시각 회귀는 돌리지 않는다. `api/openapi.yaml` 도 손대지 않는다(새 status code 없음).

- 위험과 피할 것:
  - **같이 건드리지 말 것**: `storage_error.go` 헬퍼 자체, `oidc_discovery_error.go`, `followTopic:286`(이미 이관됨), `writeError`/`writeErrorDetails`(`server.go:694`·`:700`), 미들웨어 체인(`server.go:141`), migrations, `.github/workflows/*`. `social.go` 안의 `not_found`·409·400 출구도 건드리지 않는다.
  - `joinMoim:581`·`uploadMedia`/`getMedia:1018` 의 `_ = Scan(&exists)` / `err != nil || !owned` 가 조회 오류를 404 로 감추는 것은 **이번 범위가 아니다**(보류 아이디어로 남아 있고, SELECT 실패는 sentinel BEFORE 트리거로 재현 불가라 전용 스키마 헬퍼가 선행). 이관만 하고 그 계약은 그대로 둘 것.
  - **이번 이관이 해결하지 못하는 것(솔직히 적음)**: `followUser` 의 네 출구는 `handler` 가 모두 `"followUser"` 라 로그 한 줄만으로 Begin/Exec/enqueue/Commit 을 구분하지 못한다. 실무에서는 `pg_code`·`cause_type` 이 다르게 나와 대체로 갈리지만 완전하지는 않다. 여기서 새 `stage` 필드를 발명하지 말 것 — 변경을 기계적으로 유지하는 것이 이 과제의 전제이고, 필드 추가는 별 과제다.
  - 과거 교훈: 로그에 pg 메시지 전문·`Detail`·`Hint`·사용자 입력 원문을 넣지 말 것(`oidc_discovery_error_test.go` 가 단언). 헬퍼가 이미 `Code` 만 뽑으니 **헬퍼 호출만** 하고 로그 인자를 직접 조립하지 말 것.
  - 로그인 5분/동일 IP 5회 제한이 있어 검증을 과도하게 반복하면 인증 단계에서 실패한다. 인과 증명 후 회귀 확인을 과잉 반복하지 말 것(회차 TIMEOUT 2건의 원인).

- 차선 후보: `settings.go`(9) + `preferences.go`(6) + `analytics.go`(4) = 19곳 이관. 프로덕션 파일 3개로 더 작지만 **세 파일 모두 저장 실패 integration 테스트가 없어** red→green 증명에 새 테스트 파일과 새 sentinel 트리거 설계가 필요하다(그래서 1순위보다 위험이 높다). 1순위가 성립하지 않을 때만 고를 것.
