- 과제: 수정 과제 — PR #10의 반복 CI 실패를 고치는 미병합 의존성·APK 잠금 수정 통합 (가치 5 / 위험 2 / 작업량 M)
- 왜: PR #10(5fc31df)의 Build and test는 govulncheck의 도달 가능한 gRPC 취약점과 Docker runtime-base의 사라진 APK 고정 버전 때문에 실패하며, 자리표시자 수정만으로는 해결되지 않는다. 기존 성공 커밋 8451b5b의 좁은 수정을 현재 기준선에 통합하고 같은 검증을 다시 통과시키면 워크플로를 약화하지 않고 두 차단 원인을 제거한다.
- 수용 기준: 1) 현재 최종 산출물에서 `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`가 exit 0이며 GO-2026-6348의 reachable 경로가 사라진다. 2) 실제 `docker build -t madi:ci .` 및 `bash scripts/verify-image.sh madi:ci`가 exit 0이고 기존 69 APK / 52 source origins / OCR 2모델·오프라인 실행 검증을 그대로 통과한다. 3) 라이선스 고지의 의존성 해시가 일치하고 실제 OTLP HTTP/TLS 경로 회귀가 통과한다; DB가 필요한 검사는 PostgreSQL을 연결해 SKIP 없이 실행한다. 4) 이미 성공한 PR #10의 clean-checkout 빌드 계약을 최종 산출물에서도 보존하고, journal.md에 수정 과제·변경 파일·정확한 명령/종료 코드·미실행 항목을 기록한다. 로컬 통과를 원격 릴리즈 성공으로 표현하지 않는다.
- 건드릴 파일: `go.mod`/`go.sum`: google.golang.org/grpc v1.82.1 → v1.83.1 및 필요한 합계만; `deploy/runtime-apk.lock`: ca-certificates와 ca-certificates-bundle 모두 20260909-r0, tzdata 2026d-r0; `web/public/licenses-manifest.json`/`web/public/licenses.txt`: `scripts/licenses.mjs`로 생성. 현재 main에 PR #10이 없으므로 선행으로 그 네 파일(`.gitignore`, `web/dist/.gitkeep`, `web/vite.config.ts:keepDistPlaceholder`, README 로컬 개발 절)을 기존 5fc31df 패치와 동일하게 보존/통합한다. 이것을 새 개선으로 재설계하지 않는다.
- 검증 명령: 아래 단계별 명령을 저장소 루트에서 순서대로 실행한다. `.github/workflows/ci.yml` 및 release.yml의 동일 실패 명령을 최종적으로 모두 실행한다.
- 위험과 피할 것: `.github/workflows/*`, Dockerfile의 고정 digest·APK 설치 명령, 취약점 검사 버전/레벨, `tests/image-smoke/extraction.go:verifyRuntimeSources`의 개수·해시·출처 검증, auth·SQL·마이그레이션은 바꾸지 않는다. 인증서 bundle까지 같이 갱신하지 않으면 동일 recipe의 서로 다른 commit이 남아 53 origins로 늘어나는 기존 실패가 반복된다. 전체 의존성 최신화·메이저 업그레이드·APK 무버전 설치·취약점 무시·새 사전점검 스크립트만 추가하는 접근 금지. PR #8은 open/미병합이며 거절/롤백 근거는 발견하지 못했다; 이미 성공한 패치 재사용은 실패 접근 재시도가 아니다.
- 차선 후보: 기존 패치가 현재 저장소와 더 이상 맞지 않을 때 동일 두 실패의 최소 직접 수정 — 최신 로컬 govulncheck 출력의 최소 fixed 버전과 고정 Alpine 이미지의 실제 제공 버전을 확인해 위 다섯 파일만 재생성한다. 기능/DX 후보로 전환하지 않고 예상 외 원인이면 과제서와 추정 시간을 갱신한다.

