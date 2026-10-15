- 과제: CI test job 중 지금까지 한 번도 로컬에서 실행된 적 없는 단계(`web` 의 `npm ci && npm test && npm run build`, `docker build`)를 실제로 돌려 PR #45 실패 원인을 좁히고, 그 네 단계를 한 명령으로 재현하는 `make ci-test` 를 남기기 (가치 4 / 위험 1 / 작업량 M)
- 왜: PR #44·#45 가 연속으로 CI 실패했는데 직전 회차는 lint 세 명령과 `go test -race`(CI 와 같은 두 환경변수·전용 PG+Keycloak)까지만 재현해 **전부 통과**시켜 원인을 못 찾고 끝났다. CI test job 은 그 뒤에 `npm ci && npm test && npm run build` 와 `docker build` 두 단계를 더 돌리는데(`.github/workflows/ci.yml:60-64`), 이 두 단계는 이 저장소의 어떤 회차 기록에도 로컬 실행 흔적이 없다 — 남아 있는 유일한 in-repo 재현 후보이며, 동시에 다음 회차가 같은 자리에서 또 헤매지 않게 하는 재현 수단이 없는 것이 근본 문제다.
- 수용 기준:
  1) 아래 "검증 명령" 의 ①②③④ 를 실제로 돌린 로그(명령·exit code·소요시간)를 원장에 남긴다. ②③ 는 이 저장소에서 처음 실행되는 것이므로 결과가 통과든 실패든 그 자체가 산출물이다.
  2) ②(web) 또는 ③(docker build) 가 실패하면 **워크플로가 아니라 원인 파일**(`web/*`, `Dockerfile`, `web/scripts/collect-runtime-licenses.mjs`, 필요하면 `go.mod`/`go.sum`)에서 고치고 같은 명령으로 통과를 확인한다. 실패 원문(첫 에러 줄)을 원장에 인용한다.
  3) 재현 여부와 무관하게 `Makefile` 에 `ci-test` 타깃을 추가한다: CI test job 과 **같은 순서·같은 플래그**로 ⓐ `go test -race ./...`(CI 와 동일하게 `-count=1`·`-timeout` 없이) ⓑ `cd web && npm ci && npm test && npm run build` ⓒ `docker build --pull --build-arg VERSION=v$(VERSION) -t ai-admin:ci .` 를 돌리고, `TEST_POSTGRES_DSN` 이 비어 있으면 **시작 전에 실패**해야 한다(DB 통합 테스트가 조용히 SKIP 된 채 "통과" 로 보이는 것이 지난 두 회차의 함정이었다). `TEST_KEYCLOAK_ISSUER` 는 비어 있으면 경고만 출력하고 계속한다.
  4) `make ci-test` 를 전용 PostgreSQL 컨테이너를 띄운 상태에서 실제로 한 번 돌려 exit 0 을 확인한다(③ 의 docker build 포함, 소요시간 기록).
  5) `.github/workflows/` 아래 파일은 한 줄도 바꾸지 않는다. `go test` 에 `-timeout` 을 올리거나 단계를 빼는 식의 완화 금지.
  6) `make lint`·`go build ./...`·`git diff --check` 가 exit 0. 특히 `scripts/verify-version.sh` 는 `README.md`·`docs/*.md` 안의 모든 `vX.Y.Z` 토큰이 정확히 `v1.2.37` 하나이기를 요구하므로(`verify-version.sh:33-39`, `:108-121`), 문서에 다른 버전 토큰을 새로 넣지 말 것.
- 건드릴 파일 (프로덕션 0~2개):
  - `Makefile` — `.PHONY` 에 `ci-test` 추가 + 새 `ci-test` 타깃. 기존 `lint`(:12-19)·`test`(:21-23)·`docker`(:34-35) 타깃은 **그대로 두고** 새 타깃만 추가한다. `VERSION`·`COMMIT`·`BUILD_TIME` 변수는 이미 `:3-5` 에 있으니 재사용.
  - `docs/operations.md` — `make ci-test` 가 무엇을 재현하는지, `TEST_POSTGRES_DSN`·`TEST_KEYCLOAK_ISSUER` 가 필요하다는 한두 문장. 버전 토큰(`vX.Y.Z`)을 쓰지 말 것.
  - (②③ 가 실제로 실패했을 때만) 그 원인 파일 1개.
