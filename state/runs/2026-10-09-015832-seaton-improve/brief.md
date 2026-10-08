- 과제: 좌석 일괄 배정에서 좌석번호가 빠진 XLSX 행을 누락시키지 않고 실패 행으로 알린다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/app/seats.go:bulkAssignments`는 `len(row)<2`인 행을 건너뛰므로 사번만 적은 XLSX 행이 성공·실패 어디에도 집계되지 않는다. 비어 있지 않은 행의 필수값 누락을 실패 목록에 넣으면 관리자가 어떤 행을 고쳐야 하는지 알 수 있고, USER_GUIDE의 “반영되지 않은 행은 행 번호와 사유를 보여준다”는 계약을 지킨다.
- 수용 기준: 1) 머리글 A1=사번/B1=좌석번호, A2=E002, 3행 공백, A4=E003인 실제 XLSX 업로드는 HTTP 200, success=0, failed=2를 반환하고 failures에 row=2/4, employeeNo=E002/E003, seatNo="", error="좌석 번호가 누락되었습니다"를 담는다. UI에 `0건 배정, 2건 확인 필요`, `반영되지 않은 2행`과 두 행의 사유가 보인다. 2) 앞 두 칸이 모두 공백인 행은 기존대로 무시하고, 사번만 없으면 `사번이 누락되었습니다`로 실패 처리한다. 필수값이 빠진 행은 직원 조회·좌석 조회·배정을 하지 않는다. 정상 행·부분 성공·BOM CSV·게시 도면 제한·기존 배정 실패 사유는 유지한다. 3) 실제 XLSX/CSV multipart를 프로덕션 readSpreadsheet→bulkAssignments로 보내는 Go 테스트와 실서버 업로드 E2E로 증명한다. 새 XLSX E2E는 변경 전 HEAD에서 실패, 변경 후 성공해야 하며 기존 bulk-assign.spec.ts도 통과해야 한다. 새 오류 파일 업로드 전후 직원의 좌석 배정이 동일해야 한다.
- 건드릴 파일: `internal/app/seats.go:bulkAssignments`(370~415행) — 길이를 확인하며 앞 두 칸을 빈 문자열 기본값으로 읽고 TrimSpace 적용; 두 칸 모두 비면 continue; fail 클로저를 준비한 뒤 각각의 누락을 기록하고 continue; 기존 DB 조회부터는 보존. `internal/app/seats_test.go` — 실제 excelize.NewFile/WriteToBuffer 및 multipart.Writer로 만든 요청을 httptest로 핸들러까지 통과시켜 실패 JSON 검증. `web/e2e/bulk-assign.spec.ts` — 기존 login·setInputFiles 및 fetchSeats를 써서 실제 업로드 실패 집계와 화면 표시 검증. `web/e2e/fixtures/bulk-missing-seat.xlsx`(신규 테스트 데이터) — 위 수용 기준의 4행짜리 오류 파일. 프로덕션은 1파일, 테스트/fixture 포함 4파일이다.
- 검증 명령: 루트에서 `go test ./internal/app`, `go test ./...`, `go vet ./...`, `go build ./...`, `gofmt -l internal/app/seats.go internal/app/seats_test.go`, `git diff --check`. 프런트는 `cd web && npm ci && npm test && npm run lint && npm run build`. 실서버 준비 후 `cd web && E2E_BASE_URL=http://127.0.0.1:18789 E2E_USERNAME=admin E2E_PASSWORD=ci-e2e-password-123 PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright npx playwright test e2e/bulk-assign.spec.ts`.
- 위험과 피할 것: 직원 가져오기 파서와 CSV 필드 수 규칙을 합치거나 넓히지 않는다. CSV에 `E002` 한 칸만 쓰면 readSpreadsheet의 csv.Reader가 먼저 invalid_csv로 거절하므로 짧은 행 회귀는 실제 XLSX로 증명하고, CSV의 누락 칸은 `E002,`/`,HQ-3F-001`로 시험한다. readSpreadsheet, performAssignment, SQL, 트랜잭션, auth, migrations, workflows, 의존성, 프런트 프로덕션 코드는 변경하지 않는다. “빈 행”은 기존 계약대로 앞 두 칸이 비어 있는 행이다(추가 열 처리 정책 신설 금지). 새 순수 파서/DB 주입 인터페이스는 필요 없다. 테스트에 직접 만든 [][]string만 넘기거나 소스 문자열 검사로 끝내지 않는다. 기존 E2E의 배정 변경은 keepingSeats 복구를 보존하고 공유 DB를 쓰지 않는다.
- 차선 후보: 직원 가져오기 실패 행 사유에서 사번 누락과 이름 누락을 구분 — `internal/app/employees.go:importEmployees`의 `fail("사번/이름 누락")` 한 분기와 실제 가져오기 E2E. 첫 과제가 현재 구현에서 이미 해결되어 성립하지 않을 때만 전환하며 환경 준비 지연을 이유로 둘 다 구현하지 않는다.

