- 과제: 한 번 클릭 허용(`POST /api/v1/admin/tracking/violations/allow`)이 **한 개의 출처만** 받고, 감사 기록이 실제로 저장된 것을 말하게 한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/api/tracking.go:254 allowTrackingOrigin` 은 받은 문자열을 "`http` 로 시작하는가" 만 보고 `tracking.AddAllowedHost` 로 목록에 **그대로 이어 붙이는데**, 뒤에서 그 목록을 읽는 `splitHosts`(`internal/tracking/tracking.go:432`)는 쉼표·공백·탭·줄바꿈을 구분자로 쓴다. 그래서 "출처 하나를 더한다" 는 이 경로가 `{"origin":"https://a.corp.example,https://evil.corp.example"}` 한 번으로 **두 항목**을 저장하고, 감사 행에는 저장된 것과 다른 한 덩어리 문자열이 `details.origin` 으로 남는다. 또 이미 목록에 있는 출처를 다시 허용하면 `AddAllowedHost` 가 `existing` 을 그대로 돌려줘 **아무것도 바뀌지 않는데도** `settings.update` / `"success"` 감사 행이 한 건 더 남아, 감사 추적만 보고는 목록이 언제 실제로 늘었는지 알 수 없다. 고치면 이 경로의 계약("출처 하나")과 감사 행이 저장된 값과 일치한다.

- 수용 기준:
  1) `{"origin":"https://a.corp.example,https://evil.corp.example"}` 과 `{"origin":"https://a.corp.example https://evil.corp.example"}` 이 **400** 으로 거절되고(`invalid_origin`), 저장된 `allowedHosts` 는 한 글자도 바뀌지 않는다. 현재는 200 과 함께 두 항목이 저장된다.
  2) 감사 행 `details.origin` 이 **저장된 형태**(앞뒤 공백·끝 `/` 를 떼고 `AddAllowedHost` 가 실제로 넣은 값)와 글자 단위로 같다. 예: `{"origin":" https://a.corp.example/ "}` 를 허용하면 저장은 `https://a.corp.example`, 감사도 `https://a.corp.example`.
  3) 이미 있는 출처를 다시 허용하면 목록이 그대로이고 응답은 여전히 200(화면이 쓰는 `{"allowedHosts":…}` 형태 유지), 그러나 감사 행이 그것을 구분할 수 있다 — `details` 에 변화 없음을 나타내는 값이 들어가거나, 변화가 없을 때 감사 행을 남기지 않는다. **둘 중 하나를 고르고 상수/함수 주석에 왜 그랬는지 산문으로 적을 것**(이 저장소 관례: `internal/tracking/tracking.go:46-78` 의 주석 밀도).
  4) 테스트가 증명해야 하는 것: (a) 쉼표·공백이 섞인 한 번의 POST 로 목록 항목 수가 늘지 않는다, (b) 감사 `details.origin` == 저장된 항목, (c) 중복 허용이 목록을 바꾸지 않는다. (a)~(c) 를 **실제 라우터·관리자 세션·CSRF** 로 통과시킬 것 — 손으로 만든 핸들러 호출이 아니라 `Server.Handler()` 로.

