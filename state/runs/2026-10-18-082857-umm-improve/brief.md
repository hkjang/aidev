- 과제: 수정 과제 — 러너의 검증 명령 `npm test --silent`(저장소 루트)가 새 체크아웃에서 **항상** 실패한다: 루트 포워더가 web 의존성을 설치하지 않는다 (가치 4 / 위험 3 / 작업량 S)

- 왜: 이번 회차의 저장된 실패를 베이스(main@745dcf3, 작업 트리 clean)에서 그대로 재현했습니다 — `npm test --silent` → `run-on-supported-node: vitest is not installed: nothing resolves vitest/package.json from /home/hkjang/.cache/auto-improve-wt/umm/web/package.json. Run \`npm ci --prefix …/web\` first. This is a missing install, not an unsupported interpreter.` / **EXIT=1**, 그리고 `ls -d web/node_modules` → 없음. 즉 제품 버그가 아니라 **루트 `package.json` 의 포워더가 설치를 전제로만 동작**하는 것이고, 러너는 `npm ci --prefix web` 를 돌려 주지 않으므로 같은 실패가 2026-10-09·2026-10-16·2026-10-18 세 회차의 verify-failed 로 되풀이됐습니다. 루트 `package.json` 은 그 자신의 설명대로 "바깥에서 `npm run …` 이 저장소에 묶이도록" 존재하는 얇은 포워더이므로, 이 한 경로만 **스스로 설치하도록** 만들면 그 반복이 끝나고 CI·`make test-web`·web 의 게이트들은 지금처럼 큰 소리로 거절하는 계약을 그대로 유지합니다(느슨하게 만드는 변경이 아닙니다 — 검사를 빼지 않고 준비 단계를 더합니다).

- 수용 기준:
  1) `web/node_modules` 가 **없는** 트리에서 저장소 루트의 `npm test --silent` 가 수동 설치 없이 **EXIT=0** 으로 끝난다(vitest 가 실제로 돌아 파일 수·시험 수가 출력된다). 고치기 전 같은 명령이 위의 문장 그대로 EXIT=1 인 것을 먼저 로그로 남길 것.
  2) 설치가 이미 되어 있는 트리에서는 새 단계가 **아무 일도 하지 않는다** — `npm ci` 를 다시 돌리지 않고(= `web/node_modules` 를 지우지 않고) 출력도 없이 통과하며, 두 번째 `npm test --silent` 의 벽시계 시간이 첫 번째보다 **짧다**(재설치가 없다는 관찰 가능한 증거).
  3) 설치가 실패하는 경우(네트워크 없음 등) 명령은 **비영 종료**로 끝나고, 사람이 무엇을 해야 하는지(`npm ci --prefix <web 절대경로>`) 를 이름 붙여 말한다 — 조용히 통과하지 않는다. 시험이 이것을 증명해야 한다.
  4) 시험이 증명할 것: ① 해석되면 설치 명령을 **부르지 않는다**, ② 해석되지 않으면 `npm ci --prefix <web 절대경로>` 를 **정확히 그 인자로** 부른다, ③ 그 설치가 비영으로 끝나면 스크립트도 비영으로 끝나고 안내 문장을 낸다. 셋 다 소스 문자열 검사가 아니라 **실제 자식 프로세스**로 관찰할 것(PATH 앞에 기록용 `npm` 셸 셤을 두는 방식 — `web/scripts/require-installed.test.mjs:28-40` 의 `runNpmScript` 가 npm_config_*·node_modules PATH 를 걷어내는 방식을 그대로 따를 것).

- 건드릴 파일 (프로덕션 2 + 시험 1):
  - `web/scripts/ensure-install.mjs` (**신규, 프로덕션**) — `require-installed.mjs` 와 같은 모양으로 `createRequire(join(packageRoot,'package.json')).resolve('vitest/package.json')` 를 try/catch. 해석되면 **아무것도 출력하지 않고 exit 0**. `MODULE_NOT_FOUND` 면 한 줄 알리고 `spawnSync(npm, ['ci','--prefix',packageRoot], {stdio:'inherit'})` → 그 종료코드를 그대로 전달하고, 실패 시 `npm ci --prefix <packageRoot>` 를 직접 돌리라는 문장을 stderr 에 낸다. **`MODULE_NOT_FOUND` 가 아닌 오류는 relabel 하지 말고 rethrow**(`require-installed.mjs:48-52` 의 계약과 동일). `packageRoot` 는 cwd 가 아니라 `import.meta.url` 에서 구할 것(루트에서 불리므로 cwd 는 저장소 루트다).
  - `package.json` (**루트**, 한 줄 추가) — `"pretest": "node web/scripts/ensure-install.mjs"`. 기존 `test`·`typecheck`·`lint`·`build`·`verify:pwa`·`test:offline-queue` 본문과 긴 설명 주석은 **한 글자도 바꾸지 말 것**. 루트에는 `version` 필드가 없다(`check-version.sh` 가 그 부재에 기대고 있다) — 더하지 말 것.
  - `web/scripts/ensure-install.test.mjs` (**신규, 시험**) — 위 수용 기준 4의 세 가지. 실제 `npm` 을 돌려 네트워크를 타지 말고, 임시 디렉터리에 실행 가능한 `npm` 셤(인자를 파일에 적고 지정한 코드로 종료)을 만들어 PATH 앞에 두고 스크립트를 **자식 프로세스로** 띄울 것. 픽스처는 `require-installed.test.mjs` 처럼 실제 `web/package.json`·실제 스크립트를 **복사**해 쓸 것(복사본이 원본과 드리프트하지 않게).

