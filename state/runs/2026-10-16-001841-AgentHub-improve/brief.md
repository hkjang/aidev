- 과제: 런타임 게이트웨이가 DLP 보고를 거절당한 것을 로그로 남긴다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `cmd/runtime-proxy/dlp.go:144 send()` 는 전송 오류(`client.Do` 의 err)만 `log.Printf("agenthub-dlp report failed: …")` 로 남기고, 응답이 401·403·400·500 이어도 `_ = response.Body.Close()` 로 조용히 끝낸다(:155-160, 직접 확인). 그래서 토큰이 만료되거나 본문 계약이 어긋나 관제면이 보고를 전부 거절하는 동안 Pod 로그는 완전히 깨끗하고, 관리자 감사 추적의 `dlp.tool` 행만 사라진다 — 운영자가 알 방법이 없다. 상태 코드 한 줄을 남기면 전송 실패와 거절이 같은 눈높이에서 보인다.
- 수용 기준:
  1) 관제면이 비-2xx(예: 401, 500)로 답하면 Pod 로그에 **상태 코드가 포함된 한 줄**이 남는다(예: `agenthub-dlp report rejected: 401 Unauthorized`). 기존 `report failed` 줄과 구분되는 문구여야 한다.
  2) 2xx(200/202/204) 응답에서는 **아무 줄도 늘지 않는다** — 보고 한 건마다 로그가 두 줄이 되면 기존 `scanlimit_test.go:70-78` 이 지키는 "조용한 정상 경로" 의도가 깨진다. 또한 거절이 호출 자체를 실패시키거나 지연시키지 않는다(`send` 의 best-effort 계약 유지, `scanner.record` 반환값 없음).
  3) 테스트가 증명할 것: 실제 `scanner.inspect(...)` → `record` → `send` 경로를 타고(손으로 만든 분기 호출이 아니라 `scannerFor(...)` + httptest 관제면, `scanlimit_test.go:21-78` 과 같은 방식) ① 401 을 돌려주는 관제면에서 캡처한 로그에 상태 코드가 있다 ② 202 를 돌려주는 관제면에서는 거절 줄이 없다 ③ 거절에도 `inspect` 의 반환값(치환 문자열·finding)이 바뀌지 않는다.
- 건드릴 파일:
  - `cmd/runtime-proxy/dlp.go:144 send()` — `r.client.Do` 성공 후 `response.StatusCode` 가 200..299 밖이면 `log.Printf` 로 상태를 남긴다. `response.Body.Close()` 는 지금처럼 반드시 호출(지금 코드는 `_ = response.Body.Close()` 한 줄; defer 로 바꾸든 유지하든 Close 누락 금지).
  - `cmd/runtime-proxy/dlpreport_test.go` (신규) — 위 세 단정. 로그 캡처는 `log.SetOutput(buf)` 후 `t.Cleanup` 으로 되돌린다(이 패키지의 `approval_test.go`·`main_test.go`·`mcp_test.go`·`encoding_test.go` 가 이미 `log.SetOutput` 또는 `httptest.NewServer` 를 쓴다 — 거기서 쓰는 헬퍼가 있으면 새로 만들지 말고 그것을 쓸 것).
  - `BASE_VERSION` — 현재 `0.26.0`. `runtime-images.json:5-24` 의 `base` 이미지 `sourcePaths` 에 `cmd/runtime-proxy` 가 있으므로(직접 확인) 올려야 한다. 0.27.0 을 권장.
  - (조건부) 다른 `*_VERSION` 파일 — `check-versions` 가 추가로 이름을 대면 그것만. `runtime-images.json` 에서 `buildDependencies` 가 비어 있지 않은 이미지는 `jupyter`(:137) 하나뿐이고, 그것이 `base` 를 가리키는지와 스크립트가 의존성을 전파하는지는 **미확인** — 스크립트 출력에 따를 것. 프로덕션 코드 파일은 1개, 전체 4개 안쪽.
