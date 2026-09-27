- 과제: 인증 없이 들어오는 CSP 위반 보고의 문자열을 룬 경계로 잘라 관리자 화면에 Pod/브라우저가 정한 크기가 그대로 실리지 않게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/api/server.go:130` 이 `POST /api/v1/tracking/csp-report` 를 세션·CSRF 없이 등록하고(같은 파일 127-129행 주석이 의도를 밝힘), `receiveCSPReport` 는 본문만 16KiB(`internal/api/tracking.go:41 maxCSPReportBytes`)로 막은 뒤 `blocked-uri`·`effective-directive`·`document-uri` 를 검사 없이 `Recorder.Record` 로 넘긴다. `internal/tracking/violations.go:46 Record` 는 `page` 만 `page[:200]` 으로 자르는데 이것이 **바이트** 절단이라 한글 document-uri 는 잘린 룬이 남아 JSON 응답에 U+FFFD 가 나가고, `origin`·`directive` 에는 상한이 아예 없어 한 번의 무인증 POST 로 ~16KiB 문자열이 관리자 화면(`GET /api/v1/admin/tracking/violations`)과 맵 키에 그대로 들어간다.
- 수용 기준:
  1) `Record` 가 저장하는 `Violation.Origin`·`Directive`·`Page` 세 필드가 각각 상한 룬 수 이하이고 `utf8.ValidString` 이 참이다 — 16KiB 짜리 blocked-uri/directive/document-uri 를 넣어도 마찬가지다.
  2) 한글(또는 임의 멀티바이트) document-uri 를 200자 넘게 넣으면 결과가 정확히 상한 **룬** 수이고 잘린 바이트가 남지 않는다. 수정 전 코드에서 이 테스트가 "200바이트 = 66자 + 깨진 룬" 으로 실패하는 것을 먼저 확인한다.
  3) 절단은 맵 키(`directive + " " + strings.ToLower(origin)`, violations.go:66)를 만들기 **전에** 일어난다 — 같은 과대 origin+directive 로 두 번 보고하면 항목은 1개, `Count` 는 2 다(절단 후에 키를 만들면 이 성질이 깨진다).
  4) 평범한 짧은 보고는 값이 한 글자도 바뀌지 않는다 — `internal/tracking/tracking_test.go:174-209` 의 기존 기대값이 그대로 통과한다.
  5) 손으로 만든 대역이 아니라 프로덕션 배선으로 한 번 증명한다: `internal/api/tracking_test.go:216 TestPolicyViolationsAreRecordedByOrigin` 이 이미 `trackingServer(t, settings)` 로 세운 실제 `Server` 의 `server.Handler().ServeHTTP` 에 `httptest.NewRequest(http.MethodPost, tracking.ReportPath, …)`(Content-Type `application/csp-report`)로 POST 하고 `server.violations.List(settings)` 로 읽는다(DB 불필요). 같은 형태로 과대·멀티바이트 문자열 보고 케이스를 한 건 더해 핸들러→Record→List 를 통과시킨다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/tracking/violations.go` — 상한 상수(예: `maxOriginRunes`/`maxDirectiveRunes` 와 기존 200 을 `maxPageRunes` 로) + 룬 경계 절단 헬퍼를 두고, `Record` 안에서 `originOf`/`directive` 정규화 **뒤**, 맵 키 계산 **앞**에 세 값을 자른다. `page[:200]` 바이트 절단을 헬퍼 호출로 교체한다. `evictOldest`·`List`·`matchesWildcard`·`MaxViolations` 는 건드리지 않는다.
  - `internal/tracking/tracking_test.go` — 위 1)~4) 회귀 테스트 추가.
  - `internal/api/tracking_test.go` — 5) 의 배선 테스트 1건 추가.
