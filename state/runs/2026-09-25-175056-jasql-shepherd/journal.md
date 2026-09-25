# PR 처리기 노트 2026-09-25-175056-jasql-shepherd — jasql PR #8
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-25-170059-jasql-improve)
# 회차 노트 2026-09-25-170059-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:01] base pinned — main@376cf4c
- [러너 17:01] autonomy release — 

## 정찰 노트
- 고른 이유: `search.go:832` 의 접미사 목록에서 `별로` 가 `로` 뒤에 있어 도달 불가능하고, 그 결과 `회원사별`→`회원사` vs `회원사별로`→`회원사별` 로 같은 뜻이 갈린다 — 소스를 직접 읽어 연역으로 확정한 실제 결함이고, `tokenize`/`stripKoreanSuffix` 테스트가 0 개라 결함 수정과 테스트 공백 보강이 한 번에 된다. 차선인 gofmt+CI 게이트는 기계적이라 가치가 낮아 밀었다.
- 피한 것: `TrimStatement` 부재·`caf7a00` 이 `auto/2026-09-22-1244` 에만 있음을 확인해 SQL trailing 정리·eval MISS·timeparse 순서(1507e00/caf7a00/c712da1) 세 과제는 미머지 PR 중복이므로 제외했다. auth·sqlguard·execguard·ci.yml 보호 경로도 제외.
- 미확인(추측으로 적은 것): 이번 세션은 Bash 승인 게이트로 `gofmt`·`go build`·`go test`·`go run ./cmd/jasql-eval` 이 전부 거부됐다 — 골든 평가 베이스라인 숫자, gofmt 드리프트 재확인, 테스트 소요 시간은 모두 이전 회차 값이다. 구현자가 직접 찍을 것.
- 구현자 주의: 블라스트 반경이 넓다(`tokenize` 호출부 8+). 접미사 한 개의 순서만 옮기고 `expandTokens`·가중치·구분자 목록은 손대지 말 것. `월별로`(3 룬)는 길이 가드가 `별로` 에서 실패해 `로` 로 흘러내려가 기존 출력 `월별` 이 유지되어야 한다 — 이 케이스를 반드시 테스트에 넣어 과도 절단을 막을 것.
- 중단 조건: `go run ./cmd/jasql-eval` 점수가 수정 후 내려가면 순서 변경은 되돌리고 특성화 표 테스트만 제출하고 관측 수치를 보고할 것. 정직한 축소이지 실패가 아니다.
- [러너 17:06] scout done — 토크나이저의 도달 불가능한 한국어 접미사 `별로` 를 살려 `…별로` 와 `…별` 이 같은 토큰이 되게 하고, 

## 구현 노트
- 무엇/왜: `stripKoreanSuffix` 의 접미사 목록에서 `"별로"` 를 `"로"` 앞으로 옮겼다(커밋 31da0d4). 첫 일치에서 `return` 하므로 `"로"` 가 앞에 있으면 `"별로"` 분기는 도달 불가였고, 그 탓에 `회원사별`→`회원사` 인데 `회원사별로`→`회원사별` 로 갈렸다. 신규 `internal/catalog/tokenize_test.go` 가 `tokenize`/`stripKoreanSuffix` 의 첫 테스트다.
- 확신 없는 곳·검증 못 한 것: (1) 골든 평가는 수정 전후 **수치가 완전히 동일**했다(cases 80 / table 0.94 / column 0.9 / join 0.94 / metric 1 / sql_valid 1) — 비회귀는 증명됐지만 이 변경이 리콜을 **개선**한다는 주장은 이 80 케이스로는 증명되지 않았다. 골든셋에 `…별로` 표기 케이스가 있는지는 세지 않았다. (2) 실제 검색 품질(사용자 질의)에 대한 효과는 미측정. (3) `data/kcb` 실데이터 외 다른 메타데이터에서의 동작은 미확인.
- 일부러 하지 않은 것: `internal/oracle/profile.go`·`oracle_test.go` 의 gofmt 드리프트는 이번에도 `gofmt -l` 로 실재 확인했으나 diff 오염을 피해 고치지 않았다. `analyze.go`·`patterns.go`·`template_match.go` 의 원문 기반 `hasAny`/`Contains` 경로는 계약이 달라 통합하지 않았다(소스로 원문 기반임을 확인). `expandTokens`·가중치·구분자 목록·길이 가드 식은 건드리지 않았다.
- 다음 역할이 조심할 것: `tokenize` 는 호출부 8+ 의 공용 프리미티브라 블라스트 반경이 넓다 — 이 파일을 다시 만질 때는 반드시 `go run ./cmd/jasql-eval` 전후 비교를 함께 할 것(베이스라인은 위 숫자). `go test ./internal/catalog` 는 약 57 초, `./internal/mcp` 약 10 초 걸린다. `-run Tokenize` 는 신규 4 개 테스트를 모두 잡는다(`TestTokenizeStripKoreanSuffix` 로 이름을 맞춰 둠).
- [러너 17:11] brief accepted — 채택 — 과제서의 근거(`search.go:832` 접미사 순서로 `별로` 분기 도달 불가, `tokenize`/`stripKoreanSuffix` 테스트 0 개)가 현재 코�
- [러너 17:11] verify passed — 검증 3개 통과 (auto)
- [러너 17:11] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 17:11] pr created — https://github.com/hkjang/jasql/pull/8

## 심사 노트
- 확인한 것: origin/main 의 search.go 로 되돌려 재실행해 신규 테스트가 실제로 FAIL(회원사별로/지점별로/상품별로/연도별로) 하고 HEAD 에서 PASS 함을 직접 확인했다 — 대역 없이 프로덕션 tokenize/stripKoreanSuffix 를 호출한다. 확인 후 워킹트리 clean 복원.
- 확인한 것: go build ./... OK, gofmt -l internal/catalog 무출력, go test ./... 전 패키지 ok(catalog 56.2s/mcp 10.1s), 골든 평가 수정 전후 동일(80/0.94/0.9/0.94/1/1).
- 확인한 것: 같은 "별" 값을 읽는 analyze.go·patterns.go·timeparse.go 는 모두 원문 문자열 파서라 토큰 경로와 계약이 달라 영향 없음. glossary 의 "성별" 은 길이 가드로 보존됨.
- 못 본 것: 골든 80 케이스 밖 실사용 질의의 검색 품질 효과, data/kcb 외 메타데이터 동작. 어간이 "별" 로 끝나는 4룬+ 단어에 "로" 가 붙는 어형(예: "무차별로")은 더 잘리나 도메인에 실례가 없어 차단하지 않았다.
- 권고 근거: 결함은 실재·최소 범위, 테스트가 결함을 실제로 고정, 비회귀 실측, 보호 파일·개인정보·되돌릴 수 없는 변경 없음 → approve/merge, risk low.
