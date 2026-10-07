## 추론 엔드포인트가 문서에 없는 페이지 번호를 돌려줬는데 요청 잘못이라며 `400`·재시도 불가로 답하던 문제 수정

지난 회차에서 `placeRegion`이 좌표를 올릴 수 없다고 거절하는 경우 중 **응답이 원인인** 자리들을 `*masking.RegionPlacementError`로 옮겨 `502 masking_failed`/`retryable:true`로 보고하게 했지만, `maskPDFFileInternal`의 페이지 범위 검사 한 자리가 범위에서 빠져 있었습니다. 그래서 업로드는 정상적으로 열리는 PDF이고 추론 엔드포인트도 `200`을 돌려줬는데 **보고된 페이지 번호만 문서에 없었던** 요청이 `mapError`의 fallback을 타고 `400 Bad Request`에 `error.retryable`이 `false`인 채로 돌아왔습니다.

이 경우 잘못된 것은 업로드가 아니라 응답입니다. 1페이지 문서에 `"page": 3`을 돌려준 답은 사실상 다른 문서를 설명하고 있는 것이고, 같은 파일을 다시 보내면 다른 응답을 받아 성공할 수 있습니다. 그런데 호출자에게 남은 그 단 하나의 수단을 "요청이 잘못됐고 재시도해도 소용없다"는 보고가 막고 있었던 셈입니다. 이제 이 거절도 나머지 응답 원인 거절들과 같은 부류로 보고합니다.

- `maskPDFFileInternal`이 영역의 페이지 번호가 문서 범위를 벗어났다고 거절할 때 평범한 `fmt.Errorf` 대신 `placementError`를 돌려줍니다. 포맷과 인자가 그대로이므로 `mask region refers to page %d but the document has %d page(s)` 메시지 문구는 한 글자도 바뀌지 않았습니다(`RegionPlacementError.Detail`이 메시지 전체를 담습니다). 응답에 `page`가 빠져 페이지 번호가 `1`보다 작은 경우도 같은 분기이므로 함께 교정됩니다.
- 동기 요청은 `400 processing_failed`/`retryable:false` 대신 `502 masking_failed`/`retryable:true`로, 비동기 작업은 `failed` 상태에 같은 `masking_failed`·`retryable:true`로 남습니다. `mapError`의 `errors.As` 분기와 `handleMask`의 기본 `502`가 이미 있어 `service.go`·`server.go`는 손대지 않았습니다.
- `RegionPlacementError` 독스트링에 "문서에 없는 페이지 번호도 같은 부류의 잘못"이라는 설명을 더하고, README의 오류 코드 목록에서 해당 줄을 `processing_failed`에서 `masking_failed` 쪽으로 옮겼습니다.
- 업로드가 원인인 거절은 그대로입니다. 업로드한 PDF의 페이지가 크기를 전혀 알려주지 않는 경우(`page %d has no usable dimensions`)는 여전히 평범한 오류로 남아 `400 processing_failed`/`retryable:false`로 보고됩니다.

검증은 `MaskPDFFile`을 실제로 호출해 `errors.As(*RegionPlacementError)`·`PageNumber`·메시지 전문 동일성까지 단언하는 표 테스트(문서 범위 초과, 페이지 번호 누락 두 경로)와, `config` 조립부터 실제 `httptest` 리스너까지 프로덕션 배선 전체를 지나는 회귀 테스트로 했습니다. 1페이지 PDF에 `"page":3`의 페이지 안쪽 좌표를 돌려주는 업스트림 핸들러로 `POST /v1/mask`가 `502`·`masking_failed`·`retryable:true`·`metadata.status == "failed"`를, `POST /v1/jobs`가 폴링 후 `failed`에 같은 코드를 내는 것을 확인했습니다.
