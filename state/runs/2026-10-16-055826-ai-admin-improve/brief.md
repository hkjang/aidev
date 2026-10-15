- 과제: PR #44(83f264c)의 CI 실패를 CI 와 같은 조건으로 로컬 재현해 원인을 확정하고, `TestApprovalNULWrites` 를 기존 승인 fixture 에 합쳐 `internal/server` 패키지가 28번째 스키마 재구축 없이 끝나게 하기 (가치 4 / 위험 2 / 작업량 M)
- 왜: 지난 회차 구현(83f264c — `workflow.go` 승인 제목·코멘트 NUL 400)은 로컬 `go test -race -count=1 ./...`·`make lint` 가 모두 exit 0 이었는데 PR #44 의 CI 는 `lint=failure, test=failure` 로 두 번 떨어졌다(러너 노트 `2026-10-16-022847.../journal.md:30`). 로컬과 CI 의 차이는 ① CI 가 `TEST_KEYCLOAK_ISSUER` 를 넘겨 Keycloak OIDC E2E 를 **실제로 실행**한다는 점, ② CI 는 2코어 러너에서 `go test -race ./...` 를 **기본 10분 타임아웃**으로 돌린다는 점뿐이다. 승인 NUL 수정 자체는 비평에서 approve 된 올바른 변경이므로, 되돌리지 말고 **왜 CI 에서만 깨지는지**를 닫아야 이 수정이 릴리즈까지 간다.

## 먼저 할 일 — 원인 확정 (이 순서 그대로, 추측으로 고치지 말 것)
기준 커밋은 **83f264c**(`fix: 승인 제목·결정 코멘트의 NUL 을 저장 전 400 으로 거부`)다. 현재 워크트리는 `main@2b2ef67` 이고 83f264c 는 `auto/2026-10-16-0558` 브랜치 팁에만 있다 — 먼저 그 커밋 위에서 시작할 것.

1. **lint 재현(1분, DB 불필요).** CI lint job(`.github/workflows/ci.yml:12-33`)은 `make lint` 와 **완전히 같은 세 명령**이다(`Makefile:12-20`): `gofmt -l .` → `go vet ./...` → `./scripts/verify-version.sh`. 83f264c 체크아웃 상태에서 세 명령을 따로 돌려 exit code 와 출력을 그대로 로그에 남긴다.
   - 세 명령이 모두 통과하면 **lint 실패는 작업 트리 내용 때문이 아니다** — 두 job 이 공유하는 앞 단계(`actions/checkout`, `actions/setup-go go-version 1.26.6 cache:true`)나 러너의 체크 판정 문제다. 그 경우 "트리는 깨끗하다" 는 근거(명령 출력)를 원장에 남기고 2번으로 넘어가 test job 만 고친다. **워크플로 파일을 느슨하게 바꿔 통과시키는 것은 금지**다(타임아웃 상향·`continue-on-error`·`gofmt` 대상 축소·vet 비활성화 모두 금지).
   - 어느 하나가 실패하면 그것이 원인이다. 그 자리만 고친다. (참고: 이전 회차 로컬 로그 `make-lint.log` 는 `Version metadata is consistent: 1.2.37` 로 통과했다. `verify-version.sh` 의 `expect_current_v_tokens docs/api.md` 는 파일 안의 모든 `vX.Y.Z` 토큰이 정확히 `v1.2.37` 이어야 하는데, 83f264c 가 추가한 docs/api.md 문장에는 버전 토큰이 없다 — 내가 diff 로 확인했다. 즉 이 경로는 이미 배제됨.)
2. **test job 재현 — CI 와 같은 두 환경변수·같은 플래그로.** 이번 회차의 핵심이다. 이전 회차는 Keycloak 을 띄우지 않았고(`TEST_KEYCLOAK_ISSUER` 미설정 → E2E 1건 SKIP) `-count=1` 로 돌렸다. CI 는 둘 다 다르다.
   - 전용 Postgres: `docker run -d --rm --name ai-admin-ci-repro-pg -p 127.0.0.1:55591:5432 -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test postgres:16-alpine`
   - 전용 Keycloak: `.github/workflows/ci.yml:57-75` 의 `docker run` 을 그대로 복사해 쓴다(이미지 `quay.io/keycloak/keycloak:26.7.2`, realm 파일 `internal/server/testdata/keycloak-e2e-realm.json`, 포트 `127.0.0.1:18080`, `.well-known/openid-configuration` 이 200 이 될 때까지 대기).
   - 그리고 **CI 와 똑같이**(`-count=1` 없이, `-timeout` 없이 = 기본 10분):
     `TEST_POSTGRES_DSN=postgres://ai_admin:test-password@127.0.0.1:55591/ai_admin_test?sslmode=disable TEST_KEYCLOAK_ISSUER=http://127.0.0.1:18080/realms/ai-admin-e2e go test -race -v ./... 2>&1 | tee ci-repro-test.log`
   - 끝나면 두 컨테이너 모두 제거. **공유·운영 DB 로 돌리지 말 것** — 모든 통합 테스트가 `ai_admin`·`aiportal` 스키마를 `DROP ... CASCADE` 한다.
