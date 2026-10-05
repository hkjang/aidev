# 회차 노트 2026-10-05-202738-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:27] base pinned — main@9a3fec6
- [러너 20:27] autonomy release — 

## 정찰 노트
- 아홉 회차 연속 '증명 수단 없음'으로 탈락한 `rows.Err()` 과제를 골랐다 — 이번에 수단을 실제로 확인했다: pgx v5.7.6 `rows.go:27-57` 의 `Rows` 에 `Next() bool`·`Scan(...any) error`·`Err() error` 가 모두 있어 3메서드 인터페이스로 가릴 수 있고, 지난 회차의 `fakeReleaser`(employees_test.go:76-110)가 같은 수법의 선례다. 스캔 루프만 뽑으면 프로덕션 1파일로 끝나 '여섯 파일' 규칙도 지킨다.
- 차선(가져오기 실패 사유에 빈 열 적기)을 제친 이유: 세 회차째 밀린 S 짜리이고 값이 2 인데, 1순위는 수단이 열린 바로 이 회차에 집어야 또 아홉 회차를 기다리지 않는다. `internal/app 에 DB 하네스 조사`는 목적이 사라져 기각, `web/e2e/README` 는 예고대로 가치를 1로 내렸다.
- 과제서에 추측으로 적은 것: ① `rows.Err()` 를 `Close()` 전에 부를 때의 pgx 동작은 주석만 읽었고 실험하지 않았다(그래서 Scan 오류는 `rows.Err()` 를 보지 말고 그대로 반환하라고 적었다) ② 헬퍼 이름(`rowScanner`·`scanEmployees`)은 제안일 뿐 기존 이름 관례와 맞춰도 된다.
- 구현자가 조심할 것: `items` 를 `[]T{}` 로 시작해야 JSON 이 `null` 이 되지 않는다. `listHistory` 의 `total`/`totalCapped` 와 `listEmployees` 의 SQL·limit 500 은 그대로 둘 것. 다른 파일(seats·maps·keys·detection·mcp)의 같은 패턴은 이번에 손대지 말 것.
- 프로필은 0일 전 것을 그대로 쓴다(새로 쓰지 않음). 단 두 군데가 낡았다: `employees.go` 는 424줄이 아니라 **497줄**이고 기준 커밋은 `50768e3` 가 아니라 `9a3fec6` 다. 그 외는 이번 읽기와 맞았고 `go build`·`go vet`·`go test ./...` 는 base 에서 녹색이다.
- [러너 20:32] scout done — 직원·조직·이력 목록이 행 읽기 실패를 삼키고 부분 목록을 200 으로 돌려주는 것을 멈춘다 (가치 3 / 위험

## 구현 노트
- 세 목록 핸들러(`listOrganizations`·`listEmployees`·`listHistory`)의 `if rows.Scan(...) == nil` 스캔 루프를 `scanOrganizations`/`scanEmployees`/`scanHistory` 순수 함수로 뽑아 Scan 오류를 즉시 반환하고 루프 뒤 `rows.Err()` 도 반환하게 했다. 아홉 회차 막혔던 증명 수단은 `rowScanner`(`Next`/`Scan`/`Err` 3메서드)로 열렸다. 프로덕션 1파일(`internal/app/employees.go`), 커밋 a3cc4a5.
- 확신 없는 곳: **없다고 말하기보다 정확히 적는다.** 가짜 rows 단위테스트(5건)에 더해 실서버에서 `rows.Err()` 경로를 실제로 일으켜 변경 전/후를 대조했다 — 같은 DB에 두 이미지를 붙이고 `ALTER TABLE organizations ALTER COLUMN color TYPE integer` 로 캐시된 plan 을 깨면 `rows=0 / Scan err=nil / rows.Err()=cached plan must not change result type (0A000)` 가 되고, 변경 전은 `{"items":[]}` 200, 수정본은 `database_error` 500 이다. **다만 Scan 자체가 실패하는 경로는 실서버에서 일으키지 못했다**(pgx 가 int4→*string 을 "1" 로 강제 변환해 Scan 이 성공해 버린다) — 그 분기는 `fakeRows` 단위테스트로만 증명됐다. 비평가는 여기를 먼저 볼 것.
- 일부러 하지 않은 것: ① `seats.go`·`maps.go`·`keys.go`·`detection.go`·`mcp.go` 에 같은 패턴이 남아 있지만 과제서 지시대로 건드리지 않았다(파일 수가 재작업률을 가른다 — ideas.json 에 다음 회차 후보로 적었다) ② `pgx.CollectRows` 로 바꾸지 않았다 ③ `listEmployees` 의 SQL·`limit` 500·필터 조건과 `listHistory` 의 `total`/`totalCapped` 는 한 글자도 안 건드렸다 ④ 사용자에게 보이는 기능 변화가 없어 문서를 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트 5건은 **DB 없이 돈다**(`go test ./internal/app` 로 끝. internal/app 67→72건). `fakeRows.Scan` 은 `*string`/`**string`/`*any` 세 dest 타입만 알아서, 다른 파일의 루프로 이 수법을 넓히면 타입 분기를 더해야 한다. 헬퍼의 `items` 는 반드시 `[]T{}` 로 시작해야 한다 — `var items []T` 로 바꾸면 JSON 이 `null` 이 되어 프런트가 깨지고, 그 회귀는 테스트로 막아 두었다.
- [러너 20:43] brief accepted — 채택 — 과제서가 지목한 자리와 수단이 모두 지금 코드와 맞았다. 세 루프(employees.go 의 24-30·81-86·485-491), 올바른 본보기
- [러너 20:44] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현자가 의심한 자리를 먼저 깼다: pgx v5.7.6 `rows.go:235-260`(Scan 오류 → `fatal`) 과 `rows.go:211-218`(`fatal` 이 `rows.err` 를 세우고 `Close()`)를 읽어, Scan 오류를 그대로 반환하는 것이 `Err()` 와 같은 값이고 이미 닫힌 rows 의 `defer Close()` 도 안전함을 확인했다 — 주석과 동작이 맞다.
- 테스트가 바뀐 경로를 지난다: `ledger-entry.md:5` 의 역검증(옛 루프로 되돌리면 `scanEmployees 오류 = <nil>`)과 실서버 0A000 대조가 증상과 일치하고, `rows.scans != 1` 단언 자체가 신·구를 가른다. `go build/vet/test ./...`·`gofmt` 전부 녹색.
- 정상 데이터의 새 500 회귀 없음을 스키마로 대조(`migrations.sql:14-22·24-38·158-168` — 비포인터 dest 가 전부 NOT NULL 또는 COALESCE). 프런트 두 호출자(`EmployeesPage.tsx:92`·`SeatMapPage.tsx:375`)는 500 을 받아낸다. 보안·법무 차단 사유 없음(인증·권한·의존성·개인정보 수집 변화 없고 오류 문장은 고정).
- 못 본 것: 핸들러 배선 4줄의 자동 테스트는 없고 실서버 프로브는 조직 경로만이다(직원·이력은 읽기로만 확인). E2E·웹 테스트는 프런트 무변경이라 돌리지 않았다.
- 승인. 다음 회차가 알 것: 같은 삼킴이 `seats.go:36`·`maps.go:27·72·126`·`keys.go:28`·`dashboard.go:317`·`detection.go:468` 에 남아 있고, `employees.go:432-451` 의 listHistory 중복 doc 주석은 main 부터 있던 기존 흠이다.
- [러너 20:47] review approved — 리뷰 승인 (risk=low)
- [러너 20:47] pr created — https://github.com/hkjang/seaton/pull/44
- [러너 20:52] ci passed — 검사 2개 모두 success
- [러너 20:52] merge done — a3cc4a5

## 릴리즈 노트
- v1.4.16 을 이전 회차와 똑같은 방식으로 냈다 — 패치 한 칸(1.4.15→1.4.16), 문서 버전 표기 갱신 커밋(`docs: v1.4.16 기준으로 문서 정비`, e86f453) + 같은 본문의 **주석 태그** `v1.4.16`. detached HEAD 에서 커밋·태그했고 원격에는 아무것도 보내지 않았다.
- 바뀐 파일 10개는 직전 세 릴리즈(50768e3·61dbe3c·c051c38)와 같은 목록이다: `README.md`(8줄)·`docs/ADMIN_GUIDE.md`(9줄)·`docs/USER_GUIDE.md`(1줄)·`docs/ROADMAP_PLAN.md`(1줄) 의 버전 문자열을 고치고, `python3 scripts/build-docs.py` 로 HTML(+ROADMAP_PLAN.pdf)을, `aidev/tools/guide/md2pdf.mjs --version v1.4.16` 로 USER_GUIDE.pdf·ADMIN_GUIDE.pdf 를 다시 구웠다.
- 주의할 것 하나: `build-docs.py` 전체 실행은 `EXECUTIVE_REPORT.html`·`USER_GROUPS_ANALYSIS.html` 에도 figure/figcaption CSS 3줄을 더한다(두 파일은 v1.4.0 이후 재생성되지 않아 생성기보다 낡았다). 이번 릴리즈 범위가 아니므로 네 파일을 `git checkout` 으로 되돌렸다 — 다음에 문서 생성기 정비를 할 때 따로 한 커밋으로 묶는 편이 낫다.
- 검증: `go build ./...`·`go vet ./...`·`go test ./...`(internal/app 포함 전부 ok)·`gofmt -l .`(출력 없음)·`git diff --check` 통과. README 가 적은 로컬 릴리즈 검증도 그대로 밟았다 — `bash scripts/release-image.sh 1.4.16` 으로 `seaton:v1.4.16` 이미지를 굽고 `gzip -t SeatOn-v1.4.16.tar.gz`(47,628,268바이트) 통과, 이미지가 실제로 뜨는지까지 확인한 뒤 산출물은 지웠다(워크플로가 다시 굽는다).
- 자산은 비워 둔다(`assets: []`, `github_release: false`): `.github/workflows/release.yml` 이 `v*.*.*` 태그 push 에 반응해 `SeatOn-v1.4.16.tar.gz` 를 굽고 `gh release create --generate-notes --title "SeatOn v1.4.16"` 로 릴리즈까지 만든다. 사람이 올릴 것은 없다.
- [러너 21:02] release published — v1.4.16
- [러너 21:04] assets verified — v1.4.16 자산 1개 (이전 v1.4.15: 1)
