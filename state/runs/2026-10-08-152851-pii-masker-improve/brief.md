- 과제: 이미지에 존재하지 않는 페이지의 마스킹 영역을 성공으로 처리하지 않기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/masking/engine.go:494 MaskImageFile`은 `region.PageNumber`를 검사하지 않고 모든 영역을 첫 이미지 위에 그려 업스트림의 `page:3`에도 성공 파일을 내준다. 잘못된 페이지를 기존 `RegionPlacementError`로 거절하면 PDF와 같은 `masking_failed`/502/재시도 가능 실패로 보고하며 문서와 맞지 않는 응답을 정상 처리했다고 알리는 문제를 막는다.
- 수용 기준:
  1) `MaskImageFile` 직접 호출에서 page 2·3·-1은 반환 바이트 nil, `errors.As(err, *RegionPlacementError)` 성공, 오류의 PageNumber는 입력 번호다. 유효한 영역 다음에 잘못된 페이지 영역이 오는 경우도 결과 파일 없이 전체 실패한다. 좌표는 이미지 안쪽을 사용해 페이지 검증 자체를 증명한다.
  2) 직접 호출 page 0·1은 성공하고 디코딩한 PNG에서 지정 영역의 픽셀이 검게 가려진다. HTTP에서 page를 생략한 정상 응답도 성공하고 실제 마스킹 결과를 유지한다. 생략/0을 page 1로 보정하는 `collectMaskRegionsRecursive` 계약과 기존 PNG/JPEG 인코딩 계약을 바꾸지 않는다.
  3) 프로덕션 배선을 지나는 PNG 업로드 + page 3 응답에서 `POST /v1/mask`는 502, JSON metadata.status=failed, error.code=masking_failed, retryable=true이며 결과 바이너리가 없다. `POST /v1/jobs`는 202로 접수된 뒤 failed·동일 오류·download_url 없음, 결과 GET은 404다. 기존 PDF missing-page 회귀는 그대로 통과한다.
  4) 새 페이지 거절 테스트가 구현 전에 실패하고 구현 뒤 통과함을 기록한다. 정상/생략 페이지 테스트는 수정 전에도 통과하는 호환성 보호다. 전체 기존 스위트에 회귀가 없어야 한다.
- 건드릴 파일:
  - `internal/masking/engine.go:494 MaskImageFile` — rects를 모으는 영역 루프에서 PageNumber < 0 또는 > 1을 `placementError`로 거절한다. page 0을 그대로 허용한다. PDF와 같은 문구 `mask region refers to page %d but the document has %d page(s)` 및 페이지 수 1을 재사용할 수 있다. 프로덕션 수정은 이 1파일로 제한한다.
  - `internal/masking/engine_test.go` — `TestMaskImageFileFailsWhenARegionMissesTheImage`, `TestMaskPDFFileFailsForRegionsOnMissingPages` 주변에 잘못된 번호·혼합 영역·0/1 허용 표 테스트. `whitePNG(t,100,100)` 및 (10,10)-(50,30) 좌표 재사용.
  - `internal/httpapi/integration_test.go` — `missingPageUpstream`(:1692), `startAppServerWithUpstream`(:1893), `createBlankPNG`(:2023), `waitForJobStatus`(:1408) 재사용. 기존 동기/비동기 missing-page 테스트(:1704/:1736)를 PDF/PNG 표로 확장하거나 PNG용 테스트를 추가하고 page 생략 성공 회귀를 별도로 둔다. `buildMultipartBodyWithFilenameParam`(:1969)으로 실제 image/png 파트 헤더를 보내면 선언 MIME 헬퍼 결함과 독립적이다.
  - `README.md` — 기존 fail-closed의 “영역이 문서에 없는 페이지를 가리킬 때” 항목에 이미지에도 적용되며 생략은 첫 페이지로 간주됨을 짧게 보충한다.
- 검증 명령 (저장소 루트): `go test -count=1 ./internal/masking ./internal/httpapi`; `go test -count=1 ./...`; `go vet ./...`; `go build ./...`; `go test -race -count=1 ./internal/masking ./internal/httpapi`; `gofmt -l ./cmd ./internal`; `git diff --check`.
- 위험과 피할 것: 파서·PDF 렌더러·service.go·server.go·업스트림 인증/호스트 검사·보존/셧다운·의존성·워크플로는 범위 밖. 현재 JSON의 page 0/음수/생략은 `extractPageNumberFromMap`→`collectMaskRegionsRecursive`에서 1로 보정되므로 HTTP 음수 거절까지 요구하지 않는다(직접 함수 호출의 음수만 검증). 페이지 의미를 zero-based로 재해석하거나 기존 별칭을 정리하지 않는다. 좌표는 정상 페이지 안쪽 픽셀 단위를 사용하고 정규화 좌표 문제를 끼워 넣지 않는다. 내장 mock 변경, multipart 공통 헬퍼 정리, 에러 분류의 전면 통합도 제외한다.
- 차선 후보: `GOARCH=386` 픽셀 폭탄 테스트 픽스처를 아키텍처에 독립적으로 만들기 (가치 2 / 위험 1 / 작업량 S) — `internal/document/image_test.go:22 TestValidateImageDimensionsRejectsPixelBomb`만 범위로 한다. 40000×40000은 PNG 헤더 파서에서 dimension overflow로 먼저 거절되므로 `exceeds the maximum` 단언이 깨진다. 예를 들어 8000×8000(64M픽셀, 상한 50M 초과)처럼 작은 헤더로 바꾸어 같은 해상도 제한을 검증하는 방안을 먼저 실행 확인한다(8000 픽스처 자체는 정찰에서 미실행). 오류 문자열을 무조건 허용하거나 프로덕션 검증을 약화하지 않는다. 검증: `go test -count=1 ./internal/document` 및 `GOARCH=386 go test -count=1 ./internal/document`.

근거와 정찰 검증 (main@19b4426, 2026-10-08):
- 최초 초안을 먼저 저장한 뒤 소스·실행 결과로 이 문서를 덮어썼다. 저장소 코드 수정·커밋 없음.
- `MaskImageFile`은 `pageSizes[1]`을 한 번 읽고 각 region을 page 검사 없이 `placeRegion`에 넘긴다. `service.process`(:497, 이미지 호출 :608 부근)는 오류를 그대로 전달하며 `mapError`(:734)는 RegionPlacementError를 masking_failed/retryable:true로 바꾼다. `server.handleMask`(:137)의 기본 오류 HTTP 상태는 이미 502다.
- 빌드한 실제 cmd 서버 + 실제 config.Load 환경변수 + 로컬 HTTP 업스트림으로 흰색 400×200 PNG를 업로드했다. 업스트림은 이름 홍길동, bbox page 3, 좌표 (40,20)-(200,60)을 반환했다. 동기 200/completed/applied_regions=1, 비동기 202 후 completed/download_url 존재가 재현됐다. 같은 좌표의 page 1 및 page 생략은 동기 200/completed였다. 결과 기록: `assets/http-probe.json`(이 과제서 옆 runs 경로).
- `go test -count=1 ./...` 통과(8개 테스트 패키지), `go vet ./...`, `go build ./...`, `git diff --check` 통과. race와 신규 회귀 테스트는 정찰에서 미실행이며 구현자가 실행한다.
- `GOARCH=386 go test -count=1 ./internal/document`는 현재도 실패: `image_test.go:35: expected a resolution limit error, got failed to inspect image header: png: unsupported feature: dimension overflow`. 이번 선택과 독립인 기존 실패이므로 함께 고치지 않는다.

대안 비교와 결정 (solution-exploration):
- 선택: 이미지 렌더러 경계에서 검증하고 기존 오류 타입을 재사용. 새 상태/설정 없이 1파일로 모든 이미지 호출자에 적용되고 프로덕션 HTTP에서 문제가 재현됐다.
- 대안: service.process에서 attachment 종류와 regions를 검증. API 요청은 해결하지만 MaskImageFile 직접 호출에는 구멍이 남아 렌더러 계약과 분산된다.
- 확대안: 수집 단계의 page 별칭·음수·소수·생략을 전부 엄격하게 정규화. PDF 및 다양한 업스트림 스키마의 호환성이 바뀌어 이번 범위에 맞지 않는다.
- 현상 유지/문서만 보정: 기존 성공을 유지하지만 페이지 3을 첫 페이지에 그렸다는 잘못된 성공 보고가 남는다. 기존 README의 fail-closed 약속과도 맞지 않는다.
- 다른 과제보다 우선한 이유: 386 실패는 검증 이식성 문제이고 multipart 헬퍼는 테스트 정확성 문제다. 선택안은 관찰 가능한 API 동작 결함이며 기존 배선으로 좁게 고칠 수 있다.
- 핵심 전제: 이미지의 실제 페이지는 1이고 page 생략은 정상 입력이다. 파서의 1 보정과 실행한 생략 성공에서 확인했다. 직접 호출 page 0 허용은 호환성을 위한 보수적 선택이며 새로운 업스트림 계약을 추측해 강제하지 않는다.

실행 순서와 체크포인트 (implementation-planning, 모든 단계 pending):
1. 지정 테스트 파일에 수용 기준의 재현/호환성 테스트를 추가한다. 증명: `go test -count=1 ./internal/masking ./internal/httpapi`. 체크포인트: 현재 page 3 성공이 기대 실패를 일으키는지 기록한다(컴파일 실패나 잘못된 픽스처를 재현으로 세지 않는다). 동기 테스트는 JSON decode보다 HTTP 상태를 먼저 검사해 수정 전 multipart 200 실패를 분명하게 보여준다. 사람 승인 대기 없음.
2. engine.go의 이미지 루프에만 번호 검사와 짧은 호환성 주석을 넣고 README를 보충한다. 증명: 같은 두 패키지 명령으로 새 테스트와 기존 PDF·MIME 테스트 통과. 체크포인트: 0/1·page 생략에서 실제 마스킹 출력, 실패에서 결과 미제공 확인. 사람 승인 대기 없음.
3. 전체 테스트·vet·build·대상 race·format/diff 검증 명령을 실행한다. 체크포인트: 결과와 수정 파일 수를 기록하고, 전제가 깨졌으면 과제서를 갱신한 뒤 계속한다. 새 범위를 덧붙이지 않는다. 사람 승인 대기 없음.

산정과 예비 시간 (estimating-and-contingency):
- 방법: 함수·기존 HTTP 헬퍼 확인에 기반한 bottom-up. 재현/단위 회귀 8–10분, guard·문서 4–6분, HTTP 성공/실패 회귀 10–12분, 전체 검증 4–5분으로 기본 26–33분. 포함 범위는 위 4파일; 새 테스트 기반·의존성·배포·아키텍처 작업 제외.
- 알려진 불확실성 contingency 5–10분: multipart 200을 JSON 실패로 잘못 읽는 테스트 수정, 비동기 결과 404 검증 및 생략 응답 픽스처 구성. 기본 항목에 중복 가산하지 않았다. 총 예상 31–43분, 신뢰 중간(실제 담당자 속도 표본이 없는 계획 범위이며 보장 아님).
- 유사 산정 교차 점검: 직전 PDF 페이지 회차의 프로덕션 1파일·단위/HTTP 2테스트 파일·기존 오류 분류 재사용과 같은 규모이므로 S 판단을 뒷받침한다. 이전 회차 실측 소요시간은 제공되지 않아 시간 범위에 대한 독립적인 수치 검증은 미확인이다.
- management reserve는 이번 자율 회차에 배정하지 않음. 파서 변경 등 새로운 범위가 필요하면 45분 한도를 늘리는 대신 과제서를 수정하고 차선으로 전환한다. 재산정 시점은 단계 1의 재현 후와 첫 HTTP 회귀 통과 후다.

적용 스킬: Skill 도구가 제공되지 않아 다음 로컬 원문을 읽어 적용했다.
- `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md` 및 `references/sources.md`
- `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`
- `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`
산정 근거는 현재 코드와 제공된 이전 회차 기록이며 외부 원가·통계 모델을 인용하지 않았다.
