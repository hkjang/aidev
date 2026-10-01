# 회차 노트 2026-10-02-001239-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:12] base pinned — main@98c5081
- [러너 00:12] autonomy release — 

## 정찰 노트
- 왜 이 과제인가: 실패한 검증은 `npm run lint` 이고, 지난 두 회차가 `cd web && npm run lint` 는 clean main 에서도 exit 0 임을 실행으로 확인했다. 남은 재현 경로는 저장소 루트 실행(`npm error Missing script: "lint"`) 하나뿐인데 2026-10-01 과제서가 루트 `package.json` 생성을 금지해 원인을 그대로 둔 채 두 번 연속 verify-failed 가 됐다 — 이번엔 그 금지를 뒤집는 것이 과제다. 린트 규칙을 끄거나 ignores 로 빼는 방향(완화)은 금지로 명시했다.
- 확신 없는 곳(추측): **이 세션은 `npm` 실행이 권한으로 막혀 아무것도 직접 재현하지 못했다.** "러너가 저장소 루트에서 명령을 돈다" 는 지난 회차 원장의 실행 기록에서 끌어온 추론이며 러너 설정은 못 봤다. 그래서 과제서에 0단계(먼저 재현, (1) web 안이 exit 1 이면 과제 폐기·그쪽을 고침)를 넣었다. `npm --prefix web run lint` 가 cwd 를 web 으로 바꾸는지도 미확인이라 실행으로 판정하라고 적었다.
- 구현자가 조심할 것: `workspaces` 선언 금지(web/package-lock.json·CI 캐시 전제가 깨진다), 루트 스크립트 재귀(루트 cwd 에서 같은 이름을 다시 부르지 말 것, 테스트에서 루트 `npm test` spawn 금지), `.gitignore` 에 루트 `node_modules/` 추가(없으면 빌드 후 git status 가 더러워진다), ci.yml·web/package-lock.json 무변경.
- 머지되지 않은 선행 수정 2건을 확인했다: `1d63b7c`(Node 런처)와 `5ed6876`(.js/.mjs 린트 사각지대)은 base 98c5081 에 없다 — `web/eslint.config.js` 를 직접 열어 `files: ['**/*.{ts,tsx}']` 블록 하나뿐임을 확인했다. 루트 `npm test` 가 undici `markAsUncloneable` 로 깨지면 1d63b7c 를 되살리라고 과제서의 선택 작업에 적었다(프로덕션 최대 5파일).
- [러너 00:20] scout done — 릴리즈 검증 `npm run lint` exit 1 복구 — 저장소 루트에 web 검증을 위임하는 `package.json` 을 두어 CI 의 `defaults.

## 구현 노트
- 무엇/왜: 저장소 루트에 `package.json` + `scripts/web-run.mjs` 를 두어 `npm run lint|test|build` 가 web 의 같은 스크립트에 위임한다. 원인은 실행으로 확정했다 — 루트에서 `npm prefix` 가 `/home/hkjang` 를 반환(npm 이 저장소를 벗어나 홈의 package.json 을 집어 `Missing script: "lint"` → exit 1). 수용 기준 2 의 환경 분기가 발동해 `1d63b7c`(Node 런처 + `engines.node`)도 cherry-pick 했다. 커밋 f927cea.
- 확신 없는 곳: (a) **러너가 정말 저장소 루트에서 이 명령을 돌리는지는 끝까지 직접 확인하지 못했다**(러너 설정이 저장소 밖). 루트 exit 1 재현과 `npm prefix` 결과는 확실하지만, 러너의 실패가 이 경로였다는 연결은 여전히 추론이다. (b) CI 는 Node 24 인데 이 환경은 22.23.1 이고, 루트 위임은 CI 에서 돌지 않는다(web 잡은 여전히 `working-directory: web`) — 즉 이 변경은 CI 로 검증되지 않는 경로다. (c) 남은 미검증은 없다 — `npm_execpath`/`npm_node_execpath` 를 모두 지운 폴백(`npm` + `shell:true`)은 `env -u` 로 실제 실행해 exit 0 을 확인했고, 허용 목록은 알 수 없는 스크립트·인수 누락 모두 exit 64 를 확인했다.
- 일부러 하지 않은 것: `.github/workflows/ci.yml`·`web/package-lock.json`·`web/eslint.config.js`·`web/vite.config.ts`·`web/vitest.config.ts`·keep-dist 테스트 무변경(전부 `git status` 로 확인). `workspaces` 선언·루트 `package-lock.json`·루트 의존성 없음. 루트 `scripts/*.mjs` 를 린트 대상에 넣지 않았다(eslint 는 cwd=web 으로만 돈다 — 새 보류 아이디어로 적었다).
- 다음 역할이 조심할 것: 구현 중 실제로 밟은 함정 — 위임 스크립트가 자식 npm 에 `process.execPath` 를 넘기면 PATH 가로채인 Node 20 심이 web 까지 전파돼 런처가 `engines.node` 하한 미달로 멈춘다(루트 `npm test` exit 1). 그래서 `npm_node_execpath` 를 우선 쓴다 — 이 한 줄을 되돌리면 테스트가 깨진다. `web/src/test/root-npm-scripts.test.ts` 는 실제 `npm run lint` 를 spawn 하므로 느리고(타임아웃 180s) 루트 `npm test` 를 절대 spawn하면 안 된다(재귀). 그 테스트는 `npm` 이 PATH 나 `npm_execpath` 로 접근 가능해야 돈다.
- [러너 00:29] brief accepted — 채택 — 0단계 재현이 과제서의 분기 조건((1) exit 0 · (2) exit 1)을 그대로 만족했고 미확인 가정("루트 실행이 원인")을 `npm 
- [러너 00:30] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
