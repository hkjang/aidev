# 과제서 (2026-10-03, main@69c81d0 / v0.34.53, 작업 트리 깨끗)

- 과제: 사용자 추가·편집 다이얼로그가 실패를 서버의 영문 문장·Postgres 원문 대신 한국어로 안내한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 두 다이얼로그는 실패를 `<Alert severity="error">{create.error.message}</Alert>`(web/src/pages/AdminPage.tsx:3502-3504)·`{update.error.message}`(:3603-3605)로 **서버 문장을 그대로** 띄운다. 그래서 가장 흔한 실패인 이메일 중복은 `createUser`(internal/httpapi/admin.go:914-916)가 `writeError(w, 409, "USER_CREATE_FAILED", err.Error())` 로 pgx 원문(`ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)` — `internal/database/migrations/001_initial.sql:17` 의 `email text NOT NULL UNIQUE` 에서 나온다)을 돌려주고 그것이 화면에 뜬다. 나머지도 `you cannot grant more authority than your own`, `password must be at least 12 characters`(internal/auth/auth.go:70-78) 처럼 전부 영문이다. 한국어 콘솔에서 무엇을 고쳐야 하는지 읽을 수 있게 되고, 스키마 내부(제약 이름·SQLSTATE)가 브라우저로 새는 것도 함께 막힌다.

- 수용 기준:
  1) 새 순수 모듈 `web/src/pages/adminErrors.ts` 의 `describeUserError(error)` 가 아래 "실제 서버 코드" 를 한국어 안내로 바꾸고, **모르는 코드는 서버 메시지를 그대로 돌려준다**(새 코드가 추가돼도 안내가 사라지지 않게).
  2) 두 다이얼로그의 Alert 이 그 안내를 읽는다 — 이메일이 중복된 생성 실패에서 화면에 `users_email_key`·`SQLSTATE`·`duplicate key` 문자열이 **나오지 않는다**.
  3) `web/test/adminErrors.test.mjs` 가 (a) 실제 코드 목록 각각의 안내를 단언하고, (b) 이메일 중복 pgx 원문을 통째로 입력해 안내에 원문 문자열이 섞이지 않음을 단언하고, (c) 모르는 코드(`"SOMETHING_NEW"`)는 서버 메시지가 그대로 나옴을 단언한다. 모듈을 상수 반환 스텁으로 바꾸면 여러 건이 실패해야 한다.

- 실제 서버 코드 (admin.go 를 열어 확인한 것. `createUser` 877행~, `updateUser` 924행~):
  - `createUser`: `INVALID_PAYLOAD`(400, 888), `INVALID_USER`(400, 892·896 — **두 원인이 한 코드**: 역할이 유효하지 않음 / `auth.PasswordProblem` 의 길이 문제), `ROLE_ABOVE_CALLER`(403, 902), `USER_CREATE_FAILED`(500, 909 해시 실패 / **409, 915 INSERT 실패 = 이메일 중복**)
  - `updateUser`: `INVALID_ID`(400, 928), `INVALID_PAYLOAD`(400, 944), `INVALID_ROLE`(400, 948), `SELF_DISABLE`(400, 952), `ROLE_ABOVE_CALLER`(403, 959 역할 부여 / 976 상위 계정 관리 — **서버 문장이 서로 다르다**), `USER_NOT_FOUND`(404, 965), `QUERY_FAILED`(500, 968), `SELF_PASSWORD`(400, 988), `WEAK_PASSWORD`(400, 992), `USER_UPDATE_FAILED`(500, 997·1004)
  - 주의: 과거 보류 아이디어에 적혀 있던 `EMAIL_TAKEN` 코드는 **존재하지 않는다**(중복은 `USER_CREATE_FAILED` 409 로 온다). 그 이름으로 매핑을 쓰면 아무 것도 안 걸린다.
  - `SELF_ROLE`(400, 980)·`SELF_DISABLE`·`SELF_PASSWORD`·`ROLE_ABOVE_CALLER` 는 2026-09-26·09-29 회차가 화면에서 미리 막아 둔 것들이다. 그래도 목록이 오래된 채 저장하면 서버가 거절하므로 매핑은 남겨 두고, 문구에 "목록이 오래되었을 수 있으니 새로 고친 뒤 다시 시도하세요" 를 넣으면 그 경쟁 상태에 실제로 쓸모가 있다.

