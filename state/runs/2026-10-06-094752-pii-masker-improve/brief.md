# 과제서 — 2026-10-06 (base: main@9e32f63)

- 과제: 업스트림 좌표 때문에 마스킹이 실패한 요청을 `/v1/mask`가 400 `invalid`/`retryable:false`로 보고하는 것 고치기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `service.mapError`(service.go:721)는 `*upstage.CallError`도 `*unrecognizedPayloadError`도 아닌 **모든** 에러를 `processing_failed`로 뭉개고, `handleMask`(server.go:147-157)는 기본 502에서 `processing_failed`만 **400으로 내린다**. 그래서 업로드는 정상이고 추론 엔드포인트가 200을 돌려준 뒤 fail-closed 검사가 걸린 경우 — `placeRegion`의 "does not land on the page"(engine.go:438), "PII fields were detected but the upstream response did not contain usable bounding boxes"(service.go:585), "produced an output identical to the original document"(service.go:605) — 클라이언트는 **400 Bad Request + `retryable:false`** 를 받는다. 요청에는 아무 문제가 없었고 유일하게 성공할 수 있는 행동(재시도)이 막힌다. 이 저장소는 같은 부류(200인데 본문을 필드로 못 읽음)를 이미 `upstream_payload_unrecognized` → 502로 분류해 두었으므로(service.go:100-110), 이 세 자리는 그 설계 의도에서 빠진 구멍이다.

- 수용 기준:
  1) 업스트림이 200 + 필드 + **페이지 밖 좌표**(페이지 크기 미보고)를 돌려주는 프로덕션 배선 HTTP 회귀에서 `POST /v1/mask` 가 **502** 와 `error.code == "masking_failed"`, `error.retryable == true` 를 돌려준다(현재는 400 / `processing_failed` / `false`).
  2) 같은 업스트림으로 `POST /v1/jobs` → 작업이 끝난 뒤 `GET /v1/jobs/{id}` 의 `error.code == "masking_failed"`, `retryable == true`. (비동기 경로는 상태코드가 아니라 레코드로 보고하므로 두 경로가 같은 코드를 쓰는지 확인하는 것이 핵심 — 러너가 반복해 지적한 "같은 값을 읽는 경로가 둘이면 둘 다 확인".)
  3) **파급 차단**: 클라이언트 잘못인 실패는 400 `processing_failed` 그대로 유지된다 — 빈 파일, `MaxFileSizeBytes` 초과, 지원하지 않는 MIME(`validateAttachment` service.go:624), 페이지 수 초과/깨진 PDF(`countPages` :642). 최소 두 건을 HTTP로 단언할 것.
  4) `masking` 단위 테스트: `placeRegion`이 페이지 밖 박스에 대해 돌려주는 에러가 새 타입으로 `errors.As` 된다. 좌표가 NaN/Inf인 경우(engine.go:410), `target.Width<=0`(:406)도 같은 타입이어야 한다 — 셋 다 "업스트림 좌표를 믿을 수 없다"는 같은 사실이다.
  5) 수정 전에 1)·2)의 실패를 눈으로 확인하고, 고친 뒤 **프로덕션 파일만** 되돌려 같은 실패가 재현되는 것까지 확인한다.

- 건드릴 파일 (프로덕션 2개):
  - `internal/masking/engine.go:405 placeRegion` — 세 개의 `fmt.Errorf`를 내보낼 수 있는 타입(예: `type RegionPlacementError struct{ PageNumber int; Detail string }` + `Error()`)으로 바꾼다. **호출부 두 곳(:478 `MaskImageFile`, :550 `maskPDFFileInternal`)은 이미 `return nil, err` 로 감싸지 않고 그대로 돌려주므로** 전파를 위한 추가 변경은 필요 없다(이번 회차에 확인). 메시지 문구는 바꾸지 말 것 — 기존 테스트가 문자열을 볼 수 있다.
  - `internal/service/service.go`
    - `:585`, `:605` 의 두 fail-closed 에러를 새 내부 타입(예: `type maskingFailedError struct{ detail string }`, `unrecognizedPayloadError`(:104) 와 같은 톤의 주석 한 단락)으로 감싼다.
    - `mapError`(:721)에 `*masking.RegionPlacementError` 와 `maskingFailedError` 를 `errors.As` 로 잡아 `Code: "masking_failed"`, `Retryable: true` 를 돌려주는 분기를 `unrecognized` 분기 **옆에** 추가한다. 마지막 `processing_failed` fallback 은 그대로 둔다.
  - **`internal/httpapi/server.go` 는 건드리지 않는다.** `handleMask` 의 기본값이 이미 502 이고 `processing_failed` 만 400 으로 내리므로, 코드명이 바뀌는 순간 502 가 자동으로 나온다. 핸들러를 같이 고치면 3)의 400 경로를 깨뜨릴 위험만 생긴다.
  - README 응답/에러 문단에 한 문장(선택).

