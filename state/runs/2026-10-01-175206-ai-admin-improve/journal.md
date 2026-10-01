# 회차 노트 2026-10-01-175206-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:52] base pinned — main@18c30f4
- [러너 17:52] autonomy release — 
- [러너 17:58] scout done — 웹 테스트가 Testing Library 기본 1초 `asyncUtilTimeout` 에 걸려 간헐 실패하는 것을 셋업에서 명시적 예산으로 �

## 구현 노트
- 무엇·왜: `web/src/test/setup.ts` 에 `configure({ asyncUtilTimeout: 5_000 })` 를 더했다. 저장소에 `configure()` 가 없어 모든 `findBy*`/`waitFor` 가 `@testing-library/dom` 기본값 1000ms 만 기다렸고(`node_modules/@testing-library/dom/dist/config.js:15`), `vite.config.ts` 의 `testTimeout: 15_000` 은 비동기 쿼리에 적용되지 않는다. `AdminInteractions.test.tsx` 의 `commonFetch` 는 각 응답을 150ms 지연시켜 이 결함을 부하와 무관하게 드러낸다. 프로덕션 파일 0개.
- 확신 없는 것: ① 지연 스텁은 `AdminInteractions.test.tsx` 에만 넣었다 — 나머지 17개 파일의 31개 비동기 호출 지점은 여전히 즉시 응답 스텁 아래서만 검증됐고, 5초가 그 파일들에도 충분한지는 **확인하지 못했다**(ideas.json 에 후속 과제로 적었다). ② 150ms·5초 값은 이 머신 실측(통과 2.65초대)에 맞춘 것이라 훨씬 느린 CI 러너에서 5초가 충분한지는 미확인이다. ③ `npm test` 출력의 `Could not parse CSS stylesheet` 12줄은 원래부터 있던 것이고 내가 건드리지 않았다.
- 일부러 하지 않은 것: `vite.config.ts` 의 `testTimeout`·`.github/workflows/*`·단정 3줄·`afterEach` 블록 순서는 손대지 않았다(순서를 바꾸면 17개 파일이 서로 오염된다). `it.skip`/`retry`/`test.fails` 로 통과시키지 않았다. `web/src` 앱 코드를 바꾸지 않았으므로 `internal/ui/dist` 재빌드는 필요 없다 — 다음에 앱 코드를 고치면 `make build` 로 dist 를 반드시 교체할 것.
- 다음 역할이 조심할 것: 검증 전에 `cd web && npm ci` 가 필요하다(워크트리에 `node_modules` 가 없었다). 이 변경은 DB 가 필요 없고 Go 테스트와 무관하다 — `go test` 는 돌리지 않았고(`make lint`·`go build ./...` 만 회귀 확인), 통합 테스트는 `TEST_POSTGRES_DSN` 이 없으면 조용히 SKIP 된다. 예산 변경의 인과는 `setup.ts` 의 `5_000` 을 `1_000` 으로 되돌려 같은 테스트를 돌리면 1229ms 에 FAIL 하는 것으로 언제든 재확인할 수 있다.
- [러너 18:12] brief accepted — 채택 — 진단(기본 1초 `asyncUtilTimeout`, `configure()` 0건, `testTimeout` 무관)·건드릴 파일 2개·지연 스텁으로 red 를 만드는 방식
- [러너 18:12] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: red/green 직접 재현(`setup.ts` 5_000→1_000 임시 셋업으로 `AdminInteractions.test.tsx:48` FAIL 1.29s, 5_000 PASS) · `npx vitest run` 81/81 PASS 3회 · `npm run build`(tsc -b, src 전체 타입체크) exit 0 · diff 2파일이 테스트 전용임 · 워크트리 clean 복원.
- 구현자 불확신 ②를 해소: `taskset -c 0,1 vitest run --maxWorkers=2` 로 2 vCPU 모사, 예산을 3_000·2_000 으로 낮춰도 81개 전부 PASS — 가장 긴 단일 비동기 대기는 2코어에서도 2초 미만이고 5초는 2.5배 이상 여유다. 불확신 ①도 같은 근거로 결함 아님(즉시 응답 스텁 파일들은 예산 변화에 둔감).
- 못 본 것: Go 테스트·통합 테스트·`make lint`(이번 변경과 무관해 생략) · 실제 GitHub 러너에서의 실행 · 개별 `waitFor` 호출의 내부 소요(총 소요로만 바운드).
- 승인이어도 남는 우려: ① 2코어에서 AdminInteractions 테스트 **총 소요 5089ms** 는 예산 5000ms 를 넘어 보이지만 단일 대기가 아니라 합계라 통과한다 — `setup.ts:9` 주석의 "최대 2.2초대" 가 단일 대기 기준임이 안 적혀 있어 다음 독자가 오독할 수 있다. ② 예산 1초→5초로 진짜 실패가 5초씩 매달리므로, 비동기 대기 여러 개가 함께 실패하는 테스트는 `testTimeout: 15_000` 에 먼저 걸려 DOM 차이 대신 타임아웃만 남길 수 있다 — 예산을 더 올리려면 `vite.config.ts` 의 testTimeout 도 함께 보라.
- 판정: approve / risk low / blocking 없음(새 엔드포인트·인가·비밀값·의존성·개인정보 신규 처리 0건).
- [러너 18:19] review approved — 리뷰 승인 (risk=low)
- [러너 18:19] pr created — https://github.com/hkjang/ai-admin/pull/36
- [러너 18:27] ci passed — 검사 2개 모두 success
- [러너 18:27] merge done — 429f282
- [러너 18:41] release published — v1.2.32
- [러너 18:43] assets verified — v1.2.32 자산 2개 (이전 v1.2.31: 2)
