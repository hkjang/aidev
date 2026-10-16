# PR 처리기 노트 2026-10-16-090557-ai-admin-shepherd — ai-admin PR #46
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-16-083035-ai-admin-improve)
# 회차 노트 2026-10-16-083035-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:30] base pinned — main@2b2ef67
- [러너 08:30] autonomy release — 

## 정찰 노트
- 직전 회차가 lint 세 명령과 `go test -race`(CI 와 같은 두 환경변수·전용 PG+Keycloak)를 이미 통과시켰으므로 같은 재현을 다시 돌리는 것은 뺐다. 남은 유일한 in-repo 재현 후보는 CI test job 의 뒤쪽 두 단계(`ci.yml:60-64` 의 web npm 3단계·`docker build`)이고, 이 저장소의 어떤 회차 기록에도 로컬 실행 흔적이 없어 이것을 1순위로 골랐다. 승인 payload NUL·errKeyStale 등은 PR #45 가 건드린 `workflow.go` 와 충돌해 제외했다.
- 이번 회차에 실제로 읽은 것: `.github/workflows/ci.yml`·`release.yml`, `scripts/verify-version.sh`(전문), `scripts/package-offline.sh`, `scripts/verify-offline.sh`, `Dockerfile`, `Makefile`, `go.mod` 머리, 브랜치/diff 통계. 테스트·빌드는 **아무것도 실행하지 않았다**(이 환경에서 `go`·`gh`·임시 디렉터리 명령이 권한 거부됨 — `gh` 로 PR #45 체크런을 읽는 것도 불가).
- 과제서에서 추측인 부분: ② web npm·③ docker build 가 실제로 실패하는지는 **미확인**(돌려 본 적 없음이 근거이고 실패 자체는 가설). `Dockerfile:36-49` 의 라이선스 수집 루프가 결정적 실패 지점이라는 것도 코드 읽기 기반 추정. CI 2 vCPU 타임아웃 가설은 순차 DB 작업이 지배적이라 약하다고 판단해 내렸다(미측정).
- 구현자가 조심할 것: 워크플로 파일 수정 금지, `npm ci` 가 `web/package-lock.json` 의 `version`(1.2.37)을 바꾸면 `verify-version.sh:61-79` 로 lint 가 깨진다, `internal/ui/dist` 를 재생성하지 말 것, 문서에 새 `vX.Y.Z` 토큰 금지.
- ②③ 가 전부 통과해도 `make ci-test` 는 반드시 남겨 다음 회차가 한 명령으로 CI test job 을 재현하게 할 것 — 두 회차 연속 "재현 못 함" 으로 끝난 근본 원인이 재현 수단의 부재다.
- [러너 08:37] scout done — CI test job 중 지금까지 한 번도 로컬에서 실행된 적 없는 단계(`web` 의 `npm ci && npm test && npm run build`, `docker bu

## 구현 노트
- 무엇/왜: CI test job 네 단계를 전부 로컬 실행해 PR #45 실패 원인을 좁히려 했고, 네 단계가 **모두 exit 0** 이라 원인 파일을 고치지 않았다. 대신 다음 회차가 같은 자리에서 헤매지 않도록 `Makefile` 에 `ci-test`(CI 와 같은 순서·플래그 + `TEST_POSTGRES_DSN` 없으면 시작 전 중단)와 `docs/operations.md` 한 절만 추가했다. 프로덕션 Go/웹 코드는 0파일.
- 확신 없는 곳·검증 못 한 것: **PR #44·#45 의 CI 실패 원인은 여전히 미확인**이다. 이번에 배제한 것 — test job 네 단계 전부(①②③④ exit 0), `Dockerfile:36-49` 라이선스 루프(통과), 툴체인 불일치(`go.mod` go 1.26.0/toolchain go1.26.6 vs CI 핀 1.26.6 일치), node 메이저 차이(로컬 v22·docker 내 26 둘 다 통과). 남는 자리는 두 job 이 공유하는 `actions/checkout`·`actions/setup-go`(`cache: true`)이고 private·`gh` 미인증으로 로그를 못 읽어 **추측으로 고치지 않았다**. `make ci-test` 를 CI 러너(2 vCPU)에서 돌려 본 것도 아니다 — 32코어에서 2m48.992s 였다.
- 일부러 하지 않은 것: `.github/workflows/` 전부 무수정(완화 금지). `web/*`·`Dockerfile`·`go.mod`/`go.sum` 무수정(②③ 가 통과했으므로 고칠 근거가 없다). `internal/ui/dist` 재생성 안 함(② 는 `web/dist` 만 만든다). `workflow.go`·`docs/api.md` 무수정(PR #45 와 충돌). 문서에 새 `vX.Y.Z` 토큰 안 넣음(`verify-version.sh` 가 깨진다). `web/package.json` 에 `engines` 추가 안 함(로컬 `npm ci` 를 막을 위험).
- 다음 역할이 조심할 것: `make ci-test` 는 **Docker 와 전용 폐기 PostgreSQL 이 있어야** 돈다 — 통합 테스트가 `ai_admin`·`aiportal` schema 를 DROP CASCADE 하므로 공유/운영 DSN 을 절대 넘기지 말 것. `TEST_POSTGRES_DSN` 없이 돌리면 exit 2 로 **의도적으로** 실패한다(조용한 SKIP 방지 가드이니 버그로 보지 말 것). 실행 로그 원문은 이 런 디렉터리의 `logs/` 에 있다(저장소의 `logs/` 는 `.gitignore` 에 없어 커밋 전 삭제했다).
- [러너 08:45] brief accepted — 채택 — 과제서가 지정한 ①②③④⑤ 를 모두 그대로 돌렸고, PR #45 의 diff 범위가 `web/`·`Dockerfile`·`go.mod` 를 건드리지 않�
- [러너 08:46] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것(직접 실행): `make -n ci-test` 전개가 `ci.yml:80·87·89` 와 플래그·태그(`v1.2.37`/`ai-admin:ci`)까지 일치, DSN 없는 가드 경로 재현(`Error 1` → make exit 2, `go test` 미실행, DSN 값 미출력), `make lint` exit 0(`1.2.37`), `docs/operations.md` 의 vX.Y.Z 토큰은 v1.2.37 단독(verify-version.sh:108-121 안전), `git diff --check` 통과, 커밋 변경은 Makefile·docs 2파일뿐(프로덕션 0파일). 못 본 것: 4단계 전체 실행(전용 폐기 PostgreSQL 필요) — `logs/ci-test-make.log`(internal/server 153.105s, `cached` 0건, exit 0)로 대신 검증했고 2 vCPU 러너에서의 소요는 여전히 미측정.
- 판정 approve / risk low / blocking 없음 — 신규 엔드포인트·인증/인가·암호·마이그레이션·의존성·개인정보 변경이 0이고, 순수 가산이라 revert 로 완전 복구된다. 보안·법무 차단 사유 없음.
- 남는 우려 1(다음 회차용): `Makefile:36-38` 의 비대칭 — DSN 은 hard fail인데 `TEST_KEYCLOAK_ISSUER` 는 경고만이다. `make ci-test` 의 exit 0 은 CI test job 전체가 아니라 **Keycloak 기동 단계(ci.yml:59-78)와 OIDC E2E 1건을 뺀 것**이니 "CI 완전 일치" 로 읽지 말 것(주석·경고·docs 세 곳에 공개돼 있어 거절 사유는 아님).
- 남는 우려 2: `Makefile:39` 에 `-count=1` 이 없다(CI 플래그 일치를 위한 의도적 선택). 같은 DSN 으로 연달아 돌리면 go test 캐시가 `(cached)` 를 반환해 통합 테스트가 실제로 안 돌 수 있고, 이 경우는 가드가 못 막는다.
- 릴리즈 노트에 반드시: 이번 회차는 PR #44·#45 의 CI 실패를 **고치지도 재현하지도 못했다**. 산출물은 재현 수단(`make ci-test`)+문서 1절뿐. in-repo 재현 후보는 소진됐고 다음 단계는 인증된 `gh` 로 체크런 로그를 읽는 것(이 세션에는 GitHub 권한 없음). 부작용은 로컬 `ai-admin:ci` image·`web/dist`(.gitignore:4) 뿐으로 무해.
- [러너 08:49] review approved — 리뷰 승인 (risk=low)
- [러너 08:50] pr created — https://github.com/hkjang/ai-admin/pull/46
- [러너 08:50] ci failed — 성공이 아닌 검사: lint=failure, test=failure · 실패한 검사: ? 잡: test 

## 수리 노트
- 지적이 **틀렸다**(코드 결함 아님): run 38825920638 양쪽 job 주석이 `The job was not started because recent account payments have failed or your spending limit needs to be increased.` 다. 러너가 시작조차 안 해 lint·test 가 3초에 failure 가 됐고, PR #44(38779853854)·#45(38809887099)도 같은 주석이다. 마지막 성공은 과금 문제 전인 2026-10-08. 비평 노트가 "다음 단계는 인증된 `gh`" 라고 적은 그대로, `GH_CONFIG_DIR=/home/hkjang/.config/gh` 로 체크런을 읽어 확정했다.
- 맞았던 것: 앞선 두 회차가 in-repo 원인을 못 찾은 것은 정확했다 — 원인이 저장소 밖이었다. `make ci-test` 추가는 결과적으로 올바른 산출물이었다.
- 고친 방법: **없음, 커밋 0건**. 고칠 코드가 없고 가능한 "수정" 은 `.github/workflows` 완화뿐인데 금지이며 러너 미시작이라 효과도 없다. 해결은 계정 Billing & plans 정리 후 CI 재실행(저장소 밖 작업).
- 검증: `make lint` exit 0, 전용 폐기 PG(포트 55617, 끝나고 제거)로 `make ci-test` **exit 0** — go test -race 전체 `(cached)` 0건·`internal/server` 152.092s, vitest 18파일/81건, `npm run build`, `docker build --pull` 네 단계 전부. 비평 노트 "우려 2"(캐시로 통합 테스트 건너뛰기)는 이번 실행에서 `(cached)` 0건으로 발생하지 않음을 확인.
- 확신 없는 곳: CI 가 실제로 돌았을 때 2 vCPU 에서 통과할지는 여전히 미측정(여기선 32코어 2m50s 급). 과금 해소 시점은 내 통제 밖이므로 이 PR 의 체크는 사람이 결제를 고치기 전까지 계속 failure 로 남는다 — 중재자는 이것을 코드 문제로 재해석하지 말 것.
