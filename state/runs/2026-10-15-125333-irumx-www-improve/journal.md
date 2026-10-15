# 회차 노트 2026-10-15-125333-irumx-www-improve — irumx-www
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:53] base pinned — main@94eb01f
- [러너 12:53] autonomy release — 

## 정찰 노트
- 지난 두 회차가 고친 `pretest` 는 베이스 94eb01f 에 **없다**(Read 로 확인) — 그것만 다시 올리면 세 번째 같은 실패다. 그래서 남은 원인 ②(npm 이 루트까지 상위 `node_modules/.bin` 을 원래 PATH 앞에 넣는다 — `@npmcli/run-script/lib/set-path.js:27-37` 을 직접 읽어 확정, 홈의 `node` 패키지는 20.19.2)를 **같은 파일 안에서 함께** 고치는 쪽을 골랐다. 다른 후보(의존성 선언·ci.yml·inquiry·사이트맵)는 전부 `npm run build`/`npm test` 로 검증하는데 그게 지금 작업 트리에서 red 라 증명 자체가 불가능해 제쳤다.
- 확신 없는 곳: 러너의 `npm` 이 v22 node 로 돌아 `npm_node_execpath` 가 v22 를 가리킨다는 가정(`npm exec` 가 이 세션에서 차단돼 못 읽음 — 과제서에 미확인으로 명시). 이게 깨지면 처방이 무력하다. `scripts/serve-static.mjs`·`verify-build.mjs`·`inquiry.ts`·`worker/` 는 이번에 열지 않았다.
- 구현자가 조심할 것: 중첩 `npm run` 이 PATH 를 다시 덮으므로 교정 조각을 `build`·`check`·`test:build`·`pretest`·`test` **전부**에 붙여야 한다. 검증은 반드시 `/home/hkjang` 아래 작업 트리에서 — `/tmp` 클론은 고치기 전에도 통과해 증거가 못 된다. 가정이 깨지면 차선(serve-static 하드닝, npm 없이 직접 spawn 해 검증 가능)으로 전환하라.
- [러너 12:59] scout done — 수정 과제 — `npm test --silent` 가 **러너 작업 트리에서** 통과하게 고침(npm 이 PATH 에 끼워 넣는 상위 `node_mod

## 구현 노트
- 바꾼 것: `package.json`(각 스크립트 앞 PATH 교정 + `pretest`), 새 `scripts/pin-engine-node.mjs`(`node_modules/.bin/node` → `npm_node_execpath` 심링크), `tests/pin-engine-node.test.mjs`, README 2곳. 커밋 1355643.
- **과제서가 놓친 원인을 하나 더 고쳤다**: PATH 교정만으로는 부족하다 — 중첩된 `npx` 가 PATH 를 다시 덮고, 그 `npx` 는 손댈 수 없는 `playwright.config.ts:24`(`npx wrangler dev`)와 `scripts/deploy.mjs:32` 에 있다. 교정만 넣은 상태의 `npm test` 가 똑같은 Wrangler v20 메시지로 exit 1 인 것을 실측했다. 그래서 cwd 의 `.bin`(npm 이 넣는 경로 중 언제나 맨 앞)에 node 를 고정하는 겹을 더했다.
- **확신 없는 곳**: (1) `node_modules/.bin/node` 심링크는 "조금 영악한" 처방이다 — 깨끗한 환경에서는 불필요하고(그래도 해롭지 않다: 가리키는 대상이 곧 npm 을 돌리는 node 라 버전을 낮추는 일이 없다), Windows 기본 셸에서는 `export`·`${v%/*}` 와 함께 깨진다(README 에 sh 전제를 적었고 `script-shell` 은 넣지 않았다). (2) `deploy` 는 `npm run build` 를 거쳐 고정이 되므로 `npx wrangler deploy` 도 함께 구제되지만, **실제 배포는 돌려 보지 않았다**. (3) `worker/`·`src/`·`verify-build.mjs`·`serve-static.mjs` 는 이번에 열지 않았다.
- 일부러 하지 않은 것: `playwright.config.ts` 의 `npx wrangler` → 직접 경로 치환(그게 더 단순하지만 과제서가 금지한 파일이고 기준 4 가 diff 공백을 요구). `engines` 하향·`test.skip`·타임아웃 변경 0건. 홈의 node 패키지 삭제(사용자 소유).
- 다음 역할이 조심할 것: `tests/pin-engine-node.test.mjs` 는 `node --test` 로 돌린다 — `npm test`(playwright)는 각 project 의 `testMatch` 가 특정 spec 만 잡으므로 이 파일을 수집하지 않는다(게이트에 넣는 건 보류 아이디어로 적었다). 그 시험의 3번은 `npx` 를 실제로 띄우므로 오염되지 않은 기계에서는 고치기 전에도 통과한다(오염된 기계에서만 빨개진다). 포트 8820~8823 은 여전히 고정이다.
- [러너 13:11] brief accepted — 채택 — 지목한 두 원인과 수용 기준 5개를 그대로 충족했고, "미확인" 이라 밝힌 가정(러너의 npm 이 v22 로 돈다 → `npm_node
- [러너 13:12] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 판정 approve(low). 구현자가 "확신 없다" 고 적은 자리를 먼저 쳤다: dist 3벌 + `node_modules/.bin/node` 를 지운 상태에서 작업 트리 `npm test --silent` → exit 0 · 77 passed · 4 skipped(1m28s), 반대로 심링크를 치우고 `/home/hkjang/node_modules/.bin` 을 PATH 앞에 두면 wrangler·astro 가 v20.19.2 로 exit 1 — 증상이 정확히 일치한다. `node --test tests/pin-engine-node.test.mjs` 3/3 통과.
- 못 본 것: 실제 운영 배포(`npm run deploy` → `npx wrangler deploy`), Windows 셸, worker/·src/·verify-build.mjs·serve-static.mjs(이번 diff 밖).
- 승인이어도 남는 우려 ①: `export PATH="${npm_node_execpath%/*}:$PATH"` 가 node 디렉터리를 **저장소의 node_modules/.bin 보다 앞**에 둔다. 이 기계엔 충돌 바이너리가 없지만, astro·wrangler·playwright 가 전역 설치된 기계에선 고정 판본 대신 전역 판본이 돈다.
- 우려 ②: `tests/pin-engine-node.test.mjs` 는 playwright testMatch(`*.spec`)에 안 걸리고 CI 도 없어 **어느 게이트도 돌리지 않는다** — 다음 회차에서 `test:unit` 또는 pretest 끝에 묶을 것. 우려 ③: pretest 로 `npm test` 가 28초 → 약 89초(README 에 명시됨).
- 보안·법무 차단 없음: 인증·인가·비밀값·개인정보 경로 무변경, 새 의존성 0, 생성물은 전부 gitignore 안이라 커밋 1건 revert + `rm node_modules/.bin/node` 로 완전히 되돌아온다.
- [러너 13:17] review approved — 리뷰 승인 (risk=low)
- [러너 13:17] pr created — https://github.com/hkjang/irumx-www/pull/6
- [러너 13:22] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
