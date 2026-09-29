# 검증 기록

- 기준: eac2424, Qurio v1.4.8. 오프라인 AI SQL 워크벤치; Go/pgx/PostgreSQL + React/TypeScript. 최근 30개 커밋, README, architecture/requirements, Makefile, CI/release 검증 절차를 확인했다. 저장소/상위 경로의 AGENTS.md 및 저장소 CLAUDE.md는 없었다.
- Skill 전용 도구는 제공되지 않았다. 대신 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/` 아래 completion-verification, systematic-debugging, test-driven-development의 SKILL.md를 직접 읽고 적용했다.
- 후보: ideas.json에 기존 12개 보존·현황 갱신, 신규 2개 추가. WITH ORDINALITY 오탐 선택(2/2/S); 허용 정책 완화라 기존 위험 1을 2로 조정. 다른 반려 접근, 릴리즈·빌드 설정, 인증 변경은 선택하지 않았다.

## 재현과 원인

`SELECT * FROM unnest(ARRAY[11,22]) WITH ORDINALITY g(value,position)`은 정상적인 PostgreSQL SQL이지만, alias 판별기가 직전 `ORDINALITY`를 알아보지 못해 g()를 비허용 함수로 분류한다. AS를 추가한 대조군은 수정 전부터 통과한다. 파서 계약을 통합하거나 함수 허용목록을 넓히지 않고 `) WITH ORDINALITY` 접미사와 기존 FROM 문맥을 함께 요구하는 분기 8줄만 추가했다.

- 먼저 `go test ./internal/domain/sqlsafe -run Ordinality -count=1 -v`: 정상 8개 중 7개 FAIL, AS 대조군 PASS. 차단 유지 8개 PASS. red-unit.txt.
- 실제 폐기 PostgreSQL 17(127.0.0.1:55482)에 기존 New/Upsert/Validate/Execute를 연결한 `go test -tags=integration ./internal/domain/dbexec -run '^TestPostgresOrdinalityAliasValidationAndPaging$' -count=1 -v`: 6개 중 AS 대조군만 PASS, 나머지 5개는 Validate와 Execute 모두 g() 오탐 FAIL. red-integration.txt.
- 수정 후 두 패키지 `go test -race -tags=integration ... -run Ordinality -count=1 -v`: 모두 PASS. 실데이터 두 번째 페이지가 value=22, position=2이며 컬럼명 및 HasNext도 확인. green-targeted.txt.
- 프로덕션 파일만 git HEAD 버전으로 되돌려 동일 회귀 테스트 실행: 두 패키지 다시 FAIL. 이후 수정본을 그대로 복원. reverted-red.txt.
- 기존/신규 SQL 142개 × PostgreSQL/Oracle = 284개 Analysis를 수정 전후 실제 실행 비교: 정상 ordinality 7개만 readOnly false→true; 새 오탐 0개, Oracle 변경 0개. 위험 함수 포함 4개는 허위 g() 이유만 사라지고 계속 blocked. corpus-base.json/corpus-fixed.json/corpus-diff.json. 임시 probe 테스트는 제거했다.

## 전체 검증

- `go test ./...`: exit 0, 28개 패키지 ok. unit.txt.
- `make lint && go build ./... && go vet -tags=integration ./...`: exit 0. lint-build.txt.
- 기존 TestIntegrationDatabaseMigrations로 bootstrap 후 CI와 동일하게 세 DSN을 위 폐기 DB로 지정, `go test -race -p=1 -tags=integration ./... -count=1 -json`: exit 0, 30개 패키지 PASS, 테스트/하위 테스트 이벤트 787 PASS. integration.jsonl.
- 통합 테스트의 실제 Skip: 전용 DSN 미설정 시 Skip을 검증하는 의도적 하위 테스트 3개, Oracle 라이브 테스트 1개(Oracle 서비스 미구동). 나머지 패키지 Skip은 test files 없음이다.
- `npm --prefix web test -- --run && npm --prefix web run build`: exit 0; 27 files / 134 tests PASS, Vite build PASS. web.txt.
- `npm --prefix web audit --omit=dev --audit-level=high`: exit 0, found 0 vulnerabilities. npm ci의 개발 의존성 포함 moderate 경고는 의존성을 수정하지 않고 남겼다.
- `make check-go-format`, `git diff --check`: exit 0.
- 브라우저 E2E는 최초 임시 홈에 실행 파일이 없어 14개가 실행 전 실패했다(e2e.txt). `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright npm --prefix web run test:e2e -- --workers=2`로 기존 동일 버전 Chromium 경로를 지정해 재실행: exit 0, 14 passed (2.3m). e2e-with-browser.txt.

## 범위와 미검증

프로덕션 1개 파일(+8줄), 테스트 2개 파일. 릴리즈·빌드 경로, 함수 허용목록, Oracle 분석 경로는 변경하지 않았다. Oracle 실서비스·오프라인 이미지 릴리즈·govulncheck/gosec는 실행하지 않았다. 이번 변경은 PostgreSQL 구문 오탐이며 실제 PostgreSQL 실행을 검증했고, 릴리즈는 별도 러너 역할이다. AS 분기 의심, 일반 비한정 테이블 별칭, 다수 CTE 성능은 별도 후보로 남겼다. 폐기 컨테이너는 제거했다.

최종 커밋: 7cda7f5 (hkjang, 트레일러 없음). 생성 web/dist·web/test-results 제거 완료.
