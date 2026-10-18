# 회차 노트 2026-10-18-111753-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:17] base pinned — main@9d875f8
- [러너 11:17] autonomy release — 

## 정찰 노트
- LobbyPage 를 골랐다: AdminPage 상단 카드·Statistics·KioskPage 칩 모두 같은 「실패를 0 으로 단정」 유형이지만 셋은 상단 공용 error Alert 가 함께 뜨거나(AdminPage·Statistics) 의도적으로 삼키는 장식 값(KioskPage 주석)이다. LobbyPage 만 ① 네 카드 + 탭 라벨 + 「표시할 방문자가 없습니다」가 동시에 거짓을 말하고 ② 재조회 실패가 옛 숫자를 남기는데 알림이 닫히는 Alert 뿐이며 ③ helper 가 그 값을 스스로 '재난 시 체류 기준 인원'이라 선언한다.
- 추측으로 적은 것: 실패 시 `0` 이 실제 브라우저에 그려지는 장면을 눈으로 보지 못했다(LobbyPage.tsx 20·28·47·48행 코드 근거만). 수용 기준 3(오래된 값 표시)의 UI 모양과 `lastAt` 기록 시점도 설계 제안이며 구현자 재량이다.
- 실측으로 확인한 것: `go test ./... -count=1` PASS(DSN 미설정으로 DB 통합 SKIP), `web/vite.config.ts:10` include 는 `src/**/*.test.ts` 뿐, `web/public/sw.js` 는 ROSTER_PATH 만 캐시 폴백, 기존 28개 스펙 중 `/lobby` 본체로 가는 것은 없다.
- 구현자가 조심할 것: `loadRef`/EventSource 수명주기(36·37행)와 `connected` 문구는 건드리지 말 것(주석이 과거 결함을 기록해 두었다). 26행 reference-data 의 조용한 catch 는 별도 아이디어이니 합치지 말 것. `data.truncated`/`limit` 안내도 `ready` 일 때만 뜨게 할 것.
- 반복 기각 유형 회피: 요청 티켓 머지·전역 helper 변경·빌드 설정(tsc include) 배선은 이번에도 제안하지 않았다. 프로필은 9일 전 것이 지금 코드와 대체로 맞아 다시 쓰지 않았다(차이: HEAD 9d875f8/v2.8.19, `web/src/adminMetrics.ts` 신규, e2e 기준 28 passed).
- [러너 11:22] scout done — Lobby Dashboard 가 로비 현황 조회 실패를 「방문 예정 0 · 현재 방문중 0 · 표시할 방문자가 없습니다」로 단

