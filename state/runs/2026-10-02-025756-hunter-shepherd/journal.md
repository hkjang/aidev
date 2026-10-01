# PR 처리기 노트 2026-10-02-025756-hunter-shepherd — hunter PR #16
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-02-004739-hunter-improve)
# 회차 노트 2026-10-02-004739-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:47] base pinned — main@f0f7667
- [러너 00:47] autonomy release —

## 정찰 노트
- 우선 과제가 지정돼 있어 새 아이디어를 고르지 않았다. `npm test` 를 그대로 돌려 `node: bad option: --experimental-strip-types` / exit 9 를 1회로 재현했고, 원인은 `web/package.json:10` 한 줄(저장소 내 유일한 출현 위치)이라 다른 후보를 볼 필요가 없었다.
- 추측으로 적은 것: 플래그를 **빼면** 통과한다는 부분. 이 세션에 `node` 직접 실행 권한이 없어(`node --test …` 가 승인 차단) 검증하지 못했다. 근거는 `web/src` 에 enum/namespace 등 비지울수있는 TS 구문이 없다는 grep 과 Node 22.18/23.6 이후 타입 스트리핑 기본 동작이라는 일반 지식뿐이다.
- 구현자가 조심할 것: package.json 을 고치기 **전에** 테스트 한 파일을 플래그 없이 돌려 확인할 것. 안 되면 `node --help` 출력으로 실재하는 플래그 이름을 확인해 바꿀 것(추측 금지).
- 워크플로(`.github/workflows/*`)와 `node-version: '26'`·Dockerfile digest 는 건드리지 말 것 — 느슨하게 만들어 통과시키는 것은 반려 사유다. `web/node_modules` 는 이 워크트리에 미설치이므로 build/typecheck 전 `npm ci` 가 필요하다.
- 이전 프로필의 "로컬 Node 22.23.1 에서 npm test 105 통과" 는 지금 환경과 어긋난다(같은 `node --version` 인데 플래그가 거부된다). profile.md 를 이 사실로 갱신했다.
- [러너 00:53] scout done — `web/package.json` 의 `test` 스크립트가 지금 Node 에서 제거된 `--experimental-strip-types` 를 넘겨 테스트가 한 건도 �

