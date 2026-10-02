# 회차 노트 2026-10-03-034756-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:48] base pinned — main@69c81d0
- [러너 03:48] autonomy release — 

## 정찰 노트
- 보류 목록에서 유일하게 **서버 코드를 직접 열어 확인할 수 있는** 후보여서 골랐다. 숨긴 열 검색·CSV format·rowKey 는 모두 "의도일 수 있다"·"재현/문서 확인 선행" 이라 네 회차 연속 미선정이었고, DataTable 은 15개 파일·62곳이 쓰는 단일 구현이라 취향 판정 위험이 더 크다.
- 확인한 것: createUser(admin.go:877)·updateUser(924)의 코드 전부, 이메일 UNIQUE(001_initial.sql:17), Alert 두 곳(AdminPage.tsx:3502·3603)이 서버 message 를 그대로 띄우는 것, 전역 onError 가 **없는** 것(main.tsx), 선례 queryError.ts, node_modules 부재.
- **추측으로 적은 것**: `ApiError` 가 `status` 를 필드로 노출하는지 미확인(client.ts:42-47 의 `code` 만 봤다). 409 중복과 500 해시 실패를 가르는 데 status 를 쓰려면 먼저 확인하고, 없으면 메시지의 유일성 위반 표지로 갈라야 한다.
- 구현자가 조심할 것: 보류 목록에 오래 적혀 있던 `EMAIL_TAKEN` 코드는 **존재하지 않는다**(중복은 USER_CREATE_FAILED 409). 그 이름으로 매핑하면 아무 것도 안 걸린다. `create.error` Alert 은 세 곳(1995·3243·3502)에 있으니 사용자 두 곳만 건드려 프로덕션 2파일로 묶을 것.
- 테스트는 돌리지 않았다(web/node_modules 부재, Postgres 부재). 프로필은 1일 전 것이 현재 코드와 맞아 다시 쓰지 않았다(기준만 12ba629 → 69c81d0/v0.34.53 으로 한 릴리즈 전진).
- [러너 03:52] scout done — 사용자 추가·편집 다이얼로그가 실패를 서버의 영문 문장·Postgres 원문 대신 한국어로 안내한다 (가치 3 / 

## 구현 노트
- 두 사용자 다이얼로그의 Alert 이 서버 영문·pgx 원문을 그대로 띄우던 것을 새 순수 모듈 `web/src/pages/adminErrors.ts` 의 `describeUserError` 로 한국어 안내로 바꿨다(queryError.ts 선례: shape 으로 code 읽기, `default` 는 서버 메시지로 되돌림). 프로덕션 2파일 + 테스트 1파일.
- **확신 없는 곳**: (1) `ROLE_ABOVE_CALLER` 를 `/administer/` 로, `INVALID_USER` 를 `/password must be/` 로 가르는 것은 **서버 문장에 의존한다** — admin.go 의 그 문장이 바뀌면 안내가 조용히 '둘 다 덮는' 쪽으로 떨어진다(틀린 안내는 아니지만 덜 구체적이 된다). 비평가가 볼 가치가 있는 곳은 여기다. (2) 한국어 문구의 어휘 선택(예: "이 사용자를 가리킬 수 없습니다")은 취향 판정 여지가 있다. (3) `web/dist` 는 지웠고 `git status` 로 3파일만 확인했다.
- **검증 못 한 것**: Go 테스트는 돌리지 않았다(`MOMENTO_TEST_POSTGRES_DSN` 없어 통합 테스트가 조용히 skip 되고, 서버는 애초에 손대지 않았다). 실제 Postgres 가 정말 그 문장을 내는지는 재현하지 않았고 과제서가 인용한 원문을 그대로 입력으로 썼다.
- **일부러 하지 않은 것**: `APIError.status` 는 실제로 노출돼 있지만 `USER_CREATE_FAILED` 를 가르는 데 쓰지 않았다 — 그 INSERT 는 무엇이 틀려도 409 라서 끊긴 연결을 "이미 등록된 이메일" 로 설명하게 된다. 서버가 중복에 전용 코드를 주는 쪽이 더 깔끔하지만 범위 밖(서버 변경은 이 환경에서 증명 불가). AdminPage.tsx 의 나머지 Alert 세 곳(1996·2120·3244)은 다른 핸들러의 코드 집합이라 ideas.json 에 M 으로 남겼다.
- **다음 역할이 조심할 것**: `adminErrors.ts` 는 `./passwordRule.ts` 를 **확장자까지 적어** import 한다 — 값 import 라서 필수다(type-only 인 `attention.ts` 와 다르다). 빼면 `npm test` 가 `ERR_MODULE_NOT_FOUND` 로 떨어진다. 테스트가 `PASSWORD_RULE` 문장("12자 이상, 72바이트(한글 24자) 이하.")을 글자 그대로 단언하므로 비밀번호 규칙을 바꾸면 `adminErrors.test.mjs` 도 함께 고쳐야 한다. 브라우저 확인 하네스는 `/tmp/verify-adminErrors` 에만 있고 저장소에는 없다.
- [러너 04:03] brief accepted — 채택 — 인용한 행 번호(admin.go 888·892·896·902·909·915 및 updateUser 전체 코드 목록, AdminPage.tsx:3502-3504·3603-3605, queryError.ts:44-
- [러너 04:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: admin.go createUser/updateUser 전문과 테스트 픽스처 20건의 (status, code, message) 대조 — 전부 일치. ROLE_ABOVE_CALLER 3곳(902·959·976)에서 `administer` 가 976 에만 있는 것, auth.go:70-78 의 `password must be`, users 테이블 UNIQUE 가 email 뿐이고 oidc_subject 는 INSERT 열에 없어 NULL 인 것(= '이미 등록된 이메일' 오설명 경로 없음), updateUser 가 email 을 바꾸지 않는 것. 게이트는 실제로 돌렸다: lint 무출력, npm test 184 pass/0 fail, vite build 성공, git status 깨끗.
- 구현자가 의심한 자리(서버 영문 문장 의존)를 시험했고 **현재 코드에서는 정확**했다 — 결함이 아니라 서버가 전용 코드를 주지 않는 데서 오는 구조적 취약함이다. 승인.
- 못 본 것: 브라우저 DOM 배선은 구현자의 /tmp 하네스 기록을 신뢰했고 직접 재현하지 않았다. Go 테스트·통합 테스트는 서버 0줄 변경이라 돌리지 않았다. 한국어 문구의 어휘는 취향 판정으로 두었다.
- 승인 후에도 남는 우려: (1) INVALID_USER 분기만 detail 을 싣지 않아 서버에 세 번째 원인이 생기면 그 문장이 화면에서 사라진다 — 모듈 주석의 '모르는 것은 서버 메시지로 되돌린다' 원칙과 이 한 곳만 어긋난다. (2) USER_CREATE_FAILED(비중복)·USER_UPDATE_FAILED·QUERY_FAILED 는 여전히 Postgres 원문을 detail 로 띄운다(수정 전과 동일 노출, orgAdmin 게이트 뒤 — 신규 노출 아님). 릴리즈 노트는 "이메일 중복 등 사용자 추가·편집 실패를 한국어로 안내" 로 적고 '모든 서버 오류가 한국어' 라고 쓰지 말 것.
- 다음 회차용: 근본 수리는 서버에 중복 전용 코드(EMAIL_TAKEN 류)와 ROLE_ABOVE_CALLER 분리이고, 보류 목록의 나머지 Alert 세 곳(AdminPage.tsx:1996·2120·3244)이 같은 패턴을 기다린다.
- [러너 04:07] review approved — 리뷰 승인 (risk=low)
- [러너 04:07] pr created — https://github.com/hkjang/Momento/pull/22
- [러너 04:12] ci passed — 검사 1개 모두 success
- [러너 04:12] merge done — cd5e23c
- [러너 04:25] release published — v0.34.54
- [러너 04:27] assets verified — v0.34.54 자산 2개 (이전 v0.34.53: 2)
