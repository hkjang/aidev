# PR 처리기 노트 2026-10-06-113910-pii-masker-shepherd — pii-masker PR #34
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-06-094752-pii-masker-improve)
# 회차 노트 2026-10-06-094752-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:48] base pinned — main@9e32f63
- [러너 09:48] autonomy release — 

## 정찰 노트
- 고른 이유: 저장소는 이미 "200인데 해석 불가"를 upstream_payload_unrecognized→502로 분류해 뒀는데, 같은 부류인 좌표 fail-closed 세 자리(engine.go:438/410/406, service.go:585/605)만 mapError의 processing_failed fallback에 떨어져 400+retryable:false로 나간다. 설계 의도가 코드 주석에 이미 적혀 있어 "판단 문제"가 아니라 빠진 구멍이라고 논증할 수 있고, 프로덕션 파일 2개(server.go 무수정 — 기본값이 이미 502)로 끝난다. 보류 1·3·5위(config/public 경계, config 정규화 테이블)는 이번에 계산해 보니 오버플로가 구조적으로 불가능해 순수 회귀 고정뿐이라 제쳤고, 386 실패는 또 재현 못 해(미확인) 제쳤다.
- 추측으로 적은 것: 새 타입 이름(RegionPlacementError / maskingFailedError)과 코드명 "masking_failed"는 제안일 뿐 — 구현자가 더 맞는 이름을 쓰면 된다. retryable:true로 두는 판단도 제안이다(같은 업스트림이 다음에 쓸만한 박스를 줄 수 있으므로).
- 확인한 것: placeRegion 호출부 두 곳(engine.go:478,:550)이 에러를 감싸지 않고 그대로 반환 → errors.As 전파에 추가 변경 불필요. 저장소 전체에서 processing_failed 단언은 integration_test.go:489 한 건(픽셀 폭탄=클라이언트 원인, 400 유지가 맞음)뿐.
- 구현자가 조심할 것: 범위 확대 금지. MaskImageFile:469 "failed to decode image"와 pdfcpu 계열(:538/:564/:575), 인코딩 실패(:517)는 손상된 업로드일 수 있어 400에 남겨야 한다. 내장 mock은 자기 좌표를 주므로 못 쓰고 startAppServerWithUpstream(integration_test.go:1637) + 직접 만든 핸들러로 페이지 밖 박스를 돌려줄 것.
- 프로필: 2026-10-05판(0일 전)이 지금 코드와 맞아 다시 쓰지 않았다. 한 줄만 낡았다 — "현재 기준: main@5608238 / v1.0.34" 는 실제로 main@9e32f63 / v1.0.35 이고 재선정 금지 목록에 10/05 "masked_value 원문 반환" 을 더해 읽어야 한다.
- [러너 09:55] scout done — 업스트림 좌표 때문에 마스킹이 실패한 요청을 `/v1/mask`가 400 `invalid`/`retryable:false`로 보고하는 것 고치기 (

## 구현 노트
- `mapError`(service.go)가 `processing_failed` 로 뭉개던 좌표 fail-closed 여섯 자리를 `masking_failed`/`retryable:true` 로 분류해, 업로드는 정상이고 업스트림 200 이후에 막힌 요청이 400 대신 502 로 나가게 했다. `placeRegion` 네 자리는 `*masking.RegionPlacementError`(메시지 문구 불변), `service.go:585/:605` 는 `maskingFailedError`. `server.go` 무수정 — `handleMask` 기본값이 이미 502 다.
- 확신 없는 곳: **`retryable:true` 판단**. 과제서의 수용 기준이 요구한 값이고 "요청에는 잘못이 없다"는 뜻으로는 맞지만, 같은 입력에 결정적으로 같은 답을 주는 엔드포인트라면 재시도가 도움이 안 된다(저장소의 선례인 `upstream_payload_unrecognized` 는 `retryable:false`). 추론이 비결정적이라는 전제에 기대고 있다.
- 과제서가 "placeRegion 의 `fmt.Errorf` 세 개"라 했으나 실제로는 **네 개**였고(페이지 크기 없음 / NaN·Inf / 페이지 크기 미보고 좌표 초과 / 페이지에 안 걸침) 네 개 전부 바꿨다.
- 일부러 안 한 것: `maskPDFFileInternal` 의 "refers to page %d but the document has %d page(s)" 는 같은 부류지만 과제서 범위 밖이라 `processing_failed`·400 으로 남겼다(README 에 그대로 명시, ideas.json 신규 항목). `failed to decode image`·pdfcpu 계열·인코딩 실패도 손상 업로드일 수 있어 400 유지.
- 검증 못 한 것: `GOARCH=386`, Docker 빌드. 파급 차단 테스트는 빈 업로드·미지원 MIME 두 건만 HTTP 로 단언했고 `MaxFileSizeBytes` 초과·페이지 수 초과는 안 했다(기존 `integration_test.go:489` 픽셀 폭탄 테스트가 무수정 통과하는 것으로 갈음).
- 다음 역할이 조심할 것: 신규 통합 테스트 세 개는 `startAppServerWithUpstream` 에 직접 만든 핸들러(`offPageBoxUpstream`)를 물린다 — 내장 mock 은 자기 좌표를 돌려주므로 쓸 수 없다. 외부 네트워크·DB 불필요. 비동기 테스트는 `waitForJobStatus` 폴링(최대 1초)에 의존한다.
- [러너 10:01] brief accepted — 채택 — 과제서의 근거(`mapError`의 fallback과 `handleMask`의 400 강등, 호출부 두 곳이 placeRegion 에러를 감싸지 않음, `processing_fa
- [러너 10:02] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 판정 **reject**. 수리가 가장 먼저 볼 파일은 `internal/masking/engine.go:428` — `target.Width<=0` 분기는 `target`이 두 호출부 모두 **문서**에서만 오고(이미지는 image.go:22가 0을 이미 거름) PDF 페이지 크기만이 유일한 경로여서 구조적으로 업스트림 원인이 될 수 없는데 `*RegionPlacementError`로 묶여 502/`retryable:true`가 됐다. `/MediaBox [0 0 0 0]` PDF를 손으로 만들어 재현 확인(정상 PDF는 nil, zero는 `isPlacement=true`). 이 한 줄을 `fmt.Errorf`로 되돌리고 engine_test.go:259 표의 해당 케이스를 빼면 끝 — 나머지 세 자리와 service.go 두 자리는 분류가 맞다.
- 같은 뿌리로 engine.go:399-405 주석과 README:85가 코드와 반대를 말한다("parse가 먼저 실패했을 것", "재시도하면 성공할 수 있다") — 수리 시 문구도 같이.
- 검증은 실재한다. 프로덕션 2파일만 `git checkout main --`로 되돌려 신규 통합 테스트를 직접 돌렸고 원장 `- 실패 재현:`과 글자 단위로 같은 두 줄이 재현됐다. 복원 후 build/vet/gofmt/`go test ./...`(8패키지 ok) 통과. 가드 테스트 2건이 수정 전에도 통과하는 점은 원장이 정직하게 적어 둠.
- 보안·법무 차단 없음: 인증·인가·식별자·비밀값·의존성 무변경, 새 `Error.Detail`은 좌표·페이지 크기뿐(PII·업스트림 본문 없음), 수신자는 업로더 본인. UI는 이미 detail을 렌더한다.
- 못 본 것: Docker 빌드·`GOARCH=386`·Windows Rename(프로필 미확인 그대로). 다음 회차용: `service.go:618`(출력이 원본과 동일)의 `retryable:true`가 남은 최약점이고, README:80-83 열거에 "page N has no usable dimensions"가 아예 없다. 릴리즈 노트에 `masking_failed` 신규 코드 + 동기 400→502 = 사실상 계약 변경임을 명시.
- [러너 10:06] review rejected — 리뷰 거절: internal/masking/engine.go:428 `target.Width <= 0 || target.Height <= 0` 분기는 업스트림 응답이 아니라 **업로드한 문서**가 원인인데도 `*RegionPlacementError`로
- [러너 10:06] pr created — https://github.com/hkjang/pii-masker/pull/34

## 수리 노트
- 비평 두 건 모두 맞았다. `/MediaBox [0 0 0 0]` PDF로 재현해 `errors.As(err, &*RegionPlacementError)`=true 를 눈으로 확인했고, `target`이 두 호출부(engine.go:497 이미지 bounds, :571 pdfPageDims) 모두 문서에서만 오며 이미지 쪽은 `ValidateImageDimensions`(image.go:50)가 0을 이미 거르는 것도 코드로 확인했다. 틀린 지적은 없었다.
- 고친 방법: 지적대로 `engine.go:431` 한 줄만 `fmt.Errorf`로 되돌리고, 표에서 해당 케이스 제거 + 대상을 실제로 돌리는 회귀 테스트 추가(`TestMaskPDFFileBlamesTheDocumentForAPageWithoutDimensions`, 기존 `blankPDF(0,0)` 헬퍼 재사용). 주석(engine.go:399-406)과 README:84-85도 같이 바로잡고 README 열거에 빠져 있던 항목을 더했다. 커밋 c068b40 — rebase/amend 없음.
- 인과 증명: 그 한 줄을 다시 `placementError`로 바꾸면 신규 테스트가 실패하고, 되돌리면 통과한다. `go test -count=1 ./...` 8패키지 ok, build/vet/gofmt/diff --check 통과.
- 여전히 확신 없는 곳: `masking_failed` 자체의 `retryable:true`(구현 노트가 적은 약점 그대로 — 추론이 비결정적이라는 전제에 기댄다)와 `service.go:618`(출력이 원본과 동일) 분류는 범위 밖이라 손대지 않았다. Docker 빌드·`GOARCH=386`·Windows Rename 미확인.
- 다음 역할에게: 이번 수정으로 README의 `processing_failed` 열거가 6항목이 됐으니 릴리즈 노트에서 "동기 400→502 계약 변경"을 말할 때 **페이지 크기 없음은 400에 남았다**는 점을 함께 적어야 정확하다.

## 심사 노트
- 확인한 것: 거절 사유 2건 모두 수리됐다. engine.go:431 을 `placementError` 로 다시 바꾸면 `TestMaskPDFFileBlamesTheDocumentForAPageWithoutDimensions` 가 실패하고 되돌리면 통과해 인과가 증명된다. 프로덕션 2파일을 origin/main 으로 되돌리면 신규 통합 테스트 2건이 실제로 실패하며(400/processing_failed 수신), 이 테스트는 `app.New`+httptest 로 띄운 실제 배선에 실제 HTTP 로 단언한다. 주석·README:83-84 도 코드와 일치한다. build/vet/gofmt/diff --check/`go test ./...`(8패키지 ok)·`GOARCH=386 go build` 통과.
- 못 본 것: Docker 빌드·Windows Rename. 다만 이 디프는 Dockerfile·릴리즈·CI 경로를 건드리지 않는다(변경 5파일). `GOARCH=386 go test ./internal/httpapi` 의 `TestMaskRejectsImageWithOversizedDeclaredResolution` 실패는 기존 결함 — integration_test.go 디프가 126줄 순수 추가라 그 테스트는 main 과 동일하다.
- 권고 근거(approve/merge, risk low): 인증·인가·데이터·의존성·공개 경로 변경이 없고 새 `Error.Detail` 은 좌표와 고정 영문뿐이라 차단 소견 없음. 남은 약점은 `masking_failed` 의 `retryable:true` 가 추론 비결정성 전제에 기대는 점과 `minPlacedSize` 의 극소 페이지 경계인데, 자동 재시도 소비자가 저장소에 없어(UI 문구만) 피해가 안내 수준이고 세 노트가 이미 약점으로 기록해 뒀으므로 차단하지 않고 notes 로 남겼다.
