- 과제: 제품 /guide 가져오기 설명에 지원되는 TSV·일반 텍스트를 명시하기 (가치 2 / 위험 1 / 작업량 S)
- 왜: 제품의 GuidePage 가져오기 절은 엑셀·CSV·워드·PDF·마크다운만 안내하여, 이미 업로드 가능한 TSV와 일반 텍스트를 사용자가 발견하기 어렵다. 두 형식의 확장자와 읽는 방식을 기존 안내에 보완하면 사용자가 불필요한 형식 변환 없이 가진 자료를 가져올 수 있다.
- 수용 기준:
  1) 실제 /guide의 #import 절 첫 문단에 TSV(.tsv)와 일반 텍스트(.txt)가 보인다. 확장자는 기존 JSX 관례인 <code>로 표시한다.
  2) 같은 절의 읽는 방식 목록에서 TSV는 엑셀·CSV와 함께 표 데이터로, 일반 텍스트는 워드·마크다운과 함께 제목·요점으로 읽힌다는 안내를 받는다. 일반 텍스트가 임의의 자연어 첫 줄을 무조건 제목으로 인식한다고 약속하지 않는다. 필요하면 '.txt도 마크다운과 같은 규칙으로 읽습니다'라고 짧게 명시한다.
  3) /presentations 링크, #import 앵커, 목차 이동, 나머지 형식 설명을 보존한다. 기존 .pptx 목록 전체에 TSV도 그림·차트 가져오기를 지원하는 것처럼 새로운 주장을 붙이지 않는다.
  4) 웹 typecheck/build 및 기존 Vitest가 통과한다. 이 문구 수정만을 위한 소스 문자열 검사나 구현을 그대로 복제하는 새 테스트는 만들지 않는다. 실제 /guide에 접근하여 두 형식이 표시되고 목차·링크가 작동하는지 확인한다. 기존 docs 테스트 통과만으로 UI 표시를 검증했다고 쓰지 않는다.
- 건드릴 파일: web/src/pages/GuidePage.tsx:GuidePage — id="import" 절의 첫 문단(115~121행)과 형식별 읽기 방식 목록(131~132행)만 보완. 프로덕션 1파일, 새 테스트 파일 0개를 기본 범위로 한다.
- 검증 명령:
  - 저장소 루트에서 `cd web && npm ci && npm run typecheck && npm run build && npm test` (기존 package.json 스크립트 확인; npm ci는 구현 단계에서 실행).
  - 저장소 루트에서 `git diff --check` 및 `git diff --stat`.
  - 백엔드 계약을 다시 확인할 때 `cd server && go test -count=1 ./internal/docs`.
  - 화면 확인은 개발 웹 `cd web && npm run dev` 또는 기존 실행 환경의 로그인된 /guide#import에서 수행한다. API/로그인 환경이 없으면 화면 확인은 미확인으로 남기며 소스 검색으로 대체하지 않는다. App.tsx:App은 로그인 검사를 지난 뒤 /guide를 GuidePage로 연결한다.
- 위험과 피할 것: 서버 파서·upload accept·API·정본 가이드의 구조·README·CSS·auth·migrations·.github/workflows·package*.json·tsconfig·버전·배포 파일은 이번 범위 밖이다. 이전 USER_GUIDE.md 규칙 분리 작업은 verify-failed 기록이 있고 현재 main에도 없으므로 반복하지 않는다. 목록 안 인용 수정 dd7f719는 이전 회차에서 구현되어 review-pending이나 현재 main@eba092b에는 아직 없다. 다시 구현하거나 이 과제에 섞지 않는다. 의존성 설치/빌드가 기존 문제로 실패하면 결과와 실패 원인을 기록하고 임의의 업그레이드로 과제를 확장하지 않는다.
- 차선 후보: 로컬 `make test`에서 기존 Vitest도 실행하기 (가치 3 / 위험 2 / 작업량 S) — 1순위가 이미 해결된 경우만. Makefile:test의 기존 웹 typecheck/build에 npm test를 추가하고 의존성 설치 후 `make test`의 실제 실행에서 Vitest가 수행·통과하는지 확인한다. CI·의존성·테스트 구현은 변경하지 않는다. 웹 테스트 실패는 감추지 않으며 다른 수정으로 확장하지 않는다.

