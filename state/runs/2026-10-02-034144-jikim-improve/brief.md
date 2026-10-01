# 과제서 — 2026-10-02 (run 2026-10-02-034144-jikim-improve, base main@eeca591)

- 과제: 프런트 검증 `npm --prefix web test` 가 이 머신에서 jsdom 환경 테스트를 시작조차 못 하는 원인을 재현해 좁게 고치고, 저장소가 요구하는 Node 버전을 툴체인이 실제로 강제하게 하기 (가치 5 / 위험 3 / 작업량 M)

- 왜: 지난 회차(2026-10-02-013734)가 Go 변경은 전부 green 이었는데도 `cd web && npm test --silent` (exit 1) 하나로 verify-failed 로 버려졌고, 같은 자리에서 두 번 막혔다. 지금은 "실패가 내 변경 때문인지 환경 때문인지" 를 아무도 판정할 수 없어서 — 저장소가 문서(`CONTRIBUTING.md:18`, `README.md:114`)·Docker(`Dockerfile:3`)·CI(`ci.yml:35,56`, `release.yml:71`) 네 곳에서 Node 24 를 요구하면서도 `web/package.json` 에 `engines` 가 없고 `.nvmrc` 도 없어 `scripts/verify.sh:48-50` 이 PATH 의 아무 node 로나 돌기 때문이다. 고치면 프런트 검증이 다시 돌고, 다음부터 런타임 불일치는 jsdom 내부 크래시가 아니라 읽을 수 있는 한 줄로 끝난다.

## 0단계 — 먼저 재현 (이것부터, 추측 금지)

정찰 세션에서는 `npm ci` 와 `node -e` 가 승인 거부로 막혀 **실패를 직접 재현하지 못했다. 아래 원인 가설은 전부 미확인이다.** 구현자는 반드시 먼저 다음을 돌리고 **실제 출력 전문을 journal.md 에 붙여라**:

```
node -v                                   # 기대: v22.23.1 (nvm 에 20.20.2 / 22.23.1 / 23.11.1 / 25.0.0 만 있고 24 는 없다 — 확인함)
npm --prefix web ci --no-audit --no-fund
npm --prefix web test -- --maxWorkers=1
```

이 뒤에 어느 분기인지 판정한다:

- **분기 A — 재현된다 (jsdom/undici 크래시).** 1단계로.
- **분기 B — 그냥 통과한다.** 지난 회차의 실패는 끊긴 `npm ci` 가 남긴 부분 설치였다는 뜻이다. 이때는 1단계를 하지 말고 **2단계만** 하라(그것만으로도 독립적으로 가치가 있다). 그리고 journal 에 "재현 안 됨 + 설치 출력" 을 남겨라.
- **분기 C — 다른 테스트가 깨진다** (예: 실제 Vite 개발 서버를 띄우는 `web/src/lib/vite-proxy.test.ts`가 샌드박스에서 포트/타임아웃으로 실패). 그러면 과제는 그 테스트 하나를 고치는 것으로 좁혀라.

## 1단계 (분기 A 일 때) — 원인을 좁혀 고친다

보고된 증상은 `webidl.util.markAsUncloneable is not a function` 으로 jsdom 환경 테스트 파일이 시작조차 못 하는 것이다(지난 회차 원장 기록, 내가 확인하지는 못했다). **지난 회차가 적은 "jsdom/undici 가 Node 24 를 요구한다" 는 설명은 lockfile 과 어긋난다 — 내가 확인한 사실은 이렇다:**

- `web/package-lock.json` 의 `node_modules/jsdom` = **30.0.1**, `engines.node` = `^22.22.2 || ^24.15.0 || >=26.0.0` → **v22.23.1 은 이 범위를 만족한다.**
- `node_modules/undici` = **8.10.1**, `engines.node` = `>=22.19.0` → 역시 만족한다.
- `node_modules/whatwg-url` 은 최상위 **17.1.0** 과 `data-urls/node_modules/whatwg-url` **16.0.1** 두 벌이 있다.
- lockfile 의 의존성 집합은 v0.2.22 이후 바뀐 적이 없다(`git show --stat eeca591 -- web/package.json web/package-lock.json` → 3줄, 버전 문자열뿐). 즉 같은 lockfile 로 2026-09-29 에는 vitest 59개가 통과했다.

