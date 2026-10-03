# 회차 노트 2026-10-04-011735-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:17] base pinned — main@2692aae
- [러너 01:17] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 고른 이유: 지난 회차 차선(`choice()` enum 대조)을 재평가하다 **받는 쪽 정본이 `migrations/` 의 CHECK 제약**임을 실측했다(42개/12파일). 컬럼명 하나당 집합이 하나뿐인 6개 키만 떼면 손 매핑 없이 집합 등식을 세울 수 있어 M 이 S 로 줄었다 — `release.yml`(보호 경로·롤백 이력)과 `esbuild`(package-lock 재생성이 네트워크)는 10분 회차에 안 맞아 제쳤다.
- 확신 있는 것: CHECK 42개와 컬럼별 분포, 6개 키가 오늘 전부 일치, `$ref` 는 `parameters.go:307` 한 자리이며 `map[string]any` 라 "map 아니면 skip" 으로는 안 걸러진다, `migrations.Files` embed FS 존재.
- 추측으로 적은 것: `migrations.Files` 를 `internal/api` 테스트에서 import 해도 컴파일된다(소스로만 확인, 돌려 보지 않음) · `001_initial.sql` 의 4개 제약이 어느 컬럼인지 미확인.
- 구현자 주의: `migrations/*.sql` 과 `parameters.go` 를 **고치지 말 것** — 오늘 불일치는 없고 이 과제는 그물이다. 섭동은 `git checkout --` 로 되돌려 최종 diff 가 테스트 1파일인지 확인. 개수 가드 없는 스캐너는 영원히 green 이다(2026-10-02 ESCAPE 검사). grep 이 아니라 섭동 red 를 증거로 낼 것.
- [러너 01:22] scout done — OpenAPI 가 공표하는 **enum 질의 파라미터**를 `migrations/` 의 `CHECK (col IN (...))` 와 대조하는 불변식 테스트 (가�

## 구현 노트
- 무엇·왜: `internal/api/enum_contract_test.go`(신규, 프로덕션 0줄) 가 `api.OpenAPI()` 가 실제로 내보내는 문서의 enum 질의 파라미터를 `migrations.Files` 의 `CHECK (col IN (...))` 와 집합 대조한다. 두 손 목록을 묶는 것이 없어 한쪽만 고치면 전부 green 인 채로 문서가 거짓말을 했다. 커밋 37fd5e4.
- **과제서에서 정제한 한 곳(비평가가 먼저 볼 자리)**: 과제서의 "컬럼명 최신 승자" + "집합 다르면 모호" 는 `provider`(010 5값→014 6값)·`event_type`(008 9값→016 10값)에서 자기모순이다. SQL 을 열어 판정 기준이 집합 차이가 아니라 **표**임을 확인하고(014·016 은 같은 표에 `DROP`+`ADD CONSTRAINT` 하는 진짜 교체; `resource`·`severity`·`status` 는 다른 표의 다른 컬럼) 제약을 `(표, 컬럼)` 으로 식별했다. CHECK 42개 → 유효 40쌍(교체 정확히 2건), 모호 컬럼 정확히 `resource`·`severity`·`status`.
- 확신 없는 곳: 표 귀속은 CHECK 위치 **앞의 가장 가까운** `CREATE/ALTER TABLE` 로 정한다 — 오늘 42자리 전부 올바르지만(출력으로 표 이름을 눈으로 확인) 한 문장 안에 표가 둘 섞이는 SQL 이 들어오면 틀릴 수 있다. 값 추출은 `IN (` 뒤 괄호 균형으로 끊으므로 값 안에 `'` 나 `(` 가 들어간 리터럴은 미검증(오늘 그런 값 없음). `enum` 이 `[]string` 이 아니게 되면 조용히 0건이 아니라 `t.Fatalf` 로 터지게 해 뒀다.
- 일부러 하지 않은 것: `status`·`severity` 대조(집합이 여럿이라 operation→표 손 매핑이 필요 — 다음 회차 후보로 ideas.json 에 적었다) · `parameters.go`·`migrations/*.sql` 수정(오늘 6개 전부 일치, 고칠 것이 없다) · `make test`(과제서 지시; Go 테스트 1파일이고 `npm ci` 가 네트워크를 탄다 — 프런트는 미검증).
- 다음 역할 주의: DB 불필요, 순수 소스·embed 불변식이다 — **실제 PostgreSQL 이 그 값을 받는지는 증명하지 않는다.** `migrations/` 나 `parameters.go` 의 enum 을 건드리면 이 테스트가 짚어 준다. 개수 가드 2개(`knownEnumQueries = 26`·`knownCheckConstraints = 42`)는 정당하게 늘어날 때 함께 올려야 한다. 최종 `git diff --stat` 빈 출력, 신규 1파일만.
- [러너 01:29] brief accepted — 채택 — 과제서의 실측이 거의 전부 코드와 일치했다: CHECK **42개/12파일**과 컬럼별 분포(`status` 11·`severity` 3·`voice_type`·`r
- [러너 01:29] verify passed — 검증 10개 통과 (auto)
