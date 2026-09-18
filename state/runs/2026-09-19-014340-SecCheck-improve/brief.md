# 과제서 (2026-09-19, 정찰)

## 먼저 알아야 할 사실 (정찰이 확인한 것)

- 마지막 회차 `2026-09-18-161346-SecCheck-improve` 의 `error` 는 **저장소 실패가 아니다**. `run.json` 은 시작 16:13:46 → 종료 16:13:47 (1초), `stages.json` 은 `improve: hold — "회차 예산($22)이 오늘 남은 상한을 넘음"`. 즉 러너의 일일 예산 상한에 걸려 아무 것도 실행되지 않았다. 코드·워크플로·테스트 어느 것도 이 오류의 원인이 아니다.
- 그 앞 회차 `2026-09-17-162321` 은 검증 7개 통과 후 PR #7(MCP OAuth 캠페인, 커밋 a6bcb07, 브랜치 `auto/2026-09-17-1623`)을 만들고 `guard: held-expected`(auth·migrations 보호 경로) 로 사람 심사 대기 중이다. main(`eb2a3b0`)에는 아직 없다.
- 마지막 릴리즈 v1.0.146(`eb2a3b0`)의 security-ci 는 `ci-eb2a3b044bef.json` 기준 test-build-scan·build·deploy 모두 `success`. 러너 release.json 은 `status: released, github_release: false, assets: []` — 러너가 태그만 밀고 GitHub Release 는 release.yml 이 만드는 구조라 이 값만으로 워크플로 실패를 단정할 수 없다.
- **미확인**: GitHub 쪽 release.yml 실행 결과. 이 정찰 환경에서는 `gh run list`·`curl api.github.com` 둘 다 승인 거부로 막혀 원격 로그를 볼 수 없었다. "같은 이유로 두 번 실패" 라는 자동 배정 문구의 근거는 로컬 상태 파일 어디에도 없다.

## 과제: 릴리즈 게이트(release.yml)를 main@eb2a3b0 에서 로컬로 전부 재현해 실제로 깨진 단계가 있으면 고치고, 없으면 '예산 홀드 오탐' 으로 원장에 닫기 (가치 4 / 위험 1 / 작업량 S~M)

- 왜: 자동 배정은 "릴리즈 워크플로 두 번 실패" 를 전제하지만 로컬 증거는 예산 홀드뿐이다. 실제로 깨졌는지 판정하려면 release.yml 의 "Build and test image" 와 "Verify the built image answers" 단계를 같은 명령으로 돌려 봐야 하고, 깨진 곳이 있으면 그것이 이번 수정 과제다. 워크플로를 느슨하게 하는 것은 금지.
- 수용 기준:
  1) `gh run list --workflow=release.yml --repo hkjang/SecCheck --limit 5 --json status,conclusion,headBranch,createdAt` 을 먼저 실행해 v1.0.145·v1.0.146 의 결론을 확인하고 결과를 원장 요약에 그대로 적는다(정찰은 못 봤다). 실패했다면 `gh run view <id> --log-failed` 로 실패 단계를 읽는다.
  2) release.yml 의 순서대로 로컬에서 돌린다: 임시 Postgres 컨테이너(`docker run -d --name seccheck-pg -e POSTGRES_PASSWORD=seccheck_ci -e POSTGRES_DB=seccheck -p 55432:5432 postgres:16-alpine`) → `TEST_POSTGRES_DSN=postgres://postgres:seccheck_ci@127.0.0.1:55432/seccheck?sslmode=disable go test ./...` → `npm ci --prefix web --no-audit --no-fund && npm --prefix web run build` → `docker build --build-arg VERSION=1.0.146 -t seccheck:v1.0.146 .` → 이미지를 별도 DB 로 띄워 `/app/seccheck healthcheck` 와 `/app/seccheck selftest --username selftest --full`(release.yml 116~139행 그대로, BOOTSTRAP_ADMIN·ENCRYPTION_KEY 환경변수 포함) → 가능하면 `trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 seccheck:v1.0.146`(trivy 가 없으면 `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:latest image …`).
  3) 어느 단계가 실패하면 그 단계의 **원인**을 고친다 — 예: trivy 게이트면 2026-09-13 교훈대로 Dockerfile `apt-get upgrade`·의존성 버전 올리기(`go get golang.org/x/…@latest` 등), selftest 면 해당 명령 코드(`cmd/`·`internal/`)의 버그. 고친 뒤 같은 단계를 다시 돌려 통과를 보인다. 워크플로의 `exit-code`·`severity`·`ignore-unfixed`·`--full` 을 바꾸는 것은 금지.
  4) 모든 단계가 통과하면 코드를 바꾸지 않고, 원장에 "수정 과제 — 릴리즈 게이트 로컬 재현 전부 통과, 09-18 error 는 러너 일일 예산 홀드(stages.json) 였음" 으로 기록한 뒤 아래 **차선 후보** 를 그 회차에 이어서 한다.
  5) 마지막에 `bash scripts/precheck.sh` 전체 통과(gofmt·go vet·go test·tsc+vitest+vite build·가이드 그림·PDF 신선도·gitleaks).
- 건드릴 파일 (실패 단계에 따라서만): `Dockerfile`(베이스 이미지 패키지 업그레이드), `go.mod`/`go.sum`(취약 의존성), `scripts/package-image.sh`(패키징 단계), `cmd/seccheck`·`internal/` 의 selftest·healthcheck 경로. **`.github/workflows/release.yml`·`ci.yml` 은 읽기만 하고 고치지 말 것.**
- 검증 명령: 위 2)의 순서 그대로 + `bash scripts/precheck.sh`. `go test ./...` 는 DSN 이 있으면 약 3분(이전 회차 166초).
- 위험과 피할 것:
  - PR #7(`auto/2026-09-17-1623`, a6bcb07) 은 사람 심사 대기 중이다. 그 브랜치 위에서 작업하거나 그 커밋을 다시 만들지 말 것 — main@eb2a3b0 기준으로만.
  - `pkill -f <경로>` 는 자기 셸까지 죽인다(종료 코드 144) — 컨테이너는 이름으로, 프로세스는 PID 로 정리.
  - 파괴적 확인(`git checkout -- …`) 전에는 반드시 `git add -A && git commit` 으로 WIP 커밋(세 회차 연속 교훈).
  - `TestVerifyingTheSchemaReportsWhatTheBuildIsMissing` 은 PR #4(`fix/scratch-schema-identifier`) 로 이미 main 에 고쳐져 있다 — 실패하면 그것은 새 문제다.
  - 릴리즈 컨테이너는 `--rm` 없이 뜨므로 끝나면 `docker rm -f seccheck-release-check seccheck-pg` 와 임시 이미지 삭제.
- 차선 후보: **메일 탭 테스트 발송에 받는 주소 입력 두기** (가치 2 / 위험 1 / S) — `POST /admin/settings/mail/test` 는 `recipient` 를 받는데 `web/src/pages/Settings.tsx` 는 본인 프로필 주소로만 보내 이메일 없는 관리자가 422 를 본다. 입력란(기본값 본인 주소)을 두고 본문에 `recipient` 를 실으면 된다. `web/src/lib/payloads.test.ts` 가 리터럴 본문 키를 `internal/web/payloads.go` 표와 대조하므로 그 표에 이미 `recipient` 가 있는지 먼저 확인하고, ADMIN_GUIDE 3-5 의 "테스트 메일 보내기" 문장과 features.md 18절이 동작을 서술하면 함께 고친 뒤 PDF 재생성. 검증: `cd web && npx tsc --noEmit && npm test --silent && npm run build`, `go test ./internal/web/...`.
