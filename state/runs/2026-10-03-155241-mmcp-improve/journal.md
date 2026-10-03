# 회차 노트 2026-10-03-155241-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:52] base pinned — main@b782c5f
- [러너 15:52] autonomy release — 

## 정찰 노트
- 과제는 자동 배정(verify-failed)이라 선택의 여지가 없었다. 실패를 실제로 재현했다: `cd web && npm test --silent` → `sh: 1: vitest: not found`. 원인은 base `b782c5f` 의 `web/package.json:10` 이 `"vitest run"` 이고 `web/node_modules` 가 없다는 것 — 2026-10-02 의 수정 PR 이 `review-pending` 으로 머지되지 않아 같은 결함이 그대로 남았다. 사람 반려가 아니므로 재제출 금지 규칙에 걸리지 않는다.
- 릴리즈 워크플로는 읽어서 확인한 결과 web 테스트를 부르지 않는다(`release.yml` 은 verify-version → package-offline → verify-offline → gh upload). 실패한 단계는 러너 검증 명령 쪽이다. `ci.yml:57` 은 바로 위에 `npm ci` 가 있어 초록이었고, 이것이 CI 와 러너가 갈린 구조적 원인이다.
- 추측으로 적은 것(미확인): npm lifecycle 의 PATH 오염(상위 `node_modules/.bin` 의 Node 20 가로채기)을 이번에는 **재확인하지 못했다** — 해당 경로는 세션 읽기 허용 범위 밖이고 `npm ci` 도 권한으로 막혀 실측이 불가능했다. PATH 의 node 는 v22.23.1 로 확인했다. 그래서 과제서는 2026-10-02 의 실측을 근거로 `"${npm_node_execpath:-node}"` 고정을 처방했다 — 오염이 없으면 무해한 no-op 이다.
- 구현자가 조심할 것: 건드릴 파일을 3개(`web/package.json`, `Makefile:17`, `ci.yml:57`)로 묶었다. 같은 파일의 `build`/`dev`/`preview` 도 `.bin` 의존이지만 **의도적으로 범위에서 뺐다**(별 아이디어로 기록). `release.yml`·`package-lock.json`·빌드 산출물은 건드리지 말 것.
- [러너 15:56] scout done — `cd web && npm test --silent` 가 깨끗한 체크아웃에서 항상 실패 — `web/package.json` 의 `test` 를 패키지 경로 호출�

## 구현 노트
- 무엇을 왜: `web/package.json` 만 `"vitest run"` 으로 `.bin` 심링크+기설치를 전제해 깨끗한 체크아웃에서 `npm test` 가 깨졌다. 패키지 경로 호출 + `pretest` 설치 가드(`build-ui.sh:6` 과 동일) + `"${npm_node_execpath:-node}"` 인터프리터 고정으로 고치고, `Makefile:17`·`ci.yml:57` 을 `npm test --silent` 로 모아 vitest 호출을 저장소에 한 곳만 남겼다. 프로덕션 파일 3개, 커밋 f3ab958.
- 정찰이 "미확인" 으로 남긴 PATH 오염은 **이번에 실측 확인했다**: `/home/hkjang/node_modules/.bin/node` = v20.19.2, `markAsUncloneable` → `undefined`. 따라서 `npm_node_execpath` 고정은 no-op 이 아니라 필수다. 두 조각을 각각 뮤테이션으로 양방향 검증했다(바레 node → `markAsUncloneable` 오류 / 가드 제거 → `MODULE_NOT_FOUND`).
- 확신 없는 곳: ① 이 PATH 오염은 **이 머신의 홈 디렉터리 상태**다. CI 러너에는 없을 것이므로 `npm_node_execpath` 는 CI 에서 무해한 no-op 이지만, CI 에서 실제로 돌려보지는 못했다(push 권한 없음) — `ci.yml:57` 변경의 유일한 미검증 지점이다. ② `pretest` 가드가 도는 `npm ci` 는 레지스트리에 닿는다(여기선 성공, ~9s). 완전 오프라인 환경에서는 `node_modules` 가 없으면 가드가 실패하는데, 이는 `build-ui.sh` 의 기존 동작과 동일하므로 새 제약은 아니다.
- 일부러 하지 않은 것: 같은 파일의 `build`/`dev`/`preview` 도 `.bin` 의존이지만 범위에서 뺐다(ideas.json 에 [2/1/S] 로 기록). `release.yml`·`package-lock.json`(`npm ci` 만 썼으므로 미변경 확인)·빌드 산출물은 건드리지 않았다. 검증을 느슨하게 하는 수단(테스트 제외/`--passWithNoTests`/`|| true`/`continue-on-error`)은 쓰지 않았다.
- 다음 역할이 조심할 것: `web/node_modules` 는 gitignore 되어 있고 지금 워크트리에는 설치돼 있다 — `npm test` 재현 시 설치 상태를 먼저 확인할 것. Go 통합 테스트는 `TEST_POSTGRES_DSN` 이 없으면 조용히 skip 되므로 이번 `go test -race ./...` ok 는 통합 경로를 돌지 않았다(이 변경이 Go 를 건드리지 않으므로 무관).
- [러너 16:01] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했다(`web/package.json:10` 만 `.bin` 의존, `Makefile:17`·`ci.yml:57` 은 이미 패키�
- [러너 16:01] verify passed — 검증 7개 통과 (auto)
- [러너 16:01] pr created — https://github.com/hkjang/mmcp/pull/2
- [러너 16:01] guard held — .github/workflows/ci.yml 
