## 보고된 PII 중 일부에만 bounding box가 온 응답을 `200 completed`로 통과시키던 문제 수정

fail-closed 검사는 **응답 전체가 영역을 하나도 만들지 못했는지**만 물었습니다(`service.process`, `len(summary) > 0 && len(regions) == 0`). 그런데 `CollectMaskRegions`는 좌표가 없는 엔티티를 조용히 건너뛰는 한편 `BuildFieldEntries`는 좌표와 무관하게 필드를 담습니다. 그래서 보고된 두 필드 중 **한쪽에만 `boundingBoxes`가 붙은 응답**은 영역이 1개 나오므로 검사를 그대로 통과했습니다.

결과는 이 서비스가 가장 하지 말아야 할 보고였습니다.

- 가려지지 않은 전화번호가 그대로 읽히는 파일이 `200 completed`로, 실제 PNG 바이트와 함께 내려갔습니다.
- 그러면서 `pii_summary`는 그 필드에 `mobile_phone_number` 규칙을 적용했다고 보고했습니다(`applied_regions:1`). 호출자는 마스킹이 끝났다고 믿게 됩니다.

이제 **좌표 없이 보고된 필드가 하나라도 있으면** 그 응답을 거절합니다.

- `internal/masking/engine.go`의 `FieldEntry`에 `HasBoundingBox`(`json:"-"`)를 추가했습니다. `CollectMaskRegions`가 침묵으로 건너뛰므로, 필드가 보고되긴 했는데 덮을 수 없었다는 사실이 남는 자리는 여기뿐입니다.
- 새 헬퍼 `FieldsMissingGeometry`가 그런 필드의 **키만** 수집 순서대로 중복 없이 돌려줍니다. `service.process`는 기존 검사 **바로 아래**에서 이것을 보고 `masking_failed`(`retryable:true`)로 거절합니다. detail에 키만 넣는 이유는 그 문구가 응답과 `job.json`에 그대로 실리기 때문입니다 — 값을 넣으면 거절로 지키려던 PII를 그 자리에서 흘립니다.
- 거절 범위는 `SupportedRuleNames()`의 알려진 규칙 12개로 좁혔습니다. 수집기는 응답 본문 전체를 재귀로 훑고 `extractEntityKey`가 type·category·class를, `extractEntityValue`가 text·content·label을 받으므로 `{"category":"paragraph","content":"본문"}` 같은 **엔티티가 아닌 요소**도 fallback 규칙 아래 엔트리가 됩니다. 그런 요소에는 애초에 좌표가 없으니, 좌표 없는 엔트리를 전부 미마스킹으로 보면 정상 응답을 `502`로 거절하게 됩니다.
- 같은 key+value가 두 번 나오는 응답도 오탐하지 않습니다. 중복 제거(`seen`)를 인덱스 맵으로 바꿔, 뒤에 나온 사본에 좌표가 있으면 첫 엔트리의 `HasBoundingBox`를 OR합니다.

검증은 프로덕션 배선(config 조립 → `app.New` → 실제 리스너 → upstage 클라이언트 → 서비스)을 그대로 지나는 `POST /v1/mask`·`POST /v1/jobs` 회귀로 했습니다. 혼합 응답은 `502 masking_failed`·`retryable:true`·**Content-Type이 `application/json`**(= 파일 바이트가 나가지 않음)이고 detail에 키는 있으나 값 `010-1234-5678`은 없으며, 잡 경로는 `failed`·빈 `download_url`입니다. 두 필드 모두 좌표가 있는 대조군과 비엔티티가 섞인 대조군은 `200 completed`를 그대로 유지합니다. 내장 mock은 5개 필드 전부에 좌표를 붙여 이 결함을 재현할 수 없으므로 mock은 손대지 않고 별도 핸들러를 세웠습니다.

기존 `:597` 검사와 그 문구, `mapError`·`server.go`·`summarizeFields`의 JSON 계약, `CollectMaskRegions`·`buildSubRegionsFromBBox`는 전부 그대로입니다. 변경은 프로덕션 파일 2개(`internal/masking/engine.go`, `internal/service/service.go`), 테스트 2개 파일, README 한 줄입니다. `go test -count=1 ./...`, `go test -race ./internal/masking ./internal/httpapi`, `go vet ./...`, `go build ./...`, `gofmt`가 모두 통과합니다.
