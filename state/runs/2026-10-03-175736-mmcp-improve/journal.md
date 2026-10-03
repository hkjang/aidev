# 회차 노트 2026-10-03-175736-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:57] base pinned — main@b782c5f
- [러너 17:57] autonomy release — 

## 정찰 노트
- 선택 이유: 러너가 우선 과제로 배정했고 실패를 그대로 재현했다 — `web/node_modules` 없음 + `cd web && npm test --silent` → `sh: 1: vitest: not found`. 다른 후보(mattermost 테스트, logbuf Group 평탄화)는 이 검증이 막혀 있는 동안 어떤 회차도 초록으로 끝낼 수 없어 제쳤다.
- 중요한 구조적 사실: `release.yml` 에는 npm·node·web 테스트 단계가 **없다**(grep 으로 확인). 즉 "릴리즈 워크플로 실패" 는 GitHub 워크플로가 아니라 러너의 검증 명령이다 — 구현자는 `release.yml`·`VERSION`·`verify-version.sh` 를 건드리면 안 된다.
- 추측으로 적은 것(미확인): `npm` lifecycle PATH 오염(`/home/hkjang/node_modules/.bin/node` = v20.19.2)을 이번 회차에는 샌드박스가 그 경로 조회와 `npm ci` 를 막아 **재확인하지 못했다**. 2026-10-03 회차의 실측 기록에만 근거한다. `vitest.config.ts:7` 의 `environment: 'jsdom'` 은 이번에 직접 읽어 확인했으므로 Node ≥22 요구는 유효하고, `npm_node_execpath` 고정은 hijack 이 없으면 no-op 이다.
- 구현자가 조심할 것: 같은 처방의 PR 이 두 번 `review-pending` 으로 머지되지 않아 base 가 안 고쳐진 것이 이 루프의 원인이다. 시작 전 `web/package.json:10` 을 열어 이미 패키지 경로면 과제서의 차선 후보로 넘어갈 것. 그리고 기준 1 은 "exit 0" 이 아니라 "테스트 4건이 실제로 돌았다" 다 — `--passWithNoTests`·테스트 제외로 통과시키는 것은 금지다.
- [러너 18:01] scout done — `cd web && npm test --silent` 가 깨끗한 체크아웃에서 항상 실패 — `web/package.json` 의 `test` 를 패키지 경로로 바�

## 구현 노트
- 무엇을 왜: base 의 `web/package.json:10` 만 `"vitest run"` 으로 `.bin` 심링크+기설치를 전제해 깨끗한 체크아웃에서 러너 검증이 떨어졌다. 패키지 경로 호출 + `scripts/build-ui.sh:6` 과 글자까지 같은 `pretest` 가드 + `"${npm_node_execpath:-node}"` 인터프리터 고정으로 고치고, `Makefile:17`·`ci.yml:57` 을 `npm test --silent` 로 돌려 vitest 호출을 저장소에 한 곳만 남겼다. 프로덕션 3파일, 신규 파일 없음.
- 확신 없는 곳·검증 못 한 것: **CI 러너에서는 돌려보지 못했다**(이 세션은 push 불가). `ci.yml:56` 의 `npm ci` 를 그대로 뒀으므로 CI 에서 `pretest` 가드는 `node_modules` 가 이미 있어 no-op 이고, `npm_node_execpath` 는 GitHub 러너에 `.bin/node` 그림자가 없으면 no-op 이다 — 둘 다 구조상 안전하지만 실측은 로컬뿐이다. `npm_node_execpath` 는 npm 이 세팅하는 변수라 `npm test` 를 거치지 않고 `node_modules/.bin/npm` 아닌 다른 러너(yarn/pnpm)로 부르면 `:-node` 폴백이 쓰인다.
- 일부러 하지 않은 것: `web/package.json` 의 `build`·`dev`·`preview`·`screenshots` 는 아직 `.bin` 심링크 의존이지만 과제서가 범위 밖으로 지정해 손대지 않았다(CI·`build-ui.sh` 는 패키지 경로라 안 깨진다). `release.yml`·`VERSION`·`verify-version.sh`·`package-lock.json` 미변경.
- 다음 역할이 조심할 것: 기준 1 을 재현하려면 반드시 `rm -rf web/node_modules` 를 먼저 할 것 — 이미 설치된 상태에서는 가드가 no-op 이라 결함이 안 보인다. 첫 실행은 `npm ci` 때문에 ~11s 걸리고 그 뒤는 ~2s 다. `go test ./...` 는 Postgres 없어 통합 테스트가 조용히 skip 된다(CI 가 진짜 전체).
- [러너 18:05] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했다(`web/package.json:10` 이 여전히 `"vitest run"`, `Makefile:17`·`ci.yml:57` 은 이
- [러너 18:05] verify passed — 검증 7개 통과 (auto)
- [러너 18:05] pr created — https://github.com/hkjang/mmcp/pull/3
- [러너 18:05] guard held — .github/workflows/ci.yml 
