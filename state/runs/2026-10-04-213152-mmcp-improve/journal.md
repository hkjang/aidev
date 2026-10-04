# 회차 노트 2026-10-04-213152-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:31] base pinned — main@b782c5f
- [러너 21:32] autonomy release — 

## 정찰 노트
- 고른 이유: 선택의 여지가 없었다 — 러너가 verify-failed(`cd web && npm test --silent`)를 우선 과제로 배정했고, base `b782c5f` 에서 직접 재현했다(`web/node_modules` 없음 + `sh: 1: vitest: not found`). 차선(`internal/mattermost` 테스트)은 이 검증 명령이 막혀 있는 동안은 어떤 회차도 통과할 수 없어 뒤로 미뤘다.
- 범위 정정: 배정문은 "릴리즈 워크플로" 라고 했지만 `release.yml` 37줄 전문에 web/npm 단계가 **없다**. 실패 지점은 러너 자신의 검증 단계다 — 구현자가 `release.yml` 을 뒤지지 않도록 과제서에 명시했다.
- 과거 전제 하나를 뒤집었다: 세 회차가 근거로 삼은 `/home/hkjang/node_modules/.bin/node`(v20.19.2, PATH 가로채기)가 **이번 환경에는 없다**. `npm_node_execpath` 고정은 방어로 유지하되 재현 근거로는 쓰지 말라고 적었다.
- 미확인: `npm ci` 의 레지스트리 도달을 재확인하려다 권한 거부로 못 돌렸다(과거 두 회차 실측은 성공). 구현자가 기준1 에서 가장 먼저 확인할 지점이다.
- 운영자에게: 같은 수정이 네 회차째다(10-02, 10-03×2 모두 '성공'·'채택' 인데 review-pending 으로 머지 안 됨). 코드가 아니라 머지 파이프라인 문제로 보인다 — 다섯 번째 반복을 막으려면 그쪽을 봐야 한다. `ideas.json` 에 항목으로 남겼다.
- [러너 21:36] scout done — `cd web && npm test --silent` 가 깨끗한 체크아웃에서 항상 실패 — `web/package.json` 의 `test` 를 패키지 경로로 바�

## 구현 노트
- `web/package.json` 의 `scripts.test` 를 패키지 경로(`node_modules/vitest/vitest.mjs`)로 바꾸고 `pretest` 설치 가드를 넣었다. `Makefile:17`·`ci.yml:57` 은 `npm test --silent` 를 부르게 해 vitest 실행 문장을 저장소에 한 곳만 남겼다. 커밋 `bd048a2`, 프로덕션 파일 3개.
- **과제서의 환경 정정이 틀렸다 — 비평가가 여기를 먼저 볼 것.** 과제서는 `/home/hkjang/node_modules/.bin/node` 가 "이번 환경에는 없다, 고정은 no-op 일 수 있다" 고 했지만, 실측 결과 **존재하고 v20.19.2** 다. lifecycle 에서 `command -v node` → 그 경로 / `$npm_node_execpath` → v22.23.1 을 출력으로 확인했고, 고정을 빼는 뮤테이션으로 `markAsUncloneable` 실패를 재현했다. 즉 `"${npm_node_execpath:-node}"` 고정은 방어가 아니라 **필수**다.
- 확신 없는 것: 없음. 수용 기준 5개를 모두 실행해 출력을 확인했고(기준1 은 `rm -rf node_modules` 에서 4 passed/skip 0), 두 조각(`pretest` 가드·인터프리터 고정)을 각각 빼는 뮤테이션으로 양방향 인과를 확인한 뒤 되돌렸다. 과제서가 "미확인" 으로 남긴 `npm ci` 레지스트리 도달도 확인했다(191 packages, 9s).
- 일부러 하지 않은 것: `web/package.json` 의 `build`·`dev`·`preview`·`screenshots` 의 `.bin` 의존(과제서가 명시적으로 범위 밖 — `ideas.json` 에 남겼고 같은 파일이라 충돌하므로 이 커밋이 머지된 뒤에), `release.yml`(web/npm 단계가 없어 무관), Go 쪽·보호 경로, `package-lock.json`.
- 다음 역할이 조심할 것: `npm test` 는 이제 `web/node_modules` 가 없으면 `npm ci` 로 레지스트리에 **나간다**(오프라인 러너라면 첫 실행이 여기서 멈춘다 — 가드를 캐시 기반으로 바꾸지 말고 먼저 보고할 것). `go test ./...` 는 Postgres 가 없어 `internal/server` 통합 테스트가 조용히 skip 됐다(이번 변경은 Go 와 무관해 추가로 띄우지 않았다). `web/node_modules` 는 워크트리에 설치된 상태로 남아 있고 gitignore 되어 커밋에는 없다.
- 운영자에게: 이 수정이 **다섯 번째**다(10-02, 10-03×2, 10-04 모두 '성공'·'채택' 인데 base 는 그대로). 구현·검증이 아니라 머지가 막혀 있다 — 그쪽을 보지 않으면 여섯 번째가 온다.
- [러너 21:42] brief accepted — 채택 — 과제서의 근거가 현재 코드와 글자 단위로 일치했고(`web/package.json:10` 이 `"vitest run"`, `Makefile:17`·`ci.yml:57` 은 이미
- [러너 21:42] verify passed — 검증 7개 통과 (auto)
- [러너 21:43] pr created — https://github.com/hkjang/mmcp/pull/4
- [러너 21:43] guard held — .github/workflows/ci.yml 
