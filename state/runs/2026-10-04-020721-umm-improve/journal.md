# 회차 노트 2026-10-04-020721-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:07] base pinned — main@b4bddea
- [러너 02:07] autonomy release — 

## 구현 노트
- 지정된 `npm test --silent` (exit 1) 의 원인은 **`web/node_modules` 미설치**다. 결정 실험: 설치 전 exit 1 → `npm ci`(web) → 같은 명령 EXIT=0 / 20파일 218시험. 워크플로는 멀쩡하다(`ci.yml:114` 이 프런트엔드 앞에서 `npm ci`, `release.yml` 에 npm 테스트 없음) — 손대지 않았다.
- 그래서 고친 것은 '통과시키기' 가 아니라 그 실패가 두 번 오진된 이유다: `run-on-supported-node.mjs:164` 의 MODULE_NOT_FOUND 가 안 잡혀 출력이 `Node.js v20.19.2` 로 끝나고, 그 바로 위 줄이 같은 스크립트의 인터프리터 전환 통지라서 설치 누락이 Node 가림 문제로 읽혔다. 이제 의존성 이름·manifest·`npm ci --prefix` 를 이름 붙여 exit 1 로 거절한다(MODULE_NOT_FOUND 외는 rethrow, exit 코드·engines 범위·워크플로 전부 그대로).
- **확신 없는 곳**: ① 이 명령은 설치 없이는 여전히 exit 1 이다 — 러너 검증 단계가 프런트엔드 명령 앞에 `npm ci` 를 넣어야 다음 회차가 같은 자리에서 멈추지 않는다(저장소 쪽으로는 더 할 것이 없다고 판단했다; 래퍼가 대신 설치하는 것은 운영자 규칙 10번). ② 시험이 쓰는 임시 패키지 루트는 `/tmp/umm-web-uninstalled-*` 인데, 만약 `/tmp/node_modules/vitest` 가 있는 기계라면 resolve 가 성공해 시험이 실패한다 — 이 환경에는 없고 그 경우 조용히 통과하지 않고 **실패**한다.
- 일부러 하지 않은 것: 파일 머리말의 "three separate ways" 는 인터프리터 선택이라는 한 주제의 결정 기록이라 설치 누락을 섞지 않았다(결정은 catch 자리 인라인 주석에 적음). 자동 설치·workspaces 전환·CI 수정도 하지 않았다.
- 다음 역할이 조심할 것: 새 시험(`web/scripts/run-on-supported-node.test.mjs`)은 vitest 안에서 **실제 래퍼를 자식 프로세스로 띄우고** 임시 디렉터리를 만든다(afterAll 에서 지움) — DB 는 필요 없지만 프로세스 생성과 `/tmp` 쓰기가 필요하다. Go 코드는 0줄 바뀌었고, `go test ./...` 는 POSTGRES_DSN 없이 돌려 DB 통합은 SKIP 이다(돌았다고 읽지 말 것).
- [러너 02:14] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 핵심 확인: main 의 스크립트를 node_modules 없는 임시 루트에 복사해 돌려 수정 전 출력(`Cannot find module 'vitest/package.json'` … `Node.js v22.23.1`, `npm ci` 문구 없음)을 직접 재현했고, 그러므로 새 시험 `run-on-supported-node.test.mjs:63,70-72` 가 수정 전에 실제로 실패한다 — 항상 참인 단언이 아니다. HEAD 는 같은 상황에서 이름 붙인 메시지 + EXIT=1.
- 함께 실행: 프런트엔드 전체 218시험/20파일 PASS(새 시험 3개 포함, `scripts/**/*.test.mjs` 가 vitest include 에 이미 있었다), typecheck·lint(prettier 포함)·check-i18n(1060키)·check-version 전부 통과. 못 본 것: 원장이 적은 `npm audit`·`docker build`·`go test`(Go 0줄이라 생략)·Playwright·DB 통합.
- 구현자 ②번 의심 검증: 이 기계에 `/tmp/node_modules` 도 `/node_modules` 도 없다. 단 시험 루트가 os.tmpdir() 아래라 해석이 상위로 거슬러 오르므로, TMPDIR 이 node_modules 를 품은 트리인 기계에서는 시험 1·2 가 제품 결함 없이 깨진다(조용히 통과하진 않는다).
- 승인 후에도 남는 우려: catch 가 MODULE_NOT_FOUND 만 이름 붙이므로 "설치는 있으나 깨진" 경우(ERR_PACKAGE_PATH_NOT_EXPORTED, 부분 설치)는 여전히 `Node.js v…` 로 끝나는 스택이다 — 같은 오진 모양의 좁은 갈래가 남았다. 머리말 "three separate ways" 와 넷째 실패 모드의 불일치는 의도적 보류로 받아들였다.
- 릴리즈 노트용: 사용자 가시 변화 없음(개발자 진단 메시지), 0.76.4 유지. 다음 회차 러너는 프런트엔드 명령 앞에 `npm ci --prefix web` 를 넣어야 한다 — 이 명령은 설치 없이는 설계대로 exit 1 이다.
- [러너 02:18] review approved — 리뷰 승인 (risk=low)
- [러너 02:19] pr created — https://github.com/hkjang/umm/pull/165
- [러너 02:32] ci passed — 검사 1개 모두 success
- [러너 02:32] merge done — 9755334
- [러너 02:51] release published — v0.76.5
- [러너 02:52] assets verified — v0.76.5 자산 3개 (이전 v0.76.4: 3)
