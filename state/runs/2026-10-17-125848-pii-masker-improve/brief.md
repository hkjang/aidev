- 과제: 일부 PII 필드에만 bbox가 없는 업스트림 응답을 fail-closed 로 거절하기 (가치 4 / 위험 2 / 작업량 M)

- 왜: `service.process`(service.go:597)의 fail-closed 조건이 `len(summary) > 0 && len(regions) == 0` 뿐이다. `collectMaskRegionsRecursive`(engine.go:118~147)는 엔티티에 박스가 없으면 **아무 region 도 append 하지 않고 조용히 넘어가는데**, `collectFieldEntriesRecursive`(engine.go:63~89)는 박스와 무관하게 FieldEntry 를 만든다. 그래서 필드 둘 중 하나만 `boundingBoxes` 를 가진 응답은 `len(regions) == 1` 이라 guard 를 통과하고, 가려지지 않은 PII 가 그대로 보이는 문서를 **200 `completed`** 로 내려보내면서 `pii_summary` 에는 그 필드에 규칙을 적용한 것처럼 보고한다. 고치면 "성공 응답이면 보고된 PII 는 전부 그려 가려져 있다"는 README fail-closed 계약이 혼합 응답에서도 성립한다.

- 수용 기준:
  1) 필드 2개(`개인정보.이름` = boundingBoxes 있음, `개인정보.휴대폰번호` = boundingBoxes 키 자체가 없음)를 돌려주는 업스트림에 대해 `POST /v1/mask` 가 **502 + `masking_failed` + `retryable:true`** 를 반환하고 파일 바이트를 전혀 돌려주지 않는다(JSON only). 같은 업스트림으로 `POST /v1/jobs` 는 폴링 후 `failed` + 같은 코드/retryable, `download_url` 없음.
  2) 두 필드 모두 boundingBoxes 를 가진 응답은 **200 `completed`** 를 유지한다(기존 동작 불변 — 내장 mock 기반 기존 테스트 전부 무수정 통과가 이 기준의 일부다).
  3) 인식되지 않는 키의 엔티티(규칙이 `generic_full_mask` 로 떨어지는 것, 예: `{"category":"paragraph","content":"본문"}`)가 bbox 없이 섞여 있어도 **200 을 유지**한다. 즉 새 guard 는 `SupportedRuleNames()`(policy.go:88)에 있는 12개 규칙이 적용된 필드에만 걸린다.
  4) 단위 테스트가 `BuildFieldEntries` 결과에서 박스 유무가 필드별로 올바르게 구분됨을 증명한다. 특히 **같은 key+value 가 두 번 나오고 두 번째에만 박스가 있는 payload** 에서 그 필드는 "박스 있음" 으로 판정되어야 한다(`seen` 중복 제거가 첫 등장만 남기므로 OR 하지 않으면 오탐으로 502 가 난다).
  5) 거절 메시지에 **필드 값(PII 원문)을 담지 않는다** — 필드 key 만 담는다. 에러 detail 은 응답 JSON 과 `job.json` 에 그대로 적힌다.

- 건드릴 파일 (프로덕션 2 + 테스트 2 + 문서 1):
  - `internal/masking/engine.go:38 FieldEntry` — `HasBoundingBox bool \`json:"-"\`` 추가. JSON 은 `core.PIISummaryItem`(service.go:675 `summarizeFields`)이 따로 만들므로 API 계약은 바뀌지 않는다. `json:"-"` 로 두어 혹시 모를 직렬화 경로도 막는다.
  - `internal/masking/engine.go:55 collectFieldEntriesRecursive` — `bboxes := parseBoundingBoxes(extractGeometryCandidate(typed))` 를 한 번만 계산해(지금은 :82 else 분기에서만 계산) `entry.HasBoundingBox = len(bboxes) > 0` 로 넣고, `seen map[string]struct{}` 를 `seen map[string]int`(entries 인덱스)로 바꿔 **중복 signature 가 다시 나오면 기존 엔트리에 박스 유무를 OR** 한다(수용 기준 4). 기존 page 추론 분기(:80~86)의 동작은 그대로 둔다.
  - `internal/masking/engine.go` (새 함수, `BuildFieldEntries` 바로 아래 권장) — `func FieldsMissingGeometry(entries []FieldEntry) []string`: 규칙명이 `SupportedRuleNames()` 에 있고 `!HasBoundingBox` 인 엔트리의 `Key` 를 중복 없이 순서대로 반환. `generic_full_mask`·`empty` 는 제외(수용 기준 3). 왜 좁히는지 주석 한 단락 — payload 전체를 재귀로 훑는 수집기라 `{"category":…,"content":…}` 같은 비(非)엔티티 맵도 FieldEntry 가 되므로, 전부 거절하면 실제 스키마에서 정상 요청이 전멸할 수 있다.
  - `internal/service/service.go:597~601` 바로 **아래**, 렌더링 블록(:603) **앞** — `if missing := masking.FieldsMissingGeometry(fieldEntries); len(missing) > 0 { err = &maskingFailedError{detail: fmt.Sprintf("PII fields %v were detected but the upstream response reported no bounding box for them", missing)}; metadata.Error = mapError(err); return metadata, nil, err }`. 기존 :597 guard 와 그 문구는 **손대지 말 것**(모두 없는 경우는 계속 기존 메시지로 나간다). `mapError`(:734)의 `maskingFailedError` 분기와 `server.go` 의 기본 502 가 이미 있으므로 **`server.go`·`mapError` 는 무수정**이다.
  - `internal/masking/engine_test.go` — `BuildFieldEntries` 표 테스트: (a) 박스 있는 필드 + 박스 없는 필드 → `HasBoundingBox` true/false, `FieldsMissingGeometry` 가 휴대폰 key 하나만 반환 (b) 같은 key+value 중복, 두 번째에만 박스 → missing 비어 있음 (c) 인식 안 되는 key(`"category":"paragraph"`) 박스 없음 → missing 비어 있음.
  - `internal/httpapi/integration_test.go` — `singleBoxUpstream`(:1677)을 본떠 새 핸들러(예: `partiallyBoxedUpstream`)를 쓴다. 내장 mock(`internal/mock/upstage.go:117`)은 5개 필드 전부에 박스를 붙이므로 혼합 응답을 만들 수 없다 — **mock 은 수정하지 말고** 직접 만든 핸들러를 쓸 것. 배선은 `startAppServerWithUpstream`(:2021) + `buildMultipartBody`(:2070), 업로드는 기존 400x200 PNG 픽스처. 동기/비동기/대조군 2건을 표로.
  - `README.md` — fail-closed 문단에 "일부 필드에만 좌표가 없으면 거절한다" 한 문장.

