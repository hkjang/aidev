- 과제: 수정 과제 — 미병합된 깨끗한 체크아웃 빌드 수정을 현재 기준선에 통합하고 최종 산출물에서 재검증 (가치 5 / 위험 1 / 작업량 S)
- 왜: 현재 main@fd2c3b5에서 `go build ./...`가 `web/embed.go:7:12: pattern all:dist: no matching files found`로 exit 1 하며, 성공한 수정 e8e0960은 현재 HEAD의 조상이 아니다. 이미 CI를 통과한 기존 수정의 필요한 부분을 이번 구현 산출물에 포함하면 매 회차 웹 산출물 없는 기준선으로 돌아가 발생하는 동일 실패를 막을 수 있다.
- 수용 기준: 1) 웹 빌드 전, 실제 추적 파일만 있는 최종 커밋의 깨끗한 체크아웃에서 `go build ./...`와 `go vet ./...`가 exit 0이다. 2) `npm run build` 이후에도 추적한 빈 `web/dist/.gitkeep`이 남고 git diff가 없으며 실제 index.html 및 assets가 생성된다; 서비스 워커 캐시 목록에 .gitkeep이 들어가지 않는다. 3) 테스트는 실제 Go 컴파일러와 Vite를 실행하여 양쪽 순서에서 통과를 증명하고, 임시 untracked 파일·Fake FS·소스 문자열 검사만으로 성공 처리하지 않는다. 4) 워크플로·Dockerfile·인증·SQL은 변경하지 않으며, 수정 과제의 검증 전/후 출력과 최종 커밋을 journal에 기록한다. 이번 수정의 완료를 원격 릴리즈 게시 성공으로 표현하지 않는다.
- 건드릴 파일: `.gitignore` — generic `dist/` 뒤에 `!web/dist/`, `web/dist/*`, `!web/dist/.gitkeep` 순서로 예외 적용; `web/dist/.gitkeep` — 빈 파일을 실제 추적 산출물에 포함; `web/vite.config.ts:defineConfig` 및 기존 패치의 `keepDistPlaceholder()` — Vite가 비운 뒤 generateBundle에서 빈 .gitkeep을 재생성, `offlineAssets()`의 허용 캐시 필터 유지; `README.md:로컬 개발` — 컴파일용 자리표시자와 실행 가능한 웹 자산을 구분하는 기존 수정 반영.
- 검증 명령: 아래 실행 순서 참조. 실제 정찰에서 실패한 것은 `go build ./...`; 통과한 것은 `go test -count=1 ./tests/deployment-contract`, `node --test tests/natural-date.mjs tests/silent-sso.mjs`, `git diff --check`이다. 수정 후 성공은 구현자가 확인할 항목이며 정찰에서 실행했다고 간주하지 않는다.
- 위험과 피할 것: 임시 .gitkeep만 만들어 빌드한 뒤 지우는 과거 방식 금지. generic dist/ 재포함 누락 금지. `emptyOutDir:false`, embed 범위 확장, 테스트 skip/continue-on-error, 워크플로 검증 축소 금지. 컴파일 성공만으로 웹 UI 실행 가능하다고 말하지 않는다. 이번에는 grpc·APK·라이선스·인증·마이그레이션·보호 경로 .github/workflows를 함께 고치지 않는다. PR #7/#8은 이미 존재하므로 중복 해결책이나 새 중복 PR을 만들지 말고 기존 수정 중 필요한 변경을 통합한다. 러너 기준선 main 갱신/원격 merge/push는 이 정찰의 권한 밖이며 자동 실행하지 않는다.
- 차선 후보: 같은 수정 과제의 인계 경로 변경 — 구현 시작 시 이미 .gitkeep 수정이 들어와 있다면 재구현하지 말고 실제 `go build ./...` 첫 오류와 최종 추적 파일을 확인하여 패치 누락/산출물 누락을 바로잡는다. 빌드가 이미 통과하면 기존 수정으로 해결됨을 기록하고 별도 기능으로 범위를 바꾸지 않는다.

범위와 근거
- 사용자가 정찰의 코드 변경을 금지하므로 수정 및 수정 후 검증은 구현자에게 인계한다. 원격 실패 두 건의 동일 원인 주장은 미확인이다.
- `.github/workflows/release.yml`, `ci.yml`, `compatibility.yml`은 모두 npm 웹 빌드가 Go 검증보다 앞선다. release.yml에는 독립 `go build ./...` 단계가 없다. Dockerfile도 web 스테이지 산출물을 복사한 후 Go 바이너리를 빌드한다.
- `scripts/release-image.sh`는 Docker 빌드/저장/압축 검증, `scripts/verify-image.sh`는 실제 폐쇄망 컨테이너 및 `tests/image-smoke`를 실행한다. `scripts/verify-browser.sh`는 실제 서버·격리 PostgreSQL 및 Playwright 경로를 사용한다. `tests/deployment-contract/deployment_test.go:TestDeploymentContracts`, `TestMainCIRequiresParallelOfflineImageVerification`은 필수 검증 계약을 검사한다. 이를 느슨하게 할 이유가 없다.
- `web/embed.go:Assets`는 `all:dist`, `cmd/madi/main.go:main`은 `fs.Sub(web.Assets, "dist")`를 사용한다. 현재 `git ls-files web/dist`는 빈 출력이다.
- `git show e8e0960`에서 위 네 파일의 원본 패치를 직접 읽었다. 재설계 대상이 아니라 아직 합쳐지지 않은 기존 성공 수정이다. 실패한 접근/반려·롤백된 PR을 되살리는 것이 아니다: 공개 API에서 PR #7/#8 모두 open, merged=false 및 최신 Build and test/compatibility 성공을 확인했다.
- PR #7은 e8e0960 + 1525a82f, PR #8은 a0066bd + 8451b5b이며 후속 커밋은 grpc/APK 외부 드리프트 수정이다. 필요한 네 파일의 e8e0960 패치를 검토하여 반영하고 전체 후속 커밋을 무조건 적용하지 않는다. 과거 문제와 현재 main 누락을 구분한다.
- 원격 증거: https://github.com/hkjang/madi/pull/7 , https://github.com/hkjang/madi/actions/runs/35454361881 (성공), https://github.com/hkjang/madi/pull/8 , https://github.com/hkjang/madi/actions/runs/35444328456 (성공). release.yml API가 반환한 최근 두 run 34369441095/34203932033도 success이다. 따라서 사용자 오류 문구는 현재 확인 가능한 원격 태그 릴리즈 이력과 일치하지 않으며 자동 러너 verify 실패는 별도로 확정한다.

