- 과제: AI 스트림에서 조회할 수 없는 사람만 SSE 시작 전 404로 반환 (가치 3 / 위험 2 / 작업량 M)
- 왜: `POST /api/v1/ai/stream`에 존재하지 않거나 다른 사용자 소유인 정상 UUID를 넣으면 `relationshipContext`의 첫 조회 오류를 `streamAI`가 500으로 감싼다. 사람 조회 실패만 404 JSON으로 구분하면 삭제된 사람을 선택한 화면·API 호출자가 요청 문제를 알 수 있고 서버 장애로 오인하지 않는다.
- 수용 기준:
  1) AI 활성 상태에서 없는 UUID와 타 사용자 소유 UUID는 동일한 `404`, `error.code=not_found`, `error.message=사람을 찾을 수 없습니다.` JSON을 반환한다. Content-Type은 application/json이고 SSE meta/delta/done 및 외부 제공자 요청은 없다.
  2) 자신의 정상 사람(relationships 행 포함, 기억 없음)과 빈 person_id는 기존 200 SSE meta/delta/done을 유지한다. 잘못된 UUID의 기존 400과 AI 비활성 상태의 기존 503도 유지한다.
  3) 사람은 있지만 기억의 복호화 키를 찾지 못한 경우는 기존 500 internal_error를 유지한다. 설정 조회 실패도 404로 바꾸지 않는다. 모든 pgx.ErrNoRows를 일괄 404로 바꾸면 안 된다.
  4) 실제 PostgreSQL·store.Open·실제 streamAI 핸들러로 1~3을 검증한다. 변경 전 없는 UUID/타인 UUID가 500이라 실패하고, 변경 후 404로 통과하는 테스트 로그를 남긴다. 소스 문자열 검사나 오류만 반환하는 가짜 저장소로 대체하지 않는다.
- 건드릴 파일:
  - `internal/server/ai.go:relationshipContext, streamAI` — 첫 people JOIN relationships QueryRow에서만 pgx.ErrNoRows를 AI 전용 비공개 sentinel(예: errAIRelationshipNotFound)로 변환한다. streamAI는 그 sentinel만 errors.Is로 검사해 기존 not_found 응답을 쓰고 나머지는 internalError 유지. errors 패키지는 이미 있으며 pgx import만 추가하면 된다. 프로덕션 1파일.
  - `internal/server/ai_db_test.go` (신규) — `TestStreamAIPersonLookup` 아래 실제 DB·핸들러 회귀 시험과 테스트 전용 설정 복원 헬퍼. 기존 ai_test.go의 malformed UUID 시험은 그대로 활용.
- 검증 명령:
  - 정찰에서 실행 완료: `go test ./...` 통과(대부분 캐시; DB DSN 없이 실행). 아래 새 DB 회귀 시험과 race/vet은 구현자가 실행해야 하며 정찰에서는 미실행.
  - `go test -count=1 -v ./internal/server -run 'TestStreamAI|TestExtractDelta'`
  - `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:<격리된포트>/orbit?sslmode=disable' go test -race -count=1 -v ./internal/server -run 'TestStreamAIPersonLookup|TestOrbit'` — <격리된포트>를 실제 포트로 치환. SKIP은 성공 증거가 아니다.
  - `gofmt -l internal/server` (출력 없어야 함), `go vet ./...`, `go test -race -count=1 ./...`.
- 위험과 피할 것: auth·secure·migrations·workflows·빌드·프런트 변경, UUID 정책 확대, SSE 파서 개편, 다른 REST/MCP 오류 계약 변경은 범위 밖. `relationshipContext → queryMemories → queryMemoriesLimit → dataKeyVersion`도 pgx.ErrNoRows를 올릴 수 있어 첫 조회에만 sentinel을 붙여야 한다. 관계 행 없는 사람도 JOIN 결과가 없으므로 동일 404로 취급한다. nil-store 시험은 첫 설정 조회에서 막히므로 이번 결함의 증거로 쓰지 않는다. 설정은 사용자별 데이터가 아닌 전역 행이므로 병렬 시험 금지·기존 행 백업/복원 필수. 공유/운영 DB 사용 금지.
- 차선 후보: `orbitAt` contexts 분류 집계 실제 postgres 회귀 테스트 (가치 2 / 위험 1 / 작업량 S) — 1순위 결함이 이미 해결돼 있다면 선택. timetravel_db_test.go의 seedRelationship 후 categories를 UPDATE하고 포함된 사람만 contexts에 기여하는지, 현재/과거 getOrbit 응답의 노드 categories와 집계가 일치하는지 확인. DB 환경 자체가 없으면 이 후보도 검증할 수 없으므로 성공 처리하지 않는다.

