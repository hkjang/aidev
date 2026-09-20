- 과제: CSV·XLSX 가져오기에서 동일 필드의 중복 헤더와 실제 선택 열을 경고 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/app/import.go:importHeaders`는 별칭을 canonical 필드로 바꾼 뒤 map에 계속 대입하여 `휴대전화,연락처`처럼 중복된 필드의 왼쪽 값을 조용히 버린다. 기존 오른쪽 열 우선 계약을 유지하면서 사용된 열을 미리보기 경고로 알려 잘못된 전화번호·회사·동의 값의 제출을 줄인다.
- 수용 기준: 1) 실제 선택된 헤더 행에 같은 canonical 필드가 두 번 이상 있으면 필드별 경고 하나를 반환한다. 경고에는 헤더 행 번호(기존 파서의 1부터 시작하는 행 번호 규칙), 중복 열의 1부터 시작하는 열 번호·헤더명, 마지막 열을 사용한다는 안내가 있어야 한다. 예: `이름,휴대전화,연락처,개인정보동의`는 2·3열 중 3열 적용을 알린다. 다른 별칭뿐 아니라 동일 이름 반복과 현재 정규화가 같게 읽는 `PHONE, phone `도 잡고, 미인식 헤더끼리 중복은 경고하지 않는다. 2) 기존 오른쪽 열 우선 결과를 그대로 보존한다. 마지막 값이 공백이어도 앞 열로 대체하지 않는다. 중복 없는 파일의 visitors와 warnings는 그대로이고, 이름·전화·동의·지수 표기 경고와 100명 상한도 유지한다. 헤더 경고는 방문자 수만큼 반복하지 않고 기존 warnings 배열 맨 앞에 결정적인 열 순서로 넣어 현재 화면의 상위 5건 표시에서도 보이게 한다. 응답 필드와 UI 구조는 변경하지 않는다. 3) 실제 CSV bytes 및 excelize 워크북을 `readVisitorImportRows → visitorInputsFromRows`에 통과시키고, 실제 PostgreSQL의 `newTestEnv → uploadImport → previewVisitorImport` HTTP 경로에서도 visitors의 적용 값과 warnings가 일치함을 증명한다. 전화 별칭 중복, 회사/동의 등 선택 필드 중복, 동일 필드 3중 중복, 앞 제목 행, 오른쪽 빈 값, 중복 없는 대조군을 검증한다. fixture에는 두 전화 값을 서로 다르게 넣고 동의는 왼쪽 false/오른쪽 true도 포함한다.
- 건드릴 파일: `internal/app/import.go:importHeaders, importHeaderRow, visitorInputsFromRows` — 선택 헤더의 중복 메타데이터를 구하고 warnings에 결합. 별칭 정규화와 실제 열 선택에 동일한 규칙을 재사용하고 헤더 탐색 중 탈락한 행은 경고하지 않는다. `internal/app/import_test.go:uploadImport, importedVisitors, TestVisitorImportAcceptsExcelExports` — 기존 헬퍼를 활용해 독립 회귀 테스트 추가; HTTP 업로드의 기존 요청 데드라인 유지. `docs/USER_GUIDE.md:협력사 열 명을 한 번에 신청하기`, `docs/API_AND_MCP.md:POST /api/v1/visits/import/preview` — 오른쪽 열 사용과 중복 경고를 짧게 설명. PDF 변경은 정해진 도구가 실제 사용 가능할 때만 하고 미갱신이면 명시.
- 검증 명령: 저장소 루트 `go test ./... -count=1`, `go vet ./...`, `git diff --check`. 새 테스트 이름을 `TestImportDuplicateHeaders...`, `TestVisitorImportDuplicateHeaders...`로 두면 `go test ./internal/app -run 'Test(ImportDuplicateHeaders|VisitorImportDuplicateHeaders)' -count=1 -v`로 집중 실행 가능. 실제 CREATE DATABASE 권한이 있는 PostgreSQL을 준비한 뒤 `VISITFLOW_TEST_DSN='postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable' go test ./... -count=1` 실행; 이 DSN은 CI 설정의 값이며 현재 로컬 서버 존재는 미확인이다. DB 미실행을 PASS로 보고하지 않는다.
- 위험과 피할 것: auth.go·settings.go·server.go·migrations/·.github/workflows/와 미머지 메일/MCP OAuth 브랜치를 건드리지 않는다. `normalizePhone`, 전화 복원, 동의 표기, 첫 시트 선택, 헤더 탐색 10행, importHeaders의 마지막 열 우선 규칙을 변경하지 않는다. 개인정보 셀 값은 경고·감사에 추가하지 않는다(헤더명과 열 위치만). 이 과제는 모호한 열을 사용자에게 알리는 수정이며 차단 또는 자동 병합을 하지 않는다. CSV의 물리적 빈 줄이 제거되는 별도 행 번호 문제는 이번 범위에서 제외한다. 변이를 쓴다면 파일 복사본으로 복원하고 `git checkout --`로 미커밋 편집을 버리지 않는다. 새 기능 대역이나 소스 문자열 검사만으로 성공을 주장하지 않는다.
- 차선 후보: 방문 신청 화면의 회사명 필수 정책 안내 — 중복 경고가 이미 구현된 경우에만 선택. `internal/app/visits.go:referenceData`에 기존 `visit.company_required`의 boolean을 노출하고 `web/src/types.ts:ReferenceData`, `web/src/pages/VisitFormPage.tsx:VisitFormPage`에서 일반/현장 등록 모두 회사명 필수 표시와 공백 제출 방지를 연결한다. 서버 `createVisitRecord`의 기존 검증은 유지하며 실제 설정 변경→API→화면 흐름으로 검증한다(작업량 M).

근거와 계획
- 기준 HEAD 6a3ed81. import.go의 map 대입, visitorInputsFromRows의 cell 조회, previewVisitorImport의 warnings 응답과 VisitFormPage.tsx의 importVisitors/Alert를 직접 확인했다. 현재 warnings 생성은 방문자 행의 이름·전화·동의에 한정되어 헤더 중복을 검사하지 않는다. 중복 파일의 현재 런타임 재현은 미실행이며 위 근거는 코드 읽기 결과다.
- 비교: 회사 필수 안내는 API·UI·브라우저 검증까지 필요해 M, 헤더 행/미인식 열 응답 확대는 새 응답·UI 계약이 필요해 M, 이메일 경고는 허용 주소 계약 미정으로 위험 2다. 중복 경고는 이미 있는 warnings 배선을 쓰므로 S로 선정했다.
- 45분 예산(추정): 재현·테스트 10분, 파서 수정 10분, 실제 HTTP/전체 Go 검증 10분, 문서 5분, 예비 10분. DB 준비가 지연되면 기능 범위를 늘리지 말고 검증 미완료를 명시한다.
- 정찰 검증: `go test ./... -count=1` 통과(app 0.135s). VISITFLOW_TEST_DSN 미설정으로 DB 통합은 SKIP. 프런트 빌드·브라우저·vet는 이번 정찰에서 미실행이다.
- 적용 스킬: [pmo:estimating-and-contingency](/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [technology:implementation-planning](/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [technology:solution-exploration](/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md). Skill 도구는 없으나 확장 검색으로 원문을 찾아 읽고 적용했다.

접근 선택(solution-exploration)
- 원하는 결과: 업로드한 사용자가 중복 열 때문에 어떤 값이 채택됐는지 제출 전에 알 수 있어야 한다. 기존 API 소비자와 일반/현장 신청 화면은 visitors/warnings 응답과 마지막 열 우선 결과를 사용한다.
- 선택 A: 기존 warnings에 중복 필드와 채택 열 안내(S). 응답 확장이 없고 기존 UI가 곧바로 표시한다. 자동 복구를 제공하지 못하지만 호환성이 가장 높다.
- 대안 B: 중복 헤더 파일을 invalid_import로 거절(S). 모호한 데이터를 확실히 막지만 지금 허용되는 파일을 깨뜨린다. 중복 금지가 제품 요구로 명시될 때만 적합하다.
- 대안 C: 열 매핑 미리보기와 사용자 선택 UI(M~L). 앞으로 다양한 공급업체 서식을 지원한다면 적합하지만 새 응답·화면 상태·검증이 필요해 45분 범위를 넘는다.
- 대안 D: 문서에 오른쪽 열 규칙만 설명(S). 런타임 변화는 없지만 문서를 읽지 않는 업로더의 실수는 계속된다. 사용자 영향이 없다고 확인된 경우에만 충분하다.
- A를 권고한다. 가장 중요한 가정은 마지막 열 우선 호환성과 사용자 확인이 자동 차단보다 적합하다는 점이다. 이것은 운영자에게 확인된 요구가 아니라 정찰 판단이다. 구현 중 이 가정에 반하는 저장소 계약을 발견하면 과제서를 수정하고 차선 판단 근거를 남긴다.

실행 순서와 검증 지점(모두 구현자 진행 전)
1. [pending] 재현·환경 확인: import.go의 실제 선택 규칙과 import_test.go 헬퍼를 재확인하고 서로 다른 값을 가진 CSV/XLSX fixture를 설계한다. 증명: `go test ./internal/app -run 'Test(VisitorImportRows|ImportFindsHeaderBelowTitleRows|ImportScientificPhoneFiles)' -count=1 -v`. 현재 기준 통과와 DB 준비 여부를 기록한 뒤 다음 단계로 간다. 사람 승인 지점 없음.
2. [pending] 파서와 회귀 테스트를 한 단계로 완성: import.go에서 동일한 헤더 정규화 규칙으로 중복 경고를 만들고 import_test.go에 실제 파일 및 DB HTTP 테스트를 추가한다. 증명: `go test ./internal/app -run 'Test(ImportDuplicateHeaders|VisitorImportDuplicateHeaders|VisitorImportAcceptsExcelExports)' -count=1 -v`를 VISITFLOW_TEST_DSN이 설정된 상태에서 실행한다. 중복/무중복과 실제 결과 선택까지 통과해야 다음 단계로 간다. 사람 승인 지점 없음.
3. [pending] USER_GUIDE/API_AND_MCP 안내를 맞추고 범위·회귀를 확인한다. 증명: DB가 설정된 `go test ./... -count=1`, `go vet ./...`, `git diff --check`. 문서의 경고 설명을 실제 응답과 대조한다. PDF를 갱신하지 못하면 결과에 명시한다. 사람 승인 지점 없음; 이후 정규 비평 단계에 전달한다.
각 지점은 검증 실패 시 원인을 해결하거나 과제서를 고친 뒤 진행한다. 상태는 구현자가 증명 명령을 실제 실행한 뒤에만 done으로 기록한다.

추정 근거(estimating-and-contingency)
- 방법: 위 작업의 bottom-up 합계 35분을 중심값으로 잡았다(재현/테스트 10 + 구현 10 + 통합 검증 10 + 문서 5). 추정 입력은 이번 코드 탐색과 제공된 이전 회차 기록이며 시간 실측 데이터는 아니다.
- 교차 확인: 이전 지수 전화 경고 작업도 import.go/import_test.go와 가이드·실제 파일/HTTP 경로를 다뤘으므로 유사 추정으로 S 범위는 지지된다. 이전 구현의 총 소요 분은 제공되지 않아 독립적인 수치 추정이나 두 방법의 25% 차이 판정은 불가능하다.
- 범위: 기존 의존성과 테스트용 DB를 바로 사용할 수 있다는 조건에서 30~45분, 주관적 약 70% 신뢰 범위이며 통계적 보장은 아니다. DB 환경 신규 구축·새 UI·CSV 물리 행 번호 수정·PDF 도구 설치는 포함하지 않는다.
- contingency: 알려진 변동인 별칭 중복 경계 5분과 DB 연결/fixture 보정 5분, 합계 최대 10분을 35분 밖에 한 번만 잡는다. 각 작업 시간에는 별도 예비분을 중복 포함하지 않았다.
- management reserve: 미발견 범위에 대한 예산은 이 회차에 배정하지 않는다(0분). 범위 추가가 필요하면 보류 아이디어로 남기고 45분 제한 안에 몰래 끼워 넣지 않는다.
- 첫 실제 파일 재현과 첫 HTTP 통합 완료 시 추정을 다시 확인한다. DB 환경이 준비되지 않으면 45분 신뢰 가정이 깨진 것으로 명시하고 검증을 완료했다고 보고하지 않는다.
- 방법론 참고: 작업 분해·가정·위험 분석·실측에 따른 추정 갱신 원칙은 [US GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 공개 개요를 확인했다. 위 분 단위 수치는 GAO 수치가 아니라 이번 정찰의 조건부 판단이다.
