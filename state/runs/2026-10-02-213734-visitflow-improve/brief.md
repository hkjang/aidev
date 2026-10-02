- 과제: 실제 dist 를 임베드한 실제 서버 + 실제 브라우저 e2e 를 저장소의 한 명령으로 고정하기 (`scripts/local-e2e.sh`) (가치 4 / 위험 2 / 작업량 M)
- 왜: 직전 회차 improve 단계가 `error: agent produced no result (TIMEOUT )` 로 끝났다(`state/runs/2026-10-02-182737-visitflow-improve/stages.json`: scout done 18:33 → improve error 19:13). 코드 변경 자체는 끝나 있었고(원장의 성공 요약) 예산을 태운 것은 「dist 빌드 → `cmd/visitflow/webdist` 임베드 → Postgres+서버 기동 → Chromium 연결 → 스텁 복원」 하네스를 매 회차 /tmp 에서 **손으로 다시 알아내는 일**이다 — 원장 2026-09-27·09-28·09-30·09-30·10-02 다섯 회차가 전부 "이번에도 손으로 재구성했다" 를 적고 있고, 같은 환경 사실(8080 점유 → 전용 네트워크 + 18099, 번들 chromium 없음 → `channel:"chrome"`, 스텁 복원)을 매번 다시 발견한다. 이것을 저장소에 고정하면 다음 회차부터 그 시간이 사라지고, 이 저장소에서 1급 증거로 요구되는 브라우저 검증이 싸진다.
- 주의(정찰의 판단): 이번 실패는 **저장소 코드의 결함이 아니라 구현 에이전트의 벽시계 초과**다. `.github/workflows/release.yml`·`ci.yml` 은 읽어 봤고 이상이 없다(아래 "확인한 것"). 그래서 "워크플로를 느슨하게 해서 통과시키기" 는 애초에 해당 사항이 없고, 대신 타임아웃을 만든 반복 비용을 저장소 안에서 제거한다.

- 수용 기준:
  1) 깨끗한 체크아웃에서 `bash scripts/local-e2e.sh` **한 번**으로 실제 `npm run build` 산출물을 임베드한 서버가 뜨고 `npm run test:e2e` 가 그 서버를 상대로 돌며, 마지막 줄에 Playwright 결과와 사용한 base URL 이 찍힌다. 8080·5432 가 이미 다른 컨테이너에 잡혀 있는 환경에서도 성공해야 한다 — 포트를 코드에 박지 말고 비어 있는 포트를 고르거나 env(`VISITFLOW_LOCAL_E2E_PORT` 같은)로 받는다.
  2) 스크립트가 끝난 뒤 `git status --porcelain` 이 **빈 출력**이다. 특히 `cmd/visitflow/webdist/` 에 추적된 스텁 `index.html` 하나만 남는다(`cmd/visitflow/main.go:26` 의 `//go:embed webdist`). 테스트가 실패해서 끝나든 Ctrl-C 로 끊기든(trap) 복원된다. 복원은 **임시 복사본**으로 하고 `git checkout --` 를 쓰지 않는다(과거에 미커밋 테스트를 잃은 기록).
  3) `VISITFLOW_E2E_BROWSER_CHANNEL`(이름은 자유)을 **설정하지 않으면** `web/playwright.config.ts` 의 동작이 지금과 완전히 동일하다 = 번들 chromium. CI e2e 잡은 이 env 를 설정하지 않으므로, PR 의 CI e2e 잡이 그대로 통과하는 것이 이 기준의 증거다.
  4) 스크립트는 검사를 끄거나 느슨하게 하지 않는다. `npm run test:e2e` 를 그대로 호출하고, **테스트가 실패하면 0 이 아닌 종료 코드**를 낸다 — e2e 스펙 하나를 일부러 깨뜨린 상태로 한 번 돌려 종료 코드가 0 이 아닌 것을 보이고 되돌린다(임시 복사본).
  5) `README.md:124-129` 의 "서비스를 실행한 뒤 아래를 실행하면…" 문단이 이 명령을 가리킨다. 기존 수동 절차(`npx playwright install`, `VISITFLOW_BASE_URL=… npm run test:e2e`)와 모순되지 않게, 정본은 하나로 둔다(옛 수동 가이드를 남겨 둘 거면 "스크립트가 하는 일" 로 위치를 바꿀 것).

