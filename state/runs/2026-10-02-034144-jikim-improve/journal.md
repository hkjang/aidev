# 회차 노트 2026-10-02-034144-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:41] base pinned — main@eeca591
- [러너 03:41] autonomy release — 

## 정찰 노트
- 우선 과제가 배정돼 새로 고르지 않았다. 다만 지난 회차가 남긴 원인 설명("jsdom/undici 가 Node 24 를 요구")은 lockfile 과 어긋난다 — jsdom 30.0.1 engines 는 `^22.22.2 || ^24.15.0 || >=26.0.0`, undici 8.10.1 은 `>=22.19.0` 로 로컬 v22.23.1 을 둘 다 만족하고, lockfile 은 v0.2.22 이후 버전 문자열 말고 바뀐 적이 없는데 2026-09-29 에는 같은 lockfile 로 vitest 59개가 통과했다. 그래서 과제서를 "원인 단정" 이 아니라 "0단계 재현 → 분기 A/B/C 판정" 으로 썼다.
- **확신 없는 곳(과제서에서 추측):** 실패를 직접 재현하지 못했다. 이 세션에서 `npm ci` 와 `node -e` 가 승인 거부로 막혔고 `web/node_modules` 는 비어 있다. 증상 문자열(`webidl.util.markAsUncloneable is not a function`)은 지난 회차 원장을 인용한 것이지 내가 본 것이 아니다. 1단계의 수정 형태(jsdom 핀 / undici overrides)도 가설이다.
- **확인한 것:** nvm 에 Node 24 가 **없다**(20.20.2 / 22.23.1 / 23.11.1 / 25.0.0). "24 로 올려 통과시킨다" 는 이 머신에서 불가능하니 구현자가 그 길로 새지 않게 과제서에 못박았다. `web/package.json` 에 `engines` 없음, `.nvmrc` 없음, `scripts/verify.sh:48-50` 은 PATH 의 아무 node 로나 도는 것도 직접 확인했다.
- **구현자가 조심할 것:** 워크플로 `node-version: 24` 를 낮추거나 테스트를 skip/exclude 해서 통과시키면 반려다. `package-lock.json` 을 손으로 고치지 말고 npm 이 쓰게 한 뒤 `npm ci` 가 깨끗이 도는지 재확인할 것. 0단계가 분기 B(재현 안 됨)로 끝나면 1단계를 억지로 하지 말고 2단계만 하고 journal 에 설치 출력을 남길 것 — 그것만으로도 독립적으로 가치가 있다.
- 프로필은 0일 전 것이고 지금 코드와 어긋나는 곳을 못 찾아 새로 쓰지 않았다(위의 Node/engines 사실만 이 노트에 추가).
- [러너 03:48] scout done — 프런트 검증 `npm --prefix web test` 가 이 머신에서 jsdom 환경 테스트를 시작조차 못 하는 원인을 재현해 좁게 �

## 0단계 재현 출력 (실제 명령, 전문 요약)

```
$ node -v
v22.23.1
$ npm --prefix web ci --no-audit --no-fund
added 336 packages in 10s
$ npm --prefix web test -- --maxWorkers=1
Caused by: TypeError: webidl.util.markAsUncloneable is not a function
 ❯ new CacheStorage node_modules/undici/lib/web/cache/cachestorage.js:20:17
 ❯ Object.<anonymous> node_modules/undici/index.js:179:25
 ❯ Object.<anonymous> node_modules/jsdom/lib/api.js:12:33
 Test Files  1 passed (1)
      Tests  15 passed (15)
     Errors  8 errors
```
→ 분기 A 재현. 그러나 원인 가설은 전부 틀렸다. 측정으로 좁힌 결과:

```
$ node -p "require('./node_modules/undici/package.json').version"   # web/
8.10.1            (lockfile 과 일치)
$ node -p "require('./node_modules/jsdom/package.json').version"
30.0.1            (lockfile 과 일치 → 설치 오염 아님)
$ node -p "typeof require('node:worker_threads').markAsUncloneable"
function          (셸 런타임에는 있다 → 셸 Node 문제 아님)
$ node node_modules/vitest/vitest.mjs run --maxWorkers=1   # 셸 Node 로 직접
 Test Files  9 passed (9)
      Tests  59 passed (59)          ← 저장소·lockfile 은 멀쩡하다
```

