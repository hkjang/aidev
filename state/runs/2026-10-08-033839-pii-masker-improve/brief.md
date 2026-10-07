- 과제: 업스트림이 문서에 없는 PDF 페이지 번호를 보고한 요청을 400 `processing_failed` 대신 502 `masking_failed`/`retryable:true`로 보고하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/masking/engine.go:571`의 `mask region refers to page %d but the document has %d page(s)`만 평범한 `fmt.Errorf`로 남아 있어, 업로드는 열리는 정상 PDF이고 추론 엔드포인트가 200과 함께 존재하지 않는 페이지 번호를 돌려준 경우가 `mapError`(service.go:734~)의 fallback을 타고 400 `processing_failed`/`retryable:false`로 나간다 — 요청에는 잘못이 없는데 "요청이 잘못됐고 재시도해도 소용없다"고 보고하며, 유일하게 성공할 수 있는 행동(재시도)을 막는다. 2026-10-06 회차가 `placeRegion`의 거절 네 자리를 `*masking.RegionPlacementError`로 옮기면서 이 한 자리는 범위에서 제외했고(그 회차 보류 목록에 남겨 둠), 사실상 같은 부류인데 혼자 다른 코드로 나간다.

- 수용 기준:
  1) `engine.go:571`의 에러가 `errors.As(err, &*masking.RegionPlacementError)`로 잡히고 `PageNumber`가 업스트림이 보고한 페이지 번호(문서에 없는 그 번호)이며, **메시지 문구는 한 글자도 바뀌지 않는다**. `service.go`·`server.go`는 손대지 않는다 — `mapError`에 이미 `errors.As(err, &placement)` 분기(service.go:759)가 있고 `MaskPDFFile`→service.go:607의 `err`는 감싸지 않고 그대로 올라오므로 자동으로 `masking_failed`/`retryable:true`가 된다. 구현자는 이 전파가 실제로 되는지 ②로 증명할 것(소스 읽기로 끝내지 말 것).
  2) 프로덕션 배선 전체(`config` 조립 → `app.New` → 실제 `httptest` 리스너)를 지나는 동기 회귀: 1페이지 PDF 업로드 + `"page":3`을 돌려주는 직접 만든 업스트림 핸들러 → `POST /v1/mask`가 **502** + `error.code == "masking_failed"` + `error.retryable == true` + `metadata.status == "failed"`.
  3) 같은 업스트림으로 `POST /v1/jobs` → 폴링 후 `failed` 상태에 같은 `error.code`·`retryable`(동기/비동기가 같은 코드로 보고하는지 — 10/06 회차가 두 경로를 쌍으로 고정한 것과 같은 모양).
  4) 파급 차단: `TestMaskPDFFileBlamesTheDocumentForAPageWithoutDimensions`(engine_test.go:378)가 **무수정으로 계속 통과**해야 한다 — `page %d has no usable dimensions`(placeRegion, engine.go:432)는 업로드 탓이므로 평범한 에러로 남아 `processing_failed`/400을 유지한다. 이 과제는 그 분류를 건드리지 않는다.
  5) 고치기 전에 ②③의 실패를 눈으로 확인하고, 고친 뒤 **프로덕션 변경만** 임시로 되돌려 같은 실패가 재현되는 것까지 확인할 것(이 저장소에서 다섯 회차 연속 채택된 증명 방식).

- 건드릴 파일 (프로덕션 1개 + README + 테스트 2개):
  - `internal/masking/engine.go:571` (`maskPDFFileInternal`) — `fmt.Errorf("mask region refers to page %d but the document has %d page(s)", …)` → 같은 포맷·같은 인자로 `placementError(pageNumber, …)`. `placementError`는 같은 파일 engine.go:418에 이미 있다. 더불어 `RegionPlacementError`의 독스트링(engine.go:399~408)이 "a page that reports no size at all"만 평범한 에러라고 적고 있으므로, "문서에 없는 페이지"도 업스트림 답의 문제라는 한 문장을 그 주석에 더할 것.
  - `README.md:83` — `- 영역이 문서에 없는 페이지를 가리킬 때 (`processing_failed`)` 를 `masking_failed` 쪽(README.md:81 바로 아래)으로 옮길 것. README.md:84(`/MediaBox`가 빈 페이지 → `processing_failed`)는 그대로 둔다. README.md:86의 "`processing_failed`는 같은 파일을 다시 보내도 같은 결과" 설명은 그대로 맞는다.
  - `internal/masking/engine_test.go:364` (`TestMaskPDFFileFailsForRegionsOnMissingPages`) — 지금은 `err == nil`만 본다. `errors.As`로 `*RegionPlacementError`를 잡고 `PageNumber == 3`과 메시지 동일성을 단언하도록 넓힐 것. 페이지 번호 `0`(업스트림이 `page`를 생략한 경우 — `pageNumber < 1` 분기)도 케이스로 더하면 네 자리 중 두 경로가 다 덮인다.
  - `internal/httpapi/integration_test.go` — `offPageBoxUpstream()`(:1678) 바로 아래에 같은 모양의 핸들러를 하나 더 만들고(`"boundingBoxes":[{"page":3,"vertices":[…]}]`, 좌표는 페이지 안쪽 값으로 — 페이지 범위 거절이 먼저 걸리는지를 보는 테스트이므로 좌표까지 이상하게 만들면 어느 분기가 걸렸는지 모호해진다), `TestMaskReportsUnusableUpstreamCoordinatesAsRetryableBadGateway`(:1690)와 `TestAsyncJobReportsUnusableUpstreamCoordinatesAsRetryableMaskingFailure`(:1722)를 그대로 본떠 두 테스트를 더할 것. 업로드는 `buildMultipartBody(t, "sample.pdf", "application/pdf", createBlankPDF(400, 400), nil)` — `createBlankPDF`(:1995)는 `/Count 1`인 1페이지 PDF이고 이 호출 형태는 :788·:871·:944에서 이미 통과한다. 내장 mock은 쓸 수 없다(원하는 페이지 번호를 돌려주지 않음) — 반드시 직접 만든 핸들러를 `startAppServerWithUpstream`(:1812)에 물릴 것.

- 검증 명령 (전부 이 저장소에서 실제로 돌며, 이번 정찰에서 `go test -count=1 ./...`가 8패키지 ok로 깨끗함을 확인했다):
  - `go test -count=1 ./internal/masking ./internal/httpapi`
  - `go test -count=1 ./...`
  - `go vet ./...` / `go build ./...`
  - `gofmt -l ./cmd ./internal` (무출력이어야 함)
  - `git diff --check`
  - `go test -race -count=1 ./internal/masking ./internal/httpapi`

- 위험과 피할 것:
  - **메시지 문구를 바꾸지 말 것.** `RegionPlacementError.Detail`이 메시지 전체를 그대로 담는 설계(engine.go:408 주석)이고, 10/06 회차도 네 자리의 문구를 한 글자도 바꾸지 않고 옮겼다. 문구를 바꾸면 README와 `Detail`을 읽는 쪽이 어긋난다.
  - **`placeRegion`의 `page %d has no usable dimensions`(engine.go:432)를 같이 옮기지 말 것.** 그것은 업로드(빈 `/MediaBox`) 탓이라 의도적으로 평범한 에러이고, engine_test.go:378이 그걸 못 박고 있다. 같이 건드리면 그 테스트가 깨지고 10/06 회차의 결론을 되돌리게 된다.
  - `internal/service/service.go`와 `internal/httpapi/server.go`는 손대지 말 것 — 배선이 이미 있다. (10/06 회차도 `server.go` 무수정으로 끝냈다: 기본값이 이미 502이고 `processing_failed`만 400으로 강등된다.)
  - `maskPDFFileInternal`의 다른 두 에러(`pdfcpu watermark create error on page %d`, `MaskPDFFile`의 `recover()`가 만드는 `pdfcpu panic: %v`)는 **이번 범위 밖**이다. 라이브러리 실패이지 좌표 문제가 아니므로 분류 판단이 따로 필요하다 — 아이디어 파일에 남겨 두고 건드리지 말 것.
  - `buildMultipartBody`(:1686)는 `contentType` 인자를 쓰지 않는다(내용 스니핑으로 결정). PDF는 `%PDF` 헤더로 정상 스니핑되어 기존 PDF 테스트들이 통과하므로 이 과제에는 영향이 없다 — 그 헬퍼를 고치려 들지 말 것(별도 아이디어).
  - 보호 경로(auth·migrations·workflows) 해당 없음. DB·마이그레이션 없음, CI 설정 파일 없음.

- 작업량 산정과 여유 (한 세션 45분 기준):
  - 분해: ① engine.go:571 한 줄 교체 + 주석 한 문장 (5분) ② README.md:83 한 줄 이동 (2분) ③ engine_test.go:364 단언 확장 + PageNumber 0 케이스 (8분) ④ integration_test.go 업스트림 핸들러 1개 + 동기/비동기 테스트 2개 (15분) ⑤ 수정 전 실패 확인 → 수정 → 프로덕션만 되돌려 재현 → 복원 (8분) ⑥ 검증 명령 6개 (5분). 합계 43분.
  - 방법 두 가지가 일치: **유사 추정** — 2026-10-06 회차가 같은 축에서 프로덕션 2파일·에러 네 자리 + 신규 테스트 3개를 한 세션에 release-ready로 끝냈고, 이번은 그보다 작다(프로덕션 한 줄). **파라메트릭** — 이 저장소 최근 6회차의 실제 단가는 "프로덕션 파일 1~2개 + 신규 테스트 2~5개 = 1세션"이고 이번 단위 수는 6으로 그 범위 안이다. 두 방법의 차이가 25% 안쪽이다.
  - 범위: 35~50분(10회 중 8회), 최악 70분. 산정 근거의 전제: `MaskPDFFile`의 에러가 `service.go:607`에서 감싸지 않고 올라온다는 것 — 코드로는 확인했으나 실행으로는 미확인. **이 전제가 틀리면 `service.go`에 감싸기 한 자리가 추가되어 +15분**이고, 거기가 이 산정이 깨질 유일한 자리다.
  - 여유: 과제 내 contingency는 위 +15분 한 건으로 한정한다(이미 각 단계에 패딩을 넣지 않았으므로 이중 계상 아님). 그 밖의 발견(pdfcpu 실패 분류, `MaskImageFile`의 PageNumber 무시)은 이 과제의 여유로 쓰지 말고 아이디어 파일에 남겨 다음 회차로 넘길 것 — 회차 예산은 과제 소유가 아니다.
  - 명시적 제외: `service.go`·`server.go` 수정, pdfcpu 실패 두 자리, `MaskImageFile` 경로, `buildMultipartBody` 헬퍼, `GOARCH=386` 조사.

- 차선 후보: `MaskImageFile`(engine.go:499)이 `region.PageNumber`를 아예 보지 않아, 이미지 업로드에 업스트림이 `page: 3`을 보고해도 거절 없이 1페이지 위에 상자를 그리고 `completed`로 보고한다 (PDF 경로는 같은 입력을 거절한다 — 같은 비정상에 두 포맷이 다르게 답한다). 다만 `page`를 생략한 업스트림(PageNumber 0)이 이미지에서는 지금 동작하므로 조이면 기존 동작이 바뀐다(위험 3). 1순위가 성립하지 않을 때만 고르고, 고를 경우 "PageNumber가 0 또는 1이 아닐 때만 거절"처럼 범위를 좁힐 것.
