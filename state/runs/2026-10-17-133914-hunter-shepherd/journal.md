# PR 처리기 노트 2026-10-17-133914-hunter-shepherd — hunter PR #24
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-17-114840-hunter-improve)
# 회차 노트 2026-10-17-114840-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:48] base pinned — main@46c8b16
- [러너 11:48] autonomy release — 

## 정찰 노트
- 고른 이유: 2026-10-07 에 머지된 "한 값으로 저장"(contribution_points) 과 같은 계약의 미기록 결함을 설정 쪽에서 찾았다 — `validateSettings` 의 ai·agents·security 만 `asInt` 로 검사하고 원값을 저장해, 소수가 불변 실행 스냅샷(`agents.go:293`)에 동결되고 보고서(`agent_reports.go:115-122`)가 집행되지 않은 한도를 인쇄한다. 서버 1파일로 끝나고 DB 없이 검증된다.
- 제친 후보: settings.tsx 계열 전부(inventory integer, ai 교차 제약, 도구 7종 문구) — 2026-10-08 의 integer PR 이 아직 main 에 없어(grep 무출력, 그 파일 마지막 커밋 daff3bc) 같은 파일 충돌. engines 는 no-change 이력이라 rejected. 담당자 공유 벡터는 프로덕션 0파일.
- 확신 없는 곳(추측): 관리자 폼이 그룹 값을 원형 PUT 한다는 전제는 2026-10-08 기록에서 가져왔고 `settingsGroupValues` 본문까지는 읽지 않았다. `agents_tools.go` 의 실제 도구 수(7 대 8)도 미확인.
- 구현자가 조심할 것: 소수를 400 으로 바꾸지 말 것(이미 소수가 저장된 행의 그룹 저장이 전부 막힌다 — 2026-10-16 grandfathering 교훈). 되쓰기로 값 타입이 int 가 되지만 이 키들에 `.(float64)` 단정하는 독자는 없음을 확인했다. 오류 문구 3개와 판정 순서는 보존.
- 실행해 본 것: `go test -run '^TestSettingsLimits$' -count=1 -v ./internal/app` → PASS(0.023s, DB 불필요). 코드는 바꾸지 않았고 작업 트리는 clean 이다.
- [러너 11:55] scout done — 설정의 정수 한도를 저장값과 집행값이 같은 하나의 값으로 고정 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- `validateSettings` 의 `ai`·`agents`·`security` 세 case 가 `asInt` 로 검사만 하고 원값을 저장하던 것을 통과 시 `v[k] = n` 으로 되쓰게 했다(프로덕션 1파일, 커밋 50aab02). 저장값·관리자 화면·불변 실행 스냅샷·오프라인 보고서가 집행값과 같아진다. 새 거절은 만들지 않았다 — 소수는 계속 저장되고 정규화만 된다(기존 행 grandfathering).
- 확신 없는 곳·검증 못 한 것: `HUNTER_TEST_DSN` 이 없어 **HTTP PUT → DB 저장 → GET 왕복을 실제로 돌리지 못했다**(testApp 128 SKIP). 되쓰기가 저장값이 된다는 근거는 핸들러가 검증기에 넘긴 바로 그 맵을 `json.Marshal` 하는 코드 독해뿐이다. 실제 agent run 생성→보고서 PDF 생성의 end-to-end 도 DSN 필요라 미실행이며, 보고서 일치는 `reportDocument`+`reportText` 를 직접 호출하는 단위 경로로만 증명했다. `go test -race ./...`·원격 CI·Docker 도 미실행.
- 일부러 하지 않은 것: `web/src/settings.tsx` 는 열지 않았다(2026-10-08 미머지 PR 과 같은 파일). `asInt`·`defaultSettings`·핸들러·`auth.go`·`keys.go`·`sla`/`risk`/`inventory` 검증기도 손대지 않았다. 거대한 소수(1e300)의 `int()` 미정의 동작은 기존 범위 검사가 거르므로 새로 다루지 않았다.
- 다음 역할이 조심할 것: 되쓰기로 이 8개 키의 값 타입이 `float64`→`int` 로 바뀐다. 새로 `.(float64)` 단정을 추가하지 말 것(현재 `internal/app` 에 이 키들을 그렇게 읽는 독자는 없음). 새 테스트 4개는 전부 DB 불필요(`go test -run 'TestSettings|TestAgentReportPrintsTheEnforcedLimit' ./internal/app`).
- [러너 11:59] brief accepted — 채택 — 지목한 파일·행(`settings.go:262-288` 의 세 case, `asInt` 절삭, `agents.go:293` 스냅샷, `agent_reports.go:115-122` 원값 인쇄)과 수
- [러너 11:59] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 구현자가 의심한 자리(되쓰기가 저장값이 되는가)를 직접 시험했다: main 을 detached worktree 로 꺼내 새 테스트 파일만 복사해 돌려 `float64(24.9)`·`float64(12.9)`·`float64(8192.9)` 저장과 보고서 `max_tool_calls: 40.9` 로 **수정 전 실패를 재현**했고 HEAD 에서 4개 PASS. 경계·문구 테스트 2개는 수정 전에도 통과하는 회귀 가드지만 변경을 증명하는 테스트가 따로 있어 문제 없음.
- 8개 키의 프로덕션 독자를 전수 확인해(auth.go:106, keys.go:71, ai.go:80, agents_llm.go:34-36, agents_runner.go:112·233, agents_tools.go:42·45·89, agents_control.go:167, agent_models.go:401·534) 전부 asInt 경유 — 집행은 불변이고 `.(float64)` 독자는 없다. validateSettings 프로덕션 호출자는 settings.go:180 한 곳이며 되쓰기는 mcp_oauth.go:103 선례와 같다. 권한·비밀값·의존성·원본 312파일 변화 없음 → security/legal 차단 사유 없음.
- 못 본 것: HUNTER_TEST_DSN 부재로 PUT→DB→GET 왕복과 PDF end-to-end 미실행(128 SKIP). 실행한 것: go build / go vet / gofmt -l / verify-pentagi 전부 통과.
- 승인이어도 남는 우려(릴리즈 노트): 정규화는 **저장 시점에만** 적용돼 기존 소수 행·기존 run 스냅샷은 다시 저장할 때까지 계속 미집행 값을 인쇄한다. audit 는 제출 원값(40.9), 저장은 40. settings.tsx `integer` 선언은 여전히 없어 입력 측 완성이 남았다. VERSION 미변경(1.25.0).
- 판정: approve / risk low / blocking 없음.
- [러너 12:03] review approved — 리뷰 승인 (risk=low)
- [러너 12:03] pr created — https://github.com/hkjang/hunter/pull/24
- [러너 12:30] resume held — 리뷰 결과 없음/무효 — 사람 판단

