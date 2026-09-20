## 2026-09-21
- 선택: 사이드바 재정렬 PUT 응답·WebSocket 이벤트를 실제 저장된 전체 순서로 통일 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: UpdateOrder 성공 후 Order를 한 번 조회해 JSON 응답과 sidebar_category_order_updated의 data.order에 같은 전체 저장 순서를 사용하고, 되읽기 오류는 기존 ID의 500으로 처리해 성공 이벤트를 막았다. 실제 PostgreSQL 16·sidebar.Service·PUT/GET 핸들러·실행 중 ws.Hub와 Client.Send로 중복/유령/타 사용자·팀/생략/빈 입력, DB 행 보존, 대상 사용자 두 탭 이벤트 일치, 저장 및 빈 입력 뒤 되읽기 장애, 400/413을 검증했으며 변경 전 실패를 확인했다. 실제 DB DSN으로 관련 패키지 -race -p 1 -count=1, 전체 go test -race -p 1 ./..., go vet ./..., 소스 크기 검사가 통과했다. 비어 있지 않은 UPDATE 커밋 직후 SELECT만 실패하는 장애·웹·실제 플러그인 archive 시나리오는 미검증이고, 요청된 technology 스킬 세 개는 도구/로컬 검색에서 찾지 못해 원문 적용을 확인할 수 없었다.
- 보류 아이디어:
  - IsMember DB 오류 403 위장 분리 (가치 2 / 위험 1 / 작업량 M): 차선 세 핸들러만 S로 한정 가능, 이번에는 미변경.
  - getPreferenceByName 404/500 분리 (가치 3 / 위험 2 / 작업량 S): 옛 PR 반려 여부 확인 전 보류.
  - 자동화 dead run 소유자 메일 (가치 3 / 위험 2 / 작업량 S): mail 선행 기능 머지 뒤 재평가.
  - silent SSO 거절 e2e (가치 3 / 위험 1 / 작업량 S): 브라우저·서버 준비 비용으로 보류.
- 과제서: 채택 — 입력 에코와 저장 정규화 불일치가 현재 코드에 그대로 존재하며 저장 로직을 변경하지 않고 지정된 핸들러 수정과 실제 DB·hub 회귀 테스트로 해결했다.