확인 근거와 검증의 한계
- 기준 HEAD: eba092b, VERSION 1.69.58. git log -30, README, docs/roadmap-v2.md, USER_GUIDE, Makefile, .github/workflows/ci.yml 확인. 로드맵은 v0.44 당시 계획이다. 저장소 안 CLAUDE.md/AGENTS.md 및 README·docs·server·web/src의 TODO/FIXME 검색은 결과 없음.
- server/internal/docs/docs.go:Read는 .tsv를 readSeparated(..., '\t')로, .txt/.md/.markdown을 readMarkdown으로 연결한다. web/src/pages/PresentationsPage.tsx:146의 실제 file input accept에도 .tsv/.txt가 있다. docs/USER_GUIDE.md:274~276은 이미 두 확장자를 설명한다.
- web/src/pages/GuidePage.tsx:GuidePage에는 두 형식이 빠져 있다. web/src/App.tsx:61은 /guide를 이 컴포넌트로 연결한다. 코드로만 추정한 별도 가이드 사본을 통합하지 않는다.
- 정찰 실측: `cd server && go test -count=1 ./internal/docs` → PASS, 3.358초.
- 실제 Read를 호출한 저장소 밖 Go overlay probe도 PASS: 탭 구분 매출.tsv가 ::columns와 두 숫자 행 및 !source를 생성했고, '# 분기 요약\n- 매출이 늘었습니다.\n'인 보고서.txt가 제목·요점·출처를 생성했다. 재실행: `cd server && go test -overlay=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-005834-ptium-improve/scout-overlay.json -count=1 ./internal/docs -run TestScoutSupportedFormats -v`. probe는 정찰 증거용이며 제품 테스트로 복사할 필요 없다.
- told_test.go:TestEveryFormatReadIsAFormatSaid는 README.md와 docs/USER_GUIDE.md만 검사한다. UI 문구 검증 근거로 쓰면 안 된다.
- Node/npm 실행 파일은 존재하나 web/node_modules는 없다. 정찰에서는 npm ci·typecheck·build·Vitest·audit·브라우저 실행을 하지 않았다. 웹 및 현재 취약점 상태는 미확인이다. 전체 Go race/vet도 이번 정찰에서는 실행하지 않았다.

구현 순서 및 체크포인트 (모두 구현자 자체 확인, 사람 승인 대기 없음)
1. [대기] 변경 전 GuidePage의 두 누락과 docs.go:Read 배선을 위 위치에서 다시 확인하고, 웹 의존성을 준비해 기존 검증을 실행한다. 증명: `cd web && npm ci && npm run typecheck && npm run build && npm test`. 기존 실패가 있다면 변경 탓과 구분하여 기록한다.
2. [대기] GuidePage.tsx의 위 두 자리만 편집한다. 권장 표현은 첫 문단의 'CSV·TSV(.tsv)'와 '마크다운·일반 텍스트(.txt)', 목록의 '엑셀·CSV·TSV'와 '워드·마크다운·일반 텍스트'이다. 증명: `git diff --check`, `cd web && npm run typecheck && npm run build && npm test`. 범위가 1파일을 벗어나면 먼저 계획을 고친다.
3. [대기] 실제 /guide#import에서 표시·목차·/presentations 이동을 확인하고 결과를 기록한다. 화면 확인이 불가능하면 그 조건을 명시한다. 증명: 실제 렌더 화면 관찰 및 `git diff --stat`; 완료/미확인 상태를 이 계획에 갱신한다.

대안 비교 및 선택 가정
- 채택: 기존 문단·목록 두 곳만 보완. 기능·배선은 이미 있으므로 사용자에게 발견 가능하게 만드는 최소 변경이며 런타임 회귀 면적이 작다.
- 별도 지원 형식 공통 데이터/컴포넌트: 형식이 크게 늘 때에는 타당하지만 현재 한 화면의 두 누락에 비해 배선·테스트 범위가 커진다. 이번에는 채택하지 않는다.
- 정본 가이드 링크만 추가: 유지 비용은 낮지만 파일을 올리기 전 보는 제품 안내에는 여전히 누락이 남는다. 이번 문제를 직접 해소하지 못한다.
- 현 상태 유지: 파서 기능에는 영향이 없으나 이미 지원하는 입력 형식을 발견할 수 없는 불일치가 지속된다.
- 핵심 가정: 이 체크아웃의 GuidePage가 현재 제품 안내이며 수정할 때도 누락이 남아 있다. 위치·본문이 바뀌었으면 재확인하며, 이미 해결됐으면 차선만 별도 선택한다.

작업량 근거와 예비 시간
- 방법: 파일·실행 단계에 근거한 bottom-up 추정. 과거 기록에는 신뢰할 벽시계 소요가 없어 유사 사례 소요를 수치로 꾸미거나 정량 신뢰도를 주장하지 않는다.
- 기본 작업: 사실/기존 웹 검증 8~12분 + 문구 편집 3~5분 + 변경 후 검증/화면 확인/기록 8~13분 = 19~30분. 로컬 설치 캐시와 실행 가능한 로그인 환경이 있다는 조건의 경험적 범위이며 통계적 보장 아님, 확신은 중간.
- contingency: npm 설치·검증 편차에 5~10분을 별도로 둔다(합계 24~40분). 기존 의존성 문제, DB 기동, 브라우저 설치가 필요하면 45분 내 완료 여부를 재평가하고 미확인을 남긴다. management reserve는 이 과제에 배정하지 않으며 임의의 추가 기능·복구 공수를 포함하지 않는다.
- 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md를 읽어 적용했다. 전용 Skill 도구는 이 세션에 없어 로컬 파일로 접근했다. 비용 추정 스킬 references/sources.md도 확인했으며 위 시간 범위는 외부 통계가 아닌 이 과제의 작업 분해에 따른 판단이다.
