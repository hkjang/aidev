# 회차 노트 2026-10-07-234305-irumx-www-improve — irumx-www
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:43] base pinned — main@80e35fc
- [러너 23:43] autonomy release — 

## 정찰 노트
- main@80e35fc 에 앞선 네 회차의 성공 결과가 **하나도 없다**(`.github/` 없음, `free-ports.mjs`·`verify-deps.mjs` 없음, `pretest` 없음 — 직접 확인). 그래서 그 셋(npm test 독립 실행·harfbuzzjs 선언·ci.yml)은 전부 제쳤다: npm test 수정은 `auto/2026-10-07-2258` 이 origin 에 푸시되어 review-pending 이라 세 번째 중복이 되고, harfbuzzjs 는 똑같은 과제가 이미 verify-failed 였다(막힌 원인이 npm test 환경이고 아직 미해결).
- 그래서 **`package.json`·`playwright.config.ts` 를 건드리지 않는 파일 1개 과제**를 골랐다 — `scripts/serve-static.mjs`. 2258 이 머지되든 안 되든 충돌하지 않고, 순수 node 로 검증되므로 `npm test` 환경 문제에 걸리지 않는다.
- 추측으로 적은 것: 404 크래시의 **런타임 재현을 못 했다**. 이 세션에서 `node`/`curl`/`npm ci` 실행 권한이 거부됐고 워크트리에 `node_modules` 가 없다. 근거는 `serve-static.mjs:53` 의 try/catch 없는 `await readFile` 독해뿐이다 — 그래서 수용 기준 1을 "먼저 재현해 증명, 재현 안 되면 중단하고 보고" 로 못 박았다.
- 구현자가 조심할 것: `404.html` 이 **있을 때**의 응답은 바이트 단위로 지금과 같아야 한다(`tests/site.spec.ts` 의 404 단정). 폴백은 없을 때만. 기동 점검은 폴더·`index.html` 두 가지로만 — 늘리면 범위가 터진다.
- 차선(verify-build 서비스 목록 드리프트 감지)은 값을 4→3 으로 내렸다: `services.ts` slug 11개와 두 손목록이 **지금 정확히 일치**하고, 사이트맵은 어차피 dist 를 훑어 자동 생성되므로 드리프트는 사이트 파손이 아니라 점검 약화다.
- [러너 23:48] scout done — 테스트용 정적 서버(`scripts/serve-static.mjs`)가 요청 한 번에 죽지 않게 하고, 제공할 폴더가 준비되지 않았으

## 구현 노트
- `scripts/serve-static.mjs` 한 파일. 404 분기를 "읽고 나서 헤더 쓰기" 로 바꿔 `404.html` 없을 때 404+평문으로 답하게, `listen` 전 `base`·`index.html` 점검 후 없으면 한 줄+exit 1, 탈출 가드를 `base + sep` 로, 핸들러 전체에 catch 를 둬 남은 `readFile` 실패도 그 요청만 500 으로 끝내게 했다.
- **과제서 근거 한 군데가 틀렸다(정정).** 기준 5의 `/../dist-draft/x` 는 탈출하지 못한다 — WHATWG URL 파서가 dot segment 를 먼저 정규화한다(`%2e%2e` 도 같음). 실제로 통하는 벡터는 **`..%2f`**: `%2f` 가 URL 정규화를 살아남고 뒤이은 `decodeURIComponent` 가 진짜 슬래시로 되돌려 `join` 이 형제 폴더로 해석한다. 결함·처방은 유효하며 이 벡터로 재현·검증했다.
- **확신 없는 곳**: ① 과제서 범위 밖인 **핸들러 전체 catch** 를 내 판단으로 더했다("요청 한 번에 죽지 않게" 가 과제 제목이고 `res.end(await readFile(hit))` 에 같은 구조적 위험이 남아서). 조용히 삼키지 않고 URL+사유를 stderr 로 찍는다. 과하다고 보면 이 블록만 빼도 수용 기준 1~5 는 그대로 충족된다. ② `npm test` 를 **사설 netns**(`unshare -rn`)에서 돌렸다 — 8788·8789 가 다른 프로젝트에 점유돼 있고(`lsof`: `node-Main 2806847`→:8788 'your', `2814318`→:8789 '옆자리 인사이트') `playwright.config.ts` 가 포트를 하드코딩해 환경변수 우회가 없다. 그래서 외부 연결이 없고 Turnstile 3건이 실패한다(71통과·3실패·4건너뜀). **내 변경을 되돌린 원본으로 같은 3건이 똑같이 실패함을 확인**해 선재 실패로 증명했다. 연결 있는 실행에서의 75통과는 이 세션에서 확인하지 못했다.
- **검증 못 한 것**: 이번에 고친 **404 폴백 분기는 저장소 안 테스트가 지나지 않는다.** `serve-static` 은 draft(8789)·email(8791) 프로젝트만 쓰고 그 둘의 `testMatch` 는 `contact.spec`·`contact-email.spec` 이라 404 를 안 건드린다. `tests/site.spec.ts` 의 「없는 주소는 404」 는 baseURL 8788 = **wrangler dev** 라서 Workers 정적 자산을 보는 것이고 `serve-static` 이 아니다. 폴백 증거는 `/tmp/ss-harness/check.mjs`(실제 프로세스 spawn + 생소켓, 주입 없음) 뿐이다 — `ideas.json` 에 항목으로 남겼다. 반면 **기동 점검과 200 경로·경로 가드는 `npm test` 의 draft·email 서버가 실제로 통과한다**(프로덕션 배선).
- **일부러 하지 않은 것**: `package.json`·`playwright.config.ts` 무수정(2258 review-pending 과 충돌 회피), 하네스를 저장소에 커밋하지 않음(단위 러너가 없어 넣으면 `package.json` 을 건드려야 함), 기동 점검을 폴더·`index.html` 둘로만 제한, `.github/` 미생성.
- **다음 역할이 조심할 것**: 이 기계에서 `npm test` 를 그냥 돌리면 8788·8789 의 **남의 서버**를 테스트해 엉뚱하게 실패한다 — `unshare -rn sh -c 'ip link set lo up && npm test'` 로 돌릴 것. `npm ci`(6초)·`npx playwright install chromium`(114MB) 이 먼저 필요하다. 하네스 재실행: `node /tmp/ss-harness/check.mjs <저장소>/scripts/serve-static.mjs`(포트를 스스로 받아 쓴다 — 고정 포트를 쓰면 안 된다). 커밋 49011a7, 변경 1파일(+43/-4).
- [러너 00:04] brief accepted — 채택 — 수용 기준 5개를 모두 충족했다. 다만 기준 5의 근거가 한 군데 틀렸다: `/../dist-draft/x` 는 WHATWG URL 파서가 dot segment
- [러너 00:08] verify failed — 실패한 검증: npm test --silent (exit 1)