근거와 확인 범위

- 기준 HEAD는 `026d7a7`(v1.4.17 문서), 작업 트리는 깨끗하다. `git log -30`, README, ROADMAP_PLAN/ARCHITECTURE/USER_GUIDE, CI/릴리즈 설정, 관련 테스트를 읽었다. 저장소의 CLAUDE.md/AGENTS.md 및 internal·web/src·scripts의 TODO/FIXME는 검색 결과 없었다.
- 정찰에서 실제 실행한 `go test ./...`는 전 패키지 통과했다(internal/app 11.535s). npm 의존성이 없는 체크아웃이라 프런트 및 실서버 E2E는 미실행이다. 위 검증 명령은 go.mod/package.json/Playwright/CI에서 확인한 것이며, go test 외의 성공을 주장하지 않는다.
- 읽은 실제 경로: `readSpreadsheet`는 15 MiB 요청 제한 후 excelize.OpenReader→GetRows(첫 시트)를 사용한다. `bulkAssignments`의 380행 부근은 길이 2 미만이면 DB 접근 전에 행을 버린다. `EmployeesPage.assignFromFile` 및 282~307행은 failures를 이미 렌더하므로 UI 수정이 필요 없다. `Server.Routes`는 POST `/api/v1/seat-assignments/bulk`를 실제 핸들러에 연결한다.
- `assets/xlsx_probe.go`를 저장소 루트에서 go run한 실제 결과: row=1 len=2 머리글, row=2 len=1 [E002], row=3 len=0 [], row=4 len=2 [E003 HQ-3F-001]. excelize v2.9.1의 실제 객체를 직렬화하고 다시 읽은 결과다. 원인인 “후행 빈 칸 생략”은 확인했으며, 정찰에서 HTTP 응답·UI까지 재현하지는 않았다. 구현자의 위 E2E가 마지막 근거를 채워야 한다.

구현 순서와 확인 지점 (현재 모두 미착수, 사람 승인 단계 없음)

1. 실서버 기준선과 실패 증거 확보: 위의 오류 전용 XLSX fixture와 새 Playwright 케이스를 추가하고 `npx playwright test e2e/bulk-assign.spec.ts -g '누락'`를 변경 전 이미지에서 실행한다(테스트 제목에 “누락” 포함). 예상 실패는 failed=0 또는 오류 목록이 안 보이는 것이어야 한다. 서버 기동/로그인 실패를 역검증으로 세지 않는다. 테스트는 빌드 가능한 채로 유지하며 예상된 회귀 실패를 기록한 뒤 다음 단계로 간다.
2. 최소 수정과 Go 증명: bulkAssignments의 입력 두 칸 처리만 고친다. `TestBulkAssignments...` 이름의 핸들러 테스트를 같은 단계에서 추가해 `go test ./internal/app -run TestBulkAssignments -count=1` 통과 후 다음 단계로 간다. 실제 XLSX에서 행 2/4 및 빈 행 3을 시험하고, 실제 CSV의 `E002,` 및 `,HQ-3F-001`도 시험한다. 누락만 있는 요청에서는 DB가 필요 없으므로 DB 없는 Server에 실제 핸들러 요청을 보낼 수 있다. 이 테스트만으로 라우팅을 증명했다고 하지 말고 3단계를 필수로 한다.
3. 수정 이미지로 실제 UI 검증: 새 회귀 E2E와 기존 bulk-assign 두 케이스를 통과시킨다. 신규 오류 전용 파일은 상태를 바꾸지 않아야 하므로 fetchSeats 전후 결과를 비교한다. 정상+실패의 부분 성공은 기존 CSV E2E가 실제 배정까지 증명한다. 새 E2E를 별도로 돌린 뒤 전체 bulk-assign spec과 Go/프런트 기본 검증을 완료하고 결과를 기록한다. 사실이 계획과 다르면 과제서를 수정하고 범위를 다시 평가하며 조용히 다른 리팩터를 붙이지 않는다.

