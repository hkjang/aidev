- 과제: Momento 프록시가 인코딩된 경로(`%2F`·`%2E%2E%2F`)를 디코드해 수집기에 다른 경로로 전달하는 문제 수정 (가치 3 / 위험 2 / 작업량 S)
- 왜: `momentoProxy`(internal/httpapi/tracking.go:232-233)가 이미 디코드된 `r.URL.Path` 로 바깥 경로를 조립하고 `RawPath` 를 비워, 브라우저가 `%2F` 로 보낸 경로 조각이 진짜 경로 구분자로 바뀐 채 수집기에 도착한다. 정찰에서 실제 httptest 수집기의 `r.RequestURI` 로 재현했고, `/momento/collect/%2E%2E%2F%2E%2E%2Fadmin` 은 `/base/collect/../../admin` 으로 전달되어 운영자가 `momento_url` 에 지정한 경로 접두사(`/base`)를 벗어난다 — 중간 프록시나 수집기가 `..` 를 정규화하면 지정하지 않은 경로로 요청이 간다.
- 수용 기준:
  1) 실제 `routes`→`securityHeaders`→`momentoProxy` 를 통과한 요청에서, httptest 수집기가 본 **`r.RequestURI`** 가 인바운드 escape 를 보존한다: `/momento/collect/a%2Fb` → `/base/collect/a%2Fb` (지금은 `/base/collect/a/b`), `/momento/collect/%2E%2E%2F%2E%2E%2Fadmin` → `/base/collect/%2E%2E%2F%2E%2E%2Fadmin` (지금은 `/base/collect/../../admin`).
     - 수집기 쪽 `r.URL.Path` 는 Go 가 다시 디코드하므로 **단정 대상이 아니다**(고친 뒤에도 `/base/collect/a/b` 로 보인다). 반드시 `r.RequestURI` 로 단정할 것.
  2) 기존 계약이 그대로다: escape 가 없는 경로는 지금과 바이트 단위로 같게 전달되고(`/momento/collect/v1/events` → `/base/collect/v1/events`), 질의 문자열은 이미 보존되므로 그대로여야 하며(`?site=a%2Fb&q=1` 정찰에서 보존 확인), `Cookie`·`Authorization`·`X-Vault-Token` 제거, 응답의 `Set-Cookie`·`Content-Security-Policy` 제거, `ProxyActive()` 꺼짐 404, GET/HEAD/POST 외 405, `maxMomentoProxyBytes`(64KiB), 10초 타임아웃, `apiPolicy` 헤더가 모두 유지된다.
  3) 테스트가 증명할 것: 수정 전에는 1)의 두 경우가 실패하고(수집기가 디코드된 경로를 본다) 수정 후 통과하며, 기존 `TestMomentoProxyForwardsWithoutCredentials` 는 손대지 않고 그대로 통과한다. 구현자는 수정만 되돌려 같은 두 단정이 다시 실패하는 것을 확인하고 복원할 것.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/httpapi/tracking.go:momentoProxy` — `Rewrite` 콜백 안의 두 줄(`request.Out.URL.Path = target.Path + strings.TrimPrefix(r.URL.Path, …)` / `request.Out.URL.RawPath = ""`)만. 정찰에서 실제로 돌려 본 형태(기존 테스트 통과, 수용 기준 1 충족):
    ```go
    escaped := strings.TrimPrefix(r.URL.EscapedPath(), tracking.MomentoProxyPrefix)
    if decoded, err := url.PathUnescape(escaped); err == nil {
        request.Out.URL.Path, request.Out.URL.RawPath = decoded, escaped
    }
    request.SetURL(target)   // SetURL 의 joinURLPath 가 target.Path 와 RawPath 를 올바르게 잇는다
    request.SetXForwarded()
    request.Out.Host = target.Host
    ```
    핵심은 경로 조립을 손으로 하지 않고 `SetURL` 에 맡기는 것이다. `url.URL.EscapedPath()` 는 `RawPath` 가 `Path` 의 유효한 인코딩일 때만 그것을 쓰므로, `Path` 와 `RawPath` 를 짝으로 맞춰 넣어야 한다(한쪽만 넣으면 조용히 디코드된 경로로 되돌아간다). `url` 은 이미 import 되어 있다(파일에서 `url.Parse` 사용 중) — 그래도 빌드로 확인할 것.
  - `internal/httpapi/tracking_test.go` — 기존 `trackingServer`·`momentoConfig` 하네스를 재사용해 새 테스트 하나를 추가. 수집기 핸들러에서 `r.RequestURI` 를 기록할 것. `httptest.NewRequest` 의 target 에 **공백을 넣지 말 것**(정찰에서 `"/momento/collect/a b"` 가 `malformed HTTP version` 으로 panic 했다). 설정은 `tracking.ReadConfig(map[string]any{"enabled": true, "provider": "momento", "allow_insecure_http": true, "momento_url": collector.URL + "/base/", "momento_site_id": "jikim"})` 와 `server.httpClient = http.DefaultClient` 로 둘 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/httpapi/ -run 'Momento|Tracking|CSPReport' -count=1 -v`
  - `go test ./internal/httpapi/ -race -count=5`
  - `go test ./... -count=1`
  - `go vet ./...` / `gofmt -l internal/httpapi/tracking.go internal/httpapi/tracking_test.go` (출력 없어야 함)
  - `bash scripts/verify.sh` (프런트 포함, 오래 걸림 — 2026-10-15 회차에서 exit 0 이었다)
- 위험과 피할 것:
  - `decorateIndex`·`pagePolicy`·`shippedPolicy`·`receiveCSPReport` 는 건드리지 말 것. 추적 off 시 CSP 헤더가 `shippedPolicy` 와 바이트 단위로 같아야 하는 계약이 있다.
  - `momento_url` 파싱(`TrimRight`/`TrimSpace`), `ProxyActive()` gate, 메서드 허용 목록, 헤더 제거 목록, 64KiB·10초 한도는 보안 경계다. 이번 범위는 **바깥 경로 조립 한 곳**뿐이다.
  - 경로 정규화(`..` 를 직접 제거하거나 거절)를 새로 넣지 말 것 — 새 계약이 되고, escape 보존만으로 접두사 이탈이 닫힌다. 교훈: 파서를 넓히지 말고 좁히는 방향으로만 고친다.
  - 디코드되지 않은 리터럴 `/momento/collect/../../admin` 요청이 핸들러에 닿는지(ServeMux 가 먼저 리다이렉트하는지)는 **미확인**이다. 수용 기준에 넣지 말 것.
  - `internal/tracking/tracking.go` 의 스니펫 생성(`MomentoProxyPrefix` 사용처)은 이번에 바꾸지 않는다. 스니펫이 보내는 경로에는 escape 가 없다.
- 차선 후보: CSP 진단/스니펫 경로의 UTF-8 절단 방지 (2/1/S) — `page[:512]` 류 경계가 다중 바이트 문자를 쪼갤 수 있다. 다만 최종 관리자 응답까지 관찰 가능한 차이를 먼저 증명해야 하며, 정찰에서 해당 줄을 직접 열어 확인하지는 못했다(미확인).
