# 회차 노트 2026-09-28-211204-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:12] base pinned — main@4d579ad
- [러너 21:12] autonomy release — 

## 정찰 노트
- 보류 목록의 "머지 안 된 회차의 rewind 고침이 main 에 없음"을 실제로 확인했고(CanvasPage.tsx:512-519 에 setRewind 없음), 읽어 보니 값이 목록의 3보다 크다 — 배너의 거짓 시각만이 아니라 readOnly(323)·이벤트 스트림 차단(692)까지 따라와 새 공간이 통째로 얼어붙는다. 파일 하나·상태 해제 하나라 위험 1.
- 제친 것: exportOutline 이름(가치 2, 차선으로 남김), safeFilename(가치 2이고 기존 단언 attachments_test.go:42 를 함께 바꿔야 함), 자리 기억 fit(같은 effect 뭉치를 건드려 충돌), 메일 개인 스위치(마이그레이션 = 보호 경로, M 초과).
- 추측으로 적은 것: Playwright 설정(`web/playwright.config.ts`)을 열지 않았다 — 과제서에 "미확인" 으로 적었고 구현자가 먼저 읽어야 한다. presentation/service.go 의 공백 제목은 `title := req.Title` 한 줄만 봤다. LoginPage 의 SSO return_to 는 v0.76 이 이미 고쳤을 수 있어 확인 전엔 고르지 말 것.
- 구현자가 조심할 것: e2e 에서 `page.goto` 로 공간을 바꾸면 새로고침이라 상태 누수가 재현되지 않는다 — 공간 메뉴(2275행)로 클라이언트 라우팅해야 한다. `setRewind(undefined)` 는 `params.spaceId !== activeSpace` 가드 안에만.
- CanvasPage 에는 단위 시험 파일이 없어(web/src 의 *.test.* 17개 확인) 증명은 실제 바이너리 + PG 를 지나는 e2e 뿐이다. 손으로 만든 대역은 이 배선 결함을 구조적으로 못 본다.
- [러너 21:17] scout done — 되감은 채 다른 공간으로 옮기면 그 공간이 거짓 날짜를 달고 얼어붙는다 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `CanvasPage.tsx:513` 의 `params.spaceId` effect 가드 **안**에 `setRewind(undefined)` 한 줄. 공간 메뉴는 클라이언트 라우팅이라 되감기 상태가 따라와, 새 공간의 현재 노트 위에 옛 공간의 시각이 배너로 뜨고 `readOnly` 가 켜지고 스트림이 막혔다. 프로덕션 1파일 1줄 + e2e 1개.
- 확신 없는 곳: **전체 e2e 는 이 환경에서 초록이 아니다.** `npx playwright test` 를 3회 돌렸고 매번 다른 3~6개가 실패했는데 stderr 가 `Received signal 11 SEGV_MAPERR` — 헤드리스 크로뮴이 죽는 것이고 제품 실패가 아니다. 근거: 고침을 되돌린 기준선 실행도 4개가 실패했고(auth, keyboard, presentation, 그리고 새 시험 = 의도한 red), 실패한 spec 을 격리해 돌리면 전부 통과한다. 그래서 collateral damage 는 전체 초록이 아니라 인접 spec 22개(space-rewind·canvas·space-memory·canvas-place) 3회 연속 전부 통과로 판정했다. 비평가가 전체를 돌려 다른 실패를 본다면 이 SEGV 를 먼저 의심할 것.
- 그 외 검증 못 한 것: Go 를 건드리지 않아 `go test` 는 돌리지 않았다. `vite build` 는 e2e 가 실제 번들을 필요로 해 여러 번 돌았고 통과.
- 일부러 하지 않은 것: `rewindTo`·`rewindRef` 동기화 effect·`readOnly` memo 는 손대지 않았다(`rewindRef` 는 effect 로 따라오므로 별도 갱신 불필요, 선언 순서상 스트림 등록 전에 갱신됨). i18n 새 문자열 없음. 같은 effect 뭉치를 건드리는 '자리 기억 fit' 과 별 과제인 '공백 한 칸 발표 제목' 은 과제서 지시대로 뺐다.
- 다음 역할이 조심할 것: 새 e2e 는 **실제 바이너리 + 실제 PostgreSQL 17 + 크로뮴**이 있어야 돈다(이번엔 도커 `umm-e2e-pg`, 포트 15441 — 15433/15434 는 다른 세션이 이미 쓰고 있었다). `UMM_E2E_COMMAND` 로 바이너리를 주고 `npm run build` 를 먼저 해야 서버가 최신 번들을 낸다. 시험 안의 공간 전환을 `page.goto` 로 바꾸면 초록인데 결함이 남는다.
- [러너 21:47] brief unstated — 구현자가 과제서 판정을 적지 않음
- [러너 21:47] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: `CanvasPage.tsx:512-523` 가드와 공간 전환 경로 6곳(openSpace 1767, addSpace 1783, 삭제 후 1829, 스위처 2297, 공간 관리 3395, 부트스트랩 476) — 전부 `navigate` 만 하므로 effect 가 반드시 지나고, 부트스트랩은 마운트 1회라 회귀 없음. 주석의 세 증상도 실제 코드(배너 2910 · readOnly 323 · 스트림 697)와 일치.
- 새 e2e 는 수정 전 코드에서 통과할 수 없다: `.space-switcher` 로 클라이언트 라우팅을 지나고 배너·`aria-pressed`·읽기 전용 문구에 더해 `readOnly` 가 대체하는 캡처 바(2837)에 실제로 써서 서버에서 돌아오는 것까지 단언. 원장의 실패 재현 출력이 이 증상과 정확히 맞음.
- 못 본 것: e2e 를 직접 돌리지 않았다(실제 바이너리+PG 17+크로뮴 필요, 예산 밖). 구현자가 보고한 전체 e2e 의 크로뮴 SEGV 불안정도 미확인 — 인접 spec 22개 3회 통과 판정을 신뢰했다.
- 승인이어도 남는 우려(다음 회차 몫): `rewindTo`(571)가 resolve 시점에 `activeSpace` 가 여전히 그 공간인지 보지 않는다 — 스냅샷 요청 중 공간을 바꾸면 같은 누수가 되살아난다. 로딩 덮개(z-index 10)가 툴바(5)를 가려 사람 클릭으로는 닿기 어려워 차단하지 않았다. 릴리즈 노트에는 "되감은 채 다른 공간을 열면 그 공간이 지금으로 열린다" 한 줄이면 충분.
- 보안·법무 차단 없음: 인가·비밀값·마이그레이션·개인정보 경로 무관, 1줄 revert 로 완전히 되돌아온다.
- [러너 21:54] review approved — 리뷰 승인 (risk=low)
- [러너 21:54] pr created — https://github.com/hkjang/umm/pull/160
- [러너 22:06] ci passed — 검사 1개 모두 success
- [러너 22:06] merge done — b65e032
- [러너 22:21] release published — v0.76.1
- [러너 22:23] assets verified — v0.76.1 자산 3개 (이전 v0.76.0: 3)
