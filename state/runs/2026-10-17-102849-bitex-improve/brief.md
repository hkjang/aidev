# 과제서 — 2026-10-17-102849-bitex-improve

- 과제: `report` 패키지 내보내기 안전성 — CSV 수식 주입 차단 + 파일명 UTF-8 절단 버그 + 첫 테스트 (가치 4 / 위험 2 / 작업량 S~M)

- 왜: `report/findings.go` 는 발견 사항을 CSV·Markdown·ZIP 으로 내보내는 사용자 노출 경로인데(`server/server.go:2174,2191`, `server/finding_traffic_export.go:31,41`) 테스트가 0개다(`go test ./report` → `no test files`). 그 안에 (a) Excel 전용 BOM 을 붙여 내보내면서 공격자/LLM 이 좌우하는 이름·개요·작업설명 필드를 그대로 쓰는 CSV 수식 주입 구멍과 (b) ZIP 엔트리 이름을 바이트로 자르는 `name[:120]` 때문에 한글 제목이 120바이트 경계에서 깨지는 버그가 있다. 둘 다 순수 함수 안에서 끝나고 DB 가 필요 없어 한 세션에 고치고 테스트로 못 박을 수 있다.

## 수용 기준

1. `report.FindingsCSV` 가 내보내는 셀 중 `=`, `+`, `-`, `@`, 탭(`\t`), 캐리지리턴(`\r`) 으로 시작하는 값은 Excel/LibreOffice 가 수식으로 해석하지 못하도록 중성화된다(권장: 앞에 `'` 를 붙이거나 값 앞에 공백이 아닌 안전한 접두사를 붙이는 방식 중 하나를 골라 일관되게 적용). 숫자 ID 열과 날짜 열 등 정상 값의 표시는 바뀌지 않는다.
2. `report.FindingFilename` 의 결과는 항상 유효한 UTF-8 이다. 한글 제목이 길어 120바이트에서 잘릴 때도 룬 경계에서 잘리며(바이트 슬라이스 금지), 결과는 여전히 `.md` 로 끝나고 경로 구분자·상위 경로(`..`)를 포함하지 않는다.
3. 새 테스트 `report/findings_test.go` 가 다음을 증명한다:
   - 이름이 `=cmd|' /C calc'!A0` 인 발견 사항을 넣으면 CSV 해당 셀이 `=` 로 시작하지 않는다(`encoding/csv` 로 파싱해 셀 값을 직접 비교).
   - CSV 첫 3바이트가 UTF-8 BOM 이고 헤더 10개 열이 유지되며, 행 순서가 `sortFindingsForExport` 규칙(심각도 critical→high→medium→low→빈값, 동급은 `CreatedAt` 내림차순)과 일치한다.
   - 150자 이상 한글 제목으로 만든 파일명이 `utf8.ValidString` 을 통과하고 `strings.HasSuffix(name, ".md")` 이며 `strings.ContainsAny(name, "/\\")` 가 false 다.
   - `Severity`/`Status`/`Name`/`VulnClass` 가 모두 빈 발견 사항에서 `FindingsMarkdown`·`SingleFindingMarkdown`·`FindingFilename` 이 패닉 없이 기본값(`info`/`pending`/`미분류`, `finding_<ID>.md`)을 낸다.
   - `FindingsMarkdown`/`FindingsCSV` 가 호출자가 넘긴 슬라이스의 원본 순서를 바꾸지 않는다(이미 `append([]*db.DBFinding(nil), fs...)` 로 복사하고 있음 — 회귀 방지용).
4. `go test ./report` 가 통과하고, 기존 `go build ./...` 이 깨지지 않는다.

## 건드릴 파일 (3개, 그중 프로덕션 1~2개)

