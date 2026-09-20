- 과제: IMPORTDATA도 업로드처럼 true/false를 불리언으로 읽는다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 현재 `internal/importexport/service.go:parseScalar`는 대소문자를 무시한 true/false를 bool로 담지만 `internal/external/fetcher.go:parseCSV`는 같은 CSV의 값을 string으로 남긴다. 두 경로가 같은 불리언 타입을 반환하게 하면 원격으로 받은 표와 업로드한 표에서 논리값의 타입이 달라지는 문제를 없앨 수 있다.
- 수용 기준: 1) 공백 없는 true/TRUE/True/false/FALSE/FaLsE는 두 경로 모두 bool true/false가 된다. 2) `truex`, `falsehood`, `t`, `f`, `'true`, ` true`, `false `는 원문 글자로 남으며, 1/0은 계속 숫자이고 `00123`·20자리 번호는 글자다; IMPORTDATA의 기존 숫자 trim(` 12`→12), 업로드의 공백 보존(` 12`→글자), 빈칸·누락 칸 및 아포스트로피 처리는 변하지 않는다. 3) 실제 `httptest.NewTLSServer` → `Fetcher.Resolve` → IMPORTDATA 결과를 같은 바이트의 `importexport.Parse` 결과와 좌표별로 대조하여 불리언의 값뿐 아니라 Go 타입/JSON 타입도 일치함을 증명한다. 기존 `TestImportDataKeepsNumbersThatUploadKeepsAsText`와 XLSX 17자리 실수 회귀 테스트가 계속 통과한다.
- 건드릴 파일: `internal/delimited/boolean.go`(신규 제안): `Boolean(text string) (bool, bool)` — 기존 parseScalar의 strings.ToLower 기반 true/false 판정만 옮긴다(숫자·공백·따옴표 정책 없음); `internal/importexport/service.go:parseScalar` — unguardDelimitedValue 뒤에 새 공통 판정을 호출; `internal/external/fetcher.go:parseCSV` — 원문 record[column]으로 공통 불리언 판정 후, 실패하면 기존 TrimSpace→delimited.Number→원문 반환 순서를 유지; `internal/external/fetcher_test.go` — 아래 실제 두 경로 비교 테스트 추가; `internal/importexport/scalar_test.go` — 필요 시 업로드 공백/불리언 경계 회귀 사례 추가. 신규 helper 자체 테스트만으로 수용하지 않는다.
- 검증 명령: 저장소 루트에서 `go test ./internal/delimited ./internal/external ./internal/importexport`; `go test ./...`; `go vet ./...`; `go build ./...`; `gofmt -l ./cmd ./internal ./pkg`(출력 없어야 함); `./scripts/check-release-docs.sh`; `./scripts/check-commit-identities.sh HEAD`. 정찰에서 앞의 패키지 테스트·전체 테스트·두 스크립트 실행 성공(대부분 Go 캐시 사용). vet/build는 명령과 구성을 확인했지만 이번 정찰에서는 미실행. 웹 변경이 없으므로 npm/브라우저 검증은 이 과제의 필수 범위가 아니다.
- 위험과 피할 것: `parseXLSXValue`는 수정하지 않는다. 실제 main의 b52a394는 XLSX 형식 없는 칸에 15자리 한도를 적용했다가 2.2000000000000002가 글자가 된 문제를 되돌렸다(이전 요약보다 최신인 근거). delimited.Number·formula.DecimalNumber·격자 숫자 파서·숫자 trim·LazyQuotes·인코딩·캐시/SSRF 정책·외부 허용 목록·auth/migrations/workflows/handoff를 건드리지 않는다. strconv.ParseBool은 t/f/1/0도 받아 기존 계약보다 넓으므로 대체하지 않는다. 원격에 unguardDelimitedValue를 도입하지 않는다. 재현용 새 테스트 실행 전에는 실패를 실제 재현했다고 기록하지 않는다.
- 차선 후보: 파일 숫자와 붙여넣기의 15자리 경계를 공용 픽스처로 고정 — `internal/delimited/number_test.go`, `web/src/lib/clipboardNumber.test.ts`, 신규 `testdata/file-number-boundaries.json`에서 공통인 평문 소수/정수 사례만 비교한다. 서로 다른 지수·장식·공백 계약까지 같게 만들지 않는다. 1순위가 이미 해결됐거나 실제 두 경로에서 차이가 없다고 확인될 때만 선택하며 먼저 웹 의존성 및 테스트 실행을 확인한다.

