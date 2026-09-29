- 과제: 직원 화면의 조회 필터를 URL에 보존 (가치 3 / 위험 2 / 작업량 M)
- 왜: `EmployeesPage`는 q·status·assignment·organizationId를 빈 useState로 시작하고 최초 effect에서 `load("", "", "", "")`를 호출하여 새로고침·공유 때 조건이 사라진다. 네 필터를 주소에서 복원하고 확정한 조회를 주소에 기록하면 같은 직원 명단을 다시 열 수 있다.
- 수용 기준: 1) `/admin/employees?q=…&organizationId=…&status=active&assignment=assigned` 직접 진입·새로고침 시 입력값, 세 Select, 실제 API 조건과 표가 일치한다. 2) 검색 제출·Select 변경은 다른 조건과 모르는 URL 키를 보존하며 `replace`로 갱신하고, 전체/빈 값은 해당 키를 지운다. 타이핑만으로 요청하거나 주소를 바꾸지 않으며 같은 조건의 검색 재제출·가져오기 후 재조회도 계속 작동한다. 3) Vitest가 읽기/쓰기/API 변환 왕복·공백·빈 값·알 수 없는 enum·입력 URLSearchParams 불변을 검증하고 실서버 E2E가 주소·컨트롤·조회 결과의 일치와 새로고침 복원을 증명한다.
- 건드릴 파일: `web/src/lib/employeeQuery.ts:employeeQuery/EmployeeFilters` — readEmployeeParams/writeEmployeeParams와 공통 정규화 규칙; `web/src/pages/EmployeesPage.tsx:EmployeesPage/load/search/upload/assignFromFile 및 Select onChange` — URL 읽기·쓰기 배선(프로덕션 2파일); `web/src/lib/employeeQuery.test.ts` — 왕복/경계 테스트; `web/e2e/employee-filter.spec.ts` — 기존 3건 유지하고 URL 복원 테스트 추가; `docs/USER_GUIDE.md:3.4 직원`과 생성 `docs/USER_GUIDE.html` — 주소 공유 설명 한 문장.
- 검증 명령: 저장소 루트에서 `go test ./...`(정찰에서 통과). 구현 때 `cd web && npm ci && npm test && npm run lint && npm run build`; 준비된 실서버에서 `cd web && E2E_BASE_URL=http://127.0.0.1:8080 E2E_USERNAME=admin E2E_PASSWORD=ci-e2e-password-123 npx playwright test employee-filter.spec.ts employee-export.spec.ts login.spec.ts`; `python3 scripts/build-docs.py USER_GUIDE`; `git diff --check`.
- 위험과 피할 것: 서버/auth/migrations/workflows, 직원 가져오기 계약, CSV 포맷, 좌석맵 URL 규칙은 범위 밖. API limit은 계속 500이며 주소의 limit 값을 신뢰하지 않는다. enum은 status=active|leave|retired, assignment=assigned|unassigned만 허용하고 나머지는 전체로 읽고 쓰기에서도 제거한다. q·organizationId는 trim하며 키 이름을 서버와 맞춘다. 조직 목록이 늦게 오거나 실패했다고 유효한 주소의 organizationId를 지우지 않는다(선택지 로딩 중에는 빈 표시 또는 임시 항목 사용). 주소 감시와 직접 load가 같은 요청을 중복 실행하지 않게 하고, 동일 조건 재검색을 생략하지 않는다. 검색 전 입력 q도 Select 변경 시 적용하던 기존 동작을 유지한다. 오래된 요청이 새 목록을 덮는 별도 문제를 대규모 공통 훅으로 확장하지 않는다. PDF·캡처 재생성 금지. E2E는 손으로 만든 응답 객체/route.fulfill 없이 실제 서버·DB로 검증한다.
- 차선 후보: E2E 환경 의존 테스트 준비 절차를 `web/e2e/README.md`에 문서화 — 네트워크 구성 때문에 기능 E2E를 회차 내 준비할 수 없을 때만 전환. 아래 환경 설명과 실제 두 spec을 기준으로 작성하고 “항상 실패하는 테스트”로 고정하지 않는다.

## 확인한 근거와 설계 선택
- 기준 `6ae2417`(v1.4.12 문서), 작업 트리 깨끗함. `employeeQuery.ts`는 API 쿼리만 만들며 주소 read/write가 없다. `EmployeesPage.tsx` 44~99 및 326~379에서 빈 초기값·초기 load·각 이벤트를 직접 확인했다. `listEmployees`(`internal/app/employees.go` 59~84)는 같은 네 키를 받고 limit 상한 500, total 없음.
- 옵션 A(선택): 기존 employeeQuery와 React Router useSearchParams를 확장. 프로덕션 2파일, 공유 가능, 기존 seatMapLink 규칙과 일치. 옵션 B: localStorage에 조건 보존(작지만 타인과 공유 불가·숨은 상태). 옵션 C: 공통 필터 훅/저장된 검색 기능(확장성은 좋으나 여러 화면 영향과 새 계약으로 45분 초과). 옵션 D: 현행 유지와 수동 재선택(비용 0이나 주소 공유 목적 미충족). 가장 중요한 가정은 네 조건만으로 동일한 조회를 표현할 수 있다는 것이며 서버와 화면에서 확인했다.
- read는 네 문자열을 반환, write는 기존 params 복사 후 주어진 키만 갱신하고 undefined는 유지한다. 정규화는 읽기/쓰기/API에 공통 적용하되 API 전용 limit과 페이지 주소 계약을 억지로 합치지 않는다. 모르는 키는 URL에 보존하되 API에는 보내지 않는다.
- 제출/Select는 확정 조건을 갱신하고 URL을 replace한다. 주소 변경(뒤로/앞으로 포함)에서도 상태와 목록을 복원한다. 구현 방식은 URL effect 단일 조회 + 동일값 재조회 신호 또는 중복 방지 키 중 단순한 쪽을 택하되 효과 루프와 중복 요청을 테스트한다.