- 선례(그대로 따를 것): `web/src/components/queryError.ts` 의 `describeQueryError(error, options)` 가 이미 같은 일을 한다 — 에러 클래스를 import 하지 않고 **shape 으로 `code` 를 읽고**(queryError.ts:44-53), `switch (code)` 로 한국어 문구를 돌려주고, 서버 원문이 유일한 단서인 경우만 `detail` 로 따로 들고 있고, `default:` 에서 `message || "…"` 로 되돌아간다. `describeUserError` 도 `{ message: string; detail?: string }` 같은 작은 결과를 돌려주고 같은 `default` 되돌림을 두면 된다. (`ApiError.code` 는 `web/src/api/client.ts:42-47` 에 있고, 코드가 없는 응답은 `"REQUEST_FAILED"` 로 채워진다 — 그 코드도 매핑에 둘지 판단할 것.)

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `web/src/pages/adminErrors.ts` — 신규. `describeUserError`.
  - `web/src/pages/AdminPage.tsx` — `UsersAdmin` 의 Alert **두 곳만**: 3502-3504(생성), 3603-3605(편집). import 한 줄 추가.
  - `web/test/adminErrors.test.mjs` — 신규. `import { describeUserError } from "../src/pages/adminErrors.ts";` — **`.ts` 확장자를 반드시 적을 것**(roleScope.test.mjs 선례; 빼면 모듈을 못 찾는다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build`
  - `npm ci` 는 **이 워크트리에 `web/node_modules` 가 없어서 선행 필수**이고 수 분 걸린다(이번 정찰에서 `ls web/node_modules` → 없음 확인. 그래서 정찰은 테스트를 돌리지 않았다 — 미실행).
  - `npm test` 는 `"${npm_node_execpath:-node}" --test test/*.test.mjs`(web/package.json:10). v0.34.53 에서 PATH 가림을 고쳤으니 그대로 쓰면 된다. 기준선은 테스트 178건.
  - 실패를 먼저 보일 것: 모듈 없이 테스트를 돌려 `ERR_MODULE_NOT_FOUND`, 그 다음 상수 반환 스텁으로 여러 건 실패를 확인.
  - 가능하면 2026-09-26~10-01 회차처럼 빌드된 dist 를 `/api/v1/me`·`/api/v1/users` 를 흉내 낸 임시 서버에 올려 headless Chrome(`google-chrome` 설치돼 있음, `puppeteer-core` 는 없어 /tmp 에 따로 설치)으로 `/admin?section=users` 의 Alert DOM 을 읽어 확인할 것. 생성 POST 에 `409 {"error":{"code":"USER_CREATE_FAILED","message":"ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505)"}}` 를 돌려주면 그 경로가 그대로 재현된다. 임시 하네스와 `web/dist` 는 커밋 전에 지울 것.

- 위험과 피할 것:
  - **서버를 손대지 말 것** — `internal/httpapi/admin.go`·`internal/auth` 는 권한의 정본이고, Go 통합 테스트는 `MOMENTO_TEST_POSTGRES_DSN` 없이 조용히 skip 되므로 이 환경에서 서버 변경을 증명할 수 없다. 서버가 코드를 더 잘 나눠 주는 것(중복을 전용 코드로)이 더 깔끔하지만 이번 회차 범위가 아니다.
  - **`USER_CREATE_FAILED` 는 409(중복)와 500(해시 실패) 둘 다 쓴다.** 코드만 보고 "이미 등록된 이메일" 이라고 단정하면 해시 실패를 거짓으로 설명한다. 메시지의 유일성 위반 표지(`duplicate key` / `23505` / `users_email_key`)나 `ApiError.status`(client.ts 에 있음 — 착수 전 필드 이름을 확인할 것, 정찰에서 `status` 필드 존재는 생성자 시그니처로만 봤고 노출 여부 **미확인**) 중 하나로 갈라야 하고, 갈리지 않으면 중복이라 단정하지 말고 "계정을 만들지 못했습니다" + `detail` 로 둘 것.
  - **`INVALID_USER` 도 두 원인(역할/비밀번호)이 한 코드다.** 메시지에 `password` 가 들어 있을 때만 비밀번호 문구를 쓰고, 아니면 둘 다 덮는 문구를 쓸 것. 두 원인 모두 폼이 이미 막고 있다(`assignableRoles`, `passwordWithinBounds`).
  - `AdminPage.tsx` 에는 `create.error` Alert 이 **1995·3243·3502 세 곳**에 있다(각각 사이트·다른 섹션·사용자). 이번 과제는 **사용자 두 곳만**이다 — 나머지를 같이 바꾸면 파일 수와 검증 범위가 늘고 과거 교훈대로 사람 손을 타게 된다.
  - 기존 테스트 중 **출력 문자열을 글자 그대로 단언하는 것이 많다**. 새 모듈만 추가하므로 충돌은 없어야 하지만, 성공 토스트(`meta.successMessage`, main.tsx:21-26)와 `showToast` 경로는 건드리지 말 것 — 전역 `onError` 는 **없다**(main.tsx 확인). 즉 이 다이얼로그 Alert 이 유일한 안내 지점이다.
  - `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 의존성·CI 단계 없음).
  - 보호 경로(`internal/auth`, `internal/database/migrations`, `.github/workflows`, `docs/openapi.yaml`)는 전혀 건드리지 않는다.

- 차선 후보: 표 검색이 숨긴 열의 값까지 뒤져 보이는 곳에 일치가 없는 행을 내놓는 것을 고친다 — `DataTable.tsx` 의 `filtered` 가 `columns` 전체를 훑는데 화면 칸·Highlighted·CSV 는 `visibleColumns` 만 쓴다. 필터 술어를 순수 모듈로 빼고 `visibleColumns` 를 넘긴다. '숨긴 열도 검색 대상' 이 의도일 수 있어 취향 변경으로 판정될 여지가 있다(위험 2).
