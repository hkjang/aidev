- 과제: IMPORTDATA가 업로드처럼 잘못된 UTF-8 입력을 거절한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 같은 CSV 바이트를 업로드하면 `CSV must be UTF-8 encoded`로 거절하지만 IMPORTDATA는 표로 받아들여 JSON 직렬화 시 잘못된 바이트를 U+FFFD로 바꾼다. 인코딩을 밝힌 파일을 `ToUTF8`로 변환한 직후 유효성을 검사하면 손상된 표를 성공 결과로 보여 주지 않으면서 정상 UTF-8·BOM UTF-16 지원을 유지한다.
- 수용 기준: 1) 잘못된 UTF-8(예: `id,v\n1,` + 0xff + `\n`)과 BOM 있는 UTF-32LE/BE CSV는 IMPORTDATA에서 `Err.Code == "#VALUE!"`, Rows/Columns 0, Values 없음으로 끝나며 인코딩 문제임을 알리는 고정 메시지를 준다(본문 원문을 오류에 넣지 않는다). 2) 정상 UTF-8, UTF-8 BOM, BOM UTF-16LE/BE의 한국어·이모지·숫자·불리언 CSV는 기존 타입과 값으로 읽히며 숫자/공백/아포스트로피 계약을 유지한다. 3) 기존 `serve`·`testFetcher`·`withInsecureTLS`로 실제 HTTPS 응답을 받아 `Fetcher.Resolve`를 통과시키고 같은 바이트를 `importexport.Parse`에도 전달해 잘못된 입력은 양쪽 모두 실패, 정상 입력은 좌표별 타입·값·JSON이 같음을 증명한다. 새 실패 사례가 현재 코드에서 실패하고 검사 추가 뒤 통과해야 한다.
- 건드릴 파일: `internal/external/fetcher.go:parseCSV` — `delimited.ToUTF8` 변환 직후, csv.Reader 생성 전에 `utf8.ValidString` 검사와 #VALUE! 반환 추가(utf8는 이미 import됨); `internal/external/fetcher_test.go` — `TestImportDataReadsBooleansLikeUpload`의 실제 HTTPS/업로드 비교 패턴을 사용한 테이블 회귀 테스트 추가. `internal/importexport/service.go:parseDelimited`와 `internal/delimited/encoding.go:ToUTF8/fromUTF16`은 읽기 전용 계약 참고.
- 검증 명령: `go test ./internal/delimited ./internal/external ./internal/importexport`; `go test ./...`; `go vet ./...`; `go build ./...`; `gofmt -l internal/external/fetcher.go internal/external/fetcher_test.go`; `./scripts/check-release-docs.sh`; `./scripts/check-commit-identities.sh HEAD`. 모두 저장소 루트에서 실행. 이번 정찰에서 전체 Go 테스트 통과(대부분 캐시); 웹·DB·브라우저는 미실행.
- 위험과 피할 것: 검사 위치는 반드시 ToUTF8 뒤다(앞이면 정상 UTF-16도 거절). ToUTF8의 깨진 UTF-16 복구·홀수 바이트 처리, BOM 없는 인코딩 추측, NUL 차단, UTF-32 지원 추가는 범위 밖이다. 유효한 U+FFFD 문자는 거절하지 않는다. WEBSERVICE, fetch/SSRF/허용 목록/TLS/캐시 수명, 숫자·Boolean·trim·LazyQuotes·구분자 규칙, auth·migrations·.github/workflows는 바꾸지 않는다. 새 #VALUE!도 현재 one의 캐시 규칙을 따르게 두며 오류 캐시 개선을 끼워 넣지 않는다. b52a394가 되돌린 XLSX 15자리 제한을 다시 적용하지 않는다. 운영 TLS/루프백 정책을 테스트 때문에 완화하지 않는다.
- 차선 후보: 서버·격자 평문 숫자 15자리 경계를 공용 픽스처로 고정 — 재현 전제가 다를 때만 선택. `testdata/file-numbers.json`(신규), `internal/delimited/number_test.go`, `web/src/lib/clipboardNumber.test.ts`에서 공백·통화·지수를 제외한 공통 평문 숫자 경계를 검증하고 파서는 통합하지 않는다. `go test ./internal/delimited`, `cd web && npm test -- src/lib/clipboardNumber.test.ts`(web 의존성 설치 필요; 정찰에서 웹 실행 미확인).

