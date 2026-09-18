# 과제서 — 2026-09-19 (Vendra, 수정 과제)

## 먼저 알아야 할 것 — 우선 과제의 전제를 확인한 결과

우선 과제는 「마지막 회차가 error 로 끝났고 릴리즈 워크플로가 같은 이유로 두 번 실패했다」였다. 이 세션에서 확인한 사실:

- 마지막 회차(2026-09-18-161351)의 stages.json 은 `improve: hold — "회차 예산($22)이 오늘 남은 상한을 넘음"` 하나뿐이다. **코드·워크플로가 아니라 러너 예산 상한에서 멈춘 것**이고 저장소를 건드리지 않았다.
- 그 앞 회차 2026-09-17-023301 은 v0.7.54 를 `release: published` 로 끝냈고(release.json `status: released`, `github_release: false` — 러너가 GitHub Release 를 만들지 않는 설정), 2026-09-17-170328 은 PR #125 를 만들고 마이그레이션 018 때문에 `guard: held-expected` 로 사람 대기 중이다. 두 회차 모두 실패 기록이 없다.
- `.github/workflows/release.yml`(태그 push → `scripts/offline-release.sh` → gzip/`docker image inspect` 검증 → softprops/action-gh-release@v2)과 `scripts/offline-release.sh`, `Dockerfile` 을 읽었다. 로컬에서 보이는 결함은 없다. **GitHub Actions 의 실제 실행 결과는 이 세션에서 gh 호출이 거부되어 미확인**이다.

그래서 구현자는 **0단계로 30초만** 써서 실제 상태를 본다:

```
gh run list --repo hkjang/Vendra --workflow=release.yml --limit 4
```

- 최근 두 실행이 **같은 단계에서 실패**해 있으면 → `gh run view <id> --log-failed` 로 원인을 읽고 그 단계(오프라인 스크립트·Dockerfile·release action)를 고친다. 워크플로의 검증 단계(`gzip -t`, `docker image inspect`)를 지우거나 `continue-on-error` 를 다는 것은 금지. 로컬 재현은 `sh scripts/offline-release.sh 0.0.0-local && gzip -t dist/vendra-v0.0.0-local.tar.gz && docker image inspect vendra:v0.0.0-local`(docker build 가 수 분 걸림). 원장에는 '수정 과제' 로 기록.
- 실패가 없거나(v0.7.54 성공) 실패 이유가 GitHub 쪽 일시 장애면 → 고칠 것이 없다. 원장에 「우선 과제의 전제(워크플로 2회 실패)는 성립하지 않음 — 마지막 error 는 예산 hold」라고 한 줄 적고 **아래 1순위 과제를 그대로 수행**한다.

## 1순위 과제

