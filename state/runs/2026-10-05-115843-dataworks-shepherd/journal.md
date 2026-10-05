# PR 처리기 노트 2026-10-05-115843-dataworks-shepherd — dataworks PR #34
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-05-111723-dataworks-improve)
# 회차 노트 2026-10-05-111723-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:17] base pinned — main@98c5081
- [러너 11:17] autonomy release — 
- [러너 11:26] scout timeout — 단계 제한 시간 초과
- [러너 11:26] scout done — 릴리즈 검증 게이트 복구 — `cd web && npm test --silent` 과 루트 `npm run lint` 를 "구버전 node 가 PATH 를 가로챈 상

## 구현 노트
- 무엇을/왜: 릴리즈 게이트의 두 명령을 세 원인에서 고쳤다 — (1) `web/node_modules` 부재 시 런처가 ENOENT 로 죽던 것(→ 부재 보고 + bin 없을 때만 `npm ci` 1회, 설치 실패는 그 종료코드 그대로), (2) npm 자신이 Node 20 심으로 뜨면 후보 2개가 다 미달이던 것(→ NVM/usr 후보 확장), (3) **과제서에 없던 원인**: `--silent` 가 `npm_config_loglevel=silent` 를 물려줘 `root-npm-scripts.test.ts` 가 중첩 npm 배너를 못 보고 `expected '' to contain 'eslint'` 로 실패 — 이것이 의존성이 설치된 러너 환경의 exit 1 을 설명한다. 커밋 a67c06a(cherry-pick f927cea)·9d43941.
- **확신 없는 곳**: 폴백 후보를 과제서의 "버전 내림차순" 대신 **오름차순(하한 넘기는 최저)** 으로 바꿨다. 근거는 실측(Node 25.0.0·25.9.0 에서 `silent-sso.test.ts` FAIL, 22.23.1·23.11.1 PASS)이지만 그 테스트가 **왜** Node 25 에서 깨지는지는 조사하지 않았다 — ideas.json 에 신규 후보로 남겼다. CI(Node 24)는 폴백 경로를 타지 않으므로 영향 없음. 또 러너 환경에 네트워크/npm 캐시가 있는지는 여전히 확인 불가라 `npm ci` 실패 경로는 **실제 네트워크 차단으로는 검증하지 못했고** 스텁 npm(exit 7)으로만 확인했다.
- 일부러 안 한 것: `5ed6876`(.js/.mjs 린트 사각지대)은 과제서 지시대로 미포함. `web/src` 제품 코드와 `internal/` Go 코드 무변경(Go 변경 0). 보호 파일 5종 무변경, `engines.node` 도 그대로(상한 추가는 지원 범위 선언이라 자율 변경 제외).
- 다음 역할 주의: 새 런처 테스트 5건은 tmp 에 가짜 web 루트를 만들고 **실제 프로세스를 spawn** 한다(DB 불필요, `/bin/sh` 와 쓰기 가능한 tmpdir 필요, 각 120s 타임아웃). `installs dependencies…` 사례는 스텁 npm 이라 네트워크를 쓰지 않는다. `falls back to…` 사례는 현재 node 의 major+1/+2 를 가짜 nvm 버전으로 만들므로 node 버전과 무관하게 돈다.
- [러너 11:40] brief accepted — 채택 — 0단계 재현이 과제서의 예측(127 과 1 의 구분, ENOENT → exit 1, cherry-pick 무충돌)을 모두 확증했고 수용 기준 1~6 을 �
- [러너 11:41] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인: 게이트 4종을 실제로 돌려 전부 exit 0(`cd web && npm test --silent` 11파일/36테스트, 루트 test·lint·build, build 후 git clean). `npm_config_loglevel=silent` 로 루트 lint 출력이 0자가 되는 것을 재현해 --silent 수정이 load-bearing 임을 확증했고, 런처 테스트 6건이 실제 spawn 으로 수정 경로를 지나는 것도 확인했다. `engines` 추가가 `npm ci` lock 동기 검사를 깨지 않음은 임시 디렉터리 `npm ci --dry-run`(exit 0)으로 확인.
- 구현자의 '확신 없는 곳' 확증: `/usr/bin/node`(v25.9.0)에서 `silent-sso.test.ts:77` 이 `expected true to be false` 로 실패한다. **선존 결함**(f596392)이고 이번 범위 밖이지만, CI node-version 이 24 를 넘기면 그 순간 게이트가 깨진다 — 다음 회차 최우선. 폴백 오름차순 선택은 이 측정에 비추어 타당하고 CI 는 폴백 경로를 타지 않는다.
- 승인이어도 남는 우려(릴리즈 노트용): `web/eslint.config.js` 의 files 가 `**/*.{ts,tsx}` 뿐이라 신규 `.mjs` 2개는 린트 룰이 0개다(--print-config 로 확인). `scripts/web-run.mjs:12-14` 머리 주석은 42행 동작과 모순(36-40행이 맞음). `parseVersion` 은 engines 상한을 무시한다.
- 못 본 것: 실제 네트워크 차단 하의 `npm ci` 실패(구현자와 같이 스텁 npm 으로만), Node 24 CI 실행, PostgreSQL·Playwright e2e. Go 변경 0건이라 Go 테스트는 미실행.
- 보안·법무 차단 사유 없음: 신규 엔드포인트·인가 경로·비밀값·개인정보·신규 의존성 모두 0, `shell: true` 2곳은 인자가 고정 배열이라 주입 경로 없음. revert 로 완전 복구 가능(마이그레이션·외부 상태 변경 없음).
- [러너 11:48] review approved — 리뷰 승인 (risk=low)
- [러너 11:48] pr created — https://github.com/hkjang/dataworks/pull/34
- [러너 11:51] ci failed — 성공이 아닌 검사: Web lint / test / build=failure · 실패한 검사: ? 잡: Web lint / test / build 

