- 과제: QR 스캐너의 기준정보 조회 실패를 명시하고 QR 검증 뒤에도 안내 유지하기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `ScannerPage.tsx:21`의 reference-data Promise에 catch가 없어 조회가 실패하면 처리 로비가 빈 선택지로 남고 오류는 unhandled rejection이 된다. 기준정보 전용 실패 안내를 유지하면 담당자가 통신 실패와 정상 빈 목록을 구별하고 새로고침으로 복구할 수 있다.
- 수용 기준:
  1) `/lobby/scan` 진입 중 `/api/v1/reference-data` 조회 실패 시 QR 입력 전부터 한국어 안내(예: 「로비 목록을 불러오지 못했습니다. 페이지를 새로고침해 주세요.」)가 보이고 pageerror에 해당 rejection이 남지 않는다. 서버/네트워크 원인은 필요하면 한국어 안내 뒤 괄호로 덧붙인다.
  2) 실제 QR을 검증해 「유효한 방문증」이 표시된 뒤에도 기준정보 실패 안내와 「처리 로비」의 error/helperText가 유지된다. `verify`나 `startCamera`가 공용 error를 비워도 기준정보 장애를 정상으로 바꾸지 않는다.
  3) 실패 여부는 Promise 실패로만 정한다. 아직 로딩 중인 ref=null, 정상 200의 lobbies=[], scope 필터 후 빈 목록은 실패 근거가 아니다. 정상 응답에서는 기존 scope 필터·첫 허용 로비 선택·수동 선택이 유지된다.
  4) 차단 해제 후 페이지를 다시 열어 정상 응답을 받으면 실패 안내가 사라지고 실제 로비를 선택할 수 있다. 기존 keyboard-wedge 체크인→대피 명단 E2E도 통과한다. 체크인에 새 필수 로비 조건을 추가하지 않는다.
  5) 영구 Playwright 테스트가 실제 빌드 UI에서 1~4를 검증한다. 최소 실패→QR 검증→정상 재진입 경로와 정상 빈 응답 경로를 검증하고, 수정 전에는 오류 안내 단정이 실패하고 수정 후에는 통과한 실행 결과를 남긴다.
- 건드릴 파일:
  - `web/src/pages/ScannerPage.tsx:ScannerPage` — useEffect의 API 요청에 catch를 추가하고 기준정보 전용 오류 문자열 하나를 둔다. 별도 Alert와 처리 로비 TextField의 error/helperText가 같은 값을 읽게 한다. 기존 공용 error만 쓰면 verify가 지우므로 부족하다. 새 재시도 버튼 없이 브라우저 새로고침 안내로 한정한다.
  - `web/e2e/visit-flow.spec.ts:login/createVisit 및 scanner 스펙 주변(246행)` — 실제 화면 회귀 스펙 추가. 프로덕션 1파일, 테스트 1파일이다.
- 검증 명령:
  - 저장소 루트에서 `cd web && npm ci && npm run lint && npm test && npm run build` (의존성 미설치 상태이면 npm ci 필요).
  - 저장소 루트에서 `bash scripts/local-e2e.sh` (인자를 받지 않음). 실제 dist 임베드·새 PostgreSQL·실제 서버·브라우저까지 통과해야 한다.
  - 저장소 루트에서 `git diff --check`.
  - 정찰 실측: `go test ./... -count=1` PASS(internal/app 0.141s), `bash -n scripts/local-e2e.sh` 및 `git diff --check` PASS. VISITFLOW_TEST_DSN 미설정으로 DB 통합 SKIP. node_modules가 없어 npm/브라우저는 이번 정찰에서 미실행이다.
- 위험과 피할 것: 카메라 startCamera/stopCamera/timer/cleanup/verify의 QR 처리, scope 계산, checkin, api.ts, 서버, auth/session, migrations, workflows, 릴리즈 스크립트는 변경하지 않는다. 큰 JSX 한 줄을 통째로 포맷하지 않는다. `checkIn`(internal/app/visits.go:1532)은 빈 lobbyId를 받는 계약이 있으므로 로비 미선택을 새 체크인 차단 조건으로 만들지 않는다. 과거 기각된 LobbyPage/VisitsPage 요청 티켓 접근을 합치지 않는다. PDF 재생성·공용 데이터 로더 도입도 제외한다.
- 차선 후보: `web/e2e` 정규 TypeScript 검사 배선 (가치 2 / 위험 2 / S) — `web/tsconfig.json`의 참조에 별도 `tsconfig.e2e.json`을 연결하고 `npm run lint`/build가 e2e와 Playwright config의 타입 오류를 잡게 한다. 현재 타입 오류의 개수는 미확인이다. Scanner 결함이 실제로 이미 해결됐거나 전제가 틀릴 때만 별도 과제로 선택하고 두 일을 합치지 않는다.

