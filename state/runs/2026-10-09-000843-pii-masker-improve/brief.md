- 과제: 정규화 bbox의 부분 마스킹에서 좌표 단위를 보존하여 엉뚱한 위치의 성공 출력을 막기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `buildSubRegionsFromBBox`가 0~1 비율 좌표에도 최소 폭 1을 적용해 `placeRegion`이 이를 픽셀 좌표로 오인하며, 실제 HTTP 실행에서 올바른 bbox가 502로 거절되거나 좌상단 4픽셀만 가린 채 200/completed로 반환됐다. 원본 bbox의 정규화 단위를 보존하면 README의 비율 좌표 지원 약속대로 의도한 문자 위치가 가려지고, 잘못 가린 파일을 성공으로 내보내는 문제를 막을 수 있다.
- 수용 기준:
  1) 흰 400×200 PNG와 이름 `홍길동`/key `개인정보.이름`, page 1, bbox (0.1,0.1)~(0.7,0.3)에서 POST /v1/mask가 200/completed, applied_regions=1, masked_value=`홍*동`을 반환한다. 결과 PNG의 (160,40)은 검정, (80,40)/(240,40)/(0,0)은 흰색이어야 한다. 기대 가림 영역은 약 x=120..200, y=20..60이다.
  2) bbox 높이를 (0.1~0.7)로 늘린 경우에도 의도한 (160,80)은 검정, (80,80)/(240,80)/(0,0)은 흰색이다. 현재 HEAD의 200/completed 및 applied_regions=1만으로는 성공을 입증할 수 없다. 결과 바이트를 PNG 디코딩하여 픽셀을 검사한다.
  3) 원본 정규화 bbox가 분할된 뒤에도 좌표가 원본 범위와 단위를 유지한다. 같은 테스트를 보고된 페이지 크기 유/무로 검사하여 0~1 좌표의 우선순위를 보존한다. 기존 픽셀 bbox (40,20)~(280,140), 보고된 페이지 크기 기반 스케일링, 다중 bbox 전체 가림은 계속 통과해야 한다.
  4) engine 테스트는 `CollectMaskRegions`를 거친 영역을 실제 `MaskImageFile`로 렌더링한다. HTTP 테스트는 `startAppServerWithUpstream`의 app.New→실제 리스너→업스트림 클라이언트→서비스 경로로 최종 PNG 픽셀을 검사한다. 기존 `TestPlaceRegionScalesNormalizedCoordinates`만 늘리는 것으로 끝내지 않는다.
  5) 실패 재현→수정 후 통과를 기록한다. 변경한 프로덕션 부분만 잠시 되돌렸을 때 신규 회귀가 다시 실패하는지 확인하고 복원한다. 공유 수집 함수가 PDF에도 쓰이므로 정규화 bbox의 수집→placeRegion 결과를 PDF 목표 크기에서도 확인하고 기존 PDF 회귀를 유지한다(새 PDF 래스터 도구 도입 불필요).
- 건드릴 파일:
  - `internal/masking/engine.go:159 buildSubRegionsFromBBox` — polygonBounds 직후 원본 bbox의 `isNormalizedBox` 여부를 판정하고, 정규화 bbox에는 분할 전 단위의 최소 폭 1 보정을 적용하지 않는다. 실제 페이지 단위 보정은 기존 `placeRegion`에 맡긴다. 비정규화 bbox의 기존 최소 폭 보정은 유지한다. 이유를 짧은 주석으로 남긴다.
  - `internal/masking/engine_test.go` — `TestPlaceRegionScalesNormalizedCoordinates`(:204) 인근에 수집부터 렌더링까지 검사하는 회귀와 비정규화·다중 bbox 호환성 검사 추가. 기존 `TestMultiBoxFieldsAreCoveredInFull`(:164), 페이지 크기 스케일링(:217), PDF 테스트(:367/:410) 유지.
  - `internal/httpapi/integration_test.go` — `startAppServerWithUpstream`(:1893), `createBlankPNG`(:2023), `parseMultipartMaskResponse`(:1997)를 재사용해 위 두 bbox의 표 테스트 추가. 내장 mock 대신 실제 HTTP handler가 아래 payload를 반환하게 한다. 명시 MIME은 `buildMultipartBodyWithFilenameParam`(:1969)을 사용하면 된다.
  - 합계: 프로덕션 1개, 테스트 2개. README는 기존 지원 약속이 이미 맞으므로 변경 필수 아님.
