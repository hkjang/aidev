- 과제: 유실된 CI e2e 실패 요약 커밋 `ab36254`를 현재 main에 복원 (가치 4 / 위험 2 / 작업량 M)
- 왜: main@1d13889의 image 잡은 실패한 화면·route를 job summary에 쓰지 않고 artifact만 올려 구현·수리 담당자의 원인 파악이 늦어진다. 이전 회차에서 실제 결과로 검증했으나 main에 들어오지 못한 진단 코드 두 파일을 복원하면 기존 테스트 판정을 유지하면서 실패 단서를 바로 볼 수 있다.
- 수용 기준: 1) `ab3625475346be37da7e1bdc63bacc79e2b93e3f`의 두 파일 변경을 반영하고 `Summarize e2e failures`가 `if: failure()`로 기존 artifact 업로드 전에 실행되어 `$GITHUB_STEP_SUMMARY`에 append한다. 2) 실제 시각 회귀 실패 결과에서 화면 id·차이 비율·diff 경로를, 실제 browser smoke 실패 결과에서 locator/오류와 script frame을 출력하고 성공한 결과는 실패로 나열하지 않는다. 3) 파일이 없거나 JSON 구문이 깨져도 진단 스크립트는 exit 0이고 그 상태를 설명한다(임의의 JSON schema 전부에 대한 무오류 보장은 범위 밖). 4) 정상 e2e와 이미지 패키지 검증은 통과하며 테스트 exit code·기본 임계값·artifact·cleanup은 보존한다.
- 건드릴 파일: `scripts/e2e-failure-summary.mjs`(신규 복원): `readResult`, `addSection`, `addError`, `addItems` 및 최상위 세 결과 reader — 검증된 원본 복원; `.github/workflows/ci.yml`: `jobs.image.steps` — 진단 단계 7줄 추가. 서비스 프로덕션 파일 0개, 실행 스크립트 1개, workflow 1개. 다른 파일 변경은 이번 과제에 넣지 않는다.
- 검증 명령: 아래 단계별 명령. 정찰에서 실제 `make check` exit 0(OpenAPI 120개), `git show ab36254 --format= --binary | git apply --check` exit 0, 상태 경로에 추출한 원본의 `node --check`와 없는 결과 디렉터리 실행 exit 0을 확인했다. 실제 앱/e2e·package·원격 CI/릴리즈는 이번 정찰에서는 미실행이다.
- 위험과 피할 것: 보호 경로 중 ci.yml만 명시된 7줄로 제한하고 release.yml·auth·migrations는 건드리지 않는다. UI·베이스라인·임계값·재시도·continue-on-error·테스트 제외 변경 금지. 잘못된 version/임계값은 실패 증거를 만드는 해당 명령에만 지정하고 CI에 저장하지 않는다. 로컬 renderer 차이로 baseline을 재승인하지 않는다. artifact나 Actions summary의 공개 접근 가능성은 미확인이므로 '비로그인 누구나 볼 수 있음'을 약속하지 않는다. 현재 스크립트가 임의 schema에도 항상 안전하다고 확대하지 않는다. 대역 JSON만으로 실제 배선을 검증했다고 주장하지 않는다.
- 차선 후보: `followTopic` 저장 오류를 500으로 분리하는 `972113f` 복원 — e2e 요약이 이미 반영됐거나 기능상 무효일 때만. social.go의 현재 `decodeOptionalJSON` 호출을 보존하고 기존 실제 PostgreSQL 트리거 테스트를 사용한다. 이번 두 과제를 함께 하지 않는다.

범위와 근거:
- `git diff ab36254^ HEAD -- .github/workflows/ci.yml` 출력이 비었고, 세 producer(`e2e/visual-regression.mjs`, `accessibility-regression.mjs`, `browser-smoke.mjs`)도 해당 기준 이후 diff가 없다. 패치 적용 사전검사가 통과했다. 실제 cherry-pick은 정찰에서 하지 않았다.
- 원본은 로컬/원격 추적 `auto/2026-09-25-2149`에 남아 있고 현재 main에는 없다. 지난 기록상 CI 성공·실제 브라우저 결과 검증이 있었으나 원격 성공 상태를 이번에 재조회하지 않았다. 사람에게 반려된 접근이라는 근거는 없다.
- 현재 e2e/package.json의 실행 순서는 visual → accessibility → smoke다. CI source PostgreSQL 16, image PostgreSQL 17이며 Node 24다. 정찰 호스트 Node는 22.23.1이므로 호스트 make check 통과를 CI 동등 검증으로 부르지 않는다.

