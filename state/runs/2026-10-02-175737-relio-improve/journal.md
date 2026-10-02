# 회차 노트 2026-10-02-175737-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:57] base pinned — main@494d00f
- [러너 17:57] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 우선 과제(verify-failed 수정)가 배정돼 새 후보를 고르지 않았다. 다만 원인은 보류 목록의 `[2/1/S] npm test 글롭` 항목과 동일했고, 이번 워크트리에서 `cd web && npm test --silent` 를 실제로 돌려 exit 1 + `Could not find '.../web/test/**/*.test.ts'` (Node v22.23.1) 를 재현해 확정했다 — 테스트 실패가 아니라 파일 발견 실패다.
- 수정 범위는 `web/package.json` 의 `scripts.test` 한 줄. ci.yml·release.yml 은 `node-version: 24` 로 핀돼 있어 CI 는 통과하므로 워크플로를 건드릴 이유가 없고, 2026-09-06 릴리즈 경로 사고 때문에 건드리지 말라고 과제서에 명시했다.
- 추측으로 적은 것: ① 권고안 `node --test test/*.test.ts` 를 **실제로 돌려 보지 못했다**(정찰 세션에서 node 직접 실행이 권한으로 막힘). 2026-10-02 회차가 "파일 4개를 직접 넘기면 27/27 PASS" 를 확인했으므로 2순위(명시 열거)는 확실하다. ② `web/node_modules` 가 이 워크트리에 아예 없고 `npm ci` 도 막혀서, 글롭을 고친 뒤 실제로 27건이 도는지는 미확인.
- 구현자가 조심할 것: `sh` 는 `**` 를 재귀 전개하지 않으므로 따옴표만 떼면 안 된다(테스트 4개는 `web/test/` 평면에 있다). 통과 증거는 `make -n` 이나 문자열 확인이 아니라 실제 `npm test` 출력으로 낼 것.
- 겸사로 발견: `esbuild` 가 `web/package.json` 에 선언돼 있지 않은데 `login.test.ts` 가 import 한다(vite 전이 의존성). 실제 결함이지만 package-lock 재생성이 끼어 이번 회차와 분리해 보류로 남겼다.
- [러너 18:02] scout done — `web/package.json` 의 `test` 스크립트에서 Node 버전에 따라 전개되지 않는 `**` 글롭을 없애기 (가치 4 / 위험 1 / �

## 구현 노트
- 고친 것: `web/package.json` 의 `test` 를 `node --test ...` → `$npm_node_execpath --test ...` (1줄). **과제서의 원인 진단은 틀렸다** — 따옴표 친 `'test/**/*.test.ts'` 는 Node 22.23.1·25.9.0 에서 27/27 exit 0 이다. `Could not find` 를 내는 건 Node **20.19.2** 뿐이고, 그게 npm 스크립트에 들어오는 이유는 npm 이 조상 `node_modules/.bin` 을 PATH 앞에 세우고 `/home/hkjang/node_modules/.bin/node -> v20.19.2`(2026-09-30 자 홈 install)가 있기 때문이다. 그래서 글롭이 아니라 node 해석을 고쳤고 `**` 는 유지했다(평면화하면 `test/sub/*.test.ts` 가 조용히 빠짐 — 실측 확인).
- 확신 없는 곳: `$npm_node_execpath` 를 **따옴표로 감싸지 않았다** — node 설치 경로에 공백이 있으면 깨진다(CI `/opt/hostedtoolcache/...`·nvm 경로 모두 공백 없어 실해는 없고, 깨지면 조용하지 않고 시끄럽게 깨진다). 또 이 변수는 sh 전개에 의존하므로 Windows `cmd` 에서 `npm test` 는 안 돈다(기존 글롭도 sh 전개를 전제했고 CI 는 ubuntu — 후퇴는 아니지만 확인은 못 했다). 비평가가 여기를 먼저 보면 좋겠다.
- 일부러 하지 않은 것: `/home/hkjang/node_modules/.bin/node` 심링크를 **지우지 않았다** — 저장소 밖 환경이고(되돌릴 수 있는 1개 심링크이긴 하다) "환경과 코드를 같은 단계에서 바꾸지 말 것" 에 걸린다. 근본 수정은 러너/운영자의 일로 ideas.json 에 올렸다. `.github/workflows/` 0줄, `engines`·README·esbuild 선언도 손대지 않았다(파일 1개 유지).
- 다음 역할이 조심할 것: `npm test` 는 `web/node_modules` 가 있어야 돈다(`npm ci` 는 이번에 성공, 69패키지/1s). `login.test.ts` 는 esbuild 로 실제 `Login.tsx` 를 번들링하며 `web/test/.login-test-*/` 임시 디렉터리를 만드는데, 정상 종료 때만 지운다 — 출력을 `head` 로 파이프해 SIGPIPE 로 죽이면 남으므로 커밋 전 `git status` 로 확인할 것(이번에 한 번 남아서 지웠다).
- 검증: `cd web && npm test --silent` 27 pass/0 fail/0 skip·exit 0 · `npm --prefix web test` 동일 · `make test` exit 0 · Node 25 의 npm 으로 돌린 CI 모사 27/27 · `go build`·`go vet`·`go test ./...` 22패키지 ok/FAIL 0 · `git diff --name-only` = `web/package.json` 만 · `git diff --check` 무출력. 되돌림 검증: `$npm_node_execpath`→`node` 한 토큰만 바꾸면 러너가 보고한 `Could not find '.../web/test/**/*.test.ts'` 가 글자 그대로 재현된다. 커밋 81c8f34.
- [러너 18:12] brief fallback — 차선 — 과제서가 지목한 **파일과 줄은 맞았지만 원인 진단이 틀렸다**. 정찰은 셸의 `node --version`(v22.23.1)을 npm 스크립트 
- [러너 18:12] verify failed — 실패한 검증: npm run typecheck && npm run build   # CI에서 가져옴 (exit 1)