- 검증 명령 (모두 2026-10-17 고정 HEAD 038dd3a 에서 green 확인됨):
  - `go test -count=1 ./internal/masking ./internal/service ./internal/httpapi`
  - `go test -count=1 ./...` (테스트 있는 8패키지, 각 0.01~0.63초)
  - `go vet ./...` · `go build ./...` · `gofmt -l ./cmd ./internal`(무출력) · `git diff --check`
  - `go test -race -count=1 ./internal/masking ./internal/httpapi`
  - 고치기 전 빨강을 눈으로 확인하고, 고친 뒤 **프로덕션 2파일만** `git checkout` 으로 되돌려 같은 실패가 재현되는지 확인할 것(이 저장소의 관례).

- 위험과 피할 것:
  - **넓은 버전을 고르지 말 것.** "박스 없는 FieldEntry 가 하나라도 있으면 거절" 은 구현이 더 짧지만, `extractEntityKey` 가 `type`·`category`·`class` 를, `extractEntityValue` 가 `text`·`content`·`label` 을 받으므로 실제 스키마의 비엔티티 요소가 전부 "마스킹 안 된 필드" 로 잡혀 정상 요청을 502 로 만들 수 있다. 실제 Upstage 응답 본문은 이 세션에서 **미확인**이다. 반드시 `SupportedRuleNames()` 로 좁힐 것.
  - `collectMaskRegionsRecursive`·`buildSubRegionsFromBBox`·`placeRegion`·`isNormalizedBox` 은 건드리지 말 것 — 10/16 회차(8f2d71f)가 방금 좌표 단위 계약을 고정했다.
  - `summarizeFields`/`core.PIISummaryItem` 의 JSON 필드를 늘리지 말 것(응답 계약).
  - 에러 detail 에 필드 **값**을 넣지 말 것(감사·응답으로 재노출된다). key 만.
  - `internal/service`·`internal/app` 의 기존 테스트가 쓰는 payload 에 "인식되는 키인데 박스 없는 필드" 가 있는지는 **미확인**이다. `go test ./...` 가 빨강이 되면 그 픽스처를 먼저 읽고, 프로덕션 조건을 넓히는 쪽이 아니라 픽스처가 비현실적인지 판단할 것.
  - 보호 경로(auth/migrations/workflows) 해당 없음. `internal/upstage`·`internal/jobs` 는 이번 범위 밖.

- 산정 근거(basis): 분해 — engine.go 플래그+중복 OR ~25줄 / 새 헬퍼 ~15줄 / service.go guard ~8줄 / engine_test 3케이스 ~60줄 / integration 4케이스 + 핸들러 ~120줄 / README 1줄. 바텀업 35~55분(10회 중 8회 이 범위 안). 유일한 불확실 전제는 위의 "기존 service/app 픽스처" 한 줄이며 여기에만 +15분 contingency 를 둔다. 범위를 늘려 소비하지 말 것.

- 차선 후보: 업스트림이 보고한 소수 페이지 번호(`"page": 1.9`)가 `intValue` 에서 1로 절삭되어 정상 페이지로 받아들여지는지 실제 HTTP 로 재현하고, 재현되면 `placementError` 로 거절 (가치 3 / 위험 3 / 작업량 M). 1순위가 기존 픽스처와 충돌해 성립하지 않을 때 이것을 고를 것.
