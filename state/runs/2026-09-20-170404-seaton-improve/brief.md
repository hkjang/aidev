- 과제: 좌석 편집창에서 조직 구역 지정·변경·해제 (가치 3 / 위험 2 / 작업량 M)
- 왜: 좌석맵은 조직 목록과 구역 표시를 이미 제공하지만 SeatEditor에는 조직 입력이 없고 saveSeat도 organizationId를 보내지 않아 빈 좌석의 구역을 화면에서 지정할 수 없다. 기존 생성·수정 API를 연결하면 좌석 관리자가 API 호출 없이 구역을 설정하고 구역 불일치 표시까지 확인할 수 있다.
- 수용 기준: 1) 좌석 관리자 이상의 좌석 추가·보정 창에 접근 가능한 이름 «조직 구역»의 선택란과 «미지정» 항목이 있고 기존 값을 미리 표시한다. 기존 좌석의 조직 지정·다른 조직으로 변경·해제가 저장 및 새로고침 후 유지되며 다른 필드만 수정하면 구역이 보존된다. 2) 신규 좌석도 조직 지정 또는 미지정으로 정상 생성된다. 취소는 저장하지 않으며 직원 배정·직원의 소속을 변경하지 않는다. 기존 조직 색의 직원 소속 우선 규칙과 구역 표시·불일치 판단은 유지되고 저장 후 새 데이터에 따라 갱신된다. 3) 실제 SeatOn+PostgreSQL을 대상으로 Playwright가 UI 조작→POST/PATCH→GET 재조회→페이지 새로고침 후 재개방을 검증한다. 생성의 지정/미지정, 수정의 지정/변경/해제/다른 필드만 수정/취소를 다루며 배정 직원의 소속·배정이 유지되고 다른 조직을 지정하면 불일치 표시가 생기고 해제하면 사라짐을 증명한다. API 모킹·소스 문자열 검사를 증거로 삼지 않는다.
- 건드릴 파일: `web/src/pages/SeatMapPage.tsx: SeatMapPage/SeatEditor/saveSeat/openNewSeat` — loadBase가 이미 읽은 organizations를 편집창에 전달하고 선택란 및 요청 직렬화만 추가한다. `web/e2e/seat-organization.spec.ts`(신규 계획 파일) — 읽어 본 `seat-edit.spec.ts`와 `helpers.ts: login/csrfToken` 패턴으로 실제 서버 회귀 검증을 추가한다. `docs/USER_GUIDE.md: §2 배치 편집 및 구역 용어` — 현재 «화면에서 직접 고치지 않고»라는 설명을 갱신한다. `docs/USER_GUIDE.html` — 기존 스크립트로 생성한다.
- 검증 명령: 저장소 루트 `go test ./...`; `cd web && npm ci && npm test && npm run build`; 실제 서버 환경을 아래와 같이 띄운 뒤 `cd web && E2E_BASE_URL=http://127.0.0.1:18782 E2E_USERNAME=admin E2E_PASSWORD=seaton-org-e2e-pass-123 npx playwright test e2e/seat-organization.spec.ts e2e/seat-edit.spec.ts e2e/actions.spec.ts e2e/seatmap.spec.ts`; `python3 scripts/build-docs.py USER_GUIDE`; 마지막 `git diff --check`.
- 위험과 피할 것: auth·migrations·workflows·settings·seats.go·MCP/메일 배선은 변경하지 않는다. PATCH updateSeat는 organizationId 빈 문자열일 때 NULL로 해제하며 JSON null/생략은 기존 값을 보존한다. 반면 POST createSeat는 빈 문자열을 그대로 FK에 넣으므로 미지정은 null 또는 생략으로 보내야 한다. 생성과 수정에 같은 빈 문자열 body를 무조건 쓰지 말 것. chooseMap은 선택·undo를 초기화하는 기존 동작이므로 저장 후 자동 재선택을 새로 설계하지 않는다. e2e/helpers.ts의 restoreSeat는 x/y만 복구하므로 구역 복구에 재사용하지 말고 원래 organizationId를 직접 보관해 finally에서 PATCH(원래 없음이면 빈 문자열)한다. 생성한 테스트 좌석은 배정하지 않고 finally에서 DELETE하며 응답 성공도 확인한다. 조직 편집이 직원 소속 편집으로 보이지 않게 선택란에 «좌석에 지정할 구역이며 직원 소속은 바뀌지 않습니다»를 안내한다. PDF·스크린샷 재생성 및 조직 일괄 변경은 범위 밖이다.
- 차선 후보: PATCH /users/{id}의 없는 사용자에 대한 204를 404로 수정 — UI 과제가 이미 다른 변경으로 해결된 경우에만 선택. 확인한 `internal/app/auth.go:updateUser`는 UPDATE Exec의 RowsAffected를 버리고 곧바로 감사 기록과 204를 남긴다. tag.RowsAffected()==0이면 기존 not_found 응답으로 종료하고 실제 서버 `web/e2e/admin.spec.ts`에서 존재하지 않는 id의 role-only/active-only/email 요청 모두 404, 기존 대상 정상 요청 204와 감사 로그 미생성을 확인한다. 보호 경로 auth.go는 이 함수에 한정하고 authenticate/OIDC는 손대지 않는다.