- 건드릴 파일 (총 4개, 프로덕션 Go/React 0개):
  - `scripts/local-e2e.sh` — 신규. 기존 `scripts/release-image.sh` 의 관례를 따를 것(`#!/usr/bin/env bash` + `set -euo pipefail`, 인자 검증, 경로는 저장소 루트 기준). 단계: web 의존성 확인(`node_modules` 없으면 `npm ci`) → `npm run build` → `cmd/visitflow/webdist` 를 임시 복사본으로 백업하고 `web/dist/.` 를 복사 → `CGO_ENABLED=0 go build -o <tmp>/visitflow ./cmd/visitflow` → Postgres 와 서버를 전용 네트워크/고유 이름으로 기동 → `/readyz` 폴링 → `VISITFLOW_BASE_URL` 을 주고 `npm run test:e2e` → trap 으로 컨테이너·네트워크 정리와 webdist 복원.
  - `web/playwright.config.ts` — `projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }]` 에 env 기반 channel 을 **조건부로만** 넣는다. `...(process.env.VISITFLOW_E2E_BROWSER_CHANNEL ? { channel: process.env.VISITFLOW_E2E_BROWSER_CHANNEL } : {})` 형태(스프레드 조건부)가 `channel: undefined` 보다 안전하다 — `undefined` 가 "미지정" 과 동일한지 Playwright 1.56 에서 확인하기보다 아예 키를 넣지 않는 쪽을 택할 것. 이 파일에서 다른 줄은 건드리지 말 것(`baseURL`·`timeout`·`retries: process.env.CI` 모두 CI 가 쓴다).
  - `README.md` 124-129행 — 한 문장 + 명령 블록.
  - `.gitignore` — 스크립트가 저장소 안에 산출물을 만들면 그때만. **만들지 않는 쪽(모두 `mktemp -d` 아래)이 1순위**이고, 그러면 이 파일은 건드리지 않는다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `bash -n scripts/local-e2e.sh` (문법), 있으면 `shellcheck scripts/local-e2e.sh`
  - `bash scripts/local-e2e.sh` → 종료 코드 0, Playwright 가 실제로 브라우저를 띄워 통과
  - 스크립트 종료 직후 `git status --porcelain` → 빈 출력
  - `cd web && npm ci && npm run lint && npm test && npm run build` (lint=`tsc -b`, test=vitest 79개)
  - `go build ./...` · `go vet ./...` · `gofmt -l .`(빈 출력) · `git diff --check`
  - `VISITFLOW_TEST_DSN='postgres://visitflow:visitflow@127.0.0.1:<port>/visitflow?sslmode=disable' go test ./... -count=1` — DSN 없으면 통합 테스트가 **SKIP** 되므로 PASS 가 실행을 뜻하지 않는다(internal/app 약 55~60초)
  - 수용 기준 3 은 PR 의 CI `e2e` 잡 통과로 확인한다 (`.github/workflows/ci.yml` 의 e2e 잡이 `npm run test:e2e` 를 번들 chromium 으로 돌린다)

