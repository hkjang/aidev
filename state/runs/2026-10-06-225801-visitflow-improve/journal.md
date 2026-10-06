# 회차 노트 2026-10-06-225801-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:58] base pinned — main@41309b4
- [러너 22:58] autonomy release — 
- [러너 23:03] scout done — 「내 API 키」 화면이 받아 보지 못한 정책을 사실처럼 말하고 폐기 실패를 삼키는 것 닫기 (가치 3 / 위험 1

## 구현 노트
- `KeysPage.tsx` 의 `policy` 를 하드코딩 추측값(`{read,write,mcp}/90일/10개`) 초기값에서 `ApiKeyPolicy | null` 로 바꿨다 — 정책 조회가 실패해도 info Alert·생성 대화상자 체크박스·「생성」이 그 추측을 사실처럼 쓰던 것을 닫았고, `revoke` 를 같은 파일 `rotate` 꼴 try/catch 로 감쌌다. 프로덕션 1파일.
- 과제서는 `policyLoaded` 플래그를 제안했지만 null 로 갔다: 플래그는 추측값을 그대로 들고 있는 둘째 진실원이 되고, null 이면 `tsc` 가 모든 독자에게 null 처리를 강제한다. 독자가 "한 값만 읽는다" 는 요구는 지켰다.
- **확신 없는 곳**: ① 「생성」 `disabled` 의 `!policy` 항은 **현재 코드에서 중복이다** — `policy===null` 이면 `openCreate` 가 `scopes=[]` 로 두므로 `!scopes.length` 가 이미 막는다. 의도를 게이트에 명시하는 방어로 남겼고 e2e 가 단독으로 그 항을 입증하지는 않는다(비평가가 먼저 볼 곳). ② `load()` 실패 시 **이전에 성공한 정책은 null 로 되돌리지 않는다**(RosterPage v2.8.16 의 "마지막으로 받은 값" 선례). 즉 `policy===null` 은 "한 번도 받지 못함" 이다 — 늦은 실패는 기존 error Alert 만 띄운다.
- 일부러 하지 않은 것: 서버 0줄(`apiKeyPolicy` vs `validScopes` 의 빈 값 fallback 불일치는 시드·설정 검증 때문에 대역 없이 재현 불가라 ideas.json 에 남겼다), `api.ts`·CSRF·`confirm()` 문구·`scopes` 초기값 미변경, `.tsx` 컴포넌트 테스트 미작성(`web/vite.config.ts:10` 의 `include:["src/**/*.test.ts"]` 때문에 조용히 0개로 수집된다).
- 다음 역할이 조심할 것: 새 스펙 3개는 `bash scripts/local-e2e.sh`(실 dist 임베드·전용 DB·실서버·실제 브라우저)에서만 돈다 — vitest 는 98개 그대로다. 「폐기 실패」 스펙은 실제 키 1개를 API 로 만들고 꼬리에서 성공 폐기까지 하므로 DB 에 폐기된 키 행 1개를 남긴다(활성 한도 10에 영향 없음). `confirm()` 때문에 `page.on("dialog", d => d.accept())` 가 필수다(Playwright 기본은 dismiss). `web/e2e` 는 여전히 `npm run lint` 의 타입 검사 대상이 아니다.
- [러너 23:15] brief accepted — 채택 — 지정한 파일 2개·행 번호(51·56-59·80·81·94-98·122-125·223·247)·수용 기준 1~5 가 지금 코드와 정확히 맞았고 프로덕�
- [러너 23:15] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 프로덕션 diff 1파일(`KeysPage.tsx` policy→nullable, 5개 독자 전부 null 처리, `revoke` 를 `rotate` 꼴 try/catch), `npm run lint`(tsc -b) 무출력 통과, `git status` 깨끗·diff 가 의도한 2파일, `test(` 20→23(원장 24 passed·358행 for 루프 설명과 일치), `revokeAPIKey`(keys.go:189-203) grace_until=NULL 이라 성공 폐기 후 버튼 disabled 가 실제로 성립, `api.ts:38-41` 이 error.message 를 넘겨 '활성 API 키가 없습니다' 단언 성립, Playwright workers:1·fullyParallel:false 라 3개 스펙 순차. 못 본 것: local-e2e 재실행(원장의 수정 전 2 failed → 수정 후 24 passed 출력을 신뢰).
- 보안·법무: 차단 없음. 새 엔드포인트·인증/인가 변경·권한 확대 0, 비밀값 노출 없음(스펙이 만든 원문 키는 로그·단언에 안 들어간다), 새 개인정보 수집·전송 0, 외부 약속 문구 없음. 관리자가 좁힌 Scope 를 넓게 말하지 않게 된 점은 오히려 소폭 개선.
- 승인이어도 남는 우려 ①: `policy===null` 이 '조회 중'과 '조회 실패'를 겸한다 — 첫 마운트 왕복 동안 info Alert 이 '불러오지 못해' 라는 아직 받지 못한 실패를 말한다(KeysPage.tsx:137-139). 이전엔 같은 창에서 틀린 숫자를 단정했으니 순 개선이라 결함으로 보지 않았다. 다음 회차 후보: 세 번째 상태/로딩 표시.
- 우려 ②: `visit-flow.spec.ts:71` 의 `expect(creates).toEqual([])` 는 공허하다(스펙이 「생성」을 누르지 않아 수정 전에도 [] — 원장 프로브의 POST 1건은 프로브가 눌렀기 때문). 실제 가드는 바로 위 `toBeDisabled()`. 스펙 전체는 수정 전 실제로 실패하므로 거절 사유가 아니지만 커버리지로 취급 금지.
- 릴리즈 노트 주의: 새 스펙 3개는 local-e2e/CI e2e 잡 전용이고 vitest 는 98개 그대로, `web/e2e` 는 여전히 `tsc -b` 밖이다. 「폐기 실패」 스펙은 e2e DB 에 폐기된 키 행 1개를 남긴다(활성 한도 무영향).
- [러너 23:20] review approved — 리뷰 승인 (risk=low)
- [러너 23:20] pr created — https://github.com/hkjang/visitflow/pull/38
- [러너 23:25] ci passed — 검사 2개 모두 success
- [러너 23:26] merge done — 625264b
- [러너 23:37] release published — v2.8.18
- [러너 23:38] assets verified — v2.8.18 자산 1개 (이전 v2.8.17: 1)
