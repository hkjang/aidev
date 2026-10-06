## 추론 엔드포인트가 쓸 수 없는 좌표를 돌려줬는데 요청 잘못이라며 `400`·재시도 불가로 답하던 문제 수정

`mapError`는 `*upstage.CallError`도 `*unrecognizedPayloadError`도 아닌 오류를 전부 `processing_failed`로 묶었고, `handleMask`는 바로 그 코드만 기본값 `502`에서 `400`으로 내립니다. 그래서 업로드가 받아들여지고 검증도 통과하고 엔드포인트도 `200`을 돌려줬는데 **보고된 좌표만 문서에 올릴 수 없었던** 요청이 `400 Bad Request`에 `error.retryable`이 `false`인 채로 돌아왔습니다. 요청에는 잘못이 없으니 같은 파일을 다시 보내면 다른 응답을 받아 성공할 수 있는데, 호출자에게 남은 그 단 하나의 수단이 막혀 있었던 셈입니다. 이제 실패의 원인이 **응답인지 업로드인지**를 나누어 보고합니다.

- `internal/masking`에 `RegionPlacementError`를 추가했습니다. `placeRegion`이 좌표를 올릴 수 없다고 거절하는 경우 중 응답이 원인인 세 가지 — 좌표가 유한한 수가 아닐 때, 응답에 페이지 크기가 없는데 좌표가 페이지를 넘을 때, 박스가 페이지에 걸치지 않을 때 — 가 이 타입을 돌려줍니다. `Detail`이 메시지 전체를 담아 각 지점의 문구는 그대로입니다.
- `internal/service`에 `maskingFailedError`를 추가해, `ProcessSync`의 fail-closed 검사 두 곳(PII 필드는 있는데 쓸 수 있는 bounding box가 없을 때, 덮을 영역이 있었는데도 결과가 원본과 한 바이트도 다르지 않을 때)이 이 타입을 올립니다.
- `mapError`가 두 타입을 기존 `upstream_payload_unrecognized` 분기 옆에서 `masking_failed`·`retryable: true`로 매핑합니다. 동기 요청은 `502`를 돌려주고, 비동기 작업은 상태코드 대신 `failed` 상태에 같은 `error.code`를 담습니다. `server.go`는 손대지 않았으므로 클라이언트가 원인인 실패는 그대로 `processing_failed`와 `400`입니다.
- 업로드한 페이지가 크기를 전혀 알려주지 않는 경우(`/MediaBox`가 비었을 때처럼)는 응답이 아니라 업로드가 원인이므로 `RegionPlacementError`가 아닌 평범한 오류로 남겨 `processing_failed`와 `400`을 유지합니다. 같은 파일을 다시 보내도 같은 결과이므로 `retryable`은 `false`입니다.
- `README.md`의 fail-closed 목록을 경우별 실제 코드에 맞게 고치고, `masking_failed`가 `retryable: true`인 이유와 `processing_failed`가 `false`인 이유를 적었습니다.
- 회귀 테스트는 `internal/masking/engine_test.go`와 `internal/httpapi/integration_test.go`에 추가했습니다.
