# v0.76.5 — 설치가 빠진 실패는 Node 를 탓하지 않고 설치를 말합니다

이번 회차에도 **제품은 한 줄도 바뀌지 않았습니다.** umm 을 쓰는 사람이 보는 화면, API, 데이터는
v0.76.4 와 똑같습니다. 달라진 것은 **`npm ci` 를 아직 돌리지 않은 체크아웃에서 `npm test` 를 친
사람**이 받던 답입니다. 전에는 그 답이 Node 버전으로 끝났고, 그래서 두 번의 릴리즈 검증이 설치가
아니라 인터프리터를 쫓았습니다.

```
    web/node_modules 없이  저장소 루트에서  npm test

            전                                            후
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    Error: Cannot find module                 run-on-supported-node: vitest is
      'vitest/package.json'                     not installed: ... Run
    Require stack:                              `npm ci --prefix <...>/web` first.
    - .../web/package.json                      This is a missing install, not an
      at Function._resolveFilename              unsupported interpreter.
      at .../run-on-supported-node.mjs:164
      code: 'MODULE_NOT_FOUND'
    Node.js v20.19.2                          EXIT=1  (그대로)
    EXIT=1
```

두 경우 모두 종료 코드는 1 입니다. **고친 것은 통과 여부가 아니라 무엇이 잘못됐는지입니다.**

## 마지막 줄이 하필 Node 였습니다

`npm test` 는 루트에서 `web/` 으로 넘어가고, 거기서 `web/scripts/run-on-supported-node.mjs` 가
받습니다. 이 래퍼는 지원되는 인터프리터를 고른 뒤 돌릴 대상의 실행 파일을 `node_modules` 에서
resolve 합니다. `npm ci` 가 돌지 않은 체크아웃에는 resolve 할 것이 없고, 그 `MODULE_NOT_FOUND` 는
**잡히지 않은 채로 올라갔습니다.** Node 가 잡히지 않은 예외를 찍는 방식은 정해져 있습니다 —
스택을 찍고, **마지막 줄에 자기 버전을 적습니다.**

하필 그 바로 위 줄이 같은 스크립트가 찍은 인터프리터 전환 통지였습니다.

```
    run-on-supported-node: PATH gave Node 20.19.2 (...), which engines.node
    ^22.22.2 || ^24.15.0 does not cover; running on 22.23.1 (...) instead.
    ...
    Node.js v20.19.2
```

읽는 사람에게 이것은 한 덩어리입니다. 위에서 Node 이야기를 하고, 아래에서 Node 버전으로 끝납니다.
그래서 **설치 누락이 이 스크립트가 해결하려고 존재하는 바로 그 문제(가려진 Node)처럼 읽혔고**,
릴리즈 검증이 두 번 연속으로 같은 자리에서 인터프리터를 뒤졌습니다. 이 파일이 v0.76.4 에서 선언한
계약은 "어떤 실패 메시지도 Node 나 버전을 입에 올리지 않으니 발견할 수 없다" 를 고치는 것이었고,
그 계약이 **정확히 이 한 분기에서만 거꾸로 깨져 있었습니다.**

## 이름을 붙여 거절합니다

그 resolve 하나만 감싸서, 의존성의 이름 · resolve 의 기준이 된 manifest · 돌려야 할 명령 ·
그리고 **인터프리터 문제가 아니라는 사실**을 적고 거절합니다.

```
    run-on-supported-node: vitest is not installed: nothing resolves
    vitest/package.json from /.../web/package.json. Run
    `npm ci --prefix /.../web` first. This is a missing install, not an
    unsupported interpreter.
```

경로는 사람이 다시 조립할 필요가 없도록 **실제로 쓰인 절대경로**로 나갑니다. 네 가지를 함께 적는
이유는 이 실패를 처음 보는 사람이 묻게 되는 것이 그 네 가지뿐이기 때문입니다 — 무엇이 없는지,
어디를 기준으로 찾았는지, 무엇을 치면 되는지, 그리고 **내가 지금 Node 를 뒤져야 하는지**.

