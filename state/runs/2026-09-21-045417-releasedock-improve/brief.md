- 과제: make test에서 PostgreSQL 통합 테스트 생략을 명확히 알리기 (가치 2 / 위험 1 / 작업량 S)
- 왜: 현재 Makefile의 test는 TEST_POSTGRES_DSN이 비어 있어도 안내 없이 Go 테스트를 실행하고, DB fixture는 t.Skip으로 빠져 일반 출력에 패키지 ok만 남습니다. 시작 시 한 번 경고하고 준비 방법을 문서화하면 개발자가 DB 통합 검증까지 통과한 것으로 오해하는 일을 줄일 수 있습니다.
- 수용 기준: 1) 저장소 루트에서 TEST_POSTGRES_DSN 미설정·빈 문자열·공백만 있는 값으로 make test를 실행하면 첫 Go 테스트 전에 WARN과 TEST_POSTGRES_DSN, PostgreSQL 통합 테스트가 생략된다는 설명이 정확히 한 번 출력되며 기존 테스트는 계속 실행된다. 2) 유효한 테스트 DB DSN이 있으면 생략 경고 없이 실제 통합 테스트가 실행되고, 잘못된 비어 있지 않은 DSN이면 기존 Go 테스트 실패가 make의 비영 종료 코드로 전달된다; 안내는 DSN 실제 값을 출력하지 않는다. 3) README 개발 절에 npm ci 선행 조건, DB 미설정 시 검증 범위, TEST_POSTGRES_DSN을 설정한 make test 방법을 적고, 검증은 실제 make/go/npm 프로세스의 출력과 종료 코드로 증명한다(가짜 실행 파일·소스 문자열 검사로 대체하지 않는다).
- 건드릴 파일: Makefile:test — 기존 세 명령 앞에 환경변수의 빈 값/공백을 검사하는 경고 recipe 한 개 추가; README.md:개발 — npm 의존성 설치와 선택적 DB 통합 테스트 준비·경고 의미 설명. 아래 테스트 파일들은 근거 확인용이며 수정하지 않는다: backend/internal/server/server_test.go:newRollbackRetryFixture, backend/internal/server/simple_batch_test.go:newSimpleBatchFixture, backend/internal/store/store_integration_test.go:TestFreshDatabaseMigrationsAndBootstrap, runner/internal/store/store_integration_test.go:TestFinishJobMaintainsDeploymentHeadChain.
- 검증 명령: 루트에서 `(cd web && npm ci)` 후 `env -u TEST_POSTGRES_DSN make test`, `TEST_POSTGRES_DSN='' make test`, `TEST_POSTGRES_DSN='   ' make test`; 준비된 테스트 전용 PostgreSQL DSN을 환경에 설정하고 `make test`, `(cd backend && go test ./internal/store -run '^TestFreshDatabaseMigrationsAndBootstrap$' -count=1 -v)`, `(cd runner && go test ./internal/store -run '^TestFinishJobMaintainsDeploymentHeadChain$' -count=1 -v)`로 PASS/SKIP 확인. 잘못된 DSN 경로는 `TEST_POSTGRES_DSN='postgres://invalid@127.0.0.1:1/postgres?sslmode=disable&connect_timeout=1' make test`로 경고 없음과 비영 종료 확인(의도된 실패). Go 캐시 때문에 설정된 DSN 검증을 생략했다고 오해하지 않도록 위 두 DB 테스트에는 -count=1을 유지한다.
- 위험과 피할 것: auth/session, migrations, .github/workflows, VERSION, package-lock, 로그 예산·SSE 동작은 범위 밖이다. main에는 make vet가 없으므로 미머지 정적 검사 PR을 재구현하거나 그 타깃에 의존하지 않는다. 환경변수는 shell에서 따옴표로 읽고 Makefile의 $$ 이스케이프를 사용한다; DSN을 Make 확장으로 recipe에 삽입하거나 실제 값을 로그에 출력하지 않는다. 미설정 경고로 실패시키거나 DB를 자동 기동하지 않는다. 기존 go/npm 실패를 무시하는 `|| true`나 파이프의 종료 코드 손실을 넣지 않는다. 실제 DB 검증은 테스트 전용 DB에서만 한다.
- 차선 후보: simple SSE에서 500행을 꽉 채운 페이지 뒤에는 즉시 다음 페이지 읽기 — backend/internal/server/simple.go:streamSimpleRunLogs의 LIMIT 500 이후 항상 1초 대기를 줄이는 작업(가치 2 / 위험 2 / M). 경고 과제가 이미 구현돼 있거나 실효성이 없다는 코드 근거가 생긴 경우에만 전환한다. 실제 PostgreSQL + HTTP 핸들러 스트림으로 500행 초과 순서·누락 없음·종료 end를 검증하고, 빈 페이지의 폴링 및 취소/30분 제한을 보존해야 한다; 대역 타이머나 무조건 continue로 검증을 대체하지 않는다.

