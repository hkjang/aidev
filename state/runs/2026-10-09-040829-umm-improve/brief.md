- 과제: 수정 과제 — CI Go vulnerability scan을 막는 x/text GO-2026-6629 해소 (가치 5 / 위험 1 / 작업량 S)
- 왜: PR #170의 CI는 `golang.org/x/text v0.40.0`의 GO-2026-6629를 `Store.beginExternalLease → pgx.ConnectConfig → precis.Profile.String` 경로에서 검출하며, 현재 main@e54875c도 같은 명령으로 실패한다. 수정 버전 v0.41.0만 반영하고 웹 의존성을 올바른 위치에 설치한 뒤 검증하면 보안 게이트와 후속 러너의 서로 다른 실패를 함께 처리할 수 있다.
- 수용 기준: 1) 현재 베이스에서 같은 GO-2026-6629/호출 경로로 실패한 기록과 수정 후 `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` exit 0 기록을 남긴다. 2) 저장소 변경은 go.mod/go.sum의 x/text v0.40.0→v0.41.0 및 해당 체크섬에 한정한다; 기존 PR 웹 변경이 있는 트리면 그대로 보존한다. 3) 기존 Go 시험·vet·build·모듈 검증과 설치 후 웹 검증이 통과하고, 원장에 '수정 과제'로 기록한다; DB 없이 SKIP한 시험을 DB 통과로 기록하지 않는다.
- 건드릴 파일: `go.mod:require` — indirect `golang.org/x/text` 한 줄을 v0.41.0으로; `go.sum` — x/text 본체와 go.mod 체크섬 두 항목 갱신. 앱 코드/신규 시험 파일 0개, 의존성 파일 2개.
- 검증 명령: 아래 실행 순서의 명령을 저장소 루트에서 실행. `npm ci --prefix web`를 생략하거나 루트 `npm ci`로 대체하지 않는다.
- 위험과 피할 것: `.github/workflows/*`, Makefile, Dockerfile, Go toolchain, VERSION, 웹 훅/가드, auth, migrations 수정 금지. `go get -u ./...`·대규모 tidy·스캔 제외·취약점 무시·continue-on-error 금지. PR #170의 dev/preview/e2e 진단을 다시 구현하거나 make test-go 확장 실패 접근을 다시 올리지 않는다. 수정 버전이 v0.x minor이지만 모듈 경로 변경/메이저 업그레이드는 없으며 기존 시험으로 호환성을 확인한다.
- 차선 후보: 같은 취약점 수리가 구현 시작 트리에 이미 있으면 같은 게이트와 설치 후 검증으로 수리 상태를 확인하고 원장만 보강한다. 다른 취약점으로 명령이 실패하면 식별자·호출 경로를 기록해 이 과제서를 갱신하고, 관련 없는 개선으로 대체하지 않는다.

근거와 실패 구분:
- 확인한 CI: `../2026-10-09-015837-umm-improve/ci-c71970003d48.json` — PR #170/head c719700, run 37814547052/job 113439682476, verify=failure. `ci-failure-c71970003d48.txt`의 실패 단계는 `.github/workflows/ci.yml`의 `Go vulnerability scan`이다. 이 단계는 별도 저장소 스크립트가 아니라 govulncheck CLI를 직접 실행한다.
- `.github/workflows/release.yml`은 버전→Docker image→SBOM→attestation→publish 경로이고 govulncheck/npm test가 없다. 따라서 '릴리즈 워크플로가 같은 이유로 두 번 실패'라는 문구를 실제 release.yml 실패 두 건으로 확정하면 안 된다. GitHub CLI 미인증으로 현재 원격 상태/두 번째 원격 실패는 미확인.
- 현재 `internal/store/store.go:221`의 `beginExternalLease`가 실제 pgx.ConnectConfig 호출. `internal/store/brief_integration_test.go:briefUser`는 POSTGRES_DSN이 없으면 SKIP한다. 새 가짜 PRECIS 시험 대신 기존 보안 스캔과 실제 모듈 그래프를 회귀 검증으로 사용한다.
- `../2026-10-09-034621-umm-shepherd/fix-summary.md`와 로컬 커밋 `8e877bf`는 이미 같은 두 파일 수리 및 격리 DB 검증을 담고 있으나 push 없음. 현재 main에는 없음을 go.mod와 diff로 확인했다. 이 수리는 반려/회귀된 접근이 아니다. 해당 커밋 전체 트리에는 PR 웹 변경도 있으므로 현재 베이스에 전체 트리를 덮어쓰지 말 것.
- 후속 `034621/verify.json`은 Go 검증 후 `[ -d node_modules ] || npm ci --no-audit --no-fund` 다음 `npm test --silent` 실패. 루트 package.json은 `npm --prefix web test` 위임만 하며 루트 설치로 web 의존성을 마련할 수 없다. 현재 새 checkout에서도 `npm test --silent`가 vitest missing install/exit 1로 재현됐다. 가드·러너를 저장소 코드로 우회할 문제가 아니다.
- 공식 권고: https://pkg.go.dev/vuln/GO-2026-6629 — 2026-10-07 공개, x/text/secure/precis v0.41.0 미만 영향. 스캐너 호출 경로가 사용자 입력으로 실제 악용 가능함까지 증명하는 것은 아니다.

