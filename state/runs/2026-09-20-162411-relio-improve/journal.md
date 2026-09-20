# 회차 노트 2026-09-20-162411-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:24] base pinned — main@f0811df
- [러너 16:24] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 고른 이유: 직전 회차(c191148, merged)가 감사 metadata 를 화면에 냈는데 화면의 채널 필터(WEB/API/MCP/ADMIN)로는 그 값을 싣는 LOGIN·SSO 행을 좁혀 볼 수 없고 시각 열이 연월일뿐 — 같은 흐름의 마무리라 근거가 코드로 다 확인되고 Go·SQL·openapi 무변경으로 위험이 가장 낮음. 기간 필터(from/to)는 가치가 비슷하지만 openapi·계약 테스트·httpx 헬퍼까지 번져 M 에 가까워 차순위로 둠. ClientIP/DealsAtRisk 는 M, 메일·OAuth·로그인열거 후속은 기반 커밋 미머지로 착수 불가.
- 확신 없는 곳: `web/test` 에서 `../src/api.ts` 를 import 했을 때 Node 최상위 평가가 통과하는지 미확인(1~30행에 DOM 참조 없음, 나머지 미확인) — 실패하면 새 파일로 우회하라고 적음. `PHONE`/`PIGEON` 채널 grep 결과가 VOC 접수 채널이라는 것은 추정.
- 구현자가 조심할 것: `AdminPages.tsx` 는 한 줄 압축 스타일이라 포매터 금지·해당 줄만 편집. `date()` 는 다른 페이지가 쓰니 바꾸지 말고 `dateTime` 을 새로 둘 것. 테스트는 select 문자열 grep 이 아니라 헬퍼 동작으로. web build 뒤 anchor 복원으로 `git status` 깨끗한지 확인.
- 프로필: 0일 전 것이 지금 코드와 일치해(마이그레이션 014, 미머지 브랜치 3개 그대로) 다시 쓰지 않음.
- [러너 16:27] scout done — 감사 로그 화면이 로그인 조사를 한 화면에서 끝내게 하기 — 시각에 시·분·초, 채널 필터에 LOGIN·SSO, 상�

## 구현 노트
- 바꾼 것: `web/src/api.ts` 에 `dateTime()`(ko-KR·24시간·초까지·빈 값 `—`) 추가, 감사 목록 시각 셀과 상세 모달(시각·User-Agent 두 칸)에 사용, 채널 select 에 LOGIN·SSO 추가, ADMIN_GUIDE 5.3 채널 문장 수정. Go·SQL·openapi 무변경. 커밋 53c8ab6 (4 파일, +40/-5).
- 검증: `npm test` 11 pass(기존 8 + 신규 3, red→green 확인) · typecheck·build · `go build/vet/gofmt`·`go test -race ./internal/server/`(계약 테스트 포함) · check-static-assets·check-env-contract·previous-release-tag-test · 정적 바이너리 + throwaway PostgreSQL 17 로 `?channel=LOGIN` 이 LOGIN 3행만, `?channel=SSO` 200 빈 배열, `?channel=WEB` 에 LOGIN 없음, 실제 `occurredAt` 3개가 `dateTime` 으로 16:30:00/01/02 로 갈라짐. build 뒤 `git status` 깨끗(anchor 복원).
- 확신 없는 곳: 브라우저에서 모달을 실제로 열어 보지는 않음(headless 캡처 없음) — 긴 UA 가 `.audit-meta code{overflow-wrap:anywhere}` 로 접힌다는 것은 CSS 를 읽어 판단한 것이지 렌더로 본 게 아님. SSO 채널은 Keycloak 이 없어 실제 SSO 행으로는 못 봤고 필터가 200 빈 배열을 주는 것까지만 확인.
- 일부러 안 한 것: 감사 화면 Frame 부제("Web, API, MCP, Admin, Login과 Key 작업을…")는 과제서 범위 밖이라 그대로 둠(ideas 에 추가). PDF 재생성 안 함(과제서대로). `date()` 시그니처·출력 무변경.
- 다음 역할이 조심할 것: `web/test/dateTime.test.ts` 는 `process.env.TZ='Asia/Seoul'` 을 import 앞에서 고정하지만 시각 값 자체는 단언하지 않음(정규식으로 시:분:초 존재만) — 값을 고정하는 단언을 추가하지 말 것. `AdminPages.tsx` 는 한 줄 압축 스타일이라 포매터를 돌리면 diff 가 파일 전체로 번짐.
- [러너 16:31] brief accepted — 채택 — 과제서의 근거(`date()` 27행, select 573행, 모달 575행, 채널 producer 목록, `auditItem` 의 `userAgent`/`occurredAt` 키, `.audit-meta`
- [러너 16:31] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 확인: 4파일 diff 전체, 채널 producer 목록(LOGIN 3·SSO 2 실제 존재, PHONE/PIGEON 은 VOC 값), 서버 channel 필터가 바인딩 파라미터임(admin_operations.go:92), `.audit-meta` CSS, 새 테스트를 수정 전 api.ts 로 되돌려 red(8/1) → HEAD green(11) 재현, typecheck·build 뒤 `git status` 깨끗, go build/vet/gofmt.
- 못 본 것: 브라우저에서 모달 실제 렌더(긴 UA 접힘)·실제 SSO 행 — 구현자와 같은 공백.
- 남는 우려(결함 아님): `hour12:false` 는 ko-KR 에서 h23 으로 풀리지만 `hourCycle:'h23'` 이 더 명시적; ADMIN_GUIDE 의 "개인 키 작업은 WEB 채널" 은 관리자 대행 KEY_REVOKE(ADMIN 채널) 를 빼먹음; PDF 미재생성; Frame 부제 미갱신(ideas 에 있음).
- 릴리즈 노트에 적을 것: 감사 목록 시각 초 단위, 채널 필터 LOGIN/SSO, 상세 모달에 시각·User-Agent. API·DB·openapi 무변경이라 롤백은 revert 한 번으로 끝남.
- [러너 16:34] review approved — 리뷰 승인 (risk=low)
- [러너 16:34] pr created — https://github.com/hkjang/relio/pull/27
- [러너 16:38] ci passed — 검사 2개 모두 success
- [러너 16:38] merge done — 53c8ab6
- [러너 16:38] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