결정적 측정 — vitest 프로세스를 `NODE_OPTIONS=--require` 프로브로 직접 들여다봄:

```
[PROBE] ver=v20.19.2 execPath=/home/hkjang/node_modules/node/bin/node
[PROBE] PATH=
/home/hkjang/.cache/auto-improve-wt/jikim/web/node_modules/.bin
/home/hkjang/.cache/auto-improve-wt/jikim/node_modules/.bin
/home/hkjang/.cache/auto-improve-wt/node_modules/.bin
/home/hkjang/.cache/node_modules/.bin
/home/hkjang/node_modules/.bin          ← 여기에 node -> ../node/bin/node (v20.19.2)
[PROBE] npm_node_execpath=/home/hkjang/.nvm/versions/node/v22.23.1/bin/node
```

증명된 원인: npm 은 lifecycle 스크립트의 PATH 앞에 상위 디렉터리의 `node_modules/.bin` 을
모두 붙인다. `~/node_modules/.bin/node` 심링크가 nvm 경로보다 앞서므로 vitest 의
`#!/usr/bin/env node` 셔뱅이 **Node 20.19.2** 로 해석된다. Node 20 에는
`worker_threads.markAsUncloneable`(v22.10 추가)이 없어 undici 8.10.1 의
`lib/web/webidl/index.js:161` 이 `undefined` 를 넣고, `jsdom/lib/api.js:12` 의
`require("undici")` → `undici/index.js:179` 의 `new CacheStorage` 가 터진다. 스택의 `179`
는 8.10.1 고유(7.24.6 은 177)여서 로컬 사본이 맞음을 확정했다. jsdom 환경 9개 파일이
전부 시작조차 못 하고 node 환경인 `vite-proxy.test.ts` 1개만 통과한 것(15/59)까지
이 원인 하나로 전부 설명된다.

## 구현 노트

- 무엇을 왜: 원인이 의존성이 아니라 "npm 이 vitest 를 어느 Node 로 돌리는가"였으므로
  의존성은 한 줄도 바꾸지 않았다. `web/package.json` 에 `engines.node ">=22.22.2"`(jsdom
  30.0.1 의 22 계열 하한) 선언 + `test`/`test:watch` 가 `npm_node_execpath`(npm 을 실행한
  인터프리터)로 vitest 를 직접 호출, `web/vitest.config.ts` 에 테스트를 실제로 돌리는
  프로세스의 하한 확인, `scripts/verify.sh` 의 `npm ci` 직전 선행 검사, `.nvmrc`=`24`.
- 확신 없는 곳 / 검증 못 한 것: ① `npm test` 는 CI·릴리즈 워크플로가 부르는 명령이다.
  `${npm_node_execpath:-node}` 는 POSIX sh 에서만 전개되므로 **Windows 네이티브 npm(cmd)
  에서는 깨진다** — 이 저장소는 `scripts/*.sh`·Makefile·Dockerfile 로 이미 POSIX 전제지만
  Windows 개발자는 확인하지 못했다. ② CI(Node 24)에서는 `npm_node_execpath` 가 setup-node
  의 Node 를 가리켜 동작이 동일할 것으로 판단했으나 **GitHub Actions 실행은 이 환경에서
  확인할 수 없다**(`gh` 미인증). 릴리즈 경로를 건드리는 변경이므로 비평가가 여기를 먼저 볼 것.
  ③ `.nvmrc`=24 와 `engines`=22.22.2 가 다르다 — 문서·CI·Dockerfile 이 말하는 24 를 낮추지
  않고 `engines` 에는 의존성이 실제로 요구하는 하한을 적었다(이유는 CONTRIBUTING.md 에 기록).
