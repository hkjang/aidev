# 회차 노트 2026-10-01-114222-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:42] base pinned — main@d589476
- [러너 11:42] autonomy release — 

## 정찰 노트
- `admin_users.go:72` 를 골랐다. 묶인 `err!=nil||RowsAffected()==0` 이 10곳 남았는데 HTTP 입력으로 DB 오류를 만들 수 있는(= 사용자 text 가 문장에 들어가는) 곳은 여기뿐이다 — events:104·organizations:256·coupons:350 은 파라미터가 uuid·선검증 enum 뿐이라 증거를 못 만든다. 같은 파일 `resetAdminUserMFA` 가 이미 500/404 로 갈라져 있어 비대칭 근거도 파일 안에 있다.
- 실제로 열어 확인: admin_users.go 40-90·140-175, organizations.go 215-275, events.go 75-115, router.go 145-150, 001_initial.sql 22·515-528, openapi.yaml 997-1020, integration_test.go 91-96·544-553·2229·2241-2250.
- 추측으로 남긴 것: `display_name` 의 NUL 이 22021 을 만든다는 것은 webhooks.name·portfolios.title 실측의 유추다(이 컬럼으로 직접 확인하지 않음). `docker` 가용성도 재확인 안 했다 — 둘 중 하나가 어긋나면 차선(events.go:104, 증거는 404 회귀까지)으로 가라.
- 조심할 것: `operator` 역할에는 `users.manage` 가 없다 — 테스트에서 `super_admin` 을 부여해야 403 이 아니다. 성공 경로의 세션 폐기·통지·audit 는 손대지 말고 오류 분류 한 줄만 가를 것.
- 프로필은 3일 전 것이 지금 코드와 맞아 다시 쓰지 않았다. 다만 프로필의 "묶인 곳 26곳" 은 이미 갈라진 단독 분기까지 센 수치이고 실제로 묶인 곳은 10곳이다 — ideas.json 에 바로잡아 적었다.
- [러너 11:46] scout done — 관리자 사용자 수정(PATCH /admin/users/{id})의 DB 저장 실패를 404 "사용자를 찾을 수 없습니다." 가 아니라 500 으

## 구현 노트
- `updateAdminUser`(admin_users.go:71-)의 `err != nil || tag.RowsAffected() == 0` 을 500 `user_save_failed` / 404 `user_not_found` 로 갈랐다. 같은 파일 `resetAdminUserMFA` 의 형태를 그대로 따랐고 성공 경로(세션 폐기·`AccountReactivated`·`s.audit`)는 건드리지 않았다. 프로덕션 1 + 테스트 1 + 문서 1 = 3파일.
- 확신 없는 곳: 없음에 가깝다. 실패 경로의 오류가 `*pgconn.PgError{Code:"22021"}` 이고 그 행이 실제로 존재한다(`row_exists=true`, `rows=-1`)는 것을 임시 프로브로 눈으로 확인한 뒤 프로브와 그에 딸린 `fmt`·`os` import 를 모두 제거했다(최종 diff 로 확인). 두 방향 변이(항상 500 / 항상 404=원본)로 테스트가 각각 실패하는 것도 확인.
- 검증 못 한 것 하나: `npm --prefix web test` 가 이 환경(node 22.23.1)에서 `Could not find .../web/src/**/*.test.ts` 로 **0건** 끝난다. 내 변경과 무관한 기존 결함이다 — `web/package.json` 은 HEAD 와 동일(`git diff --name-only` 에 없음)하고 유일한 테스트 파일이 `src/silentSso.test.ts` 라 `**` 가 디렉터리 0개를 못 매치한다. `node --test web/src/silentSso.test.ts` 로 직접 주면 **10건 전부 통과**하므로 프런트 회귀는 없다. ideas.json 에 후보로 적었다.
- 일부러 하지 않은 것: `admin_users.go` 의 다른 핸들러(`updateAdminUserRoles`·`resetAdminUserMFA`), `web/`·`internal/ui/dist`(프런트는 `error.message` 를 그대로 띄우므로 서버만 고치면 안내가 바뀐다), 그리고 과제서가 범위 밖으로 지정한 `s.audit` 의 `display_name` 원문 전달 — 발견으로만 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: 새 테스트 `TestIntegrationAdminUserUpdateSeparatesSaveFailureFromNotFound` 는 **`KKIIT_TEST_DSN` 이 있어야 돈다**(없으면 SKIP 이고 그건 검증이 아니다). `t.Parallel()` 을 붙이지 말 것(전역 `apiUnderTest`). 검증에 쓴 폐기 컨테이너는 `kkiit-it-pg`(127.0.0.1:55471) — 호스트의 55439 는 **다른 세션 컨테이너가 점유**하고 있어 포트를 바꿨으니 그 컨테이너를 건드리지 말 것.
- [러너 11:53] brief accepted — 채택 — docker(29.7.2)가 가용해 수용 기준 1~4 를 모두 실제 HTTP→실제 DB 로 증명했고, 과제서가 미확인으로 남긴 두 가지(`us
- [러너 11:54] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve(위험 low, 차단 없음). 구현자 컨테이너가 사라져 있어 폐기 postgres:16 을 직접 띄워 재검증했다 — 새 테스트 단독 `--- PASS`(SKIP 아님), 전체 `go test ./cmd/... ./internal/...` 통과(httpapi 84.6초 = 통합 실제 실행), vet·gofmt 무결. 러너의 `verify.json` 은 DSN 없이 2초에 끝난 `go test` 라 통합을 증명하지 않으니 다음 회차는 그 게이트를 근거로 삼지 말 것.
- 테스트 판별력은 확정이다: `user_save_failed` 가 main 에 없는 문자열이라(git grep) 수정 전 코드에서 통과할 수 없다. 원장의 `- 실패 재현:` 출력도 이번 증상(404 want=500)과 일치한다.
- 승인이어도 남는 우려 셋: ① 새 500 분기가 원인 오류를 로그하지 않아(admin_users.go:75-78) 500 을 받아도 서버에 단서가 없다 — 베낀 `resetAdminUserMFA` 도 같아 파일 안에서는 일관하지만 다음 회차 1순위 후보. ② NUL 바이트는 본래 클라이언트 오류라 400 이 더 맞다(`display_name` 에 길이·CHECK 제약 없음). ③ openapi 의 같은 블록 `get /admin/users/{id}` 는 아직 500 미기재.
- 못 본 것: 관리자 UI(`web/`) 배선은 읽지 않았다 — 프런트가 `error.message` 를 그대로 띄우고 diff 가 `web/`·`dist` 를 건드리지 않아 서버만으로 안내가 바뀐다는 전제를 코드로 확인하지 않고 받아들였다. 프런트 테스트도 0건으로 끝나는 것을 확인만 했다(기존 결함, 이번 변경과 무관).
- **릴리즈가 조심할 것**: worktree 에 verify 의 `npm run build` 가 남긴 `internal/ui/dist` 재빌드 산출물이 커밋되지 않은 채 있다. 이번 커밋은 3파일로 깨끗하니 `git add -A` 로 그 dist 를 쓸어 담지 말 것.
- [러너 11:59] review approved — 리뷰 승인 (risk=low)
- [러너 11:59] pr created — https://github.com/hkjang/Kkiit/pull/16
