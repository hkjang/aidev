# 과제서 — 2026-10-08-013831-moina-improve

- 과제: 머지되지 못한 지난 두 회차 커밋을 한 브랜치로 복원 — `7b0f70d`(AuthContext.test.tsx 경합 수정 = `98529c0` 재복원) + `238eff1`(social.go `storage_error` 29곳 `writeStorageError` 이관) (가치 5 / 위험 1 / 작업량 S)

- 왜: `origin/main` 은 여전히 `c9bfca2`(v0.1.43)이고, 2026-10-06 두 회차의 결과물이 **둘 다 main 에 없습니다**(실행 확인: `git log --oneline origin/main..auto/2026-10-06-0757` → `238eff1` 한 줄, `origin/main..auto/2026-10-06-0917` → `7b0f70d` 한 줄). `social.go` 이관(`238eff1`)은 자기 결함이 아니라 **프런트 경합**(`AuthContext.test.tsx`, image 잡의 `npm test`)으로 CI 가 두 번 죽어 verify-failed 가 됐고, 그 경합의 고침(`7b0f70d`)은 다음 회차에서 review-pending 으로 끝나 역시 못 들어왔습니다. 둘을 **같은 브랜치**에 담으면 CI 블로커가 먼저 치워진 상태로 이관이 함께 통과해, 두 회차의 작업이 더 이상 유실되지 않습니다.

- 수용 기준:
  1) `git log --oneline origin/main..HEAD` 가 **두 커밋**을 보여주고, 복원된 두 파일 집합이 원본과 **byte-identical** 이다 — `git diff 7b0f70d HEAD -- frontend/src/auth/AuthContext.test.tsx` 와 `git diff 238eff1 HEAD -- backend/internal/httpapi/social.go backend/internal/httpapi/moim_create_postgres_integration_test.go backend/internal/httpapi/moim_join_postgres_integration_test.go` 가 **모두 빈 출력**.
  2) `git diff --stat origin/main..HEAD` 가 정확히 **4개 파일**(`frontend/src/auth/AuthContext.test.tsx` 7+/1-, `backend/internal/httpapi/social.go` ~80줄 수정, `moim_create_postgres_integration_test.go` 신규 188줄, `moim_join_postgres_integration_test.go` 33+) 이고, **보호 경로 0건**(`.github/workflows/*`, `Dockerfile`, `Makefile`, `store/migrations/*`, `VERSION` 이 `git diff --name-only origin/main..HEAD` 에 없을 것).
  3) 응답 불변 증명: `git show origin/main:backend/internal/httpapi/social.go` 와 작업 트리 `social.go` 의 한글 문자열 리터럴 multiset 을 비교해 **origin/main 에만 있는 리터럴 0개**. 늘어난 것은 `blockUser`·`createMoim` 의 합친 조건 분할로 같은 message 가 한 번 더 쓰인 **2개뿐**(2026-10-06 회차 기록과 일치).
  4) `handler` 인자 검증: `^func ` 경계에서 감싼 함수 이름을 **기계적으로 유도**해 `writeStorageError(w, r, "<name>"` 의 세 번째 인자와 비교했을 때 `social.go` 의 **32개** 호출 전부 **mismatch 0**. (내역을 이번 정찰이 `git show 238eff1 -- …social.go` 로 직접 세어 확인했습니다: 치환 29 + 분할로 새로 생긴 2 + v0.1.42 때 이미 이관된 `followTopic` 1 = 32. 커밋의 `handler` 값은 `writeProfile`·`followUser`×4·`unfollowUser`·`blockUser`×4·`unblockUser`·`muteUser`·`unmuteUser`·`listTopics`×2·`unfollowTopic`·`search`·`listNotifications`×2·`readNotifications`·`createMoim`×3·`listMoims`×2·`joinMoim`·`leaveMoim`×2·`uploadMedia`·`deleteMedia`·`createReport` 로 전부 감싼 함수 이름과 일치합니다.)
  5) **실패했던 바로 그 명령** `make image` 가 exit 0 (`Dockerfile:13` 의 `RUN npm test && npm run build` 를 Node 24 컨테이너 안에서 통과). 추가로 호스트에서 `npm test --prefix frontend` 를 **연속 2회 이상** 통과(부하 의존 경합이므로 1회로 판정하지 말 것).
  6) 백엔드: DSN 을 걸고 `go test -race -count=1 -v ./...` 에서 `--- FAIL` 0줄 · `--- SKIP` **0줄**(= DSN 이 실제로 걸렸다는 증거) · 최상위 `TestPostgreSQL*` PASS **45건**(v0.1.43 의 44건 + `238eff1` 의 신규 `TestPostgreSQLCreateMoim…` 1건).
  7) 테스트가 증명해야 하는 것: (a) `joinMoim` 500 경로의 운영자 로그에 `"error_code":"storage_error"`·`"handler":"joinMoim"`·`"cause_type":"*pgconn.PgError"`·`"pg_code":"P0001"` 이 실제로 찍힌다(`moim_join_…_test.go` 의 로그 단언), (b) `createMoim` 의 분할된 조건이 `slug_taken` 409 계약을 바꾸지 않는다(`moim_create_…_test.go` 의 201 / 409 / 500 세 케이스).

