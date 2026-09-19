# 회차 노트 2026-09-20-042349-vibe-coders-improve — vibe-coders
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:23] base pinned — master@d67267b
- [러너 04:23] autonomy release — 

## 정찰 노트
- 선택 이유: `executeReadOnlyQuery` timeout 은 Go 전용·한 함수·시그니처 불변이라 러너의 web verify 결함(`pnpm run typecheck --silent`, 수정 미병합)에 걸리지 않고, "같은 설정을 세 드라이버가 다르게 읽는" 운영자 규칙 위반을 고친다. textarea·캡처 문서 후보는 web/Playwright 의존이라 뒤로 밀었고, mail·MCP SSO·가격 prefix 후속은 master 미병합이라 재평가 불가.
- 확인한 것: 진짜 modernc sqlite 로 프로덕션 함수에 200ms 데드라인을 주면 200.6ms 에 `context deadline exceeded` 로 끊김(프로브 테스트 후 삭제, 작업 트리 깨끗). 즉 데드라인 방식이 sqlite 에서 동작한다.
- 확신 없는 곳: MySQL 실서버가 없어 go-sql-driver 의 ctx 취소 동작은 문서 지식(연결 종료)에 의존; `docs/ADMIN_GUIDE.md` 의 `TEXT2SQL_STATEMENT_TIMEOUT` 설명은 grep 하지 못함(과제서에 "미확인" 표기).
- 구현자 주의: pgx 의 `SET LOCAL statement_timeout` 을 지우지 말 것, 시그니처 바꾸지 말 것(호출처 4곳), 30초 상한을 넓히지 말 것. Go 전용이면 web verify 가 도는지 러너 로그를 보고, 필요시 964d2b0 cherry-pick 은 09-18 회차 선례가 있다.
- [러너 04:27] scout done — `executeReadOnlyQuery` 의 statement timeout 을 pgx 뿐 아니라 MySQL·SQLite(기본) 경로에도 적용 (가치 3 / 위험 2 / 작업�

## 구현 노트
- 바꾼 것: `executeReadOnlyQuery` 의 컨텍스트 데드라인을 드라이버 분기 앞에서 `execQueryDeadline(stmtTimeout)` = min(30s, stmtTimeout>0) 으로 잡아 pgx·mysql·sqlite 세 경로가 `TEXT2SQL_STATEMENT_TIMEOUT` 을 같게 읽게 함(72e93dc). pgx 의 `SET LOCAL statement_timeout`·`work_mem`·시그니처·30초 상한은 그대로. `config.go:333` 주석만 함께 고침. 러너 verify 가 `pnpm run typecheck --silent` 를 만들므로 09-18 선례대로 964d2b0 을 cherry-pick(6bbc8fc) — 과제 밖이지만 없으면 verify-failed.
- 확신 없는 곳: **MySQL·PostgreSQL 실서버 미확인** — 데드라인이 분기 앞에 있어 코드는 공통이지만 go-sql-driver/mysql 이 ctx 만료 시 연결을 끊는다는 것은 문서 지식. pgx 는 서버 측 statement_timeout 과 데드라인이 같은 값이라 어느 쪽이 먼저 걸려도 오류 종류(`context deadline exceeded` vs pg `57014`)만 다르고 호출처가 그 차이를 분류하는지는 보지 않았다.
- 검증: 새 테스트 3개(진짜 modernc sqlite, driver "" 와 "sqlite"), 수정 전 (a) 가 30.01s 대기 후 실패함을 확인하고 수정 후 0.20s. gofmt·vet·`go test ./...` 12 패키지·race·api-surface-audit(FAIL 전부 빈 배열)·러너 동일 `pnpm run typecheck --silent`/`build --silent`/`test --silent`(908) 전부 exit 0.
- 일부러 안 한 것: MySQL `SET SESSION MAX_EXECUTION_TIME`(풀 반환 연결에 세션 변수가 남음), `docs/ADMIN_GUIDE.md` 157행("실행 제한" 이라고만 적혀 있어 틀린 말이 없음), 같은 함수의 work_mem 값 검증(diff 섞임 방지, ideas 에 남김), 30초 상한 확장.
- 다음 역할 주의: 느린 쿼리 테스트는 `x < 200000000` 재귀 CTE 라 데드라인이 빠지면 30초씩 두 번 기다린 뒤 실패한다(5초 상한 assert). sqlite 는 데드라인 오류를 `context.DeadlineExceeded` 또는 `interrupted` 로 낼 수 있어 둘 다 허용. `web/node_modules`·`web/dist` 는 이 회차가 로컬 검증으로 만들었고 gitignore 대상.
- [러너 04:36] brief accepted — 채택 — 과제서의 근거(pgx 만 timeout 적용, 나머지는 30s 고정)가 코드와 정확히 일치했고 수용 기준 4개를 모두 충족했다.
- [러너 04:39] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: 슬로우 쿼리 테스트를 d67267b 의 handler 로 되돌려 돌리면 30.01s 후 FAIL, 수정 후 0.20s PASS — 테스트가 변경을 실제로 잡는다. go build/vet/gofmt, web lint·format:check·run-tool 테스트·`pnpm run typecheck --silent` 전부 통과. 판정 approve, risk low, blocking 없음.
- 못 본 것: MySQL·PostgreSQL 실서버 없음. MySQL 은 클라이언트 데드라인만 걸리고(go-sql-driver 가 연결을 닫음) 서버 측 쿼리는 소켓 종료 감지까지 계속 돌 수 있다 — docstring 의 "bounded the same way as PostgreSQL" 은 그만큼 과장.
- 릴리즈 노트: 기본 15s 이므로 MySQL·SQLite 소스에서 15~30s 걸리던 execute·스케줄 리포트가 이제 timeout 으로 실패한다(의도된 동작 변경). pgx 는 오류 종류가 57014 → context deadline exceeded 로 바뀌지만 classifyText2SQLFailure 는 둘 다 timeout 으로 분류.
- 다음 회차: README.md:855·:892 가 여전히 "(postgres)" 로 적혀 있어 한 줄 수정 필요. 6bbc8fc(run-tool.mjs) 는 범위 밖 cherry-pick 이지만 09-18 선례와 동일하고 검증 통과.
- [러너 04:42] review approved — 리뷰 승인 (risk=low)
- [러너 04:43] pr created — https://github.com/hkjang/vibe-coders/pull/21
- [러너 05:03] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
