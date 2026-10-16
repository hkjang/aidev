- 과제: 정규화 bbox의 부분 마스킹에서 좌표 단위를 보존하기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `buildSubRegionsFromBBox`(internal/masking/engine.go:159)가 부분 마스킹 스팬의 폭이 1 미만이면 무조건 `endX = startX + 1`로 늘리는데, 이 "1"은 픽셀·페이지 단위를 전제한 값이라 0~1 정규화 bbox에서는 페이지 전체 폭만큼 늘어난다. 그 결과 `placeRegion`(engine.go:430)의 `isNormalizedBox` 판정이 깨져 좌표가 페이지 단위로 재해석되고, 요청에 따라 502 `masking_failed`가 나거나 — 더 나쁘게는 — 200 `completed`로 성공 보고하면서 엉뚱한 좌상단 몇 픽셀만 칠하고 **가려야 할 PII는 그대로 보이는 출력**을 돌려준다.
- 수용 기준:
  1) 400×200 PNG + 이름 필드(`홍길동`, 규칙이 가운데 룬만 가림) + 정규화 bbox `(0.1,0.1)-(0.7,0.3)` 을 돌려주는 업스트림으로 `POST /v1/mask` 가 200/`completed` 를 반환하고, 결과 PNG의 검은 픽셀이 목적지 영역(x 40~280, y 20~60) **안에** 있다. 좌상단 (0,0)~(4,4) 은 흰색으로 남는다.
  2) 픽셀 단위 bbox `(40,20)-(280,140)` 대조군의 출력 픽셀이 수정 전과 동일하다(기존 동작 불변).
  3) `buildSubRegionsFromBBox` 단위 테스트가 ① 정규화 입력에서 모든 서브리전 좌표가 단위 정사각형(0~1+slack) 안에 남고 ② 픽셀 입력에서는 기존 `>=1` 폭 보정이 그대로 적용됨을 증명한다.
  4) 회귀 테스트는 `CollectMaskRegions` → 렌더러(또는 HTTP) 경로를 지나야 한다. `placeRegion` 직접 호출 테스트는 수집 단계 결함을 못 잡으므로 근거로 삼지 말 것.
- 건드릴 파일:
  - `internal/masking/engine.go:159 buildSubRegionsFromBBox` — 폭 보정을 원본 폴리곤 단위에 맞춘다. 권장: 원본 바운즈가 `isNormalizedBox(minX,minY,maxX,maxY)`(engine.go:489) 이면 `endX = startX + 1` 보정을 건너뛴다(스팬이 비어 있지 않으면 폭은 이미 룬 1개분 > 0 이고, 하이라인 보정은 `placeRegion` 이 스케일 **후** target 단위로 이미 수행한다 — engine.go:471~479). 비정규화 입력은 기존 동작 그대로 둔다. 프로덕션 파일은 이 1개로 끝낸다.
  - `internal/masking/engine_test.go` — 수용 기준 3의 표 테스트.
  - `internal/httpapi/integration_test.go` — 수용 기준 1·2의 HTTP 회귀. `startAppServerWithUpstream`(:1893, config 직접 조립이지만 `app.New` + 실제 httptest 리스너를 통과) + 직접 만든 업스트림 핸들러(내장 mock 은 임의 bbox 를 돌려주지 않는다). 업로드 본문은 `buildMultipartBody`(:1942, contentType 무시 → octet-stream) 로 충분하다.
  - `README.md` — 좌표 단위 문단에 한 줄(선택).
- 검증 명령:
  - `go test -count=1 ./internal/masking ./internal/httpapi`
  - `go test -count=1 ./...`  (테스트 있는 9개 패키지, 수 초)
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력) / `git diff --check`
  - `go test -race -count=1 ./internal/masking ./internal/httpapi`
  - 수정 **전** 에 새 테스트 2건의 실패를 먼저 눈으로 확인하고, 수정 후 프로덕션 파일만 `git checkout` 으로 되돌려 같은 실패가 재현되는지 확인할 것.
- 위험과 피할 것:
  - `placeRegion`·`isNormalizedBox`·`minPlacedSize`(0.5) 의 판정 로직 자체를 바꾸지 말 것. 2026-10-06/10-08 회차가 이 함수의 에러 분류를 확정했고, 여기를 넓히면 그 계약이 깨진다. 이번 변경은 **수집 단계의 폭 보정 한 곳**만이다.
  - 에러 메시지 문구를 바꾸지 말 것(기존 테스트가 전문 일치로 단언한다).
  - `len(bboxes) == 1` 일 때만 서브리전이 생긴다(engine.go:137). 여러 박스 케이스는 이번 범위 밖.
  - 빈 스팬(span[0] == span[1])이 들어오면 보정을 건너뛸 때 폭 0 이 되어 `placeRegion` 이 "does not land on the page"(502)로 거절한다. 실제로 그런 스팬이 나오는지는 **미확인** — 방어가 필요하면 비정규화 경로처럼 1 을 더하지 말고 `(maxX-minX)/룬수` 같은 **같은 단위의** 최소 폭을 쓸 것.
  - 보호 경로(auth/업스트림 호스트 검사/jobs 복구/업로드 제한)는 건드리지 않는다.
  - 재선정 금지 목록(출력 MIME, 업스트림 PNG, 원문 masked_value, placeRegion 에러 분류, PDF/이미지 없는 페이지)과 겹치지 않는지 확인했다.