구현 근거 및 접근 비교

- 기준 HEAD e0956e1. `internal/app/seats.go:createSeat/updateSeat/listSeats`, `internal/database/migrations.sql:seats.organization_id`, `internal/app/server.go`의 requireSeatManager 그룹을 읽어 서버 지원과 권한을 확인했다. `web/src/types.ts:Seat/Organization`에 필요한 필드가 이미 있다.
- 채택: 기존 편집창과 기존 API 연결. 기능을 한 번에 이해할 수 있고 서버/스키마 변경이 없다. 확장안인 여러 좌석 일괄 구역 지정은 선택·undo·일괄 API까지 넓어져 45분 범위를 넘는다. API만 계속 쓰거나 처리필요의 영역 맞춤만 쓰는 현상 유지는 빈 좌석의 구역 지정 문제를 해결하지 못한다. 별도 조직 관리 화면은 이번 문제에 필요하지 않다.
- 가장 큰 전제는 프런트에서 생성/수정의 빈 값 계약을 구분하면 기존 서버만으로 완결된다는 점이다. 소스에서 확인했으나 이번 정찰에서는 실제 DB 왕복을 실행하지 않았으므로 첫 통합 검증에서 확인한다.

실행 순서 (전부 pending, 사람 승인 체크포인트 없음)

1. SeatEditor의 조직 선택과 saveSeat 직렬화를 한 단위로 구현한다. 수정은 `editor.organizationId ?? ""`, 생성은 빈 값일 때 null 또는 필드 생략으로 분기한다. 증거: `cd web && npm test && npm run build`. 빌드 성공을 확인하고 다음 단계로 간다.
2. 독립 e2e 파일을 추가한다. UI로 저장하고 API GET과 다시 연 편집창으로 값을 확인한다. 조직 목록은 API가 반환한 실제 id를 사용하고 좌석은 고유 번호로 생성한다. 기존 배정 좌석의 구역을 잠시 바꾸는 검증은 원상복구를 보장한다. 증거: 위 Playwright 네 파일 실행. 실패가 계약 차이를 드러내면 범위를 늘리지 말고 이 과제서를 수정해 기록한다.
3. 사용자 가이드의 직접 편집 불가 설명을 고치고 HTML만 생성한다. 증거: `python3 scripts/build-docs.py USER_GUIDE` 및 `git diff --check`; 변경 목록이 위 범위 안인지 확인한다. PDF는 릴리즈 문서 정비로 남긴다.

실제 서버 준비 (구현 단계에서 실행; 정찰에서는 실행하지 않음)

