# 과제서 (2026-10-05, run 2026-10-05-171742-moyro-improve)

- 과제: 커스텀 프로필 값 PATCH 두 핸들러가 거부해야 할 본문을 삼키고 200 + 이전 값 + `count:0` 감사행으로 "저장 성공" 을 보고하는 것 차단 (가치 3 / 위험 2 / 작업량 S)

- 왜: `compat_wave_handlers_final.go:1296`(`patchCustomProfileValuesGlobal`)과 `:1349`(`patchUserCustomProfileValues`)가 맵 모양 본문을 `_ = decodeCappedBody(w, r, &values)` 로 읽어 디코드 오류를 버린다. 1 MiB 를 넘는 본문이나 중간에 끊긴 JSON 이 오면 `values` 가 nil 로 남고, `customprofile.Service.PatchUserValues`(`server/internal/customprofile/service.go:199-202`)는 `if len(values) == 0 { return nil }` 로 **아무것도 쓰지 않고** nil 을 돌려준다. 그 뒤 핸들러는 `audit.ActionCustomValuesPatch` 를 `{"count":0}` 으로 기록하고, `GetUserValues` 로 **이전 값을 되읽어** 200 으로 돌려준다 — 즉 클라이언트는 그럴듯하게 채워진 프로필 맵과 함께 200 을 받고 "저장됐다" 로 읽지만 실제로는 한 줄도 쓰이지 않았다. 2026-10-04 회차가 `bulkDeleteUsers` 에서 고친 것과 같은 결함이지만, 되읽기 때문에 응답이 빈 `count:0` 이 아니라 **진짜 저장 응답과 구별이 안 되는** 더 나쁜 모양이고, `patchUserCustomProfileValues` 는 관리자가 남의 프로필을 백필하는 지원 도구 경로라 감사 원장에도 거짓 성공이 남는다.