- 검증 명령:
  - `go test ./internal/tracking ./internal/api` — 이번 회차에 실제로 실행했고 DSN 없이 둘 다 `ok` 다(api 1.5초). 이것이 주 검증이다.
  - `go build ./... && go vet ./internal/tracking ./internal/api`
  - `go test -race -p 1 ./cmd/... ./internal/...` (CI 와 같은 형태)
  - BASE_VERSION 상향은 **필요 없다** — `runtime-images.json` 의 base 항목 `sourcePaths` 는 `cmd/runtime-proxy`·`internal/dlp`·`internal/policy` 뿐이고 `internal/tracking` 은 없다(이번 회차에 파일로 확인). 확인차 `bash scripts/release-catalog-images.sh check-versions` 를 돌려도 좋다(이번 회차에는 승인 거부로 미실행).
- 위험과 피할 것:
  - `internal/api/server.go`·`auth.go`·`catalog.go` 의 라우팅/인증은 건드리지 말 것. 이 과제는 저장 직전 값 정규화만이다.
  - 상한을 `origin` 에 너무 낮게 잡으면 정당한 긴 호스트가 잘려 `allowTrackingOrigin` 의 한 번 클릭 허용이 쓸모없는 문자열을 설정에 넣는다. origin 은 scheme+host 뿐이므로 253 룬(DNS 이름 한계)보다 넉넉히, 예컨대 300 룬 정도로 잡고 그 근거를 상수 주석에 남길 것. directive 는 CSP 디렉티브 이름이므로 짧게(예: 64 룬) 잡아도 되지만, 너무 짧으면 미래 디렉티브가 잘린다.
  - 절단 결과가 `matchesWildcard`/`List` 의 `allowed` 비교와 어긋나지 않는지 볼 것 — 잘린 origin 은 허용 목록과 절대 일치하지 않으므로 `Allowed=false` 로 남는다(의도된 동작이지만 테스트로 고정할 것).
  - 2026-09-09 교훈: 감사·로그로는 식별자만 넘길 것. 이 과제는 감사 테이블을 건드리지 않지만, `allowTrackingOrigin`(tracking.go:283)이 `"origin": origin` 을 감사 details 로 넘기는 것은 **다른 경로**다 — 이번 회차 범위 밖으로 두고 손대지 말 것(아이디어 파일에 남김).
  - 문자열 grep 을 증거로 제출하지 말 것. 반드시 수정 전 실패를 먼저 보이고 그 출력을 회차 노트에 적을 것.
- 확인함: `internal/tracking/violations.go`·`internal/tracking/tracking.go:352 originOf`·`internal/api/tracking.go` 전문, `internal/api/server.go:115-145` 의 무인증 등록, `internal/api/catalog.go:260-261` 의 관리자 조회/삭제, `internal/api/tracking_test.go:216-252`, `runtime-images.json:8-30` 을 이번 회차에 직접 열었다. `go test ./internal/tracking ./internal/api` 를 실행해 통과를 봤다.
- 미확인: `page[:200]` 이 바이트 절단이라는 것은 Go 문자열 인덱싱의 성질로 확실하지만, 깨진 룬이 `writeJSON` 을 거쳐 실제로 U+FFFD 로 나가는 것은 이번 회차에 실행으로 확인하지 않았다(구현자가 수정 전 실패 출력으로 남길 것). 관리자 콘솔(`web/src`)이 이 값을 어떻게 렌더링하는지도 미확인 — 프런트는 건드리지 말 것.
- 차선 후보: 복원 실패가 `web/scripts/guide-shots.mjs:130-144` 의 problems 요약 출력을 건너뛰게 하는 것을 고친다 (2/1/S) — `withGuideSettings` 가 던지면 138행의 `if (problems.length)` 요약과 `process.exitCode=1` 이 건너뛰어져 운영자가 스택만 본다. 요약을 finally 로 옮기거나 호출을 try/catch 로 감싸 요약을 찍고 원본 오류를 재전파한다. 주의: `web/scripts/guide-settings-check.test.mjs` 하니스의 소스 슬라이스 끝 앵커가 `  if (problems.length)` 라 그 줄을 옮기면 앵커도 옮겨야 한다.
