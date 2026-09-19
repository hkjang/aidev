# 과제서 (2026-09-19, 정찰) — 수정 과제

- 과제: 릴리즈 검증 체인(verify → docker → smoke → e2e → package → verify-bundle)의 성공 경로 출력을 줄여 자동 릴리즈 세션이 budget hold로 끊기지 않게 하기 (가치 4 / 위험 1 / 작업량 S)

- 왜: 마지막 두 회차는 러너가 `hold: budget`(세션 USD 예산 소진)으로 'error' 종료했고, 코드·테스트 자체는 깨지지 않았다(이번 정찰에서 `go test ./...` 전 패키지 통과 확인, `main@7fc2212` = `v0.2.15` 태그 커밋, 작업 트리 clean). 릴리즈 세션이 도는 `make release-check`(= `scripts/verify.sh` + docker build + `smoke-offline.sh` + `e2e-docker.sh` + `package-offline.sh` + `verify-offline-bundle.sh`)는 `npm ci`·vitest·eslint·vite build·docker build·playwright의 진행 출력을 그대로 세션 컨텍스트에 흘려보내므로, 검사를 하나도 빼지 않고 성공 경로의 출력만 줄이면 같은 검증을 훨씬 적은 토큰으로 끝낼 수 있다. 워크플로(`.github/workflows/release.yml`)의 단계·순서·실패 조건은 손대지 않는다.
  - 미확인: GitHub Actions의 release.yml 실행 이력(gh CLI 승인 불가)과 러너의 이전 회차 로그(디렉터리 열람 불가)는 보지 못했다. "budget"이 러너 세션 예산이라는 판단은 러너 메시지 문구와 이번 세션의 호출당 비용(호출 1회 ≈ $0.03, 세션 $2)에서 추론한 것이다. 구현자는 시작 전에 `gh run list --workflow=release.yml --limit 3`로 Actions 쪽 실패가 따로 있는지 1회 확인하고, Actions가 실제로 실패했다면 그 단계 로그를 이 과제서보다 우선한다.

- 수용 기준:
  1) `./scripts/verify.sh`가 성공할 때 터미널 출력이 현재보다 뚜렷이 줄어든다(목표: 성공 시 100줄 이하 — 구현자가 전/후 `wc -l`로 기록). 실행하는 검사 목록은 그대로: bash -n, gofmt -l, `go test`, `go vet`, `npm ci`, vitest, eslint, vite build, `verify-docs.mjs`, `docker compose config`.
  2) 실패 시에는 원인이 보인다 — 각 도구의 오류 출력을 버리지 않는다(`>/dev/null 2>&1`로 실패까지 숨기는 방식 금지). 예: `npm ci --loglevel=error`, `npm test -- --reporter=dot --silent`(vitest), `npm run build -- --logLevel=warn`(vite), `playwright test --reporter=dot`(실패 시 playwright는 dot reporter에서도 실패 상세를 출력함 — 구현자가 일부러 한 spec을 깨뜨려 실패 메시지가 나오는지 확인하고 되돌릴 것).
  3) `docker build`는 `--quiet`를 쓰지 말 것(실패 레이어 로그가 사라진다). 대신 `--progress=plain` 출력을 그대로 두거나, 성공 시에만 요약하도록 임시 파일에 받아 실패 시 `cat`하는 패턴을 쓴다(`smoke-offline.sh`·`e2e-docker.sh`의 `docker logs` 실패 시 출력 관례와 동일).
  4) 검증 명령 전체가 로컬에서 통과한다(아래). exit code 의미와 `printf '검증 완료: …'` 최종 줄은 유지.
  5) CI(`ci.yml`, `release.yml`)는 수정하지 않는다. 스크립트 시그니처(`verify.sh [--docker] [--smoke]`, `e2e-docker.sh IMAGE`)도 그대로.

- 건드릴 파일:
  - `scripts/verify.sh` — `npm --prefix web ci --no-audit --no-fund`에 `--loglevel=error` 추가; `npm --prefix web test -- --maxWorkers=1`에 vitest 조용한 reporter; `npm run lint`는 그대로(성공 시 원래 조용함); `npm run build`에 vite `--logLevel=warn`; 그 밖의 검사는 손대지 않음.
  - `scripts/e2e-docker.sh:104` — `npm exec -- playwright test`에 `--reporter=dot` (또는 `PLAYWRIGHT_REPORTER` 환경변수 존중). `web/playwright.config.*`에 reporter가 이미 지정돼 있는지 먼저 확인(미확인).
  - (선택) `Makefile` `release-check` — 변경 없음. 단계 이름만 확인.
  - 원장 기록: 이 과제는 '수정 과제'로 표기.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `./scripts/verify.sh` (Go test·vet·gofmt, React ci·test·lint·build, docs, compose) — 2~5분
  - `./scripts/verify.sh --smoke` (docker build + 폐쇄망 스모크) — docker 필요, 5~10분
  - `docker pull postgres:17-alpine && ./scripts/e2e-docker.sh jikim:v0.2.15` — Playwright chromium 필요(`npm --prefix web exec -- playwright install --with-deps chromium`), 5~10분
  - `make release-check` — 전체(위 모두 + package + verify-bundle). 시간이 오래 걸리므로 백그라운드로 돌리고 마지막 20줄만 확인.
  - 각 실행 전후로 `… 2>&1 | wc -l`로 출력 줄 수를 기록해 수용 기준 1을 증명.

- 위험과 피할 것:
  - 검사를 빼거나 실패를 삼키는 변경은 금지(운영자 규칙: 워크플로를 느슨하게 해 통과시키지 말 것). "조용히"는 성공 경로만이다.
  - `docker build --quiet`·`playwright test --quiet`처럼 실패 상세를 잘라먹는 플래그는 금지.
  - `resource_handlers.go`·`store/settings.go`·`SettingsPage.tsx`·auth·migrations는 이 과제와 무관하니 건드리지 않는다.
  - 버전(`scripts/version.sh`의 `v0.2.15`)은 올리지 않는다 — 릴리즈 세션이 올린다.
  - 과거 교훈: 실제 출력·동작이 바뀌지 않는 수정은 반려 사유다. 전/후 줄 수 비교를 커밋 메시지나 PR 본문에 남겨 효과를 증명할 것.

- 차선 후보: settings GET이 주입하는 파생 필드(`four_eyes`·`required_approvals`·`supported_targets`)가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) — `internal/httpapi/resource_handlers.go`의 `settings()`/`updateSettings()`. 메일 PR(4f729aa)과 MCP OAuth 커밋이 모두 main에 들어왔는지 `git log --oneline main | grep -i mail`로 먼저 확인.