구현 순서와 확인 지점 (모두 미착수, 사람 승인 대기 없음):
1. `visit-flow.spec.ts`에 실패 재현을 만든다. login→createVisit로 실제 passUrl을 먼저 만들고 그 뒤 reference-data만 `page.route(...route.abort())`로 차단한다. /lobby/scan에 진입해 안내를 확인하고, QR 입력 및 Enter 후에도 안내가 남는지 확인한다. `page.on("pageerror")`를 수집한다. 증명: 수정 전 `bash scripts/local-e2e.sh`에서 새 안내 단정만 실패하는지 확인한다. 기준 스펙까지 실패하면 환경/선택자 문제부터 분리하고 과제서를 수정한다. 제품 빌드는 이 단계에서도 정상이어야 한다.
2. ScannerPage에 위 국소 변경을 하고 실패 테스트와 복구/정상 테스트를 완성한다. 차단을 풀고 재진입한 정상 응답에서 서버가 실제 준 첫 허용 로비가 선택되는지 확인한다. 정상 빈 목록은 `route.fetch()`로 받은 실제 reference JSON에서 lobbies만 []로 바꾸어 반환하는 HTTP 경계 fixture로 검사해도 된다(빈 목록 UI 증거일 뿐 서버가 빈 로비 테넌트를 생성한다는 증거로 쓰지 않는다). 증명: `cd web && npm run lint && npm test && npm run build`, 이어 `bash scripts/local-e2e.sh`. 단계 종료 시 전부 통과해야 한다.
3. 의도한 두 파일만 바뀌었는지 확인하고 전/후 결과를 회차 노트에 남긴다. 증명: `git diff --check`와 `git status --short`; webdist 스텁은 local-e2e의 백업 복원에 맡긴다. `git checkout --`로 임의 복원하지 않는다.

선택 근거와 대안 비교:
- 공용 error에 catch만 추가: 가장 작지만 verify/startCamera가 오류를 지우므로 수용 기준 2를 만족하지 못한다.
- 전용 오류 값과 국소 안내(선택): 기준정보 실패와 QR 결과를 분리하면서 1개 프로덕션 파일에 끝난다. 별도 데이터 계층이나 의존성이 필요 없다.
- 공용 조회 훅·재시도·캐시 체계: 여러 화면으로 확장할 요구가 있을 때만 정당화된다. 이번에는 기존 성공 경로와 카메라 생명주기를 불필요하게 건드린다.
- 현상 유지/문서만 안내: 로비 담당자에게 실제 실패가 보이지 않아 이번 문제를 해결하지 못한다.

견적 근거: 구현자 1명, 변경 2파일, 기존 login/createVisit 및 실제 E2E 스크립트를 재사용하는 조건이다. 상향식으로 재현 5~8분 + 화면 변경 5~8분 + 검증/스펙 보강 10~14분 + 기록/정리 3~5분 = 기본 23~35분, 알려진 불확실성(선택자·브라우저 준비)에 contingency 5~10분을 별도로 두어 총 28~45분을 예상한다. 신뢰도는 중간(정찰자의 판단이며 통계적 확률 아님). 유사 회차 AppShell/KeysPage도 프로덕션 1파일+실제 E2E였으나 실제 작업시간 기록은 없어 정량 유사견적은 미확인이다. 관리 예비는 배정하지 않으며 범위 밖 카메라/인프라 문제는 추가 과제로 기록한다. 새 브라우저 다운로드나 PostgreSQL 준비 경합으로 시간이 늘면 숨기지 말고 재견적한다.

미확인: 스캐너 전용 새 실패 스펙의 실제 브라우저 결과, MUI 처리 로비의 최종 접근성 선택자, 실제 정상 빈 로비 테넌트. 소스 근거는 확정했지만 정찰에서 UI 재현을 수행했다고 주장하지 않는다.
적용 스킬: 로컬 headcount의 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration SKILL.md를 읽고 견적 범위·예비 분리, 단계별 증명/확인 지점, 대안 비교를 반영했다. Skill 전용 호출 도구는 이 세션에 노출되지 않았다.
