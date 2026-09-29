- 과제: QR 스캔·로비 현황에서 기준 정보 로드 실패를 알리고 재시도 제공 (가치 3 / 위험 1 / 작업량 M)
- 왜: ScannerPage는 reference-data 실패를 catch하지 않고 LobbyPage는 실패를 숨겨, 일시 오류 뒤 처리 로비 목록·로비 필터를 화면 안에서 복구할 수 없다. 이미 VisitFormPage에 있는 별도 오류·로딩 상태와 재시도 방식을 적용하면 입력과 조회 상태를 유지하며 목록을 복구할 수 있다.
- 수용 기준:
  1) /lobby/scan과 /lobby의 GET /api/v1/reference-data가 실패하면 각 화면에 닫기 없는 한국어 오류 Alert와 `다시 불러오기` 버튼이 나타난다. QR/체크인 또는 현황 조회용 공용 error를 지워도 기준 정보 오류는 유지된다. ScannerPage에 미처리 promise rejection이 발생하지 않는다.
  2) 재시도 중에는 버튼 disabled 및 `다시 불러오는 중입니다` 안내를 유지한다. 한 번의 클릭에 요청 한 건, 진행 중 추가 클릭에 요청 증가 없음. 재실패하면 다시 시도할 수 있고, 성공하면 오류 안내가 사라진다.
  3) ScannerPage 성공 시 기존 scope 조건으로 첫 허용 로비를 고른다. lobby 계정의 siteScope가 전체 응답의 첫 사업장과 다른 경우에도 범위 밖 로비를 선택하지 않는다. 입력한 QR token·badgeNo와 이미 검증한 result는 재시도로 초기화하지 않는다. QR 검증·체크인 버튼의 기존 활성 조건과 빈 lobbyId 계약은 변경하지 않는다.
  4) LobbyPage는 성공 시 reference만 갱신한다. 검색 query·tab·lobbyFilter와 현황 데이터는 초기화하지 않는다. 허용 로비가 둘 이상일 때 기존 로비 필터가 다시 나타나고 기존 목록 조회와 SSE 연결이 유지된다. 로비가 0/1개인 정상 응답을 오류로 취급하지 않는다.
  5) 실제 서버·실제 빌드 UI를 쓰는 Playwright에서 두 화면 각각 최초 실패→재실패→지연된 재시도→실제 응답 성공을 검증한다. reference-data만 503으로 가로채고 성공 시 route.continue()로 실제 응답을 받는다. alert·버튼·요청 수·pageerror·로비 선택·입력 보존을 관찰해야 하며 소스 문자열 검사나 직접 주입한 React 객체로 대체하지 않는다. 기존 키보드 QR 체크인 E2E도 통과해야 한다.
- 건드릴 파일:
  - web/src/pages/ScannerPage.tsx: ScannerPage의 reference-data useEffect(현재 21행) — loadReference(), refError/refLoading, 별도 Alert. 기존 cleanup `return () => stopCamera()` 보존.
  - web/src/pages/LobbyPage.tsx: LobbyPage의 reference-data useEffect(현재 26행) — 동일한 실패/재시도 UI. load/useCallback 및 loadRef/EventSource effect는 변경하지 않는다.
  - web/e2e/visit-flow.spec.ts: 기존 login/createVisit 및 `checks a visitor in from a keyboard-wedge scan` 패턴을 이용해 위 두 화면의 회귀 시나리오 추가. 테스트 제목에 `reference-data recovery` 포함. 실제 범위 제한 계정 fixture 구성이 45분을 넘길 경우 별도 실제 브라우저 확인 결과를 남기고 가짜 reference 응답으로 대체하지 않는다.
  - docs/USER_GUIDE.md: 3.7/3.8에 실패 시 다시 불러오기 안내 한 문장씩. PDF 재생성은 제외.
  프로덕션 파일은 두 개. 공용 훅 추출·다른 페이지 이식·새 의존성·서버 및 인증/마이그레이션/CI/릴리즈 변경은 범위 밖이다.
- 검증 명령:
  - 저장소 루트: `go test ./... -count=1` — 정찰에서 PASS(app 0.138s), DSN 미지정으로 DB 통합은 SKIP. 이 결과는 화면/DB 증거가 아니다.
  - 프런트: `cd web && npm ci && npm run lint && npm test && npm run build` — package.json에서 확인한 명령. 현재 node_modules 없음; 정찰에서는 미실행.
  - 실제 UI 서버 준비: .github/workflows/ci.yml의 e2e 잡 `Build embedded UI`~`Start VisitFlow`와 같은 방식. PostgreSQL과 POSTGRES_DSN, BOOTSTRAP_ADMIN=admin, BOOTSTRAP_ADMIN_PASSWORD=e2e-bootstrap-password, 32바이트 ENCRYPTION_KEY 필요. 기존 cmd/visitflow/webdist를 먼저 임시 백업하고 `cp -r web/dist/. cmd/visitflow/webdist/`, `go build -o <임시경로>/visitflow ./cmd/visitflow` 후 해당 바이너리를 기동한다. /readyz 200 확인. 산출물은 끝에 임시 백업으로 복원한다.
  - 브라우저 설치가 필요하면 `cd web && npx playwright install --with-deps chromium`. 실제 서버가 8080에서 준비된 뒤 `cd web && VISITFLOW_BASE_URL=http://127.0.0.1:8080 npm run test:e2e -- --grep 'reference-data recovery'`, 이어 `cd web && VISITFLOW_BASE_URL=http://127.0.0.1:8080 npm run test:e2e`.
  - `git diff --check`; 테스트·UI 산출물이 변경 목록에 남지 않았는지 확인.
