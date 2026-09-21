# PR 처리기 노트 2026-09-22-034305-sqlon-shepherd — sqlon PR #11
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-22-015440-sqlon-improve)
# 회차 노트 2026-09-22-015440-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:54] base pinned — main@57f99b7
- [러너 01:54] autonomy release — 

## 정찰 노트
- 공유 프로파일 PUT의 visibility 생략이 private 저장·manage 권한 검사 우회로 이어지는 소스 경로를 확인해 선택; 빈 문서 색인보다 사용자 영향이 크고 기존 성공 과제와 겹치지 않는다.
- 신규 HTTP 재현·실제 PG·브라우저는 미확인. 구현자는 실제 로그인/POST/grant/PUT/제3자 GET 회귀부터 작성하고 red를 확인한다.
- auth/migrations/workflows·공통 ACL·단독 모드는 건드리지 말고 유효 visibility를 검사/저장/응답에서 일치시킬 것. 전체 test/vet 통과, 저장소 변경 없음.
- 요청 스킬 3개는 도구/로컬/리소스에서 발견하지 못해 고유 형식 미확인으로 기록; 기존 보류 10개 유지·재평가, 신규 2개 추가.
- [러너 01:58] scout done — 프로파일 PUT에서 visibility 생략 시 기존 공개 범위를 보존 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- ccbd0e7: 메타 DB 프로파일 PUT의 visibility 생략/빈 값은 기존 값 보존; 생성의 private 기본값 유지. CHANGELOG Unreleased 한 줄 추가.
- 실제 로그인→Register mux→POST→grant→PUT→GET 회귀 52개 경우. 수정 전 shared/owner·admin·manager 생략·빈 값 6건 실패, 수정 후 응답·저장·목록 및 name 저장 일치.
- 명시 shared↔private 변경·manage 같은 값 허용/다른 값 403·유효하지 않은 값 400·use 403·익명 401 및 거절 시 정의/공개 범위 불변 검증.
- 검증: 지정 표적 테스트, go test ./... -count=1, go vet ./..., go build ./..., git diff --check 통과; 커밋 후 작업 트리 깨끗함.
- 확신 없는 곳·검증 못 한 것: 실제 PostgreSQL 통합·브라우저 미검증. 요청 technology 스킬 3개는 도구/로컬 검색에서 찾지 못해 고유 절차/형식 미확인.
- 일부러 하지 않은 것: auth/전용 visibility API/공통 ACL/단독 모드/전체 정의 교체/JSON 디코더를 변경하지 않음. 기존 12개 아이디어 유지하고 선택 항목만 done 처리.
- 다음 역할 주의: 새 회귀는 실제 MemStore와 HTTP 배선 사용, 외부 DB 불필요. CHANGELOG 혼합 줄바꿈을 유지했으므로 전체 재포맷 금지.
- [러너 02:02] brief accepted — 채택 — 현재 코드에서 결함을 실제 HTTP로 재현했고 지정된 3개 파일만 수정하여 생성 기본값과 공개 범위 변경 권한 계�
- [러너 02:02] verify passed — 검증 3개 통과 (auto)
- [러너 02:02] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 02:02] pr created — https://github.com/hkjang/sqlon/pull/11
