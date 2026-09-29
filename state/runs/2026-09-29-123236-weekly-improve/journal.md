# 회차 노트 2026-09-29-123236-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:32] base pinned — main@59a684c
- [러너 12:32] autonomy release — 

## 정찰 노트
- 상태 Scan 오류를 202 접수로 바꾸는 forceConfluenceSync를 선택: 제품 1파일, 기존 프런트 catch 재사용으로 문서·폴링·재시도 후보보다 작고 직접적인 효과. 초안 선작성 후 확정.
- 실제 POST 실패 재현은 미실행이며 구현자가 red 시험으로 입증해야 한다. 실제 worker fixture 동기화가 최대 불확실성; advisory lock이 이미 있어 중복 실행 결함으로 주장하지 말 것.
- 실제 pgvector DB에서 기존 Confluence 시험 2개 통과(1.800s), openapi/paging 통과. DSN 없는 첫 실행은 SKIP이었고 전체 시험은 미실행.
- headcount 스킬 3개는 Skill 도구가 없어 로컬 SKILL.md로 읽음. 코드·커밋 변경 없음; fake 배선·채널 길이·임의 sleep을 검증 근거로 쓰지 말 것.
- [러너 12:38] scout done — Confluence 수동 동기화가 상태 조회 실패를 접수 성공으로 답하지 않는다 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 0473e95: 상태 Scan 실패·행 없음에 500 QUERY_FAILED를 즉시 반환; wake/audit 전 반환하고 POST OpenAPI에 500 추가. 제품 1파일 + 시험 1 + 명세 1.
- 실제 scratch DB·App·HTTP에서 구 코드의 202 및 감사 +1을 먼저 재현; 수정 후 통과, 구 코드 재복원 시 다시 실패.
- 실제 worker 첫 검색 응답 관찰 후 production advisory lock으로 실행 완료를 기다리고 fixture 보호. sleep·대역 App·교체 worker 채널 없음.
- 지정 4시험 및 전체 Go 시험(148.628s), vet/build, OpenAPI 119경로, guard 9개, diff --check 통과.
- 확신 없는 곳·미검증: 프런트 렌더링과 wake 호출 부재 자체의 동적 계측은 하지 않음; 오류 반환 위치는 코드 검토, HTTP/감사는 실제 행동 검증.
- 프런트·worker 정책·인증·마이그레이션·릴리즈는 범위 밖이라 변경하지 않았고 긴 mutation/authz 검사는 실행하지 않음.
- 다음 역할: WEEKLY_TEST_POSTGRES_DSN 없이 DB 시험은 SKIP된다. 컨테이너 설정에서 비밀 출력 없이 DSN을 구성해야 하며 lock fixture를 제거하면 worker와 경쟁할 수 있다.
- [러너 12:44] brief accepted — 채택 — 현재 코드의 오류 무시와 실제 POST 실패가 과제서 근거에 일치했으며 범위 변경 없이 수용 기준을 검증했다.
- [러너 12:46] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low, security·legal 차단 없음. 회사 스킬 3개 로컬 원본 적용; diff·커밋·원장의 실패 재현 출력 확인.
- 실제 PostgreSQL 새 회귀 시험 4개 하위 사례 통과(2.791s); 오류/행 없음, RUNNING, 복구 후 IDLE 및 감사 증감 검증.
- ADMIN/CSRF, 고정 오류 응답, wake/audit 전 반환, 프런트 catch, OpenAPI 일치 확인; 범위 이탈·비가역 변경 없음.
- 화면 렌더링·wake 부재 동적 계측·전체/권한 시험은 재실행하지 않음; OpenAPI·diff --check 통과. 릴리즈 시 DB 없는 시험 성공을 통합 검증으로 세지 말 것.
- [러너 12:48] review approved — 리뷰 승인 (risk=low)
- [러너 12:48] pr created — https://github.com/hkjang/weekly/pull/26
