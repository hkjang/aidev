# 회차 노트 2026-10-01-155204-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:52] base pinned — main@8604422
- [러너 15:52] autonomy release — 

## 정찰 노트
- 우선 과제가 자동 배정돼 후보를 고를 여지는 없었다 — 프런트엔드 `npm test` 가 깨진 채로는 어떤 과제도 릴리즈까지 못 가므로 차선 후보(지표 배선 고정)도 이 과제 뒤에 선다.
- 진단의 기계적 근거는 파일로 확인했다: `jsdom/lib/api.js:12` 의 모듈 로드 시점 `require("undici")` → `undici/lib/web/cache/cachestorage.js:20` → `undici/lib/web/webidl/index.js:5` 의 `node:worker_threads` 구조분해. 의존 버전은 v0.9.92~96 동안 안 움직였다(릴리즈 커밋의 version 줄만).
- **실행 재현은 못 했다(미확인)**: 이 worktree 에 `web/node_modules` 가 없고 `npm ci`·`node -e` 가 이 세션 권한에서 거부됐다. 그래서 과제서 기준 1을 분기점으로 뒀다 — `typeof require("node:worker_threads").markAsUncloneable` 이 `"function"` 이면 이 진단은 틀렸으니 의존성을 건드리지 말고 실제 스택에서 다시 진단할 것.
- 구현자가 조심할 것: Node 핀(22.23.0)과 `release.yaml` 은 건드리지 말 것(25.9.0 에서는 테스트 19개가 따로 깨진다), `test.pool` 우회 금지, 테스트 단언 완화 금지, 레지스트리가 막히면 반쯤 바뀐 lockfile 을 남기지 말고 멈출 것.
- [러너 15:57] scout done — 프런트엔드 테스트가 워커 시작부터 깨지는 것을 고치기 — `jsdom`→`undici` 의존 사슬을 이 저장소가 고정�

## 구현 노트
- 과제서의 진단은 반증됐다(기준 1: `markAsUncloneable` 은 Node 22.23.1 에서 `function`). 의존성·lockfile 은 건드리지 않았다. 진짜 원인은 npm 이 **cwd 의 모든 조상** `node_modules/.bin` 을 PATH 앞에 붙이는 것 + 2026-09-30 22:24 에 `/home/hkjang/node_modules` 에 생긴 `node`(v20.19.2) 심링크 → `.bin/vitest` 의 `#!/usr/bin/env node` 가 Node 20 을 집었고, Node 20 엔 `markAsUncloneable`(22.10.0 추가)이 없다. 고친 것은 `web/package.json` 스크립트 여섯 줄에 `PATH="${npm_node_execpath%/*}:$PATH"` 접두사뿐(프로덕션 파일 1개).
- **확신 없는 곳**: ① 이 접두사는 POSIX `sh` 문법이라 **Windows(cmd.exe)에서는 안 돈다** — 이 저장소는 Makefile·Go·ubuntu CI 기반이고 Windows 개발을 지원한다는 기록을 못 찾았지만 확인하지는 않았다. ② `preview` 스크립트만 실행 검증을 못 했다(서버가 떠서 종료하지 않는다) — 같은 `vite` 심을 쓰는 `dev` 는 `node-v22.23.1` 을 찍어 확인했다. ③ GitHub 러너에서 이 접두사가 무해하다는 것은 **추론**이다(오염된 조상이 없으면 이미 맞는 node 를 맨 앞에 한 번 더 놓는 것뿐) — 실제 러너에서 돌려 보지는 못했다. 릴리즈 역할이 여기를 먼저 볼 것.
- **일부러 안 한 것**: `engines` 추가(lockfile 루트에 `engines` 가 없어 재생성이 따라붙고, 기준 1 이 금지했으며 경고만 내서 이 결함을 못 잡는다) / undici·jsdom 다운그레이드(undici 8.10.0 은 무죄 — 잘못된 Node 를 숨기는 일) / Node 핀·`release.yaml`·`test.pool`·테스트 단언 변경 / `/home/hkjang/node_modules` 의 `node` 삭제(저장소 밖 되돌릴 수 없는 환경 변경 — 운영자 규칙).
- **vitest 테스트를 새로 추가하지 않은 이유**: 워커가 수집 **전에** 죽어서 어떤 vitest 수준 테스트도 고치기 전에 실패-후-통과를 보일 수 없고, 파일을 더하면 과제서 기준 2 가 못박은 29파일/161테스트 기준선이 깨진다. 재현은 게이트 명령 자체로 했고 되돌림까지 확인했다(`npm exec -c 'vitest run'` 이 옛 형태로 26 errors 재현).
- **다음 역할이 조심할 것**: 이 섀도잉은 worktree 경로에서만 일어난다 — `/mnt/c/...` 본 체크아웃에는 오염된 조상이 없어 거기서 `npm test` 는 고치기 전에도 녹색이었다. 그러니 "내 쪽에선 되는데" 로 이 커밋을 불필요하다고 판단하지 말 것. `$HOME` 이 임시라는 것도 방어가 안 된다(npm 은 `$HOME` 이 아니라 cwd 를 거슬러 올라간다). Go 쪽 테스트는 `eval "$(scripts/test-services.sh)"` 를 같은 셸에서 먼저 해야 돈다.
- [러너 16:32] brief fallback — 차선 — 과제가 지목한 **증상과 수용 기준은 그대로 채택했지만 원인 진단은 기각**했다. 과제서가 분기점으로 지정한 �
- [러너 16:32] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현자가 의심한 세 곳을 모두 직접 확인했다. ① Windows: README:203 이 Bash·Make·OpenSSL 을 이미 요구하므로 cmd.exe 는 지원 경로가 아니다 — 결함 아님. ② `preview`: 미검증이나 같은 `vite` 심을 쓰는 `build`·`dev` 가 돌았고 PATH 접두사가 심 선택에 영향을 주지 않는다. ③ 러너 무해성: `npm_node_execpath` 가 npm 10.9.8 에서 실제로 설정됨을 확인했다(`[/home/hkjang/.nvm/.../bin/node]`).
- 실행으로 재현·반증 모두 했다: 옛 형태 `npm exec -c 'vitest run'` → **26 errors**, 새 형태 `npm test` → **29파일/161테스트 통과**, `npm run lint` 무경고, `npm run build` 성공. `/home/hkjang/node_modules/.bin/node` → **v20.19.2**(9/30 22:24) 도 확인. 커밋 메시지의 주장과 코드가 일치한다.
- **남는 우려 1 (릴리즈가 알 것)**: 접두사는 npm 의 node 디렉터리를 `node_modules/.bin` **앞**에 놓으므로, 전역 설치된 `tsc`·`eslint`·`vite`·`vitest` 가 있는 개발자 기계에서는 고정된 로컬 버전 대신 전역 버전이 조용히 쓰인다. 이 기계의 nvm bin 에는 node/npm/npx/corepack/yarn/pnpm/codex 뿐이라 지금은 안 터진다. 더 좁은 대안은 PATH 를 건드리지 않고 `"$npm_node_execpath" ./node_modules/vitest/vitest.mjs run` 식으로 인터프리터만 지정하는 것.
- **남는 우려 2**: `npm_node_execpath` 가 비면(npm 이외 러너) `PATH=":$PATH"` 가 되고 POSIX 에서 빈 항목은 **cwd** 다 — `web/` 의 동명 파일이 먼저 실행된다(실측 확인). npm 경로로는 도달 불가라 차단은 아니다. `${npm_node_execpath:+${npm_node_execpath%/*}:}` 가드가 공짜 방어. 또 `build` 만 `sh -c` 라 `npm run build -- <flag>` 의 인자가 `$0` 로 조용히 삼켜진다(현재 호출처 없음: Makefile:81,91 · ci.yaml:67 · release.yaml:66 전부 인자 없음).
- **범위 밖 기존 조건**: `webui/dist/index.html` 이 세션 시작부터 더러웠다(커밋 메시지는 복원했다고 적었다). 재빌드해 보니 main 에 커밋된 dist 의 `index-*.js`·`mui-*.js` 해시가 지금 빌드 산출물과 다르다 — 이 PR 탓이 아닌 main 의 **낡은 산출물**이다. 내가 `git checkout -- webui/dist` 로 되돌려 워크트리는 깨끗하다. 다음 회차가 dist 재생성 커밋을 따로 볼 가치가 있다.
- [러너 16:36] review approved — 리뷰 승인 (risk=low)
- [러너 16:36] pr created — https://github.com/hkjang/ReSSO/pull/34
- [러너 16:45] ci passed — 검사 2개 모두 success
- [러너 16:45] merge done — c43bf5e
- [러너 16:54] release missing — 릴리즈 결과 없음/손상: missing
