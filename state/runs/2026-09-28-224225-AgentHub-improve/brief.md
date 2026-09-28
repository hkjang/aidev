- 과제: 추적 허용 출처(allowedHosts)의 한 항목이 CSP 헤더의 지시문 경계를 넘거나 무한히 길어지지 못하게 Validate 에서 막는다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/tracking/tracking.go:305-307` 의 `PolicySources()` 는 `splitHosts(s.AllowedHosts)` 가 돌려준 문자열을 **그대로** 반환하고(다른 provider 분기 — Momento·Matomo — 는 전부 `originOf()` 를 거친다), `internal/api/tracking.go:101-110` 의 `pagePolicy()` 가 그 값을 `strings.Join(..., " ")` 로 `Content-Security-Policy` 헤더에 직접 붙인다. `Validate()` 127-131 행은 `originOf(host) != ""` 와 `http` 접두사만 보고 host 자체는 바꾸지 않으므로, 세미콜론이 든 항목(예: `https://a.corp.example/;script-src-elem`)은 검사를 통과해 저장되고 모든 페이지 응답의 정책 헤더에서 지시문을 하나 끝내고 새 지시문을 여는 값이 된다 — `splitHosts` 가 공백·쉼표·줄바꿈만 끊으므로 세미콜론은 아무 곳에서도 걸리지 않는다. 길이 상한도 없어 한 항목이 그대로 헤더와 `internal/api/tracking.go:283` 의 감사 details `"origin"` 으로 나간다. Validate 를 조이면 두 쓰기 경로가 같은 한 지점에서 막히고, 관리자가 실수로 붙여 넣은 값이 콘솔 전체의 정책을 망가뜨리는 길이 닫힌다.

- 수용 기준:
  1) `허용 출처` 항목에 `;` 가 들어 있으면 `Settings.Validate()` 가 기존 형식의 한국어 오류(`허용 출처 %q 는 …`)로 거절하고, 그 값은 저장되지 않는다.
  2) 한 항목이 정해진 룬 상한(권장 300 — `internal/tracking/violations.go:26` 의 `maxOriginRunes` 와 같은 값, 같은 근거: DNS 이름 253자 + scheme + 포트)을 넘으면 거절한다. 길이는 룬으로 세고 바이트로 세지 않는다.
  3) 두 쓰기 경로가 모두 막히는 것을 `internal/api` 의 실제 라우터로 증명한다 — (a) `PUT /api/v1/admin/settings/{key}`(routes.go:1736 이 `Validate()` 를 부른다), (b) `POST /api/v1/admin/tracking/violations/allow`(`allowTrackingOrigin`, tracking.go:274 가 같은 `Validate()` 를 부른다). 둘 다 400 이고 `s.store` 에 쓰기가 일어나지 않아야 한다.
  4) **회귀 금지** — 기존에 통하던 값은 그대로 통한다: 와일드카드 `https://*.corp.example`(tracking_test.go:195 가 의존), 포트 `https://b.corp.example:8443`(tracking_test.go:135), 뒤 슬래시 `https://other.corp.example/`(tracking_test.go:99), 쉼표·줄바꿈 혼합 구분자. 정규화(값 다시 쓰기)가 **아니라** 거절로 구현할 것 — 저장된 기존 설정을 조용히 바꾸면 안 된다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/tracking/tracking.go:116 Validate()` — 127-131 행의 `for _, host := range splitHosts(s.AllowedHosts)` 루프에 두 검사를 더한다: 세미콜론 포함 거절, 룬 상한 초과 거절. 상한은 파일 상단(43행 `MaxSnippetBytes` 옆)에 이름 붙은 상수로 두고, "이 값이 그대로 정책 헤더의 source 가 된다" 는 이유를 산문 주석으로 남길 것(이 저장소의 주석 밀도에 맞출 것 — violations.go:16-23 이 본보기).
  - `internal/tracking/tracking_test.go` — `TestValidate…`(116행 표 기반 테스트) 에 거절 사례와 기존 통과 사례를 더한다.
  - `internal/api/tracking_test.go` — 실제 `server.Handler()` 로 두 라우트를 때리는 테스트(기존 `trackingServer` 헬퍼 재사용, 241행 부근이 본보기). DB 없이 도는 테스트로 맞출 것.

- 검증 명령:
  - `go test ./internal/tracking ./internal/api` (2026-09-28 이 회차에 실제로 실행, DSN 없이 ok / api 1.6초)
  - `go build ./... && go vet ./internal/tracking ./internal/api`
  - `go test -race ./cmd/... ./internal/...` (CI 와 같은 형태; DSN 없이도 통과해야 한다)
  - `internal/tracking` 은 `runtime-images.json` 의 base sourcePaths(Dockerfile.base·go.mod·go.sum·`cmd/runtime-proxy`·`internal/dlp`·`internal/policy`·deploy/runtime/*.sh)에 없으므로 BASE_VERSION 상향은 필요 없다.

- 위험과 피할 것:
  - **정규화로 바꾸지 말 것.** `originOf()` 로 항목을 다시 쓰면 와일드카드 `https://*.corp.example` 가 살아남는지가 이 회차에 **미확인**이다(`url.Parse` 가 `*` 를 Host 로 받는지 실행해 확인하지 못했다 — Bash 승인 거부). 거절 방식은 이 미확인에 기대지 않으므로 그 길로 갈 것. 굳이 정규화를 하고 싶다면 먼저 `url.Parse("https://*.corp.example")` 를 실제로 돌려 보고 결정할 것.
  - `splitHosts`(364행) 와 `AddAllowedHost`(378행) 의 구분자·중복 판정은 건드리지 말 것 — 콘솔의 textarea(`web/src/pages/AdminSettings.tsx:176`)와 한 줄 추가 경로(tracking.go:273)가 그 동작에 붙어 있다.
  - `pagePolicy()`·`basePagePolicy`(internal/api/tracking.go:34,101)는 건드리지 말 것. 이번 과제는 헤더 조립이 아니라 들어오는 값을 막는 것이다.
  - `POST /api/v1/tracking/csp-report` 는 무인증 구간(server.go)이다 — 라우팅은 손대지 말 것. 이번 변경은 관리자 쓰기 경로만 건드린다.
  - 이미 저장된 설정이 새 검사에 걸리면 `allowTrackingOrigin` 의 한 줄 추가도 400 이 된다(274행이 전체 문서를 Validate 하므로). 의도된 동작이지만 오류 문구가 어느 항목인지 말해 주어야 한다 — 기존 `%q` 형식을 유지할 것.
  - 손으로 만든 Settings 값만으로 증명하지 말 것 — 수용 기준 3) 은 프로덕션 라우터를 통과해야 한다.

- 차선 후보: 복원 실패가 `web/scripts/guide-shots.mjs:130-144` 의 problems 요약 출력을 건너뛰게 하는 것을 고친다 (2/1/S) — `withGuideSettings` 가 던지면 138행 `if (problems.length)` 요약과 `process.exitCode=1` 이 건너뛰어져 운영자가 스택만 본다. 요약을 finally 로 옮기고 원본 오류를 재전파. 주의: `guide-settings-check.test.mjs` 하니스의 소스 슬라이스 끝 앵커가 `  if (problems.length)` 다. 이 회차에 재확인하지 않음(2026-09-26 기록 근거).