3. 2번이 재현한 실패 그대로를 고친다. 가장 유력한 두 가설(내가 로그를 못 봤으므로 **둘 다 미확인**):
   - **(A) 기본 10분 타임아웃 초과 — 내가 가장 유력하게 본다.** `internal/server` 는 `TEST_POSTGRES_DSN` 을 읽는 통합 테스트가 **28개 파일**이고 그 각각이 `DROP SCHEMA ... CASCADE` + `db.Migrate` + `db.Seed`(bcrypt) 를 처음부터 다시 한다(`DROP SCHEMA IF EXISTS ai_admin CASCADE` 28회 / 26파일 — grep 으로 확인). 이전 회차 로컬 측정이 **internal/server 160.2초**였고 CI 는 2코어 + `-race` + 서비스 컨테이너라 수 배 느리다. 83f264c 는 여기에 **28번째 전체 재구축 + 2회 bcrypt 로그인(`signIn`·`signInReviewer`) + 17 서브테스트**를 더했고, CI 는 추가로 Keycloak E2E 1건(로컬에서는 SKIP 되던 것)까지 실행한다. 증상은 `panic: test timed out after 10m0s` + 해당 시점 goroutine dump 이며, 결정적이라 **두 번 연속 같은 이유로 실패**하는 모양과 맞는다.
     → 고치는 방향(워크플로를 느슨하게 하지 않는 유일한 방향): `TestApprovalNULWrites` 의 **셋업을 새로 만들지 말고** 이미 같은 승인 경로를 세워 둔 `TestApprovalCommentContract`(`internal/server/approval_comment_integration_test.go:26-62`) 안으로 서브테스트로 옮긴다. 두 셋업은 거의 동일하다 — 기존 것이 `DROP`→`Migrate`→`Seed`→`signIn`→`system_setting workflow.enabled=true`→`workflow_policy api_key_privileged enabled=true`→`signInReviewer` 를 이미 하고, 새 테스트가 더 필요한 것은 `UPDATE ai_admin.workflow_policy SET enabled=true WHERE code='general_review'` **한 줄**뿐이다. 이렇게 하면 패키지가 얻는 셋업 비용이 0 이 되고 **단정은 하나도 줄지 않는다**(아래 수용 기준 2 참조).
   - **(B) Keycloak E2E 가 실제로 실행되면서 깨짐.** 로컬에서는 SKIP 되던 `internal/server/oidc_keycloak_e2e_test.go` 가 CI 에서만 돈다. 다만 그 테스트도 `:46` 에서 자기 스키마를 `DROP SCHEMA IF EXISTS ai_admin CASCADE` 하므로 `TestApprovalNULWrites` 가 남긴 `workflow.enabled=true`·정책 2개가 새 나갈 경로는 **없다**(내가 grep 으로 28개 전부 DROP 하는 것을 확인했다). 그래서 (B) 는 순위가 낮지만, 2번 재현에서 이 테스트가 실패하면 그것이 원인이다 — 그때는 승인 NUL 수정과 무관한 별개 결함일 수 있으니 원장에 분리해 적을 것.
   - 재현이 **아무 실패도 내지 않으면**(전부 PASS) 거기서 멈추고 고치지 말 것. `ci-repro-test.log` 와 lint 세 명령 출력을 증거로 "트리는 CI 와 같은 조건에서 통과한다 — 실패는 러너/GitHub 쪽" 으로 원장에 적고 끝낸다. **통과시키기 위해 워크플로나 테스트를 약화시키는 변경은 금지.**

- 수용 기준:
  1) `.github/workflows/ci.yml` 의 lint job 세 명령과 test job 의 `go test -race ./...`(TEST_POSTGRES_DSN + TEST_KEYCLOAK_ISSUER 둘 다 설정, `-count=1`·`-timeout` 없이)를 로컬에서 그대로 돌린 **출력이 로그 파일로 남아 있고**, CI 가 보고한 실패가 재현됐는지 여부가 명시적으로 적혀 있다(재현 못 했으면 "재현 못 함" 이라고 적는다 — 추측으로 고치지 않는다).
  2) (A) 가 원인일 때: `TestApprovalNULWrites` 의 **13개 서브테스트 단정이 하나도 사라지지 않은 채** 기존 fixture 를 재사용하도록 바뀌었다 — 제목 NUL 3사례(`approval_request` 행 수 불변), 코멘트 NUL 6사례(reject/approve 각 3, `status=pending`·`decision_comment=NULL`·`approval_action` 0행 불변), `workflow.enabled=false` 에서도 NUL 제목이 400 이고 NUL 없는 같은 제목만 200 `bypassed`, 리터럴 여섯 글자 `\u0000` 의 제목·코멘트 round-trip, 기존 상한 회귀(300 통과 / 301·공백만 400 `approval_invalid`, 2001자 400 `comment_too_long`, 빈 반려 400 `comment_required`, 2000자 통과). `workflow.enabled` 를 `false` 로 바꾸는 서브테스트는 **반드시 `defer` 로 `true` 로 되돌려** 같은 fixture 를 쓰는 뒤 사례를 깨뜨리지 않게 한다.
  3) `internal/server` 패키지의 전체 소요 시간이 83f264c 대비 **줄었거나 같다**(before/after 숫자를 둘 다 기록). `go test -race ./...` 가 기본 10분 타임아웃 안에서 끝난다.
  4) `workflow.go` 의 두 400 검증(제목 `approval_invalid`, 코멘트 `comment_invalid`)과 `docs/api.md` 의 계약 문장은 **그대로 유지**된다. 이 회차는 83f264c 를 되돌리는 회차가 아니다.
  5) `make lint`·`go build ./...`·`git diff --check` 가 exit 0 이고 전용 컨테이너 2개가 제거돼 있다.

