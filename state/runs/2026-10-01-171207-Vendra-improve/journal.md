# 회차 노트 2026-10-01-171207-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:12] base pinned — main@887bd84
- [러너 17:12] autonomy release — 

## 정찰 노트
- 우선 과제가 배정되어 후보를 고르지 않았다. 지난 런 verify.txt 를 읽어 원인을 특정했다: 테스트 실패가 아니라 `jsdom` import 시점에 `undici` 가 터져(`webidl.util.markAsUncloneable is not a function`, cachestorage.js:20 ← undici/index.js:179 ← jsdom/lib/api.js:12) 22개 파일이 **하나도 시작하지 못한다**(exit 1 in 1s). web/ 최신 커밋은 수 주 전 35fd306 이므로 소스가 아니라 의존성 그래프 문제다.
- 확신 없는 곳(과제서에 「미확인」으로 적었다): **어느 undici 버전이 동작하는지 확정하지 못했다.** 이 세션은 npm 실행 권한이 거부되어 로컬 재현도 못 했다 — lock(undici 8.10.0, jsdom 30.0.1→`^8.9.0`)과 node v22.23.1 까지만 읽었다. 원인이 undici 핀인지 Node patch 차이인지 구현자가 설치된 `node_modules/undici` 를 읽어 먼저 가려야 한다(과제서 「진단 절차」 1~3).
- 구현자가 조심할 것: 깨진 것이 **전부**이므로 `vitest.config.ts` 의 `exclude`·`ci.yml:72` 삭제·`continue-on-error` 는 모두 「워크플로를 느슨하게」에 해당해 금지다. `package-lock.json` 은 손으로 편집하지 말고 재생성하며, `rm -rf node_modules` 뒤 clean install 에서 통과해야 CI 와 같은 것을 증명한다. Vitest 는 `/mnt/c` 에서 돌지 않으니 Linux 워크트리에서 돌릴 것.
- 차선은 `compare_suppliers` 설명 좁히기로 두었지만, 웹 검증이 막힌 동안은 어떤 과제도 통과하지 못하므로 1순위를 먼저 끝낼 것. 프로필은 0일 전이라 다시 쓰지 않았다(단, 「웹 테스트 실행」이 미확인 항목으로 남아 있던 자리가 바로 이번 실패 지점이다).
- [러너 17:17] scout done — 웹 테스트 스위트가 jsdom 을 import 하지 못해 22개 테스트 파일이 **하나도 시작하지 못하는** 것 고치기 — `

## 구현 노트
- 과제서의 전제(undici lockfile 핀)는 거짓이다. 같은 `package-lock.json` 으로 상위 오염이 없는 체크아웃(`/var/tmp/vendra-web-proof`)에서 `npm ci --ignore-scripts && npm test` 가 `Test Files 22 passed (22) / Tests 98 passed (98)` exit 0 이다. 실제 원인은 npm 이 **모든 상위 `node_modules/.bin`** 을 PATH 앞에 붙이고 vitest shebang 이 `#!/usr/bin/env node` 라, `/home/hkjang/node_modules/.bin/node`(= node 20.19.2)가 nvm v22.23.1 을 가리는 것. Node 20 에는 `worker_threads.markAsUncloneable` 이 없어 jsdom→undici import 가 터진다. fork 워커 안에서 `process.version`/`execPath` 를 직접 읽어 증명했고(프로브 파일은 삭제), 의존성은 한 글자도 바꾸지 않았다. 바꾼 것: `web/package.json` engines+pretest, `web/scripts/check-node-version.mjs`, `docs/operations.md` 한 절.
- **확신 없는 곳**: ① CI(`ubuntu-latest`, `setup-node@v4 node-version: 22`)에서 이 가드가 통과하는 것을 **실행해 보지 못했다** — `/home/runner` 위쪽에 `node` bin 패키지가 없다는 전제에 기대며, `node-version: 22` 가 해석되는 patch 가 22 메이저인 한 통과한다(가드는 메이저만 본다). ② `npm ci --ignore-scripts` 가 `pretest` 를 끄지 않는다는 것은 플래그가 그 명령에만 붙으므로 맞다고 보지만 CI 에서 확인하지는 않았다. ③ `go test` 는 DSN 셋을 unset 으로 돌려 **DB 통합 테스트가 skip** 됐다(httpapi 1.495s) — Go 코드를 안 건드렸으므로 회귀 확인용일 뿐, 「DB 검증」이 아니다.
- **일부러 하지 않은 것**: `overrides` 로 undici 를 내리는 것(= jsdom 이 engines 로 지원 안 한다고 선언한 Node 20 을 지원하려 의존성을 끌어내리는 일), `vitest.config.ts`·`ci.yml`·`web/src/**` 손대기, `"test"` 스크립트를 `$npm_node_execpath` 로 바꿔 잘못된 Node 를 우회하기(Windows cmd 에서 깨지고 깨진 머신을 저장소에 새기는 일), 러너 홈의 `node_modules/node` 지우기(저장소 밖 전역 환경 변경). `npm` 의 `scripts-prepend-node-path` 는 npm 10 에서 더 이상 동작하지 않음을 실험으로 확인했다.
- **다음 역할이 조심할 것**: 이 머신에서는 **고친 뒤에도 `cd web && npm test` 가 exit 1 이다** — 이제 120줄 벽 대신 `web tests need Node >= 22 ... resolved node: /home/hkjang/node_modules/node/bin/node` 한 줄로 끝난다. 그것은 저장소 결함이 아니라 러너 환경(`/home/hkjang/node_modules/node`, `/tmp/node_modules/node`)이다. 스위트 자체를 보려면 `/home/hkjang` 과 `/tmp` 밖(예: `/var/tmp/…`)에 복사해 돌리거나, PATH 에서 그 `.bin` 들을 뺀 뒤 `node node_modules/vitest/vitest.mjs run` 으로 직접 돌릴 것.
- [러너 17:29] brief rejected — 기각 — 과제서의 전제(「`undici` 가 lockfile 에서 깨진 조합으로 고정됐다」)가 거짓이다. lockfile 을 한 글자도 바꾸지 않고
- [러너 17:29] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
