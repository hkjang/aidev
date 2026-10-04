# 과제서 — 2026-10-04-190204-moyro-improve (moyro)

- 과제: `bulkDeleteUsers` 가 상한 초과·깨진 본문을 삼키고 200 `{"status":"OK","count":0}` 으로 "삭제 성공" 을 보고하는 것 차단 (가치 3 / 위험 2 / 작업량 S)

- 왜: `compat_wave_handlers_final.go:1114` 이 `_ = decodeCappedBody(w, r, &body)` 로 디코드 오류를 버린다. `decodeCappedBody`(`request_body.go:57`)는 `json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(dst)` 뿐이고 **응답을 쓰지 않는다** — 1 MiB 를 넘는 `user_ids` 배열이나 깨진 JSON 이 오면 Decode 가 실패하고 `body.UserIDs` 는 nil 로 남아, 핸들러가 `writeJSON(w, 200, {"status":"OK","count":0})` 을 돌려준다. 관리자 도구는 200 을 받고 "일괄 삭제가 처리됐다" 로 읽지만 약속된 id 별 감사 행(`audit.ActionUserBulkDelete`)은 한 줄도 남지 않는다. 같은 파일의 `request_body.go:26-30` 이 쓰기 배치에 대해 **스스로 세운 계약**("silently dropping half of a *write* would be worse than refusing it, so the write batches reject instead")을 이 핸들러가 깨고 있다 — 상한 초과는 절반이 아니라 전부를 버리고도 200 이다. 고치면 거부된 쓰기 배치가 413/400 으로 끊겨 발신자가 재시도할 수 있고, 거짓 성공 응답이 사라진다.

- 수용 기준:
  1) `POST /api/v4/users/bulk_delete` 와 `DELETE /api/v4/users` 에 `collectionBodyMaxBytes`(1 MiB, `request_body.go:24`)를 넘는 `{"user_ids":[...]}` 본문을 보내면 **413** 과 `api.user.bulk_delete.invalid_body` 가 오고, 200 `{"status":"OK","count":0}` 이 아니다.
  2) 깨진 JSON 본문(예: `{"user_ids":[`)은 **400** 과 같은 id 가 온다.
  3) **본문 없는 요청(Content-Length 0)은 오늘과 같이 200 `{"status":"OK","count":0}` 을 유지한다** — 아래 "위험" 의 io.EOF 항목을 반드시 읽을 것.
  4) 거부된 요청은 `audit` 행을 남기지 않는다. 양성 대조를 먼저 관찰할 것: 정상 2개 id 요청이 `user.bulk_delete` 행 2개를 남기는 것을 `waitForAuditRow` 로 확인한 **뒤에**, 거부 요청에 대해 3.2초 동안 행이 없음을 관찰한다(`audit.LogAsync` 는 goroutine + 3초 timeout).
  5) 바뀌지 않는 것: 정상 본문 → 200 `{"status":"OK","count":N}` **바이트 동일**; id 201개 → 400 `api.user.bulk_delete.too_many`(`tooManyBatchItems`, `maxBulkItems=200`); 시스템관리자 아님 → 403 `api.context.permissions.app_error` "system_admin required".
  6) 테스트가 증명해야 하는 것: 수정 **전** 에 기준 1·2·4 의 서브테스트만 RED 이고 3·5 는 수정 전후 모두 PASS. 수정 후 전부 PASS. 그리고 바꾼 분기 한 곳을 되돌리면 **그 서브테스트만** 다시 깨진다(역방향 확인).

- 건드릴 파일 (프로덕션 1개 + 신규 테스트 1개):
  - `server/internal/httpapi/compat_wave_handlers_final.go:1106 bulkDeleteUsers` — 1114 의 `_ = decodeCappedBody(w, r, &body)` 를 오류를 보는 형태로 바꾼다. **`decodeCollectionBody` 로 단순 치환하지 말 것**(io.EOF 문제, 아래 참조). 권장 형태:
    ```go
    if err := decodeCappedBody(w, r, &body); err != nil && !errors.Is(err, io.EOF) {
        var tooLarge *http.MaxBytesError
        if errors.As(err, &tooLarge) {
            writeError(w, http.StatusRequestEntityTooLarge, "api.user.bulk_delete.invalid_body", ...)
        } else {
            writeError(w, http.StatusBadRequest, "api.user.bulk_delete.invalid_body", err.Error())
        }
        return
    }
    ```
    `tooManyBatchItems` 호출(1116)·감사 루프(1119)·성공 응답(1124)은 그대로 둔다. import 상태 미확인 — 이 파일에 `errors` 는 있고(`1241` 의 `errors.Is(err, customprofile.ErrFieldNotFound)`) `io` 는 **미확인**이므로 필요하면 추가할 것. 분기 로직을 `request_body.go` 에 작은 헬퍼로 빼는 것도 허용(그 쪽이 깔끔하면), 단 기존 `decodeCappedBody`/`decodeCollectionBody` 의 동작은 한 글자도 바꾸지 말 것 — 다른 호출자 16곳이 걸려 있다.
  - `server/internal/httpapi/bulk_delete_body_cap_postgres_test.go` (신규) — 아래 "검증" 의 배선으로 6개 서브테스트.
  - 새 오류 id 는 `api.user.bulk_delete.invalid_body` 하나뿐. 이 저장소의 관례와 일치한다(`api.user.ids.invalid_body`, `api.post.create.invalid_body` 등 21곳이 `.invalid_body` 를 쓴다 — `compat_wave_handlers_early.go:192`, `handlers.go:1608` 등). 새 id 를 피하고 싶으면 `api.user.bulk_delete.too_many` 재사용도 가능하지만 깨진 본문에 "too_many" 는 의미가 어긋나므로 권장하지 않는다.

