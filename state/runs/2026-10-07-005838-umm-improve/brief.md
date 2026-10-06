- 과제: 수정 과제 — 설치 누락을 이름 붙여 말해 주는 것이 프런트엔드 게이트 여덟 개 중 세 개뿐이다 (`typecheck`·`lint`·`build` 는 `tsc: not found` 로 죽는다) (가치 4 / 위험 2 / 작업량 S)

## 먼저 읽을 것 — 지정된 실패의 성격 (정찰이 실제로 재현했음)

지정된 검증 `npm test --silent` 를 베이스(main@29149e5, `VERSION`=0.76.8) 그대로 재현했습니다. **EXIT=1 이고 출력은 이것입니다:**

```
run-on-supported-node: PATH gave Node 20.19.2 (/home/hkjang/node_modules/node/bin/node), which engines.node ^22.22.2 || ^24.15.0 does not cover; running on 22.23.1 (/home/hkjang/.nvm/versions/node/v22.23.1/bin/node) instead.
run-on-supported-node: vitest is not installed: nothing resolves vitest/package.json from /home/hkjang/.cache/auto-improve-wt/umm/web/package.json. Run `npm ci --prefix /home/hkjang/.cache/auto-improve-wt/umm/web` first. This is a missing install, not an unsupported interpreter.
```

`ls -d web/node_modules` → **없음**. 즉 **이것은 저장소 결함이 아닙니다.** `npm ci` 가 돌지 않은 체크아웃에서는 어떤 npm 프로젝트의 `npm test` 도 통과할 수 없고, 이 저장소는 이미 그 사실을 이름 붙여 말해 주고 있습니다(9755334 가 바로 그것을 고쳤습니다). **이 전제는 2026-10-03·2026-10-05 두 회차가 이미 같은 결론에 도달했습니다 — 세 번째입니다.**

**그러므로 하지 말 것 (금지):**
- 워크플로(`.github/workflows/*`)·`Makefile`·`Dockerfile`·`engines` 를 느슨하게 만들어 통과시키기. **워크플로는 멀쩡합니다** — `ci.yml` 은 프런트엔드 단계 앞에서 `npm ci`(`working-directory: web`)를 돌리고 `release.yml` 에는 npm 테스트 단계가 없습니다(두 회차가 확인). 이번 회차에도 워크플로 파일은 **열어 읽기만 하고 바꾸지 마세요.**
- 테스트·검증 타깃이 `npm ci` 를 스스로 돌리게 만들기 — 멀쩡한 `node_modules` 를 지우는 되돌릴 수 없는 환경 변경입니다(운영자 규칙 10번). `Makefile:test-web` 주석이 이미 그 결정을 적어 두었습니다.
- 메시지 문구를 다시 쓰기 — `run-on-supported-node.mjs` 의 그 문장은 2026-10-04 에 이 오진을 막기 위해 고른 것이고 시험이 못 박고 있습니다.

**그래서 이번에 고칠 것은 이 실패가 드러낸, 아직 남아 있는 같은 종류의 결함입니다** (앞 두 회차와 다른 자리입니다): 설치 누락을 이름 붙여 말해 주는 경로가 **프런트엔드 게이트 여덟 개 중 세 개뿐**입니다.

## 왜

`web/package.json` 의 scripts 를 직접 읽어 확인했습니다 — `run-on-supported-node.mjs` 를 거치는 것은 `test`·`test:watch`·`test:offline-queue` **셋뿐**이고, `typecheck`(`tsc -b --pretty false`)·`lint`(`oxlint src && prettier --check src`)·`build`(`tsc -b && vite build`)는 `node_modules/.bin` 의 심을 PATH 로 직접 부릅니다. 설치가 없는 체크아웃에서 이 셋은 `sh: 1: tsc: not found` 류의 **이름 없는 exit 127** 로 죽습니다(아래 "미확인" 참조). 그 출력은 설치 누락을 가리키지 않고, 바로 위에 같은 스크립트가 찍는 인터프리터 전환 알림도 없어서 **2026-10-04 가 `vitest` 한 자리에서 고친 그 오진을 다른 세 자리에서 그대로 되풀이할 수 있습니다.** CI 는 항상 `npm ci` 를 먼저 돌아 안전하지만, 저장소 루트 `package.json` 이 `typecheck`·`lint`·`build` 를 그대로 전달하고 러너의 검증 목록도 이 명령들을 개별로 돌리므로 — 설치 단계가 빠지거나 실패한 체크아웃에서 사람이 보는 첫 줄이 `tsc: not found` 입니다. 고치면 설치 누락은 **어느 게이트로 들어와도** 같은 한 문장으로 자기 원인과 `npm ci --prefix …` 를 말합니다.

