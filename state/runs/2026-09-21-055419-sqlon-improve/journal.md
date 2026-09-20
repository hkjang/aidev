# 회차 노트 2026-09-21-055419-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:54] base pinned — main@57f99b7
- [러너 05:54] autonomy release — 

## 정찰 노트
- 비동기 잡 만료를 선택: API의 10분/404 계약과 두 접근 경로가 불일치하며 이전 성공 과제와 독립적이다. 문서 색인보다 실제 동작 가치가 높고 auth/CI보다 위험이 낮다.
- 근거는 소스 호출 경로; 새 HTTP 재현·race·실 DB는 미확인. 전체 test/vet는 통과. 실제 제출→완료 뒤 시간만 조정하여 조회·취소를 각각 첫 접근으로 검증할 것.
- 선행 인증·감사 수정은 main에 없다. 재수행하지 말고 만료와 무관한 보호 경로/CRLF 정리를 피한다. 요청 스킬 3개는 제공 도구·검색한 로컬 경로에서 미발견.
- [러너 05:58] scout done — 비동기 쿼리 잡의 10분 만료를 조회·취소 경로에서도 적용 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- c941cb9: jobView/cancelJob의 기존 mutex 안에 prune 두 줄 추가; 완료 후 10분 지난 잡의 첫 조회·취소를 404로 만들고 저장소에서 제거. CHANGELOG Unreleased 한 항목만 추가, 기존 줄바꿈 보존.
- 실제 로그인→Register mux→HTTP 202 제출→백그라운드 완료 후 FinishedAt만 조정; 수정 전 GET 200+SQL 노출, 별도 fixture의 첫 취소 200을 재현. 미만료 소유자/admin·익명 401·타인 조회 403/취소 404 및 실제 TCP에서 대기 중인 오래된 잡 조회·취소 검증.
- 검증: TestAsync 1.554초, 대상 race 16.164초, 전체 test(mcp 4.453초), vet, go build ./..., diff --check, 변경 Go 파일 gofmt 모두 통과.
- 확신 없는 곳·검증 못 한 것: 라이브 DB 성공 결과행·integration 태그·브라우저는 미검증; 만료 검증은 실제 프로파일 조회 실패로 완료된 잡을 사용한다.
- 일부러 하지 않은 것: 정기 청소 goroutine·TTL/권한/실행 정책 변경·선행 인증/감사 커밋 재구현·릴리즈/푸시는 범위 밖이라 제외.
- 다음 역할 주의: SELECT FROM DUAL은 fixture 카탈로그가 거절해 TS.TBL1을 사용. race 첫 실행은 PostgreSQL AST의 WASM 컴파일로 5초를 넘어 스택으로 확인 후 테스트 대기 상한을 30초로 설정; 외부 DB 없이 실행 가능.
- 스킬: technology 3종과 callable Skill 도구는 제공 목록·로컬 검색에서 미발견. 로컬 superpowers systematic-debugging/test-driven-development/verification-before-completion SKILL.md를 읽어 보조 적용했으며 technology 원문의 절차·반환 형식 준수는 확인 불가.
- [러너 06:03] brief accepted — 채택 — 현 코드의 결함을 실제 HTTP 회귀로 재현했고 두 접근 경로에 최소 수정으로 만료 계약을 적용했다.
- [러너 06:04] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- approve / low, blocking 없음. diff·로그·HTTP 인증/인가 배선·만료 잠금/경계·테스트 단언·CHANGELOG·범위와 revert 가능성 확인; 새로 도입된 결함 없음.
- 전체 test, 대상 TestAsync race(15.228초), vet, diff --check 통과. 수정 전에는 첫 접근의 404·저장소 삭제 단언을 만족하지 못함을 코드로 확인(수정 전 재실행은 하지 않음).
- 라이브 DB 성공 결과행·integration·브라우저 미검증; 성공/실패 공통 FinishedAt 삭제 경로 확인. 요청 없는 동안 정기 청소는 없으므로 릴리즈에서 물리적 10분 삭제 보장으로 확대 금지.
- 요청한 세 부서 스킬과 Skill 도구는 제공 목록·검색한 로컬 경로에서 미발견하여 원문 절차/반환 형식 준수 확인 불가; 사용자 지정 심사 기준 적용.
- [러너 06:05] review approved — 리뷰 승인 (risk=low)
- [러너 06:05] pr created — https://github.com/hkjang/sqlon/pull/10
- [러너 06:10] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
