# 회차 노트 2026-09-21-085420-SecCheck-improve — SecCheck
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:54] base pinned — main@eb2a3b0
- [러너 08:54] autonomy release — 

## 정찰 노트
- 날짜 입력 검증을 선택: 실제 오류 응답 개선이며 문서 정리보다 가치가 높고 반복된 vulndb 재판정을 피한다.
- reportFilter의 REST·Excel·MCP 배선을 확인했으나 잘못된 날짜 HTTP 재현은 미확인; 기존 통합 테스트는 DSN 없어 SKIP, CSV 세 테스트는 PASS.
- 세 필수 스킬은 headcount/plugins에서 발견·읽음; 이전 스킬 부재 판정을 그대로 반복하지 말 것.
- 공통 필터 두 호출자를 함께 고치고 auth/migrations/workflows·기존 PR 내용은 손대지 말 것. CI 최신 상태는 미확인.
- [러너 08:59] scout done — 심의 리포트 날짜 필터를 REST·Excel·MCP 공통으로 검증 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 350d36c: reportFilter 공통 검증으로 REST·Excel 422+필드 details와 MCP 안전한 날짜 도구 오류를 제공한다. SQL 날짜 포함 범위는 유지했다.
- 실제 PostgreSQL·NewServer·로그인·HTTP 회귀 세 테스트 추가: 날짜 오류, 정상 JSON/MCP/XLSX 집계 일치, 일반 DB 오류 비공개.
- Red→Green→수정 되돌림 Red→복원 Green 확인. 최종 회귀 PASS 83/FAIL 0/SKIP 0(부모·하위 포함), 제한 race PASS, 전체 Go 테스트 PASS(web 159.641s), vet exit 0.
- 확신 없는 곳·검증 못 한 것: 원격 CI·govulncheck·릴리즈 최신 상태, 브라우저 UI, 문자열 아닌 MCP 날짜 인자. 로컬 통과를 CI green으로 해석하지 말 것.
- 일부러 하지 않은 것: auth·migrations·워크플로·의존성·프런트·다른 목록 필터 변경 및 새 감사로그 추가는 범위 밖이다.
- 다음 역할 주의: TEST_POSTGRES_DSN 없으면 새 테스트도 SKIP. fixture는 심의를 모두 생성한 뒤 날짜를 바꿔야 count 기반 채번 충돌을 피한다. 전용 테스트 컨테이너는 종료 시 제거한다.
- 상세 증거: verification.md와 baseline/red/reverted-red/green/race/all-tests/vet.log. Skill 전용 도구 부재로 지정된 세 technology SKILL.md 원문을 직접 읽고 적용했다.
- [러너 09:07] brief accepted — 채택 — 날짜 검증 누락과 두 호출자 배선이 현재 코드와 일치했고 전용 DB에서 500 및 비정규 날짜·역전 기간의 잘못된 �
- [러너 09:07] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- reject / security: internal/web/analytics.go:211 및 server.go:245부터 확인. Momento 활성 시 익명 GET·DELETE 임의 수집기 경로 전달을 실제 HTTP로 재현했다.
- 로컬 main=c32eec7은 pinned eb2a3b0와 다름: 차단은 main...HEAD 누적 변경에 있고 날짜 수정 350d36c 단독의 차단 결함은 찾지 못했다.
- 전용 PostgreSQL 16에서 날짜·리포트·추적 회귀 PASS, 저장소 무수정 Go overlay 보안 재현 PASS; 테스트 컨테이너 제거.
- 원격 CI·govulncheck·브라우저·누적 변경 전수 검증 및 공급자 개인정보 계약은 미확인; MCP 비문자열 날짜 누락과 036 롤백 제약은 review.json에 남김.
- [러너 09:10] review rejected — 리뷰 거절: internal/web/analytics.go:211 [P1 보안·릴리즈 차단] Momento 프록시가 활성화되면 server.go:245의 비인증 라우트가 방문자의 임의 경로·쿼리·HTTP 메서�
- [러너 09:10] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 09:10] pr created — https://github.com/hkjang/SecCheck/pull/14
