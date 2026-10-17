# 회차 노트 2026-10-18-082857-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:29] base pinned — main@745dcf3
- [러너 08:29] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00

## 정찰 노트
- 지정된 실패를 베이스에서 그대로 재현했습니다: 루트 `npm test --silent` → `run-on-supported-node: vitest is not installed …` / EXIT=1, `web/node_modules` 없음. 제품 버그가 아니라 **루트 포워더가 설치를 전제로만 동작**하는 것이고, 같은 실패가 세 회차의 verify-failed 였으므로 다른 후보(toolchain↔Dockerfile 드리프트 검사, preformat)를 제치고 이것을 1순위로 적었습니다.
- 추측으로 적은 것(= 미확인): ① 이 샌드박스가 `npm ci --prefix web` 를 승인하지 않아 **설치 자체를 실행해 보지 못했습니다** — 구현자는 이것을 가장 먼저 확인하고, 네트워크가 없으면 차선 후보로 갈 것. ② 기준선 시험 수(24파일/260시험)는 2026-10-16 기록이며 이번에 재측정하지 못했습니다.
- 구현자가 조심할 것: `web/package.json` 에는 `pretest` 를 달지 말 것(그쪽은 계속 거절해야 하고, 그 경계가 "느슨하게 만들지 않았다" 의 근거). `npm ci` 는 node_modules 를 지우므로 해석 성공 시 **절대로 부르지 말 것**. `run-on-supported-node.mjs`·`require-installed.mjs` 는 열어 읽기만 하고 통합하지 말 것(운영자 규칙 3번).
- 프로필은 2026-10-16 판이 지금 베이스(main@745dcf3)와 일치해 다시 쓰지 않았습니다 — 이번에 직접 확인한 것만 덧붙이면, `ci.yml` 의 프런트엔드 절은 `- run: npm ci` + `working-directory: web` 로 **자기 손으로 설치**하고 루트 스크립트를 쓰지 않으므로 이번 변경은 CI 에 영향이 없습니다.
- [러너 08:33] scout done — 수정 과제 — 러너의 검증 명령 `npm test --silent`(저장소 루트)가 새 체크아웃에서 **항상** 실패한다: 루트 �

