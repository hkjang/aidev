# v0.76.4 — npm 은 이 저장소 안에 머물고, 시험은 돌아갈 수 있는 Node 에서 돕니다

이번 회차에 **제품은 한 줄도 바뀌지 않았습니다.** umm 을 쓰는 사람이 보는 화면, API, 데이터는
v0.76.3 과 똑같습니다. 달라진 것은 umm 을 **내려받아 시험을 돌려 보는 사람**이 만나던 두 가지입니다.
둘 다 "umm 이 고장났다" 처럼 보였고, 둘 다 umm 의 잘못이 아니었습니다.

```
    저장소 루트에서  npm test

            전                                  후
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━━━━━━━━━━
    npm error Missing script: "test"    19개 파일 215개 시험 통과
    (또는 집 디렉터리의 남의 스크립트)

    web/ 에서  npm test

            전                                  후
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━━━━━━━━━━
    TypeError:                          19개 파일 215개 시험 통과
      webidl.util.markAsUncloneable
      is not a function
    Test Files  no tests, 18 errors
```

## 루트의 npm 은 체크아웃 밖으로 걸어 나갔습니다

프런트엔드는 `web/` 에 있으므로 저장소 루트에는 `package.json` 이 없었습니다. npm 은 스크립트를
찾을 때 **부모 디렉터리로 계속 올라갑니다.** 그래서 루트에서 `npm test` 를 치면 npm 은 체크아웃을
지나쳐 조상 디렉터리에 우연히 놓인 `package.json` 에 묶였습니다 — 개발자 기계에서 그것은 보통
집 디렉터리입니다.

직접 찍어 확인한 모습입니다. 루트에서 물어본 이름과 의존성이 이 저장소의 것이 아니었습니다.

```
    $ npm pkg get name                  # 저장소 루트
    {}
    $ npm pkg get dependencies
    { "@hkjang/openpro": ..., "playwright": ... }   # 이 저장소가 선언하지 않는 것들
    $ npm --prefix web pkg get name
    "umm-web"
```

운이 좋으면 `Missing script: "test"` 로 끝납니다. 운이 나쁘면 **그 남의 파일이 같은 이름의
스크립트를 정의하고 있고, 그것이 대신 돌아갑니다.** 종료 코드만 보는 자리에서는 둘을 구별할 수
없습니다.

루트에 `private` `package.json` 하나와 의존성 0개 lockfile 을 두어 npm 을 이 저장소에 못 박았습니다.
스크립트는 CI 가 `working-directory: web` 으로 돌리는 것들을 Makefile 이 이미 쓰던 `npm --prefix web`
형태로 그대로 넘깁니다.

```json
"scripts": {
  "typecheck": "npm --prefix web run typecheck",
  "lint": "npm --prefix web run lint",
  "test": "npm --prefix web test",
  ...
}
```

`version` 필드는 **일부러 넣지 않았습니다.** `scripts/check-version.sh` 는 `VERSION` 을
`web/package.json` 과 경로로 지켜보는데, 같은 숫자가 여기 한 번 더 적히면 릴리즈가 깨질 자리가
하나 더 늘어납니다.

## npm 이 시험에 건네는 PATH 는 파일 시스템 뿌리까지 올라갑니다

루트에서 위임만 해도 **시험은 여전히 전부 실패했습니다.** 이쪽이 더 고약합니다.

npm 은 lifecycle 스크립트의 `PATH` 를 만들 때 패키지 디렉터리에서 위로 걸어 올라가며 **모든 조상의
`node_modules/.bin` 을 앞에 붙입니다** — 체크아웃을 넘어 파일 시스템 뿌리까지, 어디서 멈추라는
제한 없이(`@npmcli/run-script` 의 `set-path.js` 에 그렇게 적혀 있습니다). 클론보다 위의 아무
디렉터리에나 `node` 라는 **패키지**가 설치돼 있으면, 그것이 개발자가 고른 인터프리터를 가립니다.
`#!/usr/bin/env node` 로 시작하는 모든 것이 그 위에서 부팅합니다.

이 기계에서 실제로 그랬습니다.

```
    직접 실행한 node                      v22.23.1
    lifecycle 문맥의 node       /home/hkjang/node_modules/node/bin/node   v20.19.2
```

## 세 가지 실패는 어느 것도 Node 를 말하지 않았습니다

