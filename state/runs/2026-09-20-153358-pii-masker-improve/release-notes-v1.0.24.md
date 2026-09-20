## 결과 다운로드 URL에 `HEAD` 요청이 `405`로 거절되던 문제 수정

`routes()`는 `/v1/jobs/{job_id}/result`를 `Methods(http.MethodGet)`으로만 등록했습니다. gorilla/mux는 등록되지 않은 메서드를 핸들러에 닿기 전에 `405 Method Not Allowed`로 끊기 때문에, `HEAD` 요청은 `Allow` 헤더도 없이 거절됐고 `documentResponse` 래퍼도 실행되지 않아 `Cache-Control`·`Content-Length`·`Content-Disposition`이 모두 비어 있었습니다. 다운로드 관리자, `curl -I`, 리버스 프록시처럼 **본문을 받기 전에 파일의 존재와 크기를 먼저 묻는 클라이언트**는 `download_url`이 유효한데도 실패로 판정했습니다.

- 결과 다운로드 라우트에 `http.MethodHead`를 추가했습니다. 핸들러는 이미 `http.ServeContent`를 쓰므로 `HEAD`에 대해 GET과 같은 헤더(`Content-Type`, `Content-Length`, `Content-Disposition`, `Accept-Ranges: bytes`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`)를 본문 없이 돌려줍니다
- 없는 job ID에 대한 `HEAD`는 GET과 마찬가지로 `404`로 도착합니다
- JSON 라우트(`/v1/jobs/{job_id}`, `/v1/history` 등)는 `Content-Length`를 직접 계산하지 않아 `HEAD`의 의미가 달라 그대로 두었습니다. gorilla/mux `405` 응답의 `Allow` 헤더 부재는 라우터 전체 동작 변경이라 별개 과제로 남겼습니다

검증: 실제 리스너와 mock 업스트림을 지나는 통합 테스트 `TestJobResultAnswersHeadProbes` 추가 — 완료된 PDF job에 GET으로 본문 길이를 잰 뒤 `HEAD` → `200`, `Content-Length`가 GET 본문 길이와 같고 본문은 0바이트, 위 헤더들이 GET과 동일 / 없는 job ID에 `HEAD` → `404`. 라우트를 `Methods(http.MethodGet)`으로 되돌리면 이 테스트가 `unexpected head status 405 (allow="")`로 실제로 실패하는 것을 확인했습니다. 기존 `TestJobResultServesRangeRequests`·`TestJobResultReturnsNotFoundWhenFileIsGone`·`TestJobResultSanitizesInjectedUploadFilename`은 수정 없이 통과, `gofmt`·`go vet`·`go build`·`go test -count=1 ./...`·`go test -race -count=3 ./internal/httpapi/...` 전부 통과. README 엔드포인트 목록에 `HEAD` 허용 한 줄 추가.