- 건드릴 파일 (프로덕션 **1개** + 테스트 3개):
  - `frontend/src/auth/AuthContext.test.tsx` — `7b0f70d` 그대로 복원. 추가되는 것은 `markSignedOut()` 한 줄 + `await waitFor(() => expect(silentSsoAttempted()).toBe(false))` **두 곳**(:71 부근 1단락, :95~100 의 마지막 단락) + 주석 4줄. 프로덕션 `AuthContext.tsx` 는 **건드리지 않는다**.
  - `backend/internal/httpapi/social.go` — `238eff1` 그대로 복원. `writeError(w, http.StatusInternalServerError, "storage_error", …)` **29곳**(현재 main 기준 `grep -c` 로 29 확인)을 `writeStorageError(w, r, "<handler>", err, …)` 로 이관하고, `blockUser`·`createMoim` 의 `Exec 실패 || tx.Commit(...) != nil` **합친 조건 2곳을 분할**(Commit 거절 시 `err==nil` 이라 `cause_type:"<nil>"` 이 찍히는 것을 막기 위함).
  - `backend/internal/httpapi/moim_create_postgres_integration_test.go` — **신규**(188줄). 이 파일이 현재 저장소에 없다는 것을 확인했습니다(`backend/internal/httpapi/moim_*_test.go` 글롭 결과: conversation·join·leave 셋뿐). `createMoim` 의 201 / unique index 위반 409 `slug_taken` / sentinel 트리거 500 세 케이스 + 로그 단언.
  - `backend/internal/httpapi/moim_join_postgres_integration_test.go` — 기존 파일에 33줄 추가(로그 단언 4건: `error_code`·`handler`·`cause_type`·`pg_code`).