- 검증 명령:
  - `go test ./cmd/runtime-proxy` — HEAD 에서 실제로 돌려 `ok … 0.384s` 확인했다.
  - `go test -race -p 1 ./cmd/... ./internal/...` (DSN 없으면 live 는 skip)
  - `go build ./...`, `go vet ./cmd/runtime-proxy`, `gofmt -l cmd/runtime-proxy`
  - `bash scripts/release-catalog-images.sh check-versions` — **이번 회차에서 실행 권한이 없어 돌리지 못했다(미확인)**. 구현자는 수정 전/후 모두 돌려서, 수정 후 "every image whose inputs changed … already has a new version" 가 다시 나오는 것까지 확인할 것. `validate` 도 함께.
  - 실패 재현: 상태 검사를 넣기 **전에** 401 관제면으로 새 테스트를 돌려 "거절인데 로그가 비어 있다" 로 실패하는 것을 먼저 보고, 넣은 뒤 통과를, 그 줄만 다시 지워 재실패를 확인할 것.
- 위험과 피할 것:
  - **응답 본문을 로그에 찍지 말 것.** 상태 코드(와 필요하면 메서드·경로)만 남긴다. 관제면이 되돌려 준 본문에는 보낸 이벤트가 그대로 들어 있을 수 있고, 이 저장소의 반복된 교훈은 "검출/스캔 대상 문자열은 식별자만 남긴다" 다. 같은 이유로 `r.token` 을 절대 로그에 넣지 말 것.
  - `send` 를 blocking/재시도/에러 반환으로 바꾸지 말 것 — :139-143 주석이 best-effort 를 명시적 계약으로 적어 두었고, 거절된 보고 때문에 도구 호출이 실패하면 안 된다. 큐·재시도·백오프는 이번 범위 밖.
  - `record`(:107-117)의 기존 `log.Printf("agenthub-dlp %s", raw)` 한 줄은 건드리지 말 것. finding 자체는 이미 Pod 로그에 있다.
  - 연결 재사용을 위한 본문 드레인(`io.Copy(io.Discard, …)`)은 **이번 범위 밖** — 하고 싶어도 별도 회차로.
  - `internal/dlp`·`internal/policy`·`cmd/runtime-proxy/approval.go` 의 같은 이름 env 상수(`approval.go:33-35`)는 공유 상수이므로 수정하지 말 것. 보호 경로(auth·session·`internal/store/migrations`·`.github/workflows`)는 건드리지 않는다.
  - `BASE_VERSION` 을 잊는 것이 이 저장소에서 반복된 실패 지점이다("runtime 이미지 버전 누락"). 커밋 전에 check-versions.
- 차선 후보: **guide-shots 의 `problems` 요약이 복원 실패로 사라지지 않게 한다** (2/1/S) — `web/scripts/guide-shots.mjs:129` 의 `await withGuideSettings(...)` 가 throw 하면 바로 아래 `:138-143` 의 `n건이 계획대로 되지 않았습니다` 요약과 `process.exitCode = 1` 이 건너뛰어져, 그때까지 모은 실패 목록(`:67 problems`, `:70 note`)이 전부 사라진다(이번 회차에 코드로 직접 확인). 요약 출력을 `finally` 쪽으로 옮기되 `withGuideSettings` 의 "네 복원을 모두 시도하고 원래 오류를 우선한다" 계약과 `browser.close()` 순서는 바꾸지 말고, `web/scripts/guide-settings-check.test.mjs` 의 실제 source 실행 방식(해당 테스트는 source 를 `requestTimeoutMs` 부터 `if (problems.length)` 전까지 잘라 쓴다 — **그 종료 앵커가 바로 이 줄이므로 테스트의 슬라이스 경계도 함께 손봐야 한다**)에 주의할 것. 검증: `cd web && node --test scripts/*.test.mjs`(HEAD 기준 154건 pass), `node --check web/scripts/guide-shots.mjs`.
