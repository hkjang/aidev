- 과제: 마스킹된 이미지의 실제 인코딩을 응답이 광고하는 MIME 타입과 일치시키기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `masking.MaskImageFile`(engine.go:499-503)은 인코더를 고를 때 **디코딩된 실제 포맷을 먼저** 보므로(`format == "png" || mimeType에 png 포함`), 클라이언트가 `Content-Type: image/jpeg`로 올린 **실제 PNG 바이트**는 PNG로 다시 인코딩되는데, `service.go:614-620`은 `Output.MIMEType`을 업로드가 선언한 `image/jpeg`로 그대로 채웁니다. 그래서 `/v1/jobs/{id}/result`는 PNG 바이트를 `Content-Type: image/jpeg` + `X-Content-Type-Options: nosniff`(server.go:104)로 내보내고, 스니핑이 막힌 브라우저는 결과를 못 그리며 파일 이름도 `masked_<원래>.jpg`로 내용과 어긋납니다(확장자만 바꿔 둔 PNG를 올리면 브라우저가 확장자 기준으로 `image/jpeg`를 붙이므로 일상적으로 재현됨).
- 수용 기준:
  1) 선언 MIME이 `image/jpeg`이고 내용이 실제 PNG인 업로드를 마스킹하면 결과 바이트가 JPEG(`image.DecodeConfig`/`image.Decode`가 format `"jpeg"`로 보고)이고, 같은 응답의 `output.mime_type`이 `image/jpeg`로 둘이 일치한다.
  2) 기존 동작 불변: 선언 `image/png`+실제 PNG → PNG, 선언 `image/png`+실제 JPEG → PNG(둘 다 `output.mime_type`과 일치), 선언이 비어 있거나 png/jpeg를 담지 않을 때는 지금처럼 디코딩된 `format`으로 결정된다(`format=="png"`면 PNG, 그 밖에는 JPEG).
  3) 테스트는 "응답이 광고하는 타입과 실제 바이트의 포맷이 같다"는 불변식을 증명해야 한다 — `mimeType` 인자와 반환 바이트를 함께 보는 `MaskImageFile` 테이블 테스트 + 프로덕션 배선(`app.New`→`Serve`→실제 리스너)을 지나는 `POST /v1/mask` 회귀 1개로, multipart 첫 파트의 `output.mime_type`과 두 번째 파트 바이트의 디코딩 포맷이 일치하는지 단언.
     - **중요(확인함)**: 내장 mock은 `isMockSupportedMIMEType`(`internal/mock/upstage.go:166`)이 `application/pdf`·`image/png`만 허용하므로 선언 `image/jpeg` 업로드는 mock에서 거부되어 마스킹까지 가지 못한다. 그래서 HTTP 회귀는 `startAppServerWithConfig`(기본값 = 내장 mock)가 아니라 **`startAppServerWithUpstream(t, 직접 만든 handler, customize)`**(`internal/httpapi/integration_test.go:1514`)로 돌리고, 그 handler가 `fields` + `boundingBoxes`(mock의 `upstage.go:117-122` 모양)를 돌려주게 할 것. mock 자체는 손대지 말 것(별 과제).
- 건드릴 파일:
  - `internal/masking/engine.go:499-503` (`MaskImageFile`의 인코더 선택 switch) — 우선순위를 뒤집어 **응답이 광고할 `mimeType`을 먼저** 보게 한다. 제안 형태: `lower := strings.ToLower(mimeType)` 뒤 `case strings.Contains(lower,"png"): png.Encode` / `case strings.Contains(lower,"jpeg"), strings.Contains(lower,"jpg"): jpeg.Encode` / `case format == "png": png.Encode` / `default: jpeg.Encode`. 왜 선언 타입이 이기는지(= 같은 값이 `Output.MIMEType`과 `Content-Disposition` 파일명 확장자로도 나간다) 주석 한 줄.
  - `internal/masking/engine_test.go` — 위 네 조합 테이블 테스트. 입력 PNG/JPEG는 기존 헬퍼를 쓰고(없으면 `image/png`·`image/jpeg`의 `Encode`로 수 픽셀 이미지를 만들어) 반환 바이트를 `image.DecodeConfig`가 아니라 `image.Decode`의 두 번째 반환값(format 문자열)으로 확인.
  - `internal/httpapi/integration_test.go` — `POST /v1/mask` 회귀 1개(선언 `image/jpeg` + 실제 PNG 바이트). 기존 `startAppServerWithConfig`/`buildMultipartBody`/`createBlankPNG` 헬퍼와 multipart 응답 파싱 헬퍼를 재사용할 것(파일 상단·`TestMaskAcceptsUploadUsingTheFullConfiguredFileSize` 주변 참고).
  - 프로덕션 파일은 **1개**(`internal/masking/engine.go`)로 끝낸다. README에 한 문장(결과 파일의 포맷은 업로드가 선언한 타입을 따른다) 추가는 선택.
- 검증 명령:
  - `go test -count=1 ./...`
  - `go test -count=1 ./internal/masking ./internal/httpapi`
  - `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal`(무출력이어야 함), `git diff --check`
  - `go test -race -count=1 ./internal/masking ./internal/httpapi`
  - 고치기 전에 새 테스트가 먼저 **실패**하는 것을 보고, 고친 뒤 프로덕션 수정만 임시로 되돌려 같은 실패가 재현되는지 확인할 것.
- 위험과 피할 것:
  - `document.NewAttachment`/`detectAttachmentMIMEType`(attachment.go:42)과 `validateAttachment`(service.go:627)는 손대지 말 것. 선언 MIME을 신뢰하는 정책 자체를 바꾸면 범위가 터지고, 이미지 경로의 `mimeType`은 `ValidateMIMEType`을 이미 통과해 `image/png`·`image/jpeg` 둘 중 하나임이 보장된다(지원 목록은 config.go:153).
  - `Output.MIMEType`·`MaskedFilename`을 실제 포맷에 맞춰 바꾸는 반대 방향 수정은 고르지 말 것 — `service.go`+`internal/document`까지 번지고 다운로드 파일명 계약이 바뀐다.
  - `MaskPDFFile`·`placeRegion`·좌표 정규화 로직은 건드리지 말 것.
  - `internal/document/image.go`는 bmp/tiff 디코더도 등록해 두었다. 선언 MIME이 png/jpeg가 아닌 값으로 이 함수에 도달하는 경로는 **미확인**이므로, `format` 기반 기존 분기를 지우지 말고 fallback으로 남겨 둘 것(수용 기준 2).
  - 보호 경로(auth/migrations/workflows)는 이 과제와 무관하다. `documentResponse`의 `no-store`/`nosniff` 헤더는 그대로 둘 것.
- 차선 후보: `/v1/config/public`이 노출하는 제한값의 양수 불변식 테스트 + `uploadBodyHeadroomBytes` 덧셈 경계 테스트 (가치 2 / 위험 1 / 작업량 S, 프로덕션 변경 0 — `internal/httpapi/server.go:29,116,315`의 불변식을 `config.Load` 경유로 고정)