범위·선택 근거 및 구현 순서 (스킬 적용)

- 사용한 스킬: `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. 전용 Skill 도구가 제공되지 않아 로컬 원문으로 읽었다. estimating의 references/sources.md도 확인했다. 아래 시간은 이 저장소를 읽은 정찰자의 공학적 추정이며 통계적 신뢰수준이나 외부 원가 기준을 주장하지 않는다.
- 대안 비교: (A, 선택) make test 경고 + 짧은 README는 기존 선택적 DB 테스트 계약을 지키며 코드 두 곳만 바꾼다. (B) DB 필수 타깃/자동 컨테이너 기동은 강한 보장이 있지만 실행 환경과 유지보수 범위가 늘어난다. (C) README만 고치거나 아무것도 하지 않으면 실행 출력의 오해가 남는다. SSE 성능 후보는 사용자 효과가 있으나 실제 연결·타이밍 통합 검증이 이번 변경보다 크다.
- 가장 중요한 가정: DB 없는 로컬 테스트도 허용한다는 현재 계약을 유지한다. fixture 네 곳이 모두 TrimSpace 후 Skip하며 CI는 별도로 PostgreSQL 서비스를 준비한다는 코드에 근거한다.
- [pending] 1. Makefile:test에 경고 recipe와 README 개발 절을 함께 수정한다. 증명: `env -u TEST_POSTGRES_DSN make test`로 경고가 첫 명령보다 먼저 나오고 기존 테스트 순서를 유지하는지 확인. 체크포인트: 구현자 자체 검토, 사람 승인 불필요.
- [pending] 2. 위 검증 명령으로 빈 값/공백·유효 DB·잘못된 DB 분기를 실행하고 실제 출력과 종료 코드를 기록한다. 증명: 유효 DB의 두 명시적 -v 테스트가 SKIP 없이 PASS. 체크포인트: 조건 누락이나 환경 문제면 과제서 상태에 미확인을 적고 수정한 계획대로 진행; 성공으로 표시하지 않는다.
- [pending] 3. `git diff --check` 및 `git diff --stat`으로 Makefile/README만 변경됐는지 확인하고 결과를 다음 역할에 인계한다. 체크포인트: 이후 비평 역할이 별도로 검토한다.
- 작업량 근거(bottom-up): 구현·문서 5–8분, 실제 명령 검증 8–15분, diff/인계 3–5분 = 기본 16–28분. 알려진 변동(의존성 설치·테스트 DB 준비)에만 예비 5–10분을 더해 총 21–38분으로 추정한다. 이전 회차의 make 정적 검사 변경과 규모를 비교하면 더 좁은 2파일 변경이나 과거 실제 소요 시간은 미확인이라 수치 보정에는 쓰지 않았다. 별도 관리 예비는 배정하지 않았으며 DB/도구 준비가 10분 이상 막히면 검증 미완료로 기록하고 범위를 확장하지 않는다.

정찰의 실제 확인 결과

- 기준: main@4990505, VERSION 0.5.17, 시작·마지막 확인 시 작업 트리 변경 없음. CLAUDE.md/AGENTS.md/별도 로드맵 및 TODO/FIXME는 저장소 검색에서 발견하지 못했다. README, docs 네 문서, 최근 git log -30, CI/release, Makefile, 웹 테스트 설정 및 위 함수들을 읽었다.
- `make test`: backend와 runner go test ./...는 성공했으나 웹에서 `vitest: not found`로 make 종료 2. web/node_modules 미설치 상태이며 정찰은 저장소 쓰기 금지 때문에 npm ci를 실행하지 않았다. 전체 테스트 성공으로 해석하지 말 것.
- `(cd backend && env -u TEST_POSTGRES_DSN go test ./internal/store -run '^TestFreshDatabaseMigrationsAndBootstrap$' -count=1 -v)`: 실제 SKIP 메시지 후 PASS/ok 확인. 유효 PostgreSQL을 연결한 검증은 정찰에서 미실행.
- Go 1.26.7은 이 환경에서 정상 실행됐다. 이전 프로필의 Go 실행 권한 문제는 현재에는 해당하지 않는다.
