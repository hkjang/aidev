# 회차 노트 2026-10-04-090203-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:02] base pinned — main@e076cae
- [러너 09:02] autonomy release — 

## 정찰 노트
- v0.34.54 가 사용자 섹션 두 Alert 을 한국어로 바꿔 release-ready 로 끝났으므로 같은 틀(adminErrors.ts + …ErrorAlert + apiError 헬퍼 테스트)을 **핸들러 하나 분량**으로 이어 사이트 생성·설정 두 Alert(AdminPage.tsx:1997·2121)을 골랐다. 보류의 M 짜리 "Alert 세 곳" 을 사이트(1순위)/망 구분(차선)으로 쪼개 프로덕션 2파일에 묶었다.
- 제친 후보: 숨긴 열 검색(세 회차 연속 차선에도 못 든 취향 판정 여지), rowKey 중복·DataTable 성능(관측·프로파일 없음), lint/build PATH 가림(재현 없음), 플레이크 조사(Postgres 없음).
- 추측으로 적은 것: SITE_CREATE_FAILED 500 의 현실적 원인(workspaces 조회가 비는 경우, admin.go:264)은 코드 경로로만 읽었고 재현하지 않았다. 문구를 중립으로 두라는 근거인 "sites.name 에 UNIQUE 없음"(001_initial.sql:34-49)은 확인했다.
- 이 회차는 읽기만 했다(worktree 에 web/node_modules 없음, Postgres 없음) — 테스트 184건 기준선은 앞 회차 기록이고 미재확인.
- 구현자 주의: describeUserError 의 기존 문구를 건드리면 adminErrors.test.mjs 의 기존 6건이 글자 그대로 깨진다. admin.go 는 손대지 말고, 상태 코드만으로 원인을 단정하지 말 것.
- [러너 09:06] scout done — 사이트 추가·설정 다이얼로그가 실패를 서버의 영문 문장·pgx 원문 대신 한국어로 안내한다 (가치 3 / 위�

## 구현 노트
- 두 사이트 다이얼로그의 Alert 이 서버 영문·pgx 원문을 그대로 띄우던 것을 `adminErrors.ts` 의 새 `describeSiteError` + `SiteErrorAlert` 로 한국어 안내로 바꿨다(프로덕션 2파일 + 테스트 1파일). `describeUserError` 와 합치지 않았고, 에러에서 code·message 를 읽는 `refusal` 리더와 레이아웃(`AdminNoticeAlert`)만 공유한다.
- **확신 없는 곳**: `SITE_CREATE_FAILED` 500 의 실제 원인을 **DB 로 재현하지 못했다**(Postgres 없음). admin.go:264 의 workspaces 조회가 빈다는 것은 코드로만 읽었다 — 그래서 그 추측에 기대지 않고 문구를 중립으로 두고 `no rows in result set` 을 특별 취급하지 않았다. 또 `INVALID_NAME` 이 공백만으로 실제로 온다는 것도 서버의 `TrimSpace`(admin.go:235) 를 읽은 것이고 실서버로 확인하지 않았다(브라우저 확인은 가짜 서버가 그 응답을 돌려주게 해서 했다).
- 일부러 안 한 것: 서버(admin.go)는 코드·상태·문장 모두 그대로 뒀다(검증의 정본). 나머지 `error.message` Alert 들(2410·2507·2954·3109·3245·3750, 1365·2758)은 각각 다른 핸들러의 코드 집합이라 손대지 않았다 — 망 구분(3245)이 다음 회차 1순위 유력.
- 과제서에 없던 것: 두 함수가 공통으로 쓰는 세 문장을 상수(`PASS_TO_ADMIN`·`UNREADABLE_PAYLOAD`·`LOST_REQUEST`)로 묶어 `describeUserError` 쪽도 그것을 쓰게 했다. 글자는 그대로이고, 기존 사용자 6건이 그 문장을 글자 그대로 단언하므로 안 바뀐 것이 테스트로 증명된다(191/191 통과).
- 다음 역할 주의: `web/test/adminErrors.test.mjs` 는 순수 모듈을 `.ts` 확장자까지 적어 import 한다 — 빼면 모듈을 못 찾는다. 새 테스트 7건은 DB·브라우저 없이 `node --test` 만으로 돈다. 브라우저 확인에 쓴 하네스는 `/tmp/sitewire` 에만 있고 커밋에 없다(`web/dist` 도 지웠다). `npx prettier --check` 는 게이트가 아니다.
- [러너 09:16] brief accepted — 채택 — 인용한 행 번호(AdminPage.tsx:1997·2121, admin.go 232·236·249·256·269·318·322·335·339·343·347·352·376, UserErrorAlert 3331-3349, �
- [러너 09:16] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 새 코드 9개를 admin.go 221~380(createSite·updateSite)과 1:1 대조해 전부 실재하는 코드임을 확인했고, 생성 다이얼로그에 시간대 자유입력 칸은 있고 세션 만료 칸은 없다는 주석의 주장(AdminPage.tsx:1986·2008)도 맞다. 직접 돌린 `node --test test/*.test.mjs` 191/191, `npm run lint`, `npm run build` 모두 통과. 기존 사용자 6건이 문장을 글자 그대로 단언하며 통과하므로 상수 추출이 문구를 바꾸지 않았음이 증명된다.
- 못 본 것: Postgres 가 없어 500 경로를 실서버로 재현하지 못했다(구현자와 같은 한계). 브라우저 DOM 배선도 재확인하지 않았다. Go 변경이 없어 go test 는 돌리지 않았다.
- 원장에 필수 `- 실패 재현:` 줄이 없다. 대신 diff 로 직접 확인했다 — describeSiteError 는 신규 함수라 수정 전에는 import 가 깨지고, `아는 코드의 안내는 서버의 영문 문장이 아니다` 가 결함을 직접 짚는다. 다음 회차는 그 줄을 남길 것.
- 승인이어도 남는 우려 두 개. (1) NOT_FOUND 를 「이미 삭제되었습니다」로 단정하는데, resolveSiteByID(analytics.go:202)는 가시 범위를 벗어난 사이트에도 같은 404 를 준다 — 「찾을 수 없습니다」가 정확하다. (2) 서버가 500 본문에 pgx 원문을 계속 싣는다(admin.go:269·352·376); 이번 변경으로 detail 캡션으로 내려가 노출은 좁아졌으나 근본은 서버 쪽이다. 둘 다 다음 회차 후보.
- 보안·법무 차단 사유 없음: 인가·세션·마이그레이션·의존성 무변경, 개인정보 신규 수집 없음, 노출면은 전보다 좁아졌다.
- [러너 09:19] review approved — 리뷰 승인 (risk=low)
- [러너 09:19] pr created — https://github.com/hkjang/Momento/pull/23
- [러너 09:23] ci failed — 성공이 아닌 검사: test=failure · 실패한 검사: ? 잡: test 
