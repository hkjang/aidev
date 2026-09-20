# PR 처리기 노트 2026-09-21-034616-moyro-shepherd — moyro PR #20
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-21-030413-moyro-improve)
# 회차 노트 2026-09-21-030413-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:04] base pinned — main@9acbddd
- [러너 03:04] autonomy release — 

## 정찰 노트
- 재정렬 서비스의 ID 정규화와 핸들러 입력 에코 불일치를 확인해 선택; 메일 선행 기능 부재·브라우저 준비 비용·옛 PR 반려 여부 불확실 후보보다 작고 근거가 명확하다.
- 응답·GET·실제 ws.Hub 이벤트를 함께 검증하도록 지시. 정렬 동률/동시 요청 스냅샷 정책은 확대하지 않는다.
- 관련 Go 테스트·소스 크기 검사 통과, DB 테스트는 DSN 없어 skip. 되읽기만 실패하는 DB 주입 방법과 옛 PR 판정은 미확인.
- 요청한 세 부서 스킬의 도구·파일을 찾지 못해 원문 절차 적용은 미확인; 프로필은 main의 마지막 migration이 000017인 점을 바로잡았다. 코드·커밋 변경 없음.
- [러너 03:08] scout done — 사이드바 재정렬 PUT 응답·WebSocket 이벤트를 실제 저장된 전체 순서로 통일 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- UpdateOrder 성공 뒤 Order 한 번으로 되읽어 PUT 응답과 WebSocket 이벤트를 같은 전체 저장 순서로 통일; 저장 서비스는 그대로 유지.
- 신규 sidebar_order_postgres_test.go: 실제 PG16·서비스·PUT/GET·hub.Run·두 대상 Client.Send와 타 사용자로 혼합/빈 요청 및 DB 행·이벤트·400/413 검증. 변경 전 혼합/빈 요청 실패 확인.
- 관련 패키지 -race -p 1 -count=1, 전체 go test -race -p 1 ./..., go vet ./..., 소스 크기·diff 검사 통과(실제 전용 DB DSN).
- 미검증: 비어 있지 않은 UPDATE 커밋 직후 SELECT만 실패하는 장애. 빈 입력은 UPDATE가 무쿼리 성공하므로 DROP TABLE로 Order만 실패하는 500·미발행을 검증; 이미 끝난 UPDATE의 롤백 보장은 없다.
- 미검증: 웹/브라우저·실제 플러그인 archive 시나리오; technology 스킬 세 개는 도구·로컬 검색에서 찾지 못했으며 원문 적용을 주장하지 않음.
- 동률 정렬 정책·동시 요청 스냅샷·서비스/허브 구현·auth·릴리즈는 범위 밖이라 변경하지 않음. 회귀 테스트는 MOYRO_TEST_POSTGRES_DSN이 없으면 skip됨.
- [러너 03:16] brief accepted — 채택 — 입력 에코와 저장 정규화 불일치가 현재 코드에 그대로 존재하며 저장 로직을 변경하지 않고 지정된 핸들러 수�
- [러너 03:16] verify passed — 검증 2개 통과 (policy)
- [러너 03:16] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 03:17] pr created — https://github.com/hkjang/moyro/pull/20