## 수리 노트
- 맞았던 것: CI 가 정말 깨져 있었다 — Test 스텝만 11초에 failure(Install·Lint success, Build skipped). 틀렸던 것: 구현 노트의 "새 런처 테스트 5건은 각 120s 타임아웃" — 실제로는 `test-script-interpreter.test.ts` 의 **첫 사례만** 120s 였고 런처 5건과 `root-npm-scripts` 3건은 vitest 기본 5000ms 였다.
- 원인: `runs the web eslint from the repository root` 가 5000ms 안에 npm → npm → eslint 를 끝내야 하는데 이 저장소의 전체 린트는 CI Lint 스텝만으로 5초다. 재현: Node 24.21.0 + 깨끗한 체크아웃 + `taskset -c 0` → `Test timed out in 5000ms`. 코어가 많으면 4029ms 로 간신히 통과해 로컬에선 녹색이었다 = 하드웨어 속도에 걸린 게이트였다.
- 고친 방법: 프로세스를 띄우는 사례에만 명시적 제한시간(`spawnSync` 상한과 같은 180s/120s)을 붙였다. 단언·검증 명령·워크플로는 무변경이고 그래서 통과가 "실제로 eslint 를 끝까지 돌린 7330ms" 로 남는다. 커밋 cf3131c.
- 확신 없는 곳: CI 로그 본문은 토큰이 없어 못 받았다(logs API 403). 스텝 결과·소요시간과 로컬 재현으로만 좁혔으니, 러너에서 다른 사례가 같이 넘어갔을 가능성은 배제하지 못한다 — 다만 그 경우도 같은 수정이 덮는다.
- 그대로 남긴 것: 비평가가 적은 선존 결함 2건(Node 25 의 `silent-sso.test.ts`, `.mjs` 린트 룰 0개)과 `scripts/web-run.mjs:12-14` 주석/동작 불일치. 범위 밖이라 손대지 않았다.

## 심사 노트
- 확인: 게이트 4종을 루트·web 두 진입점에서 실행해 전부 exit 0(36테스트, build 후 git clean). 구현자·비평가가 못 봤다던 신규 워크트리 경로를 **스텁 아닌 진짜 npm ci** 로 양쪽(web-run.mjs:61, run-with-supported-node.mjs:162) 모두 끝까지 통과시켰다.
- 테스트가 load-bearing 임을 실측: PATH 앞에 node probe 를 끼우고 수정 전 스크립트 문자열을 돌리면 probe 가 `.bin/vitest` 를 띄워 test-script-interpreter.test.ts:65 가 깨지고, 수정 후에는 런처 줄만 남는다. `taskset -c 0` 에서 root-npm-scripts 가 13.80s → vitest 기본 5000ms 가 CI 실패 원인이었음도 재현해 cf3131c 의 유효성을 확증했다.
- 못 본 것: CI 의 Node 24 실행, 실제 네트워크 차단 하의 npm ci 실패(exit 7 스텁만), Docker 이미지 빌드, PostgreSQL·Playwright e2e. Go 변경 0건이라 Go 잡은 무영향.
- 남은 선존 결함(범위 밖): /usr/bin/node v25.9.0 에서 silent-sso.test.ts 실패 — CI node-version 을 24 에서 올리는 순간 게이트가 깨진다. 신규 .mjs 2개는 eslint 룰 0개.
- 권고: approve / merge (risk=low). 보호 파일·인가 경로·비밀값·개인정보·신규 의존성 0건, 마이그레이션 없어 revert 로 완전 복구. 주석/동작 불일치 3건과 interpreter 테스트의 spawnSync timeout 누락은 차단 사유 아닌 notes 로 남겼다.