실서버 검증 준비 예시 (정찰에서 미실행, 기존 CI/Dockerfile에 맞춘 명령)

변경 전 HEAD는 지정 출력 디렉터리에 `git archive HEAD`로 새로 풀어 `seaton:bulk-before-1009`로 빌드한다. 기존 동명 이미지 재사용 금지. 수정본은 현재 체크아웃에서 `docker build --build-arg VERSION=e2e -t seaton:bulk-after-1009 .`로 만든다. 아래의 이미지명만 교체해 전후를 시험한다. 포트 18789 사용 가능 여부는 구현 시 확인하고 충돌하면 빈 포트로 바꾼다.

```bash
docker network create seaton-bulk-1009
docker run -d --name seaton-bulk-pg-1009 --network seaton-bulk-1009 -e POSTGRES_USER=seaton -e POSTGRES_PASSWORD=seaton -e POSTGRES_DB=seaton postgres:16-alpine
docker exec seaton-bulk-pg-1009 pg_isready -U seaton
# 위 준비 확인이 성공한 뒤 실행
docker run -d --name seaton-bulk-app-1009 --network seaton-bulk-1009 -p 127.0.0.1:18789:8080 -e POSTGRES_DSN='postgres://seaton:seaton@seaton-bulk-pg-1009:5432/seaton?sslmode=disable' -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD=ci-e2e-password-123 seaton:bulk-before-1009
curl -fsS http://127.0.0.1:18789/readyz
```

readyz 성공 뒤 위 Playwright 명령 실행(globalSetup이 시드 생성). 전후를 바꿀 때 이 회차 컨테이너만 정리하고 PostgreSQL도 새로 만들어 시드 오염을 피한다. 브라우저가 없으면 같은 PLAYWRIGHT_BROWSERS_PATH로 `npx playwright install chromium`을 먼저 실행한다. 전체 E2E의 MCP OAuth·tracking 환경 문제를 해결하는 작업은 범위 밖이고, 전체를 안 돌렸으면 안 돌렸다고 기록한다.

대안 비교와 추정 근거

- 채택: bulkAssignments에서 필수값 누락만 처리. 기존 failures/UI를 그대로 이용하며 값이 없는 행의 DB 조회도 없앤다. 핵심 전제는 XLSX가 길이 1인 행을 반환한다는 것이고 실제 실험으로 확인했다.
- 보류: readSpreadsheet에서 모든 행을 두 칸으로 채우기. 직원 가져오기는 머리글 기반 다열 계약이므로 공유 파서를 바꾸면 영향이 넓다. CSV FieldsPerRecord를 풀어 두 입력 경로를 함께 바꾸는 것도 제외한다.
- 보류: 업로드 전에 UI에서 XLSX 검증. REST/MCP 등 직접 클라이언트에 효력이 없고 새 라이브러리·중복 규칙이 생긴다.
- 현상 유지: 비용은 없지만 오류 행을 숨기므로 문서가 약속한 실패 안내를 충족하지 못한다. 최근 no-change인 MCP 스캔 과제 재선택이나 예방 리팩터보다 이번 결함의 사용자 영향과 재현 조건이 명확하다.
- 방식: 작업 분해 기반 추정. fixture·실서버 실패 재현 5~8분, 핸들러·Go 회귀 8~12분, 수정 이미지·E2E·기본 검증 10~15분으로 기본 23~35분. 알려진 변동(이미지/브라우저/의존성 준비)에 최대 10분 contingency를 따로 두어 총 23~45분, 신뢰도 중간(통계적 백분위 추정 아님)으로 본다. 비교 가능한 과거 직원 입력검증 작업은 프로덕션 1파일+E2E였으나 실제 소요 시간이 없어 분 단위 analogous 추정으로 둔갑시키지 않았다. 추가 기능을 위한 management reserve는 0이며 새 범위가 생기면 다음 회차로 남긴다. 이 예상은 Docker/npm 다운로드가 정상이라는 전제이고, 준비만 오래 걸리면 추정을 갱신하고 검증 미완료를 숨기지 않는다.
- 적용 스킬: 별도 Skill 도구가 노출되지 않아 로컬 원문을 읽었다. [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md). PMO references/sources.md도 확인했으며 외부 기관의 수치나 검증되지 않은 확률은 인용하지 않았다.
