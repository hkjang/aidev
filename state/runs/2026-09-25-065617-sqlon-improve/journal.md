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

## 구현 노트
- `POST /api/query/submit` 이 `binds` 를 받아 `ExecOptions.Binds` 로 넘기게 고쳤다(13acd4e). 관측점은 `asyncJob` 의 비공개 `opts` 필드이고, 백그라운드 실행이 그 필드를 그대로 `executeGuarded` 에 넘기므로 테스트가 보는 값 = 실제 실행 계약이다. 잡 조회 JSON 응답은 늘지 않았다(`TestAsyncSubmitWithoutBindsStaysNil` 이 binds/opts/exec_options 키 부재를 단언).
- 확신 없는 곳: (1) **결과 캐시 키가 여전히 binds 를 무시한다**(execguard.go `cacheKey(profile, sql, maxRows)`). 이번 변경으로 비동기 결과도 binds 에 따라 달라지므로, binds 를 무시한 키로 `put` 된 비동기 결과를 다른 binds 의 동기 요청이 읽을 수 있다. 과제서가 재구현을 금지해 손대지 않았으나 **이 브랜치에는 57563cf 가 없어 결함이 살아 있다** — 비평가/러너가 먼저 볼 곳. (2) 웹 UI 의 비동기 제출 폼이 binds 를 보내는지 확인하지 않았다(서버가 생략을 허용하므로 호환은 됨).
- 일부러 하지 않은 것: 캐시 키 수정, 동기/비동기 요청 구조체 통합(최소 diff 유지), `requireQueryActor`·`canUseProfileID`·`ValidateSQL` 게이트 순서, 감사 details 문자열(binds·SQL 원문 미포함 유지).
- 검증 공백: 동기 핸들러가 만드는 `ExecOptions` 를 런타임으로 직접 들여다보지는 못했다(Server 에 테스트용 훅을 넣지 않으려고 포기). 대신 두 엔드포인트가 **같은 JSON 본문**을 똑같이 받아들이는 것을 실제 HTTP 로 확인했다(`TestSyncAndAsyncAcceptTheSameQueryBody`).
- 다음 역할이 조심할 것: `test/integration/integration_test.go` 의 `TestExecuteBindsReachDriver` 는 `-tags integration` 이고 **도커 컨테이너 3종이 떠 있어야 돈다**(`docker compose -f deploy/test/docker-compose.yml up -d --wait`). 이번 회차엔 띄워서 통과시킨 뒤 `down -v` 했다.
- CHANGELOG 는 편집 도구가 파일 전체 줄바꿈을 뒤집어(97줄 diff) 되돌린 뒤 바이트 단위로 다시 넣었다. 이 파일은 LF/CRLF 혼합이니 다음 역할도 Edit/Write 로 만지지 말 것 — `git diff --numstat` 으로 5줄만 바뀐 것을 확인했다.
- [러너 07:08] brief accepted — 채택 — 지정한 결함을 현 HEAD 에서 실제 HTTP 로 재현했고 지정 파일만 최소 수정했다. 다만 동기 경로의 라우트는 과제서
- [러너 07:08] verify passed — 검증 3개 통과 (auto)
- [러너 07:08] pr created — https://github.com/hkjang/sqlon/pull/13
- [러너 07:08] merge stopped — 긴급 중지