범위와 대안 (solution-exploration)
- 목표는 같은 CSV 불리언의 타입 일치다. 수식 계산법이나 모든 파일 파서의 통합은 요구하지 않는다.
- 선택: 공통 Boolean 판정 하나를 두 호출부에서 사용. 숫자/공백 정책을 그대로 두고 중복 판정도 없앤다. 업로드의 기존 true/false 규칙을 원격에도 적용한다는 가정이 핵심이다.
- 최소 대안: parseCSV에 true/false 분기만 복사. 파일 하나로 끝나지만 두 규칙이 다시 갈라질 여지가 남아 차선 설계다.
- 확장 대안: Scalar 함수로 bool/number/string/trim까지 묶기. 지금 두 경로의 trim·아포스트로피 계약이 달라 회귀 위험이 높으므로 이번에는 제외.
- 유지 대안: 현 상태를 문서화. 배포 영향은 없지만 확인한 타입 불일치가 남아 선택하지 않는다. 라이브러리 도입은 두 문자열 판정에 불필요하다.

구현 순서와 체크포인트 (implementation-planning; 모두 pending)
1. 공통 Boolean 함수와 parseScalar 연결을 한 단계로 추가한다. 업로드 계약이 변하지 않음을 `go test ./internal/delimited ./internal/importexport`로 확인한다. 사람 승인 체크포인트 없음; 실패하면 이 단계에서 수정하고 다음으로 가지 않는다.
2. 기존 `serve`, `testFetcher`, `withInsecureTLS`, `TestImportDataKeepsNumbersThatUploadKeepsAsText`를 본보기로 실제 HTTPS→Resolve와 Parse 비교 테스트를 추가한다(예: `TestImportDataReadsBooleansLikeUpload`). 빈 셀 없는 직사각형 CSV로 대소문자·비슷한 단어·숫자·보호할 번호를 비교하고, 공백·아포스트로피는 별도 명시 기대값으로 확인한다. `go test ./internal/external -run '^TestImportDataReadsBooleansLikeUpload$' -count=1`의 불리언 타입 실패를 먼저 확인한 뒤 같은 단계 안에서 parseCSV 연결을 구현하고 같은 명령이 통과하도록 한다. 단계 종료 시 빌드/테스트는 정상이어야 한다. 사람 승인 없음.
3. 위 검증 명령을 실행하고 diff가 범위를 지켰는지 확인한다. 구현자가 각 단계 상태와 실행 결과를 이 계획에 기록한다. 실제 계약이 다르면 과제서를 먼저 고치고 범위를 넓히지 않는다. 새 DB·설정·배포 작업 없음.

작업량 근거 (estimating-and-contingency)
- 바닥부터 추정: 공통 판정/업로드 연결 5–8분, 실제 두 경로 테스트와 원격 연결 12–17분, 전체 검증/diff 점검 8–10분 = 기본 25–35분. 알려진 불확실성(테스트 좌표·공백 기대값 조정)에 대응하는 예비 5–10분을 따로 두어 총 30–45분, 신뢰도 중간인 판단 범위다(통계적 80% 보장 아님).
- 유사 과제 비교: 09-20 숫자 규칙 공유와 같은 두 호출부·HTTPS/업로드 비교 구조이며 당시 XLSX 확장을 제외하므로 더 좁다. 과거 실제 작업 분 단위 자료가 없어 유사 방식으로 독립적인 수치 추정은 미확인; 수치를 꾸며 평균 내지 않는다.
- 전제: 현재 Go 테스트 환경/의존성 사용 가능, 웹·DB·문서/PDF 변경 없음. 정찰자가 코드와 실행 결과를 근거로 추정했다. 별도 관리 예비는 배정하지 않는다. 예상 밖 제품 계약 문제는 45분 안에 억지로 추가하지 않고 계획을 수정한다.
- 적용 스킬: `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. 전용 Skill 도구가 없어 로컬 원문을 읽어 적용했다. 추정 스킬의 references/sources.md도 확인했으며 외부 지침에 의한 통계적 예비율은 주장하지 않는다.
