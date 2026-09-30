- 과제: 추적 허용 출처 **목록 전체**의 크기에 상한을 두어 정책 헤더가 무한히 커지지 못하게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: 2026-09-28 회차가 `Settings.Validate()` 에 허용 출처 **한 항목**의 300 룬 상한과 세미콜론 거절을 넣었지만, 항목 **개수와 합계 길이**에는 아무 제한이 없다. `PolicySources()` 는 `splitHosts(s.AllowedHosts)` 의 모든 항목을 `everywhere()` 로 img-src·connect-src·script-src 세 지시문에 각각 넣고(`internal/tracking/tracking.go:330-332`), `pagePolicy()` 가 그것을 공백 join 해 **모든 페이지 응답의 Content-Security-Policy 헤더**로 내보낸다(`internal/api/tracking.go:101-111`) — 즉 허용 목록 n 룬은 헤더에서 3n 룬이 된다. 설정 PUT 본문 상한(1MiB, respond.go)까지 채우면 수 MiB 짜리 응답 헤더가 되어 리버스 프록시·브라우저가 콘솔 페이지 자체를 거절한다.
- 수용 기준:
  1) 합계가 상한을 넘는 `allowedHosts` 를 담은 설정 PUT 이 400 과 한국어 사유로 거절되고, `system_settings` 의 기존 행이 바뀌지 않는다(저장 뒤 거절이 아니라 저장 전 거절).
  2) 상한 이하의 정상 목록은 지금과 똑같이 통과하고 `PolicySources()`·`pagePolicy()` 의 출력이 바뀌지 않는다(기존 테스트 무수정 통과).
  3) 테스트가 증명할 것: 상한을 넘는 목록이 들어왔을 때 (a) Validate 가 거절한다, (b) 한 번 클릭 허용(`POST /api/v1/admin/tracking/allow`)도 같은 상한에서 거절되어 반복 클릭으로 우회되지 않는다, (c) 상한 **바로 아래** 목록으로 만든 CSP 헤더의 길이가 계산 가능한 한계 안에 있다 — 즉 "한 항목 300 룬" 만으로는 헤더가 안 묶인다는 것을 헤더 문자열 길이로 보인다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/tracking/tracking.go` — 상수 추가(예: `MaxAllowedHostEntries`, `MaxAllowedHostsTotalRunes = 2048`)와 `Validate()` 의 기존 `for _, host := range splitHosts(s.AllowedHosts)` 루프 뒤에 합계 검사 1건. 상한 근거는 상수 주석에 산문으로 적을 것 — 기존 `MaxAllowedHostRunes` 주석(44-55행)이 본보기다: 합계 2048 룬은 헤더에서 세 번 쓰여 약 6KB 가 되고, 프록시가 흔히 두는 응답 헤더 8KB 한계 아래에 남으며, 현실적 origin(≈30자) 60여 개를 담는다.
  - `internal/tracking/tracking_test.go` — Validate 의 거절/통과 경계 테스트(상한 바로 위/아래, 룬 경계로 세는지 확인할 한글 호스트 1건).
  - `internal/api/trackingallowlist_live_test.go` — 2026-09-28 회차가 만든 live 파일. 여기에 (1) 목록 PUT 거절 뒤 DB 미변경, (2) 한 번 클릭 허용의 반복이 상한에서 멈춤을 실제 라우터(`server.Handler()`)+관리자 세션/CSRF 로 추가. 이 파일이 이미 그 배선을 세우고 있으니 새 파일을 만들지 말고 붙일 것.
  - 헤더 길이 확인(수용 기준 3c)은 DB 없이 `internal/api` 의 `pagePolicy(settings, nonce)` 를 직접 불러 문자열 길이를 재는 편이 싸다 — `internal/api/tracking_test.go` 에 추가.
- 검증 명령:
  - `go test ./internal/tracking ./internal/api` (DSN 없이 통과 — 2026-09-30 이 저장소에서 실행 확인, api 약 2초)
  - `go test -race ./internal/tracking ./internal/api` (CI 가 -race 로 돈다)
  - live 부분은 `AGENTHUB_TEST_DSN` + base64 32바이트 `AGENTHUB_ENCRYPTION_KEY` 가 필요하다. 없으면 선례대로 `docker run --rm -d -p 55447:5432 -e POSTGRES_PASSWORD=… postgres:16-alpine` 로 일회용 DB 를 띄우고 `go test -p 1 ./internal/api` 를 돌릴 것. DSN 없는 성공은 DB 동작의 증명이 아니므로, 그 경우 브리프에 "live 미실행" 이라고 적고 단위 수준 증명만 주장할 것.
  - `runtime-images.json` 의 base sourcePaths 는 Dockerfile.base·go.mod·go.sum·`cmd/runtime-proxy`·`internal/dlp`·`internal/policy`·deploy/runtime/*.sh 뿐이다. `internal/tracking`·`internal/api` 만 바꾸면 BASE_VERSION 상향은 필요 없다(2026-09-28 회차와 같은 판정).
- 위험과 피할 것:
  - **정규화가 아니라 거절로** 할 것. 바로 앞 회차가 같은 이유(관리자가 읽어 낸 출처는 그들이 허용한 그 값이어야 한다)로 거절을 골랐고, 여기서 조용히 잘라 쓰면 그 결정과 어긋난다.
  - 기존 항목별 검사(`originOf`/`http` 접두/세미콜론/300 룬)를 건드리지 말 것 — 그 세 줄은 2026-09-28 에 들어왔고 테스트가 고정하고 있다.
  - `splitHosts`·`AddAllowedHost`·`PolicySources`·`pagePolicy` 의 동작을 바꾸지 말 것. 이번 변경은 **쓰기 경로의 거절 하나**다. `PolicySources` 에서 자르면 저장값과 헤더가 어긋나 같은 값을 읽는 두 경로가 갈라진다(운영자가 반복해 지적한 실패 유형).
  - 두 쓰기 경로가 모두 `Validate()` 를 지나는 것은 2026-09-30 에 직접 확인했다: 설정 PUT 은 `internal/api/routes.go:1728` 의 `case tracking.SettingKey:`(`decodeTrackingSettings` → `settings.Validate()`), 한 번 클릭 허용은 `internal/api/tracking.go:274` 의 `settings.Validate()`. 둘 다 저장 **전** 에 거절하므로 수용 기준 1 의 "DB 미변경" 은 이 배선으로 자동으로 성립한다 — 다만 테스트로 실제 확인할 것.
  - 보호 경로(auth·session gateway·store 마이그레이션·.github/workflows)는 건드리지 않는다.
  - 상한을 너무 낮게 잡지 말 것 — 한 번 클릭 허용이 정상 운영 중에 막히면 그것이 회귀다. 2048 룬/64 항목이면 실사용에서 닿지 않는다.
  - Bash 승인이 명령마다 다르다. `&&` 로 잇지 말고 한 번에 하나씩 보낼 것.
- 차선 후보: guide-shots 의 problems 요약이 복원 실패로 건너뛰어지는 것을 고친다 — `web/scripts/guide-shots.mjs:130-146`. 130행 `await withGuideSettings(...)` 가 던지면 138행 `if (problems.length)` 요약과 `process.exitCode = 1` 이 실행되지 않고 145행 finally 의 `browser.close()` 로 빠져, 촬영 중 무엇이 어긋났는지 목록이 사라진다(2026-09-30 에 현재 코드에서 재확인). 요약 출력을 try/finally 로 옮기고, 던져진 오류는 그대로 재전파할 것. 테스트는 `web/scripts/guide-settings-check.test.mjs` 의 vm 하니스를 쓰되 슬라이스 앵커(`  if (problems.length)`)가 옮겨 가는 것에 주의 — 앵커를 새 위치의 첫 줄로 옮겨야 한다. 검증: `cd web && node --test scripts/guide-settings-check.test.mjs`, `node --check scripts/guide-shots.mjs`.
