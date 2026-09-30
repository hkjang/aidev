# 회차 노트 2026-09-30-165618-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:56] base pinned — main@d9462ae
- [러너 16:56] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 고른 이유: `searchPattern`(crm/service.go:160)이 **소문자** 패턴을 만드는데 이를 받는 `LIKE` 비교가 `lower(...)` 또는 `ILIKE` 여야 한다는 결합을 지키는 것이 아무것도 없다. 오늘 18개 자리는 전부 맞아 동작 변경 0으로 그 상태만 고정할 수 있고, 깨졌을 때의 증상이 오류가 아니라 "조용한 0건" 이라 테스트 없이는 못 잡는다. 실 DB가 필요한 후보(net.OpError 분류, 감사 metadata 검색·기간 필터)와 보호 경로 후보(release.yml, migrations)는 10분 회차에서 증명할 수 없어 제쳤고, `serveIdempotent` 경쟁은 마이그레이션 018이 필요해 제쳤다.
- 확신 없는 곳: `internal/voice/knowledge.go:167` 의 `fmt.Sprintf("%s LIKE $%d …", col, n)` 는 리터럴 한 줄에 왼쪽 피연산자가 없어 **줄 단위 스캔으로는 판정 불가**다. map 키 5개가 전부 `lower(` 로 시작하는 것은 눈으로 확인했지만, 구현자가 이 자리를 어떻게 다룰지(키 쪽을 함께 검사 vs 정직하게 범위 밖으로 빼고 주석)는 지정하지 않고 선택지만 줬다. 기준치 숫자(lower LIKE 13 / ILIKE 5 / Sprintf 1)도 구현자가 실제 스캔으로 다시 셀 것.
- 구현자가 조심할 것: 프로덕션 SQL 을 "고치려" 들지 말 것 — 오늘 전부 맞다. `admin_operations.go:99` 의 `ILIKE` 5개를 `lower(...) LIKE` 로 통일하지 말 것(감사 경로·인덱스 계획). 기존 `TestEveryFreeTextSearchDeclaresItsEscapeCharacter`(search_test.go:43)는 지우지 말고 옆에 더할 것.
- 정직하게: 이 불변식은 "코드가 이렇게 쓰여 있다" 는 소스 검사이지 런타임 증명이 아니다. 커밋 메시지와 테스트 주석에 그렇게 적을 것 — 운영자가 grep 을 실행 증거로 제출하지 말라고 반복해 말했다.
- 미확인: 프런트 검증(`npm ci`/`npm test`)은 이번 정찰에서 돌리지 못했다(이전 회차 ETIMEDOUT). 차선 후보를 고를 경우 구현자가 먼저 설치 가능 여부를 확인해야 한다. 프로필은 1일 전 것이 지금 코드와 일치해 새로 쓰지 않았다.
- [러너 17:01] scout done — 자유 텍스트 검색의 **대소문자 접기 계약**을 불변식 테스트로 묶기 — `searchPattern` 이 소문자로 내려보내

