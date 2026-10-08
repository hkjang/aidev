- 과제: MCP 매핑 실패·비활성 매핑의 도구 거부와 반복 오류 집계를 HTTP 통합 테스트로 고정 (가치 4 / 위험 1 / 작업량 S)
- 왜: 기존 TestIdentityMappingBindsExactUsernameAndIsStable은 Mapper 직접 호출로 최초 매핑·재바인딩 거부를 확인하지만, auth.identityFor가 매핑 오류를 삼킨 뒤 실제 MCP 호출을 차단하는 배선과 identity_mapping_errors의 반복 집계는 검증하지 않는다. 기존 서버 하네스에서 HTTP 응답과 PostgreSQL 행을 함께 단정하면 매핑 실패가 권한 우회로 이어지거나 관리용 오류 기록이 누락·중복되는 회귀를 잡는다.
- 수용 기준:
  1) 새 gateway에서 preferred_username="ghost"인 유효한 서명 토큰으로 bitbucket_get_repository(project="AI", repository="text2sql")를 두 번 호출한다. 매번 HTTP 200, JSON-RPC result.isError=true, result.structuredContent.code="IDENTITY_UNMAPPED"이고 해당 sub의 매핑 행은 0개다. 오류 테이블은 해당 sub 기준 정확히 1행이며 occurrences는 1→2, id와 first_seen_at은 불변, occurred_at은 감소하지 않고 username="ghost"와 비어 있지 않은 reason을 보존한다. 이 두 요청 외에 initialize/login/bitbucket_me를 끼워 넣지 않는다(요청마다 매핑을 재시도해 횟수가 달라진다).
  2) 별도 gateway에서 기본 토큰으로 bitbucket_me를 호출해 응답 bitbucketUserId=142, bitbucketUsername="hkjang"와 실제 매핑 행을 확인한다. 그 행의 active만 false로 설정한 뒤 같은 토큰으로 bitbucket_me를 다시 호출하면 ID=0·username=""이며, 권한 필요 도구 bitbucket_get_repository는 위와 동일한 IDENTITY_UNMAPPED 오류로 거부된다. 저장된 원래 매핑 ID/Bitbucket ID는 유지되고 active=false다. 비활성 거부에는 recordError 호출이 없으므로 새 오류 행을 요구하지 않는다.
  3) 실제 api.New(...).Router()를 띄우는 newGateway와 keycloakStub.token, gateway.post/pool을 재사용해 위 결과를 증명한다. Principal/Mapper를 대역으로 교체하거나 소스 문자열로 검사하지 않는다. 신규 테스트 2개와 기존 전체 테스트가 실제 PostgreSQL에서 Skip 없이 통과한다. 현재 버그 재현 과제가 아니라 보안 계약의 테스트 공백 보강이며, 수정 전 실패를 만들기 위해 제품 동작을 바꾸지 않는다.
- 건드릴 파일: internal/api/mcpoauth_e2e_test.go — 신규 TestMCPIdentityMappingFailureIsDeniedAndDeduplicated, TestMCPIdentityInactiveMappingDeniesPermissionedTool 추가; 필요하면 같은 파일 안에 응답/행 단정용 작은 helper만 둔다. newGateway/newKeycloakStub/keycloakStub.token/gateway.post는 기존 그대로 재사용 가능하고 Bitbucket 스텁 확장은 필요 없다. 프로덕션 변경 0개, 테스트 변경 1개.
- 검증 명령:
  - 정찰에서 실행한 기준선: `TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable' go test ./... -count=1 -p 1 -v` — 종료 0, PASS 123건, SKIP 0건, FAIL 0건, 9개 패키지 통과. 실행 로그는 같은 회차 디렉터리의 baseline-tests.log.
  - 구현 뒤 신규 테스트: `TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable' go test ./internal/api -run '^TestMCPIdentity(MappingFailureIsDeniedAndDeduplicated|InactiveMappingDeniesPermissionedTool)$' -count=1 -v`.
  - 이후 위 전체 기준선 명령을 다시 실행하고 `go build ./...`, `go vet ./...` 실행. 마지막 두 명령은 CI에 있는 명령이며 정찰에서는 별도 실행하지 않았다. 전용 DB 컨테이너 bbmcp-test-pg가 현재 기동 중이다. 다른 용도의 DB를 쓰지 말 것.