(`verify:pwa`·`audit` 는 범위 밖입니다 — `verify:pwa` 는 `node scripts/verify-pwa.mjs` 로 의존성이 필요 없고, `audit` 는 락파일만 읽습니다. `make test-web` 은 이미 안전합니다: 첫 게이트 `test:offline-queue` 가 래퍼를 거쳐 이름 붙은 실패를 냅니다 — **`Makefile` 은 손대지 마세요.**)

## 수용 기준

1) `web/node_modules` 가 없는 체크아웃에서 `npm --prefix web run typecheck`·`run lint`·`run build` 가 각각 **이름 붙은 한 문장**으로 실패한다 — 없는 의존성 이름, 해석 기준 manifest 경로, `npm ci --prefix <packageRoot>`, "missing install, not an unsupported interpreter" 를 담고, 종료 코드는 **비영 그대로**(느슨하게 만든 것 없음).
2) 같은 출력에 `MODULE_NOT_FOUND`·`Require stack`·`/Node\.js v\d/`·`tsc: not found`(또는 `oxlint`/`vite` 의 not found)가 **없다** — 즉 그 오진의 정확한 문구가 사라졌음을 못 박는다.
3) **통제 시험**: 실제로 설치된 `web/` 에 대해 같은 가드가 EXIT=0 이고, `npm --prefix web run typecheck`·`run lint`·`run build` 가 수정 전과 똑같이 통과한다(가드가 "전부 거절" 로 통과하는 것이 아니라는 증거이자 릴리즈 경로가 멀쩡하다는 증거).
4) 시험이 **프로덕션 배선을 통과한다** — 가드 모듈을 직접 부르는 것만으로는 부족하고, `node_modules` 없는 임시 패키지 루트에 대해 **실제 `npm run typecheck` 를 자식 프로세스로 띄워** 새 메시지가 나오는 것을 단언해야 합니다(운영자 규칙 5번).

## 건드릴 파일 (프로덕션 2 + 시험 1)

- `web/scripts/require-installed.mjs` — **새 파일**. `node scripts/require-installed.mjs <dep> [dep...]` 로 받은 각 이름에 대해 `createRequire(<web/package.json 절대경로>).resolve(`${dep}/package.json`)` 을 try/catch 하고, `error?.code !== 'MODULE_NOT_FOUND'` 는 **relabel 하지 않고 rethrow**, MODULE_NOT_FOUND 면 `run-on-supported-node.mjs` 의 `fail()` 과 **글자 그대로 같은 문장**으로 stderr 에 쓰고 `process.exit(1)`. 전부 해석되면 아무것도 출력하지 않고 0 으로 끝납니다(게이트 로그를 더럽히지 않음). 머리말 주석에 **왜 `run-on-supported-node.mjs` 에서 import 하지 않는가** 를 적을 것 — 그 파일은 로드되는 즉시 대상을 `spawnSync` 하는 **진입점**이라 import 할 수 없습니다(`verify-offline-queue.mjs` / `verify-pwa.mjs` 가 서로 다른 계약으로 따로 사는 것과 같은 이유). 그 파일을 **수정하지 마세요** — 그 안의 try/catch 와 메시지는 그대로 둡니다.
- `web/package.json` — `scripts` 에 세 줄만 추가: `"pretypecheck": "node scripts/require-installed.mjs typescript"`, `"prelint": "node scripts/require-installed.mjs oxlint prettier"`, `"prebuild": "node scripts/require-installed.mjs typescript vite"`. **기존 `typecheck`·`lint`·`build`·`test*` 의 본문은 한 글자도 바꾸지 마세요.** `engines`·`version`·의존성 범위도 손대지 않습니다. (`typescript` 는 `tsc` 의 패키지 이름입니다 — bin 이름이 아니라 **패키지 이름**으로 resolve 해야 합니다. `oxlint`·`prettier`·`vite` 는 이름이 같습니다. 세 이름이 실제로 `web/package.json` 의 devDependencies 에 있는지 먼저 읽어 확인할 것 — 정찰은 `@playwright/test`·`@types/node` 까지만 보고 잘렸습니다.)
- `web/scripts/require-installed.test.mjs` — **새 파일**. 기존 `web/scripts/run-on-supported-node.test.mjs` 의 설치 누락 시험 3개와 **같은 방식**을 쓰세요(그 파일을 먼저 읽을 것): 실제 `web/package.json` 과 실제 `scripts/*.mjs` 를 실행 시점에 복사한 임시 패키지 루트(= `npm ci` 전 상태, `node_modules` 만 없음)를 만들고 자식 프로세스로 띄우기. 손으로 만든 대역·가짜 resolve 금지.

