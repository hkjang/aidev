# 회차 노트 2026-10-02-145736-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:57] base pinned — main@494d00f
- [러너 14:57] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 보류 1순위였던 ESCAPE 검사 이전을 골랐다. 셈을 직접 해 보니 생각보다 나빴다 — LIKE 비교 19개가 8줄에 몰려 있고 한 줄에 하나뿐인 자리는 knowledge.go:167 하나라, 줄 단위 Contains 검사는 19자리 중 1자리에서만 작동한다(그래서 가치 2→3). 테스트 1파일·프로덕션 0파일이라 위험이 가장 낮다.
- 제친 후보: 감사 기간 필터(3/1/M)는 openapi·계약 테스트·httpx·화면까지 번져 "6파일 안쪽" 을 넘긴다. net.OpError·EOF(3/2/S)와 serveIdempotent(4/4/M)는 폐기용 PostgreSQL 기동이 필요해 10분 범위 밖. AdminPages 부제(1/1/S)는 가치가 너무 낮아 차선으로 돌렸다.
- 확신 없는 곳: ① 공용 걷기 헬퍼의 `likeSite` 모양은 내가 설계한 제안이고 아직 컴파일해 보지 않았다 — 구현자가 더 단순한 모양을 찾으면 그쪽을 쓰되 "두 테스트가 같은 걷기를 쓴다" 는 결론은 유지할 것. ② 차선 후보의 AdminPages.tsx:619-620 은 이번에 열어 보지 못했고 2026-09-30 기록 근거다(과제서에 미확인으로 표시).
- 구심할 것: 섭동으로 프로덕션 SQL 의 ESCAPE 를 지웠다가 반드시 `git checkout --` 로 되돌리고 최종 diff 에 search_test.go 외 파일이 없는지 확인할 것. admin_operations.go:99 의 ILIKE 5개는 ESCAPE 검사에서 건너뛰면 안 된다(접기 검사의 ILIKE skip 분기를 복사하지 말 것).
- 이것은 소스 불변식이지 런타임 증명이 아니다 — 2026-09-30 회차와 같은 표현으로 요약에 적을 것.
- [러너 15:03] scout done — ESCAPE 선언 검사를 **줄 단위**에서 **비교 단위**로 옮기기 — 19개 비교 중 18개가 "같은 줄의 다른 ESCAPE" 에 

## 구현 노트
- `internal/crm/search_test.go` 1파일만 바꿨다(프로덕션 0). 두 검색 불변식이 같은 걷기를 쓰도록 `forEachLikeComparison` 을 뽑고, ESCAPE 검사를 줄 단위 `Contains` 에서 비교 단위 창(연산자 끝 ~ 같은 리터럴의 다음 비교 시작)으로 옮겼다. 19자리 중 1자리에서만 작동하던 그물이 19자리 전부에 닿는다. `knownComparisons=19` 를 파일 수준 상수로 올려 두 테스트가 함께 개수 가드를 건다.
- 확신 없는 곳·검증 못 한 것: ① `make test` 는 **실패**한다 — 이 환경 Node 가 v22.23.1(README 요구 24+ 미달)이라 `node --test 'test/**/*.test.ts'` 의 글롭이 전개되지 않아 `Could not find …/web/test/**/*.test.ts` 로 죽는다. 내 변경은 Go 테스트 1파일이라 무관하고, 4개 파일을 직접 넘기면 27/27 PASS 임을 확인했다. Node 24+ 환경에서 `make test` 전체를 다시 돌려 확인해 주면 좋겠다(ideas.json 에 후보로 적었다). ② 비교가 **두 리터럴에 걸쳐** 쪼개져 있으면 창이 리터럴 끝에서 멈춰 오탐이 난다. 오늘 그런 자리는 없고(19개 전부 한 리터럴 안) 조용한 통과보다 시끄러운 실패가 낫다고 보아 그대로 뒀다 — 코드에 주석으로 적어 뒀다.
- 일부러 하지 않은 것: 접기 불변식의 판정 본문(`leftOperand`/`checkFormattedOperand`/`enclosing`/`operandValues`)은 한 글자도 고치지 않고 걷기만 공용화했다. 두 테스트 이름과 기존 실패 문구(`uses %s without ESCAPE; …`)도 유지했다(2026-09-30 섭동 ⑤ 가 이 문구를 기대값으로 쓴다). `admin_operations.go:99` 의 `ILIKE` 5개는 읽기만 했고 `lower(...) LIKE` 로 통일하지 않았다(인덱스 계획). 접기 검사의 `ILIKE` skip 분기는 ESCAPE 쪽에 복사하지 **않았다** — 이스케이프는 대소문자와 무관하다.
- 다음 역할이 조심할 것: 섭동 6종은 전부 `git checkout --` 로 되돌렸고 최종 `git status` 는 `internal/crm/search_test.go` 하나뿐이다(커밋 0affd5e). `s.stack` 은 `ast.Inspect` 가 내려가며 잘라내므로 **콜백 안에서만** 읽어야 한다 — 보관하면 Sprintf 인자 추적(섭동 ⑥)이 조용히 깨진다. 이 테스트는 DB 가 필요 없다. **소스 불변식이지 런타임 증명이 아니다** — 실제 PostgreSQL 이 `%`·`_` 를 리터럴로 맞추는지는 답하지 않는다.
- [러너 15:09] brief accepted — 채택 — 과제서가 센 19개/8줄 분포와 지목한 자리(`search_test.go:58-63` 의 줄 단위 `Contains`, `:89` 정규식 재사용 가능성, `:102` 
- [러너 15:09] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