CI의 Dockerfile·postgres:16-alpine·환경변수 구성을 따르되 별도 네트워크/포트/컨테이너 이름을 사용한다. 동일 이름이 이미 있거나 18782가 점유됐으면 다른 검증 세션을 삭제하지 말고 이름·포트를 변경한다.

```sh
docker build --build-arg VERSION=e2e -t seaton:org-e2e .
docker network create seaton-org-e2e-net
docker run -d --name seaton-org-e2e-db --network seaton-org-e2e-net -e POSTGRES_USER=seaton -e POSTGRES_PASSWORD=seaton -e POSTGRES_DB=seaton postgres:16-alpine
docker run -d --name seaton-org-e2e-app --network seaton-org-e2e-net -p 127.0.0.1:18782:8080 -e POSTGRES_DSN='postgres://seaton:seaton@seaton-org-e2e-db:5432/seaton?sslmode=disable' -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD=seaton-org-e2e-pass-123 seaton:org-e2e
curl --retry 30 --retry-connrefused --retry-all-errors --retry-delay 1 --fail http://127.0.0.1:18782/readyz
cd web
npx playwright install --with-deps chromium
E2E_BASE_URL=http://127.0.0.1:18782 E2E_USERNAME=admin E2E_PASSWORD=seaton-org-e2e-pass-123 npx playwright test e2e/seat-organization.spec.ts e2e/seat-edit.spec.ts e2e/actions.spec.ts e2e/seatmap.spec.ts
```

Playwright globalSetup는 seed.mjs로 실제 CV 분석을 포함한 시드를 만든다. 검증을 마치면 이번에 만든 컨테이너 두 개와 네트워크만 정리한다. tracking.spec는 이 범위에 없으며 전체 스위트를 브리지로 돌릴 때는 별도의 수집기 호스트 연결 설정이 필요하다.

추정 근거 (pmo:estimating-and-contingency)

- Bottom-up 기본 작업: UI/직렬화 8~10분, 실제 서버 테스트 추가 10~12분, 이미지·브라우저 검증 9~11분, 문서·최종 점검 3~4분으로 30~37분. 알려진 변동(이미지 빌드/셀렉터 안정화)에 contingency 5~8분을 별도로 두어 총 35~45분, 확신은 중간이며 통계적 달성확률은 미확인이다. 관리 예비비는 배정하지 않으며 새 서버 기능은 범위 확장으로 남긴다.
- 유사 추정 교차 확인: 09-17 계정 메일/사용 여부 편집은 서버까지 바꾼 S 작업이었고 이번에는 서버 변경이 없다. 다만 생성/해제와 실제 화면 검증 조합 때문에 M으로 보수적으로 분류했다. 과거 실제 소요 시간 기록이 없어 수치 교차 검증은 할 수 없다.
- 전제: npm/Docker/Chromium 설치·이미지 접근 가능. 정찰 환경에는 node/npm/docker 명령은 있으나 web/node_modules는 없다. 환경 준비가 길어지면 검증을 생략하지 말고 소요 추정을 갱신한다.
- 추정 방식의 출처: 작업 분해·전제·위험·결과 기록은 [GAO Cost Estimating and Assessment Guide 요약](https://www.gao.gov/products/gao-20-195g)의 절차를 참고했다. 위 분 수치는 정찰자의 코드 기반 추정이며 GAO에서 제시한 수치가 아니다.

정찰 검증 상태

`go test ./...` 성공(app/tracking 캐시 사용, platform 실행). 프런트와 Docker e2e는 미실행. 검증 명령은 package.json·Playwright 설정·CI·Dockerfile에서 확인했으며 신규 e2e 파일은 구현 시 추가해야 한다. CLAUDE.md/AGENTS.md는 저장소 검색에서 없었고 TODO/FIXME 검색에도 결과가 없었다. 코드 수정·커밋 없음.