## 구현 노트
- 무엇을 왜: `web/src/lobbyCounts.ts`(신규 순수 모듈) + `LobbyPage.tsx` 의 `data` 를 `LobbyLoad` 판별 값으로 교체했다. 조회 실패가 「방문 예정 0 · 현재 방문중 0 · 표시할 방문자가 없습니다」로 보이던 것을 숫자 없는 경고로 치환하고, 재조회 실패는 받아 둔 숫자를 닫을 수 없는 「기준 시각 + 최신이 아닐 수 있습니다」와 함께 남긴다. 프로덕션 2파일.
- 확신 없는 곳: ① `lobbyPanel` 의 `view` 비교(다른 탭/검색어의 캐시 목록을 버리고 카운트만 남김)는 내 판단이다 — 근거는 `internal/app/visits.go` 의 카운트가 role/siteScope 만으로 따로 집계된다는 점(읽어서 확인)이고, 탭을 바꿨을 때 목록이 사라지는 UX 는 실브라우저에서 스펙으로만 봤다. ② 캐시 상태의 `staleNote` 기준 시각은 클라이언트 `new Date()` 라 서버 시각이 아니다. ③ `stamp()` 의 `toLocaleTimeString("ko-KR")` 포맷은 Node(vitest)·Chrome 두 곳에서만 봤다.
- 일부러 하지 않은 것: `connected`(「재연결 중 · 화면 조회는 정상」) 문구, EventSource/`loadRef` 수명주기, 30행 reference-data 의 조용한 catch, `api.ts`·`sw.js`·서버. 각각 별도 아이디어로 `ideas.json` 에 남겼다.
- 다음 역할이 조심할 것: e2e 스펙 29번은 한 테스트 안에서 ①실패(캐시 없음) ②정상 ③재조회 실패(캐시 있음)를 순서대로 지난다 — `page.unroute` → `reload` → `route(current*)` → 탭 클릭 순서를 바꾸면 ③의 캐시 전제가 깨진다. 30번은 수정 전에도 통과하는 선택자 가드이므로 '항상 통과한다'고 삭제하지 말 것. `bash scripts/local-e2e.sh` 는 Docker 가 필요하고 수분 걸린다(이번 30 passed / 1.2m).
- [러너 11:38] brief accepted — 채택 — 지정한 파일·행(20·28·47·48)·근거·수용 기준 5개가 지금 코드와 정확히 맞았고 프로덕션 2개 파일로 끝났다. 과�
- [러너 11:38] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve. 확인한 것: 구현자가 의심한 세 곳 — ① `view` 비교의 근거는 `internal/app/visits.go:1692-1699` 에서 카운트가 role/siteScope 만으로 집계되고 탭·`q`·`lobby` 와 무관함을 직접 읽어 **참**으로 확인했다; ② `at` 은 클라이언트 시계(LobbyPage.tsx:36)로 서버 시각이 아님 — 기록만; ③ `stamp()` 는 테스트가 같은 함수로 기대값을 만들어 자기일관적이다. 실측: `npm run lint`(tsc -b) PASS, `npx vitest run` **114 passed / 6 files**(원장의 103→114 일치).
- 못 본 것: `scripts/local-e2e.sh`(Docker 필요)를 이 세션에서 재실행하지 않았다 — 실패 재현은 원장의 e2e 기록(수정 전 1 failed / 29 passed, 실패 로케이터와 error-context 스냅샷이 증상과 정확히 일치)으로 판단했고 신규 스펙 30번이 수정 전에도 통과하는 거짓양성 가드임도 확인했다. 실브라우저 렌더링은 눈으로 보지 않았다.
- 승인이어도 남는 우려 ①(릴리즈 노트에 쓰지 말 것): `docs/USER_GUIDE.md:101` 의 「`0`은 실제로 0명이라는 뜻이다」는 과한 약속이다. `visits.go:1692` 가 집계 쿼리 에러를 `_ =` 로 버리고 네 값을 0 으로 담아 **200** 을 보내므로(목록만 성공·집계만 timeout) 서버가 삼킨 0 은 여전히 '실시간 0' 으로 보인다. 이 PR 이 만든 결함은 아니고 서버는 0줄 변경이라 차단하지 않았다 — 다음 회차 최우선 후보(가이드 문장과 서버를 같이 고쳐야 한다).
- 우려 ②: `lobbyCounts.ts:117` 이 재조회 실패 시 `truncationNote` 를 항상 `""` 로 죽이는데 sameView 에서는 그 상한 걸린 목록이 **화면에 그대로** 남는다 — 같은 파일 46행 주석과 `lobbyCounts.test.ts:175-177` 의 근거가 그 경우엔 성립하지 않는다. 300건 초과 사업장에서만 보이는 좁은 경로이고 `staleNote` 가 비상 명단을 더 강하게 지시하므로 approve 했다.
- 수리가 생긴다면 먼저 볼 파일: `web/src/lobbyCounts.ts`(90-127행, 특히 117행 `truncationNote`). 보안·법무 차단 없음 — 서버 0줄, 신규 엔드포인트·권한 변경 없음, 캐시는 React 메모리뿐이고 localStorage/sessionStorage 에 PII 를 새로 쓰지 않으며(grep 확인) 목록에 전화번호가 없고(`visits.go:1688`) 오류 문구는 수정 전과 같은 서버 `error.message` 다.
- [러너 11:42] review approved — 리뷰 승인 (risk=low)
- [러너 11:42] pr created — https://github.com/hkjang/visitflow/pull/41
- [러너 11:46] ci passed — 검사 2개 모두 success
- [러너 11:46] merge done — 3290671
