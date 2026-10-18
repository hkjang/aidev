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
