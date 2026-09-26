# 과제서 — 2026-09-26-203224-moyro-improve (moyro)

- **과제**: custom profile 값 PATCH 두 경로의 되읽기 오류 삼킴을 형제 GET 과 같은 500 으로 통일 (가치 3 / 위험 1 / 작업량 S)

- **왜**: `PATCH /api/v4/custom_profile_attributes/values` 와 `PATCH /api/v4/users/{userID}/custom_profile_attributes` 는 쓰기 뒤 `out, _ := h.customProf.GetUserValues(...)` 로 되읽으면서 오류를 버려, 저장소 장애 때 `200` + 본문 `null` 을 낸다. 같은 `GetUserValues` 호출이 형제 경로 `GET /api/v4/users/{userID}/custom_profile_attributes` 에서는 `500 api.custom_profile.values.get.app_error` 이므로, **같은 값을 읽는 세 경로가 같은 장애에 다른 답을 한다**. 고치면 클라이언트가 `null` 을 "이 사용자는 커스텀 속성이 없다"로 오해해 화면의 속성을 지우는 일이 없어지고, 세 경로의 오류 계약이 하나가 된다.

- **수용 기준**:
  1. `PATCH /api/v4/custom_profile_attributes/values` 가 되읽기 실패 시 `200` + `null` 이 아니라 `500` + 오류 id `api.custom_profile.values.get.app_error`(형제 GET 과 **같은 id**) 를 낸다.
  2. `PATCH /api/v4/users/{userID}/custom_profile_attributes` 도 같은 입력에 같은 상태 코드·같은 오류 id 를 낸다(두 경로 불일치 제거).
  3. 정상 경로의 응답은 불변: 값이 있는 PATCH 는 여전히 `200` + `{field_id: value}` 맵(빈 결과는 `{}`), 401 / 503 / `PatchUserValues` 실패의 `500 api.custom_profile.values.patch.app_error` 와 `requireUserParamAccess` 의 401/403 은 그대로다. 새 오류 id·마이그레이션·서비스 계약 변경 없음.
  4. 테스트가 증명할 것: **수정 전 코드에서 실제로 실패**하는 회귀 테스트가, 되읽기만 실패하는 요청에 대해 두 PATCH 경로가 각각 `500` 을 내고 본문이 `null` 이 아님을 보이며, 같은 장애 상태에서 형제 GET 도 `500` 임을 같은 테스트 안에서 확인한다(세 경로 일치). 정상 PATCH 의 `200` + 맵 반환도 같은 테스트가 지킨다.

- **되읽기만 실패시키는 결정론적 방법 (이번 정찰의 핵심 발견 — 지난 회차가 막혔던 지점)**:
  `internal/customprofile/service.go:199` `PatchUserValues` 는 **`if len(values) == 0 { return nil }`** 로 시작한다 — 빈 맵이면 DB 를 전혀 건드리지 않고 성공한다. 반면 `GetUserValues`(같은 파일 173) 는 무조건 `SELECT field_id, value FROM custom_profile_values WHERE user_id=$1` 를 돈다. 따라서 격리 스키마에서 `DROP TABLE custom_profile_values` 를 하고 **본문 `{}` 로 PATCH** 하면 `PatchUserValues` 는 성공(no-op)하고 `GetUserValues` 만 실패한다 → 정확히 되읽기 실패 경로. 손으로 만든 대역·주입 훅이 전혀 필요 없고 프로덕션 배선과 실제 타입 그대로다.
  대조군(같은 테스트에 함께 두면 좋다): 같은 `DROP TABLE` 상태에서 **본문에 값이 있는** PATCH 는 `PatchUserValues` 가 먼저 실패하므로 지금도 `500 api.custom_profile.values.patch.app_error` 다 — 이 단언은 수정 전후 모두 통과해야 한다(패치 오류 매핑을 건드리지 않았다는 증거).

