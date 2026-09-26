- 과제: 업스트림 리다이렉트의 HTTPS→HTTP 강등 차단 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/upstage/client.go`의 `httpClient()`가 `CheckRedirect`에서 `c.checkURLAllowed(request.URL)`로 **호스트만** 검사하고 스킴은 보지 않는다. `performParseRequest`는 본문을 `*bytes.Buffer`로 넘기므로(`client.go:277` `http.NewRequestWithContext(ctx, POST, requestURL, &body)`) Go가 `GetBody`를 채워 307/308 리다이렉트에서 멀티파트 본문을 **그대로 재전송**한다. 즉 `https://api.upstage.ai/...`로 나간 요청이 같은(또는 허용 목록에 있는) 호스트의 `http://`로 리다이렉트되면, 마스킹 대상 원문 PII가 평문으로 네트워크에 다시 실린다(같은 호스트라면 Go가 `Authorization: Bearer …`까지 유지할 가능성이 높으나 이 세션에서 Go 소스로 확인하지 못했다 — **미확인**. 본문 재전송만으로도 수정 근거는 충분하다). 최초 요청이 https였으면 리다이렉트도 https만 따르게 하면 이 평문 경로가 닫히고, 내장 mock 같은 http 시작 체인은 영향이 없다.
- 수용 기준:
  1) 최초 요청이 https인 체인에서 `http://` 대상으로의 리다이렉트는 요청이 오류로 끝나고(`Client.Do`가 에러 반환), 강등 대상 서버 핸들러가 호출되지 않는다.
  2) 최초 요청이 http인 기존 동작은 불변 — `TestParseDocumentFollowsRedirectToAllowedHost`(client_test.go:244)와 `TestParseDocumentRejectsRedirectToDisallowedHost`(client_test.go:205)가 수정 없이 통과하고, 내장 mock 경로(`internal/app` 통합 테스트)도 통과한다.
  3) 새 테스트가 수정 전에는 실패하고 수정 후 통과하는 것을 확인한다(구현자가 직접 되돌려 볼 것).
- 건드릴 파일 (프로덕션 1개):
  - `internal/upstage/client.go:httpClient` — `CheckRedirect` 안에서 리다이렉트 허용 판단을 순수 헬퍼로 분리한다. 예: `func redirectAllowed(originalScheme string, target *url.URL) bool`(또는 오류를 돌려주는 형태) — `strings.EqualFold(originalScheme, "https")`인데 `target.Scheme != "https"`면 거부. 원 스킴은 `via[0].URL.Scheme`에서 얻는다(`via`는 이미 보낸 요청들이고 0번이 최초 요청). 기존 `c.checkURLAllowed(request.URL)` 호출은 그대로 두고 그 **앞** 또는 뒤에 스킴 검사를 더한다. 거부 사유는 기존 `blockedHostError`와 섞지 말고 별도 에러(예: `fmt.Errorf("refusing to follow a redirect from https to %s", target.Scheme)`)로 두어 진단 문자열이 "차단된 호스트"로 오인되지 않게 한다. 왜 막는지(본문·인증 헤더 평문 재전송) 한 줄 주석을 남길 것.
  - `internal/upstage/client_test.go` — (a) 새 헬퍼의 테이블 테스트(https→https 허용 / https→http 거부 / http→http 허용 / http→https 허용 / 대소문자 `HTTPS` 처리), (b) 기존 리다이렉트 테스트 옆에 http 시작 체인이 여전히 따라가는지 확인하는 회귀 단언. **httptest.NewTLSServer로 end-to-end https 테스트를 시도하지 말 것** — `httpClient()`가 기본 Transport를 쓰므로 자체 서명 인증서에서 TLS 검증 실패로 원인이 흐려진다. https 축은 헬퍼 단위 테스트로 덮는다.
  - (선택) `README.md` — 업스트림/보안 문단에 "https로 나간 요청은 http로 리다이렉트되지 않는다" 한 문장. 이 저장소 관례상 동작 변경 시 한 문장 추가.
- 검증 명령 (이 저장소에서 실제로 도는 것, 전체 2초 미만):
  - `go test -count=1 ./internal/upstage`
  - `go test -count=1 ./...`  (2026-09-26 22:0x 기준 전부 통과 확인함)
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal` (무출력이어야 함) / `git diff --check`
  - `go test -race -count=3 ./internal/upstage ./internal/app`
- 위험과 피할 것:
  - `hostAllowed`의 "bare host 항목은 모든 포트를 허용" 규칙은 주석에 명시된 **의도된 계약**이다(client.go:497-498). 이번 과제에서 손대지 말 것 — 허용 목록 의미를 바꾸면 docker-compose/기본 설정이 조용히 깨진다.
  - `allowedHosts()`의 폴백(base URL 호스트)과 `config.normalizeAllowHosts`도 건드리지 말 것. 이 회차의 범위는 스킴 한 축이다.
  - 내장 mock 업스트림은 http이고 `app.New`가 같은 mux에 마운트한다(2026-09-26 회차). http 시작 체인을 막으면 mock 전체가 죽으므로, 검사 조건을 반드시 "최초가 https일 때만"으로 한정할 것.
  - 이 저장소의 반복 함정은 "같은 값을 읽는 경로가 여럿"이다. 리다이렉트 검사는 `httpClient()` 한 곳에만 있으므로(`checkHostAllowed`는 최초 URL 전용) 두 곳을 동시에 고칠 필요는 없지만, 최초 요청 자체의 스킴은 검사 대상이 아님(사용자가 명시적으로 http 업스트림을 설정하는 것은 허용된 구성)을 헷갈리지 말 것.
- 차선 후보: `internal/config` Load 경유 나머지 정규화·기본값 테이블 테스트 (2/1/S) — `normalizeAllowHosts`/`normalizePIILang`/`normalizePIISchema`/`normalizeAuthMode`/`envInt`/`envNonNegativeInt`/`envBool`가 `Load()` 경유 검증이 없다. `t.Setenv`를 쓰므로 `t.Parallel` 금지.