- `report/findings.go:145 FindingsCSV` — 셀 쓰기 직전에 중성화 헬퍼(예: `csvSafe(string) string`)를 통과시킨다. 헬퍼는 같은 파일 아래쪽에 추가.
- `report/findings.go:126 FindingFilename` — `name = name[:120]` 을 룬 기반 절단으로 교체(`[]rune(name)` 사용 또는 `utf8.DecodeLastRuneInString` 로 깨진 꼬리 제거). 절단 후 다시 `strings.Trim(name, "._")` 를 적용할지 판단해 `_` 로 끝나는 지저분한 이름을 피한다.
- `report/findings_test.go` — 신규. `db.DBFinding` 리터럴만 쓰고 DB 연결·`testmain` 불필요(이 패키지에는 `testmain_test.go` 가 없다). 필드는 `db/findings.go:17` 의 정의를 확인했고 전부 내보내진 필드다: `ID int64`, `Name/VulnClass/Severity/Status/Summary/Evidence/Report/TaskDescription string`, `CreatedAt time.Time`, `EvidenceVersion/ReportEvidenceVersion int64`, `TrafficBindings []db.FindingTrafficBinding`. 트래픽 묶음까지 쓰는 테스트가 필요하면 `db/finding_traffic.go:91` 의 `FindingTrafficBinding{ID, Role, Note, Snapshot}` 과 그 안의 `TrafficEvidenceSnapshot{Method, URL, Status}` 를 채우면 된다 — 다만 수용 기준에 꼭 필요하지 않으니 생략해도 된다.
- (선택, 한 줄) `.github/workflows/release.yml:57` `go test ./cmd/bitex ./config …` 목록에 `./report` 추가. 현재 `report` 는 릴리스 워크플로의 패키지 목록에도, `scripts/test-integration.go` 의 기본값 `agent,db,server,identity` 에도 없어서 **새 테스트가 CI 에서 한 번도 돌지 않는다.** 워크플로는 보호 경로이므로 이 한 줄만 바꾸고 다른 줄은 손대지 말 것. 부담되면 생략하고 과제서에 생략 사실을 남길 것.

## 검증 명령

```
cd <worktree>
go test ./report            # 필수. PostgreSQL 불필요. 최초 실행 시 모듈 다운로드 있음
go vet ./report
go build ./...              # 호출부(server) 컴파일 깨짐 확인
```

`server`·`db`·`agent`·`identity` 테스트는 `BITEX_TEST_PG_DSN` 과 격리 PostgreSQL 이 필요하다(릴리스 워크플로는 `postgres:16-alpine` 서비스를 띄운다). 이번 과제는 `server` 를 수정하지 않으므로 그 쪽을 돌릴 필요는 없다 — 돌리지 못했다면 "미실행" 이라고 적을 것.

## 위험과 피할 것

- **CSV 접두사 방식의 호환성**: `'` 접두사는 Excel 에서는 사라지지만 일반 CSV 파서에서는 값에 남는다. 프런트엔드가 이 CSV 를 되읽어 파싱하는 곳은 없음을 확인했다 — `web/src/lib/api.ts:559` 와 `web/src/app/(main)/function/findings/page.tsx:254` 에서 `csv` 는 내보내기 형식 선택값(`md-single|md-zip|csv|json`)일 뿐이고 응답은 그대로 다운로드된다. 따라서 접두사 방식이 안전하다. 수용 기준 3의 테스트가 그 전제를 명시하도록 쓰고, 선택한 방식을 함수 주석에 한 줄로 기록할 것.
- `report/report.go` 의 `Markdown`·`parseFinding`·`sevRank`·`nz` 는 건드리지 말 것. `sevRank`/`nz` 는 두 파일이 공유하므로 시그니처 변경 금지.
- `server/finding_traffic_export.go` 의 중복 파일명 처리(`used[base]`)는 `base` 가 `.md` 로 끝난다고 가정한다. 파일명 절단을 고칠 때 `.md` 접미사는 절단 **뒤에** 붙는 현재 구조를 유지할 것.
- 보호 경로 회피: `identity/`, `server/auth*`, `db/` 마이그레이션, `.github/workflows/` (위 선택 한 줄 제외)는 열지 말 것.
- 이 저장소는 커밋 메시지가 한국어이고 `type(scope): 설명` 형식이다(`fix(report): …`). 최근 커밋이 모두 그 형태다.

## 차선 후보

`enrich` 패키지 첫 테스트 (가치 3 / 위험 2 / 작업량 S) — `enrich/enrich.go` 의 순수 헬퍼 `extractTitle`(248), `hostOf`(256), `isBlank`(264), `uniq`(278) 와 `Engine.onCooldown`(162) 쿨다운 로직에 테스트가 전혀 없다. `report` 와 마찬가지로 `enrich` 역시 릴리스 워크플로와 통합 러너 양쪽 패키지 목록에 빠져 있다. 네트워크·DB 가 필요한 `doDNS`/`doHTTP` 는 건드리지 말고 헬퍼만 덮을 것.