**이 네 파일 밖은 건드리지 마세요** — 특히 `Makefile`, `.github/workflows/*`, `Dockerfile`, `web/scripts/run-on-supported-node.mjs`, 루트 `package.json`(전달만 하므로 바꿀 것이 없습니다), `VERSION`(개선 회차는 버전을 올리지 않습니다), `web/package-lock.json`(의존성을 더하지 않으므로 바뀔 이유가 없습니다 — `git status` 로 확인).

## 검증 명령 (이 저장소에서 실제로 도는 것)

착수 순서를 지키세요. **① 먼저 수정 전 실패를 찍으세요** (`web/node_modules` 가 지금 없으므로 베이스 상태가 곧 실패 상태입니다 — 이 기회를 설치로 날리기 전에 먼저 기록):

```
cd /home/hkjang/.cache/auto-improve-wt/umm
npm --prefix web run typecheck ; echo "TYPECHECK_EXIT=$?"
npm --prefix web run lint      ; echo "LINT_EXIT=$?"
npm --prefix web run build     ; echo "BUILD_EXIT=$?"
```

세 출력을 그대로 보고에 붙이세요(이것이 "이름 없는 실패" 의 증거입니다). **② 그 다음 설치**: `npm ci --prefix web`(CI 의 `ci.yml` 프런트엔드 단계와 같은 것). **③ 고치고** ④ 전부 돌리세요:

- `npm --prefix web test --silent` → 기준선 **22파일 / 230시험** + 새 시험만큼 (EXIT=0)
- `npm --prefix web run typecheck`, `npm --prefix web run lint`(prettier "All matched files use Prettier code style!" — 남은 경고는 기존 파일의 것; 새 파일 2개를 가리키는 경고는 0건이어야 함), `npm --prefix web run build`, `npm --prefix web run verify:pwa`(150 assets), `npm --prefix web run test:offline-queue`, `npm --prefix web audit --audit-level=high`, `node web/scripts/check-i18n.mjs`(1060키)
- `make test-web` → **게이트 8개 전부 EXIT=0** (make 자신의 에코로 8줄 확인). `make test-go` 는 Go 0줄이므로 생략해도 되지만 돌리면 `POSTGRES_DSN` 없이는 DB 통합이 **SKIP** 임을 반드시 밝힐 것.
- `scripts/check-version.sh` (`0.76.8 is consistent`)
- **릴리즈 경로 확인(운영자 규칙 4번 — `prebuild` 가 Docker 빌드의 `npm run build` 앞에 끼어들기 때문에 필수)**: `docker build --build-arg VERSION=0.76.8 .` EXIT=0, 검증 이미지 삭제. 앞 두 회차가 이 환경에서 성공시켜 두었으므로 기준선이 있습니다. **이것이 통과하지 않으면 `prebuild` 를 빼고(typecheck·lint 둘만) 그 사실을 보고에 적으세요** — 통과하지 않은 것을 통과했다고 쓰지 마세요.
- ⑤ **수정 전/후 대조**: 고친 뒤 `web/node_modules` 를 잠시 다른 이름으로 옮겨(`mv`, 지우지 말 것) 세 명령이 새 메시지를 내는 것을 **실제 경로에서** 확인하고 되돌리세요(2026-10-04 회차가 한 end-to-end 확인과 같은 방식). 되돌린 뒤 `make test-web` 이 다시 EXIT=0 임을 확인.
- `git status` 로 산출물 미커밋 확인(`web/dist` 는 ignore).