지원되지 않는 인터프리터에서 시험 입구는 서로 다른 세 방식으로 깨지고, **셋 다 Node 나 버전을
입에 올리지 않습니다.**

- `npm test` — vitest 가 jsdom 을 띄우고, jsdom 이 undici 를 읽고, undici 가 require 시점에
  `node:worker_threads` 에서 `markAsUncloneable` 을 꺼냅니다. 그 심볼은 **Node 22.10** 에
  들어왔으므로 그 아래에서는 시험 하나 돌기 전에 모든 워커가
  `TypeError: webidl.util.markAsUncloneable is not a function` 으로 죽고, 실행은 "no tests" 에
  종료 코드 1 을 답합니다
- `npm test` — **Node 25** 부터는 `localStorage` 전역이 플래그 없이 실립니다. `globalThis` 의
  접근자이고 `--localstorage-file` 을 요구하는 저장소를 돌려주는데, jsdom 이 설치하는 것보다
  **우선순위가 높습니다.** 그래서 웹 저장소를 만지는 시험들이 `localStorage.clear is not a function`
  으로 넘어집니다 — 25.9.0 에서 186개 중 58개. 이쪽은 발견하기 어려운 수준을 넘어 **이 제품에 대한
  58개의 실패 주장**으로 읽힙니다
- `npm run test:offline-queue` — `src/offline-queue.ts` 를 직접 import 해 타입 벗기기를 Node 에
  맡깁니다. 그것이 플래그 없이 되는 것은 **22.18** 부터이고, 그 아래에서는
  `ERR_UNKNOWN_FILE_EXTENSION ".ts"` 를 던집니다

## 하한이 아니라, 구멍이 난 범위입니다

위의 둘째 항목이 답을 정합니다. 아래로만 막으면 **위에서 들어오는 실패를 막지 못합니다.** 그래서
`web/package.json` 은 열린 하한이 아니라 **돌아가는 두 구간**을 적습니다.

```json
"engines": { "node": "^22.22.2 || ^24.15.0" }
```

23.x 가 빠진 것은 jsdom 이 그 줄을 제외하기 때문이고, 25 와 26 이 빠진 것은 위의
`localStorage` 때문입니다. 그 사이의 구멍은 **지어낸 것이 아니라 관찰된 실패가 깎아낸 모양**이고,
`web/scripts/node-range.test.mjs` 에 그 판정들이 한 줄씩 시험으로 남아 있습니다. 어떤 버전을
한쪽 목록에서 다른 쪽으로 옮기는 일은 시험이 어디서 도는지에 대한 주장이므로, 이 파일의 수정이
아니라 **그 인터프리터에서 실제로 돌려 본 결과**를 요구합니다.

## 못 찾으면 통과시키지 않고 거절합니다

`web/scripts/run-on-supported-node.mjs` 가 `test` · `test:watch` · `test:offline-queue` 를
감쌉니다. 순서대로 봅니다: 자기 프로세스의 인터프리터 → npm 자신이 돌고 있는 것
(`npm_node_execpath`, 개발자가 고른 그것) → **`node_modules` 아래가 아닌 `PATH` 항목**.

마지막 한 단계가 필요한 이유가 있습니다. `npm_node_execpath` 는 **한 홉만 살아남습니다.** 루트가
`npm --prefix web test` 로 위임하면, 그 중첩된 npm 자신이 가려진 PATH 에서 발견돼 가리는
인터프리터로 부팅하고, 그러고 나서 **그것을** `npm_node_execpath` 로 보고합니다. 이것도 직접
찍어 확인했습니다 — 바깥 `npm_node_execpath` 는 v22.23.1, 안쪽은 v20.19.2 였습니다.

```
    바깥 npm  npm_node_execpath = .../v22.23.1/bin/node
    안쪽 npm  npm_node_execpath = /home/hkjang/node_modules/node/bin/node   (v20.19.2)
```

`node_modules` 아래를 건너뛰는 것이 정확한 구분선입니다. 거기 들어 있는 인터프리터는 **어떤 패키지의
의존성**이지 개발자가 고른 툴체인이 아니고, npm 이 앞에 붙이는 것이 바로 그 항목들입니다. nvm,
시스템 패키지, CI 의 `setup-node`, Dockerfile 의 이미지 — 보통의 설치는 모두 `node_modules` 밖에
있으므로 여기서 찾아집니다.