## 심사 노트
- 확인: main(46c8b16)을 별도 worktree로 꺼내 새 테스트만 복사해 돌려 **수정 전 4개 FAIL / HEAD 4개 PASS**를 직접 재현했다(보고서가 `max_tool_calls: 40.9`를 인쇄). 핸들러 settings.go:150-211을 읽어 `validateSettings`가 받은 맵이 그대로 `json.Marshal`된다는 것, 되쓰기가 roles case의 기존 선례와 같다는 것, 오류 경로에서 부분 저장이 없다는 것을 확인했다.
- 확인: 8개 키의 `.(float64)` 독자 없음, 집행은 전부 asInt(auth.go:106·keys.go:71 직접), 기존 DB 테스트도 asInt/정수 리터럴뿐이라 CI PostgreSQL 작업이 깨지지 않음. openapi.json이 이미 `"type":"integer"`라 구현이 계약에 맞춰진다. int와 float64(40)이 `fmt.Sprint`에서 같은 출력이라 DB 왕복 후에도 효과가 유지된다. race 포함 전 검사 통과(vet·build·gofmt·verify-pentagi 312).
- 못 본 것: PostgreSQL 부재(5432 closed, DSN 미설정)로 **PUT→DB→GET 왕복과 PDF end-to-end 미실행 — 51 PASS/0 FAIL/128 SKIP의 SKIP을 통과로 보지 않는다.** 실브라우저·관리자 화면도 미확인.
- 기존 결함 발견(이 PR 소관 아님, notes로 넘김): settings.go:205가 요청 원문을 감사 detail로 넘기는데 server.go:159의 redact()가 `api_key`를 마스킹하지 않아 AI 공급자 키가 평문으로 audit_logs에 들어가고 GET /api/audit로 나간다. main에 그대로 있는 줄이라 차단하지 않았다.
- 권고 **approve / merge / risk low** — 새 거절 없는 정규화라 기존 소수 행의 grandfathering이 유지되고, 권한·비밀값·마이그레이션·원본 312파일 무변화에 코드 2파일 revert로 되돌아간다.