- 건드릴 파일 (프로덕션 2개):
  - `internal/api/tracking.go:254 allowTrackingOrigin` — 받은 문자열이 **단일 항목인지** 먼저 확인한 뒤(아래 참고) `AddAllowedHost` 를 부르고, 감사의 `details["origin"]` 을 저장된 값으로 바꾼다. 거절 메시지는 기존 `invalid_origin` 코드와 한국어 문장 스타일을 유지.
  - `internal/tracking/tracking.go` — 단일 항목 판정을 패키지 안에 두려면 작은 노출 헬퍼 하나(예: `SingleHost(raw) (string, bool)`, 내부에서 기존 `splitHosts` 를 그대로 써서 `len(hosts)==1` 이면 그 항목을 돌려준다). `splitHosts`·`AddAllowedHost`·`Validate`·`PolicySources`·`pagePolicy` 의 **기존 동작은 바꾸지 말 것** — 설정 폼(여러 항목을 한 textarea 로 받는 경로)은 이 판정을 거치면 안 된다.
  - 테스트(프로덕션 아님): `internal/api/trackingallowlist_live_test.go` 에 서브테스트 추가가 자연스럽다 — 이미 `TestAnAllowedOriginCannotReachThePolicyHeaderUnchecked` 가 실제 DB·관리자 세션·CSRF 로 두 쓰기 경로(설정 폼, `/admin/tracking/violations/allow`)를 다 통과한다(파일 1-60행에서 확인). 감사 행은 `db.AuditTrail` 로 읽는 선례가 `internal/api/dlpreport_live_test.go` 에 있다(2026-09-27 회차). 단일 항목 헬퍼 자체는 `internal/tracking/tracking_test.go` 의 표 테스트로 DB 없이 고정.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go build ./...`
  - `go vet ./internal/api ./internal/tracking`
  - `go test ./internal/tracking ./internal/api` — DSN 없이도 통과한다(live 파일은 `t.Skip`). 2026-09-30 회차에 실측됨.
  - live: DSN 이 없으면 `docker run --rm -d -p 55447:5432 -e POSTGRES_PASSWORD=… postgres:16-alpine` 로 일회용 DB 를 띄워 `AGENTHUB_TEST_DSN` 과 base64 32바이트 `AGENTHUB_ENCRYPTION_KEY` 를 주고 `go test -race -p 1 ./internal/api -run TestAnAllowedOriginCannotReachThePolicyHeaderUnchecked -v`. 전체는 `go test -race -p 1 ./cmd/... ./internal/...`. (이 선례는 2026-09-27·09-30 회차에 실제로 돌았다.)
  - BASE_VERSION: `runtime-images.json` 의 어떤 이미지도 `internal/api`·`internal/tracking` 을 `sourcePaths` 에 두지 않으므로 상향 불필요 — **매니페스트를 직접 읽어 다시 확인할 것**(스크립트 실행이 Bash 승인에 막힌 선례가 있다).

- 위험과 피할 것:
  - **설정 폼 경로를 같이 건드리지 말 것.** `routes.go` 의 `case tracking.SettingKey:`(약 1728행) 는 textarea 한 덩어리를 받아 여러 항목을 저장하는 것이 정상이다. 단일 항목 판정은 `allowTrackingOrigin` **한 곳에만** 걸어야 한다. 두 경로가 같은 값을 다르게 읽게 만들지 말 것(운영자 지시: 같은 값을 읽는 경로가 둘이면 end-to-end 로 둘을 함께 확인).
  - `Validate()` 를 손대지 말 것. 2026-09-28·09-30 두 회차가 여기에 상한 네 개를 넣었고 테스트가 그 숫자를 고정하고 있다.
  - 응답 형태를 바꾸지 말 것 — `web/src/pages/AdminSettings.tsx:201` 이 `{allowedHosts: string}` 을 읽어 화면을 갱신한다. 프런트는 이번 회차에서 건드리지 않는다.
  - 감사에 **스크럽 이전 원문을 넘기지 말 것**(운영자가 되풀이한 지시). 여기서 넘겨야 하는 것은 저장된 항목이다.
  - 손으로 만든 대역으로 증명하지 말 것 — 이 결함은 핸들러·`AddAllowedHost`·`splitHosts` 의 **배선 차이**이므로 프로덕션 라우터를 통과해야 보인다.
  - 보호 경로(`internal/api/auth.go`, `mcpoauth.go`, `internal/store` 마이그레이션, `.github/workflows`)는 건드리지 않는다. 이 과제는 그 어느 것도 필요하지 않다.
  - 미확인: `AuditTrail` 이 `details` 를 그대로 돌려주는지는 이번 세션에서 열어 보지 않았다(2026-09-27 회차 기록으로만 알고 있다). 수용 기준 2 의 검증 방법은 구현자가 `internal/store/audit.go` 를 먼저 읽고 확정할 것.
  - 미확인: `decodeJSON` 의 본문 상한(프로필은 `respond.go:99` 의 1MiB 로 적고 있다)은 이번에 직접 확인하지 않았다. 단일 항목 판정이 앞에 서면 길이 문제는 `Validate()` 의 300룬 상한으로 이미 막히므로 새 상한을 더하지 말 것.

- 차선 후보: **콘솔 설정 화면이 허용 출처 목록의 상한(64개 / 4096자)을 미리 알려 준다** — `web/src/pages/AdminSettings.tsx` 의 tracking 탭 textarea 는 placeholder 만 있고 상한을 말하지 않아, 관리자는 저장을 눌러 400 을 받아야 안다. 프로덕션 파일 1개. 숫자를 프런트에 손으로 적으면 `MaxAllowedHostEntries`·`MaxAllowedHostsTotalRunes` 와 어긋날 수 있으니, 설정 GET 응답에 상한을 실어 보내는 편이 맞는지 먼저 판단할 것(그렇게 하면 프로덕션 파일이 2개가 된다).