- 작업 순서 (중요 — 블로커를 먼저 담아 중단에도 살아남게):
  1. `git cherry-pick 7b0f70d` — 프런트 블로커 먼저. 두 커밋 모두 부모가 `c9bfca2`(= `origin/main`)이고 **파일이 겹치지 않으므로** 충돌 없이 적용된다(확인: `238eff1` 은 backend 3파일, `7b0f70d` 는 frontend 1파일).
  2. 프런트 검증(`npm ci` → `npm run lint` → `npm test` ×2 → `npm run build`) 통과를 본다.
  3. `git cherry-pick 238eff1`.
  4. 백엔드 검증 + `make check` + `make image`.
  - 시간이 모자라면 **1~2 까지만 남겨도 가치가 있다**(CI 블로커 단독 복원). 그 경우 3~4 를 다음 회차 몫으로 노트에 남길 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  # 프런트 (실패했던 그 테스트)
  npm ci --prefix frontend && npm run lint --prefix frontend && npm test --prefix frontend && npm test --prefix frontend && npm run build --prefix frontend
  # 백엔드 (throwaway postgres:16-alpine 를 띄우고 DSN 을 걸 것)
  cd backend && go build ./... && go vet ./... && gofmt -l . && \
    MOINA_TEST_POSTGRES_DSN='postgres://…' go test -race -count=1 -v ./... 2>&1 | grep -E '^--- (FAIL|SKIP)|^=== RUN   TestPostgreSQL'
  # 공용
  make check          # OpenAPI route 120개 유지 확인
  make fmt            # 변경 없음이어야 함
  make image          # ← CI 가 두 번 죽었던 단계. exit 0 을 반드시 볼 것
  ```
  - `npm run lint` 경고 상한 **40**, 직전 값 **39**. 늘어나면 멈출 것.
  - `--- SKIP` 0줄을 눈으로 확인하지 않으면 DB 테스트가 돌았다는 증거가 없다.

- 위험과 피할 것:
  - **같은 접근의 재제출로 보이지 않게**: `238eff1` 자체가 반려된 적은 없다. PR #42 의 CI 실패 원인은 프런트 경합이었고 이번에는 그 고침을 **같은 브랜치에 먼저** 담는다 — 운영자가 금지한 "릴리즈를 깨뜨린 접근을 같은 방식으로 다시 올리는 것" 이 아니다. 커밋 메시지·PR 본문에 이 인과를 적을 것.
  - **보호 경로 금지**: `.github/workflows/*`·`Dockerfile`·`Makefile`·`store/migrations/*`·`VERSION` 을 손대지 말 것. 두 커밋 모두 원래 이것들을 건드리지 않는다.
  - `git stash` 금지(워크트리 공유). 작업을 치워야 하면 임시 WIP 커밋을 쓸 것.
  - 로컬 `main` 은 **낡았다**(`95255f3` = v0.1.28). 판정은 **반드시 `origin/main` 기준**으로, 커밋 포함 여부는 `git merge-base --is-ancestor <sha> origin/main` 로 할 것.
  - 프런트 경합은 **부하 의존**이다. `npm test` 1회 통과를 근거로 삼지 말고 `make image` 까지 볼 것(`Source tests` 잡이 통과해도 image 잡이 같은 테스트로 깨진 전례가 있다).
  - `cherry-pick` 이 아니라 `git checkout <sha> -- <paths>` 로 파일만 가져오면 커밋 메시지(두 회차의 인과 설명)가 사라진다. **cherry-pick 을 쓸 것.**
  - 과거 교훈: 검증 과잉 반복으로 회차가 TIMEOUT 된 적이 2건 있다. 위 명령을 한 번씩(프런트 테스트만 2회) 돌리고 끝낼 것.
  - `social.go` 의 `RowsAffected()==0` 404 4곳 · `IsConflict` 409 · `owner_cannot_leave` 409 · `store.ErrNotFound` 404 분기는 **그대로**. `storage_error.go` 헬퍼도 손대지 말 것.

- 이번 정찰이 **실행으로 확인한 것** (구현자가 다시 안 해도 되는 것):
  - `git log --oneline -3 origin/main` → `c9bfca2` / `5a5494f` / `76010d8`. 즉 `origin/main` = 이번 회차 base = v0.1.43.
  - `git log --oneline origin/main..auto/2026-10-06-0757` → `238eff1` **한 줄**. `…0917` → `7b0f70d` **한 줄**. 둘 다 부모가 `c9bfca2` 이므로 cherry-pick 이 rebase 없이 그대로 적용된다.
  - `git merge-base --is-ancestor 98529c0 origin/main` → **NO**(= 2026-09-29 의 원본 커밋도 여전히 main 에 없다). `7b0f70d` 가 `AuthContext.test.tsx` 에 대해 `98529c0` 과 byte-identical 임은 `git diff 98529c0 7b0f70d --stat` 출력에 그 파일이 **나타나지 않는 것**으로 확인했다.
  - 두 커밋의 파일 집합이 **겹치지 않는다**: `238eff1` = backend 3파일(social.go + moim_create 신규 + moim_join), `7b0f70d` = frontend 1파일. → 순서 무관, 충돌 없음.
  - `grep -c 'writeError(w, http.StatusInternalServerError, "storage_error"'` (현재 트리): `social.go` **29** · `admin.go` 25 · `settings.go` 9 · `workflow.go` 9 · `auth.go` 7 · `preferences.go` 6 · `analytics.go` 4 · `smtp.go` 3 · `outbox.go` 2 · `oidc.go` 1 · `posts.go` **0**(완료). 프로필의 이관 현황과 일치.
- **미확인(추측)**: 이번 정찰은 테스트를 한 번도 돌리지 않았습니다(읽기 예산). 수용 기준 6의 `TestPostgreSQL*` **45건**과 프런트 **36파일 250테스트**는 2026-10-06 두 회차의 원장 기록을 그대로 옮긴 값이며 이번에 재확인하지 않았습니다 — 구현자가 실제 출력으로 확정할 것. `make image` 가 지금 통과한다는 것도 미확인입니다(0917 회차가 통과를 보고했으나 이 워크트리에서 재현하지 않았습니다).

- 차선 후보: **`admin.go` 의 남은 `storage_error` 500 출구 25곳을 `writeStorageError` 로 이관** (가치 3 / 위험 1 / 작업량 M). main 기준 `grep -c` 로 25곳 확인됨. `admin_report_resolve_postgres_integration_test.go` 에 로그 단언 배선이 이미 있어 red→green 수단이 갖춰져 있다. 이관 시 `grep -n 'Commit(r.Context()) != nil'` 로 합친 조건을 먼저 찾아 **반드시 분할**할 것. 단, 1순위를 건너뛰면 이 PR 도 같은 프런트 경합으로 image 잡에서 죽을 수 있으므로, 1순위가 성립하지 않을 때만 고를 것.
