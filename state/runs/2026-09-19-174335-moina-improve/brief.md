# 과제서 (2026-09-19, 정찰)

## 먼저 읽을 것 — 우선 과제의 전제 확인

- 배정 사유는 "마지막 회차가 error로 끝남 — `hold: budget`" 입니다. `hold: budget`은 **러너의 USD 예산 상한**으로 회차가 중단됐다는 뜻이지 코드나 워크플로가 실패했다는 신호가 아닙니다(이 정찰 회차도 $2 예산 안에서 돌고 있고, 명령 실행 대부분이 승인 거절돼 검증을 못 했습니다).
- 이 worktree에서 확인한 사실: branch `auto/2026-09-19-1743`, HEAD `e8a4131 release: moina v0.1.32`, `VERSION=v0.1.32`, 작업 트리 clean. `.github/workflows/release.yml`은 tag push 시 Validate → Require CI → Source checks → Build image → Runtime smoke(PostgreSQL + `scripts/qa-api-smoke.sh` + `npm test --prefix e2e`) → Package/verify → Release 순서(파일을 열어 읽음).
- **미확인**: 실제 release 워크플로 run 결과. `gh run list`가 이 세션에서 승인 거절돼 보지 못했습니다. 9/13 회차 기록에서는 v0.1.25~v0.1.28 run이 모두 success였고 그 뒤 v0.1.29~v0.1.32까지 실패 기록이 원장에 없습니다.

## 구현자의 첫 5분 (분기)

1. `gh run list --workflow=release.yml --limit 6 --json conclusion,displayTitle,databaseId` 를 실행하세요.
2. **failure가 있으면** `gh run view <id> --log-failed` 로 실패 단계를 읽고 아래 "과제 A" 로 갑니다.
3. **모두 success이거나 실패가 없으면** "hold: budget"은 러너 예산 중단이었던 것이므로 고칠 워크플로 결함이 없습니다. 원장에 "수정 과제: 재현 불가 — 러너 예산 중단이 error로 기록된 것, release run 전부 success 확인" 이라고 적고 아래 "과제 B" 를 수행하세요.

---

## 과제 A: 릴리즈 워크플로 실패 단계 복구 (가치 5 / 위험 2 / 작업량 S~M) — 실패 run이 있을 때만

- 왜: 릴리즈가 막히면 이후 모든 개선이 배포되지 않습니다. 워크플로를 느슨하게 하지 않고 실패한 스크립트·테스트 쪽을 고쳐야 합니다.
- 가장 가능성 높은 단계(과거 이력 기준): **Runtime smoke with PostgreSQL** — `scripts/qa-api-smoke.sh` 또는 `e2e` 시각 회귀(`e2e/VISUAL_REGRESSION.md` 참고, 베이스라인은 공식 `mcr.microsoft.com/playwright:v1.62.1-noble` renderer + `127.0.0.1:18080` origin에서만 갱신, 부분 갱신은 `MOINA_VISUAL_ONLY=<slug>`). 두 번째는 **Source checks** 의 `MOINA_REQUIRE_SCREENSHOTS=1 node scripts/qa-pages.mjs` / `check-openapi-routes.mjs`(PR #25가 route 2개를 더했음).
- 수용 기준: 1) 실패한 단계의 명령을 로컬에서 같은 인자로 재현해 실패를 먼저 본다 2) 원인을 스크립트/테스트/베이스라인 쪽에서 고친다 3) 같은 명령이 로컬에서 통과하고 `.github/workflows/release.yml`은 diff 0줄.
- 건드릴 파일: 실패 로그가 가리키는 것만. `release.yml`·`ci.yml` 은 읽기 전용.
- 검증 명령: 실패 단계와 동일한 명령(Source checks 6개 명령은 `release.yml:60-65` 그대로; smoke는 `make image` 후 docker PostgreSQL과 함께 `bash scripts/qa-api-smoke.sh http://127.0.0.1:18080 <admin> <pw> v0.1.32` → `npm test --prefix e2e`), 그리고 `go test -race ./...`.
- 위험과 피할 것: 시각 회귀 실패를 로컬 WSL Chromium으로 갱신하지 말 것(폰트가 달라 52장 전부 어긋남 — 9/13 교훈). 워크플로 조건·타임아웃·`--fail` 을 완화하지 말 것.

## 과제 B (차선이자 예산 중단일 때의 본 과제): `updatePost`가 DB 오류를 409 `not_editable`로 감춤 (가치 2 / 위험 1 / 작업량 S)

- 왜: `backend/internal/httpapi/posts.go:831-835` 에서 `UPDATE posts …` 의 `err != nil` 과 `tag.RowsAffected() == 0` 이 한 분기라, DB 연결 끊김·제약 위반 같은 저장 오류가 사용자에게 "본인의 공개 Moin만 수정할 수 있습니다"(409)로 보이고 운영자는 storage 문제를 알 수 없습니다. 같은 파일의 삭제 경로(`posts.go:965-971`)는 이미 `storage_error`(500)와 `not_found`를 분리하고 있어 관례가 명확합니다.
- 수용 기준: 1) `err != nil` 이면 `500 storage_error` ("Moin을 저장할 수 없습니다" 류 기존 문구 재사용) 2) `err == nil && RowsAffected()==0` 이면 지금과 같은 `409 not_editable` 과 같은 메시지(기존 클라이언트·문서 계약 불변) 3) 테스트: 실제 PostgreSQL integration(기존 `posts_media_update_test.go` 의 방식대로 진짜 서버 배선) 에서 (a) 남의 게시글/remoin 수정 → 409 그대로, (b) DB 오류 유도(예: 트랜잭션을 미리 닫거나 `content` 길이 제약을 넘기는 값이 있으면 그것) → 500 `storage_error`. 대역 DB나 소스 문자열 검사로 대신하지 말 것(운영자 지시).
- 건드릴 파일: `backend/internal/httpapi/posts.go:updatePost` (분기 두 개로 나눔) · `backend/internal/httpapi/posts_media_update_test.go` 또는 새 `posts_update_test.go`(integration) · `api/openapi.yaml`의 `PUT/PATCH /posts/{id}` 응답에 500이 이미 있는지 확인, 없으면 추가(`node scripts/check-openapi-routes.mjs` 통과 유지).
- 검증 명령: `cd backend && go test -race ./internal/httpapi/ -run 'Update|Post' -count=1` (PostgreSQL 필요 — throwaway docker `postgres:17-alpine` 로 DSN 환경변수 설정, 방식은 기존 integration 테스트의 skip 조건을 따름) → `go test -race ./...` → `make fmt` → `make check`.
- 위험과 피할 것: 응답 코드가 바뀌는 것은 "DB 오류" 경우뿐이어야 함 — 정상 409 경로의 코드·메시지는 그대로. 프런트 `frontend/API_CONTRACT.md` 에 409만 적혀 있으면 500 한 줄 추가(문서 정본 하나 유지). auth·migrations·workflows 미접촉.

## 차선 후보 (B도 성립하지 않을 때)

- Makefile `test` 를 CI와 같은 `go test -race ./...` 로 맞추기 (가치 2 / 위험 1 / S) — Makefile 한 줄, 검증은 `make test` 실행. 단, "실제 출력·동작이 바뀌지 않는 수정" 지적을 피하려면 `-race` 로 새로 잡히는 것이 없더라도 CI 동등성이 목적임을 커밋 메시지에 적을 것.
