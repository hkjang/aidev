- 과제: (수정 과제 — 직전 회차 `error: interrupted before PR` 재시도) `admin.go` 의 `storage_error` 500 출구 25곳을 `writeStorageError` 로 이관한다 (가치 3 / 위험 1 / 작업량 M)
- 왜: 직전 회차가 같은 과제를 받고 **PR 을 만들기 전에 중단**돼 아무 산출물도 남기지 못했다(이 워크트리 `git status` clean · 브랜치 `auto/2026-10-17-1808` 이 `origin/main`=`c9bfca2` 그대로 · `git branch -r` 에 새 브랜치 없음 — 셋 다 실행 확인). 과제 자체는 지금 코드에서 그대로 유효하다: `admin.go` 의 500 출구 **25곳**(실행 `grep -c` 확인)이 전부 `err` 를 버려, 운영자가 "역할 정책을 저장할 수 없습니다" 500 을 보고도 `adminPutRoles` 의 **다섯 쿼리 중 어느 것**이 거절됐는지 알 수 없다.

- 수용 기준:
  1) `grep -c 'writeError(w, http.StatusInternalServerError, "storage_error"' backend/internal/httpapi/admin.go` 가 **0**.
  2) **응답 불변**: `git show origin/main:backend/internal/httpapi/admin.go` 와 작업 트리 파일의 한글 문자열 리터럴 multiset 비교에서 **origin/main 에만 있는 리터럴 0개**이고, 늘어난 리터럴도 **0개**(admin.go 는 합친 조건 분할이 필요 없으므로 posts/social 때와 달리 +1 이 생기면 안 된다 — 아래 참조). status·`code`·message 한 글자도 바뀌지 않는다.
  3) `handler` 인자를 **수기로 적지 말 것**. `^func ` 경계에서 감싼 함수 이름을 기계적으로 유도해 치환하고, 치환 후 같은 유도를 다시 돌려 **전 호출 mismatch 0** 을 출력으로 보일 것(2026-10-05·10-06 회차가 쓴 방법, 그때 수기 매핑과 100% 일치했다).
  4) **실제 핸들러 경로 red→green**: `adminPutRoles` 에 sentinel BEFORE 트리거 integration 테스트를 새로 넣어, `admin.go` 를 고치기 **전에** 로그 4단언(`"error_code":"storage_error"`·`"handler":"adminPutRoles"`·`"cause_type":"*pgconn.PgError"`·`"pg_code":"P0001"`)이 **없어서** FAIL 하는 것을 관측하고, 이관 후 PASS 를 관측한다. **같은 red 출력에서 응답 단언(500 + `code=storage_error` + 기존 message)은 통과**해야 한다 — red 의 원인이 응답이 아니라 없는 관측성임이 한 출력으로 분리돼야 한다.