- **건드릴 파일** (프로덕션 1개 + 테스트 1개):
  - `server/internal/httpapi/compat_wave_handlers_final.go:1290` — `patchCustomProfileValuesGlobal` 의 `out, _ := h.customProf.GetUserValues(r.Context(), uid)` 를 오류 검사로 바꾸고, 오류면 `writeError(w, 500, "api.custom_profile.values.get.app_error", err.Error())` 후 return. 형제 GET(같은 파일 1305 `getUserCustomProfileValues`)이 쓰는 문장과 동일하게.
  - `server/internal/httpapi/compat_wave_handlers_final.go:1334` — `patchUserCustomProfileValues` 의 같은 줄을 같은 방식으로. 두 곳이 같은 id 를 쓰는지 확인할 것.
  - `server/internal/httpapi/custom_profile_values_errors_postgres_test.go` (**신규**) — 회귀 테스트. 같은 디렉터리의 `preferences_errors_postgres_test.go` / `reminder_errors_postgres_test.go` 를 형틀로 쓸 것(일회용 PostgreSQL 16 + 격리 스키마 + `store.Migrate` + 실제 `customprofile.New(db)` + 실제 라우터/핸들러). `internal/customprofile` 과 custom profile 용 httpapi 테스트 파일은 **현재 하나도 없다** — 이 파일이 첫 커버리지다.
  - 세 줄 이상 늘어나면 쪼갤 것. 이 과제는 프로덕션 파일 1개로 끝나야 한다.

- **검증 명령** (이 저장소에서 실제로 도는 것):
  - `cd server && go vet ./... && go build ./...`
  - `cd server && MOYRO_TEST_POSTGRES_DSN=... go test -race -p 1 -count=1 ./internal/httpapi/` — **DSN 이 없으면 DB 테스트가 통째로 skip 되므로 `ok` 만 보고 통과라고 하면 안 된다.** 로컬 컨테이너는 포트 `55433` 을 쓴다(55432 는 점유된 적 있음).
  - 전체: `cd server && MOYRO_TEST_POSTGRES_DSN=... go test -race -p 1 ./...` (`-p 1` 유지, 수 분)
  - `gofmt -l` 로 수정·신규 파일만 clean 확인(기존 9개 파일의 드리프트는 이번에 건드리지 말 것)
  - 루트 `bash scripts/check-source-sizes.sh` — `compat_wave_handlers_final.go` 도 상한이 있으니 확인
  - 웹 변경 없음 → webapp 검증 불필요. 만약 `webapp` typecheck 를 돌려 `useDraft.test.tsx:196 TS2345` 가 보이면 **이 변경 탓이 아니다**(교훈 2026-09-09, `@types/node` 스코프 문제).

- **위험과 피할 것**:
  - **감사 로그를 건드리지 말 것.** 두 핸들러는 되읽기 *이전에* `h.audit.LogAsync(..., audit.ActionCustomValuesPatch, ...)` 를 남긴다. 값이 있는 PATCH 는 쓰기가 실제로 커밋된 뒤이므로 그 감사 기록은 옳다 — 되읽기 실패로 500 을 내더라도 감사 기록을 지우거나 순서를 바꾸지 말 것(범위 밖이고 별개 판단이 필요하다).
  - `decodeCappedBody` 의 버려진 오류(`_ = decodeCappedBody(w, r, &values)`, 1282·1326)를 **이번에 같이 고치지 말 것** — 차선 후보로 따로 둔다. 빈 본문(`io.EOF`)까지 400 으로 만들면 지금 200 을 받는 호환 클라이언트가 깨질 수 있어 판단이 따로 필요하다. 다만 위의 `{}` 주입 기법이 이 경로와 겹치므로, 테스트는 **본문을 명시적으로 `{}` 로 보내** 빈 본문/decode 오류와 섞이지 않게 할 것.
  - `customprofile.Service` 의 시그니처·SQL 을 바꾸지 말 것. `PatchUserValues` 의 `len(values)==0` 조기 반환은 그대로 둘 것(이것이 테스트의 주입 수단이다).
  - 보호 경로 무관: auth/session/migrations/.github/workflows 를 건드릴 이유가 없다. `requireUserParamAccess` 의 `ok, _ := HasRole`(fail-closed, 의도적일 수 있음)도 건드리지 말 것.
  - `DROP TABLE` 은 반드시 테스트의 격리 스키마 안에서만(기존 테스트들의 패턴 그대로). `CASCADE` 필요 여부는 실제로 확인할 것.
  - 문자열 grep 을 증거로 제출하지 말 것 — RED 실행(수정 전 실패) 로그가 증거다.