## 위험과 피할 것

- **`prebuild` 가 릴리즈 경로를 지납니다** — `Dockerfile` 이 `npm run build` 를 부르면 가드가 먼저 돕니다. 설치가 있으면 조용히 0 으로 끝나야 하고, 조금이라도 빌드를 막으면 안 됩니다. `docker build` 가 유일한 증거입니다.
- **npm 의 `pre*` 훅 동작을 가정하지 말고 실행으로 확인하세요** — 가드가 비영으로 끝날 때 npm 이 본 스크립트를 **돌리지 않고** 중단하는지 ④/⑤ 에서 직접 보세요. 정찰은 이것을 실행으로 확인하지 못했습니다(아래 미확인).
- 메시지 문장을 두 파일이 각자 갖게 됩니다. **통합하려 하지 마세요** — 한쪽은 진입점(spawn)이고 한쪽은 가드이며, 운영자 규칙 3번이 금지하는 "여러 경로의 같은 종류를 통합" 에 해당합니다. 대신 새 파일 주석이 그 자리를 가리키게 하세요.
- 임계값·범위를 줄이거나 게이트를 빼지 마세요. 이 변경은 **검사를 더하기만** 합니다.
- 보호 경로(`internal/auth/`, `migrations/`, `.github/workflows/`)는 이번 과제에 전혀 등장하지 않습니다. Go 코드 0줄.
- 2026-10-06 에 PR #167 이 머지 뒤 되돌려져 자율화가 low-risk 로 강등됐습니다. 범위를 네 파일로 묶어 두세요.

## 미확인 (정찰이 확인하지 못한 것 — 추측으로 쓰지 않았습니다)

- `npm --prefix web run typecheck`/`run lint`/`run build` 의 **실제 실패 출력과 종료 코드**: 이 세션에서 그 명령의 실행 권한이 거부되어 돌리지 못했습니다. `tsc: not found` / exit 127 은 **npm 의 알려진 동작에 대한 추론**입니다. 구현자는 위 ① 을 가장 먼저 돌려 실제 문구를 확인하고, 만약 npm 이 이미 설치 누락을 이름 붙여 말해 준다면 **이 과제는 성립하지 않습니다** — 그때는 아래 차선 후보로 가고 그 판정을 보고에 적으세요.
- `typescript`·`oxlint`·`vite` 가 `web/package.json` devDependencies 에 실제로 있는지: 출력이 `@types/react-dom` 에서 잘려 끝까지 보지 못했습니다. 첫 작업으로 그 블록을 읽으세요.
- npm 의 `pre*` 훅이 비영 종료 시 본 스크립트를 중단하는지(위 "위험" 참조).
- 러너가 왜 `npm ci` 없이 검증했는지: 러너 설정은 이 저장소 밖이고 이번 범위 밖입니다. **그 쪽을 고치려 하지 마세요.**

## 차선 후보

**`edge-vocabulary.ts` 의 알려진 라벨 12개를 실제 타입·실제 `translate` 로 못 박기 (가치 2 / 위험 1 / S)** — `web/src/lib/` 에서 `.test.ts` 가 없는 유일한 모듈입니다. 프로덕션 **0파일**(시험 1파일). 폴백 `?? relation` 은 `migrations/010_memory_graph.sql:56`(relation 6값)·`:60`(origin 6값) 의 CHECK 가 값을 묶어 **도달 불가**이므로 — 앞 회차 정찰이 그 두 줄을 열어 사전의 12키와 일치함을 확인했습니다 — **버그 수정이 아니라 테스트 공백 보강 + 드리프트 방어**로 정직하게 적으세요(7번째 값을 DB·Go 에 넣고 프런트 사전을 잊으면 원시 식별자가 화면에 뜹니다). 시험은 `web/src/api.ts` 의 실제 `EdgeRelation`/`EdgeOrigin` 과 실제 `setLocale`/`translate`/`en.ts` 로 ko·en 양쪽 12개를 단언. 선행: `npm ci --prefix web`.