- 건드릴 파일 (프로덕션 **1개** + 테스트 1개):
  - `backend/internal/httpapi/admin.go` — 25곳을 `writeStorageError(w, r, "<감싼 함수명>", err, "<기존 message 그대로>")` 로. 함수 매핑(실행 확인한 `^func ` 경계 기준, 현재 줄번호):
    `:38` `adminStats` · `:55`,`:63` `adminListUsers` · `:108` `adminCreateUser` · `:174` `adminUpdateUser` · `:240`,`:250` `adminListPosts` · `:273` `adminUpdatePost` · `:292` `adminDeletePost` · `:333`,`:342` `adminListReports` · `:419`,`:429` `adminListRoles` · `:483`,`:489`,`:494`,`:499`,`:506` `adminPutRoles` · `:523`,`:534` `adminListAudit` · `:545` `adminListKeys` · `:564`,`:585`,`:604` `adminUpdateKey` · `:618` `adminRevokeKey`.
    - **유일한 실질 함정 — `rows.Scan` 오류를 인라인에서 버리는 4곳**: `:249`·`:341`·`:428`·`:533` 이 `if rows.Scan(...) != nil {` 형태라, 그 `if` 본문(`:250`·`:342`·`:429`·`:534`)에서 변수 `err` 는 **Query 의 err = nil** 이다. 그대로 `err` 를 넘기면 `cause_type:"<nil>"`·`pg_code:""` 가 찍혀 이 회차가 없애려는 "원인 없는 500" 을 새로 만든다. `if scanErr := rows.Scan(...); scanErr != nil {` 로 캡처해 **`scanErr`** 를 넘길 것.
    - `tx.Commit` 은 admin.go 에서 **이미 `if err := tx.Commit(ctx); err != nil` 로 분리돼 있다**(`:320`·`:408`·`:505`, grep 확인). posts.go/social.go 때 필요했던 `Exec 실패 || Commit 실패` 합친 조건 **분할은 admin.go 에 필요 없다** — 제어 흐름을 재구성하지 말 것.
    - `adminPutRoles` 의 다섯 출구는 전부 자기 `err` 가 스코프에 있다(실행 확인: Begin 의 `err`, 세 Exec 의 `if _, err := tx.Exec(...)`, Commit 의 `err`). 단순 치환으로 끝난다.
    - `resolveReport`(`:375~`)·`adminResetPassword`(`:191~`)·`softDeletePost`(`:303~`)은 **이미 이관 완료**다. 위 25곳에 들어 있지 않으니 손대지 말 것.
  - 새 테스트 `backend/internal/httpapi/admin_roles_put_postgres_integration_test.go` — 배선 원본은 **`admin_report_resolve_postgres_integration_test.go:118~132`**(실행 확인: `New(repository, secrets, "v0.1.38-test")` **앞**에서 `logs := &lockedBuffer{}` + `previousLogger := slog.Default()` + `slog.SetDefault(slog.New(slog.NewJSONHandler(logs, nil)))` + `t.Cleanup` 복원 — `Handler()` 가 `slog.Default()` 를 캡처하므로 **순서가 전부**다). `lockedBuffer` 는 `storage_error_test.go:22` 의 패키지 타입이라 새로 정의하지 말 것. admin 계열 integration 테스트는 현재 이 한 파일뿐이므로 세션 row + cookie + CSRF 셋업은 그 파일에서 복사한다. 권한은 `roles:manage` 가 필요하다(`migrations/001_initial.sql` 의 admin 역할 시드에 포함).
    sentinel 트리거는 **`role_permissions` 의 BEFORE INSERT** 로 걸고 `NEW.permission` 이 **sentinel 문자열 한 값**일 때만 `RAISE EXCEPTION`(→ SQLSTATE `P0001` → `*pgconn.PgError`). 그러면 `PUT /api/v1/admin/roles` 의 **가장 안쪽 INSERT(`:499`)** 가 거절돼 다섯 출구 중 가장 깊은 곳을 지난다. CI source 는 PostgreSQL **한 DB 를 네 패키지가 공유**하므로 sentinel 한 값 외에는 절대 거절하지 말고 `t.Cleanup` 에서 트리거를 DROP 할 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd backend && go build ./... && go vet ./... && gofmt -l .` (전부 exit 0 / 무출력)
  - DB: throwaway `docker run --rm -d -e POSTGRES_PASSWORD=<pw> -p <port>:5432 postgres:16-alpine` → `MOINA_TEST_POSTGRES_DSN=<dsn> go test -race -count=1 -v ./... 2>&1 | tee /tmp/t.log`; `grep -c -- '--- SKIP' /tmp/t.log` → **0** (DSN 이 실제로 걸렸다는 증거), `grep -c -- '--- FAIL'` → **0**, 최상위 `TestPostgreSQL*` PASS **44건 이상**(v0.1.43 기준 44 + 이번 신규 1 = 45 기대).
  - `make check` exit 0 (OpenAPI route **120개** 유지), `make lint`, `make fmt` 변경 없음.
  - **프런트·`make image`·e2e 는 돌리지 말 것** — 응답이 불변이고 프런트 파일을 하나도 건드리지 않는다. 과거 회차 TIMEOUT 2건이 검증 과잉에서 나왔고, 직전 회차는 PR 전에 끊겼다.

- 실행 순서 (각 단계에 증명을 붙이고, 끝난 즉시 커밋해 중단에 대비할 것):
  1. 25곳 치환 + `scanErr` 4곳 → 증명: `go build ./...` exit 0 · 기준 1·2·3 출력. **여기서 바로 커밋**(중단되더라도 PR 거리가 남는다).
  2. `admin.go` 를 `git stash` 하지 말고 `git checkout origin/main -- backend/internal/httpapi/admin.go` 로 **잠시 되돌려** 새 테스트의 red 를 관측 → 증명: 로그 4단언만 FAIL, 응답 단언 PASS. 그 뒤 `git checkout HEAD -- backend/internal/httpapi/admin.go` 로 1단계 결과 복구.
  3. 전체 `go test -race -count=1 -v ./...` + `make check` → 증명: 위 검증 명령의 수치.
  4. 커밋 amend/추가 → **푸시 → PR**. (직전 회차가 실패한 지점이 바로 여기다.)
  - **범위 밖**: 다른 파일의 `storage_error` 이관(settings 9 · workflow 9 · auth 7 · preferences 6 · analytics 4 · smtp 3 · outbox 2 · oidc 1), `storage_error.go` 헬퍼 수정, 새 로그 필드, OpenAPI·프런트·e2e.

- 위험과 피할 것:
  - **이번 회차 1순위 위험은 코드가 아니라 중단이다.** 위 실행 순서의 1단계가 끝나면 무조건 커밋하고, 검증을 반복하지 말 것(한 번 통과하면 그것이 증거다).
  - `backend/internal/httpapi/social.go` 와 `frontend/src/auth/AuthContext.test.tsx` 를 **절대 건드리지 말 것** — `origin/auto/2026-10-08-0138` 이 두 회차 분량(`7b0f70d`+`238eff1`)을 담고 사람의 머지를 기다린다. 같은 복원을 또 올리지도 말 것(이미 두 번 올렸고 운영자가 반복 재제출을 금지했다).
  - 보호 경로 금지: `.github/workflows/*` · `Dockerfile` · `Makefile` · `backend/internal/store/migrations/*` · `VERSION`.
  - `cursor_error` 500 · `RowsAffected()==0` 404 · `store.IsNotFound`/`IsConflict` 409 · `super_admin_required` 409 분기는 그대로 둘 것.
  - **`stage` 같은 새 로그 필드를 발명하지 말 것.** `writeStorageError` 의 4필드만 쓴다. `adminPutRoles` 의 다섯 출구가 `handler` 만으로는 구분되지 않는 한계는 `followUser` 때와 같이 **그대로 남긴다**(지난 두 회차의 선례).
  - 감사 로그·API 키 경로를 지나므로 pg 메시지 전문·`Detail`·`Hint` 는 절대 찍지 않는다 — 헬퍼가 이미 그렇게 설계돼 있으니 헬퍼를 고치지 말 것. 테스트 단언도 sentinel 문자열이 로그에 **없음**을 확인하는 쪽으로 쓸 것.
  - 미확인: sentinel 트리거를 `role_permissions` 에 거는 것이 `adminPutRoles` 의 `:499` INSERT 를 실제로 거절하는지는 **실행으로 확인하지 않았다**(코드상 그 INSERT 가 `role_permissions` 로 들어가는 것만 읽었다). 거절이 다른 출구로 떨어지면 `handler` 는 같으므로 테스트의 기대 줄만 조정하면 된다.

- 추정 근거 (basis of estimate): 같은 작업을 두 번 했다 — posts.go 41곳(M, 1세션), social.go 29곳(M, 1세션). admin.go 는 25곳으로 더 작고 합친 조건 분할이 없어 **그 둘보다 쉽다**. 추가 변수는 새 integration 테스트 파일 1개(세션/CSRF 셋업 복사 ~120줄)뿐이다. 범위: 이관만이면 S, 새 테스트 포함이면 M. 컨틴전시는 **실행 순서 1단계 조기 커밋**으로 잡는다 — 2단계 이후가 끊겨도 PR 가능한 산출물이 남는다.

- 차선 후보: `settings.go`(9) + `preferences.go`(6) 15곳 이관 — 같은 기법이지만 두 파일 모두 저장 실패 integration 테스트가 **0개**라 sentinel 트리거 설계가 선행이고 위험이 한 단계 높다. 1순위가 성립하지 않을 때만 고를 것.

- 다른 후보를 제친 이유 (solution exploration): (가) 지시대로 "워크플로를 고치는" 길 — `.github/workflows/*` 를 읽고 고치는 것은 **원인이 워크플로가 아니라 중단**이므로 성립하지 않는다(워크플로 실패 로그 `ci-failure-*.txt` 가 이 run 디렉터리에 없다 — 실행 확인). 보호 경로를 이유 없이 건드리는 것은 운영자 지시 위반이다. (나) 미머지 브랜치 재제출 — 운영자가 반복 재제출을 명시적으로 금지했다. (다) 프런트 `findBy` 경합 전수 점검(가치 4) — 가치는 더 높지만 `make image` 까지 봐야 해 중단 재발 위험이 가장 큰 과제라 이번 회차에는 부적합. 남은 것 중 **근거·수단·선례가 모두 갖춰진 유일한 M** 이 admin.go 다.