## 구현 노트
- 과제서 전제가 틀렸다. `--experimental-strip-types` 는 v22.23.1·v23.11.1·v25.0.0·v25.9.0 모두 **수락**하고 v20.20.2 만 거부한다. 실제 원인은 npm 이 상위 모든 `node_modules/.bin` 을 스크립트 PATH 앞에 붙이는 것 — `/home/hkjang/node_modules/.bin/node`(node@20.19.2, 9/30 설치)가 `npm run` 의 `node` 만 가로채 셸의 v22.23.1 과 달라졌다. 그래서 `web/package.json:10` 을 `${npm_node_execpath:-node}` 로 고정하고 불필요한 플래그를 제거했다.
- 플래그 삭제만 하는 과제서 안은 고치지 못한다. 가로채진 Node 20 에서 플래그 없이 돌리면 `# tests 19 / pass 0 / fail 19`(import 단계 실패)로 exit 1 이다. 이 반증을 실측해 두었다.
- 검증 못 한 것: **원격 CI(Node 26)·Docker 빌드·릴리즈 미실행.** Node 26 이 로컬에 없어 가장 가까운 v25.9.0/v25.0.0 에서 플래그 없는 `.ts` import 만 확인했다. `npm_node_execpath` 가 CI npm 에서 설정된다는 것도 로컬 npm 10.9.8 에서만 확인했다(미설정이면 `node` 로 폴백해 Node 26 기본 스트리핑으로 동작하도록 두었다). `go test -race ./...` 는 DSN 필요 + Go 무변경이라 생략.
- 일부러 안 한 것: `/home/hkjang/node_modules` 의 가로채는 `node` 를 지우지 않았다(저장소 밖 되돌릴 수 없는 전역 변경이고 홈은 임시라 다음 회차에 재발한다). `.github/workflows/*`·Dockerfile·`node-version: '26'`·`VERSION`·버전 필드도 손대지 않았다. `engines` 선언은 별도 회차 후보로 남겼다.
- 다음 역할이 조심할 것: `web/package.json` 의 `${npm_node_execpath:-node}` 를 맨 `node` 로 "정리" 하면 이 버그가 조용히 돌아온다(AGENTS.md 7장에 근거를 남겼다). `npm test` 결과는 exit 코드만 보지 말고 `# pass` 수(19파일·104통과)를 확인할 것. 변경 파일은 `web/package.json`(프로덕션 1) + `AGENTS.md`(문서) 2개다.
- [러너 01:00] brief fallback — 차선 — 지목한 파일·행(`web/package.json:10`)과 증상(exit 9, 0건 실행)은 정확했지만 정찰이 미확인으로 남긴 핵심 전제("이 No
- [러너 01:01] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 실측 재현(3상태): 수정 전 `npm test` → `bad option: --experimental-strip-types`/exit 9·0건, 과제서안(플래그만 삭제) → 19파일 pass 0/fail 19, 현재 HEAD → 19파일 104 pass/0 fail/exit 0. npm 스크립트 안의 `command -v node`=v20.19.2(가로채기) vs `npm_node_execpath`=v22.23.1 도 직접 확인했다. 구현 노트가 과제서 전제를 뒤집은 것이 맞다 — 승인.
- 못 본 것: 원격 CI(Node 26)·Docker 빌드·릴리즈. 다만 플래그 제거는 안전한 방향(Node 26 이 거부한다면 main CI 가 이미 깨져 있어야 함)이고 워크플로·digest·release.sh·VERSION 무변경이라 릴리즈 경로를 느슨하게 만든 곳은 없다.
- 남는 우려 1 — 릴리즈 노트 문구: 커밋의 "silently skip" 은 부정확하다(수정 전은 exit 9 로 요란했다). ledger-entry.md 의 "exit 9, 0건 실행" 표현을 쓸 것.
- 남는 우려 2 — 같은 드리프트가 build 에 남아 있다: `node_modules/.bin/{vite,tsc}` 가 `#!/usr/bin/env node` 라 `npm run build`·`typecheck` 는 여전히 가로채진 v20.19.2 로 돈다(실측, 지금은 통과). 구현자가 보류한 `engines`(>=22.18) 를 다음 회차에 살릴 것.
- 보안·법무 차단 없음: 인증·권한·비밀값·개인정보·의존성·라이선스 표면 무변경, Go 와 pentagi 312파일 무변경, revert 로 완전 복구된다.
- [러너 01:04] review approved — 리뷰 승인 (risk=low)
- [러너 01:04] pr created — https://github.com/hkjang/hunter/pull/16
- [러너 01:23] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함

## 심사 노트
- 확인한 것: 프로덕션 배선 그대로 3상태 실측 — main 스크립트 exit 9/0건, 과제서안 19파일 fail 19, PR HEAD 104 pass(`npm test`·`npm --prefix web test` 양쪽). 저장소 밖 임시 package.json 의 `npm run` 에서 BARE=v20.19.2 vs EXEC=v22.23.1 로 PATH 가로채기 근본 원인까지 재현했다.
- 플래그 제거 안전성: 플래그 없는 `.ts` import 가 v22.23.1·v23.11.1·v25.0.0 에서 각각 104 pass(22.18 이후 기본 스트리핑). `npm_node_execpath` 미설정 폴백도 CI Node 26 에 떨어진다. verify-pentagi 312파일 일치, `git diff --check` 깨끗, 릴리즈 표면(workflows/Dockerfile/release.sh/VERSION/잠금) 무변경.
- 못 본 것: 원격 CI(Node 26) 실행·docker build·릴리즈 자체. 릴리즈 경로 파일이 하나도 안 바뀌었고 Dockerfile 이 npm test 를 돌리지 않아 느슨해진 곳은 없다고 판단했다.
- 남긴 결함(차단 아님): AGENTS.md:150 의 "테스트가 0건 실행됩니다" 는 그 줄이 경고하는 회귀의 실제 증상(19파일 fail 19, exit 1)이 아니라 수정 전 조합의 증상이다. 커밋 제목 "silently" 도 같다. 지시와 기전은 정확하므로 승인하고 릴리즈 노트는 "exit 9, 0건 실행" 을 쓰도록 적었다.
- 권고 merge(risk=low) 근거: 2줄 변경, 인증·권한·개인정보·의존성·라이선스 표면 무변경, 마이그레이션 없음, revert 로 완전 복구. 보안·법무 차단 소견 없음. 후속으로 `engines: >=22.18`(build/typecheck 는 아직 가로채진 node 로 돈다)을 다음 회차에 남겼다.