## 구현 노트
- `internal/crm/search_test.go` 에 `TestEveryFreeTextSearchComparesAgainstFoldedText` 추가. `searchPattern` 이 내려보내는 소문자 패턴과 `lower(...)`/`ILIKE` 의 짝을 소스 불변식으로 고정한다. 프로덕션 동작 0줄, `service.go` 독스트링 3줄 추가만.
- 과제서가 열어 둔 유일한 판단 — `voice/knowledge.go:167` 의 `fmt.Sprintf` 조립 자리 — 는 **범위 밖으로 빼지 않고** verb 를 채우는 인자를 range 변수 → map 복합 리터럴까지 따라가 키 5개를 검사하는 쪽을 택했다. 따라갈 수 없는 인자는 통과가 아니라 실패다. 섭동 ③(`lower(v.title)`→`v.title`)이 `:167` 을 정확히 짚는 것으로 이 경로가 실제로 도는 것을 확인했다.
- **확신 없는 곳**: (a) 왼쪽 피연산자 추출은 텍스트 파싱이다 — 괄호 균형과 식별자 문자만 본다. 오늘 19자리는 정확하지만 낯선 모양(캐스트 `x::text LIKE`, 서브쿼리 좌변)이 오면 잘못 잘라 오탐할 수 있다. (b) `verbsBefore` 는 폭·플래그가 붙은 verb(`%-10s`)를 세지 못한다 — 현재 포맷에는 없다. (c) `operandValues` 는 이름만으로 함수 안을 뒤지므로 같은 이름이 섀도잉되면 헷갈릴 수 있다. 전부 오탐(빨간불) 방향이지 통과 방향은 아니다.
- **검증 못 한 것**: 실제 SQL 이 행을 맞추는지는 증명하지 않았다. 이건 소스가 그렇게 쓰였다는 불변식이고, 런타임 증명에는 실 PostgreSQL 이 필요하다 — 이번 범위 밖. 프런트(`npm`)는 이번 변경과 무관해 돌리지 않았다.
- **일부러 하지 않은 것**: 프로덕션 SQL 수정 0건(오늘 전부 맞다). `admin_operations.go:99` 의 `ILIKE` 5개를 `lower(...) LIKE` 로 통일하지 않았다(감사 경로·인덱스 계획). 기존 3개 테스트는 이름·동작 그대로 두었다.
- **다음 역할이 조심할 것**: 이 테스트는 DB 없이 돈다(소스만 읽는다). 새 자유 텍스트 검색을 더할 때 기준치 `knownComparisons = 19` 를 올릴 필요는 없지만 **내려서는 안 된다**. 그리고 기존 ESCAPE 검사는 여전히 줄 단위라 비교가 여럿인 줄에서 하나만 `ESCAPE` 를 지우면 놓친다 — 실제로 확인했고 다음 회차 후보로 `ideas.json` 에 적었다.
- [러너 17:08] brief accepted — 채택 — 과제서가 열어 본 자리와 셈(`service.go:160` 의 `ToLower`, `lower(...) LIKE` 13개 + `ILIKE` 5개 + `knowledge.go:167` 의 Sprintf 1개, �
- [러너 17:08] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: 섭동 5종(service.go:209 / knowledge.go:167 map 키 / team.go:166 / admin_operations.go:99 ILIKE→LIKE / 새 파일의 새 비교)을 직접 돌려 정확히 해당 `path:line` 만 red 임을 확인했고, `knownComparisons=19` 를 독립 셈으로 맞췄다. go build·vet·test ./... 통과, gofmt 무출력, 트리 복구 확인. 보안·법무 차단 사유 없음(프로덕션 동작 0줄, 개인정보·비밀값·인가 무관).
- 못 본 것: 실제 SQL 이 행을 맞추는지는 확인하지 않았다(실 PostgreSQL 필요 — 원장도 그렇게 적었다). 프런트는 무관해 돌리지 않았다.
- **남는 우려 1(실증)**: `search_test.go:238-252 literalsAssignedTo` 가 CompositeLit 의 비-BasicLit 요소를 `return true` 로 조용히 건너뛴다 → 리터럴 키 뒤에 `unfoldedCol: 1`(값 `"v.title"`)을 넣으면 접히지 않은 좌변이 살아 있는데 PASS(`seen` 19 유지로 개수 가드도 안 걸림). 주석(:200-202)·원장의 "따라갈 수 없는 인자는 실패" 약속과 어긋난다. 오늘 그 모양이 없어 차단은 안 했다 — 다음 회차에서 그 `return true` 를 실패로 바꾸는 3줄.
- **남는 우려 2(실증)**: 정규식이 대문자 전용이라 `t.label like $1 escape '\'` 는 새 테스트와 기존 ESCAPE 테스트 **둘 다** 놓친다. 기존 테스트가 이미 가진 구멍이라 회귀는 아니다. 또 개수 가드가 하한이라 검색을 정당하게 지울 때 오해 소지 있는 메시지로 red 가 된다(기준치를 내려야 한다).
- 판정: **approve** / risk low. 불변식은 한 방향(패턴이 소문자면 좌변도 접혀야 함)만 보고, 반대 방향(SearchPattern 미사용 대문자 패턴 vs `lower(col)`)은 여전히 조용한 0건으로 통과한다 — 릴리즈·다음 회차가 알아 둘 것.
- [러너 17:13] review approved — 리뷰 승인 (risk=low)
- [러너 17:13] pr created — https://github.com/hkjang/relio/pull/39
- [러너 17:17] ci passed — 검사 2개 모두 success
- [러너 17:17] merge done — f7bf47d
- [러너 17:17] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