실행 계획 (각 체크포인트는 자동 검증; 사람 확인 대기 없음)
1. [pending] 현재 HEAD 및 패치 존재 확인: `git status --short`, `git log -1 --oneline`, `git show e8e0960 -- .gitignore web/dist/.gitkeep web/vite.config.ts README.md`, `go build ./...`. 에러가 정찰과 다르면 과제서의 진단을 갱신한 뒤 진행한다.
2. [pending] 기존 패치의 네 파일만 현재 구현 산출물로 반영한다. 증명: `git check-ignore web/dist/.gitkeep`는 exit 1, `git check-ignore web/dist/index.html`는 exit 0; `go build ./...`, `go vet ./...`, `go test -count=1 ./web ./cmd/... ./tests/deployment-contract`. .gitkeep이 최종 추적 파일에 포함됐는지 `git ls-files --error-unmatch web/dist/.gitkeep`로 확인한다. 체크포인트: 웹 빌드 없이 컴파일 통과.
3. [pending] 실제 프런트 빌드: `(cd web && npm ci && npm run build)`, `test -f web/dist/index.html`, `test -f web/dist/.gitkeep && test ! -s web/dist/.gitkeep`, `git diff --exit-code -- web/dist`, `! rg -q '\.gitkeep' web/dist/sw.js`, `go build ./...`, `go vet ./...`. 체크포인트: 실제 자산과 자리표시자가 공존하며 기존 offlineAssets 캐시 범위 불변.
4. [pending] 구현 커밋 완성 뒤 최종 추적 파일만 별도 디렉터리에 풀어 검증한다: `MADI_VERIFY_DIR=$(mktemp -d)` 다음 `git archive HEAD | tar -x -C "$MADI_VERIFY_DIR"` 다음 `(cd "$MADI_VERIFY_DIR" && go build ./... && go vet ./...)`. 이는 npm 빌드 잔여물/로컬 캐시 파일로 통과하는 착시를 막는 수용 검증이다. `git diff --check` 후 journal에 수정 과제/커밋/각 명령 exit를 기록한다. 이 과제의 전체 Go race·브라우저·Docker 검증 중복 수행은 불필요하며 기존 CI 필수 검증은 그대로 둔다.

대안 비교 및 추정 근거
- 선택: 기존 성공 패치의 통합 + 최종 clean-checkout 증명. 네 파일만 다루고 프로덕션 임베드/배포 순서를 유지한다.
- 미선택: 러너에 npm 선행 빌드를 넣기 — 문서상 올바른 순서지만 러너는 이 저장소 범위 밖이며 지정된 bare Go 검증 실패를 이 저장소에서 해결하지 못한다.
- 미선택: fallback FS/build tag 도입 — 배포·로컬 빌드 경로를 나누고 진짜 프런트 누락을 숨길 위험; 이번 소규모 수리에 불필요하다.
- 미선택: 아무 변경 없이 PR 병합 대기 — 운영자 개입 없는 이번 러너는 main에 같은 실패를 재현한다. 대신 이미 있는 패치의 통합 과제로 처리한다.
- 가장 큰 가정: 러너가 이번 최종 구현 커밋을 검증한다. 다음 회차까지 main이 갱신되지 않으면 근본적으로 같은 기준선이 다시 선택될 수 있으며 코드 패치만으로 원격 병합을 보장할 수 없다.
- 작업 분해/바텀업 추정: 기준 확인 3~5분, 기존 패치 통합 4~7분, 웹/Go 검증 8~15분, 최종 체크아웃/기록 3~5분 = 18~32분. npm 네트워크/캐시 변동 대응 3~8분은 별도 contingency, 합계 21~40분. 관리 예비비/별도 릴리즈 수리는 포함하지 않는다. 통계적 P80 수치가 아닌 캐시/네트워크 정상 전제의 중간 확신 범위이며, 과거 동일 네 파일 성공 사례는 가능성 근거일 뿐 실행시간 측정이 없어 시간 보정 계수로 쓰지 않았다.
- 적용 스킬: 로컬 `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md` 원문을 읽었다. 전용 Skill 도구는 제공되지 않음. 추정 스킬 references/sources.md도 확인했으며 외부 비용·효익 수치를 차용하지 않았다.