- **차선 후보**: `patchCustomProfileValuesGlobal` / `patchUserCustomProfileValues` 의 버려진 `decodeCappedBody` 오류 — 깨진 JSON 이나 1MiB(`collectionBodyMaxBytes`) 초과 본문이 `values == nil` 로 떨어져 `PatchUserValues` 가 no-op 성공하고, 클라이언트는 **`200` + 기존 값 그대로** 를 받아 "패치가 적용됐다"고 믿는다(쓰기가 조용히 사라진다). 고치는 방향은 깨진 JSON·초과 본문은 `400`, **빈 본문(`errors.Is(err, io.EOF)`)은 지금처럼 no-op 200** 으로 호환 유지 — 저장소에 `io.EOF` 를 이렇게 갈라 쓰는 선례가 있다(`native_work_items.go:91`, `native_automations.go:60`, `browser_session.go:23`). 1순위가 성립하지 않으면 이것을 고를 것.

---
### 작업량 산정 근거 (bottom-up, 무엇이 포함/제외인가)
- 포함: 프로덕션 2개 호출 지점 수정(각 ~4줄) / 신규 postgres 테스트 파일 1개(~120줄, 기존 형틀 재사용) / RED 실행 1회 + GREEN 실행 1회 / `go vet`·`go build`·전체 `go test -race -p 1 ./...`·`gofmt -l`·소스 크기 검사.
- 제외(이번 범위 아님): `decodeCappedBody` 오류 처리(차선 후보), `customprofile` 패키지 단위 테스트 신설, 감사 로그 순서 조정, 웹 변경.
- 유추 기준: 같은 모양의 직전 두 회차 — 2026-09-22 `getPreferenceByName` 404/500 분리(프로덕션 1파일 + 테스트 1파일, S, 완료), 2026-09-25 리마인더 404/403→500(프로덕션 2파일 + 테스트 1파일, S, 완료). 이번 건은 그 사이 규모다.
- 추정: **25~40분**(10번 중 8번 이 안에 든다). 가장 큰 변동 요인은 전체 `go test -race -p 1 ./...` 의 수 분과 postgres 컨테이너 준비.
- 우발 대비(known unknown): 아래 "미확인" 의 주입 전제가 틀릴 확률을 약 2/10 로 본다 → 그 경우 차선 후보로 전환하며 같은 세션 안에서 끝난다(차선은 주입이 필요 없고 요청 본문만으로 재현된다). 이 여유는 과제 안에 이미 반영했고 추정치에 따로 덧붙이지 않았다.

### 미확인으로 남긴 것
- 이번 세션에서 **테스트를 실제로 돌리지 않았다**(정찰은 코드 변경 금지 + postgres 컨테이너 준비 비용). 위 `{}` + `DROP TABLE` 주입은 `PatchUserValues` 의 `len(values)==0` 조기 반환과 `GetUserValues` 의 무조건 `Query` 를 **소스로 읽어** 도출한 것이고, 실제 실행으로 확인하지는 않았다. 구현자는 RED 실행으로 먼저 이것을 확인할 것 — 만약 `{}` PATCH 가 예상과 달리 이미 500 이라면 주입 전제가 틀린 것이니 과제를 재평가하고 차선 후보로 갈 것.
- `compat_wave_handlers_final.go` 의 `check-source-sizes.sh` 상한 여유분은 이번에 확인하지 않았다.
- 프로필(2026-09-22, 4일 전)은 이번에 본 범위(httpapi 구조·오류 분기 관례·검증 명령·함정)와 어긋나지 않아 새로 쓰지 않았다.