- 건드릴 파일 (프로덕션 0개 — 전부 테스트/워크플로 외부):
  - `internal/server/approval_nul_integration_test.go` — (A) 면 이 파일의 `TestApprovalNULWrites` 를 통째로 제거하고 그 서브테스트를 아래 파일로 옮긴다(파일 삭제). 독립 실행이 필요하면 셋업만 공유하는 형태로.
  - `internal/server/approval_comment_integration_test.go:26-62` — `TestApprovalCommentContract` 셋업에 `UPDATE ai_admin.workflow_policy SET enabled=true WHERE code='general_review'` 한 줄 추가 + NUL 서브테스트 13개를 이 테스트 안으로. 기존 코멘트 정규화 단정은 손대지 않는다.
  - (2번 재현이 다른 원인을 가리키면) 그 원인이 있는 파일 1~2개만. 파일 4개를 넘기면 과제를 쪼개고 이번에는 한 조각만.
  - **건드리지 않을 것**: `internal/server/workflow.go`, `docs/api.md`, `.github/workflows/*`, `internal/database/migrations`, `internal/auth`, `VERSION`, `CHANGELOG.md`, `internal/ui/dist`, `web/`.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `make lint` (= `gofmt -l .` + `go vet ./...` + `./scripts/verify-version.sh`; 1.2.37 이 나와야 한다)
  - `go build ./...`
  - 좁은 확인: `TEST_POSTGRES_DSN=... go test -race -count=1 -run 'TestApprovalCommentContract$' -v ./internal/server` (서브테스트 전부 PASS, SKIP 아님을 `-v` 로 확인)
  - CI 재현: 위 2번의 Postgres + Keycloak 두 컨테이너 + `TEST_POSTGRES_DSN=... TEST_KEYCLOAK_ISSUER=http://127.0.0.1:18080/realms/ai-admin-e2e go test -race -v ./... 2>&1 | tee ci-repro-test.log` (`-count=1`·`-timeout` 없이 — CI 와 동일)
  - `git diff --check`

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들지 말 것** — `-timeout` 상향, `continue-on-error`, `gofmt`/`go vet` 대상 축소, Keycloak 단계 제거, 테스트 `t.Skip` 추가는 전부 금지. 통과시키는 것이 목적이 아니라 원인을 닫는 것이 목적이다.
  - `workflow.go` 는 보호 구역이고 이미 비평에서 approve 됐다. 이번 회차에 로직을 또 손대면 지난 회차 검증이 무효가 된다.
  - fixture 를 합칠 때 `TestApprovalCommentContract` 의 기존 단정(코멘트 정규화 1회·`null` 계약·2000자 상한)을 바꾸지 말 것. 서브테스트 실행 순서에 의존하는 상태(`workflow.enabled`, 정책 enabled, 만들어 둔 pending 요청)를 공유하게 되므로, 상태를 바꾸는 서브테스트는 `defer` 로 원복한다. 이것이 이 과제의 유일한 실질 위험이다.
  - 통합 테스트는 `ai_admin`·`aiportal` 스키마를 DROP 한다 — 전용 컨테이너를 새 포트로 띄우고 끝나면 제거(다른 회차 컨테이너 재사용 금지).
  - 감사/오류에 요청 원문(NUL 포함 문자열)을 넣지 말 것. 새 헬퍼 이름 `jsonObjectHasNUL`·`storablePreferenceObject` 는 미머지 브랜치와 충돌하므로 만들지 말 것.
  - `storableInVarchar`(`internal/server/users.go:421`)는 읽기만 — 이름·구현을 바꾸면 프로필·사용자·역할·키·공급자 다섯 경로가 같이 움직인다.

- 차선 후보: **CI lint/test job 에 실패 원인을 드러내는 진단만 추가(워크플로 완화 아님)** — 2번 재현이 아무 실패도 내지 못했고 lint 세 명령도 전부 통과했다면, `internal/server` 의 통합 테스트 셋업 중복(28회 DROP/Migrate/Seed) 자체가 CI 를 10분 벼랑 끝에 두고 있다는 사실만 측정해 원장에 숫자로 남기고, 같은 승인 경로를 쓰는 테스트 2개(`approval_comment`·`approval_nul`)만 fixture 를 합치는 최소 변경으로 마무리한다(테스트 2파일, 프로덕션 0파일).
