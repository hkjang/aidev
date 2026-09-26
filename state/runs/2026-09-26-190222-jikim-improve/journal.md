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

## 비평 노트
- 판정 approve (blocking 없음). 게이트(`internal/httpapi/tracking.go:146`)를 실제로 지우고 새 테스트를 돌려 off 5건·outage 2건이 관리자 목록에 들어가며 두 서브테스트가 FAIL 함을 확인했다 — 테스트는 대상을 정말 실행한다. 복원 후 `go test ./... -count=1`·`go vet ./...`·`gofmt -l .` 통과, 트리 깨끗.
- 방향 검사: `pagePolicy` 는 `Active(path) && nonce!=""` 일 때만 `report-uri` 를 붙이고 `ReportingActive()` 는 `Active` 에서 경로 규칙만 뺀 조건이라 `Active(path) ⊆ ReportingActive()`. report-uri 를 실제로 받은 페이지의 정상 리포트가 새로 버려지는 경로는 추적을 끄는 순간의 경합뿐이고 그건 의도된 동작이다. `ADMIN_GUIDE.md:472`·`SettingsPage.tsx:1461` 이 이미 "추적 on 동안만"이라 적고 있어 문서 수정 불필요.
- 못 본 것: 실제 브라우저의 report-uri 왕복, 실제 PostgreSQL 경유 `TrackingConfig`, 프런트 전체 검증(npm ci 미실행 — 이번 diff 가 `web/` 을 건드리지 않아 생략).
- 남는 우려(차단 아님): `store.GetSetting` 에 캐시가 없어 인증 없는 리포트 1건당 DB 쿼리 1회가 늘었다 — 인증 없는 `GET /` 이 이미 같은 쿼리를 하므로 새 공격 종류는 아니지만 두 경로 모두 속도 제한이 없다. 설정 읽기 실패 시 리포트마다 Warn 한 줄이 남아 DB 장애 + 폭주가 겹치면 로그가 비례해 늘어난다.
- 릴리즈 노트에 "추적 설정을 읽지 못하는 동안 CSP 신고는 기록되지 않는다" 한 줄을 넣으면 운영자가 빈 목록을 버그로 오해하지 않는다. 다음 회차: `ReportingActive()` 가 `Active()` 앞 세 줄을 복제하므로 "추적이 켜졌다" 정의를 한쪽만 바꾸면 조용히 갈라진다.
- [러너 19:22] review approved — 리뷰 승인 (risk=low)
- [러너 19:22] pr created — https://github.com/hkjang/jikim/pull/45
- [러너 19:26] ci passed — 검사 2개 모두 success
- [러너 19:26] merge done — 90bee37
- [러너 19:31] release ci-blocked — 릴리즈 커밋 CI: failed — 성공이 아닌 검사: 폐쇄망 이미지 · 브라우저 E2E=failure (태그 보류)
