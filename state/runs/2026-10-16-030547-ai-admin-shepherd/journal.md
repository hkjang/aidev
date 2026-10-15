# PR 처리기 노트 2026-10-16-030547-ai-admin-shepherd — ai-admin PR #44
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-16-022847-ai-admin-improve)
# 회차 노트 2026-10-16-022847-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@2b2ef67
- [러너 02:28] autonomy release — 

## 정찰 노트
- 고른 이유: 승인 제목/코멘트의 NUL 은 세 번 연속 release-ready 로 끝난 "컬럼이 담을 수 없는 값을 쓰기 전에 400" 패턴의 마지막 남은 varchar/text 경로이고, 기존 헬퍼(`users.go:421 storableInVarchar`)와 기존 셋업(`approval_comment_integration_test.go:26-62`)을 그대로 써서 프로덕션 1파일로 끝난다. 차선인 chat stream/max_tokens 테스트는 프로덕션 0파일이지만 값이 낮고(2) 2026-10-07 테스트전용 회차가 review-pending 으로 끝난 전례가 있어 뒤로 뒀다. updatePreferences(3/1/S)는 미머지 f66d25c, updateKeyScope 는 verify-failed b5146f2 때문에 계속 제외.
- 추측으로 적은 것: 현재 코드가 NUL 제목/코멘트에서 실제로 500 을 낸다는 전제는 DB 를 띄워 재현하지 않았다(컬럼 타입 `varchar(300)`/`text` 와 과거 3회 동일 패턴이 근거). red 가 400/202 로 나오면 차선으로 교체할 것.
- 조심할 것: `workflow.go` 는 보호 구역이다 — 두 검증 지점만 고치고 승인 실행·키 회전·`createApprovalRecordIn` SQL 은 건드리지 말 것. 검사는 `Begin` 보다 앞에 둔다.
- 범위 밖: payload jsonb 의 NUL(새 헬퍼 필요 + 미머지 브랜치의 `jsonObjectHasNUL`·`storablePreferenceObject` 이름 충돌). 그 두 이름은 만들지 말 것.
- 코드 변경·커밋 없음. 이번 회차에 DB·테스트 실행 없음(읽기만).
- [러너 02:34] scout done — 승인 요청 제목(`varchar(300)`)·결정 코멘트(`text`)의 NUL(U+0000) 을 저장 전 400 으로 막기 (가치 3 / 위험 2 / 작업

## 구현 노트
- 무엇·왜: `workflow.go` 의 두 검증 지점만 고쳤다 — `createApproval` 제목에 `|| !storableInVarchar(request.Title)`(기존 `approval_invalid` 재사용), `decideApproval` 의 `comment_too_long` 바로 뒤에 `comment_invalid` 400. `varchar(300)`·`text` 가 NUL 을 담을 수 없어 500 으로 나가던 호출자 입력 오류를 400 으로 되돌린다. 검사는 둘 다 `tx.Begin` 보다 앞이고 `storableInVarchar`(`users.go:421`)는 읽기만 했다.
- 확신 없는 곳: `comment_invalid` 는 이번에 새로 만든 오류 코드다(과제서 권장). 웹(`web/`)이 이 코드를 모르면 기본 오류 표시로 떨어질 텐데, 웹 쪽 처리는 확인하지 않았다 — 기존 `comment_too_long`·`comment_required` 와 같은 400 envelope 이라 형태는 같다. 전체 suite 를 `-v` 없이 돌려 SKIP 목록을 다시 세지는 않았다(신규 테스트는 `-v` 로 따로 PASS 확인).
- 일부러 안 한 것: 승인 payload 의 jsonb NUL(새 헬퍼 필요 + 미머지 `f66d25c` 의 `jsonObjectHasNUL`·`storablePreferenceObject` 이름 충돌), `mcp.go:299` 의 같은 제목 검사(REST 경로만 고쳤다 — ideas.json 에 신규 항목으로 적었다), `updatePreferences`·`updateKeyScope`, 승인 실행·키 회전 SQL, 웹·VERSION·CHANGELOG·migration.
- 다음 역할이 조심할 것: `TestApprovalNULWrites` 는 `TEST_POSTGRES_DSN` 이 있어야 돌고(없으면 `t.Skip`) `ai_admin`·`aiportal` 스키마를 `DROP ... CASCADE` 한다 — 공유·운영 DB 로 돌리지 말 것. 전용 컨테이너 `ai-admin-approval-nul-pg`(포트 55581)는 제거했다. `general_review`·`api_key_privileged` 두 정책과 `workflow.enabled` 를 모두 켜야 INSERT 까지 내려간다.
- [러너 02:41] brief accepted — 채택 — 결함·코드 위치(`workflow.go:160-164`·`:235-243`)·컬럼 타입·헬퍼 위치(`users.go:421`)·재사용 셋업(`approval_comment_integratio
- [러너 02:41] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: `git diff main...HEAD`(3파일) 전체, `workflow.go:160-164`·`:244-248` 의 검사 위치(둘 다 `tx.Begin`·`createApprovalRecord` 앞), `users.go:421 storableInVarchar`, 새 테스트 248줄, red/green/revert-red/related 로그, race 전체 통과(internal/server 160s), make-lint·gofmt·`git diff --check`. red 출력의 증상(500 `approval_create_failed`/`approval_failed`, bypass 200)이 이번 변경이 고치는 것과 일치하고 계약 회귀 3건은 수정 전후 모두 PASS — 테스트는 실제로 바뀐 경로를 지난다.
- 못 본 것: DB 를 직접 띄워 재실행하지 않았고(기존 로그로 대체 판단), 웹 UI 를 실제로 띄워 `comment_invalid` 표시를 눈으로 확인하지는 않았다 — 다만 `web/` 에 승인 오류 코드를 코드별로 분기하는 곳이 0건이어서 기존 400 과 동일하게 message 가 표시된다.
- 판정: approve / risk low / blocking 없음(보안·법무 차단 사유 없음 — 새 엔드포인트·권한 확대·비밀값 없음, 입력 검증을 좁히는 방향, 오류에 입력 원문 미포함, revert 로 완전 복구).
- 남는 우려(릴리즈): 같은 계열이 두 군데 남았다 — `mcp.go:299` 의 MCP `create_approval_request` 제목(여전히 500)과 승인 `payload` jsonb(여전히 500). 릴리즈 노트는 "REST 승인 제목·결정 코멘트" 로 한정해 쓸 것, "승인 NUL 전부 차단" 으로 쓰지 말 것.
- 다음 회차: `approval_invalid` 메시지가 "1~300자" 만 말하면서 NUL 에도 쓰이는 점은 `users.go updateProfile` 선례와 같아 그대로 뒀다. MCP 제목 NUL 은 ideas.json 신규 항목으로 남아 있다.
- [러너 02:44] review approved — 리뷰 승인 (risk=low)
- [러너 02:44] pr created — https://github.com/hkjang/ai-admin/pull/44
- [러너 02:45] ci failed — 성공이 아닌 검사: lint=failure, test=failure · 실패한 검사: ? 잡: test 

## 수리 노트
- 틀린 지적: "lint·test fail" 은 이 변경의 결함이 아니다. CI 와 동일한 검증을 83f264c 그대로 로컬 재현 → 전부 통과(gofmt 무출력, go vet, verify-version 1.2.37, `go test -race -count=1 ./...` EXIT=0·FAIL 0건·internal/server 160.3s, web npm ci/test 81건/build, docker build 모두 0). 재현 실패이므로 고칠 대상이 없다.
- 맞았던 것: 없음(검사 상태 자체는 사실이지만 원인 귀속이 틀렸다). 원인은 스텝 실행 전 실패 — 두 잡이 17:44:36Z→17:44:38Z 2초, step 출력 전무(output.* 전부 null), 잡별 annotations_count 2. checkout+setup-go 만으로도 2초를 넘으므로 스텝이 하나도 안 돌았고, 워크플로 YAML 은 이 3파일 diff 밖이다.
- 고친 방법: 없음 — 절대 규칙(코드가 이미 맞으면 커밋 금지)에 따라 커밋하지 않았다. HEAD=83f264c, 트리 clean. 조치는 CI 재실행.
- 변경 타당성 재확인: `storableInVarchar` 주석이 varchar/text 양쪽 NUL 불가를 명시해 `text` 코멘트에 쓰는 것이 맞고 `updateProfile` 선례와 동형. 전 회차 red(500 approval_create_failed/approval_failed·bypass 200) → green(전부 400) 로그가 테스트의 실효성을 보인다.
- 여전히 확신 없는 곳: private 저장소 + GitHub 인증 없음(gh 미설정, API 404)으로 annotation 2건 원문을 못 읽었다 — 인프라 원인의 정확한 문구는 미확인(결제/지출 한도 또는 러너 가용성 추정). TEST_KEYCLOAK_ISSUER 미설정으로 OIDC E2E 1건은 로컬 SKIP.
