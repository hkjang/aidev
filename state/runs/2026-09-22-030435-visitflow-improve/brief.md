- 과제: 일반·현장 방문 신청 화면에 회사명 필수 정책을 반영 (가치 3 / 위험 1 / 작업량 M)
- 왜: 서버의 createVisitRecord는 visit.company_required=true일 때 공백 회사명을 400 company_required로 거절하지만, referenceData 응답과 VisitFormPage에는 정책 전달·필수 표시·제출 전 검사가 없어 작성자가 제출한 뒤에야 이유를 알게 된다. 일반 신청과 현장 등록이 공유하는 화면에 같은 정책을 연결하면 수동 입력·파일 가져오기·템플릿으로 들어온 모든 방문자의 누락을 제출 전에 알 수 있다.
- 수용 기준:
  1) GET /api/v1/reference-data에 companyRequired라는 JSON boolean을 추가한다. 기존 visit.company_required 값이 정확히 "true"일 때만 true이며 기본값·키 없음은 false다. 관리자 설정 전체나 비밀값을 노출하지 않는다.
  2) /visits/new 및 /lobby/walk-in에서 정책이 켜지면 모든 방문자 회사명에 required 표시와 "현재 정책상 회사명은 필수입니다" 안내를 제공한다. 한 명이라도 company.trim()이 비면 제출 버튼을 비활성화하고 submit에서도 같은 조건을 검사한다. 정책이 꺼지면 빈 회사명을 계속 허용한다. 다른 필수값·체크리스트·담당자 조건은 그대로 유지한다.
  3) PostgreSQL 실제 라우터 테스트에서 PUT /api/v1/settings로 false→true→false를 저장하고 매번 reference-data boolean과 방문 생성 결과가 일치함을 증명한다. 실제 브라우저에서 두 화면 모두 빈 값·공백만·정상 값·두 번째 방문자 누락을 확인하고, 실제 CSV 업로드로 들어온 회사명 누락도 같은 제출 차단에 걸림을 증명한다. true에서 회사명을 채우면 실제 등록 성공, false에서 회사명 없이 실제 등록 성공을 확인한다. API 직접 호출의 기존 400 company_required 보호도 유지한다.
- 건드릴 파일:
  - internal/app/visits.go: Server.referenceData — 기존 selfRegistrationEnabled 옆에 companyRequired 추가. Server.createVisitRecord / Server.createWalkIn은 읽고 계약을 보존하며 변경하지 않는 것이 기본.
  - web/src/types.ts: ReferenceData — companyRequired 필드 추가(구버전 응답 호환을 위해 optional boolean 가능, 화면에서는 === true로 읽기).
  - web/src/pages/VisitFormPage.tsx: VisitFormPage, submit, 회사명 TextField, 제출 Button — 동일한 파생 유효성 값을 표시·버튼·submit 가드에서 재사용. importVisitors/templateVisitorDraft가 만든 visitors에도 동일하게 적용. 입력 값을 자동 보완하거나 잘라내지 않는다.
  - internal/app/company_policy_test.go: 신규 TestCompanyRequiredPolicy — 기존 integration_test.go의 newTestEnv, env.json, env.do, visitBody를 재사용. 미머지 메일 브랜치도 integration_test.go를 수정했으므로 그 파일의 편집은 피한다.
  - web/e2e/visit-flow.spec.ts: login/createVisit 패턴을 활용한 company policy 회귀 테스트 — 실제 서버·DB·브라우저 사용. 설정 원래 값은 테스트 시작에 읽고 finally에서 복원한다.
  - docs/USER_GUIDE.md: 방문 신청 설명에 정책 활성 시 필수 안내와 제출 차단을 한 문장 추가. 가져오기 관련 문단은 진행 중 브랜치와 충돌하므로 건드리지 않는다. PDF 재생성 환경은 이번 미확인; 필요 시 별도 결과로 명시하고 변경 범위를 키우지 않는다.
- 검증 명령:
  - 이번 정찰에서 실제 실행·통과: `go test ./... -count=1` (app 0.137s). VISITFLOW_TEST_DSN 미설정으로 DB 통합은 SKIP; 이 결과는 수용 기준 3의 증거가 아니다.
  - 구현 후 DB 준비 상태에서: `VISITFLOW_TEST_DSN='postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable' go test ./internal/app -run TestCompanyRequiredPolicy -count=1 -v`
  - 이어서 동일 DSN으로 `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `git diff --check`.
  - 프런트: `cd web && npm ci && npm run lint && npm test && npm run build` (lint는 tsc이며 vitest는 src/**/*.test.ts만 수집한다).
  - E2E: 실제 최신 UI를 제공하는 서버 준비 후 `cd web && VISITFLOW_BASE_URL=http://127.0.0.1:8080 VISITFLOW_E2E_ADMIN=admin VISITFLOW_E2E_PASSWORD=e2e-bootstrap-password npm run test:e2e -- --grep 'company policy'`. 테스트 제목에 해당 문자열을 넣는다. 마지막에는 `npm run test:e2e`로 기존 흐름도 확인한다.
  - 서버 준비 계약은 .github/workflows/ci.yml의 e2e 잡과 README 개발·검증 절에서 확인했다: PostgreSQL, npm 빌드, web/dist를 cmd/visitflow/webdist로 복사, `go build -o visitflow ./cmd/visitflow`, POSTGRES_DSN·BOOTSTRAP_ADMIN·BOOTSTRAP_ADMIN_PASSWORD·ENCRYPTION_KEY 네 환경변수로 실행, `/readyz` 확인, `npx playwright install --with-deps chromium`. 실행 중 서버/Chromium 가용 여부는 정찰에서 미확인이다. 기본 임베드 스텁만 든 Go 바이너리로 UI 검증했다고 보고하지 않는다.
