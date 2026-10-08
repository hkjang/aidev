- 과제: chat 최상위 null을 공급자 조회 전에 400 invalid_json으로 거부 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/server/providers.go:chatCompletions`는 JSON `null`을 nil map으로 받아 공급자가 없으면 503 `provider_not_configured`, 기본 공급자가 있으면 `payload["model"]` 대입 panic으로 500 `internal_error`를 반환한다(이번 실제 HTTP 재현). 첫 디코딩에서 nil map을 거부하면 호출자의 잘못된 본문이 공급자 설정에 따라 서버 장애로 바뀌지 않고 일정한 400 응답으로 끝난다.
- 수용 기준:
  1) `Content-Type: application/json`으로 보낸 `null` 및 앞뒤 JSON 공백을 두른 `null`은 기본 공급자 미설정/정상 설정 두 상태 모두 400 `invalid_json`을 반환하고 upstream 호출 증가가 0이다. 정상 설정 사례는 실제 생성 API로 공급자를 만들고 설정 API로 기본 공급자를 지정한 뒤 확인한다.
  2) 배열·문자열·숫자·boolean 최상위 본문은 기존 400 `invalid_json`을 유지한다. 정상 객체는 기본 공급자 선택·모델 기본값 보충 후 실제 upstream에 1회 전달되어 200 정상 JSON을 받는다. 기존 messages/seed=42 보존·provider_id 제거·후행 공백·정확히 4 MiB 허용·초과 및 후행 JSON 거부 8사례를 모두 유지한다.
  3) 테스트는 전용 DB의 Migrate/Seed → `New(db,cipher,logger).Handler()` → `signIn` → 실제 HTTP 경로를 통과한다. 수정 전 null 사례의 503/500 실패와 수정 후 400을 확인하고, nil 검사를 잠시 제거하면 같은 사례가 다시 실패함을 확인 후 복원한다. 직접 handler 호출/가짜 principal 주입/소스 문자열 검사는 증거로 삼지 않는다.
- 건드릴 파일:
  - `internal/server/providers.go:558 chatCompletions` — 572행 첫 `decoder.Decode(&payload)` 오류 조건에 `payload == nil`을 포함해 기존 400 코드와 한국어 메시지로 종료. 577행 두 번째 Decode/EOF 검사를 그대로 보존. 공급자 조회(581행 이후)보다 앞이어야 한다.
  - `internal/server/chat_request_json_integration_test.go:TestChatRequestJSONContract` — 기존 `post`, `marshal`, `upstream`, `callCount` fixture를 확장. raw bytes를 보내는 현재 방식 유지. 설정 변경은 `session.do(..., http.MethodPatch, "/api/v1/settings", {"values":{"ai.default_provider_id": providerID}})` 또는 실제 HTTP PATCH helper로 수행하고 200을 검사한다. 미설정 사례를 먼저 실행하고 정상 공급자 설정 사례를 뒤에 실행하며 `t.Parallel`은 쓰지 않는다. 정상 기본 공급자 회귀는 `stream:false`를 명시한다.
  - `docs/api.md:OpenAI 호환 chat` — 첫 bullet의 단일 JSON 문서를 단일 JSON 객체로 명확히 하고 최상위 null 거부를 한 문장으로 적는다.
  - 범위: 프로덕션 1파일, 테스트 1파일, 문서 1파일. 새 의존성 없음.
- 검증 명령:
  - 전용 DB 준비(동명/포트가 비어 있을 때): `docker run -d --rm --name ai-admin-null-contract-pg -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test -p 127.0.0.1:55571:5432 postgres:16-alpine`; `docker exec ai-admin-null-contract-pg pg_isready -U ai_admin -d ai_admin_test`가 성공한 뒤 진행.
  - `TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55571/ai_admin_test?sslmode=disable' go test -count=1 -run '^TestChatRequestJSONContract$' -v ./internal/server`
  - `TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55571/ai_admin_test?sslmode=disable' go test -race -count=1 ./...`
  - `make lint`; `go build ./...`; `git diff --check`; 정리: `docker stop ai-admin-null-contract-pg`.
  - 이번 정찰 실행: 위 단독 테스트 명령이 기존 8사례 PASS, SKIP 없음(0.661초). `git diff --check` 통과. 전체 race/lint/build/웹은 미실행이며 구현 후 실행할 명령이다. `TEST_KEYCLOAK_ISSUER` 미설정이면 실제 OIDC E2E는 SKIP하므로 전체 외부연동 검증이라고 보고하지 않는다.