구현 순서와 체크포인트(모두 구현자 자체 검토, 사람 승인 대기 없음; 상태는 미착수):
1. [ ] 복원: `git cherry-pick ab3625475346be37da7e1bdc63bacc79e2b93e3f`. 증명: `git diff HEAD^ HEAD --stat`가 위 두 파일뿐이고 `git diff ab36254 HEAD -- .github/workflows/ci.yml scripts/e2e-failure-summary.mjs`가 비어야 한다. 체크포인트: 예상 밖 충돌/파일이 있으면 먼저 과제서·journal에 원인을 기록하고 범위를 다시 판단한다.
2. [ ] 빠른 검사: `node --check scripts/e2e-failure-summary.mjs`, `make check`, `node scripts/e2e-failure-summary.mjs "$RUN_DIR/missing-results"`. RUN_DIR은 이번 회차 상태 디렉터리 절대 경로로 지정. 구문이 깨진 JSON은 상태 디렉터리 아래 별도 디렉터리에만 만들고 해당 경로 인자로 검사한다. 이는 입력 견고성 검사이며 실제 producer 검증을 대신하지 않는다.
3. [ ] 실제 정상 배선: `make image`; ci.yml의 Start ephemeral PostgreSQL and moina 블록을 동일 조건으로 실행(전용 이름/네트워크, 새 PostgreSQL 17 DB, 앱 127.0.0.1:18080, VERSION 기반 이미지). `npm ci --prefix e2e` 후 e2e/VISUAL_REGRESSION.md의 공식 `mcr.microsoft.com/playwright:v1.62.1-noble` 실행법을 사용하여 `npm test`를 실행한다. 컨테이너에 `MOINA_E2E_BASE_URL=http://127.0.0.1:18080`, 테스트 계정/비밀번호, `MOINA_E2E_VERSION=$(cat VERSION)`를 전달한다. 정상 52장 비교·접근성·smoke의 결과 JSON을 별도 보존한 뒤 `node scripts/e2e-failure-summary.mjs`가 모두 통과로 기록하는지 본다. 체크포인트: e2e 선행 실패를 요약 스크립트 결함으로 단정하지 않는다.
4. [ ] 실제 실패 배선: 새 빈 DB로 리셋한 앱에서 같은 공식 renderer에 명령 단위로 `MOINA_VISUAL_MAX_DIFF_RATIO=0`을 전달하고 `npm run test:visual`을 실행한다. 실패가 관측된 실제 JSON을 보존하고, `MOINA_E2E_VERSION=v0.0.0-invalid`를 전달한 `npm run test:smoke`의 비정상 종료와 JSON을 보존한다. 요약 실행 stdout을 상태 디렉터리 Markdown에 append해 실제 화면 id·비율, smoke locator/stack이 담기는지 직접 확인한다. 픽셀 비교가 완전히 같아 실패하지 않으면 그것을 실패 증거라고 부르지 말고 smoke의 실제 실패로 핵심 배선을 확인하며 시각 실패 미검증을 남긴다. 같은 IP 로그인은 5분/5회 제한이므로 전체 suite 반복 금지.
5. [ ] 패키지·인계: `make package && make verify-package`를 한 번 통과시킨다. 체크포인트: PR의 source/image CI 성공을 확인하고 릴리즈 담당자는 release.yml의 exact commit CI gate와 실제 릴리즈 성공을 확인해야 한다. 정찰·구현만으로 릴리즈 완료라 쓰지 않는다. 변경 범위·검증 결과·미확인을 journal에 남긴다.

대안 비교와 선택:
- 원본 두 파일 복원: 기존 실제 producer와 동일하며 검증 이력을 재사용한다. 최소 변경으로 선택.
- Go 실패 요약부터 추가: 유용하지만 현재 알려진 Go flaky는 a2a0f65로 반영됐고 원본 e2e 복원보다 새 설계·검증이 필요하여 후순위.
- artifact만 유지: 코드 비용 0이지만 지난 회차의 진단 지연이 남는다.
- JSON 공통 reporter/producer 전면 정비: 출력 경로 및 접근성 필드도 정리할 수 있으나 별도 과제로 분리한다. 이번 복원의 선행조건이 아니다.

작업량 추정 및 예비 시간:
- Bottom-up: 복원·diff 3~5분, 입력/계약 검사 2~3분, 이미지·정상 배선 12~17분, 실패 증거 5~8분, 패키지·인계 3~5분 = 기본 25~38분. 로컬 캐시와 Docker 사용 가능을 전제한 공학적 예상이며 측정한 소요 시간이 아니다.
- 알려진 변동(이미지 레이어·브라우저 시작)에 contingency 3~7분을 별도로 두어 총 28~45분(M), 신뢰 중간. 통계적 확률을 계산한 추정은 아니다. 미지 범위용 management reserve는 이번 회차 0분; 새 결함 수정은 후속 아이디어로 넘긴다. 다운로드/빌드가 상한을 넘기면 통과로 꾸미거나 검증을 축소하지 않고 미완료 사유를 인계한다.
- Analogous 교차검토: 09-25 동일 두 파일과 실제 결과 검증 성공 기록이 있어 신규 reporter보다 구현 불확실성은 낮다. 당시 시간 실측이 없어 독립적인 수치 교차검증은 불가능하다. 현재 moina:v0.1.39 이미지는 없고 Playwright noble 이미지는 있으므로 앱 재빌드가 최대 가정이다. 최초 build 종료 시 추정을 갱신한다. 수십 회 반복 검증으로 TIMEOUT 난 교훈을 적용해 정상 1회·의도 실패 각 1회로 한정한다.

스킬 적용: 전용 Skill 도구는 제공되지 않아 다음 로컬 SKILL.md를 직접 읽고 적용했다. [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md). PMO references/sources.md도 확인했다. 추정 수치는 외부 비용모형이 아니라 현재 파일 범위·제공된 회차 기록을 근거로 하며 외부 통계나 정량 신뢰수준을 주장하지 않는다.
