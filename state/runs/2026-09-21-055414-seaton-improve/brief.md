- 과제: 좌석 상세에서 직원 소속과 지정 구역 구분 (가치 3 / 위험 1 / 작업량 S)
- 왜: SeatMapPage의 상세 ‘조직’은 selected.organizationName(좌석 구역)을 직원 소속보다 먼저 표시하고, 구역이 없으면 검색 결과에 없는 직원의 소속을 ‘정보 없음’으로 표시한다. 이미 좌석 API가 보내는 employeeOrganizationName을 사용하고 구역을 별도 행으로 보여 주면 검색 여부와 무관하게 실제 소속을 확인하고 구역 불일치의 양쪽 조직을 구분할 수 있다.
- 수용 기준: 1) 배정 좌석의 상세에 ‘직원 소속’과 ‘지정 구역’을 서로 다른 행으로 표시한다. 직원 소속은 selected.employeeOrganizationName으로 표시하고 비어 있으면 ‘소속 없음’, 지정 구역은 selected.organizationName 또는 ‘미지정’으로 표시한다. 검색 결과(selectedEmployee)나 지정 구역을 직원 소속의 대체 값으로 사용하지 않는다. 2) 직원 검색 없이 도면 좌석을 직접 클릭해도 표시가 정확하며, 빈 좌석에도 ‘지정 구역’ 행과 기존 ‘배정된 직원이 없습니다.’ 안내가 보인다. 직원 소속과 다른 구역 지정·해제 후 저장 및 새로고침 때도 양쪽 값이 정확하고 직원 배정·소속·조직 색·불일치 표시는 유지된다. 3) 실제 SeatOn+PostgreSQL Playwright 테스트가 김개발(개발팀)의 구역을 다른 팀으로 설정했을 때 상세의 직원 소속=개발팀/지정 구역=다른 팀을 각각 해당 행에서 검증하고, 구역 해제 시 직원 소속 보존/지정 구역=미지정, 빈 좌석의 구역 지정/미지정도 검증한다. 기존 구현에서 핵심 불일치 사례가 실패하고 수정 후 성공해야 한다. 소속 없는 직원은 가능하면 임시 직원 fixture로 검증하고 반드시 정리한다.
- 건드릴 파일: web/src/pages/SeatMapPage.tsx:SeatMapPage의 selected.employeeId 상세 분기(현재 약 2338~2387행), Info — 기존 ‘조직’ 행을 직원 소속으로 바꾸고 지정 구역 행을 배정 여부 분기 밖에 추가; selectedEmployee는 근무지에서 사용하므로 삭제하지 않는다. web/e2e/seat-organization.spec.ts:기존 ‘기존 좌석의 지정·변경·해제와 직원 소속 우선 색을 유지한다’ 테스트와 신규 좌석 테스트 — 상세 행별 확인을 추가하거나 같은 파일에 독립 회귀 테스트 추가. docs/USER_GUIDE.md:§2 좌석맵 및 첫 검색 설명 — 두 값의 의미 설명. docs/USER_GUIDE.html — 해당 가이드만 재생성.
- 검증 명령: 저장소 루트에서 `go test ./...`(정찰에서 성공, 캐시 사용); `cd web && npm ci && npm test && npm run build`; 실서버 준비 후 web에서 `E2E_BASE_URL=http://127.0.0.1:18789 E2E_USERNAME=admin E2E_PASSWORD="$SEATON_E2E_PASSWORD" npx playwright test e2e/seat-organization.spec.ts e2e/seatmap.spec.ts`; 루트에서 `python3 scripts/build-docs.py USER_GUIDE` 및 `git diff --check`. npm 스크립트·Playwright 파일·문서 생성 명령은 현재 저장소에서 확인했지만 정찰에서는 프런트 의존성 미설치로 실행하지 않았다.
- 위험과 피할 것: 서버 API·auth.go·migrations.sql·.github/workflows는 변경 불필요. 좌석의 organizationName과 employeeOrganizationName을 섞지 말고, 현재 직원 소속 우선 색상 규칙(seatOrgId)도 바꾸지 않는다. 테스트가 바꾼 organizationId는 finally에서 원값 또는 빈 문자열로 PATCH해 복구한다(helpers.restoreSeat는 좌표만 복구). 신규 임시 좌석은 삭제하고, 조직명 전역 검색으로 통과시키지 말고 상세의 라벨/값 행에 범위를 좁힌다. 최근 구역 편집 과제의 재구현이 아니라 남아 있는 상세 표시 오류 수정이다. PDF·캡처·기존 다른 문서 산출물은 범위 밖. USER_GUIDE는 미머지 OAuth 브랜치에도 변경이 있으니 관련 문단만 좁게 수정한다.
- 차선 후보: build-docs.py 파서 단위 테스트 — 첫 과제가 다른 변경으로 이미 해결된 경우에만 parse_markdown의 이미지·순서 목록 번호 이어 세기를 Python 표준 unittest로 검증한다. scripts/build-docs.py의 parse_markdown과 Block/Document를 실제로 읽었으며, 새 테스트 파일은 구현자가 추가해야 한다. 기존 CI는 변경하지 않는다.

