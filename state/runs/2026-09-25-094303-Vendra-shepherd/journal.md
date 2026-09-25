# PR 처리기 노트 2026-09-25-094303-Vendra-shepherd — Vendra PR #132
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-25-073515-Vendra-improve)
# 회차 노트 2026-09-25-073515-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:35] base pinned — main@7048feb
- [러너 07:35] autonomy release — 

## 정찰 노트
- 골랐다: recommend_suppliers 가 도구 설명·USER_GUIDE 에 공개한 「최대 위험 등급」 인자가 스키마에도 SQL 에도 없다. 직전 두 회차가 채택된 「공개한 계약과 실제 동작의 불일치」와 같은 계열이고, 같은 분기의 minScore 조용한 0 까지 한 테스트에서 함께 증명할 수 있어 M 하나로 묶었다.
- 제친 이유: minScore 단독(S)은 가치가 너무 작아 1순위 안으로 흡수했고, compare_suppliers maxItems·days 스키마 730 은 「상한 값을 얼마로」라는 정책 결정이 선행이라 무인 회차에 맞지 않는다. 관제탑 200건·통화 합산은 위험 3 이고 통화 쪽은 사람이 반려한 이력이 있다.
- 추측으로 적은 것: 인자 이름 `maxRisk` 와 enum 에서 CRITICAL 제외는 내 제안이지 코드에 근거가 없다. minScore 비숫자를 거절할지 파싱할지도 계약 결정이 남아 있으니 구현자가 하나 골라 주석에 적을 것.
- 조심할 것: SQL 이 `$1..$7` 로 꽉 차 있어 인자를 더하면 번호를 다시 세야 하고 orgInScope 의 문자열 자리까지 어긋나기 쉽다. 인자 없는 호출의 결과 집합이 바뀌면 회귀다 — 기본 경로의 `NOT IN('CRITICAL')` 를 화이트리스트로 갈아치우지 말 것.
- 프로필을 새로 썼다: 이전 프로필이 mcpoauth.go 를 「부재」로 적었으나 522줄로 트리에 있다. 기준 커밋도 28cd677 → 7048feb 로 갱신했다.
- [러너 07:39] scout done — recommend_suppliers 가 설명에 적힌 「최대 위험 등급」을 실제 인자로 받게 하고, minScore 를 문자열로 받았을 �

## 구현 노트
- recommend_suppliers 가 설명·가이드에 공개한 「최대 위험 등급」을 실제 인자(`maxRisk`)로 받게 하고, `minScore` 가 문자열·불리언·객체로 오면 조용한 0 이 되던 것을 거절로 바꿨다(commit dc31174). 등급 인자가 있을 때만 조건을 **덧붙이므로** 인자 없는 호출의 SQL 문자열과 인자 7개는 글자 그대로 같다.
- 확신 없는 곳: (1) `numberArg` 는 float64 만 숫자로 본다 — `mcpCall` 이 `json.Unmarshal` 을 쓰므로 지금은 맞지만, 디코더가 `UseNumber` 로 바뀌면 정상 숫자를 거절한다(`intNumber` 도 같은 전제라 그 위험은 원래 있었다). (2) `mcpRiskCeilings = riskGrades[:len-1]` 는 CRITICAL 이 마지막이라는 순서 전제에 의존한다 — DB 없이 도는 스키마 테스트가 `"LOW MEDIUM HIGH CRITICAL"` 를 못박아 두었다. (3) `minScore` 비숫자를 「거절」로 고른 것은 계약 결정이다(주석에 이유를 적었다). 이미 `"80"` 을 보내던 클라이언트가 있다면 200 → isError 로 바뀐다.
- 일부러 하지 않은 것: `ORDER BY ... , risk_level` 의 문자열 정렬(HIGH 가 LOW 앞에 온다)은 기본 결과 집합/순서를 바꾸지 않기 위해 손대지 않았다. `docs/USER_GUIDE.md:297` 도 지시대로 그대로다. 다른 MCP 도구의 상한·days 정책·share 분모도 건드리지 않았다.
- 다음 역할이 조심할 것: 새 `internal/httpapi/mcp_recommend_integration_test.go` 의 두 통합 테스트는 DB(세 DSN)가 있어야 돌고 없으면 SKIP 된다 — 초록만 보고 검증됐다고 읽지 말 것. `TestMCPRecommendSchemaOffersTheRiskCeiling` 만 DB 없이 돈다. fixture 는 `SC-REC-` 접두사로 시드하고 병렬 실행 금지, cleanup 은 `context.Background()`.
- [러너 07:46] brief accepted — 채택 — 과제서의 근거가 코드와 정확히 맞았다(스키마에 등급 인자 없음, SQL 고정 제외, `:608` 의 조용한 0). 과제서가 예�
- [러너 07:46] verify passed — 검증 7개 통과 (auto)
- [러너 07:46] pr created — https://github.com/hkjang/Vendra/pull/132
- [러너 07:46] merge stopped — 긴급 중지

## 심사 노트
- 확인한 것: 전용 Postgres 16 을 띄워 세 DSN 으로 신규 테스트 3개 통과, 이어서 테스트는 그대로 두고 recommend_suppliers 분기만 수정 전으로 되돌려 재실행 — ceiling 3개·인자 거절 7개가 FAIL 하므로 테스트가 변경을 실제로 검증한다. 전체 `go test ./internal/... ./cmd/...` ok, gofmt/vet 무출력. 스코프(own 토큰 빈 결과)와 spend.read 게이트(annual_spend nil)도 실행으로 확인. worktree 는 dc31174 로 원복, 컨테이너 삭제.
- 못 본 것: 웹 스위트·`make build`·offline-release 는 돌리지 않았다(diff 가 Go 3개 파일뿐이고 web·릴리즈 경로를 건드리지 않아 불필요하다고 판단). 외부 MCP 클라이언트의 실제 호출 습관은 저장소 밖이라 확인 불가.
- 권고 근거: merge. 보호 파일·마이그레이션·새 공개 경로·새 의존성이 없고 SQL 은 화이트리스트 + 바인드 파라미터라 보안·개인정보 차단 사유가 없다. 인자 없는 호출의 SQL·파라미터가 글자 그대로 같아 기본 결과 집합이 보존되고, revert 로 완전히 되돌아온다.
- 남긴 참고: (1) rows.go:602 의 `riskGrades[:len-1]` 는 배열 공유·cap 4 — 훗날 append 하면 "CRITICAL" 이 덮인다(현재 append 없음). (2) integrations.go:616-621 주석의 NULL 근거는 불필요(이미 `NOT IN('CRITICAL')` 이 떨어뜨림). (3) minScore 문자열이 200 → isError 로 바뀌는 것이 유일한 호출자 가시 파괴 변경.