- 검증 명령 (PR #45 트리를 따로 체크아웃할 필요 없음 — PR #45 의 diff 는 `internal/server/workflow.go`·`internal/server/approval_comment_integration_test.go`·`docs/api.md` 뿐이고 `web/`·`Dockerfile`·`go.mod`·`go.sum` 은 main 과 동일하므로 ②③ 는 현재 브랜치에서 돌려도 같은 조건이다. 확인: `git diff --stat main origin/auto/2026-10-16-0558`):
  - ① `docker run -d --rm --name ai-admin-ci-pg -p 127.0.0.1:55601:5432 -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test postgres:16-alpine` 뒤 `TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55601/ai_admin_test?sslmode=disable' go test -race -v ./... 2>&1 | tee logs/ci-test-go.log`
  - ② `cd web && npm ci && npm test && npm run build` (각 단계 exit code 기록. 네트워크 필요)
  - ③ `docker build --pull --build-arg VERSION=v$(cat VERSION) --tag ai-admin:ci .` (CI 마지막 단계와 동일. `Dockerfile:36-49` 의 의존성 라이선스 수집 루프가 라이선스 파일 없는 모듈을 만나면 여기서 결정적으로 실패한다 — 실패하면 이게 원인이다)
  - ④ `make ci-test` (①의 컨테이너를 띄운 상태에서, `TEST_POSTGRES_DSN` 를 주고 / 또 한 번 주지 않고 → 후자는 즉시 실패해야 함)
  - ⑤ `make lint` · `go build ./...` · `git diff --check`
  - 끝나면 `docker rm -f ai-admin-ci-pg` 와 `docker image rm ai-admin:ci` 로 정리하고 작업 트리가 깨끗한지 확인.
- 위험과 피할 것:
  - **워크플로 완화 금지** — `.github/workflows/ci.yml`·`release.yml`·`pages.yml` 수정 금지. `cache: true` 제거, `-timeout` 상향, `continue-on-error`, 단계 삭제 모두 금지.
  - `internal/server/workflow.go`·`docs/api.md` 는 PR #45 가 이미 바꿨다. 같은 자리를 다시 건드리거나 `0017e72`/`ae64738` 를 다시 cherry-pick 하지 말 것(직전 회차가 이미 했고 PR 로 열려 있다).
  - `make web-build`(:25-28) 는 `internal/ui/dist` 를 지우고 덮는다. ② 는 `cd web && npm run build`(→ `web/dist`)만 하므로 `internal/ui/dist` 를 재생성하지 말 것. `npm ci` 가 `web/package-lock.json` 을 수정하면 되돌릴 것.
  - `scripts/verify-version.sh` 는 `web/package.json`·`web/package-lock.json` 의 `version` 이 `1.2.37` 이기를 요구한다(:61-79). npm 작업으로 이 값이 바뀌면 lint 가 깨진다.
  - lint job 실패는 직전 회차가 세 명령 모두 로컬 통과를 증명했고 `verify-version.sh` 는 git 상태를 보지 않는 순수 파일 검사임을 이번에 재확인했다(`:1-129`). 따라서 lint 실패를 트리 내용으로 설명하려 시도하지 말 것 — 남는 자리는 `actions/checkout`·`actions/setup-go` 이고 저장소가 private 이라 로그를 읽을 수 없다(`gh` 사용 불가, 이번 회차 확인). 추측으로 고치지 말고 "미확인" 으로 남길 것.
  - DB 통합 테스트는 `ai_admin`·`aiportal` 스키마를 DROP CASCADE 한다. 전용 컨테이너를 새 포트로 띄우고 끝나면 제거(다른 회차 컨테이너 재사용 금지).
- 차선 후보: chat `stream` 타입 및 두 토큰 한도 동시 지정의 400 거부 계약을 통합 테스트로 고정 (2/1/S) — `providers.go:611-634` 의 `stream_invalid`·`max_tokens_conflict`·`max_tokens_invalid` 세 코드가 `internal/server` 테스트에 문자열로 없다(이번 회차 재확인 안 함 — 먼저 `grep -rn max_tokens_conflict internal/server` 로 확인할 것). 프로덕션 0파일이고 기존 `chat_request_json_integration_test.go` fixture 를 **재사용**해 새 셋업을 만들지 않는 것이 전제. 승인 payload(jsonb) NUL 은 PR #45 와 같은 `workflow.go` 를 건드리므로 머지 전까지 고르지 말 것.
