# 회차 노트 2026-10-07-175805-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:58] base pinned — main@4e7911e
- [러너 17:58] autonomy release — 

## 정찰 노트
- 두 회차 연속 차선으로 밀렸던 페이지 tiebreak 를 골랐다. 이전에 M 이던 것을 admin.go 한 파일(세 ORDER BY)로 좁혀 S 로 내렸고, 재사용할 테스트 하비스(admin_total_pg_test.go 의 adminListFixture·patchUser)가 이미 있다는 것을 열어서 확인했다. 차선(ENCRYPTION_KEY 거부 회귀)은 테스트 전용이라 가치가 낮고, 바로 앞 회차가 같은 성격(secretbox 거부 회귀)이어서 뒤로 뒀다.
- serviceLocation(8회차)과 loadAPIKeyPolicyContext(5회차)는 이전 회차의 자기 지시대로 rejected 로 내렸다 — 다음 정찰이 다시 집어 들지 말 것.
- 추측으로 적은 것: ① 동일 created_at + 페이지 사이 UPDATE 로 실제 PG17 Red 가 난다는 것은 **미검증**이다(이 환경에 PG 가 떠 있는지 확인하지 못했다). 과제서에 대안 레시피와 "Red 실패 시 견고성으로 좁혀라" 를 명시했다. ② admin.go:571/709/746 의 행 번호는 이번에 직접 읽어 확인했다. ③ catalog.go:389-392(playWindows)와 README 42/44행은 재확인하지 않은 이전 근거다.
- 구현자가 조심할 것: gameSelect 상수 본문을 바꾸면 공개 게임 목록까지 번진다 — adminListGames 의 ORDER BY 절만 바꿀 것. pagedTotal/감사 CSV 공유 술어는 2026-10-04 의 계약이므로 손대지 말 것. DSN 없는 go test 는 DB 검증이 아니다.
- [러너 18:03] scout done — 관리자 목록 페이지네이션에 유일한 tiebreak 을 넣어 OFFSET 경계에서 행이 겹치거나 사라지지 않게 하기 (가