## 확인한 근거와 불확실성
- 정찰 기준선 main@fd2c3b5, git log -30 실제 10건, 작업 트리 깨끗함. CLAUDE.md/저장소 AGENTS.md 없음. README, docs/roadmap.md, BUILD_PLAN.md, 배포 기록, 테스트·CI·릴리즈 스크립트를 읽었다. TODO/FIXME 검색에서 이번 차단과 관련한 항목 없음.
- https://github.com/hkjang/madi/actions/runs/35699732698 : PR #10의 test job 106654629610 step 9 `Reject reachable Go vulnerabilities`, offline-image job 106654629493 step 4 `Build service image` 실패.
- https://github.com/hkjang/madi/actions/runs/35488581361 : 직전 PR #9도 같은 두 단계 실패. gRPC StartJobs 호출 경로는 두 run의 check annotations에서 확인. 이미지 로그 본문 API는 403이므로 원격 APK 오류 문구 자체는 미확인; 현재 HEAD의 동일 Dockerfile/lock으로 로컬 재현했다.
- `release.yml` 최근 두 run 34369441095/34203932033은 success. 따라서 '릴리즈 워크플로 두 번 실패'를 사실로 옮기지 않는다. 확인된 것은 릴리즈와 공유하는 CI 단계의 반복 실패다.
- PR #10의 go.mod/go.sum/lock/workflows는 fd2c3b5와 diff 없음. 8451b5b는 HEAD와 PR #10 어느 쪽의 조상도 아니다. PR #8(8451b5b), #7(1525a82f)은 open/미병합이며 해당 수정의 CI run 35444328456/35454361881은 success. 기존 패치 diff를 직접 읽었다.
- 정찰 실제 실행: `docker build --target runtime-base .` 실패(exit 1, 내부 apk xargs 123): ca-certificates=20260611-r0, tzdata=2026c-r0 충돌. PR #10 원본 archive를 이 회차 assets/pr10-repro에 풀어 실행한 govulncheck는 GO-2026-6348, grpc v1.82.1 → v1.83.1 보고(go run exit 1, 도구 exit 3). 이 archive는 실제 웹 빌드 산출물 없이 PR #10의 추적 placeholder를 사용한 분석이므로 웹 실행 검증은 아니다.
- HEAD의 bare `go build ./...`는 web/embed.go:7 all:dist 실패. `go test -count=1 ./tests/deployment-contract`, Node natural-date/silent-sso 2건, `node scripts/licenses.mjs --check`(423 components), `git diff --check`는 통과했다. 수정 후 통과 검증은 정찰의 코드 변경 금지 때문에 구현자에게 남긴다. 최초 archive 추출 명령은 잘못된 cwd로 실패했고 cwd를 명시해 재실행했다; 그 실패는 제품 결함이 아니다.
- API 원본 증거: assets/pr10.json, pr10-jobs.json, previous-failed-jobs.json, release-runs.json.

## 실행 계획 (모든 단계 pending, 사람이 없는 자동 회차: 사람 승인 지점 없음)
1. 기준선/선행 확인 (pending): `git show 5fc31df -- .gitignore README.md web/vite.config.ts web/dist/.gitkeep`를 읽고 이미 있으면 재적용하지 않는다. 없으면 그 네 파일 패치만 통합한다. 증명: `go build ./...` 및 `go vet ./...`; placeholder는 최종 추적 파일로 남긴다. 체크포인트: 실패하면 의존성 수정으로 넘어가기 전에 원인/계획을 기록한다.
2. 두 차단 원인 수정 (pending): `git show 8451b5b -- go.mod go.sum deploy/runtime-apk.lock web/public/licenses-manifest.json web/public/licenses.txt`와 비교해 검증된 다섯 파일 패치를 통합한다. 수동 갱신이 필요하면 `go get google.golang.org/grpc@v1.83.1`, `go mod tidy`; x/sys·x/text의 직접 의존성 이동은 실제 import가 있어 기존 패치에도 포함되어 있다. `npm ci --prefix web`, `node scripts/licenses.mjs`, `node scripts/licenses.mjs --check`, `npm run build --prefix web` 순서로 고지를 포함한 웹 자산을 생성한다. 증명: 아래 govulncheck와 runtime-base 빌드 모두 exit 0. 체크포인트: 새 취약점/다른 APK 충돌이면 무작정 버전을 올리지 말고 계획 수정.
3. 실제 런타임 회귀 (pending): 아래 OTLP 시험과 전체 이미지/오프라인 검증을 실행한다. `scripts/bundle-alpine-sources.mjs`는 설치 APK의 origin/commit별로 소스를 묶고 `verifyRuntimeSources`는 이를 검사하므로 해당 생산 경로를 통과해야 한다. 체크포인트: 원격/네트워크 오류를 제품 성공으로 기록하지 않는다.
4. 최종 산출물 확인 (pending): `go build ./...`, `go vet ./...`, `go test -count=1 ./tests/deployment-contract`, `node scripts/licenses.mjs --check`, `git diff --check`. 최종 추적 파일을 새 archive 디렉터리에 추출하여 웹 빌드 전 Go build/vet도 재확인한다. 워크플로 변경 없음과 정확히 필요한 diff만 있는지 확인하고 journal에 '수정 과제'로 기록한다. 전체 CI/릴리즈 게시 상태는 별도로 남긴다.

