# 과제서 (2026-10-04 정찰, base main@e076cae v0.34.54)

- 과제: 사이트 추가·설정 다이얼로그가 실패를 서버의 영문 문장·pgx 원문 대신 한국어로 안내한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: 두 사이트 다이얼로그의 Alert 이 `{create.error.message}`(AdminPage.tsx:1997)·`{update.error.message}`(AdminPage.tsx:2121) 로 서버 문장을 그대로 띄운다 — 자유 입력 칸인 「IANA 시간대」에 `Seoul` 처럼 적으면 `timezone must be a valid IANA timezone` 가, 500 으로 떨어지면 `SITE_CREATE_FAILED`/`SITE_UPDATE_FAILED` 가 `err.Error()` 를 그대로 실어 pgx 원문이 브라우저까지 흐른다(admin.go:269·352·376). v0.34.54 가 사용자 섹션 두 곳에 같은 일을 이미 했으므로(`adminErrors.ts` 의 `describeUserError` + `UserErrorAlert`) 선례·레이아웃·테스트 틀이 모두 있고, 이번 회차는 사이트 두 곳만 같은 자리로 옮긴다.

- 수용 기준:
  1) 사이트 생성 다이얼로그에서 시간대를 `Seoul` 로 적고 「생성」을 누르면 Alert 이 한국어로 무엇을 고쳐야 하는지 말한다(예: `시간대 이름이 올바르지 않습니다. Asia/Seoul 처럼 IANA 이름으로 적으세요.`). 사이트 설정(편집) 다이얼로그의 같은 실패도 같은 문구를 쓴다.
  2) `SITE_CREATE_FAILED`·`SITE_UPDATE_FAILED` 의 Alert 본문에 Postgres/pgx 표지(`SQLSTATE`, `violates`, `relation`, `no rows in result set`)가 섞이지 않는다. 서버 문장은 `UserErrorAlert` 와 같은 모양의 `detail` caption 으로만 남긴다(지우지는 말 것 — 관리자에게 전달할 유일한 단서다).
  3) 새 순수 모듈 함수에 대한 node:test 가, 「지금 Alert 과 똑같이 동작하는 항등 스텁」(= `error.message` 를 그대로 돌려주는 구현)에서 실패하고 구현에서 통과한다. 즉 테스트가 결함 자체를 짚는다. 모르는 코드는 서버 메시지로 되돌아가는 것(`default`)도 단언한다.

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `web/src/pages/adminErrors.ts` — `describeSiteError(error: unknown): UserErrorNotice` 를 **새로 추가**. `describeUserError` 와 합치지 말 것(코드 집합과 계약이 다르다; 운영자 지침: 계약이 다른 파서를 통합하지 않는다). `UserErrorNotice` 타입과 `STALE` 같은 문구 상수는 재사용 가능. 다뤄야 하는 코드 — createSite(admin.go:221-298): `INVALID_PAYLOAD`(232), `INVALID_NAME`(236), `INVALID_TIMEZONE`(249), `INVALID_ENGAGEMENT_THRESHOLD`(256), `SITE_CREATE_FAILED`(269, 500); updateSite(admin.go:314-381): `INVALID_ID`(318), `NOT_FOUND`(322·358·363), `INVALID_PAYLOAD`(335), `INVALID_TIMEOUT`(339), `INVALID_TIMEZONE`(343), `INVALID_ENGAGEMENT_THRESHOLD`(347), `SITE_UPDATE_FAILED`(352·376, 500). 더해 `REQUEST_FAILED`(web/src/api/client.ts 가 코드 없는 응답에 채운다)와 `default`.
  - `web/src/pages/AdminPage.tsx` — 1997 행(`새 분석 사이트` 생성 다이얼로그)과 2121 행(`SiteSettingsDialog`)의 두 `<Alert severity="error">{…error.message}</Alert>` 만 작은 `SiteErrorAlert` 로 바꾼다. `UserErrorAlert`(3331-3349)의 레이아웃(body2 message + caption detail, `wordBreak: "break-word"`)을 그대로 따르고, 두 Alert 컴포넌트를 하나로 합치려면 `notice` 를 받는 형태로만 할 것(describe\* 선택은 호출자에 남긴다). import 는 기존 77행(`import { describeUserError } from "./adminErrors";`)에 더하면 된다 — **tsx 쪽 import 는 확장자 없음이 현재 관례**다.
  - `web/test/adminErrors.test.mjs` — 기존 파일에 사이트 쪽 case 를 덧붙인다. 이 테스트들은 `"../src/pages/adminErrors.ts"` 처럼 **`.ts` 확장자를 적어** import 한다(현재 파일 그대로). 에러는 파일 18-21행에 이미 있는 `apiError(status, code, message)` 헬퍼를 쓸 것 — 이 파일은 `APIError` 를 **일부러 import 하지 않고**(16-17행 주석) code·message 를 shape 으로 읽는 계약을 테스트도 같은 방식으로 쓴다. 대역이 아니라 **진짜 `APIError`** 로 확인하는 책임은 아래 "프로덕션 배선 확인" 쪽에 있다(v0.34.54 가 그렇게 했다).

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (worktree 에 node_modules 가 없으니 `npm ci` 가 선행, 수 분. 현재 테스트 184건이 기준선 — 늘어난 수만큼 통과해야 한다)
  - Go 는 손대지 않으므로 필수는 아니지만, 사실 확인용으로 `go vet ./internal/httpapi/` 정도만.
  - 프로덕션 배선 확인(이 저장소의 최근 다섯 회차 관례): `npm run build` 의 `dist` 를 `/api/v1/me`·`/api/v1/sites` 를 흉내 낸 임시 서버에 올려 headless Chrome(google-chrome 설치됨, `puppeteer-core` 는 없으므로 /tmp 에 따로 설치)으로 `/admin?section=sites` 의 생성 다이얼로그를 실제로 채우고 「생성」을 눌러 Alert DOM 을 읽는다. 에러 객체가 손으로 만든 대역이 아니라 `api()` 가 HTTP 응답에서 만든 진짜 `APIError` 인 것이 요점이다. 임시 하네스는 /tmp 에 두고 `web/dist` 는 커밋 전에 지울 것.
  - `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 의존성·CI 단계가 없고 main 의 여러 파일이 이미 실패한다).

- 위험과 피할 것:
  - **서버를 고치지 말 것.** admin.go 의 코드·상태·문장은 그대로 두고 화면만 설명을 붙인다(서버가 권한·검증의 정본). 마이그레이션·`.github/workflows`·`docs/openapi.yaml` 도 건드릴 일이 없다.
  - **상태 코드만으로 원인을 단정하지 말 것** — v0.34.54 의 교훈이다. 특히 `sites.name` 에는 UNIQUE 가 없으므로(`internal/database/migrations/001_initial.sql:34-49`, UNIQUE 는 `site_key` 뿐) `SITE_CREATE_FAILED` 를 「이미 있는 사이트 이름」으로 설명하면 거짓이다. 이 500 의 현실적인 원인은 workspaces 조회가 비는 경우(`no rows in result set`, admin.go:264)와 DB 장애다 — 문구는 "사이트를 만들지 못했습니다 …" 처럼 중립적으로 두고 원문은 detail 로.
  - `INVALID_NAME` 은 버튼이 `disabled={!name}`(2006행) 이라 빈 칸으로는 닿지 않지만 공백만("   ") 넣으면 서버의 `TrimSpace` 에 걸려 실제로 온다 — 문구를 "이름을 입력하세요" 로만 두지 말고 공백 사례를 포함할 것.
  - 생성 다이얼로그에는 「세션 제한 시간」 칸이 **없다**(1953-1999: 이름·서비스 이름·허용 도메인·IANA 시간대·참여 기준 시간). createSite 는 0 을 30 으로 바꾸므로 `INVALID_TIMEOUT` 은 편집 쪽에서만 온다. 코드는 두 쪽 공통 함수에 넣어도 되지만 "생성에서도 난다" 고 적지 말 것.
  - 나머지 `error.message` Alert 들(AdminPage.tsx 2410·2507·2954·3109·3245·3750, 1365·2758)은 **이번 회차에서 건드리지 말 것** — 각각 다른 핸들러의 코드 집합이고, 파일 수와 검증 범위를 묶는 것이 이 회차의 설계다.
  - 기존 테스트가 출력 문자열을 글자 그대로 단언하는 곳이 많다. `describeUserError` 의 반환 문구를 바꾸면 `web/test/adminErrors.test.mjs` 의 기존 6건이 깨진다 — 사용자 쪽 문구는 손대지 말 것.

- 차선 후보: 망 구분(네트워크) 추가 실패도 한국어로 안내한다 — `AdminPage.tsx:3245` 의 `{create.error.message}` 가 `createNetwork`(admin.go:800-828) 의 코드를 그대로 띄운다. CIDR 을 손으로 적는 칸(3233-3238)이 있어 잘못된 CIDR 이 흔한 실패이고, 범위는 사이트 건과 같은 모양(파일 2+1개). 1순위가 성립하지 않으면 이것을 고를 것.
