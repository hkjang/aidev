# 회차 노트 2026-09-19-213406-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:34] base pinned — main@fd2c3b5
- [러너 21:34] autonomy release — 

## 정찰 노트
- 우선 과제(PR #7 CI 실패 수정)를 그대로 과제로 삼음. gh 가 이 환경에서 승인 없이 안 돌아 실패 단계·로그는 미확인 — 과제서는 "로그 확정 → 갈래 A/B/C/D 중 하나" 구조.
- 확인한 것: PR #7 diff 4파일, ci.yml 단계 순서, .dockerignore 가 web/dist 제외, 테스트·스크립트 어디도 .gitkeep/.gitignore 를 검사하지 않음(grep 0건), 로컬 verify.txt 는 전부 통과.
- 추측: vite emitFile(".gitkeep") 이 가장 의심되나 근거 약함. CI 실패가 PR 과 무관한 기존 플레이키 시험일 가능성도 큼.
- 구현자 주의: 워크플로 느슨화 금지, 전체 -race 스위트·브라우저 시험 전체 금지(-run 으로 좁힐 것), docs 스크린샷 diff 되돌리기.
- 프로필은 0일 전 것이 현재 코드와 일치해 갱신하지 않음.
- [러너 21:37] scout done — PR #7(`auto/2026-09-19-1923`, 커밋 e8e0960 "web/dist/.gitkeep 추적") 의 CI 실패 원인을 로그로 특정해 고치고, 같은 단�

## 구현 노트
- 원인 확정(공개 REST API, gh 미인증): run 35437829776 `test` job step 9 `Reject reachable Go vulnerabilities` 실패(이후 skipped) + `offline-image` job step 4 `Build service image` 실패. 같은 두 단계가 09-16(390b53e)·09-17(af8e021) PR 에서도 실패, main 09-14 push 는 성공 → PR #7 무관 외부 드리프트. 로그 본문은 admin 권한 필요라 못 받았고 로컬 재현 첫 줄로 갈음: govulncheck "GO-2026-6348 … grpc@v1.82.1 Fixed in v1.83.1"(exit 3), docker "ERROR: unable to select packages: ca-certificates-20260909-r0: breaks: world[ca-certificates=20260611-r0] / tzdata-2026d-r0 …"(exit 123).
- 바꾼 것(커밋 8451b5b, 앞에 e8e0960 cherry-pick=a0066bd): go.mod grpc v1.83.1 + tidy, `web/public/licenses*` 재생성(manifest 가 go.mod/go.sum 해시 — 안 하면 step 7 실패), `deploy/runtime-apk.lock` 3줄(ca-certificates·ca-certificates-bundle 20260909-r0, tzdata 2026d-r0). 워크플로·테스트 숫자 무변경.
- 확신 없는 곳: 없음에 가까움. 단 grpc v1.83.1 로 실제 gRPC 트래픽을 내는 경로는 madi 에 없고(HTTP OTLP 만 사용, grpc 는 proto 스텁 경유) `TestOperationsRealSDKRuntimeDisableAndErrors` 는 DB 없이 SKIP 이라 -race 로 돈 것은 `TestOperationsOTLPProtocolTLSRedirectAndPrivacy` 뿐. 전체 `go test -race ./...` 는 러너 예산 때문에 돌리지 않았다.
- 첫 시도(ca-certificates·tzdata 만 갱신)는 `verify-image.sh` 의 image-smoke 가 "unexpected runtime source inventory size"(origins 53) 로 실패 — bundle 도 같은 레시피 버전으로 맞춰 52 유지. 테스트 카운트를 고치는 쪽은 택하지 않음.
- 일부러 하지 않은 것: vite 플러그인 변경(갈래 A, CI 에서 npm run build 는 성공했음), 워크플로 수정, image-smoke 69/52 상수 변경, 전체 -race 스위트·브라우저 시험.
- 다음 역할 주의: `offline-image` 검증은 Docker+네트워크(aports/distfiles 다운로드, 첫 회 ~10분) 필요. go.mod 를 또 바꾸면 `node scripts/licenses.mjs` 재실행 필수. lock 을 바꾸면 image-smoke 의 origins 수(같은 aports commit 을 공유하는 패키지 묶음)가 달라질 수 있다. Playwright Chromium 을 이 임시 홈에 설치했음(tests/usability/recorder.test.mjs 통과용). PR #7 은 이 브랜치가 대체(e8e0960 포함).
- [러너 21:54] brief accepted — 채택 — 갈래 A~D 어느 것도 아니었지만(실제 원인은 govulncheck 단계와 offline-image 의 apk lock, 둘 다 PR 무관 외부 드리프트) "�
- [러너 21:54] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 직접 재실행해 확인: `go mod tidy -diff` 무변경(x/sys·x/text 의 direct 승격은 tidy 결과), `node scripts/licenses.mjs --check` ok(423), `govulncheck@v1.7.0 ./...` 0건(exit 0), `docker build --target runtime-base` 캐시 적중으로 lock 해석 확인, licenses.txt sha256 = manifest text_sha256, `.gitignore` 규칙(check-ignore) 의도대로 동작.
- 못 본 것: 전체 `offline-image` 파이프라인(runtime-sources·image-smoke 69/52)은 구현자 기록만 신뢰, 전체 -race 스위트 미실행.
- 남는 우려(승인): (1) Dockerfile 이 alpine 3.24 저장소를 스냅샷으로 고정하지 않아 lock 드리프트가 재발한다 — 다음 회차에 저장소 미러/스냅샷 고정 검토. (2) `all:dist` 임베드라 `/.gitkeep` 이 빈 200 으로 서빙됨(무해).
- 릴리즈 노트: grpc v1.82.1→v1.83.1(GO-2026-6348), ca-certificates 20260909-r0, tzdata 2026d — 사용자 기능 변화 없음. 이 브랜치가 PR #7 을 대체.
- [러너 21:56] review approved — 리뷰 승인 (risk=low)
- [러너 21:56] pr created — https://github.com/hkjang/madi/pull/8
- [러너 22:15] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