그리고 아무것도 찾지 못하면 **거절합니다.** 지원 Node 를 쓸 수 없는 실행은 초록색을 보고하는 대신
0 이 아닌 값으로 끝나야 합니다. 절대 해서는 안 되는 한 가지는 그냥 돌린 뒤 **시험을 탓하는 것**입니다.

```
    run-on-supported-node: this package needs Node ^22.22.2 || ^24.15.0; found
    20.19.2 (/home/hkjang/node_modules/node/bin/node). Install a supported Node,
    or check whether a `node` package in a directory above this checkout is
    shadowing it on the PATH npm gives to scripts.
```

읽지 못하는 `engines.node` 범위도 같습니다. `web/scripts/node-range.mjs` 는 읽을 수 없는 범위에
아무것도 돌려주지 않고, 감싸는 쪽은 그것을 **어깨를 으쓱하는 대신 실패로** 다룹니다 — 잘못 읽은
범위는 이 장치가 돌려보내려는 바로 그 인터프리터들을 통과시킵니다.

## 달라지지 않는 것

**느슨해진 것은 하나도 없습니다.** GitHub 워크플로(`release.yml` · `ci.yml`), Makefile, Dockerfile,
`npm audit` 임계값은 손대지 않았습니다 — 먼저 읽어서 워크플로 쪽은 멀쩡함을 확인했습니다
(`release.yml` 에 npm 단계 0개, `ci.yml` 의 npm 단계는 모두 `working-directory: web`).
바뀐 프로덕션 코드는 **0개**입니다. 설정과 스크립트뿐입니다.

`vitest.config.ts` 의 `include` 에 `scripts/**/*.test.mjs` 가 더해진 것은 위 범위 판정들이 시험으로
남기 위해서입니다. `src` 의 시험들은 그대로입니다.

## 검증

지정된 실패를 먼저 base 에서 그대로 재현한 뒤 고쳤고, 고친 뒤에는 **되돌려서 같은 실패가 다시
나는 것**까지 확인했습니다.

```
✓ 루트 npm test --silent          EXIT=0   19개 파일 215개 시험
✓ web/ npm test                   EXIT=0   19개 파일 215개 시험
✓ 루트 설정 2개를 치우면          Missing script: "test"  EXIT=1   (첫 실패 복원)
✓ 래퍼 배선 전으로 되돌리면       markAsUncloneable TypeError, 18 errors  EXIT=1   (둘째 실패 복원)
✓ 시험 하나를 일부러 깨면         루트에서 1 failed  EXIT=1   (위임이 실패를 삼키지 않음)
✓ engines 가 읽을 수 없는 범위면  거절  EXIT=1
✓ 도달할 수 없는 하한이면         거절  EXIT=1
```

마지막 두 줄이 요점입니다. 이 장치의 실패 방향은 **거절**이어야 하고, 통과가 되어서는 안 됩니다.

이번 릴리즈를 검증하는 동안 래퍼가 실제로 적어 보낸 줄입니다 — 가리는 쪽을 보고, 지나가고,
무엇을 골랐는지 말합니다. 조용히 넘어가지 않습니다.

```
    run-on-supported-node: PATH gave Node 20.19.2
    (/home/hkjang/node_modules/node/bin/node), which engines.node
    ^22.22.2 || ^24.15.0 does not cover; running on 22.23.1
    (/home/hkjang/.nvm/versions/node/v22.23.1/bin/node) instead.
```

---

**전체 검증**: `go vet ./...` · `go test ./... -count=1` 15개 패키지 통과(이 기계에 `POSTGRES_DSN`
이 없어 DB integration 은 건너뛰었습니다 — 돌았다고 적지 않습니다) · `gofmt -l` 무출력 ·
`go build ./cmd/...` · 저장소 루트와 `web/` **두 자리 모두** `npm test` EXIT=0 ·
`npm run lint`(oxlint + Prettier) · `npm run typecheck` · `node scripts/check-i18n.mjs` ·
`npm run test:offline-queue` · `npm run build` · `npm run verify:pwa` ·
`npm audit --audit-level=high` 취약점 0 · `scripts/check-version.sh` ·
`release.yml` 과 같은 `docker build --build-arg VERSION=0.76.4 .`
