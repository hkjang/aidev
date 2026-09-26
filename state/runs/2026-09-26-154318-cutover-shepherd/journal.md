# PR 처리기 노트 2026-09-26-154318-cutover-shepherd — cutover PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-26-144255-cutover-improve)
# 회차 노트 2026-09-26-144255-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:42] base pinned — main@3cc0077
- [러너 14:42] autonomy release — 

## 정찰 노트
- 관리자 PUT 실패 표시를 골랐다: 코드에서 오동작을 하나 더 확정했기 때문이다 — `app/admin/page.tsx:287` 의 둘째 confirm 은 `tree.map(node => …)` 의 최상위 노드 `node.title` 을 써서 하위 항목 삭제 시 엉뚱한 제목을 보여 준다. 09-20~09-22 세 회차가 이 과제를 배정받고 스킬 부재로 멈췄지만 이번 세션은 세 스킬 모두 로드됐다.
- 제친 후보: validateActivityImport 단위 테스트(차선, 가치 3), 상태 전파 순수 함수화(09-20 에 막 고친 자리라 회피), Dockerfile fail-closed·로그인 429(보호 경로, 단독 회차), OIDC·메일 에스컬레이션(L, 기각/보류).
- 추측으로 적은 것: 실패 배너의 구체 UI 와 `submitActivityChange` 헬퍼 이름은 제안일 뿐 구현자가 정해도 된다. e2e 삭제 로케이터는 `ActivityTree.tsx:122` 버튼에 `title` 속성이 없어 `title="삭제"` 추가를 전제로 적었다.
- **검증 명령은 이번 회차에 하나도 실행하지 못했다**(node_modules 부재 + 셸 권한 제한). 72건/10건 수치는 09-20 기록이며 구현자가 직접 확인해야 한다.
- 구현자 주의: SWR 1초 폴링이 실패를 덮으므로 폴링 결과를 성공 증거로 쓰지 말고, 실패 주입은 손으로 만든 대역 대신 `page.route` 로 실제 컴포넌트·실제 fetch 경로에 할 것. lint 전에 playwright-report/·test-results/ 를 지울 것.
- [러너 14:51] scout done — 관리자 콘솔의 활동 변경 실패를 화면에 표시하고, 삭제 확인창 중복(둘째 창은 엉뚱한 제목)을 제거 (가�

## 구현 노트
- 왜/무엇: 관리자 활동 PUT 6곳이 응답을 안 읽어 실패가 안 보이던 것을 `submitActivityChange()` 한 곳으로 모아 `res.ok` 검사·서버 `error` 배너·401 시 `handleUnauthorized()` 로 고쳤고, `admin/page.tsx` 의 둘째 `confirm`(제목이 `tree.map` 최상위 노드라 하위 삭제 때 오표시)을 지웠으며, 저장 실패 시 인라인 편집·상황판 제목 편집이 닫히지 않게 했다. 프로덕션 2파일(app/admin/page.tsx, components/ActivityTree.tsx) + e2e 1 + ADMIN_GUIDE 1줄.
- 확신 없는 곳: (1) 실패 후 트리 `select` 는 최대 1초간 사용자가 고른 값을 그대로 보여 주다 SWR 폴링이 되돌린다 — 배너가 뜨긴 하지만 값 자체를 즉시 되돌리진 않는다(ideas.json 에 후속 항목으로 적음). (2) 네트워크 예외 경로(`catch`)는 e2e 로 주입하지 않았다 — 500/401 만 `page.route` 로 검증했다. (3) `handleSave` 는 `onUpdateActivity` 가 `false` 를 줄 때만 편집을 유지한다. 콜백이 `void` 를 주는 비관리자 경로는 전과 같이 닫힌다(의도).
- 일부러 안 한 것: 상태 전파 규칙 통합(`onStatusChange` 계산 로직)과 `collectDescendantIds` 는 09-20 에 고친 자리라 손대지 않았고, `app/api/activities/route.ts` 의 PUT 계약도 그대로다. PDF 는 md2pdf 부재로 안 구웠다(ADMIN_GUIDE.md 6장만 1줄 추가 — md 와 PDF 가 또 벌어짐).
- 다음 역할 주의: e2e 는 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 가 필요하고 webServer 가 127.0.0.1:3100 dev 서버를 띄운다. **`npm run lint` 전에 `rm -rf playwright-report test-results`** 를 하지 않으면 trace 뷰어 번들을 검사해 수천 건이 난다. 새 spec 은 `beforeEach` 에서 activities+dashboardTitle 을 저장하고 `afterEach` 에서 되돌린다.
- 검증(전부 실행함): lint 오류 0 / `tsc --noEmit` 통과 / `test:unit` 79건·25 suites 통과 / `test:e2e` 16건 통과(기존 11 + 신규 5) / `npm run build` 통과. 신규 5건은 수정 전 전부 실패(확인창 실제 2건·둘째 메시지가 `"[E2E] 최상위 작업"`)를 먼저 확인했다.
- [러너 15:00] brief accepted — 채택 — 과제서의 근거가 코드와 정확히 일치했고(6곳 미검사 PUT, 287행 confirm 의 최상위 node.title, handleSave 의 즉시 닫기) �
- [러너 15:00] verify passed — 검증 1개 통과 (policy)
- [러너 15:00] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 15:00] pr created — https://github.com/hkjang/cutover/pull/6

## 심사 노트
- 확인한 것: node_modules 를 설치해 lint 0 / tsc 통과 / test:unit 79건 / test:e2e 16건 / build 를 직접 돌렸고, 프로덕션 2파일만 origin/main 으로 되돌려 신규 spec 5건이 전부 실패함을 반대 방향으로 확인한 뒤 HEAD 로 복원했다(트리 clean).
- 실패 주입이 page.route 로 실제 컴포넌트·실제 fetch 경로에 들어가고 GET 폴링은 fallback 으로 통과시켜, 폴링이 실패를 덮는 함정을 피한 것도 확인했다.
- 보안: 새 경로·권한 확대 없음, route.ts 미변경, 배너 문구는 서버의 고정 문자열이라 내부 정보가 새지 않으며 401 처리가 오히려 좁아졌다.
- 못 본 것: 네트워크 예외(catch) 경로는 e2e 로 주입되지 않았고(코드 검토만 함), 상태 select 가 실패 후 최대 1초간 고른 값을 보여 주는 것은 후속 과제로 남았다. docs PDF 는 md 와 여전히 벌어져 있다.
- 권고 merge 의 근거: 실행한 검증이 모두 통과하고 신규 테스트가 변경을 실제로 고정하며 revert 가능한 UI 변경뿐이다. ADMIN_GUIDE.md:327 의 `폴링이 이전 값으로 되돌립니다` 가 편집·제목 저장에는 맞지 않는 점만 notes 로 남겼다.