- 위험과 피할 것: internal/auth/*, internal/api/oauth.go, migrations, workflows 및 프로덕션 매핑 정책 변경은 범위 밖. 이 과제에서 preferred_username 변경 후 OAuth 재로그인 시나리오를 추가하지 말 것: Users.UpsertFromKeycloak이 username 충돌만 처리하고 users.keycloak_sub는 UNIQUE라, 같은 sub의 개명은 별도 auth 문제로 추정된다(실제 HTTP 재현 미확인). 기존 최초 매핑·재바인딩 테스트를 복제하지 않는다. t.Parallel 금지, 전체 실행은 -p 1 필수. 시간 비교에는 sleep·정확한 시각 대신 불변/비감소를 쓰고 SQL 오류·RowsAffected·타입 단정 실패를 반드시 검사한다. 토큰·검색 원문을 새 감사 details에 기록하지 않는다. 권한 실패는 HTTP 401이 아니라 HTTP 200의 MCP 도구 오류이고 bitbucket_me는 RequiredPerm=NONE이므로 성공하는 것이 계약이다.
- 차선 후보: permission resolver의 Reset 이후 캐시 무효화 회귀 테스트 (가치 4 / 위험 2 / 작업량 M) — 1순위가 이미 다른 테스트로 충족됨을 확인한 경우만 선택. internal/tools/integration_test.go:newFixture의 CacheTTLSec=0을 테스트별로 양수로 설정하고 기존 fakeBitbucket의 권한 응답을 바꾸어 캐시 유지→Reset→새 권한 적용을 증명하는 1개 경로로 한정한다. TTL 만료 대기·플러그인·fail-closed까지 묶지 말고 실제 fixture 가용 필드는 다시 확인할 것. 검증은 같은 DSN으로 `go test ./internal/tools -count=1 -v` 및 전체 `-p 1`.

근거와 구현 순서 (모두 미착수, 사람 승인 체크포인트 없음)
1. 기존 TestMCPOAuthDiscoveryAndCall의 bitbucket_me 응답 처리와 newGateway의 TRUNCATE를 따라 첫 번째 신규 테스트를 작성한다. 증명: 위 신규 테스트 명령에서 첫 테스트가 실행되고 PASS하며 Skip이 없다. 현재 계약과 다르면 과제서 전제를 수정·기록하고 auth 수정으로 확장하지 않는다.
2. 두 번째 신규 테스트에서 정상 HTTP 호출로 만든 매핑의 active만 DB로 설정한다(실제 저장 상태 준비이며 Mapper 대역 주입 아님). 증명: 같은 신규 테스트 명령에서 두 테스트 PASS. 요청과 DB 조회를 순차 실행한다.
3. 전체 테스트·build·vet를 실행한다. 증명: 종료 코드 0과 전체 테스트 Skip 0; diff가 위 테스트 파일 하나뿐인지 확인한다. 실제 수행한 명령과 제한을 회차 노트에 기록한다.

실제로 확인한 코드
- internal/identity/mapper.go:Resolve는 비활성 매핑을 거부; autoMap은 검색 실패를 recordError로 저장; recordError는 sub 충돌 시 occurrences만 누적하고 first_seen_at을 유지한다.
- internal/auth/service.go:identityFor는 Mapper.Resolve 실패 시 Mapping 없이 Identity를 반환; Principal은 매핑이 있을 때만 Bitbucket ID/사용자명을 설정한다.
- internal/tools/executor.go:Invoke는 RequiredPerm != NONE인 도구에서 BitbucketUsername이 비면 CodeIdentityMissing으로 반환한다.
- internal/mcp/server.go:callTool은 도구 오류를 CallResult.IsError/StructuredContent로 내보낸다. internal/tools/read.go:readTools의 bitbucket_me는 미매핑 상태에서도 실행 가능하다.
- internal/database/migrations/0003_mapping_errors_dedupe.sql은 sub별 유일 인덱스와 occurrences/first_seen_at을 정의한다(읽기 근거, 수정 금지).

대안 비교와 추정 근거
- 기존 API 하네스 확장(선택): 테스트 한 파일, 실제 요청과 저장 결과를 함께 검증하고 프로덕션 변경이 없다.
- Mapper 직접 단위/통합 검사: 더 작지만 identityFor→Principal→Invoke의 결합을 놓치고 기존 테스트와 겹쳐 제외.
- 인증/매핑 구조 정비 또는 개명 버그 수정: 가치가 있으나 최근 로그인 수정이 반복된 보호 경로라 이번 45분 과제에서 제외. 현상 재현부터 별도 회차로 진행한다.
- 아무것도 하지 않음: 현재 기준선은 통과하지만 위 회귀를 잡는 증거가 남지 않아 선택하지 않았다.
- 상향식 노력 추정(실측이 아닌 정찰 판단): 오류 집계 테스트 8–12분 + 비활성 매핑 테스트 7–10분 + 전체 검증/정리 5–8분 = 기본 20–30분. SQL/HTTP 단정 조정이라는 알려진 불확실성에 별도 contingency 5분을 두어 25–35분, 신뢰도 중간(통계적 확률 아님). management reserve는 0분이며 새 제품 결함은 범위에 편입하지 않는다. 45분 안에 가능하다는 핵심 가정은 전용 DB와 기존 하네스가 그대로 동작한다는 것이며 기준선 실행으로 확인했다. 과거 회차는 같은 하네스로 테스트 1–2개를 추가했지만 실제 소요시간이 없어 수치적 유사추정은 하지 않았다.
- 적용한 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md를 읽었다(현재 도구 목록에 Skill 호출 도구 없음). 외부 원가·편익 수치를 사용하지 않았으며 노력 범위는 위 코드 범위에 대한 정찰 판단이다.