## 구현 노트
- 루트 `package.json` 에 `pretest` 한 줄 + 새 `web/scripts/ensure-install.mjs`(세 번째 형제, 앞의 둘은 거절하고 이것만 복구한다). 베이스 EXIT=1 → node_modules 치운 트리에서 루트 `npm test --silent` EXIT=0(24파일/255시험), 설치된 트리 재실행 24s→8s·`web/node_modules` inode 불변(= `npm ci` 재실행 없음). 커밋 72e131a.
- **확신 없는 곳**: ① 되돌림 드릴 중 **전체 suite 한 번에서 `Tests 1 failed | 254 passed` 가 났고 그때 로그를 `tail -4` 로 잘라 어느 시험인지 잃었습니다.** 같은 시나리오를 그 뒤 7회(전체 4회 + 신규 파일 10회 + scripts/ 3회) 돌려 전부 초록이라 재현하지 못했고, 원인을 모릅니다 — "flake" 라고 단정하지 않습니다. 비평가는 여기를 먼저 보세요(의심 후보: `npm ci` 직후 부하에서 npm 자식을 여럿 띄우는 `require-installed.test.mjs` 의 120s 케이스들, testTimeout 20s). ② `spawnSync('npm', …)` 는 셸 없이 PATH 의 `npm` 을 찾으므로 **Windows 에서는 동작을 확인하지 않았습니다**(이 저장소의 기존 시험 코드도 같은 형태를 씁니다).
- **일부러 하지 않은 것**: `web/package.json` 에 `pretest` 를 달지 않았습니다 — `make test-web`·CI·`npm --prefix web test` 는 계속 거절해야 하고, 그 경계를 실행으로 찍어 뒀습니다(`impl-boundary.log`). `run-on-supported-node.mjs`·`require-installed.mjs` 는 열어 읽기만 했고(운영자 규칙 3번) 보호 경로·`VERSION`·락파일은 건드리지 않았습니다. 차선 후보(toolchain↔Dockerfile 드리프트 검사)는 1순위가 성립해 쓰지 않았습니다.
- **다음 역할이 조심할 것**: `web/scripts/ensure-install.test.mjs` 는 실제 자식 프로세스 + PATH 앞의 `npm` 셸 셤으로 돌아가므로 POSIX 셸이 필요하고, 픽스처에 락파일을 복사하지 않아 셤을 놓쳐도 네트워크로 새지 않습니다. 기준선은 **23파일/252시험**이었고(과제서의 24/260 은 미머지 브랜치 수치) 새 시험 3개로 **255**. Go·DB·Docker·E2E 는 Go 소스 0줄이라 돌리지 않았습니다.
- [러너 08:43] brief accepted — 채택 — 과제서가 지정한 파일 세 개(프로덕션 2 + 시험 1)만 건드렸고, 과제서의 근거는 지금 코드와 전부 맞았습니다(루�
- [러너 08:43] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 판정 **approve** (risk low, 차단 없음). 구현자가 의심한 미동정 1실패를 추적했습니다: 그것은 13회 중 1회가 아니라 **이 변경이 새로 여는 경로(설치 없음 → pretest 가 npm ci → 전체 suite)를 돌린 단 2회 중 1회**였습니다(impl-e2e.log RUN1 초록 / impl-reverted.log 'restored' 1실패). 나머지 초록은 전부 이미 설치된 트리 = 사소한 조기 exit 분기.
- 그래서 그 시나리오를 `rm -rf web/node_modules && npm test` 로 3회 직접 재현 → 3회 모두 EXIT=0·24파일/255시험. 설치된 트리 전체 suite 8회도 255 초록. 신규 경로 누적 4초록/1적색, 재현 불가·원인 미동정 — 귀속 증거가 없어 차단하지 않습니다. **다음 회차가 fresh-clone 루트 `npm test` 에서 1시험 실패를 보면 flake 로 넘기지 말고 이 자리로 돌아오세요.**
- 승인이어도 남는 우려: ① `package.json:8` 의 `pretest` 가 이번 수정의 실체인데 자동 시험이 없다 — 형제 `require-installed.test.mjs:11-15` 는 npm pre* 훅을 통해 구동하라고 명문화해 뒀고 새 시험은 `node <script>` 직접 호출로 그 관례에서 벗어났다(가장 먼저 메울 구멍). ② 픽스처의 vitest 대역에 exports 맵이 없어, 실물 vitest 가 `"./package.json"` 을 빼는 날 ensure-install.mjs:52 가 ERR_PACKAGE_PATH_NOT_EXPORTED 로 rethrow 되는 경로를 시험이 지나지 않는다. ③ `spawnSync('npm', …)` 는 run-on-supported-node.mjs 가 `node` 에 대해 길게 방어하는 PATH 셰도잉을 `npm` 에 대해선 방어하지 않는다(시험 쪽은 그 위험을 알고 걸러낸다).
- 확인한 것: diff 3파일 전수(보호 경로·Go·락파일 무접촉), typecheck·lint·check-i18n·전체 suite 직접 실행, red 로그 2종이 실재하는 적색임, CI(ci.yml:131 working-directory: web)·Makefile:28 이 루트 포워더를 쓰지 않아 경계가 실제로 닫혀 있음. 못 본 것: Go·DB·Docker·E2E(Go 0줄), Windows 네이티브(새 시험은 이 저장소 최초로 `#!/bin/sh` 셤을 요구 — CI 는 ubuntu 라 무해).
- 릴리즈 노트용: 개발자 편의 변경이며 제품 동작·API·DB 무변화. VERSION 미변경이 맞습니다. 되돌리기는 `git revert 72e131a` 로 완전.
- [러너 08:52] review approved — 리뷰 승인 (risk=low)
- [러너 08:52] pr created — https://github.com/hkjang/umm/pull/174
- [러너 08:56] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 