- 수용 기준:
  1) `PATCH /api/v4/users/me/custom_profile_attributes` 와 `PATCH /api/v4/custom_profile_attributes/values` 에 `collectionBodyMaxBytes`(1 MiB, `request_body.go:24`) 를 넘는 본문을 보내면 **413**, 중간에 끊긴/문법 오류 JSON 을 보내면 **400** 이 오고, 둘 다 응답 본문이 프로필 값 맵이 아니라 오류 객체다.
  2) 거부된 요청은 **감사행을 남기지 않는다** — `audit.ActionCustomValuesPatch` 행이 0건임을 (정상 요청으로 양성 대조를 세운 뒤) 폴링으로 확인한다. 거부는 반드시 `h.audit.LogAsync` 호출 **앞**에서 일어나야 한다(2026-09-27 업로드 스텁 회차가 정한 순서).
  3) 보존: 본문 없는 PATCH 는 **오늘과 똑같이 200** 과 현재 값 맵을 돌려준다(`decodeOptionalCollectionBody` 의 `io.EOF` 관용). 본문이 리터럴 `null` 인 PATCH, 본문이 `{}` 인 PATCH 도 200 no-op 그대로. 정상 `{field_id: value}` PATCH 는 값이 실제로 쓰이고 응답 바이트가 수정 전과 동일하며 감사행 1건이 남는다.
  4) `null` 값으로 필드를 지우는 DELETE 계약(`service.go:210-215`, `len(raw)==0 || string(raw)=="null"` → `DELETE FROM custom_profile_values`)이 그대로 동작한다.
  5) 테스트가 증명할 것: 수정 전 RED(상한 초과·깨진 본문이 200 이고 감사행이 남는 것) → 수정 후 GREEN, 그리고 **역방향** — 변경한 두 줄 중 하나를 되돌리면 그 핸들러의 서브테스트만 다시 깨지는 것.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/httpapi/compat_wave_handlers_final.go:1296` — `patchCustomProfileValuesGlobal` 의 `_ = decodeCappedBody(w, r, &values)` 를 `if !decodeOptionalCollectionBody(w, r, "api.custom_profile.values.patch.invalid_body", &values) { return }` 로. 왜 `_ =` 가 아니어도 되는지(맵도 collection 이라는 `request_body.go:11-24` 의 자기 계약, 되읽기가 거짓 성공을 가리는 것)를 2~4줄 주석으로.
  - `server/internal/httpapi/compat_wave_handlers_final.go:1349` — `patchUserCustomProfileValues` 에 같은 치환. 같은 오류 id 를 재사용한다(새 id 는 **하나만**).
  - `server/internal/httpapi/custom_profile_values_body_cap_postgres_test.go` (신규) — 아래 "검증" 의 배선으로 서브테스트 작성.
  - **헬퍼는 새로 만들지 말 것.** `decodeOptionalCollectionBody`(`request_body.go:62-75`)가 2026-10-04 회차에 이미 들어와 있고 정확히 이 모양이다(`io.EOF` 통과 / `*http.MaxBytesError` 413 / 그 외 400). `request_body.go` 는 **한 글자도 바꾸지 말 것**.

- 검증 명령:
  - 실제 DB 필요: `export MOYRO_TEST_POSTGRES_DSN=...`(이전 회차들이 쓴 컨테이너 `moyro-pg-improve`, 호스트 포트 **55433**. `docker start moyro-pg-improve` 로 깨울 수 있었다 — 접속 비밀은 출력·문서·커밋에 복사하지 말 것).
  - `cd server && go test -race -p 1 -count=1 ./internal/httpapi/` — DSN 없이 돌면 0.2초 내로 끝나므로 **DB 테스트가 skip 된 것**이다. 통과 시간이 수십 초여야 실제로 돌았다는 뜻이고, `-run` + `-v` 로 새 테스트에 `--- SKIP` 이 없음을 확인할 것.
  - `cd server && go test -race -p 1 -count=1 ./...` (49 패키지, 수 분), `go build ./...`, `go vet ./...`
  - `gofmt -l` 은 **변경·신규 파일만** 보라 — `gofmt -l server/internal` 은 기존 9개 파일을 출력한다(이 저장소의 기존 상태). 독립 포맷 PR 금지.
  - `bash scripts/check-source-sizes.sh` (`compat_wave_handlers_final.go` 는 상한 77000 에 73k 대이므로 여유가 크지 않다 — 주석을 길게 쓰지 말 것).
  - 웹 변경이 없으므로 webapp 게이트는 돌릴 필요 없다. 돌린다면 `cd webapp && npm run typecheck && npm test` (이 저장소는 `webapp/scripts/run-vitest.mjs` 로 인터프리터를 고정한다 — `vitest` 를 직접 부르지 말 것).

- 테스트 배선 (확인된 선례 두 개를 합치면 손으로 만들 대역이 없다):
  - **배선 선례 — 같은 두 핸들러를 이미 띄우는 테스트가 있다**: `server/internal/httpapi/custom_profile_values_errors_postgres_test.go`(확인함). 거기서 그대로 가져올 것:
    - `db := newOperationsTestDB(t)` → `store.Migrate(ctx, db)` → `seedSidebarHandlerFixture(t, ctx, db)` (user-a 를 심는다) → `h := &handlers{customProf: customprofile.New(db)}`.
    - 헬퍼 `customProfileValuesRequest(t, body)` (같은 파일 116-135행) 이 `/api/v4/users/user-a/custom_profile_attributes` 로 actor==target 요청을 만들고 chi `userID` 파라미터와 `userIDKey` 를 심는다 — **actor==target 이라 `requireUserParamAccess` 가 auth 서비스 없이 통과한다**. 본문 문자열이 `""` 이면 GET 이 되는 분기가 있으니, "본문 없는 PATCH"(수용 기준 3)는 그 헬퍼를 그대로 쓰지 말고 `httptest.NewRequest(http.MethodPatch, ..., http.NoBody)` 로 따로 만들라.
    - `custom_profile_fields` 시드 INSERT(42-48행) — 값이 실제로 쓰였는지 단언하려면 필드가 있어야 한다.
  - **감사 선례**: `server/internal/httpapi/bulk_delete_body_cap_postgres_test.go` — 같은 결함 class 를 같은 모양으로 증명한 2026-10-04 회차의 테스트다. 감사행 단언(`waitForAuditRow`, 3.2초 폴링 — `audit.LogAsync` 는 goroutine + 3초 timeout)과 "양성 대조를 먼저 세우고 거절 요청의 0건을 관찰" 하는 순서를 그대로 가져올 것.
  - **주의**: 위 기존 테스트는 `h.audit` 를 넣지 않는다(nil). `h.audit == nil` 이면 `LogAsync` 호출이 아예 건너뛰어지므로 수용 기준 2 는 **거짓으로 통과한다**. 감사 서브테스트에서는 실제 `audit` 서비스를 반드시 주입하라(`router.go` 가 만드는 줄을 복사).
  - 라우트는 `router.go:1013-1016`(`r.Patch("/custom_profile_attributes/values", ...)`, `r.Patch("/users/{userID}/custom_profile_attributes", ...)`). 이 두 라우트는 **관리자 전용이 아니다**. `createCustomProfileField` 쪽(`router.go:1070-1072`)의 admin 그룹과 섞지 말 것.
  - 상한 초과 본문은 맵 **값**을 길게 만들어 1 MiB 를 넘기되 키 개수는 적게 유지하라 — 이 경로에는 개수 상한이 없다(확인됨: `tooManyBatchItems` 호출자에 `final.go:1123`(bulk_delete)만 있고 1296/1349 는 없다). 그래야 413 이 유일한 정답이 된다.
  - `httptest.ResponseRecorder` 로도 `*http.MaxBytesError` 는 그대로 올라온다(2026-10-04 회차가 같은 방식으로 413 을 관찰했다).

- 위험과 피할 것:
  - **같은 PR 에 다른 `_ = decodeCappedBody` 호출자를 넣지 말 것.** 남은 14곳(`final.go` 12곳 + `enterprise_compat_handlers.go:20` + `admin_compat_handlers.go:211`)은 `request_body.go:77-81` 이 "깨진 본문을 의도적으로 관용하는 compat 스텁" 이라고 명시한 읽기 전용/스텁 경로가 섞여 있다. 특히 `createCustomProfileField`(1219)·`patchCustomProfileField`(1245)는 **단건 쓰기**라 빈 본문 계약을 따로 정해야 하므로 범위 밖이다.
  - `request_body.go` 의 기존 네 헬퍼(`decodeCollectionBody`/`decodeOptionalCollectionBody`/`decodeCappedBody`/`drainCappedBody`/`tooManyBatchItems`) 동작을 바꾸지 말 것 — 호출자가 16곳이다.
  - `decodeCollectionBody`(`io.EOF` 도 400) 로 치환하면 **본문 없는 PATCH 가 400 이 되어 수용 기준 3 을 깬다**. 반드시 `Optional` 쪽을 쓸 것.
  - 되읽기(`GetUserValues`)·감사·응답 모양은 건드리지 말 것. 1300-1306 과 1357-1358 의 기존 주석이 그 설계 근거를 적어 두었다(되읽기 실패는 200 null 이 아니라 500).
  - 보호 경로를 피했다: `server/internal/store/migrations`, `auth`/`session`/`oidcauth`, `.github/workflows` 를 건드리지 않는다. `customprofile` 서비스도 수정하지 않는다(핸들러만).
  - 과거 교훈: 이 저장소는 httpapi 상태코드 재분류 계열 과제를 09-28~10-02 네 회차 연속으로 냈다. 이번 것은 **상태코드 재분류가 아니라 거부되지 않던 입력을 거부하는 것**(2026-10-04 bulk_delete 와 같은 계열)이며, 그 회차는 채택·릴리즈됐다.
  - 테스트는 **별도 `newOperationsTestDB` 인스턴스**로 격리하라. 같은 패키지의 형제 테스트가 중간에 `DROP TABLE` / `ALTER TABLE ... RENAME` 로 장애를 주입하므로 스키마를 공유하면 간섭한다.

- 차선 후보: **`postsByIDsReactions`(`compat_wave_handlers_late.go:311`)의 본문 읽기를 패키지 계약에 맞추기** (가치 2 / 위험 1 / S). 이 핸들러만 `body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))` 로 직접 읽어 **읽기 오류를 버리고 64 KiB 에서 조용히 자른다** — 65 KiB 짜리 정상 id 배열은 중간에서 끊겨 `json.Unmarshal` 이 실패하고 400 `"expected ids array or {ids}"`(본문 모양이 틀렸다는 뜻) 를 받는다. 실제 원인은 "너무 크다"(413)다. `request_body.go:11-24` 는 "모든 collection 모양 본문은 `decodeCollectionBody` 를 거친다" 고 적고 있는데 이 한 곳이 예외다. 다만 이 경로는 **읽기** 이므로 잘림이 Mattermost 호환 동작일 수 있고(같은 함수가 `len(ids) > 200` 절단도 한다 — 읽기 절단은 계약상 유지가 맞다), 바꿀 것은 "왜 거부되는지" 의 정확도뿐이어서 가치가 낮다. 1순위가 성립하지 않을 때만 고르고, 두 `json.Unmarshal` 폴백(`[...]` 와 `{ids:[...]}` 양쪽 수용)은 반드시 보존하라.
