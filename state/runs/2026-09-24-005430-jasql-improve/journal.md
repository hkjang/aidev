# 회차 노트 2026-09-24-005430-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:54] base pinned — main@e9f3fb2
- [러너 00:54] autonomy release — 

## 정찰 노트
- 보고 단위 map 순회 비결정성을 골랐다: 실제 도구 출력(analyze_question의 aggregation_level)이 같은 질문에 day/month/year로 갈리는 것을 200회 실행으로 직접 재현했고, 문서 갱신(차선)·CI 추가·정규식 사전 컴파일은 관찰 가능한 결함이 아니거나 효과 미측정이라 제쳤다.
- 과거 성공 과제(TrimStatement, 날짜 검증, goldgen/eval)는 pinned HEAD에 없지만 프로필 지시대로 재선정하지 않고 ideas.json에 rejected로 남겼다.
- 추측으로 적은 것: "질문 등장 위치 순서"가 맞는 정렬 규칙이라는 판단은 내 설계 선택이고, 다단 GROUP BY에서 어느 단위가 옳은지는 도메인 합의 미확인(별도 아이디어로 분리).
- 구현자 주의: analyze.go의 last-wins 선택 로직과 aggregation_level 폴백은 건드리지 말고 파서만 고칠 것. 회귀는 newFixtureServer + POST /mcp tools/call로 실제 전송 경로에서 증명할 것.
- 정찰 중 임시 테스트 파일 1개를 만들어 실행 후 삭제했고 `git status --porcelain` clean을 확인했다.
- [러너 00:58] scout done — 보고 단위(일별/월별/분기별/연도별) 시간 표현을 결정적 순서로 내보내 `analyze_question`의 `aggregation_level` �

## 구현 노트
- 무엇/왜: `ParseTimeExpressions`의 보고 단위 루프가 map 리터럴 range라 TimeRange 순서가 무작위였고, `AnalyzeQuestion`의 last-wins가 이를 읽어 같은 질문의 `aggregation_level`이 회차마다 day/month/year로 갈렸다. 고정 슬라이스 + `strings.Index` 위치 기준 `sort.SliceStable`로 등장 순서 append(c712da1). analyze.go는 계약 주석 한 줄만, 로직 무변경.
- 확신 없는 곳/미검증: 보고 단위가 다른 표현의 부분 문자열로 들어간 질문(예: 따옴표·조사 붙은 변형)에서 `strings.Index`의 첫 등장 위치가 의미상 맞는지는 표 테스트 6케이스 밖에서는 미확인. `resolve_time`/`search.go:406`/`clarify.go:288` 세 호출부는 같은 파서를 읽으므로 순서가 따라 바뀌지만 이번에 전용 e2e 테스트는 `analyze_question` 한 경로만 붙였다(다른 셋은 catalog 전체 테스트 통과로만 확인).
- 일부러 안 한 것: 복수 보고 단위를 다단 GROUP BY로 살리는 개선(골든 임계값 영향 미확인), `analyze.go`의 aggregation_level 폴백·intent 추론, 파서의 다른 분기, `internal/oracle`의 기존 gofmt 드리프트 2파일(profile.go/oracle_test.go — 이번 변경과 무관).
- 다음 역할 주의: 신규 테스트는 DB/네트워크 불필요(mcp는 `newFixtureServer` + `postJSON`으로 실제 `/mcp` 핸들러 통과, catalog는 빈 `&Catalog{}`). 반복은 catalog 200회/mcp 25회로 0.02초 수준. `go test ./internal/catalog`는 전체 약 57초 걸린다.
- [러너 01:01] brief accepted — 채택 — 과제서의 근거(timeparse.go:114 map range → analyze.go:213 last-wins)가 현재 코드와 정확히 일치했고 수용 기준 1~3을 그대�
- [러너 01:01] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: 사본에서 timeparse.go/analyze.go만 main으로 되돌려 신규 테스트 4개가 실제로 FAIL함을 확인(수정 전 통과하지 않음). build/vet/catalog+mcp 전체 테스트 통과.
- 확인: 순서 변경 파급 — skeleton/search/clarify 호출부는 전부 `tr.Start != ""`로 걸러 보고 단위(Start 빈 값)에 무영향. 바뀌는 출력은 time_ranges 보고 단위 순서와 aggregation_level뿐. 구현 노트의 "세 호출부 미검증"은 이로써 해소.
- 비차단 불일치: analyze.go:213-214·timeparse.go:117의 "마지막으로 언급된 보고 단위"는 부정확. `strings.Index`는 첫 등장만 보므로 `"월별 … 연도별 … 월별"`은 agg=year(실제 마지막 언급은 월별). 결정성은 유지되어 주석 문구 문제로 판단 — 수리한다면 이 두 줄만.
- 못 본 것: 실 Oracle/PostgreSQL 경로, 골든 평가 지표 변동(golden_queries.json 미실행), 한국어 조사·따옴표가 붙은 변형 질문.
- 릴리즈 노트: "aggregation_level 비결정성 제거"로만 쓸 것. 복수 보고 단위를 다단 GROUP BY로 살리는 건 이번 범위 밖이며 여전히 마지막 하나만 남는다.
- [러너 01:04] review approved — 리뷰 승인 (risk=low)
- [러너 01:04] pr created — https://github.com/hkjang/jasql/pull/5
- [러너 01:09] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