- 검증 명령: 저장소 루트에서 `go test -count=1 ./internal/masking ./internal/httpapi`, `go test -count=1 ./...`, `go test -race -count=1 ./internal/masking ./internal/httpapi`, `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal`, `git diff --check`.
- 위험과 피할 것: Region 스키마·좌표 파서·정규화 판정 허용오차·placeRegion의 minPlacedSize·오류 분류·출력 MIME 계약을 바꾸지 않는다. 작은 정규화 영역이 실제 페이지에서도 0.5단위 미만인 경우의 거절 정책은 별도 문제다. 전체 입력 좌표를 일괄 스케일링하거나 픽셀 bbox까지 최소 폭 보정을 제거하지 않는다. 인증/호스트 검사, jobs/store, 셧다운, 워크플로에는 변경이 필요 없다. 직전 회차의 이미지 페이지 검사(88e8ace)는 성공 기록이 있지만 현재 고정 HEAD 19b4426에는 없다. 이를 재구현하거나 이번 과제에 포함하지 않는다. `buildMultipartBody`의 contentType 무시, 386 픽스처 오류도 함께 고치지 않는다.
- 차선 후보: GOARCH=386 픽셀 폭탄 테스트 픽스처를 아키텍처에 독립적으로 만들기 (가치 2 / 위험 1 / S) — 1순위가 실제 구현에서 성립하지 않을 때만 선택. `internal/document/image_test.go:TestValidateImageDimensionsRejectsPixelBomb`의 40000×40000을 50MP 초과이면서 32비트 PNG 헤더가 받는 값(예: 8000×8000, 이 대체값의 실행 검증은 아직 미확인)으로 바꾸고 `GOARCH=386 go test -count=1 ./internal/document` 및 기본 아키텍처 테스트로 증명. 프로덕션 오류 변경·아키텍처 skip 금지.

## 확인한 근거와 재현
- 기준: main@19b4426, 작업 트리 변경 없음. 소스의 `buildSubRegionsFromBBox`(:159, 폭 보정 :175)→`placeRegion`(:428)→`MaskImageFile`(:494), 서비스의 수집·렌더링 배선(:551~608)을 직접 읽었다.
- 업스트림 응답: `{"type":"mock-pii","model":"pii","result":{"apiVersion":"1.1","documentType":"pii","fields":[{"key":"개인정보.이름","value":"홍길동","boundingBoxes":[{"page":1,"vertices":[{"x":0.1,"y":0.1},{"x":0.7,"y":0.1},{"x":0.7,"y":0.7},{"x":0.1,"y":0.7}]}]}]}}`. 짧은 bbox는 마지막 두 꼭짓점의 y를 0.3으로 한다.
- 현재 짧은 bbox: 502 `masking_failed`, detail의 영역은 `(0.3, 0.1)-(1.3, 0.3)`. 긴 bbox: 200/completed, applied_regions=1이지만 (160,80)은 흰색, 검정 픽셀은 (0,0)~(1,1) 4개뿐이다. 대조군 픽셀 bbox는 (160,80)이 검정이고 x=120..199/y=20..139에 9600픽셀을 칠한다.
- 원본 bbox의 x폭 .6에서 가운데 글자 부분은 .3~.5이어야 하지만 폭 보정이 .3~1.3으로 만든다. `isNormalizedBox`가 false가 되어 단위를 오인한다. 작은 높이는 minPlacedSize 검사에 걸리고, .6 높이는 통과해 좌상단을 칠한다.
- 실행 증거: 같은 run의 `assets/probe-results.json`, `assets/normalized_tall.png`, `assets/pixels_control.png`, `assets/probe_normalized.py`. 저장소 코드를 바꾸지 않고 실제 cmd 바이너리와 로컬 HTTP upstream을 실행했다. 재현 명령: `go build -o /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-000843-pii-masker-improve/assets/pii-masker-scout ./cmd/pii-masker` 후 `python3 /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-000843-pii-masker-improve/assets/probe_normalized.py`. 이 스크립트는 관찰용이며 CI 회귀는 Go로 작성한다.
- 정찰 기본 검증: `go test -count=1 ./...` 9개 테스트 패키지 통과, `go vet ./...`, `go build ./...`, `git diff --check` 통과. 대상 race는 masking 1.074초/httpapi 1.939초로 통과했고 `gofmt -l ./cmd ./internal`은 무출력. 386 document 테스트는 기존 dimension overflow/해상도 오류 메시지 불일치로 실패했다. 구현 수정 및 PDF 최종 시각 출력은 정찰에서 미검증.