- 검증 명령:
  - `go test -count=1 ./internal/masking ./internal/service ./internal/httpapi`
  - `go test -count=1 ./...` (8패키지, 1초 내)
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력) / `git diff --check`
  - `go test -race -count=1 ./internal/masking ./internal/httpapi`

- 위험과 피할 것:
  - **범위를 넓히지 말 것.** `MaskImageFile:469` "failed to decode image", `maskPDFFileInternal:538/564/575`(pdfcpu), `:517` 인코딩 실패는 **손상된 업로드 = 클라이언트 잘못**일 수 있으므로 `masking_failed` 로 바꾸지 않는다. 이것들이 400 에 남는 것은 의도다.
  - **이번 회차에 확인**: 저장소 전체에서 `processing_failed` 를 단언하는 테스트는 `integration_test.go:489`(`TestMaskRejectsImageWithOversizedDeclaredResolution` — 픽셀 폭탄, 400 유지가 맞는 클라이언트 원인) **한 건뿐**이고, `usable bounding boxes`·`does not land` 문구를 단언하는 테스트는 **없다**. 즉 기존 단언을 고칠 필요가 없어야 한다 — 고쳐야 할 것이 생기면 범위를 잘못 넓혔다는 신호다.
  - 검증 함정(프로필 확인): 내장 mock(`internal/mock/upstage.go:167`)은 자기 좌표를 돌려주므로 못 쓴다 — `startAppServerWithUpstream`(integration_test.go:1637)에 **직접 만든 업스트림 핸들러**를 물려 `fields` + 페이지 밖 `boundingBoxes` 를 돌려줄 것(10/03·10/05 회차가 쓴 방법). `buildMultipartBody`(:1686)는 `contentType` 인자를 무시하므로 선언 MIME 이 필요하면 `buildMultipartBodyWithFilenameParam`(:1713) — 이 과제는 PNG 바이트를 올리면 되므로 선언 MIME 은 상관없다.
  - 비동기 회귀는 `queued → running → completed/failed` 를 폴링해야 한다. `t.Setenv` 를 쓰는 테스트는 병렬화 금지.
  - 보호 경로(auth/호스트 검사/드레이닝/마이그레이션)는 건드리지 않는다. `upstage` 패키지는 열지 않아도 된다.

- 차선 후보: **integration 테스트 헬퍼 `buildMultipartBody`(integration_test.go:1686)가 `contentType` 인자를 무시하는 것 고치기** (가치 3 / 위험 2 / 작업량 S, 프로덕션 파일 0개). `CreateFormFile` 을 쓰므로 파트 Content-Type 이 항상 `application/octet-stream` 이 되고, `detectAttachmentMIMEType`(attachment.go:43)이 그 값을 무시하고 내용 스니핑으로 내려가므로 **20여 개 호출부가 선언 MIME 을 실제로는 한 번도 보내지 않는다**. 최근 세 회차의 결함이 모두 "선언 MIME vs 실제 내용" 축이었는데 그것을 지켜야 할 테스트가 구조적으로 그 축을 못 본다. 헬퍼가 인자를 실제로 쓰게 바꾸고(명시적 `CreatePart` + 파트 헤더), 전체 스위트를 돌려 동작이 바뀌는 테스트를 찾아 — 바뀌는 것이 있으면 그것이 숨은 결함이다 — 의도대로 정리할 것.