- 위험과 피할 것: auth.go, migrations/, .github/workflows/, settings.go, SettingsPage.tsx, mail*, MCP OAuth는 수정하지 않는다. 서버 필수 검증을 느슨하게 하거나 관리자 설정 API를 일반 신청 화면에서 조회하지 않는다. 정책의 실시간 push 갱신·셀프 사전등록·새 정책 체계·가져오기 파서 변경은 범위 밖이다. 로드 후 관리자가 정책을 바꾸면 기존 서버 오류가 최종 방어선인 동작을 유지한다. 미머지 중복 헤더 과제(8cc7374)를 재구현하지 않는다. 변이 검증 시 git checkout --로 작업을 잃지 말고 정확한 역패치로 복원한다. 테스트 대역이나 소스 문자열 검사만으로 API→UI 배선을 증명하지 않는다.
- 차선 후보: 방문자 추가 버튼을 100명에서 차단 (가치 2 / 위험 1 / 작업량 S) — 현재 VisitFormPage의 추가 버튼은 제한 없이 append하며 createVisitRecord는 100명 초과를 거절한다. 100명에서 추가 비활성화, 99명으로 삭제하면 복구, CSV 100명 로드 뒤에도 동일하게 적용하는 실제 화면 회귀를 붙인다. 1순위가 이미 다른 변경으로 해결됐거나 45분 안에 API·UI 양쪽 검증이 불가능함을 확인한 경우에만 선택한다. 두 과제를 함께 하지 않는다.

확인 근거와 대안 판단
- 기준 HEAD 6a3ed81(v2.8.3), 깨끗한 작업 트리. 서버 검사 visits.go:452–459, 응답 referenceData:80, 프런트 ReferenceData:79 및 VisitFormPage의 회사명 필드/제출 조건을 직접 읽었다. 브라우저에서 오류를 직접 재현한 것은 아니며 위 결함 판단은 소스 배선에 근거한다.
- 선택안은 기존 reference-data에 비밀이 아닌 boolean 하나를 싣는 최소 변경이다. 서버 오류 안내만 보강하는 안은 제출 후에야 알게 되는 문제를 남긴다. 일반적인 폼 정책 스키마를 만드는 안은 확장성은 있지만 한 정책을 위해 API와 화면을 과도하게 바꾼다. 현상 유지는 보안상 서버 차단은 유지되나 반복 입력 실패를 해결하지 못한다.
- 결정의 핵심 가정: 일반·현장 화면이 공유하는 visitors 유효성 계산으로 수동·CSV·템플릿 모두를 커버할 수 있다(공유 상태 경로는 소스로 확인). 실제 브라우저 실행 환경 확보 시간은 미확인이다.

구현 순서·검토 지점 (모두 미착수, 사람 승인 대기 없음)
1. [ ] company_policy_test.go에 실제 설정 API/조회/등록 회귀를 작성하고 referenceData와 ReferenceData를 함께 수정한다. 증명: 위 TestCompanyRequiredPolicy 명령과 npm run lint. 신규 필드 없이는 회귀가 실패하는지 확인. 체크포인트: boolean·false 기본값·기존 생성 보호를 자동 검토한 뒤 진행.
2. [ ] VisitFormPage의 공통 유효성·안내·submit 가드와 실제 브라우저 company policy 테스트를 추가한다. 설정 변경은 인증된 PUT /api/v1/settings, body는 {"settings":{"visit.company_required":"true"}}이고 X-CSRF-Token은 실제 /api/v1/auth/me 응답에서 얻는다(기존 E2E의 page.evaluate 패턴 참고). 두 경로에서 API 응답을 가짜로 주입하지 않는다. 증명: lint/build 및 필터 E2E. 체크포인트: 두 화면·전체 방문자·공백·CSV 배선 확인 후 진행.
3. [ ] 사용자 설명 한 문장을 반영하고 전체 검증 명령을 실행한다. 체크포인트: 실제 실행과 미실행/SKIP를 분리해 보고하고 종료. 계획과 코드가 어긋나면 과제서 상태/근거를 갱신하고 범위를 재판단한다.

작업량 근거 (pmo:estimating-and-contingency / technology:implementation-planning / technology:solution-exploration 적용)
- 상향식 기본 추정: API·Go 통합 회귀 8–10분, 공유 화면·실제 브라우저 회귀 12–16분, 문서·전체 검증 6–9분 = 26–35분. 알려진 불확실성인 브라우저 선택자/설정 복구에 예비 5–8분을 별도로 두어 총 31–43분(M), 신뢰 중간의 주관적 계획 범위다. 통계적 보장이나 80% 확률 주장은 아니다.
- 비교: 이전 가져오기 S 회차보다 API→타입→두 화면 검증이 늘어 M이 타당하다. 이전 기록에는 총 작업시간이 없어 유사 추정을 분 단위로 보정하지 않았다. PostgreSQL·Node·Chromium 사용 가능을 전제로 하며 새 인프라 설치 문제는 범위 밖이다. 첫 체크포인트에서 재추정한다.
- 미지의 범위 확장용 관리 예비는 이번 45분 회차에 배정하지 않는다(0분). 예비분을 새 기능에 쓰지 말고, 환경 장애가 예산을 넘으면 검증 미완료를 명시한다.
