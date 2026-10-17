# 회차 노트 2026-10-17-125848-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:58] base pinned — main@038dd3a
- [러너 12:58] autonomy release — 

## 정찰 노트
- 보류 1위(혼합 bbox 응답의 fail-closed)를 골랐다. 두 회차 연속 차선으로 밀렸던 이유가 "실제 재현 미확인"이었는데, 이번에 engine.go:118~147(박스 없는 엔티티는 region을 append하지 않음)과 service.go:597의 조건을 코드로 대조해 결함 경로를 확정했고, 내장 mock payload(5개 필드 전부 박스)가 왜 이 결함을 못 잡는지도 확인했다. 소수 페이지 번호·buildMultipartBody·Store.List 복제는 전부 "미확인 전제가 결함의 유무 자체"여서 제쳤다.
- 과제서에서 추측으로 적은 것: ① 실제 Upstage 응답에 인식 키 + 박스 없는 필드가 실제로 오는지(호출 불가 — 그래서 거절 범위를 `SupportedRuleNames()` 12개 규칙으로 좁히게 했다) ② internal/service·internal/app 기존 테스트 픽스처에 그런 필드가 있는지(읽지 않음 — contingency 를 여기에만 뒀다). HTTP 혼합 응답 재현은 이번에도 하지 않았다(코드 대조까지만).
- 구현자가 조심할 것: 넓은 버전("박스 없으면 전부 거절")은 ideas.json 에 rejected 로 박아 뒀다 — `extractEntityKey` 가 type/category/class 를 받으므로 정상 요청을 502 로 만들 수 있다. 그리고 중복 제거(`seen`)를 OR 로 바꾸지 않으면 같은 key+value 가 두 번 나오는 응답에서 오탐 502 가 난다.
- 고정 HEAD 038dd3a 에서 `go test -count=1 ./...` 8패키지 green 을 직접 확인했다(빨강 기준선 없음 — 새 테스트가 빨강을 만들어야 한다).
- [러너 13:04] scout done — 일부 PII 필드에만 bbox가 없는 업스트림 응답을 fail-closed 로 거절하기 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 바꾼 것: `FieldEntry.HasBoundingBox` 추가 + `seen` 을 `map[string]int`(entries 인덱스)로 바꿔 중복 signature 에 박스 유무 OR, 새 헬퍼 `FieldsMissingGeometry`(`SupportedRuleNames()` 12개 규칙으로 좁힘), `service.go` 기존 :597 guard **아래**에 거절 분기. 혼합 bbox 응답이 200 completed + 실제 PNG 바이트로 나가던 것을 502 masking_failed/retryable:true + JSON only 로 막는다. 프로덕션 2파일 + 테스트 2 + README 1줄.
- 확신 없는 곳: ① 실제 Upstage 응답 본문은 이번에도 **미확인** — "인식 키 + 박스 없음" 이 실무에서 오는지 모른다. 그래서 거절을 12개 규칙으로 좁혔지만, 만약 실제 엔드포인트가 `{"type":"이름","text":"..."}` 류 비엔티티 요소를 보내면 `name` 규칙에 걸려 정상 요청이 502 가 될 수 있다. 비평가가 먼저 볼 곳. ② `metadata.Error.Detail` 에 들어가는 `%v` 포맷의 Go 슬라이스 표기(`[개인정보.휴대폰번호]`)가 사용자에게 보이기에 적절한지 — 과제서가 지정한 문구를 그대로 썼다. ③ Docker/Windows 실환경·GOARCH=386 은 이번에도 돌리지 않았다.
- 일부러 안 한 것: `collectMaskRegionsRecursive`·`buildSubRegionsFromBBox`·`placeRegion`(8f2d71f 가 방금 고정), `mapError`·`server.go`(기본 502 가 이미 맞음), `summarizeFields`/`core.PIISummaryItem` JSON 필드(응답 계약), 내장 mock(혼합 응답은 직접 만든 `partiallyBoxedUpstream` 으로), 기존 :597 guard 의 문구. 차선 후보(소수 페이지 번호)는 1순위가 기존 픽스처와 충돌하지 않아 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 외부 의존 없이 전부 로컬 실행 가능(DB·네트워크 불필요). `partiallyBoxedUpstream` 은 400x200 업로드 전제의 페이지 크기(400x200)를 돌려주므로 다른 크기 픽스처로 재사용하면 안 된다. `TestProcessSyncFailsClosedWhenPIIHasNoBoundingBoxes`(service_test.go:24)는 기존 :597 guard 가 먼저 걸려 통과하는 것이니 새 guard 의 커버리지로 세지 말 것 — 새 경로는 `internal/httpapi` 쪽 두 테스트와 `internal/masking` 표 테스트가 덮는다.
- [러너 13:10] brief accepted — 채택 — 과제서의 근거(engine.go:118 의 조용한 건너뛰기와 service.go:597 의 조건, 내장 mock 이 5개 필드 전부에 박스를 붙여 이
- [러너 13:10] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인: diff 전체(프로덕션 2파일 + 테스트 2 + README 1줄), engine.go 두 수집기의 대칭성(HasBoundingBox 는 collectMaskRegionsRecursive 와 같은 extractGeometryCandidate/parseBoundingBoxes 를 쓰므로 박스 없음 ⟺ region 없음), summarizeFields 가 entries 를 1:1 로 내보내 요약과 거절 대상이 어긋나지 않음, 구 guard(regions==0)가 혼합 응답에서 regions=1 로 통과함 → 새 테스트가 반드시 새 경로를 지남. `go test -count=1 ./...`(8패키지 ok)·`go vet`·`gofmt -l` 직접 재실행. 원장 `- 실패 재현:` 이 증상(200 completed + applied_regions:1 + 실제 PNG 바이트)과 일치.
- 못 본 것: 실제 Upstage 응답 본문(세 회차 연속 미확인), 프로덕션 파일 되돌린 재현(읽기 전용 유지), Docker/Windows/GOARCH=386.
- 남는 우려 1 — engine.go:63~68 주석의 "비엔티티는 generic 폴백" 단언이 부정확하다. policy.go:56 의 규칙 매칭은 부분 문자열이라 `strings.Contains("description","ip")=true`(직접 확인), "recipient"→ip_address, "filename"→name, "hotel"→telephone_number. 이런 키의 비엔티티 요소가 좌표 없이 오면 새 guard 가 정상 응답을 502 로 만든다. 방향이 fail-closed 이고 실제 응답 미확인이라 차단하지 않았으나, 응답 본문을 확보하는 회차가 가장 먼저 볼 자리.
- 남는 우려 2 — engine.go:104~110 의 중복 signature OR 때문에 같은 key+value 가 두 번 보고되고 한쪽만 박스를 가지면 거절하지 않는다. 같은 문자열이 문서에 두 번 인쇄된 경우 한쪽이 안 가려진 채 200 이 나간다. README 새 줄은 "key+value = 한 필드" 전제에서만 성립하므로 릴리즈 노트에 적을 것.
- 남는 우려 3(경미) — service.go:609 의 `%v` 가 `[개인정보.휴대폰번호 ...]` Go 슬라이스 표기로 사용자에게 노출된다. 보안·법무 차단 사유는 없음(새 엔드포인트·인가 변경 없음, detail 에 값 아닌 key 만 들어가며 key 는 이미 pii_summary·job.json 에 저장돼 왔다).
- [러너 13:15] review approved — 리뷰 승인 (risk=low)
- [러너 13:15] pr created — https://github.com/hkjang/pii-masker/pull/38
- [러너 13:16] ci passed — 검사 없음 — 정책으로 허용
- [러너 13:16] merge done — e4bc0af
- [러너 13:20] release published — v1.0.39
- [러너 13:20] gh-release created — GitHub Release v1.0.39
- [러너 13:20] manifest ok — pii-masker-image.tar.gz 
- [러너 13:20] assets uploaded — 1개
- [러너 13:20] assets verified — v1.0.39 자산 1개 (이전 v1.0.38: 1)