근거 및 구현 순서
1. internal/app/seats.go:listSeats는 organizations o(좌석 구역)와 organizations eo(직원 소속)를 별도로 JOIN해 두 이름을 이미 반환한다. web/src/types.ts:Seat도 두 필드를 갖는다. API 확장은 필요 없다.
2. web/src/pages/SeatMapPage.tsx:runSearch는 검색할 때만 employees를 채우며, selectedEmployee는 이 배열에서만 찾는다(약 1314행). 현재 상세는 selected.organizationName || selectedEmployee?.organizationName || ‘정보 없음’이다. 직원 소속은 이미 tooltip/aria-label에서 쓰는 selected.employeeOrganizationName을 사용한다.
3. 기존 seat-organization.spec.ts는 구역 변경 후 API 값·직원 불변·색·불일치·구역 수는 검사하지만 상세 ‘조직’ 행은 검사하지 않는다. 기존 반복 중 other 구역일 때 핵심 회귀를 먼저 추가해 실패를 확인한 뒤 UI를 수정한다. 검색 없이 클릭하는 경로와 새로고침도 포함한다.
4. 단순 JSX 표시 변경에 동일한 구현을 따라 쓰는 순수 함수 테스트를 새로 만들 필요는 없다. DB와 화면을 통과하는 기존 E2E가 증거다.

시간 산정과 여유(45분)
- 기존 테스트에 실패 사례 추가 7분, 화면 변경 5분, 나머지 표시/복구 검증 8분, 가이드 3분, 테스트·빌드 12분 = 기본 35분 + 환경 준비/수리 여유 10분.
- 실서버 준비가 가장 큰 불확실성이다. 기존 검증 환경이 없다면 아래 격리 환경으로 준비한다. 시간을 초과하면 실서버 검증 미완료를 보고하고 통과로 간주하지 않는다. 환경 장애만을 이유로 별도 과제를 함께 구현하지 않는다.

실서버 검증 준비(구현자 실행용, 정찰에서는 실행하지 않음)
- compose.yaml은 PostgreSQL을 띄우지 않으며 기존 latest 이미지를 사용할 수 있으므로 수정본 검증은 Dockerfile로 새 이미지를 빌드한다. 아래 이름/포트가 비어 있는지 먼저 확인하고 충돌 시 바꾼다. SEATON_E2E_PASSWORD는 검증용 12자 이상 값으로 설정한다.
```bash
export SEATON_E2E_PASSWORD='seaton-local-e2e-2026'
docker network create seaton-detail-e2e
docker run -d --name seaton-detail-db --network seaton-detail-e2e -e POSTGRES_USER=seaton -e POSTGRES_PASSWORD=seaton -e POSTGRES_DB=seaton postgres:16-alpine
docker build -t seaton:detail-e2e .
docker exec seaton-detail-db pg_isready -U seaton
docker run -d --name seaton-detail-app --network seaton-detail-e2e -p 127.0.0.1:18789:8080 -e POSTGRES_DSN='postgres://seaton:seaton@seaton-detail-db:5432/seaton?sslmode=disable' -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD="$SEATON_E2E_PASSWORD" seaton:detail-e2e
curl -fsS http://127.0.0.1:18789/readyz
cd web
npx playwright install chromium
E2E_BASE_URL=http://127.0.0.1:18789 E2E_USERNAME=admin E2E_PASSWORD="$SEATON_E2E_PASSWORD" npx playwright test e2e/seat-organization.spec.ts e2e/seatmap.spec.ts
```
- pg_isready와 readyz가 성공한 뒤 다음 단계로 진행한다. 브라우저 OS 의존성이 없으면 README/CI의 `npx playwright install --with-deps chromium`을 사용한다. global-setup.ts는 서버를 시작하지 않으며 seed.mjs가 실제 CV로 도면을 분석한다. 완료 후 이번에 만든 두 컨테이너만 `docker rm -f -v seaton-detail-app seaton-detail-db`로 제거하고 `docker network rm seaton-detail-e2e`로 네트워크를 정리한다.

정찰 제한: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 호출 가능한 Skill 도구·제공 스킬 목록 및 검색한 로컬 스킬 경로에서 발견되지 않았다. 스킬 자체의 절차/반환 형식은 미확인이고 위 비교·계획·산정은 사용자 프롬프트에 따른 작성이다. 브라우저의 실제 오표시는 미재현이나 JSX/API 데이터 경로로 결함을 확인했다.