따라서 "Node 를 24 로 올려라" 는 **이 머신에서 선택지가 아니다**(nvm 에 24 가 없다). 다음 순서로 좁혀라:

1. 설치된 실물을 확인: `node -p "require('web/node_modules/undici/package.json').version"` 류로 **실제 설치된 undici·jsdom 버전이 lockfile 과 같은지** 본다. 다르면 원인은 설치 오염이고, 고칠 것은 코드가 아니라 `rm -rf web/node_modules` 후 재설치다(이때는 2단계로).
2. 같다면 `markAsUncloneable` 이 어디서 오는지 한 번에 본다: `node -p "typeof require('node:worker_threads').markAsUncloneable"`. `undefined` 면 런타임 결함이고, `function` 이면 undici 내부 배선 문제다.
3. 고친다 — **가장 좁은 것 하나만**. 허용 가능한 형태는 ① `web/package.json` 의 `devDependencies` 에서 `jsdom` 을 이 Node 에서 도는 바로 아래 버전으로 핀(+ `package-lock.json` 갱신), 또는 ② `web/package.json` 에 `overrides` 로 undici 를 호환 버전에 고정. 둘 다 `npm --prefix web ci` 가 lockfile 과 일치해야 한다.

## 2단계 (어느 분기든 한다) — 문서에만 있는 Node 요구사항을 강제한다

1. `web/package.json`: `"private": true` 옆에 `"engines": { "node": ">=22.22.2" }`(1단계에서 확정한 실제 하한으로. 문서가 말하는 24 보다 낮다면 **문서를 코드에 맞추지 말고, 어느 쪽이 맞는지 journal 에 적고 더 좁은 쪽을 쓴다**).
2. 저장소 루트에 `.nvmrc` 추가. 내용은 CI·Dockerfile 과 같은 `24` 한 줄.
3. `scripts/verify.sh`: `npm --prefix web ci` (48행) **직전**에 선행 검사 한 블록 — `node -v` 의 major 를 읽어 `engines` 하한 미만이면 한국어 한 줄(`Node.js 24가 필요합니다. 현재: vX.Y.Z (.nvmrc 참고)`)로 `exit 1`. 기존 `set -euo pipefail` 과 출력 관례(성공 경로는 조용히)를 따를 것.

**하지 말 것:** `.github/workflows/*.yml` 의 `node-version: 24` 를 낮추지 말 것. `node-version-file: .nvmrc` 로 바꾸는 "정리" 도 이번엔 하지 말 것 — 릴리즈 경로를 건드리는 변경은 릴리즈까지 통과를 확인해야 하는데 이 환경에서는 Actions 실행을 확인할 수 없다.

## 수용 기준

1. `npm --prefix web ci --no-audit --no-fund` 다음 `npm --prefix web test -- --maxWorkers=1` 이 이 머신에서 **exit 0**, 그리고 실행된 테스트 수가 **59개 이상**(2026-09-29 기준선). 줄었으면 실패로 본다.
2. `bash scripts/verify.sh` 가 끝까지 통과하고 `검증 완료: jikim v0.2.26` 을 낸다.
3. 테스트를 하나도 끄지 않았다 — `vitest.config.ts` 의 `exclude` 에 추가 없음, `.skip`/`.todo` 추가 없음, `--passWithNoTests` 없음, 테스트 파일 삭제 없음. (비평자가 `git diff` 로 확인한다.)
4. Node 하한 미만에서 `bash scripts/verify.sh` 를 돌리면 **jsdom 내부 크래시가 아니라** 2단계의 한국어 한 줄로 즉시 멈춘다. `nvm exec 20.20.2 bash scripts/verify.sh` 로 실제 보여라(설치된 20.20.2 가 하한 미만이다).
5. Go 쪽은 손대지 않았다: `go test ./... -count=1`·`go vet ./...`·`gofmt -l .` 모두 exit 0, `git diff --stat` 에 `internal/`·`cmd/` 없음.