근거와 설계 선택
- 기준 HEAD: 742871c. `go test -overlay=<회차 경로>/evidence-overlay.json ./internal/external -run TestScoutEncodingEvidence -count=1 -v`로 저장소 수정 없이 실행 재현했다. overlay는 기존 테스트 사본 끝에 관찰 테스트만 붙인다. invalid-utf8는 원격 오류 nil, 2×2, JSON `["id","v",1,"\ufffd"]`; UTF-32LE/BE도 원격 오류 nil·1×1이며 업로드는 세 사례 모두 인코딩 오류였다. 증거 파일은 회차 디렉터리 `encoding-evidence_test.go`, `evidence-overlay.json`이며 구현 커밋에 넣지 않는다.
- 최소안(선택): parseCSV 경계에서 기존 업로드와 동일한 유효성 검사. 수정 표면 1함수이고 정상 인코딩을 그대로 지원한다.
- 확장안: ToUTF8가 오류를 반환하도록 모든 호출자를 변경. 깨진 UTF-16의 기존 복구 계약과 웹까지 재설계해야 하므로 이번에는 제외.
- 무변경/테스트만: 손상 결과를 계속 성공으로 보이게 하므로 재현된 사용자 영향 해결에 부족하다. 차선의 공용 숫자 픽스처는 유익하나 현재 데이터 손상을 직접 막는 본 과제보다 후순위다.
- 핵심 가정: 업로드와 마찬가지로 변환 후 잘못된 UTF-8을 실패로 취급하는 것이 IMPORTDATA의 원하는 계약이다. 지금 오류 없이 받는다는 사실은 재현했고, #VALUE! 선택은 parseCSV의 기존 파싱 오류 코드에 맞춘 설계 판단이다. 실제 브라우저 표시·DB 저장까지의 영향은 미확인이다.

구현 순서·검토 지점 (미착수)
1. 기존 코드에서 회귀 사례의 실패를 확인: fetcher_test.go에 실패/정상 인코딩 입력을 구성하고 `go test ./internal/external -run TestImportDataRejectsInvalidEncoding -count=1` 실행. 의도한 거절 단언만 실패하는지 구현자가 확인한 뒤 진행한다. 사람 승인 없음; 이 단계의 실패는 회귀 증거다.
2. parseCSV에 검사 추가 후 같은 명령 및 `go test ./internal/delimited ./internal/external ./internal/importexport` 통과 확인. 정상 U+FFFD와 BOM UTF-16 양쪽 endian도 통과하는지 확인한다. 사람 승인 없음; 전제가 다르면 brief에 차이를 기록하고 차선 판단.
3. 위 전체 검증 명령 실행·diff 확인 후 결과 기록. 테스트/검사 통과를 확인한 것만 완료로 표시한다. 사람 승인 없음; 후속 비평 단계에 변경을 넘긴다.

작업량 근거 (정찰자의 판단, 통계적 보장은 아님)
- 상향식 기본 25~35분: 회귀/정상 입력 구성 12~17분, 검사·메시지 3~5분, 전체 검증·diff·기록 10~13분. 기존 HTTPS 테스트 helper와 설치된 Go 환경을 재사용하며 DB·웹·문서/PDF 변경은 포함하지 않는다.
- 알려진 불확실성 예비 5~10분: UTF-16 테스트 바이트 구성과 좌표/JSON 비교 조정. 총 30~45분, 한 세션 내 가능성은 높으나 실측 속도 자료가 없어 확률 수치 미확인.
- 유사 과제 교차 확인: 9/21 불리언 일치 작업도 서버 파싱 1곳과 HTTPS/업로드 비교로 완료했다. 유사한 범위 S라는 판단은 가능하나 과거 소요 분 자료가 없어 정량 비교 미확인이다.
- 관리 예비는 이 과제에 배정하지 않는다. 캐시/디코더 API 변경이 필요해지면 범위 증가로 기록하고 다시 추정한다(기본 추정에 숨겨 넣지 않음).

적용 스킬
- Skill 도구가 이 세션에 없어 형제 headcount 저장소의 `plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `plugins/technology/skills/implementation-planning/SKILL.md`, `plugins/technology/skills/solution-exploration/SKILL.md` 원문을 직접 읽어 적용했다. pmo references/sources.md도 읽었으며 위 시간 추정은 외부 표준에서 가져온 수치가 아닌 코드·기존 회차에 근거한 정찰자의 판단이다.
