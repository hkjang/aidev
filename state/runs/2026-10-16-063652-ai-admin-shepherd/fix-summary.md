# 수리 요약 — PR #45 (ae64738) · 커밋 없음

- 고치지 않았다: CI 의 lint·test 실패를 트리에서 재현하지 못했고, 지적된 결함이 트리에 없다고 판단했다. 두 job 모두 3~4초에 실패하고 넘어온 "실패 로그(끝부분)" 는 비어 있는데, test job 은 postgres 서비스 health-check 만으로도 4초를 넘기므로 두 job 다 **첫 명령 이전**(러너 할당 / `actions/checkout` / `actions/setup-go`)에서 끝난 것이다. 트리 내용으로는 설명되지 않는다.
- 로컬 재현(전용 `postgres:16-alpine` 127.0.0.1:55841, 끝나고 제거): lint 3명령 전부 exit 0(`gofmt -l .` 무출력 / `go vet ./...` / `verify-version.sh` → "consistent: 1.2.37"), `go test -race -count=1 ./...` 전 패키지 ok(internal/server 149.679s), `TestApprovalCommentContract` 13/13 PASS(SKIP 아님, `-v` 확인), `cd web && npm ci && npm test && npm run build` → 18파일 81테스트 통과·빌드 성공, `docker build --pull ...` exit 0, `go build ./...`·`git diff --check` exit 0. CI test job 의 모든 단계를 로컬에서 통과시켰다.
- 역검증: `workflow.go` 의 두 가드(제목 `!storableInVarchar`, 코멘트 `comment_invalid`)를 임시 제거하니 같은 테스트가 500/400 불일치로 10건 FAIL(`approval_comment_integration_test.go:297·317`) → 테스트가 실제로 대상 코드를 지킨다. 즉시 `git checkout --` 로 복원했고 최종 트리는 ae64738 그대로다.
- 확인 못 한 것: 저장소가 private 이고 이 세션에 GitHub 자격증명이 없어(`gh auth` 미설정, API 404) CI 로그를 읽지 못했다. `1.26.6` 은 actions/go-versions 매니페스트에 여전히 존재하므로 setup-go 버전 해석 실패는 아니다. 남는 후보는 러너/캐시/과금 등 저장소 설정 쪽이며, 이는 `.github/workflows` 수정 금지 규칙에 걸리므로 손대지 않았다.
- 다음 단계 제안: PR #44 에 이어 두 번째로 같은 증상이라 코드 회차가 아니라 CI 설정·계정 상태를 보는 회차가 필요하다. 머지 전 로그 접근 권한이 있는 쪽에서 job 하나의 원시 로그를 확인할 것.