- 위험과 피할 것:
  - **`.github/workflows/*` 를 건드리지 말 것.** 보호 경로이고, "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것" 교훈의 대상이다. CI 와 공유되는 것은 `web/playwright.config.ts` 한 파일뿐이니 변경을 거기 한 줄로 가두고, env 미설정 경로가 **바이트 단위로 같은 동작**인지 생각하지 말고 CI 로 증명할 것.
  - `cmd/visitflow/webdist/index.html` 은 **추적된 스텁**이고 `assets/` 만 .gitignore 에 있다(`.gitignore:17`). 복원을 빠뜨리면 다음 회차의 `git status` 가 오염된다.
  - 운영자 지시: **스크립트가 전역 설정을 백업 없이 덮어쓰지 않을 것.** 호스트의 docker 전역 상태(기존 네트워크·컨테이너 이름·`~/.docker`)를 건드리지 말고, 고유 접두사 + `trap ... EXIT INT TERM` 로 자기가 만든 것만 지울 것. 기존에 돌고 있는 다른 프로젝트의 postgres/8080 컨테이너를 멈추는 코드는 절대 넣지 말 것.
  - **분량이 커지면 쪼갤 것.** 이번 조각은 「서버 기동 + e2e 실행 + 복원」까지다. 데이터 시딩, `page.route` 지연 주입, 변이 번들 생성 같은 결함 재현용 하네스는 **넣지 말 것** — 그걸 넣기 시작하면 다시 M 을 넘긴다.
  - 저장소 e2e `registers a site, a lobby and an organization from the admin console` 는 **이 환경에서 기존부터 실패**한다(원장 2026-09-30·10-02 에 수정 전/후 바이너리 + 깨끗한 DB 로 각각 재확인). 스크립트가 "전부 통과" 를 전제로 만들어지면 안 되고, 이 한 건의 실패를 숨기려고 스펙을 skip 처리해서도 안 된다 — 종료 코드는 그대로 전달하고, 이 알려진 환경 의존 실패는 README 나 스크립트 주석에 한 줄로 밝혀 둘 것.
  - 미확인(정찰이 이번 회차에 직접 재확인하지 않음, 출처는 원장과 프로필): ① 이 환경에서 8080·5432 가 다른 프로젝트 컨테이너에 점유되어 전용 도커 네트워크 + 호스트 18099 가 필요했다는 것, ② Playwright 번들 chromium 이 없어 `channel:"chrome"`(/usr/bin/google-chrome)이 필요하다는 것, ③ docker·google-chrome 바이너리의 현재 존재 여부(이번 회차에 권한 때문에 실행 확인을 못 했다). 구현자는 **먼저 이 셋을 직접 확인**하고, 다르면 스크립트를 환경에 맞추되 포트·채널을 하드코딩하지 말고 env 기본값으로 둘 것.
  - 확인한 것: `.github/workflows/release.yml`(태그 → buildx → `docker save | gzip` → `gh release create --verify-tag`)과 `ci.yml`(test 잡 + e2e 잡)을 전부 읽었고 문법·논리 결함은 없다. 직전 성공 회차 `2026-10-02-095806` 의 stages.json 은 ci passed → merge → release published v2.8.12 → assets verified 로 끝났다. 즉 GitHub 쪽 릴리즈 워크플로는 마지막으로 돈 때 정상이었고, 실패한 것은 aidev 의 improve 단계 타임아웃이다.

- 차선 후보: 로비 현황에서 늦게 도착한 이전 조회 응답 무시 (`web/src/pages/LobbyPage.tsx:30` 의 `load`) — 직전 회차에 `VisitsPage.tsx` 에 머지된 요청 티켓(`useRef<number>`, 진입에서 `++`, 커밋과 `finally` 를 `requestSeq.current === ticket` 로 가드, 상대 플래그 해제 한 줄)을 그대로 적용한다. 프로덕션 1개 파일. 보존할 계약: `useEffect(..., [])` 로 **한 번만** 하는 SSE 구독과 `loadRef`(필터·검색마다 재구독하면 이벤트가 샌다), 빈 `lobbyId` 계약, 탭·검색·로비 필터. `LobbyPage` 는 덧붙이기(`loadMore`)가 없어 오염이 일시적이므로 1순위보다 증상이 약하다.