- 위험과 피할 것: 이전 기록의 “스캔 화면이 죽는다”는 표현을 사실로 전제하지 말 것. 현재 verify/checkin은 ref 로드를 기다리지 않으며 이 과제는 로비 목록 복구만 다룬다. refError를 공용 error에 섞지 말고 재시도 시작에 Alert를 제거하지 말 것(진행 중 버튼·문구가 사라짐). scope 필터·카메라 stopCamera 정리·SSE 재구독 구조를 보존한다. `.tsx` 테스트는 vitest include에서 제외되므로 UI 배선은 브라우저로 검증한다. 미머지 메일/MCP/import 브랜치 및 auth/migrations/workflows는 피한다. MUI select은 접근성 이름을 브라우저에서 확인하고 필요한 경우 aria-labelledby를 따라간다.
- 차선 후보: TestSelfRegistrationRecordsVisitorConsent의 동시각 동의 기록 선택 비결정성 제거 (가치 2 / 위험 1 / S). internal/app/integration_test.go:472의 ORDER BY consented_at DESC LIMIT 1은 동시각에서 결정적이지 않다. WHERE source='self'로 목적을 좁히되 실제 등록 경로를 거친 host/self 기록을 함께 검증한다. DB 없이 SKIP한 결과를 통과 증거로 삼지 않는다. 본 과제의 전제가 이미 해결된 경우만 선택한다.

진행 계획 (구현자 상태 갱신: 1 완료, 2 완료, 3 완료 — 집중 E2E 4개·전체 14개 통과, 원본 산출물 복원 및 기록 완료):
1. 재현/검증 준비: 기존 실제 서버에서 두 화면에 reference-data 503을 주고 현재 안내·재시도 부재를 관찰, 회귀 테스트를 추가한다. 증명: 위 focused E2E가 안내 부재로 실패. 점검: 자동 결과 확인 후 진행, 사람 승인 없음.
2. 두 페이지에 VisitFormPage.tsx:loadReference(48~57행)의 패턴 적용. 증명: npm run lint, npm test, npm run build 및 focused E2E 통과. 실제 동작이 전제와 다르면 과제서를 수정하고 범위를 늘리지 않는다. 점검: 수용 기준 1~4 재확인, 사람 승인 없음.
3. 가이드 두 문장과 검증 결과 정리. 증명: 전체 E2E 및 git diff --check, 산출물 복원 후 diff 확인. 점검: 수용 기준 5와 파일 수 확인, 사람 승인 없음.

대안 비교와 결정:
- 선택: 페이지별 기존 패턴 적용 — 두 파일로 끝나고 오류 상태를 각 작업 오류에서 분리할 수 있다.
- 오류 Alert + 전체 새로고침 안내 — 구현은 더 작지만 token/badge/검색 상태를 잃어 복구 목표를 충족하지 못한다.
- 공용 reference-data 훅/자동 재시도 — 향후 페이지 증가에는 유리하나 effect 수명·scope 의존성과 기존 페이지 이식이 늘어 이번 범위에 불리하다.
- 현상 유지 — 일시 장애마다 사용자가 재진입해야 하므로 운영 화면 복구 가치가 더 높다고 판단했다.

추정 근거와 예비:
- 상향식 판단: 재현/실제 서버 준비 8~12분 + 두 페이지 구현 8~10분 + E2E/가이드/검증 9~13분 = 기본 25~35분. 알려진 불확실성(브라우저 fixture·MUI 선택자)에 contingency 5분, 합계 30~40분. 관리 예비(management reserve)는 0분이며 새 범위를 추가하지 않는다.
- 유사 사례: 9/27 VisitFormPage 복구 과제가 채택·릴리즈되었고 동일 UI 패턴/브라우저 검증을 재사용한다. 당시 실제 소요시간은 미제공이므로 속도 보정 근거로 꾸미지 않는다. 30~40분은 환경 준비가 순조롭다는 조건의 중간 확신 판단 범위이지 통계적 신뢰구간이 아니다. 첫 E2E 준비 후 재추정한다.
- 가장 큰 가정: 구현 세션에서 실제 PostgreSQL·브라우저 환경을 곧바로 준비할 수 있음. 정찰에서는 DB/브라우저 재현 미확인. 서버 준비가 지연돼도 가짜 테스트로 수용 기준을 낮추지 않고 미검증을 명시한다.

정찰 범위: README, docs/ARCHITECTURE·USER_GUIDE 관련 구간, 최근 git log -30, CI/릴리즈·테스트 구성, 위 페이지/테스트/API/후보 관련 구간 확인. 저장소 내 CLAUDE.md·AGENTS.md·별도 로드맵/TODO 파일은 검색에서 발견하지 못했고 README/docs/internal/web/src TODO/FIXME 검색에도 결과 없음. 스킬 전용 도구는 노출되지 않아 로컬 headcount의 pmo:estimating-and-contingency(및 references/sources.md), technology:implementation-planning, technology:solution-exploration을 직접 읽고 적용했다. 외부 비용·생산성 수치는 사용하지 않았다.