- 검증 명령 (전부 이 저장소에서 실제로 도는 것):
  - 재현(고치기 전, 베이스): 저장소 루트에서 `npm test --silent` → 위 문장 / EXIT=1.
  - 1단계 설치: `npm ci --prefix web` (이 정찰 세션에서는 샌드박스가 승인하지 않아 **실행 미확인** — 구현자는 가장 먼저 이것이 되는지 확인하고, 네트워크가 없으면 이 과제는 성립하지 않으니 차선 후보로 갈 것).
  - 단위: `npm --prefix web test --silent` (vitest; 기준선은 24파일/260시험이었다 — 2026-10-16 회차 기록, 이번 미확인), 새 시험이 그만큼 늘어나는지 볼 것.
  - **수용 기준 1 의 end-to-end(운영자 규칙 4번)**: `mv web/node_modules /tmp/umm-nm-<tag>` 로 **지우지 말고 치운 뒤** 루트 `npm test --silent` → EXIT=0 을 로그로 남기고, 끝나면 트리 상태를 되돌릴 것(이 경로가 실제로 `npm ci` 를 돌리므로 시간이 걸린다).
  - 회귀: `make test-web` 8게이트(offline-queue / audit / typecheck / lint / check-i18n / test / build / verify:pwa) 전부 EXIT=0, `./scripts/check-version.sh`.
  - 포맷: `lint` 는 `src` 만 보므로 새 `scripts/*.mjs` 두 개는 **web 디렉터리에서** `npm exec -- prettier --check scripts/ensure-install.mjs scripts/ensure-install.test.mjs` (2026-10-09 회차가 루트에서 상대경로로 돌려 exit 2 를 만난 자리다).
  - Go·DB·Docker·E2E 는 이번 범위 밖(Go 소스 0줄).

- 위험과 피할 것:
  - **보호 경로 금지**: `.github/workflows/`(CI 는 `ci.yml` 의 `- run: npm ci` + `working-directory: web` 로 이미 설치하고 **루트 스크립트를 쓰지 않는다** — 이번 변경은 CI 에 영향이 없다), `Dockerfile`, `Makefile`, `VERSION`, `migrations/`, `internal/auth`, `package-lock.json`·`web/package-lock.json`.
  - **`web/package.json` 에는 `pretest` 를 달지 말 것.** `make test-web`·CI·개발자의 `npm --prefix web test` 는 지금처럼 **거절**해야 한다. 자동 설치는 바깥 호출자를 위한 **루트 포워더 한 경로**에만 둔다. 이 경계가 "게이트를 느슨하게 만들지 않았다" 의 근거다.
  - **`run-on-supported-node.mjs`·`require-installed.mjs` 를 고치거나 통합하지 말 것**(운영자 규칙 3번). 세 번째 형제 파일을 더하는 것이 이 저장소의 관례이며 그 이유가 `require-installed.mjs:17-25` 주석에 적혀 있다 — 새 파일 주석에서 그 자리를 가리키기만 할 것.
  - `npm ci` 는 `node_modules` 를 **지우고** 다시 깐다. 그래서 해석에 성공하면 **절대로** 부르지 않는 것이 수용 기준 2다(남의 설치를 날리면 운영자 규칙 9번과 같은 종류의 사고가 된다).
  - `oxlint` 가 권하는 `toSorted()` 는 이 저장소 `lib` 설정에서 `error TS2550` 이다(2026-10-16 회차가 실제로 부딪힘). 정렬이 필요하면 `new Set(...)` 비교로 피할 것. 기존 `src/` 의 oxlint 경고 ~52건은 상시 남는다 — **새 파일을 가리키는 경고만** 보라.
  - 커밋 트레일러(Co-Authored-By 등) 금지, `VERSION` 미변경.

- 차선 후보: `go.mod` 의 `toolchain` 과 `Dockerfile:9` 빌더 태그가 같은 Go 패치여야 한다는 계약을 아무 검사도 지키지 않는다 — `scripts/check-version.sh` 와 같은 모양의 작은 드리프트 검사를 `scripts/` 에 **추가만** 하고(Makefile·워크플로 배선은 보호 경로이므로 이번 회차에서 제외), 두 파일의 현재 값이 일치함을 실행으로 찍을 것. 2026-10-18 회차가 두 숫자를 손으로 맞춘 직후이므로 지금은 초록이고, 검사는 다음 드리프트를 잡는 망이다. (가치 4 / 위험 2 / S)
