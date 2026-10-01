# 회차 노트 2026-10-01-194215-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:42] base pinned — main@887bd84
- [러너 19:42] autonomy release — 

## 정찰 노트
- 우선 과제가 배정돼 새 아이디어 대신 `cd web && npm test` 실패를 골랐다. 차선(compare_suppliers 설명 좁히기)은 과제서 맨 끝에 남겼다.
- 지난 회차가 원인을 이미 증명했으므로(워커 안 execPath=v20) 이번엔 **repo 쪽 해법**만 찾았다: npm v22.23.1 소스에서 `set-path.js:22-28`(모든 상위 node_modules/.bin 을 PATH 앞에 unshift)과 `set-envs.js:105`(`env.NODE=env.npm_node_execpath=config.execPath`)를 직접 읽어, 테스트 러너를 npm 이 쓰는 node 로 명시 실행하면 환경을 손대지 않고 통과할 수 있다고 판단했다. 느슨하게 만드는 것이 아니다(skip·exclude·의존성 하향 0).
- 확신 없는 곳(추측 아님, **미확인**): 이 정찰 세션은 `npm` 실행 권한이 없어 `npm ci`/`npm test`/`npm run env` 를 한 번도 돌리지 못했다. 그래서 ① 실제 실행에서 `npm_node_execpath` 가 v22 를 가리키는지 ② vitest bin 해석 경로 — 두 가지는 구현자가 1·2단계에서 **먼저** 확인해야 한다. 어긋나면 과제서의 차선 경로(PATH 에서 .bin 항목 제거)로 갈 것.
- 조심할 것: lockfile·의존성 금지(지난 회차에 기각된 전제), `.github/workflows` 금지, 머신의 `/home/hkjang/node_modules/node` 삭제 금지(지우면 수정의 효과를 증명할 수 없다). 수용 기준은 「한 줄로 실패」가 아니라 **exit 0** 이다 — fe87155 는 거기서 멈춰 같은 verify-failed 로 끝났다.
- [러너 19:49] scout done — `cd web && npm test` 가 npm 이 PATH 앞에 끼워 넣은 **상위 디렉터리의 node@20** 위에서 돌지 않게, 테스트 러너를 �

## 구현 노트
- `web/scripts/run-vitest.mjs`(신규) + `web/package.json`(`test`/`test:watch`/`engines`) — commit 78bcc83. npm 이 상위 `node_modules/.bin` 을 PATH 앞에 unshift 해 `vitest` 의 `#!/usr/bin/env node` 가 node@20 을 집던 것을, `npm_node_execpath` 로 인터프리터를 명시해 끊었다. 상위 오염을 치우지 않은 채 `cd web && npm test` 가 exit 0 / 22 passed / 98 passed 다.
- **확신 없는 곳·검증 못 한 것**: ① `npm_node_execpath` 가 **빈 환경**(npm 아닌 러너가 직접 스크립트를 부르는 경우)에서는 `process.execPath` 로 떨어진다 — 이 경로는 `node web/scripts/run-vitest.mjs run` 으로만 확인했고, node@20 이 래퍼를 직접 부르는 조합은 테스트하지 않았다(그 경우 가드가 터지는 것이 의도된 동작이다). ② CI(GitHub Actions)에서 실제로 도는 것은 확인하지 못했다 — 푸시 권한이 없다. 다만 `ci.yml` 은 setup-node 22 + `npm test`(`--ignore-scripts` 없음)이고 래퍼는 PATH 에 의존하지 않으므로 통과할 것으로 본다. ③ `cwd: packageRoot` 를 고정했으므로 **저장소 루트에서 상대 경로 인자**를 넘기면(`node web/scripts/run-vitest.mjs run src/x.test.ts` 를 루트에서) 그 상대 경로가 web/ 기준으로 해석된다. `npm test` 는 이미 cwd 가 web/ 이라 영향 없다.
- 일부러 하지 않은 것: `npm run build`·`eslint` 는 같은 PATH 오염 위에서 돌지만(지금은 node 20 에서도 우연히 통과) 범위 밖으로 뒀다 — 파일 수·릴리즈 위험이 늘고, 과제서가 테스트 러너 한 자리로 못 박았다. 의존성·`package-lock.json`·`vitest.config.ts`·`web/src/**`·`.github/workflows/**`·`.nvmrc` 무변경. 머신의 `/home/hkjang/node_modules/node` 는 지우지 않았다(지우면 이 수정의 효과를 증명할 수 없다). 원인 증명에 쓴 임시 프로브는 없다 — 실험을 **기존 bin 호출**로 했으므로 커밋에 남길 파일이 애초에 없었다.
- 다음 역할이 조심할 것: Go 는 한 줄도 안 바꿨고 `go test` 는 **DSN 미설정이라 DB 통합 테스트가 skip** 됐다(httpapi 1.516s) — 회귀 확인용일 뿐 DB 검증이 아니다. 가드 분기를 재현하려면 가짜 env 가 아니라 `/home/hkjang/node_modules/node/bin/node $(which npm) test` 로 진짜 npm 을 node@20 에 태울 것. `npm test -- --coverage` 는 `web/coverage/`(gitignore 됨)를 만들고, `eslint.config.js` 의 ignores 가 `dist` 만 적어 그 뒤 `npm run lint` 가 생성물에서 3 warnings 를 낸다 — 다음 회차 후보로 ideas.json 에 올렸다.
- [러너 20:02] brief accepted — 채택 — 과제서의 전제가 실행에서 그대로 맞았다. 1단계에서 `npm run env` 로 확인한 `npm_node_execpath`/`NODE` 는 둘 다 nvm `v22.23
- [러너 20:02] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 실제로 돌려 확인했다: ① 수정 전 증상 재현 — `PATH=/home/hkjang/node_modules/.bin:$PATH ./node_modules/.bin/vitest run` → `Errors 22`, exit 1 ② 수정 후 `cd web && npm test` → exit 0 / 22 files / 98 tests ③ 가드 분기 — `/home/hkjang/node_modules/node/bin/node $(which npm) test` → `web tests need Node >= 22; resolved … (v20.19.2)` 한 줄, exit 1 ④ 인자 통과 — `npm test -- --reporter=dot src/session.test.ts` → 1 passed, exit 0 ⑤ `npx eslint scripts/run-vitest.mjs` exit 0 ⑥ 트리 클린, 변경은 두 파일뿐(lockfile·의존성·workflow·web/src 무변경).
- 못 본 것: GitHub Actions 실제 실행(권한 없음 — 다만 ci.yml 은 setup-node 22 + `npm test` 이고 `.npmrc` 가 없어 engine-strict 가 아니므로 floor 는 만족, 래퍼는 PATH 비의존), Go DB 통합 테스트(DSN 미설정으로 skip — 이번 변경이 Go 를 건드리지 않아 무관).
- 승인이어도 남는 우려: ① `resolveVitestBin` 의 맨 `catch` 가 모든 resolve 실패를 "vitest is not installed" 로 보고한다 — 장래 vitest 가 `./package.json` export 를 끊으면 메시지가 오진을 유도한다 ② `resolveMinimumMajor` 는 range 의 **첫 숫자**를 floor 로 쓴다(`">=22"` 는 맞지만 `"^24 || ^22"` 처럼 쓰면 floor 가 24 로 올라간다) ③ `engines` 는 engine-strict 가 없어 경고일 뿐이고 `npm run build`·`eslint` 는 여전히 오염된 PATH 위에서 돈다(구현자가 범위 밖으로 선언한 자리 — 다음 회차 후보).
- 릴리즈 노트에 넣을 것: `npm test`/`test:watch` 가 Node >= 22 를 요구하며, 더 낮은 인터프리터에서는 스택 100줄 대신 한 줄로 즉시 실패한다. 개인정보·인증·권한·마이그레이션·비밀값 경로 무변경이라 보안·법무 차단 사유 없음.
- [러너 20:05] review approved — 리뷰 승인 (risk=low)
- [러너 20:05] pr created — https://github.com/hkjang/Vendra/pull/137
- [러너 20:07] ci passed — 검사 2개 모두 success
- [러너 20:07] merge done — 78bcc83
- [러너 20:18] release published — v0.7.66
- [러너 20:19] assets verified — v0.7.66 자산 1개 (이전 v0.7.65: 1)