## 건드릴 파일 (프로덕션 5개 이내)

- `web/package.json` — `engines` 추가, 1단계에서 필요하면 `jsdom` 핀 또는 `overrides` 한 줄
- `web/package-lock.json` — 위 변경의 결과물(손으로 고치지 말고 npm 이 쓰게 할 것)
- `.nvmrc` — 신규, `24` 한 줄
- `scripts/verify.sh` — 48행 직전 선행 검사 블록 하나
- (1단계가 분기 B 로 불필요하면 위 2개만 바뀐다)

## 검증 명령 (이 저장소에서 실제로 도는 것)

```
node -v
npm --prefix web ci --no-audit --no-fund
npm --prefix web test -- --maxWorkers=1
npm --prefix web run lint
VITE_APP_VERSION="$(./scripts/version.sh)" npm --prefix web run build
go test ./... -count=1
go vet ./...
gofmt -l .
bash scripts/verify.sh
nvm exec 20.20.2 bash scripts/verify.sh      # 수용 기준 4 — 하한 미만에서 읽을 수 있게 멈추는지
```

주의: Bash 도구가 `&&` 로 이어붙인 복합 명령과 `./scripts/*.sh` 직접 실행을 승인 요구로 막는다. 한 줄씩 나눠 돌리고 `bash scripts/verify.sh` 형태로 부를 것. `scripts/e2e-docker.sh`·Playwright 는 `docs/screenshots/*.png` 를 덮어쓰므로 돌리지 말 것.

## 위험과 피할 것

- **워크플로를 느슨하게 만들어 통과시키는 것은 금지다.** 테스트 skip·exclude·`continue-on-error`·`node-version` 하향 전부 반려 사유다.
- `web/package-lock.json` 을 손으로 편집하지 말 것. `npm install` 이 쓰게 하고, 그 뒤 `npm ci` 가 깨끗이 도는지 반드시 재확인할 것(`npm ci` 는 lockfile 불일치면 바로 죽는다).
- 의존성 **메이저 업그레이드 금지**. jsdom 을 핀한다면 30.x 안에서, undici 는 8.x 안에서.
- 이 저장소의 교훈: 소스 문자열 grep 을 증거로 내지 말 것. 수용 기준 1·4 는 **실제 명령 출력**으로 증명해야 한다.
- 보호 경로(`internal/httpapi/auth*`·`oidc*`·`mcp_oauth.go`, `internal/store/users.go`, `migrations/`, `.github/workflows/`)는 이번 과제와 무관하다. 열지 말 것.
- 릴리즈 커밋 관례상 작업 커밋에서 `CHANGELOG.md` 와 `scripts/version.sh` 를 건드리지 않는다.
- `web/node_modules` 는 작업 트리에 없다. 설치에 시간이 든다(수 분). 작업량 M 의 대부분이 이 대기다.

## 차선 후보

**문서에만 있는 Node 요구사항을 강제하기 (2단계 단독, 가치 3 / 위험 1 / 작업량 S)** — 1단계가 분기 B·C 로 성립하지 않으면 2단계만 하라. `CONTRIBUTING.md:18`·`README.md:114`·`Dockerfile:3`·CI 3곳이 Node 24 를 말하는데 `web/package.json` 에 `engines` 가 없고 `.nvmrc` 가 없다는 사실은 내가 직접 확인했다. 이것만으로도 "환경 실패가 테스트 실패처럼 보이는" 이번 회차의 손실을 다시 막는다.

그 다음 후보: **AIConfig 의 temperature 범위 클램프 누락** (`internal/store/settings.go:566`, 2/1/S) — 단, 쓰기 경로(`settings.go:228-232`)가 0~2 를 강제하므로 **관찰 가능한 변화를 먼저 증명하지 못하면 손대지 말 것**(출력이 안 바뀌는 수정은 이 저장소의 반려 사유다).
