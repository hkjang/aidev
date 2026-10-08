- 과제: web 패키지의 Node 요구 버전을 현재 개발·CI 기준인 26 이상으로 선언해 설치 시 툴체인 이탈을 알리기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/package.json`과 잠금 파일의 루트 패키지에 engines가 없어, README·AGENTS·CI·Docker가 모두 Node 26을 요구하는데도 루트 패키지가 다른 Node로 설치되는 상황을 알리지 않는다. 두 파일에 동일한 요구를 선언하면 설치 단계에서 지원 환경 이탈을 경고하면서 이미 고친 npm 테스트 실행 경로를 보존한다.
- 수용 기준:
  1) `web/package.json`의 `engines.node`와 `web/package-lock.json`의 `packages[""].engines.node`가 모두 `>=26`이다. 이는 프로젝트의 지원 기준이며 Node 22에서 모든 명령이 실패한다는 뜻이 아니다. 최소 타입 스트리핑 버전이라는 이유로 과거 후보의 `>=22.18`을 재사용하지 않는다.
  2) 버전 1.24.0, 모든 의존성 범위·resolved·integrity 및 의존 패키지 항목은 그대로다. `test`의 `${npm_node_execpath:-node} --test tests/*.test.mjs`도 바뀌지 않는다. engine-strict, preinstall 강제 종료, 새 도구·서비스는 추가하지 않는다.
  3) 실제 낮은 Node(정찰 환경 v22.23.1)로 npm install/ci를 실행하면 `hunter-web@1.24.0` 자체에 대한 EBADENGINE 경고가 관찰되고 engine-strict=false에서는 설치가 성공한다. 다른 의존성 경고를 근거로 삼지 않는다. 실제 Node 26으로 실행한 설치에서는 루트 패키지 경고가 없고 웹 테스트가 113개 이상 실제로 실행되어 실패·skip 없이 통과하며 typecheck/build가 성공한다.
  4) 잠금 파일 diff가 루트 engines 메타데이터 변경으로 제한됨을 확인한다. 선언 내용을 반복하는 소스 문자열 시험은 추가하지 않는다. npm 자체의 실제 설치 결과와 테스트 개수가 증거다.
- 건드릴 파일: `web/package.json`: 루트 engines.node 추가; `web/package-lock.json`: packages[""].engines.node 동기화. 프로덕션/빌드 설정 2파일, 테스트 추가 0파일. README의 개발 절은 이미 Node.js 26이므로 수정 불필요.
- 검증 명령:
  - 환경: `node --version`, `node -p 'process.execPath'`, `npm --version`, `npm config get engine-strict`.
  - 잠금 갱신(필요 시): `npm --prefix web install --package-lock-only --ignore-scripts --engine-strict=false`. 뒤의 diff가 의존 패키지를 건드리면 그대로 채택하지 말고 루트 메타데이터만 맞춘다.
  - 실제 각 런타임에서: `npm --prefix web ci --engine-strict=false` (낮은 버전은 경고·성공 확인, Node 26은 경고 없음 확인). 프로세스별 옵션만 사용하고 전역 npm 설정을 수정하지 않는다.
  - Node 26에서 `npm --prefix web test`, `npm --prefix web run typecheck`, `npm --prefix web run build`.
  - 빌드 연결: `mkdir -p internal/webassets/dist` 다음 `cp -a web/dist/. internal/webassets/dist/`, `node scripts/verify-pentagi.mjs`, `go vet ./...`, `go build -o <이번_회차_산출물_디렉터리>/hunter-check ./cmd/hunter`.
  - `git diff --check`, `git diff -- web/package.json web/package-lock.json`, `git status --short`.
- 위험과 피할 것: 10-02의 잘못된 전제(현재 Node가 strip-types 플래그를 제거했다)를 반복하지 않는다. 현재 v22.23.1에서도 113개가 통과한다. 경고는 설치 시점에만 발생하며 `npm test` 단독 실행을 차단하거나 지원 여부를 전부 보증하지 않는다. CI·Dockerfile·release.sh·VERSION·auth·migrations·서버·원본 코어·docs 패키지는 이번 과제 밖이다. 다른 패키지의 engines나 dependency 버전도 바꾸지 않는다. 낮은 Node 경고만 확인하고 Node 26 성공까지 확인했다고 쓰지 말 것.
- 차선 후보: 목록 CSV의 범위 밖 숫자 타임스탬프 RangeError 방어 (가치 2 / 위험 1 / 작업량 S) — engines가 이미 반영되었거나 실제 설치 경고 검증이 성립하지 않을 때만 선택. `web/src/list-export.ts:listCSV`의 날짜 변환에 `date.getTime()` 유한성 검사를 추가하고 유효 범위 밖이면 기존 숫자를 csvCell로 전달해 다른 행의 내보내기를 살린다. 재현은 `created_at:8640000000000001`로 실제 listCSV 호출 시 `RangeError: Invalid time value`(정찰 실측). `web/tests/convenience.test.mjs`의 기존 CSV 시험 위치는 구현 시 확인(파일 본문 미열람). 정상 epoch 0·날짜 범위 양끝 ±8640000000000000·범위 밖 양끝·exportValue 우선·수식 보호를 실제 listCSV 호출로 검증하고 `npm --prefix web test`·typecheck·build 실행. 서버 생성 epoch라 실제 데이터 도달 가능성은 낮다.

범위 및 실행 계획 (구현 전부 미착수, 사람 승인 체크포인트 없음)
1. 지원 기준과 두 Node 실행 경로 확인: 위 환경 명령을 실제 각 런타임에서 실행하고 경로·버전을 기록한다. Node 26 확보 가능 여부는 정찰에서 미확인이다. 10분 안에 확보가 어렵다면 환경 구축을 별도 과제로 늘리지 않고 차선 선택 이유를 기록한다.
2. 두 파일만 변경하고 lock diff 검토: package.json과 잠금 파일의 루트 요구를 맞춘다. checkpoint: JSON 파싱 및 diff가 일치할 때 다음 단계로 진행. 의존성 재해석이 발생하면 범위를 수정한 뒤 진행한다.
3. 두 런타임 설치 결과와 지원 런타임 테스트·빌드 확인: 성공/경고가 기대와 다르면 원인과 계획을 고친다. 다른 Node 실행 파일이 npm 스크립트 PATH에서 선택되는 문제를 재도입하지 않는다.
4. 자산 포함 Go 빌드·원본 확인·diff 점검 후 실제 결과를 기록한다. UI 변경이 없어 새 화면 캡처·가이드 재생성은 없다. 테스트/CI/릴리즈의 미실행은 명시한다.

선택 근거와 대안
- 후보 전체 점수·상태는 ideas.json에 보존했다. 경고만 추가하는 현 접근은 기능·보안 경로를 건드리지 않고 과거 툴체인 사고 이후 남은 진입 안내 공백을 메운다.
- 더 큰 접근(툴체인 래퍼/버전 관리자/CI 행렬)은 검증 범위가 커서 제외한다. 현상 유지는 지원 환경 안내를 놓친다. engine-strict 강제는 경고 과제보다 강한 정책이라 제외한다.
- 초안의 SLA/risk 정수 잠금은 코드로 확인된 유효 후보이나, 현재 기존 테스트는 .tsx를 직접 실행하지 않으며 실제 입력·저장 배선을 검증할 브라우저/DB 환경이 없다. 소스 문자열 검사만 늘리는 반복을 피하려고 이번 회차 선택에서 내렸다.
- 가장 큰 전제: Node 26이 프로젝트가 의도한 지원 하한이다. 근거는 AGENTS 7장, README 개발 절 167행, `.github/workflows/ci.yml`의 node-version 26, Dockerfile 첫 단계 node:26이다. Node 22 실행 성공과 이 지원 기준은 별개다.

추정 근거와 여유
- bottom-up: 환경·기준 확인 3~5분, 두 파일 변경/lock 확인 2~4분, 실제 설치·웹 검증 6~10분, Go 빌드 연결·기록 5~8분 = 기본 16~27분. 알려진 불확실성(캐시·설치 출력 확인) 여유 3~8분을 별도로 두어 총 19~35분 예상. 통계적 신뢰구간이 아닌 중간 확신의 정찰 판단이며 45분 보장이 아니다.
- 유사 사례: 10-02는 같은 package.json 인근과 설치/테스트 경로를 검증했지만 런타임 가로채기 진단까지 필요했다. 이번은 그 진단을 반복하지 않는 더 작은 범위다. 과거 소요 시간이 없어 수치 환산은 하지 않는다.
- Node 26/의존성 다운로드 환경 새 구축, Docker·원격 CI·릴리즈는 위 추정 밖이다. 관리 예비(새 범위)는 배정하지 않는다. 초기 환경 확인에서 가정이 깨지면 일정 숫자를 줄이지 말고 차선으로 좁힌다.

정찰 확인 결과와 한계 (2026-10-08)
- main@9069bcf, VERSION/web/docs 1.24.0, 작업 트리 시작·중간 깨끗. Go 1.26.7, Node v22.23.1, npm 10.9.8. HUNTER_TEST_DSN 미설정, web/node_modules 없음.
- `npm --prefix web test`: 113 PASS / 0 FAIL / 0 SKIP. `go test -run '^TestFindingOpsParsingAndValidation$' -count=1 -v ./internal/app`: 1 PASS / 0 SKIP. `git diff --check`: 통과.
- Node 26 실행, npm ci, typecheck/build, 브라우저, DB HTTP, 원격 CI·배포는 정찰에서 실행하지 않았다. 위 명령 중 이 목록은 구현자가 수행할 검증이지 정찰 통과 보고가 아니다.
- CLAUDE.md·별도 ROADMAP/TODO 파일은 검색에서 발견하지 못했다. Hunter 코드·문서의 TODO/FIXME 검색도 해당 결과가 없었다. README·docs/validation.md·최근 git log 30개·CI 3개를 확인했다.
- 지정 Skill 도구는 이 세션에 노출되지 않았다. 로컬 headcount의 `pmo/skills/estimating-and-contingency/SKILL.md`(+references/sources.md), `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`를 직접 읽어 범위·대안·단계별 증거·추정과 여유를 적용했다. 위치: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/`. 외부 원가 기준을 적용한 추정은 아니다.
