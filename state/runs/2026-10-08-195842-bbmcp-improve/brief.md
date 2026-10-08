- 과제: PR 대상 브랜치 조회 실패 시 실행 도구의 브랜치 검사 생략을 막는다 (가치 5 / 위험 2 / 작업량 M)
- 왜: `internal/tools/executor.go:prTargetBranch`가 Provider·자격증명·PR 조회 오류를 빈 문자열로 바꾸고, `Invoke`는 빈 브랜치이면 `BranchWritable`을 생략한 채 승인 단계로 진행한다. 조회 실패 또는 대상 ref 누락을 명시적으로 거부하면 브랜치 권한을 확인하지 못한 실행 요청이 승인 생성·소비·업스트림 쓰기로 넘어가지 않는다.
- 수용 기준:
  1) `RiskExecute` PR 호출에서 대상 브랜치를 알아내야 할 때 PR GET 503 또는 성공 응답의 빈 `toRef.id`(접두사 제거 후 빈 값 포함)는 `PERMISSION_UNKNOWN`으로 거부된다. 승인 여부와 무관하게 승인 검사보다 먼저 거부하며, 정상 `refs/heads/master`는 기존대로 `master`로 검사한다.
  2) 실패한 최초 요청은 `approval_requests` 행을 만들지 않는다. 정상 상태에서 발급·승인한 같은 요청이 대상 조회 실패로 거부되면 행은 `approved`로 남고 `fakeBitbucket.merged`는 false이며, 조회 복구 후 read-only 제한이 있으면 `BRANCH_RESTRICTED`, 제한이 없으면 같은 승인으로 머지 성공·`consumed`가 된다.
  3) 새 회귀 테스트는 기존 `newFixture`의 실제 Registry→Executor→Resolver→HTTP Bitbucket 클라이언트와 실제 PostgreSQL 승인 행을 사용한다. 기존 PR 버전 변경/인자 변경/동시 소비 테스트를 유지하고, 현재 버전 조회 실패를 거부하는 기존 승인 방어도 독립적으로 계속 검증한다.
- 건드릴 파일:
  - `internal/tools/executor.go:prTargetBranch, Invoke` — helper를 `(string, error)`로 바꾸고 실패를 전달; 대상 ref를 기존 방식으로 정규화한 결과가 비면 오류; 대상 조회를 하는 현재 분기에서 `e.fail(...CodePermissionUnknown...)`으로 반환. 프로덕션 파일 **1개**.
  - `internal/tools/integration_test.go:newFakeBitbucket`, 새 회귀 테스트, `TestMergeApprovalDeniedWhenPullRequestVersionUnknown` — 기존 `prFetchFails` 재사용, PR 응답의 대상 ref를 비우는 작은 스위치만 추가. 승인 버전 회귀의 아래 충돌 처리 필수. 테스트 파일 **1개**.
- 검증 명령 (저장소 루트에서 실행):
  - `TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable' go test ./internal/tools -count=1 -p 1 -v`
  - `go build ./...`
  - `go vet ./...`
  - `TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable' go test ./... -count=1 -p 1 -v`
  - 정찰 baseline: 마지막 세 명령 실제 성공. 전체 테스트 9개 패키지 통과, SKIP/FAIL 없음. 위 DSN의 `bbmcp-test-pg` 실행 확인. 새 결함 재현 테스트는 정찰에서 작성·실행하지 않았으므로 아래 수정 전 결과는 코드 추론이다.
- 위험과 피할 것: `approval.Check`, `prVersion`, `newApprovalRequest`, 정책 평가 순서, 브랜치 패턴/면제 규칙을 함께 바꾸지 않는다. auth·migrations·workflows·웹·의존성은 범위 밖. 승인 방어를 삭제하거나 오류 기대값만 바꿔 과거 회귀 증거를 지우지 않는다. 감사 details에 업스트림 응답/PR 원문/인자/토큰을 추가하지 않는다. 오류 문구는 고정 설명으로 충분하다. DB 테스트는 전용 DB에서 `-p 1`, `t.Parallel` 금지.
- 차선 후보: `Resolver.Reset` 후 권한 캐시 재조회 통합 테스트 (가치 4 / 위험 1 / 작업량 S) — 주 과제 전제가 실제로 성립하지 않을 때만 선택. `newFixture`의 TTL=0을 해당 테스트에서만 60으로 설정하고 프로젝트 권한으로 저장소 조회 성공→권한 회수 후 캐시 사용→Reset 후 PERMISSION_DENIED를 검증한다. `internal/tools/integration_test.go`만 수정하며 관리자 설정 API 배선까지 검증했다고 주장하지 않는다.

