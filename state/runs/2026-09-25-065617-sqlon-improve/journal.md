# 회차 노트 2026-09-25-065617-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:56] base pinned — main@57f99b7
- [러너 06:56] autonomy release — 

## 정찰 노트
- 비동기 제출 binds 누락을 골랐다: 동기 경로(dbapi.go:203/235 → manager.go:397)와 나란히 놓고 소스로 확인한 실제 계약 불일치이고 S 규모다. 문서 후보(docs/README.md, admin_guide curl)는 가치는 같지만 결함이 아니라 뒤로 뺐고, 캐시 키·잡 TTL·visibility·actor 계열은 모두 과거 성공(미통합)이라 재구현 금지 규칙으로 제외했다.
- 확신 없는 곳: 바인드가 드라이버까지 도달하는 것을 일반 go test 로 끝까지 볼 수 없다(테스트용 sql 드라이버 미등록, pgx/mysql 이름 고정). 그래서 과제서의 관측점을 asyncJob 의 비공개 opts 필드로 제안했는데, 이건 내가 제안한 설계이지 기존 코드에 있는 것이 아니다.
- 구현자가 조심할 것: 현 HEAD 의 cacheKey 는 binds 를 무시하고 비동기는 fresh=true 로 읽기만 건너뛰고 put 은 한다 — 캐시가 RED/GREEN 판정을 흐릴 수 있으니 관측은 ExecOptions 지점에서 하고, 캐시 키 자체는 건드리지 말 것.
- CHANGELOG.md 는 LF/CRLF 혼합이라 편집 후 diff 범위를 반드시 확인할 것. 기준선은 확인함: HEAD 57f99b7 에서 `go test ./internal/mcp -count=1` → ok 3.642s.
- 프로필은 3일 전 것이 현 HEAD(57f99b7, 무변경)와 일치해 새로 쓰지 않았다. 요청된 pmo/technology 스킬 3개는 이번 회차에 정상 로드되어 그 절차(문제 재정의→대안 비교→하나로 수렴, 단계마다 증명 명령과 범위 제외, 근거·범위 명시)를 과제서에 반영했다.
- [러너 07:00] scout done — 비동기 쿼리 제출 `POST /api/query/submit` 이 `binds` 를 받지 않아 동기 경로와 실행 계약이 어긋나는 결함 수정 