- 과제: 초안 축출이 방금 저장한 초안 자신을 버리지 않게 하기 — putFormDraft 의 상위 50개 선정에 「지금 저장한 키」를 먼저 세운다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/httpapi/productivity.go:623-624` 의 DELETE 가 `ORDER BY updated_at DESC LIMIT 50` 으로 살릴 초안을 고르는데, 방금 INSERT/UPDATE 한 행은 이 목록에 있다는 보장이 없다 — `updated_at` 동률(마이크로초 단위라 드물다)이나 다중 replica 의 시계 차이로 다른 행이 더 「최근」이면 방금 저장한 초안이 지워지면서 응답은 `{"ok":true}` 다. 사용자는 저장됐다고 믿고 떠나고 초안은 없다. 고치면 저장 직후 GET 이 항상 그 초안을 돌려준다.
- 수용 기준:
  1) `PUT /api/v1/me/drafts/{key}` 뒤 같은 키의 `GET /api/v1/me/drafts/{key}` 가 200 으로 방금 보낸 payload 를 돌려준다 — 그 사용자에게 `updated_at` 이 더 미래인 초안이 50개 있어도.
  2) 초안 개수는 여전히 50개를 넘지 않는다(기존 `TestFormDraftsAreBoundedPerUser` — `login_integration_test.go:887-928` — 과 `TestConcurrentDraftSavesKeepTheirOwnDraft` — `concurrency_test.go:277-331` — 이 그대로 통과).
  3) 테스트가 증명할 것: **프로덕션 핸들러와 실제 Postgres** 로 — 시드 50개의 `updated_at` 을 SQL 로 `'2999-01-01'` 같은 미래로 올려 둔 뒤 새 키를 PUT 하고, 그 키가 남아 있고(exists=true) 총 개수가 50 임을 읽는다. 이 테스트는 **고치기 전 코드에서 반드시 실패해야 한다**(방금 저장한 키가 evict 됨) — 먼저 실패를 보고 나서 고칠 것. 가짜 DB·문자열 검사 금지.
- 건드릴 파일:
  - `internal/httpapi/productivity.go:putFormDraft` — 축출 서브쿼리를 `SELECT draft_key FROM user_form_drafts WHERE user_id=$1 ORDER BY (draft_key=$3) DESC, updated_at DESC LIMIT $2` 로 바꾸고 인자에 `key` 를 더한다(주석 한 줄: 방금 저장한 키는 항상 살린다). 대안으로 `AND draft_key<>$3` 은 51개를 남길 수 있으니 쓰지 않는다.
  - `internal/httpapi/login_integration_test.go` — 위 기존 테스트 바로 아래에 새 테스트(예: `TestTheDraftJustSavedSurvivesEvictionWhateverTheClocksSay`). 시드는 기존 방식대로 핸들러 PUT 50회 → `UPDATE user_form_drafts SET updated_at='2999-01-01' WHERE user_id=$1` → 새 키 PUT → GET 200 + count=50. cleanup 은 기존 것과 같이 `DELETE FROM user_form_drafts WHERE user_id=$1`(t.Cleanup 안에서 `context.Background()` 를 쓸 것 — 취소된 ctx 로는 조용히 no-op 된다).
- 검증 명령(CI 와 같은 세 DSN, docker postgres:16-alpine):
  ```
  docker run -d --name vendra-pg -e POSTGRES_PASSWORD=ci -p 5432:5432 postgres:16-alpine
  for d in vendra_ci vendra_ci_migrate vendra_ci_upgrade; do PGPASSWORD=ci psql -h 127.0.0.1 -U postgres -c "CREATE DATABASE $d"; done
  export VENDRA_TEST_DSN=postgres://postgres:ci@127.0.0.1:5432/vendra_ci?sslmode=disable
  export VENDRA_TEST_MIGRATE_DSN=postgres://postgres:ci@127.0.0.1:5432/vendra_ci_migrate?sslmode=disable
  export VENDRA_TEST_UPGRADE_DSN=postgres://postgres:ci@127.0.0.1:5432/vendra_ci_upgrade?sslmode=disable
  gofmt -l internal cmd && go vet ./internal/... ./cmd/... && go test ./internal/... ./cmd/... -count=1
  ```
  (포트 5432 가 이미 쓰이면 `-p 5433:5432` 로 바꾸고 DSN 도 맞춘다. 컨테이너 정리는 `docker rm -f vendra-pg` — 패턴 pkill 금지.)
  프런트는 건드리지 않으므로 web 테스트는 불필요.
- 위험과 피할 것:
  - auth·migrations·workflows 는 건드리지 않는다(마이그레이션 불필요 — 쿼리만 바뀐다). 018 마이그레이션은 PR #125 가 들고 있으니 이 브랜치에서 018 번호를 쓰지 말 것.
  - `maintenance.go` 에도 `user_form_drafts` 를 읽는 자리가 있다 — 같은 값을 읽는 다른 경로가 축출 규칙을 복제하고 있는지 열어 보고, 있다면 같은 규칙으로 맞춘다(한쪽만 고치지 말 것). 미확인.
  - 2026-09-10 에 반려된 통화 검증 접근(2e1abb1)과 무관한 자리다.
  - 러너의 비밀정보 검사: 테스트에 12자 이상 리터럴을 password/secret/token 옆에 두지 말 것(기존 `testAdminPassword` 상수만 쓴다).
- 차선 후보: 「CI 의 go job 이 `./cmd/...` 를 빼놓아 Makefile·README 와 불일치」— `.github/workflows/ci.yml` 의 `go test ./internal/... -count=1` 을 `./internal/... ./cmd/...` 로(워크플로를 더 엄격하게 만드는 방향이라 허용). 단 cmd 에는 아직 테스트 파일이 없어(`ls cmd/vendra/*_test.go` 없음) 지금 효과는 「컴파일 검사」뿐이므로 1순위가 성립하지 않을 때만.