확인한 근거와 재현 순서:
- `internal/tools/write.go:executeTools`의 approve/decline/merge는 모두 RiskExecute, `Resolve: resolveRepoPR`; 주석도 승인과 브랜치 검사를 요구한다. `internal/tools/read.go:resolveRepoPR`는 PR ID만 채우고 Branch는 비운다. 따라서 대상 조회 helper는 정상 프로덕션 경로다.
- `Invoke`는 브랜치 검사 뒤 `checkApproval`을 호출한다. 현 코드에서 최초 요청의 PR GET이 503이면 helper와 `prVersion` 모두 빈 값/nil을 반환하고 새 승인이 생성되는 흐름이다. **승인 없이 실제 머지가 성공한다고 단정하지 않는다.** 이미 버전을 고정한 승인은 `approval.Check`가 추가로 막는다.
- 성공 JSON에서 `toRef.id`만 비고 version=14가 정상인 경우에는 그 추가 버전 방어도 대상 브랜치 누락을 감지하지 못한다. 실제 운영에서 이런 응답이 발생했는지는 미확인이다. 불완전 응답을 안전하게 거부해야 한다는 경계 조건으로 테스트한다.
- 공통 준비: `newFixture(t)`; `projectPerm["AI"]="PROJECT_WRITE"`, `repoPerm["AI/text2sql"]="REPO_WRITE"`; `registry.Update(ctx,"bitbucket_merge_pull_request",true,true,tools.RoleExecutor)`; args는 project=AI, repository=text2sql, pullRequest=7. 승인 생성은 실제 Invoke, 결정은 기존 `approvals.Decide`를 사용한다.
- 최초 실패는 503/빈 ref를 각각 테이블로 검증: 기대 오류 + 승인 COUNT(*)=0 + merged=false. 정상 상태에서 승인한 뒤 같은 오류를 주는 경우에도 approved 유지 확인; ref/조회 복구 시 readOnly=true로 거부, false로 바꾸면 같은 approvalId로 성공하는 흐름까지 확인한다. 먼저 최소 재현으로 수정 전 실패를 확보한다.

기존 테스트 충돌 — 반드시 보존할 계약:
- `TestMergeApprovalDeniedWhenPullRequestVersionUnknown`은 **comment가 아니라 merge**를 호출하고 `prFetchFails=true`로 모든 PR 조회를 503으로 만든다. 수정 후 첫 조회(대상 브랜치)에서 먼저 막혀 기존 `APPROVAL_STALE` 기대가 깨지는 것은 예상된 오류 우선순위 변경이다.
- 이 테스트를 단순 삭제/기대값 교체하지 말 것. 기존 nil-current-version 승인 방어는 comment 도구로 옮겨 테스트 이름을 맞춘다: `bitbucket_comment_pull_request`, 같은 args에 `text` 추가, 정상 요청으로 버전 고정 승인→Decide→`prFetchFails=true`→approvalId 재호출→`APPROVAL_STALE`, comments 길이 0, 승인 approved 유지. comment는 RiskWrite이며 Branch가 비어 있어 대상 조회 분기를 지나지 않으므로 `approval.Check`의 기존 방어까지 도달한다.
- merge 자체의 version 증가 방어는 기존 `TestMergeApprovalGoesStaleWhenPullRequestAdvances`를 그대로 둔다. `TestMergeBlockedByBranchRestriction`과 단일 소비 테스트도 유지한다.

실행 계획과 체크포인트 (모든 구현 단계 미착수; 사람 승인 체크포인트 없음):
1. 위 503 최초 요청의 최소 회귀 테스트로 현재 동작 확인. 증명: 첫 번째 검증 명령. 예상된 새 테스트 실패만 기록하고 즉시 2단계로 진행한다. 뜻밖의 원인이면 과제서를 먼저 갱신한다.
2. helper/호출부 수정, 빈 ref와 승인 보존·복구 회귀 추가, 기존 승인 버전 테스트를 위 방식으로 분리. 증명: 첫 번째 검증 명령에서 신규·기존 테스트 모두 PASS, SKIP 0. 이 지점을 재개 가능한 구현 완료 체크포인트로 삼는다.
3. build/vet/전체 테스트로 마감하고 결과·변경 파일을 회차 노트에 남긴다. 증명: 나머지 세 검증 명령. 프로덕션 파일 추가가 필요해지면 범위를 다시 좁힌다.

추정·대안 판단:
- Bottom-up 기준: 최소 재현 5~7분 + 수정 4~6분 + 승인 보존/기존 회귀 분리 12~17분 + 전체 검증/기록 4~5분 = 25~35분. 알려진 변동(대역 응답 스위치·테스트 충돌)에 대한 예비 5~10분을 한 번만 더해 총 30~45분; 신뢰는 중간이며 통계적 분위수는 아니다. 환경/fixture를 재사용한다는 가정이 핵심이다.
- 과거 10/03의 프로덕션 1개+통합 테스트 1개 수정과 구조는 유사하지만 실측 소요시간이 없어 수치 교차 검증은 불가. 관리 예비(새 범위)는 이번 과제에 배정하지 않는다. 추가 결함은 ideas에 남긴다.
- 선택: 대상 조회 오류만 닫기 — 1개 프로덕션 파일, 기존 도구 계약에 근거. 보류: PR 조회·승인·브랜치 정책 전체 통합 — TOCTOU와 캐시 의미까지 넓어짐. 현상 유지/테스트만 추가 — 이미 승인 방어는 있지만 대상 ref 누락 경로가 남는다. 캐시 Reset 테스트 — 더 낮은 위험이나 확인된 검사 생략을 닫는 가치가 더 높다.
- 스킬 적용: 전용 Skill 도구는 노출되지 않아 로컬 원문으로 적용했다: [estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md). 작업 분해·불확실성·예비 분리, 단계별 증명, 대안 수렴을 반영했다.