- 검증 명령 (모두 `/home/hkjang/.cache/auto-improve-wt/moyro` 기준):
  - 라우트: `router.go:998-1001` — `r.Group` 안에서 `r.Use(h.requireRole("system_admin"))` 뒤에 `r.Post("/users/bulk_delete", h.bulkDeleteUsers)` 와 `r.Delete("/users", h.bulkDeleteUsers)` 두 줄. 테스트는 이 패턴을 글자 그대로 복사하거나(기존 `team_admin_errors_postgres_test.go:88-99` 방식) 프로덕션 `NewRouter` 를 띄우는 `invite_email_route_postgres_test.go` 방식을 쓸 것. **손으로 만든 대역(가짜 서비스/주입 객체) 금지** — 실제 `auth`/`audit` 서비스와 실제 PostgreSQL 격리 스키마(`newOperationsTestDB` + `store.Migrate`)를 쓸 것.
  - DB: `docker start moyro-pg-improve` (호스트 포트 55433). `MOYRO_TEST_POSTGRES_DSN` 을 주지 않으면 DB 테스트가 **skip** 되므로 `ok` 를 통과로 오인하지 말 것 — `-v` 로 skip 없음을 확인할 것.
  - 지정 테스트: `cd server && go test -count=1 -v -run TestBulkDelete ./internal/httpapi/`
  - 패키지/전체: `cd server && go test -race -p 1 -count=1 ./internal/httpapi/` → `go test -race -p 1 -count=1 ./...` (수 분)
  - `cd server && go build ./... && go vet ./...`
  - `gofmt -l` 은 변경·신규 파일만 볼 것 — `gofmt -l server/internal` 은 기존 9파일을 출력한다(독립 포맷 PR 금지).
  - `bash scripts/check-source-sizes.sh` — `compat_wave_handlers_final.go` 는 한도에 가깝다(이전 회차 기록 72662/77000). 몇 줄 추가는 괜찮지만 반드시 돌려 볼 것.
  - 웹 변경 없음 → webapp 명령 불필요. 단 `webapp` typecheck 가 환경 탓으로 깨지면 자기 변경 탓으로 여기지 말 것(교훈의 `@types/node` 항목).

- 위험과 피할 것:
  - **io.EOF (가장 중요)**: `json.Decoder.Decode` 는 **빈 본문에 `io.EOF` 를 돌려준다**. 지금은 `_ =` 가 그것을 삼켜 본문 없는 `DELETE /api/v4/users` 가 200 `count:0` 을 받는다. `decodeCollectionBody` 로 그냥 치환하면 그 요청이 **400 으로 바뀐다** — 계약 변경이다. 본문 없는 호출자가 실제로 있는지는 **미확인**이므로, 안전한 쪽(수용 기준 3: 빈 본문은 200 유지)을 택하고 그 서브테스트로 고정할 것.
  - **범위를 넓히지 말 것**: `_ = decodeCappedBody` 호출자는 이 패키지에 17곳 있다(`compat_wave_handlers_final.go` 14곳, `enterprise_compat_handlers.go:20`, `admin_compat_handlers.go:211`). 나머지의 "깨진 본문에도 200" 은 `request_body.go:52-56` 이 **의도적**이라고 명시한다. `bulkDeleteUsers` 만 쓰기 배치여서 `request_body.go:26-30` 의 계약과 모순된다 — 이번 회차는 이 한 곳만. 차선 후보(customprofile 2곳)도 **같은 PR 에 넣지 말 것**.
  - 건드리지 말 것: `decodeCappedBody`/`decodeCollectionBody`/`drainCappedBody`/`tooManyBatchItems` 의 기존 동작, `collectionBodyMaxBytes`/`maxBulkItems` 값, `callerIsSystemAdmin`, `requireRole` 미들웨어, `router.go` 등록, 마이그레이션, `.github/workflows`.
  - 과거 교훈이 걸린 자리: 상한 초과를 삼키고 200 + 거짓 감사를 남기는 결함은 2026-09-27 회차가 업로드 스텁 3개에서 고쳐 **채택·release-ready** 로 끝났다(같은 계열의 성공 선례이지, 반려된 접근이 아니다). 그 회차의 교훈대로 **거부는 감사 호출보다 앞**에서 일어나야 한다.
  - 감사로 넘기는 값은 식별자(uid)만 — 지금 코드가 이미 그렇다. 본문 원문을 감사 payload 에 넣지 말 것.
  - 반려 이력 주의: 2026-09-29 의 "팀 이미지 업로드에 게스트 차단" 회차는 성공으로 적혀 있으나 **main 에 들어오지 않았다**(이번 세션에서 `uploadTeamImage`(final.go:134)에 `denyGuestMutation` 이 없음을 확인). 따라서 "팀 스텁에 게스트 가드 추가" 계열(`inviteGuestsByEmail` 포함)은 이번에 고르지 않았다 — 구현자도 이 과제를 그 방향으로 바꾸지 말 것.

- 차선 후보: `createCustomProfileField`(final.go:1212) / `patchCustomProfileField`(final.go:1238) 의 `_ = decodeCappedBody` — 상한 초과·깨진 본문이 빈 struct 로 서비스에 내려가 `CreateField("","",nil)` 을 호출한다. 서비스가 400 을 돌려줄 가능성이 높아(미확인) 사용자에게 보이는 증상은 덜 심각하지만, 413 이어야 할 것을 400 으로 보고하는 것은 같은 결함이다. 1순위가 성립하지 않으면(예: 빈 본문 계약을 지킬 방법이 없다고 판단되면) 이쪽을 S 로 한 개만 고를 것.