- 위험과 피할 것: null을 빈 map으로 초기화해 수용하지 말고 `len(payload)==0`으로 빈 객체까지 새로 막지도 않는다. `UseNumber`, 숫자 정밀도, messages 필수검증, provider_id 타입, 공통 `decodeJSON`, auth/recoverer, migrations, workflow, 웹/dist, VERSION/CHANGELOG는 이번 범위 밖이다. DB 테스트는 스키마 DROP을 하므로 공유/운영 DB 및 같은 DB를 사용하는 병렬 프로세스 금지. 감사 details에 요청 본문을 저장하지 않는다. f66d25c preferences 미머지와 b5146f2 key-scope verify-failed 접근을 다시 구현하지 않는다.
- 차선 후보: 프로필 email 빈 문자열의 NULL 삭제·생략/null 유지 계약 테스트·문서화 — 첫 과제가 이미 해결된 경우만 선택하며 기존 동작을 400으로 바꾸지 않는다.

실행 순서 및 체크포인트 (구현자는 각 단계를 완료할 때 상태를 갱신; 현재 모두 미착수):
1. 기존 fixture에 null 거부 사례와 정상 기본 공급자 사례를 추가하고 단독 테스트 명령으로 red를 확보한다. 정상 객체 사례가 통과하고 null에서만 예상한 503/500 실패가 나는지 자체 검토한다. 이 단계의 red는 커밋하지 않는다. 사람 승인 체크포인트 없음.
2. `chatCompletions`의 첫 Decode 조건만 보강하고 같은 단독 명령으로 green, nil 검사 제거로 red, 복원 후 green을 확인한다. EOF/상한 회귀 8사례가 유지되어야 다음 단계로 진행한다. 사람 승인 체크포인트 없음.
3. API 문장 변경 후 전체 race·lint·build·diff 검사를 순서대로 실행하고 결과/외부 서비스 SKIP/DB 정리를 기록한다. 전제가 다르면 과제서를 수정하고 범위를 늘리지 않는다. 사람 승인 체크포인트 없음.

선택 근거·대안: nil 검사는 기존 map decoder의 타입 거부 계약에 null 한 가지만 보완하므로 가장 작다. 공통 JSON 파서를 바꾸면 관리 API까지 영향이 커지고, 빈 map 초기화는 잘못된 요청을 upstream까지 보내며, 문서만 고치면 실제 panic이 남으므로 채택하지 않는다. 숫자 정밀도 후보는 지수표기 호환성 설계가 필요하고 승인/레거시 후보는 범위가 더 커서 후순위다.

추정 근거: 하향식 유추는 오늘의 424a42e와 같은 handler·HTTP fixture·문서 3파일이라는 점에 한정한다(이전 실제 작업시간은 미확인). 상향식 기본 작업은 테스트 및 red 8~12분 + 조건 수정/green·역검증 4~6분 + 문서 2~3분 + 전체 검증/정리 6~10분 = 20~31분; DB 준비/fixture 순서의 알려진 변동에 별도 예비 5~8분을 더해 25~39분, 중간 신뢰도인 실무 추정이며 통계적 80% 보장은 아니다. 별도 관리 예비는 배정하지 않으며 새로운 범위가 나오면 45분에 억지로 넣지 않는다. 가장 중요한 전제는 기존 테스트의 실제 DB/HTTP fixture를 그대로 재사용할 수 있다는 것(기존 8사례 실행으로 확인)이다.

정찰 증거: `probe-results.txt`에서 기본 공급자 없음 null→503, API로 생성·기본 지정 후 null/공백 null→500 및 upstream 0회, 배열/숫자/문자열/boolean→400, 정상 객체→200 및 기본 model/max_tokens 보충을 확인했다. `probe-server.log`에 `assignment to entry in nil map`, `providers.go:597` stack이 있다. 이는 서버 프로세스 전체 종료가 아니라 `server.go:recoverer`가 잡아 500으로 바꾼 요청 단위 panic이다.

적용 스킬: 전용 Skill 도구가 없어 설치된 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`와 `references/sources.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`를 직접 읽었다. 분해·범위/예비 분리·대안 비교·변경/증거/체크포인트를 위에 반영했다. 외부 지침의 수치나 확률을 차용하지 않았다.