실행 계획 (모든 단계 pending; 사람 승인 체크포인트 없음, 증거가 예상과 다르면 진행 전 과제서를 갱신):
1. 시작점 확인: `git status --short`, `git log -1 --oneline`, `go list -m golang.org/x/text`, `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`. 현재 main에서는 v0.40.0/GO-2026-6629/exit 1이어야 한다. 이미 v0.41.0이면 차선 조건을 적용한다.
2. 좁은 수리: `git show 8e877bf -- go.mod go.sum`로 기존 최소 diff를 확인하고 이 두 파일의 x/text 변경만 반영한다(또는 `go get golang.org/x/text@v0.41.0` 후 두 파일의 diff 확인). 증명: `git diff -- go.mod go.sum`, `go list -m golang.org/x/text`, `go mod verify`, `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`. 다른 의존성이 움직이면 원인을 확인하고 범위를 좁힌다.
3. Go 회귀: `go vet ./...`, `go test -p 1 ./... -count=1`, `go test -race -count=1 ./internal/intelligence`, `go build ./...`, `./scripts/check-version.sh`, `git diff --check`. 사용 가능한 격리 PostgreSQL 17 DSN이 있으면 전체 시험에 지정하고 `go test ./internal/store -run 'Test.*Integration' -count=1 -v`로 SKIP 아닌 PASS 확인. 운영 DB를 쓰지 말 것. 격리 DB 준비가 안 되면 그 한계를 원장에 기록한다.
4. 반복된 로컬 검증 실패 마무리: `npm ci --prefix web`, `npm test --silent`, `make test-web`. 루트 node_modules의 존재는 설치 증거가 아니다. npm ci가 manifest/lock을 바꾸면 diff를 확인하고 이번 변경에 섞지 않는다. 외부 러너 영구 수정은 이 저장소 범위 밖이므로 필요한 선행 명령을 원장에 명시한다.
5. 원장/검토: 이번 회차 `ledger-entry.md`와 `journal.md`에 '수정 과제', 변경 두 파일, 보안 실패→통과, 웹 설치 선행, DB 실행 여부를 적는다. GitHub CI/전체 이미지 릴리즈 완료는 실제로 수행한 경우에만 적는다. 빌드 이미지 검증이 필요하면 기존 `docker build --build-arg VERSION=0.76.8 .`을 사용하며 워크플로는 바꾸지 않는다.

정찰에서 직접 확인한 증거:
- 원본 저장소 `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`: exit 1, GO-2026-6629/v0.40.0/store.go:221. `scout-vuln-base.log`.
- 원본을 수정하지 않고 이 회차 디렉터리 `validated-snapshot/`에 기존 8e877bf의 Go 소스·migrations·go.mod/go.sum을 그대로 추출했다. `git diff --name-only e54875c 8e877bf`상 Go 영역 차이는 go.mod/go.sum뿐이다.
- 그 스냅샷에서 같은 govulncheck 명령: exit 0, No vulnerabilities found, 호출 가능한 취약점 0건. `scout-vuln-fixed.log`. 비호출 패키지 1/모듈 4건 경고는 남으므로 '모든 의존성 취약점 0'이라 쓰지 않는다.
- `-modfile` 환경변수 방식은 govulncheck의 package loading에서 실패하여 증거로 쓰지 않았다. 스냅샷은 실제 모듈 루트에서 동일 명령을 사용한다. 전체 시험 첫 실행은 추출에서 docs/openapi.yaml을 빠뜨려 TestOpenAPIMatchesRoutes가 실패했고, 같은 커밋의 해당 파일을 추가한 뒤 재검증했다(제품 결함 아님).
- 스냅샷 `go test -p 1 ./... -count=1`: exit 0, 15개 패키지 통과 및 migrations 시험 없음. `scout-fixed-go-tests.log`. POSTGRES_DSN을 제공하지 않아 DB 통합은 미검증. `go vet ./...`, `go mod verify`, `go build ./...`도 exit 0. 정찰에서는 race/웹 설치 후 시험/이미지 빌드 미실행.
- 원본 `npm test --silent`: web/vitest 미설치 exit 1. `scout-web-uninstalled.log`. 정찰은 npm 설치/코드 변경/커밋을 하지 않았다.

대안 비교와 추정:
- 선택: 기존 보안 수리의 두 파일만 반영. 원인과 수정 버전, 과거 수리 diff, 독립 스캔 통과가 있어 가장 작은 변경이다.
- pgx 또는 전체 x/* 갱신은 의존성 범위를 불필요하게 넓혀 제외. 스캔 완화는 요구 위반. 기다리기는 현재 스캔 실패가 지속되어 제외. 설치 가드 추가는 후속 러너의 설치 생략을 해결하지 않아 제외.
- Bottom-up 추정: 시작점/기록 3–5분, 두 파일 수정 2–4분, Go 검증 5–8분, 설치·웹 검증 6–10분, 원장 2–3분 = 기본 18–30분. 알려진 환경/네트워크 변동 contingency 5–10분을 별도 배정해 총 23–40분; 45분 세션 안 범위다. DB 컨테이너/이미지 최초 다운로드가 필요하면 초과 가능하므로 즉시 재추정한다. Management reserve는 이 회차에 임의 배정하지 않는다. 신뢰는 중간이며 통계적 성공확률은 측정하지 않았다.
- 유사 사례 교차 점검: 직전 shepherd가 같은 두 파일 수리·Go 통과를 이미 수행했으므로 변경 규모는 타당하다. 실제 작업시간 표본이 없어 시간 보정률을 지어내지 않았다. 가장 큰 가정은 v0.41.0 이외에 새 권고가 생기지 않고 웹 설치가 정상 제공된다는 것이다.
- 적용 스킬: Skill 도구가 제공되지 않아 로컬 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`의 estimating-and-contingency, implementation-planning, solution-exploration/SKILL.md를 읽어 적용했다. 지정 단계·검증·체크포인트·대안·기본 추정/예비분을 위에 포함했다.