## 실행 순서·검증·체크포인트 (아직 모두 미착수)
1. `employeeQuery.ts`와 짝 테스트에 계약을 추가. 증명: `cd web && npm test -- src/lib/employeeQuery.test.ts`. 기존 8건 유지. 사람 승인 대기 없음; 통과 후 진행.
2. EmployeesPage 배선 및 기존 `employee-filter.spec.ts`에 테스트 추가. 증명: 위 lint/build와 지정 E2E. 영업팀 id는 로그인 세션으로 GET `/api/v1/organizations`에서 실제 응답을 읽고, 영업팀+active+assigned는 시드상 2행, 김개발+개발팀은 1행으로 검증한다. 값 변경→주소 확인→reload→컨트롤/표/CSV 확인, 빈 값 제거와 기존 모르는 키 보존을 확인한다. 사람 승인 대기 없음; 실패하면 계획/구현을 수정하고 다음 단계로 넘어가지 않는다.
3. 새 E2E가 변경 전 `6ae2417` 이미지에서 URL 미복원/미갱신으로 실패하는 역검증과 수정 이미지 통과를 기록. 테스트가 실행조차 못 한 환경 오류는 역검증으로 인정하지 않는다. USER_GUIDE 문장·HTML 갱신 후 전체 단위 테스트/build/diff 검증. 사람 승인 대기 없음; 결과 기록 후 인계.

## E2E 준비와 검증 한계
- `.github/workflows/ci.yml`의 검증된 구성은 PostgreSQL 16-alpine(5432, DB/user/password=seaton), `docker build --build-arg VERSION=e2e -t seaton:e2e .`, `docker run -d --name seaton-e2e --network host -e POSTGRES_DSN='postgres://seaton:seaton@127.0.0.1:5432/seaton?sslmode=disable' -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD=ci-e2e-password-123 seaton:e2e`. 전용 테스트 DB를 준비하고 기존 컨테이너/데이터를 임의 삭제하지 않는다. 준비 확인은 `curl -fsS http://127.0.0.1:8080/readyz`(시간 제한 필수). 브라우저 없으면 web에서 `npx playwright install --with-deps chromium`.
- playwright.config.ts는 서버를 시작하지 않는다. global-setup.ts→seed.mjs가 실제 DB를 시드한다. 3조직·10명 전원 active/배정이며 기존 employee-filter 3건은 데이터를 변경하지 않는다. MCP/추적 spec은 beforeAll에서 가짜 IdP/수집기를 스스로 띄운다. 별도 Keycloak 설치가 필수라는 이전 기록은 부정확하다. Docker bridge에서는 컨테이너→호스트 접근 및 E2E_COLLECTOR_HOST 설정이 필요하다.
- 정찰 실행 결과: `go test ./...` 성공. web/node_modules가 없어 프런트 테스트·build·브라우저는 미실행; 저장소에 파일을 쓰지 말라는 정찰 제한 때문에 npm ci도 하지 않았다. E2E 실제 통과 여부와 조직 로딩 지연 표시는 미확인이다. 문자열 검색은 위치 파악에만 사용했으며 런타임 증거가 아니다.

## 작업량과 예비 시간
- 방법: 하향 예산 맞추기가 아닌 작업 분해. 헬퍼/테스트 6~8분 + 화면 배선 8~10분 + 실서버 역검증/회귀 10~15분 + 문서/기록 3~4분 = 기본 27~37분. 알려진 불확실성(캐시·조직 로딩·중복 effect)에 별도 예비 5~8분, 합계 32~45분의 조건부 추정(정찰자 판단, 통계적 신뢰수준 미산정).
- 전제: Docker·PostgreSQL·Chromium 및 의존성 캐시 사용 가능. 새 환경 설치 시간이 길면 45분 보장 불가; 첫 환경 점검에서 차선 전환 판단. 범위 미상의 작업을 위한 관리 예비는 배정하지 않음(0분); 범위 확장은 보류.
- 유사 비교: 09-27 좌석맵 URL 작업도 헬퍼+화면+실서버 역검증으로 성공. 이번에는 필터 4개와 조직 비동기 선택지가 있어 S보다 M이 타당하다. 과거 총 소요시간 데이터는 없어 숫자 배율 비교는 미확인.
- 적용 스킬: 로컬 headcount 정본 `pmo/skills/estimating-and-contingency/SKILL.md` 및 `references/sources.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`를 읽었다. 루트는 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/`. 전용 Skill 호출 도구는 제공되지 않아 파일 읽기로 적용했으며 호출했다고 주장하지 않는다. 외부 추정 표준의 수치·신뢰도를 차용하지 않았다.