## 대안 판단과 실행 계획
- 범위 밖: 페이지 파서 강화, 필드별 bbox 누락 검출, 디버그 최적화, 신기능, 의존성 변경. 질문이나 사람 승인 대기 없이 아래 자동 검증을 체크포인트로 삼는다.
- 선택안: 원본 bbox가 정규화일 때만 폭 1 보정을 생략한다. 기존 단위 판정 도우미를 재사용하고 프로덕션 1파일로 끝난다. 가장 중요한 전제는 수집 이후 placeRegion이 최종 단위에서 최소 폭을 처리한다는 것인데, 해당 함수 본문에서 확인했다.
- 대안 A: 좌표 단위를 Region에 명시해 렌더러 전반으로 전달한다. 장기 확장에는 유리하지만 계약·파일 범위가 커져 이번 45분 회차에 부적절하다.
- 대안 B: 모든 bbox를 통째로 가린다. 누출을 막을 수 있지만 선택적 문자 마스킹 계약과 보이는 문자까지 훼손한다.
- 대안 C: 현행 유지하고 386 테스트만 수정한다. 확정된 정상 입력 실패와 잘못된 성공 출력을 방치하므로 차선으로 둔다.
- [ ] 1단계(5~7분): 위 payload와 출력 검사를 테스트로 옮겨 red 확인. 증거 명령 `go test -count=1 ./internal/masking ./internal/httpapi`. 빌드는 유지하고 이 단계에서 예상한 신규 실패만 있는지 자동 검토한다.
- [ ] 2단계(4~6분): engine.go 한 지점의 정규화 분기 수정과 짧은 주석. 같은 명령으로 green 확인; 예상과 다르면 이 과제서의 가정을 먼저 정정한다.
- [ ] 3단계(10~12분): 페이지 크기 보고 유/무 및 비정규화 대조군·PDF 공유 배치 회귀 보완, 최종 PNG 픽셀 단언 점검. 같은 대상 테스트로 검증하고 다음 단계로 간다.
- [ ] 4단계(7~10분): 전체 테스트·race·vet·build·형식·diff 검사, 프로덕션 변경만 되돌린 red 재현 후 복원하여 대상 테스트 재통과. 결과를 journal에 기록한다.
- 산정: 위 bottom-up 기본 작업량 26~35분, 알려진 불확실성(부동소수점 경계/HTTP 픽셀 검사 보완)에 contingency 최대 8분을 별도로 두어 34~43분을 계획 범위로 본다. 보수적 주관 추정이며 통계적 신뢰수준은 산출할 데이터가 없다. 별도 management reserve는 0분; 파서 재설계가 필요해지면 범위를 확장하지 않는다.
- 유사 작업 교차확인: 10/03·10/08의 프로덕션 1파일+engine/HTTP 테스트 2파일 구조와 같고 기존 헬퍼를 재사용한다. 기록에 실제 소요 시간이 없어 유사 작업 시간에 따른 정량 교차검증은 미확인이다. red 재현과 첫 green 뒤 남은 시간을 다시 산정한다.
- 적용 스킬: 로컬 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`(뒤 두 경로의 루트는 동일한 headcount/plugins). 전용 Skill 도구가 노출되지 않아 파일로 읽었으며, 외부 표준의 비용 산정 수치를 적용하지 않았다.
