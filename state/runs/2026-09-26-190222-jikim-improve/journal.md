# 회차 노트 2026-09-26-190222-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:02] base pinned — main@6424731
- [러너 19:02] autonomy release — 
- [러너 19:12] scout timeout — 단계 제한 시간 초과
- [러너 19:12] scout done — 추적이 꺼져 있어도 인증 없는 CSP 리포트가 관리자 진단 목록을 채우는 문제 막기 (가치 3 / 위험 1 / 작업�

## 구현 노트
- 무엇/왜: `receiveCSPReport`가 추적 설정을 보지 않아, `report-uri`를 한 번도 내보낸 적 없는 기본 설치에서도 인증 없는 리포트가 관리자의 "차단된 출처" 목록(MaxViolations=100, 항목마다 script-src 허용 버튼)을 채웠다. `tracking.Config.ReportingActive()`를 추가하고 `if s.violations == nil` 바로 뒤에 게이트 한 줄을 넣었다. 커밋 `90bee37`(프로덕션 2파일, 테스트 2파일).
- 확신 없는 곳·검증 못 한 것: (1) 실제 브라우저가 `report-uri`로 보내는 리포트를 이 환경에서 왕복시키지 못했다 — 테스트는 `httptest` 로 만든 `application/csp-report` 요청이다. (2) 인증 없는 경로에서 설정을 읽는 호출이 하나 늘었다(리포트 1건당 store 읽기 1회). `decorateIndex`가 이미 모든 SPA 요청마다 같은 호출을 하므로 새 종류의 부하는 아니지만, 리포트 폭주 시 이 읽기가 곱해지는 점은 속도 제한 부재(ideas.json의 3/3/M 항목)와 함께 남는다. (3) `tracking.Config.Snippet("")` 를 게이트에서 부르므로 매 리포트마다 스니펫 문자열을 한 번 렌더한다 — `Active`도 같은 일을 하니 관례는 지켰지만 저렴하진 않다.
- 일부러 하지 않은 것: `Active(path)`의 본문·시그니처 불변(리포트의 `document-uri`는 공격자가 정하는 값이라 게이트로 쓸 수 없고, `IncludeAdmin` 구성에서도 리포트는 받아야 한다). `MaxViolations`·`evictOldest`·라우트 등록·`securityHeaders`·`auditablePath` 미변경. 추적 on 상태의 폭주 방어(속도 제한/동일 출처)는 새 계약이 필요해 제외. `web/`·`CHANGELOG.md`·`scripts/version.sh` 미변경.
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(`trackingServer` 하네스). 응답 계약은 **모든 분기에서 본문 없는 204** 이므로 off 상태에 403/404를 주는 "개선"은 추적 설정 노출이 되어 되돌리기다. 기존 `TestPolicyReportsAreRecordedOnceAndListedForAdministrators`는 한 글자도 수정하지 않았고 그대로 통과한다. `./scripts/verify.sh` 전체 exit 0 (Go test·vet·gofmt, npm ci·vitest 59개·lint·build, 문서·Compose).
- [러너 19:16] brief accepted — 채택 — 근거가 코드와 정확히 맞았고(tracking.go:138-142에 설정 확인이 없었다) 지정한 파일 3개와 게이트 판정식을 그대로 
- [러너 19:16] verify passed — 검증 7개 통과 (auto)