- 차선 후보: 일부 PII 필드에만 bbox 가 없는 응답의 fail-closed 보장 (가치 4 / 위험 3 / 작업량 M) — `service.process`(service.go:585)의 거절 조건이 `summary>0 && regions==0` 뿐이라, bbox 있는 필드와 없는 필드가 섞이면 가려지지 않은 필드가 있는데도 `completed` 로 나갈 가능성이 있다. 실제 HTTP 혼합 응답 재현은 미확인이므로, 먼저 재현부터 하고 재현되지 않으면 손대지 말 것.

## 실행 순서 (각 단계 끝에서 빌드는 green, 중단해도 재개 가능)
1. **빨강 재현 (코드 수정 없음)** — `internal/masking/engine_test.go` 에 `buildSubRegionsFromBBox` 표 테스트(정규화/픽셀 2케이스), `internal/httpapi/integration_test.go` 에 정규화 bbox HTTP 회귀를 추가한다. 증명: `go test -count=1 ./internal/masking ./internal/httpapi` 가 **실패**하고, 실패 줄이 "좌상단만 칠해짐/502" 를 가리키는지 눈으로 확인. 실패 출력을 `assets/` 에 남긴다.
2. **수정** — `engine.go:159` 의 폭 보정을 정규화 입력에서 건너뛴다(왜 단위가 다른지 주석 두 줄). 증명: 같은 명령이 통과. 프로덕션 파일은 이 1개뿐임을 `git status` 로 확인.
3. **파급 확인** — `go test -count=1 ./...`, `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal`(무출력), `git diff --check`(무출력), `go test -race -count=1 ./internal/masking ./internal/httpapi`.
4. **되돌림 재현** — 프로덕션 파일만 `git checkout -- internal/masking/engine.go` 로 되돌려 1단계와 **같은 실패 줄**이 재현되는지 확인하고 복원. 증명 로그를 `assets/` 에 남긴다.
5. (선택) README 한 줄. 범위 밖: `placeRegion` 로직·에러 문구·다중 bbox·PDF 렌더링 경로.

## 산정 (bottom-up, 분해 후)
- 테스트 작성(HTTP 회귀가 대부분) 20분 / 수정 5분 / 파급·되돌림 검증 10분 = **35분**. 비교용 유사 산정: 10-06·10-08 회차(같은 패키지, 같은 테스트 배선, 프로덕션 1~2파일)가 각각 한 세션 안에 끝났다 — 두 방법이 일치한다.
- 범위: 위 1~4단계. 제외: PDF 시각 확인, 다중 bbox, 성능 측정.
- 범위 표현: **30~45분(10번 중 8번)**. 기대지 않은 전제는 하나 — "정규화 입력에서 보정을 건너뛰어도 폭이 0 이 되지 않는다"(= 스팬이 항상 비어 있지 않다). 이것이 틀리면 최소 폭을 같은 단위로 계산해야 하므로 **+10분 contingency**. 그 이상(렌더러 전반 재설계)은 이번 회차 과제가 아니라 재산정 사유다.

## 근거 (이번 정찰에서 실제로 연 것)
- `internal/masking/engine.go` 120~190행(`collectMaskRegionsRecursive` 의 bbox 루프와 `buildSubRegionsFromBBox` 전문), 395~500행(`minPlacedSize`, `placeRegion`, `overflowTolerance`, `isNormalizedBox`) — 결함(`endX = startX + 1`)이 고정 HEAD `19b4426` 에 그대로 있음을 확인.
- `grep buildSubRegionsFromBBox|isNormalizedBox internal/masking/*_test.go` → 매치 없음. 두 함수를 직접 겨냥한 테스트가 하나도 없다.
- 미확인: 이번 세션에서는 HTTP 재현을 **다시 돌리지 않았다**. 재현 자료는 2026-10-09 정찰이 남긴 `assets/probe-results.json`·`probe_normalized.py`(바로 그 회차가 no-change 로 끝나 과제가 남아 있음). PDF 경로의 시각 출력도 미확인.
