# 회차 노트 2026-10-16-055826-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:58] base pinned — main@2b2ef67
- [러너 05:58] autonomy release — 

## 정찰 노트
- 고른 이유: 우선 과제(PR #44 CI 실패) 고정. CI 로그에 접근할 수 없어(gh 명령 미승인, 명령 실행 권한 없음) 원인을 단정하지 못했다 — 대신 로컬/CI 차이를 좁혀 재현 절차를 과제서로 만들었다. 차이는 둘뿐이다: CI 만 `TEST_KEYCLOAK_ISSUER` 로 Keycloak E2E 를 실제 실행하고, CI 는 2코어에서 `-count=1`·`-timeout` 없이(기본 10분) `go test -race ./...` 를 돈다.
- 추측으로 적은 것: 1순위 가설(기본 10분 타임아웃 초과)은 **미확인**이다. 근거는 간접적이다 — 로컬 internal/server 160.2초, DSN 통합 테스트 28파일이 각각 DROP SCHEMA+Migrate+Seed 반복(grep 확인), 83f264c 가 28번째를 추가. `lint=failure` 는 이 가설로 설명되지 않는다: 83f264c 트리에서 `make lint` 세 명령(CI lint job 과 동일, Makefile:12-20)이 전부 통과했고(이전 회차 make-lint.log) `verify-version.sh` 의 `expect_current_v_tokens docs/api.md` 경로는 추가 문장에 버전 토큰이 없어 내가 diff 로 배제했다.
- 배제한 것: 테스트 간 상태 누출(새 테스트가 남기는 `workflow.enabled=true`)은 28개 통합 테스트 전부가 자기 스키마를 DROP 하므로 불가능 — Keycloak E2E(`oidc_keycloak_e2e_test.go:46`)도 DROP 한다. 헬퍼 시그니처 불일치(`signIn`·`signInReviewer`·`assertFailure`·`sessionCredentials.do`)도 전부 main 과 일치해 컴파일 실패는 아니다.
- 구현자가 조심할 것: 먼저 lint 세 명령을 따로 돌려 exit code 를 로그로 남기고, 통과하면 트리 문제가 아니라고 적고 test job 재현으로 넘어갈 것. 재현이 아무 실패도 못 내면 고치지 말고 그렇게 기록할 것. 워크플로 완화(-timeout 상향·continue-on-error·Keycloak 단계 제거·t.Skip)는 금지. `workflow.go` 의 두 400 검증과 docs/api.md 문장은 유지 — 되돌리는 회차가 아니다.
- 이번 회차: 코드 변경·커밋 없음, 테스트·빌드 실행 없음(명령 권한 제한). 프로필은 0일 전이라 새로 쓰지 않았다.
- [러너 06:06] scout done — PR #44(83f264c)의 CI 실패를 CI 와 같은 조건으로 로컬 재현해 원인을 확정하고, `TestApprovalNULWrites` 를 기존 승�

## 구현 노트
- 무엇을 왜: 83f264c 를 cherry-pick(트리 동일) 한 뒤 CI 와 같은 조건을 로컬에서 재현했고, 아무 실패도 나오지 않아 가설 (A)/(B) 를 고치지 않았다. 대신 과제서의 차선 후보만 했다 — `TestApprovalNULWrites` 의 13 서브테스트를 단정 하나도 줄이지 않고 `TestApprovalCommentContract` 로 옮기고 `approval_nul_integration_test.go` 를 삭제(프로덕션 0파일). `internal/server` 161.782s → 154.461s.
- **확신 없는 곳**: PR #44 의 `lint=failure` 원인은 **모른다**. lint 세 명령(`gofmt -l .`/`go vet ./...`/`verify-version.sh`)이 83f264c 트리에서 전부 exit 0 이라 트리 내용은 아니고, 남는 자리는 두 job 이 공유하는 `actions/checkout`·`actions/setup-go(1.26.6, cache:true)` 뿐인데 저장소가 private 이라 체크런 API 가 404 로 막혀 확인하지 못했다(`1.26.6` 이 actions/go-versions 매니페스트에 여전히 있는 것만 확인). 10분 타임아웃 가설도 **증명도 반증도 못 했다** — 이 머신에서 전체가 2분 27초이므로 2코어 CI 가 10분을 넘는지는 알 수 없다.
- 일부러 하지 않은 것: `workflow.go`·`docs/api.md`·`.github/workflows/*` 는 한 줄도 안 건드렸다(역검증 때 `workflow.go` 를 임시 중성화했으나 복원해 최종 diff 0). `-timeout` 상향·`continue-on-error`·Keycloak 단계 제거·`t.Skip` 은 전부 금지 사항이라 하지 않았다. 남은 26파일의 셋업 중복도 쪼개서 다음 회차로 넘겼다.
- 다음 역할이 조심할 것: 합친 테스트는 `TEST_POSTGRES_DSN` 없으면 통째로 SKIP 된다 — `-v` 로 확인할 것. 13 서브테스트가 하나의 fixture(스키마·admin/reviewer 세션·`workflow.enabled`·정책 2개)를 **공유하고 순서에 의존**하므로, 상태를 바꾸는 서브테스트는 반드시 `defer` 로 원복해야 한다(현재 `a NUL title is refused ahead of the bypass` 하나만 그렇다). 공유 fixture 의 `requestApproval`·`decide` 는 이제 `*testing.T` 를 첫 인자로 받는다. 검증은 전용 컨테이너로만(통합 테스트가 `ai_admin`·`aiportal` 를 DROP 한다); 이번 회차 컨테이너 2개는 제거했다.
- [러너 06:19] brief fallback — 차선 — 과제서가 지정한 재현 절차(lint 세 명령, 전용 PG+Keycloak, 같은 두 환경변수, `-count=1`·`-timeout` 없이)를 그대로 돌렸
- [러너 06:19] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: workflow.go:163·248 의 두 검증이 기존 길이 검사와 같은 자리(Begin 이전)에 더해졌을 뿐이고 docs/api.md:342 문장이 코드와 정확히 일치함 / 역검증 로그의 실패 줄 297·317·336 이 현재 파일의 세 assertFailure 와 1:1 로 맞고 증상(500→400)이 커밋 주장과 같음 / 삭제된 approval_nul_integration_test.go 의 13 서브테스트·단언이 머지본에 전부 보존됨.
- 내가 직접 돌린 것: 전용 postgres:16-alpine(127.0.0.1:55732, 끝나고 제거)로 `TestApprovalCommentContract` 13/13 PASS, 전체 `go test -race -count=1 ./...` 전 패키지 ok(internal/server 151.1s), lint 세 명령·`git diff --check` 전부 exit 0. 판정 approve / risk low / blocking 없음.
- 못 본 것: PR #44 의 `lint=failure` 원인(저장소 private, 이 세션에 GitHub 권한 없음) — 트리는 내 손으로도 green 이라 러너·캐시 쪽으로 남는다. **머지 전 이 브랜치 CI 가 실제로 green 인지 한 번 더 볼 것**; 또 실패하면 코드가 아니라 CI 설정을 보는 회차가 필요하다.
- 승인이어도 남는 우려: 승인 payload(jsonb)와 MCP `create_approval_request`(mcp.go:299)는 여전히 NUL 무검증이라 같은 입력이 REST 400 / MCP·payload 500 으로 갈린다(이번 회차 회귀 아님, 원장 보류). 릴리즈 노트는 "REST 승인 제목·결정 코멘트" 로 범위를 좁혀 적을 것.
- 다음 회차가 알아야 할 것: 합친 13 서브테스트는 fixture 를 공유하고 순서에 의존한다 — 전역 상태를 바꾸는 서브테스트는 반드시 defer 복원, `requestCount` 가 테이블 전체를 세므로 t.Parallel 금지.
- [러너 06:26] review approved — 리뷰 승인 (risk=low)
- [러너 06:26] pr created — https://github.com/hkjang/ai-admin/pull/45
- [러너 06:27] ci failed — 성공이 아닌 검사: test=failure, lint=failure · 실패한 검사: ? 잡: lint 
