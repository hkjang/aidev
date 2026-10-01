# 회차 노트 2026-10-02-072743-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:27] base pinned — main@60fca20
- [러너 07:27] autonomy release — 

## 정찰 노트
- 우선 과제가 배정돼 있어 새 후보를 고르지 않았다. 실패를 그 자리에서 재현했다: 새 워크트리에 `web/node_modules` 가 없고 `cd web && npm test --silent` → `sh: 1: vitest: not found`, exit 1. 원인은 테스트 코드가 아니라 `web/package.json:9` 의 `"test": "vitest run"` 만 `.bin` 심링크와 기설치를 전제하는 것(저장소 나머지는 전부 `node node_modules/<pkg>/…` + `build-ui.sh:6` 의 설치 가드).
- 확신 없는 곳 두 군데를 과제서에 미확인으로 적었다: ① 이 환경에서 `npm ci` 가 레지스트리에 닿는지 — 샌드박스가 `npm ci` 실행을 막아 끝까지 확인 못 했다. 닿지 않으면 `pretest` 가드로도 수용 기준 1을 못 채운다. ② vitest 5.x 가 `node_modules/vitest/vitest.mjs` 를 패키지 루트에 두는지 — Makefile·CI·Dockerfile 이 이미 그 경로를 전제하니 맞을 가능성이 높지만 설치 후 눈으로 확인하라고 적었다.
- 구현자가 조심할 것: `release.yml` 은 읽어 보니 web 테스트를 호출하지 않는다(verify-version → package-offline → verify-offline → upload). 두 번 실패한 것은 러너의 verify 단계이므로 `release.yml` 을 건드리지 말 것. `--passWithNoTests`·`continue-on-error`·`test.include` 축소 같은 "느슨하게 통과" 는 금지이고, `web/package-lock.json` 이 `git status` 에 뜨면 되돌릴 것(`npm install` 이 아니라 `npm ci`).
- 프런트 소스는 수정 대상이 아니다 — 건드리면 CI 에서 `tsc -b` 가 vitest 보다 먼저 깨진다. 프로필은 0일 전 것이 지금 코드와 어긋나지 않아 새로 쓰지 않았다.
- [러너 07:32] scout done — `cd web && npm test` 가 `node_modules/.bin` 심링크와 "이미 설치돼 있음" 을 전제해 깨끗한 체크아웃에서 항상 실패

## 구현 노트
- 과제서대로 `web/package.json` 의 `test` 를 패키지 경로 호출로 바꾸고 `pretest` 설치 가드(build-ui.sh:6 과 동일)를 넣었다. `Makefile:17`·`ci.yml:57` 은 `npm test --silent` 로 통일 — vitest 호출은 `web/package.json:11` 한 곳만 남는다. 커밋 60d67c0.
- 과제서의 미확인 2건 결론: `npm ci` 는 **닿는다**(191 packages, 9s), `node_modules/vitest/vitest.mjs` 경로도 **맞다**. 대신 과제서가 몰랐던 2차 실패가 있었다 — npm 은 lifecycle PATH 앞에 **모든 상위** `node_modules/.bin` 을 붙이고, 이 머신의 `/home/hkjang/node_modules/.bin/node` 는 Node 20.19.2 다. Node 20 엔 `worker_threads.markAsUncloneable` 이 없어 jsdom 30 의 undici 8(`engines: >=22.19.0`)이 로드 중 죽는다. 그래서 인터프리터를 `"${npm_node_execpath:-node}"` 로 고정했다.
- **확신 없는 곳**: `npm_node_execpath` 는 npm 전용 변수다. yarn/pnpm 으로 `test` 를 돌리면 미설정이라 `:-node` 폴백으로 내려가고, 그때는 PATH 오염에 다시 노출된다(이 저장소는 npm 만 쓰므로 문제없다고 판단). 또 이 PATH 오염은 **이 머신의 환경 결함**이고 GitHub Actions(node 24, 상위 `node_modules` 없음)에서는 애초에 안 나므로, CI 에서 고정 효과를 실제로 관측하지는 못했다 — CI 경로는 `npm ci` → `npm test --silent` 가 no-op 가드로 도는 것까지만 로컬에서 흉내 내 확인했다.
- **일부러 하지 않은 것**: `build`·`dev`·`preview` 스크립트는 여전히 `.bin` 심링크(`tsc`/`vite`)에 의존한다 — 범위를 3개 파일로 묶기 위해 남겼고 ideas.json 에 [2/1/S] 로 적었다. `/home/hkjang/node_modules` 는 사용자 홈이라 손대지 않았다. `release.yml`·`package-lock.json`·프런트 소스·보호 경로는 무수정(`git status` 로 확인).
- 다음 역할이 조심할 것: 누군가 `"${npm_node_execpath:-node}"` 를 바레 `node` 로 "정리" 하면 이 머신에서 web 테스트가 즉시 `markAsUncloneable` 로 다시 깨진다(되돌려 양방향 확인함). 이유는 커밋 메시지와 `Makefile` 주석에 남겼다. `go test ./...` 는 Postgres 없이 돌아 통합 테스트가 조용히 skip 되고, 이 base(60fca20)에는 `internal/mattermost/client_test.go` 가 아직 없어 `[no test files]` 로 보고된다.
- [러너 07:45] brief accepted — 채택 — 과제서의 근거(`web/package.json:9` 만 `.bin` 의존 + 가드 없음, 나머지 전부 패키지 경로, `release.yml` 은 web 테스트를 부
- [러너 07:46] verify passed — 검증 7개 통과 (auto)
- [러너 07:46] pr created — https://github.com/hkjang/mmcp/pull/1
- [러너 07:46] guard held — .github/workflows/ci.yml 