- 일부러 하지 않은 것: (a) jsdom 핀·undici `overrides` — 증명된 원인에 대한 잘못된 수정이고
  멀쩡한 의존성의 하향이다. (b) `.github/workflows/*.yml` 과 `node-version: 24` — 과제서 금지.
  (c) `web/package-lock.json` — `npm install --package-lock-only` 가 `engines` 를 넣는 대신
  이 npm(10.9.8)이 optional 패키지 10곳의 `libc` 필드를 **지워버려서** 되돌렸다. lockfile 은
  바이트 단위로 그대로이고, `engines` 를 넣은 상태에서 `npm ci` 가 exit 0 인 것을 확인했다.
  (d) `~/node_modules/.bin/node` 심링크 제거 — 저장소 밖(사용자 홈)이고 되돌릴 수 없는 전역
  환경 변경은 반려 사유다. 그래서 `npx vitest` 처럼 npm 스크립트를 거치지 않는 호출은 여전히
  Node 20 으로 돈다(ideas.json 에 후보로 남김).
- 다음 역할이 조심할 것: 이 저장소를 `~/node_modules` 가 있는 이 머신에서 검증할 때
  `npx vitest` 로 부르면 아직 Node 20 으로 돌아 깨진다 — `npm --prefix web test` 또는
  `bash scripts/verify.sh` 로 부를 것. `nvm exec 20.20.2 bash scripts/verify.sh` 는
  **의도적으로** exit 1 이다(수용 기준 4). `web/node_modules` 는 커밋되지 않았고
  작업 트리에는 `.nvmrc` 1개만 새로 추가됐다.
- [러너 04:07] brief accepted — 채택 — 0단계 재현 절차와 분기 판정 구조가 정확히 제 역할을 했고(분기 A 재현), 2단계는 지시대로 구현했다. 다만 1단�
- [러너 04:07] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: `~/node_modules/.bin/node`=v20.19.2 실존 → `npm exec -- vitest run`(수정 전 경로)이 이제 vitest.config.ts:18 의 읽을 수 있는 한 줄로 멈추는 것을 직접 재현. 수정 후 `npm --prefix web test` 9파일 59테스트 통과, lint·tsc -b·vite build 통과, lockfile diff 없음, 버전 비교 경계값(20.19.2/22.22.1 거절, 22.22.2/22.23.1/24/25/100 통과)을 shell·JS 양쪽에서 확인. CI·release 둘 다 verify.sh 를 부르는 것도 확인.
- 못 봄: GitHub Actions 실제 실행(gh 미인증), Windows 네이티브 npm, macOS `sort -V`, 전체 `bash scripts/verify.sh`(프런트·변경 표면만 개별 실행).
- 승인이어도 남는 우려: ① `"${npm_node_execpath:-node}"` 는 POSIX sh 전개 — Windows 네이티브 npm(cmd)에서 `npm test` 가 깨진다(문서상 Windows 지원 선언은 없음). ② `engines ">=22.22.2"` 가 jsdom 의 `^22.22.2 || ^24.15.0 || >=26.0.0` 보다 넓어 Node 23.x·24.0~24.14 가 가드를 통과한다. ③ `./node_modules/vitest/vitest.mjs` 하드코딩은 vitest bin 이름 변경에 깨진다. ④ 이 회귀를 못박는 자동 테스트는 없고 CI 는 되돌려져도 green 이니 vitest.config.ts 가드를 지우지 말 것 — 릴리즈 노트에 ①을 적을 가치가 있다.
- 거절 아님. 차단 부서 소견 없음(인증·비밀값·개인정보·의존성 무변경).
- [러너 04:13] review approved — 리뷰 승인 (risk=low)
- [러너 04:13] pr created — https://github.com/hkjang/jikim/pull/49
- [러너 04:17] ci passed — 검사 2개 모두 success
- [러너 04:17] merge done — 7a13e85
- [러너 04:28] release published — v0.2.27
- [러너 04:32] assets verified — v0.2.27 자산 2개 (이전 v0.2.26: 2)
