# 회차 노트 2026-10-05-161703-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@7d2cac3
- [러너 16:17] autonomy release — 

## 정찰 노트
- 고른 이유: `delimited` 패키지가 "업로더와 IMPORTDATA 두 문이 같은 파일을 같게 읽어야 한다"를 주석으로 못 박았는데 한쪽에만 utf8.Valid 검사가 있다 — 같은 바이트열을 두 입구에 실제로 넣어 한쪽은 표를 내고 한쪽은 거절하는 것을 이번에 돌려 확인했다. 프로덕션 1파일·새 의존성 0.
- 제친 후보: automation `cronValue` 부호 허용(차선으로 남김, 저장된 일정을 깨뜨릴 수 있음), mail `SendNow` ctx·observability `WithAttrs`(둘 다 재현 자리가 없거나 패키지 포화), LazyQuotes 비대칭(계약 재결정이라 금지).
- 추측으로 적은 것: 깨진 바이트가 Postgres text 컬럼에서 거절되는지는 미확인(DB 필요). JSON 응답의 U+FFFD 치환도 코드로 확인하지 않았다 — 수용 기준은 utf8.Valid=false 와 두 문의 불일치만으로 세웠으니 DB 주장에 기대지 말 것.
- 조심할 것: 인코딩 추측(CP949 변환·x/text 추가) 금지, `internal/delimited`·`importexport` 손대지 말 것, 같은 함수의 `LazyQuotes` 는 그대로, 오류 코드는 같은 함수의 기존 읽기 실패와 같은 `#VALUE!` 로 두어 ADMIN_GUIDE 를 끌어들이지 말 것.
- 작업 트리: 재현용 임시 테스트(`internal/external/zz_recon_test.go`)를 넣어 돌리고 지웠다. `git status --porcelain` 출력 없음.
- [러너 16:21] scout done — IMPORTDATA 가 UTF-8 이 아닌 본문을 깨진 바이트째로 칸에 넣는다 — 업로드 문은 같은 바이트를 거절한다 (가

## 구현 노트
- `internal/external/fetcher.go:parseCSV` 의 `delimited.ToUTF8` 직후에 `utf8.ValidString` 검사 3줄을 더해, 표시 없는 비-UTF-8 본문을 업로드 문과 같이 거절한다(코드는 같은 함수의 기존 읽기 실패와 같은 `#VALUE!`, 문구 `"CSV 가 UTF-8 이 아닙니다"`). 프로덕션 1파일·새 의존성 0.
- 확신 없는 곳: 정찰이 적은 "Postgres text 컬럼 거절·JSON 응답 U+FFFD 치환" 은 **이번에도 확인하지 않았다** — 수용 기준과 테스트는 `utf8.Valid=false` 인 칸 값과 두 문의 불일치만으로 세웠으니 그 주장에 기대지 말 것. 거절이 캐시되는지(`config.CacheFor` 로 성공과 같이 보관됨)는 기존 동작 그대로 두었고 따로 테스트하지 않았다 — 비평가가 볼 자리라면 여기다.
- 일부러 하지 않은 것: 인코딩 추측(CP949 변환·`golang.org/x/text`) — `ToUTF8` 이 "표시가 없으면 추측하지 않는다"를 계약으로 둔다. `LazyQuotes` 비대칭(계약 재결정), `ADMIN_GUIDE.md:785` 정리(같은 `#VALUE!` 를 써서 문서 변경이 불필요), `internal/delimited`·`importexport/service.go`·캐시·allow-list/SSRF·docs/PDF 도 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 `httptest` TLS 서버를 띄우므로 루프백이 필요하다(DB 는 불필요). 되돌리기 확인은 검사 3줄만 지우면 재현된다 — 그때 빨강이 되는 것은 새 테스트 하나뿐이어야 한다.
- 검증: `go test ./internal/external -run TestImportData -v`(빨강 1 → 초록 6), `go test ./...` exit 0 / ok 19 / FAIL 없음, `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal ./pkg`(출력 없음), `check-release-docs.sh`(ok v0.261.0), `check-commit-identities.sh HEAD`(exit 0). 커밋 0fd9038.
- [러너 16:24] brief accepted — 채택 — 과제서의 재현이 지금 코드와 정확히 맞았고(`parseCSV` 는 rows=2 cols=2 에 `utf8.Valid=false` 인 칸 값, `importexport.Parse` 는
- [러너 16:24] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 검사 3줄을 지워 직접 재현했다 — 빨강은 새 테스트 하나뿐이고 출력이 주장한 증상(rows=2 cols=2, `\xc0̸\xa7`)과 같다. `importexport/service.go:160` 이 같은 자리에서 같은 검사를 하는 것도 읽어 두 문의 대칭을 확인했다. `go build`·`go vet`·`go test ./...`(FAIL 0)·`gofmt -l`·트리 clean 재실행.
- 구현자가 의심한 두 자리: Postgres/JSON 주장은 미확인이나 수용 기준이 기대지 않아 무해. 거절 캐시는 `config.CacheFor` 전체 동안 남고 `failureCacheFor` 단축은 fetch 단계 `remoteFailure` 에만 걸리지만, 기존 'CSV 를 읽지 못했습니다' 와 똑같아 이번 결함이 아니다.
- 못 본 것: 웹/DB 통합/E2E/PDF(이 변경이 닿지 않음), 오류 메시지 문자열 단언(양쪽 테스트 모두 코드만 본다).
- 릴리즈가 알 것: 사용자에게 보이는 동작 변경 — 비-UTF-8 CSV 가 깨진 표 대신 #VALUE! 를 낸다. 문서/PDF 동반은 불필요(ADMIN_GUIDE 에 오류 조건 표가 없고 :785 서술은 오히려 맞는다).
- 다음 회차 후보로 남는 자리: WEBSERVICE 는 여전히 비-UTF-8 본문을 `Text` 로 칸에 넣는다(`fetcher.go:181`) — 업로드 상대가 없어 이번 대칭과는 무관하다.
- [러너 16:27] review approved — 리뷰 승인 (risk=low)
- [러너 16:27] pr created — https://github.com/hkjang/kanpic/pull/41
- [러너 16:36] ci passed — 검사 2개 모두 success
- [러너 16:36] merge done — 0fd9038
- [러너 16:49] release published — v0.262.0
- [러너 16:50] assets verified — v0.262.0 자산 1개 (이전 v0.261.0: 2)