실행 계획 (모두 미착수, 사람 승인 체크포인트 없음; 각 단계의 시험 결과가 다음 단계 진입 기준)
1. `ai_db_test.go`에서 기존 `openTestStore`, `seedUser`, `seedPerson`, `seedRelationship`을 재사용한다. `httptest.NewRequest(POST, /api/v1/ai/stream, ...)`에 userContextKey의 실제 User{ID: seedUser 결과}를 넣고 `(&Server{store: st}).streamAI`를 호출한다. 기존 streamAIRequest는 ID가 u1로 고정이므로 그대로 쓰지 않는다. 먼저 없는 UUID/타인 UUID의 기대 404가 현재 500으로 실패하는지 위 DB 명령으로 확인한다.
2. `ai.go`의 첫 조회에만 sentinel 분류를 추가하고 핸들러에서 JSON 404로 매핑한다. 같은 DB 명령으로 두 실패 시험과 정상 SSE 회귀가 통과해야 한다.
3. 기억 키 누락·설정 누락·비활성 회귀를 포함해 위 전체 검증 명령을 실행한다. 계획과 다른 오류 경로가 나오면 과제서를 수정하고 범위를 넓히지 않는다.

검증 배선 (소스 확인됨, 이번 정찰에서 새 DB 재현은 미실행)
- `server.go:readSetting`은 settings(namespace='ai', key='provider')의 value와 encrypted_value를 읽는다. INSERT/UPDATE할 value는 AISettings를 JSON으로 직렬화하고 Enabled=true, BaseURL=로컬 httptest.Server.URL, Model='test', MaxOutputTokens=32, RequestTimeoutSeconds=5를 사용한다. encrypted_value=''이면 nil Vault로도 설정 조회 가능. 기존 행 전체를 보존·복구하고 원래 없으면 시험 종료에 삭제한다. Bootstrap은 부르지 않는다.
- 제공자는 HTTP 경계의 httptest.Server로만 대체하고 실제 proxyAIStream을 통과시킨다. `/v1/responses` 호출 횟수를 atomic으로 세고 `event: response.output_text.delta\ndata: {"delta":"ok"}\n\n`을 응답한다. 거부 요청은 호출 0, 정상/빈 person_id는 호출 증가와 실제 delta를 확인한다. 실제 외부 AI나 API 키는 필요 없다.
- 정상 사람은 seedPerson+seedRelationship, 기억 0건이면 복호화 키가 필요 없다. 내부 오류 회귀는 자기 사람에 approved 기억 한 건(id,user_id,person_id,title,content_cipher,key_version 필수; 예: content_cipher='', key_version=999)을 넣고 user_key_versions는 넣지 않는다. dataKeyVersion이 ErrNoRows를 반환하므로 nil Vault 호출 전 500이 나와야 한다. 이 시험 데이터는 손상 데이터의 서버 오류 분류를 검증한다.
- openTestStore는 store.Open(ctx, dsn, nil)을 부른다. postgres:16-alpine 이미지가 이 기계에 있음을 확인했다. 기존 DB 테스트 파일 헤더의 docker 실행 방식을 쓰되 이전 포트 55433·55439·15434·55471·55481·55491·55521·55537은 피하고 충돌 없는 임의 호스트 포트를 할당한다. 자신이 만든 컨테이너만 정리한다.

대안 비교와 선택 근거
- 선택: 첫 조회를 sentinel로 분류 — 프로덕션 1파일, 추가 질의 없이 원인 구분 가능. 단순 핸들러 ErrNoRows 매핑보다 키 오류를 숨기지 않는다는 이점이 있다.
- 선행 EXISTS 질의 — 이해는 쉽지만 중복 왕복과 조회 사이 삭제 경합이 생긴다. 기존 조회의 오류를 분류할 수 있어 채택하지 않는다.
- 없는 사람을 빈 맥락으로 계속 진행 — 사용자 의도와 다른 전체/빈 맥락으로 유료 AI 호출이 진행돼 채택하지 않는다.
- 현상 유지 — 코드 위험은 없지만 재현 가능한 사용자 입력이 계속 장애로 분류된다. rows.Err 누락 과제보다 결정적인 실제 DB 재현이 쉽고, 반복 차선인 OpenAPI 형식 고정보다 사용자 가치가 높다.

작업량 산정 (정찰자의 판단, 통계적 신뢰도/45분 완료 보장은 아님)
- bottom-up 기준: DB/제공자 fixture와 실패 재현 12~16분, sentinel 수정 4~6분, 회귀·race/vet 확인 12~15분. 기본 28~37분, 알려진 DB 준비/설정 복원 변동에 contingency 3~8분으로 총 31~45분을 예상한다(중간 확신).
- 비교 기준: 9/28 실제 핸들러 입력 오류 개선과 유사하나 이번에는 실DB fixture가 필수이고 키 오류 분리가 추가되므로 S가 아닌 M. 과거 실제 소요 시간 데이터는 없어 시간 범위를 유추 검증할 수 없다.
- 포함: 프로덕션 1파일·테스트 1파일·로컬 PostgreSQL 준비/정리·회귀 검증. 제외: CI DB 서비스·실제 AI·인증 변경·암호화 복구. management reserve는 배정 없음; 예기치 않은 환경 고장이나 범위 확장은 별도 후속으로 남긴다. 첫 DB 재현 후 시간이 범위를 넘으면 검증을 생략하지 말고 추정을 갱신한다.

적용 스킬: 로컬 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration` SKILL.md를 읽었다. Skill 호출 도구는 노출되지 않아 파일 읽기로 적용. PMO references/sources.md도 확인했으며 외부 비용 모델·정량 신뢰도는 인용하지 않고 위 저장소 범위에 대한 판단 추정만 사용했다.
