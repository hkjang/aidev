# PR 처리기 노트 2026-09-20-180614-weekly-shepherd — weekly PR #20
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-20-173406-weekly-improve)
# 회차 노트 2026-09-20-173406-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:34] base pinned — main@0253313
- [러너 17:34] autonomy release — 

## 정찰 노트
- 공유 DSN 직접 사용 12개를 기존 scratch 하네스로 옮기는 S 과제 선택: 반복된 DB 간섭 원인을 줄이며 보호 경로와 API를 건드리지 않는다. 현 코드 기준 db_integration은 7개가 아닌 8개다.
- weekly_u_ 청소는 강제 삭제 범위를 넓혀 별도 보류; mail·handoff 후속은 선행 코드 부재, openWeeks는 서버/화면 경계가 커서 제외했다. authz 표식은 이미 있어 문서 보완만 차선이다.
- go test ./... -count=1 통과(2.802초), DSN 미설정으로 실제 DB는 미확인. 구현자는 실제 pool·PostgreSQL로 격리와 기존 동작을 증명하고 skip을 완료로 세지 말 것.
- 요청한 회사 스킬 세 개는 도구·로컬 파일에서 찾지 못해 적용 미확인으로 기록. 프로필 갱신, 기존 후보를 유지·재평가하고 새 후보 2개를 더했다. 코드 수정·커밋 없음.
- [러너 17:38] scout done — 공유 DSN을 직접 여는 통합 시험 12개를 기존 스크래치 DB 하네스로 격리 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- e909ca0: 네 시험 파일 12곳을 createScratchDatabase로 격리하고 실제 pool의 current_database() 검증을 추가했다.
- 인증 시험의 중간 db.Close()는 유지하고 조기 실패에도 pool 종료가 scratch 정리보다 먼저 실행되도록 defer를 추가했다.
- 실제 pgvector:pg16 전용 컨테이너에서 기존 배선 실패 12건 → 변경 후 -count=2 통과 24건(skip 0); full-test 144.764초, vet·build·guard 5개 통과.
- DSN 미설정 전체 시험도 2.928초 통과. 종료 후 weekly_h_*·weekly_tmpl_* 잔여 0개 확인; 실행 로그는 이 회차 디렉터리에 있다.
- 확신 없는 곳·검증 못 한 것: 회사 스킬 세 개의 도구·원문 없음; pgvector 미설치 skip 분기는 유지했지만 별도 미설치 서버로 실행하지 않았다.
- 일부러 하지 않은 것: 제품 코드·마이그레이션·CI·fixture 정리·weekly_u_* 청소 변경, mutation/authz 실행은 이번 범위 밖이다.
- 다음 역할 주의: 대상 시험의 실제 검증에는 URL DSN과 CREATE DATABASE 권한·pgvector가 필요하며, DSN 없는 skip은 통합 검증 성공이 아니다.
- [러너 17:44] brief accepted — 채택 — 현재 코드의 직접 접속 12곳을 확인했고 전용 PostgreSQL 컨테이너를 확보해 실제 DB 반복 실행과 격리 검증까지 완�
- [러너 17:46] verify passed — 검증 7개 통과 (auto)
- [러너 17:46] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 17:46] pr created — https://github.com/hkjang/weekly/pull/20