```sh
# 2단계: 실패했던 실제 검사
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
docker build --target runtime-base .
# 3단계: DB 불필요, 실제 collector HTTP/TLS 프로토콜 회귀
go test -race -count=1 -run '^TestOperationsOTLPProtocolTLSRedirectAndPrivacy$' ./internal/server
# DB가 준비된 경우 MADI_TEST_POSTGRES_DSN을 격리 PG17 DB로 설정하고 실행(SKIP 금지)
go test -race -count=1 -run '^TestOperationsRealSDKRuntimeDisableAndErrors$' ./internal/server
# 실제 CI offline-image job과 동일
docker build -t madi:ci .
bash scripts/verify-image.sh madi:ci
```

전체 `go test -race -failfast -count=1 -timeout 45m ./...`는 PG17 및 ci.yml의 MADI_BROWSER_* 옵션·Chromium이 필요하다. 이번 45분 수정의 필수 원인 검증과 구분하며 옵션 없이 실행해 전체 CI 동등 검증이라고 주장하지 않는다. 태그 생성·push·릴리즈 게시·PR 병합은 이번 구현 과제 범위 밖이다.

## 대안 비교·추정 근거
- 선택: 기존 성공 패치 통합 — 새 움직이는 부품 없이 두 원인을 동시에 제거하고 기존 소스·라이선스 계약 보존. 가장 큰 가정은 현재 저장소에도 해당 APK 세 버전이 제공되고 grpc 1.83.1로 최신 reachable 검사가 통과한다는 것; 사전 검증으로 확인한다.
- 직접 최소 갱신: 기존 패치가 더는 유효하지 않을 때만 차선. 재생성 diff와 source inventory 재확인 비용이 늘어난다.
- 저장소 스냅샷 미러/의존성 자동갱신 구축: 장기 드리프트 예방에는 유효하나 45분·낮은 위험 범위를 넘으므로 보류. 기존 PR 병합을 기다리기만 하는 것은 이번 실행 가능한 수정 요구를 충족하지 못한다.
- bottom-up 작업 추정: 선행/패치 확인 4–6분, 다섯 파일 통합·고지 4–6분, 취약점/OTLP 검사 4–7분, 캐시 있는 이미지·오프라인 검사 12–18분, 기록 2–3분 = 기본 26–40분. 알려진 네트워크 재시도 대비 contingency 5분(중복 가산 없음), 합계 31–45분. 체감 신뢰 중간이며 통계적 80% 보장 아님; 캐시/연결을 전제로 한다. 완전 cold image·원격 전체 CI 대기는 45분 초과 가능하므로 시간을 축소 보고하지 말고 재추정한다. 미발견 범위의 management reserve는 별도 미배정, 임의 기능 추가 금지.
- analogous 비교: 9/19의 동일 다섯 파일·13행 성공 변경과 검증 조합을 근거로 M을 선택했으나 과거 실제 소요시간 자료가 없어 시간의 독립 보정은 미확인. 범위·가정·분해·불확실성 기록 원칙은 [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)를 참고했다.
- 적용 스킬 원문: `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`(뒤 두 경로도 같은 plugins 루트). 전용 Skill 도구가 없어 로컬 원문으로 읽고 절차를 적용했다.