## 범위를 넓히지 않은 자리

- **`MODULE_NOT_FOUND` 가 아닌 오류는 다시 던집니다.** 깨진 `package.json`, 읽기 권한, 심볼릭
  링크 고리 — resolve 가 실패하는 다른 이유들은 설치 누락이 아니고, 그것들에 `npm ci` 를 권하는
  것은 원래 메시지를 **쓸모없게** 만듭니다. 모든 것을 삼키는 catch 가 바로 이 릴리즈가 고치는
  종류의 오진을 만듭니다.
- **종료 코드는 1 그대로입니다.** 시작조차 못 한 실행이 초록색으로 읽히는 일은 없어야 합니다.
- `engines.node` 범위, 워크플로(`ci.yml` · `release.yml`), Makefile, Dockerfile, `npm audit`
  임계값은 **손대지 않았습니다.** 먼저 읽어서 워크플로 쪽은 멀쩡함을 확인했습니다 —
  `ci.yml` 은 프런트엔드 단계 앞에서 `npm ci` 를 돌리고(`working-directory: web`),
  `release.yml` 에는 npm 테스트 단계가 아예 없습니다.
- **래퍼가 대신 설치하게 만들지 않았습니다.** 시험을 돌리라고 부른 스크립트가 묻지도 않고
  `node_modules` 를 쓰는 것은 되돌리기 어려운 환경 변경입니다. 이 래퍼는 무엇을 해야 하는지
  말하고 멈춥니다.

## 시험은 대역을 쓰지 않습니다

`web/scripts/run-on-supported-node.test.mjs` 는 **실제 래퍼를 npm 과 같은 방식으로 자식 프로세스로
띄웁니다**(`npm_node_execpath` 까지 같게). 상대는 실행 시점에 복사한 **실제** `web/package.json` 과
**실제** `scripts/*.mjs` 가 든 임시 패키지 루트입니다 — `node_modules` 만 없는, 즉 `npm ci` 전의
체크아웃입니다. 복사를 실행 시점에 하므로 시험이 쓰는 manifest 는 **이 저장소가 내보내는 것과
어긋날 수 없습니다.**

```
✓ 이름 붙은 실패 + `npm ci` 안내가 나온다                     EXIT=1
✓ 출력에 /Node\.js v\d/ · MODULE_NOT_FOUND · Require stack 이 없다
✓ 통제 시험: 설치된 vitest --version 은 그대로 돌아간다        EXIT=0
```

가운데 줄이 이 릴리즈의 주장입니다. 그 **정확한 문구들이 없다**는 것을 못 박아, 오진을 만들던
출력이 다시 돌아오면 시험이 먼저 넘어집니다. 그리고 마지막 줄이 없으면 위의 두 줄은 "전부
거절하기" 로도 통과할 수 있습니다 — 그래서 함께 있습니다.

## 검증

```
✓ 루트 npm test --silent                 EXIT=0   20개 파일 218개 시험
✓ web/node_modules 없이 루트 npm test    EXIT=1   새 메시지 (임시 복사가 아닌 실제 경로)
✓ 프로덕션 파일만 되돌리면               같은 시험 2개가 다시 실패
✓ npm run lint (oxlint + Prettier)       EXIT=0   All matched files use Prettier code style!
✓ npm run typecheck                      EXIT=0
✓ npm run test:offline-queue             EXIT=0
✓ npm run build · npm run verify:pwa     EXIT=0   150개 build asset
✓ node web/scripts/check-i18n.mjs        EXIT=0   1060개 키
✓ npm audit --audit-level=high           EXIT=0
```

---

**전체 검증**: `gofmt -l` 무출력 · `go vet ./...` · `go build ./cmd/...` ·
`go test ./... -count=1` 15개 패키지 통과(이 기계에 `POSTGRES_DSN` 이 없어 DB integration 은
건너뛰었습니다 — 돌았다고 적지 않습니다. 이번 변경에 Go 코드는 0줄입니다) ·
`scripts/check-version.sh` · `release.yml` 과 같은 `docker build --build-arg VERSION=0.76.5 .`
